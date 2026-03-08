package vpn

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

// udpExcludedPorts son los puertos que NO se redirigen al servidor UDP.
// Corresponden a: DNS(323), ZiVPN(2200), BadVPN(7100,7200,7300), gRPC(10004,10008)
const udpExcludedPorts = "2200,7300,7200,7100,323,10008,10004"

// udpListenPort es el puerto donde escucha el servidor UDP
const udpListenPort = "2100"

// InstallUDPCustom instala el servidor UDP Custom para HTTP Custom app.
// Usa el binario /usr/bin/udp con reglas NAT completas y exclusiones.
func InstallUDPCustom(port string) error {
	// 0. Dependencias
	_ = exec.Command("apt-get", "update").Run()
	_ = exec.Command("apt-get", "install", "-y", "curl", "iptables", "libpam0g").Run()

	// Habilitar IPv4 Forwarding
	_ = exec.Command("sysctl", "-w", "net.ipv4.ip_forward=1").Run()
	_ = exec.Command("bash", "-c", "grep -q 'net.ipv4.ip_forward=1' /etc/sysctl.conf || echo 'net.ipv4.ip_forward=1' >> /etc/sysctl.conf").Run()

	archRaw := runtime.GOARCH
	var binURL string

	if archRaw == "amd64" {
		binURL = "https://github.com/firewallfalcons/FirewallFalcon-Manager/raw/main/udp/udp-custom-linux-amd64"
	} else if archRaw == "arm64" {
		binURL = "https://github.com/firewallfalcons/FirewallFalcon-Manager/raw/main/udp/udp-custom-linux-arm"
	} else {
		return fmt.Errorf("arquitectura no soportada para UDP Custom: %s", archRaw)
	}

	// Descargar binario como /usr/bin/udp
	errDL := exec.Command("curl", "-L", "-s", "-f", "-o", "/usr/bin/udp", binURL).Run()
	if errDL != nil {
		return fmt.Errorf("fallo la descarga del binario udp: %v", errDL)
	}
	os.Chmod("/usr/bin/udp", 0755)

	// Configuración JSON
	configJSON := `{
	"listen": ":` + udpListenPort + `",
	"stream_buffer": 33554432,
	"receive_buffer": 83886080,
	"auth": {
		"mode": "passwords"
	}
}`
	os.WriteFile("/usr/bin/config.json", []byte(configJSON), 0644)

	// Servicio Systemd
	svc := `[Unit]
Description=UDP Custom Server for HTTP Custom
After=network.target

[Service]
Type=simple
User=root
WorkingDirectory=/usr/bin
ExecStart=/usr/bin/udp server -exclude ` + udpExcludedPorts + ` /usr/bin/config.json
Restart=always
RestartSec=3

[Install]
WantedBy=multi-user.target`

	os.WriteFile("/etc/systemd/system/udp-custom.service", []byte(svc), 0644)
	exec.Command("systemctl", "daemon-reload").Run()
	_ = exec.Command("systemctl", "enable", "udp-custom.service").Run()
	if err := exec.Command("systemctl", "restart", "udp-custom.service").Run(); err != nil {
		return fmt.Errorf("fallo reiniciar udp-custom.service: %v", err)
	}

	// Verificar inicio
	time.Sleep(1 * time.Second)
	if err := exec.Command("systemctl", "is-active", "--quiet", "udp-custom.service").Run(); err != nil {
		return fmt.Errorf("udp-custom no pudo iniciarse. Revisa journalctl -u udp-custom.service")
	}

	// Reglas iptables: Detección robusta de interfaz
	devOut, _ := exec.Command("bash", "-c", "ip -4 route show default | awk '{print $5}' | head -1").Output()
	dev := strings.TrimSpace(string(devOut))
	if dev == "" {
		devOut, _ = exec.Command("bash", "-c", "ip link show up | grep -v loopback | grep -v 'lo:' | head -1 | awk '{print $2}' | cut -d':' -f1").Output()
		dev = strings.TrimSpace(string(devOut))
	}

	if dev != "" {
		// Limpiar reglas anteriores de UDP Custom
		exec.Command("bash", "-c", "iptables -t nat -S PREROUTING | grep 'DNAT.*:"+udpListenPort+"' | sed 's/-A/-D/' | while read line; do iptables -t nat $line 2>/dev/null; done").Run()
		exec.Command("bash", "-c", "iptables -S INPUT | grep '"+udpListenPort+"' | sed 's/-A/-D/' | while read line; do iptables $line 2>/dev/null; done").Run()

		// Redirect DNS: 53 → 5300
		_ = exec.Command("iptables", "-t", "nat", "-A", "PREROUTING", "-i", dev, "-p", "udp", "--dport", "53", "-j", "REDIRECT", "--to-ports", "5300").Run()

		// NAT DNAT completo con exclusiones (replicar configuración del servidor)
		// Rangos que se redirigen a :2100 (excluyendo puertos específicos)
		natRanges := []string{
			"1:322",
			"324:2199",
			"2201:7099",
			"7101:7199",
			"7201:7299",
			"7301:10003",
			"10005:10007",
			"10009:65535",
		}

		for _, r := range natRanges {
			_ = exec.Command("iptables", "-t", "nat", "-A", "PREROUTING", "-i", dev, "-p", "udp", "--dport", r, "-j", "DNAT", "--to-destination", ":"+udpListenPort).Run()
		}

		// Permitir en INPUT
		_ = exec.Command("iptables", "-I", "INPUT", "1", "-p", "udp", "--dport", udpListenPort, "-j", "ACCEPT").Run()

		// Masquerade
		_ = exec.Command("iptables", "-t", "nat", "-D", "POSTROUTING", "-o", dev, "-j", "MASQUERADE").Run()
		_ = exec.Command("iptables", "-t", "nat", "-A", "POSTROUTING", "-o", dev, "-j", "MASQUERADE").Run()
	}

	return nil
}

// RemoveUDPCustom desinstala el servicio y limpia reglas iptables
func RemoveUDPCustom() error {
	_ = exec.Command("systemctl", "stop", "udp-custom.service").Run()
	_ = exec.Command("systemctl", "disable", "udp-custom.service").Run()
	os.Remove("/etc/systemd/system/udp-custom.service")
	os.Remove("/usr/bin/config.json")
	os.Remove("/usr/bin/udp")

	devOut, _ := exec.Command("bash", "-c", "ip -4 route ls | grep default | grep -Po '(?<=dev )(\\S+)' | head -1").Output()
	dev := strings.TrimSpace(string(devOut))
	if dev != "" {
		// Limpiar reglas NAT de UDP Custom
		exec.Command("bash", "-c", "iptables -t nat -S PREROUTING | grep 'DNAT.*:"+udpListenPort+"' | sed 's/-A/-D/' | while read line; do iptables -t nat $line 2>/dev/null; done").Run()
		exec.Command("bash", "-c", "iptables -S INPUT | grep '"+udpListenPort+"' | sed 's/-A/-D/' | while read line; do iptables $line 2>/dev/null; done").Run()
		// Limpiar redirect DNS
		exec.Command("bash", "-c", "iptables -t nat -S PREROUTING | grep 'REDIRECT.*5300' | sed 's/-A/-D/' | while read line; do iptables -t nat $line 2>/dev/null; done").Run()
	}

	exec.Command("systemctl", "daemon-reload").Run()
	return nil
}
