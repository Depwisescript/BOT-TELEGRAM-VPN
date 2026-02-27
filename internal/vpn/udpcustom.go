package vpn

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

// InstallUDPCustom installs the UDP-Custom server specifically for HTTP Custom app.
// It uses local system users for authentication.
func InstallUDPCustom(port string) error {
	// 0. Dependencies
	_ = exec.Command("apt-get", "update").Run()
	_ = exec.Command("apt-get", "install", "-y", "curl", "iptables").Run()

	// Habilitar IPv4 Forwarding (Requerido para NAT)
	_ = exec.Command("sysctl", "-w", "net.ipv4.ip_forward=1").Run()
	_ = exec.Command("bash", "-c", "echo 'net.ipv4.ip_forward=1' >> /etc/sysctl.conf").Run()

	archRaw := runtime.GOARCH
	var binURL string

	if archRaw == "amd64" {
		binURL = "https://github.com/http-custom/udp-custom/raw/main/bin/udp-custom-linux-amd64"
	} else if archRaw == "arm64" {
		// Public community build for ARM64 as official is amd64 only
		binURL = "https://github.com/powermx/udp-custom-arm64/raw/refs/heads/main/udp-custom-linux-arm64"
	} else {
		return fmt.Errorf("arquitectura no soportada para UDP Custom: %s", archRaw)
	}

	// Directoy for UDP Custom
	os.MkdirAll("/etc/udp-custom", 0755)

	// Download binary
	errDL := exec.Command("curl", "-L", "-s", "-f", "-o", "/usr/local/bin/udp-custom", binURL).Run()
	if errDL != nil {
		return fmt.Errorf("fallo la descarga del binario udp-custom: %v", errDL)
	}
	os.Chmod("/usr/local/bin/udp-custom", 0755)

	// Configuration
	// This format uses local passwords (system users)
	configJSON := `{
	"listen": ":` + port + `",
	"stream_buffer": 33554432,
	"receive_buffer": 83886080,
	"auth": {
		"mode": "passwords"
	}
}`
	os.WriteFile("/etc/udp-custom/config.json", []byte(configJSON), 0644)

	// Systemd Service
	svc := `[Unit]
Description=UDP Custom Server for HTTP Custom
After=network.target

[Service]
Type=simple
User=root
WorkingDirectory=/etc/udp-custom
ExecStart=/usr/local/bin/udp-custom server
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

	// Verify startup
	time.Sleep(1 * time.Second)
	if err := exec.Command("systemctl", "is-active", "--quiet", "udp-custom.service").Run(); err != nil {
		return fmt.Errorf("udp-custom no pudo iniciarse. Revisa journalctl -u udp-custom.service")
	}

	// Routing (similar to ZiVPN)
	devOut, _ := exec.Command("bash", "-c", "ip -4 route ls | grep default | grep -Po '(?<=dev )(\\S+)' | head -1").Output()
	dev := strings.TrimSpace(string(devOut))
	if dev == "" {
		devOut, _ = exec.Command("bash", "-c", "ip link show up | grep -v loopback | grep -v 'lo:' | head -1 | awk '{print $2}' | cut -d':' -f1").Output()
		dev = strings.TrimSpace(string(devOut))
	}

	if dev != "" {
		// LIMPIEZA ROBUSTA: Borrar CUALQUIER regla que mencione el rango 13000:19999
		// Esto evita conflictos con ZiVPN y limpia reglas viejas
		exec.Command("bash", "-c", "iptables -t nat -S PREROUTING | grep '13000:19999' | sed 's/-A/-D/' | while read line; do iptables -t nat $line; done").Run()
		exec.Command("bash", "-c", "iptables -S INPUT | grep '13000:19999' | sed 's/-A/-D/' | while read line; do iptables $line; done").Run()

		// APLICAR REGLAS: Usar -I para prioridad máxima
		_ = exec.Command("iptables", "-t", "nat", "-I", "PREROUTING", "1", "-i", dev, "-p", "udp", "--dport", "13000:19999", "-j", "REDIRECT", "--to-port", port).Run()

		// Permitir en INPUT
		_ = exec.Command("iptables", "-I", "INPUT", "1", "-p", "udp", "--dport", port, "-j", "ACCEPT").Run()
		_ = exec.Command("iptables", "-I", "INPUT", "1", "-p", "udp", "--dport", "13000:19999", "-j", "ACCEPT").Run()

		// Masquerade
		_ = exec.Command("iptables", "-t", "nat", "-D", "POSTROUTING", "-o", dev, "-j", "MASQUERADE").Run()
		_ = exec.Command("iptables", "-t", "nat", "-A", "POSTROUTING", "-o", dev, "-j", "MASQUERADE").Run()
	}

	return nil
}

// RemoveUDPCustom uninstalls the service
func RemoveUDPCustom() error {
	_ = exec.Command("systemctl", "stop", "udp-custom.service").Run()
	_ = exec.Command("systemctl", "disable", "udp-custom.service").Run()
	os.Remove("/etc/systemd/system/udp-custom.service")
	os.RemoveAll("/etc/udp-custom")
	os.Remove("/usr/local/bin/udp-custom")

	devOut, _ := exec.Command("bash", "-c", "ip -4 route ls | grep default | grep -Po '(?<=dev )(\\S+)' | head -1").Output()
	dev := strings.TrimSpace(string(devOut))
	if dev != "" {
		exec.Command("bash", "-c", "iptables -t nat -S PREROUTING | grep '13000:19999' | sed 's/-A/-D/' | while read line; do iptables -t nat $line; done").Run()
		exec.Command("bash", "-c", "iptables -S INPUT | grep '13000:19999' | sed 's/-A/-D/' | while read line; do iptables $line; done").Run()
	}

	exec.Command("systemctl", "daemon-reload").Run()
	return nil
}
