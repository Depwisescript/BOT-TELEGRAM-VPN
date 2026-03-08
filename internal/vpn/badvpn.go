package vpn

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"time"
)

// InstallBadVPN descarga el binario precompilado de badvpn y lo configura
// en múltiples puertos (7100, 7200, 7300) como en el servidor de producción.
func InstallBadVPN(port string) error {
	// 1. Dependencias
	_ = exec.Command("apt-get", "update").Run()
	_ = exec.Command("apt-get", "install", "-y", "curl", "screen").Run()

	// 2. Descargar binario precompilado según arquitectura
	if _, err := os.Stat("/usr/bin/badvpn"); os.IsNotExist(err) {
		arch := runtime.GOARCH
		var binURL string

		if arch == "amd64" {
			binURL = "https://github.com/firewallfalcons/FirewallFalcon-Manager/raw/main/udp/badvpn-udpgw64"
		} else if arch == "arm64" {
			binURL = "https://github.com/firewallfalcons/FirewallFalcon-Manager/raw/main/udp/badvpn-udpgw-arm64"
		} else {
			// Fallback: intentar compilar desde fuente
			return installBadVPNFromSource(port)
		}

		cmd := exec.Command("curl", "-L", "-s", "-f", "-o", "/usr/bin/badvpn", binURL)
		if err := cmd.Run(); err != nil {
			// Si falla la descarga, intentar compilar
			return installBadVPNFromSource(port)
		}
		os.Chmod("/usr/bin/badvpn", 0755)
	}

	// 3. Servicio Systemd con multi-puerto (7100, 7200, 7300)
	svc := `[Unit]
Description=BadVPN UDP Gateway (Multi-Port)
After=network.target

[Service]
ExecStart=/usr/bin/badvpn --listen-addr 127.0.0.1:7100 --listen-addr 127.0.0.1:7200 --listen-addr 127.0.0.1:7300 --max-clients 500
User=root
Restart=always
RestartSec=3

[Install]
WantedBy=multi-user.target`

	svcFile := "/etc/systemd/system/badvpn.service"
	if err := os.WriteFile(svcFile, []byte(svc), 0644); err != nil {
		return fmt.Errorf("fallo escribir badvpn.service: %v", err)
	}

	exec.Command("systemctl", "daemon-reload").Run()
	exec.Command("systemctl", "enable", "badvpn.service").Run()
	if err := exec.Command("systemctl", "restart", "badvpn.service").Run(); err != nil {
		return fmt.Errorf("fallo reiniciar badvpn.service: %v", err)
	}

	// 4. Verificación
	time.Sleep(1500 * time.Millisecond)
	if err := exec.Command("systemctl", "is-active", "--quiet", "badvpn.service").Run(); err != nil {
		logCmd, _ := exec.Command("journalctl", "-u", "badvpn.service", "--no-pager", "-n", "10").Output()
		logs := string(logCmd)
		if logs == "" {
			logs = "No se pudieron obtener logs."
		}

		_ = exec.Command("systemctl", "stop", "badvpn.service").Run()
		_ = os.Remove(svcFile)
		_ = exec.Command("systemctl", "daemon-reload").Run()
		return fmt.Errorf("badvpn no pudo mantenerse activo.\n\n📝 <b>LOGS:</b>\n<pre>%s</pre>", logs)
	}

	return nil
}

// installBadVPNFromSource compila badvpn desde el repositorio oficial (fallback)
func installBadVPNFromSource(port string) error {
	deps := []string{"cmake", "g++", "make", "screen", "git", "build-essential", "libssl-dev", "libnspr4-dev", "libnss3-dev", "pkg-config"}
	for _, dep := range deps {
		_ = exec.Command("apt-get", "install", "-y", dep).Run()
	}

	buildDir := "/tmp/badvpn_build"
	os.RemoveAll(buildDir)

	if err := exec.Command("git", "clone", "https://github.com/ambrop72/badvpn.git", buildDir).Run(); err != nil {
		return fmt.Errorf("fallo clonar badvpn: %v", err)
	}

	cmdCmake := exec.Command("cmake", ".")
	cmdCmake.Dir = buildDir
	if err := cmdCmake.Run(); err != nil {
		return fmt.Errorf("fallo cmake: %v", err)
	}

	cmdMake := exec.Command("make")
	cmdMake.Dir = buildDir
	if err := cmdMake.Run(); err != nil {
		return fmt.Errorf("fallo make: %v", err)
	}

	if err := exec.Command("cp", buildDir+"/udpgw/badvpn-udpgw", "/usr/bin/badvpn").Run(); err != nil {
		return fmt.Errorf("fallo copiar binario badvpn: %v", err)
	}
	os.Chmod("/usr/bin/badvpn", 0755)
	os.RemoveAll(buildDir)

	// Llamar recursivamente para crear el servicio
	return InstallBadVPN(port)
}

// RemoveBadVPN detiene y elimina badvpn
func RemoveBadVPN() error {
	exec.Command("systemctl", "stop", "badvpn.service").Run()
	exec.Command("systemctl", "disable", "badvpn.service").Run()
	os.Remove("/etc/systemd/system/badvpn.service")
	os.Remove("/usr/bin/badvpn")
	exec.Command("systemctl", "daemon-reload").Run()
	return nil
}
