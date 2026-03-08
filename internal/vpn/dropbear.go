package vpn

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// InstallDropbear instala dropbear con soporte multi-puerto y configuración avanzada.
// Ports debe ser una cadena separada por comas, ej: "143,109"
func InstallDropbear(ports string) error {
	// 1. Instalar dropbear
	exec.Command("apt-get", "update").Run()
	if err := exec.Command("apt-get", "install", "-y", "dropbear").Run(); err != nil {
		return fmt.Errorf("fallo instalacion dropbear: %v", err)
	}

	// 2. Asegurar llaves
	os.MkdirAll("/etc/dropbear", 0755)
	if _, err := os.Stat("/etc/dropbear/dropbear_rsa_host_key"); os.IsNotExist(err) {
		exec.Command("dropbearkey", "-t", "rsa", "-f", "/etc/dropbear/dropbear_rsa_host_key").Run()
	}
	if _, err := os.Stat("/etc/dropbear/dropbear_ecdsa_host_key"); os.IsNotExist(err) {
		exec.Command("dropbearkey", "-t", "ecdsa", "-f", "/etc/dropbear/dropbear_ecdsa_host_key").Run()
	}

	// 3. Crear banner por defecto si no existe
	bannerFile := "/etc/gerhanatunnel.txt"
	if _, err := os.Stat(bannerFile); os.IsNotExist(err) {
		os.WriteFile(bannerFile, []byte("Welcome to Depwise VPN Server\n"), 0644)
	}

	// 4. Detener servicio default
	exec.Command("systemctl", "stop", "dropbear").Run()
	exec.Command("systemctl", "disable", "dropbear").Run()

	// 5. Construir flags de puerto multi-puerto
	portList := strings.Split(ports, ",")
	var portFlags []string
	for _, p := range portList {
		p = strings.TrimSpace(p)
		if p != "" {
			portFlags = append(portFlags, "-p", p)
		}
	}

	if len(portFlags) == 0 {
		return fmt.Errorf("no se especificaron puertos válidos")
	}

	// 6. Construir ExecStart con todos los flags
	// Formato: /usr/sbin/dropbear -p 143 -W 65536 -p 109 -b /etc/gerhanatunnel.txt
	execStart := "/usr/sbin/dropbear -F"
	for _, flag := range portFlags {
		execStart += " " + flag
	}
	execStart += " -W 65536 -b " + bannerFile

	// 7. Crear servicio custom
	service := fmt.Sprintf(`[Unit]
Description=Dropbear Custom SSH Service (Multi-Port)
After=network.target

[Service]
Type=simple
ExecStart=%s
KillMode=process
Restart=always

[Install]
WantedBy=multi-user.target
`, execStart)

	os.WriteFile("/etc/systemd/system/dropbear_custom.service", []byte(service), 0644)

	// 8. Reiniciar
	exec.Command("systemctl", "daemon-reload").Run()
	exec.Command("systemctl", "enable", "dropbear_custom").Run()
	if err := exec.Command("systemctl", "restart", "dropbear_custom").Run(); err != nil {
		return fmt.Errorf("fallo reinicio dropbear_custom: %v", err)
	}

	return nil
}

// RemoveDropbear desinstala el paquete y limpia archivos
func RemoveDropbear() error {
	exec.Command("systemctl", "stop", "dropbear_custom").Run()
	exec.Command("apt-get", "purge", "-y", "dropbear").Run()
	os.Remove("/etc/systemd/system/dropbear_custom.service")
	os.RemoveAll("/etc/dropbear")
	exec.Command("systemctl", "daemon-reload").Run()
	return nil
}
