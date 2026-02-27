package vpn

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

type ZivpnConfig struct {
	Listen  string `json:"listen"`
	Cert    string `json:"cert"`
	Key     string `json:"key"`
	MaxConn int    `json:"max_conn"`
	Auth    struct {
		Mode   string   `json:"mode"`
		Config []string `json:"config"`
	} `json:"auth"`
}

// InstallZivpn instals udp-zivpn server version 1.4.9 on a custom port
func InstallZivpn(port string) error {
	// 0. Dependencies
	_ = exec.Command("apt-get", "update").Run()
	_ = exec.Command("apt-get", "install", "-y", "curl", "openssl", "iptables", "libc6-i386").Run()
	installLibSSL11() // Reuse logic or call the one in the same package if available

	// Habilitar IPv4 Forwarding (Requerido para NAT)
	_ = exec.Command("sysctl", "-w", "net.ipv4.ip_forward=1").Run()
	_ = exec.Command("bash", "-c", "echo 'net.ipv4.ip_forward=1' >> /etc/sysctl.conf").Run()

	archRaw := runtime.GOARCH
	var binURL string

	if archRaw == "amd64" {
		binURL = "https://github.com/zahidbd2/udp-zivpn/releases/download/udp-zivpn_1.4.9/udp-zivpn-linux-amd64"
	} else if archRaw == "arm64" {
		binURL = "https://github.com/zahidbd2/udp-zivpn/releases/download/udp-zivpn_1.4.9/udp-zivpn-linux-arm64"
	} else {
		return fmt.Errorf("arquitectura no soportada para Zivpn")
	}

	// binario
	if _, err := os.Stat("/usr/local/bin/zivpn"); os.IsNotExist(err) {
		errDL := exec.Command("curl", "-L", "-s", "-f", "-o", "/usr/local/bin/zivpn", binURL).Run()
		if errDL != nil {
			return fmt.Errorf("fallo la descarga del binario zivpn: %v", errDL)
		}
		os.Chmod("/usr/local/bin/zivpn", 0755)
	}

	// configuraciones
	os.MkdirAll("/etc/zivpn", 0755)
	configJSON := `{"listen": ":` + port + `", "cert": "/etc/zivpn/zivpn.crt", "key": "/etc/zivpn/zivpn.key", "max_conn": 0, "auth": {"mode": "passwords", "config": ["1"]}}`
	os.WriteFile("/etc/zivpn/config.json", []byte(configJSON), 0644)

	// certificados ssl requeridos internamente
	exec.Command("openssl", "req", "-new", "-newkey", "rsa:4096", "-days", "3650", "-nodes", "-x509",
		"-subj", "/C=US/ST=CA/L=LA/O=Zivpn/CN=zivpn", "-keyout", "/etc/zivpn/zivpn.key", "-out", "/etc/zivpn/zivpn.crt").Run()

	svc := `[Unit]
Description=zivpn VPN Server
After=network.target

[Service]
Type=simple
User=root
WorkingDirectory=/etc/zivpn
ExecStart=/usr/local/bin/zivpn server -c /etc/zivpn/config.json
Restart=always
RestartSec=3
Environment=ZIVPN_LOG_LEVEL=info
CapabilityBoundingSet=CAP_NET_ADMIN CAP_NET_BIND_SERVICE CAP_NET_RAW
AmbientCapabilities=CAP_NET_ADMIN CAP_NET_BIND_SERVICE CAP_NET_RAW

[Install]
WantedBy=multi-user.target`

	// Registro Systemd
	os.WriteFile("/etc/systemd/system/zivpn.service", []byte(svc), 0644)
	exec.Command("systemctl", "daemon-reload").Run()
	_ = exec.Command("systemctl", "enable", "zivpn.service").Run()
	if err := exec.Command("systemctl", "restart", "zivpn.service").Run(); err != nil {
		return fmt.Errorf("fallo reiniciar zivpn.service: %v", err)
	}

	// 4. Verification Check
	time.Sleep(1500 * time.Millisecond)
	if err := exec.Command("systemctl", "is-active", "--quiet", "zivpn.service").Run(); err != nil {
		// Capture logs on failure
		logCmd, _ := exec.Command("journalctl", "-u", "zivpn.service", "--no-pager", "-n", "10").Output()
		logs := string(logCmd)
		if logs == "" {
			logs = "No se pudieron obtener logs."
		}

		_ = exec.Command("systemctl", "stop", "zivpn.service").Run()
		_ = os.Remove("/etc/systemd/system/zivpn.service")
		_ = exec.Command("systemctl", "daemon-reload").Run()
		return fmt.Errorf("zivpn no pudo mantenerse activo en el puerto %s.\n\n📝 <b>LOGS:</b>\n<pre>%s</pre>", port, logs)
	}

	// Enrutamiento de UDP rango externo (6000-19999) hacia (port)
	devOut, _ := exec.Command("bash", "-c", "ip -4 route ls | grep default | grep -Po '(?<=dev )(\\S+)' | head -1").Output()
	dev := strings.TrimSpace(string(devOut))
	if dev == "" {
		// Fallback detection
		devOut, _ = exec.Command("bash", "-c", "ip link show up | grep -v loopback | grep -v 'lo:' | head -1 | awk '{print $2}' | cut -d':' -f1").Output()
		dev = strings.TrimSpace(string(devOut))
	}

	if dev != "" {
		// LIMPIEZA ROBUSTA: Borrar CUALQUIER regla que mencione el rango 6000:19999
		// Esto limpia incluso si la regla apunta a otro puerto (como el 7300 que vimos en los logs) o si hay duplicados
		exec.Command("bash", "-c", "iptables -t nat -S PREROUTING | grep '6000:19999' | sed 's/-A/-D/' | while read line; do iptables -t nat $line; done").Run()
		exec.Command("bash", "-c", "iptables -S INPUT | grep '6000:19999' | sed 's/-A/-D/' | while read line; do iptables $line; done").Run()
		exec.Command("bash", "-c", "iptables -S INPUT | grep -w '"+port+"' | sed 's/-A/-D/' | while read line; do iptables $line; done").Run()

		// APLICAR REGLAS: Usar -I para prioridad máxima
		_ = exec.Command("iptables", "-t", "nat", "-I", "PREROUTING", "1", "-i", dev, "-p", "udp", "--dport", "6000:19999", "-j", "REDIRECT", "--to-port", port).Run()

		// Permitir en INPUT
		_ = exec.Command("iptables", "-I", "INPUT", "1", "-p", "udp", "--dport", port, "-j", "ACCEPT").Run()
		_ = exec.Command("iptables", "-I", "INPUT", "1", "-p", "udp", "--dport", "6000:19999", "-j", "ACCEPT").Run()

		// MASQUERADE para retorno (Crucial para Linode/KVM)
		_ = exec.Command("iptables", "-t", "nat", "-D", "POSTROUTING", "-o", dev, "-j", "MASQUERADE").Run()
		_ = exec.Command("iptables", "-t", "nat", "-A", "POSTROUTING", "-o", dev, "-j", "MASQUERADE").Run()
	}

	return nil
}

// RemoveZiVPN borra el daemon
func RemoveZiVPN() error {
	exec.Command("systemctl", "stop", "zivpn.service").Run()
	exec.Command("systemctl", "disable", "zivpn.service").Run()
	os.Remove("/etc/systemd/system/zivpn.service")
	os.RemoveAll("/etc/zivpn")
	os.Remove("/usr/local/bin/zivpn")

	devOut, _ := exec.Command("bash", "-c", "ip -4 route ls | grep default | grep -Po '(?<=dev )(\\S+)' | head -1").Output()
	dev := strings.TrimSpace(string(devOut))
	if dev != "" {
		// LIMPIEZA ROBUSTA al desinstalar
		exec.Command("bash", "-c", "iptables -t nat -S PREROUTING | grep '6000:19999' | sed 's/-A/-D/' | while read line; do iptables -t nat $line; done").Run()
		exec.Command("bash", "-c", "iptables -S INPUT | grep '6000:19999' | sed 's/-A/-D/' | while read line; do iptables $line; done").Run()
		_ = exec.Command("iptables", "-t", "nat", "-D", "POSTROUTING", "-o", dev, "-j", "MASQUERADE").Run()
	}

	exec.Command("systemctl", "daemon-reload").Run()
	return nil
}

// AddZivpnUser agrega un password al config.json de zivpn
func AddZivpnUser(password string) error {
	filePath := "/etc/zivpn/config.json"
	data, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}

	var config ZivpnConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return err
	}

	config.Auth.Mode = "passwords"
	// Evitar duplicados
	exists := false
	for _, p := range config.Auth.Config {
		if p == password {
			exists = true
			break
		}
	}
	if !exists {
		config.Auth.Config = append(config.Auth.Config, password)
	}

	newData, _ := json.MarshalIndent(config, "", "    ")
	os.WriteFile(filePath, newData, 0644)

	return exec.Command("systemctl", "restart", "zivpn.service").Run()
}

// RemoveZivpnUser quita un password del config.json
func RemoveZivpnUser(password string) error {
	filePath := "/etc/zivpn/config.json"
	data, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}

	var config ZivpnConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return err
	}

	newPasslist := []string{}
	for _, p := range config.Auth.Config {
		if p != password {
			newPasslist = append(newPasslist, p)
		}
	}
	config.Auth.Config = newPasslist

	newData, _ := json.MarshalIndent(config, "", "    ")
	os.WriteFile(filePath, newData, 0644)

	return exec.Command("systemctl", "restart", "zivpn.service").Run()
}
