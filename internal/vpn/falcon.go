package vpn

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
)

// InstallFalcon descarga e instala Falcon Proxy
func InstallFalcon(port string) (string, error) {
	arch := runtime.GOARCH
	binName := "falconproxy"
	if arch == "arm64" {
		binName = "falconproxyarm"
	}

	// Intentar primero /latest/, luego tag explícito como fallback
	urls := []string{
		fmt.Sprintf("https://github.com/firewallfalcons/FirewallFalcon-Manager/releases/latest/download/%s", binName),
		fmt.Sprintf("https://github.com/firewallfalcons/FirewallFalcon-Manager/releases/download/v1.2-RustFast/%s", binName),
	}

	downloaded := false
	for _, downURL := range urls {
		cmd := exec.Command("curl", "-L", "-s", "-f", "--connect-timeout", "15", "-o", "/usr/local/bin/falconproxy", downURL)
		if err := cmd.Run(); err == nil {
			downloaded = true
			break
		}
	}

	if !downloaded {
		return "", fmt.Errorf("fallo descarga falconproxy: no se pudo descargar desde ninguna URL")
	}
	os.Chmod("/usr/local/bin/falconproxy", 0755)

	// 2. Configuración
	configContent := fmt.Sprintf("PORTS=\"%s\"\nINSTALLED_VERSION=\"latest\"\n", port)
	os.WriteFile("/etc/falconproxy.conf", []byte(configContent), 0644)

	// 3. Crear servicio Systemd
	service := fmt.Sprintf(`[Unit]
Description=Falcon Proxy Service
After=network.target

[Service]
User=root
Type=simple
ExecStart=/usr/local/bin/falconproxy -p %s
Restart=always
RestartSec=2s

[Install]
WantedBy=multi-user.target
`, port)

	os.WriteFile("/etc/systemd/system/falconproxy.service", []byte(service), 0644)

	// 4. Iniciar servicio
	exec.Command("systemctl", "daemon-reload").Run()
	exec.Command("systemctl", "enable", "falconproxy").Run()
	if err := exec.Command("systemctl", "restart", "falconproxy").Run(); err != nil {
		return "", fmt.Errorf("fallo al iniciar falconproxy: %v", err)
	}

	return "latest", nil
}

// RemoveFalcon elimina el servicio y archivos
func RemoveFalcon() error {
	exec.Command("systemctl", "stop", "falconproxy").Run()
	exec.Command("systemctl", "disable", "falconproxy").Run()
	os.Remove("/etc/systemd/system/falconproxy.service")
	os.Remove("/usr/local/bin/falconproxy")
	os.Remove("/etc/falconproxy.conf")
	exec.Command("systemctl", "daemon-reload").Run()
	return nil
}
