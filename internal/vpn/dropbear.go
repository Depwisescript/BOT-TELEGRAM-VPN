package vpn

import (
	"fmt"
	"os"
	"os/exec"
)

// InstallDropbear instala dropbear y lo configura en un puerto custom
func InstallDropbear(port string) error {
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

	// 3. Detener servicio default
	exec.Command("systemctl", "stop", "dropbear").Run()
	exec.Command("systemctl", "disable", "dropbear").Run()

	// 4. Crear servicio custom
	service := fmt.Sprintf(`[Unit]
Description=Dropbear Custom SSH Service
After=network.target

[Service]
Type=simple
ExecStart=/usr/sbin/dropbear -F -p %s -K 60 -r /etc/dropbear/dropbear_rsa_host_key -r /etc/dropbear/dropbear_ecdsa_host_key
KillMode=process
Restart=always

[Install]
WantedBy=multi-user.target
`, port)

	os.WriteFile("/etc/systemd/system/dropbear_custom.service", []byte(service), 0644)

	// 5. Reiniciar
	exec.Command("systemctl", "daemon-reload").Run()
	exec.Command("systemctl", "enable", "dropbear_custom").Run()
	if err := exec.Command("systemctl", "restart", "dropbear_custom").Run(); err != nil {
		return fmt.Errorf("fallo reinicio dropbear_custom: %v", err)
	}

	return nil
}

// RemoveDropbear desinstala el paquete
func RemoveDropbear() error {
	exec.Command("systemctl", "stop", "dropbear_custom").Run()
	exec.Command("apt-get", "purge", "-y", "dropbear").Run()
	os.Remove("/etc/systemd/system/dropbear_custom.service")
	os.RemoveAll("/etc/dropbear")
	return nil
}
