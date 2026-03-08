package vpn

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"time"
)

// badvpnPorts son los puertos donde escucha BadVPN (como en producción)
var badvpnPorts = []string{"7100", "7200", "7300"}

// InstallBadVPN descarga el binario precompilado de badvpn-udpgw y lo configura
// en múltiples puertos (7100, 7200, 7300) usando servicios separados.
func InstallBadVPN(port string) error {
	// 1. Dependencias
	_ = exec.Command("apt-get", "update").Run()
	_ = exec.Command("apt-get", "install", "-y", "curl", "screen").Run()

	// 2. Descargar binario precompilado según arquitectura
	if _, err := os.Stat("/usr/bin/badvpn-udpgw"); os.IsNotExist(err) {
		arch := runtime.GOARCH
		var binURL string

		if arch == "amd64" {
			binURL = "https://github.com/firewallfalcons/FirewallFalcon-Manager/raw/main/udp/badvpn-udpgw64"
		} else if arch == "arm64" {
			binURL = "https://github.com/firewallfalcons/FirewallFalcon-Manager/raw/main/udp/badvpn-udpgw-arm64"
		} else {
			return installBadVPNFromSource(port)
		}

		cmd := exec.Command("curl", "-L", "-s", "-f", "-o", "/usr/bin/badvpn-udpgw", binURL)
		if err := cmd.Run(); err != nil {
			return installBadVPNFromSource(port)
		}
		os.Chmod("/usr/bin/badvpn-udpgw", 0755)
	}

	// 3. Crear un servicio systemd por cada puerto
	for _, p := range badvpnPorts {
		svcName := "badvpn-" + p
		svc := `[Unit]
Description=BadVPN UDP Gateway (Puerto ` + p + `)
After=network.target

[Service]
ExecStart=/usr/bin/badvpn-udpgw --listen-addr 127.0.0.1:` + p + ` --max-clients 500 --max-connections-for-client 8
User=root
Restart=always
RestartSec=3

[Install]
WantedBy=multi-user.target`

		svcFile := "/etc/systemd/system/" + svcName + ".service"
		if err := os.WriteFile(svcFile, []byte(svc), 0644); err != nil {
			return fmt.Errorf("fallo escribir %s: %v", svcName, err)
		}

		exec.Command("systemctl", "daemon-reload").Run()
		exec.Command("systemctl", "enable", svcName+".service").Run()
		if err := exec.Command("systemctl", "restart", svcName+".service").Run(); err != nil {
			// se verificará después
			_ = err
		}
	}

	// Limpiar servicio viejo (badvpn.service sin puerto) si existe
	exec.Command("systemctl", "stop", "badvpn.service").Run()
	exec.Command("systemctl", "disable", "badvpn.service").Run()
	os.Remove("/etc/systemd/system/badvpn.service")
	exec.Command("systemctl", "daemon-reload").Run()

	// 4. Verificación
	time.Sleep(1500 * time.Millisecond)
	activeCount := 0
	var failedPorts []string
	for _, p := range badvpnPorts {
		svcName := "badvpn-" + p
		if exec.Command("systemctl", "is-active", "--quiet", svcName+".service").Run() == nil {
			activeCount++
		} else {
			failedPorts = append(failedPorts, p)
		}
	}

	if activeCount == 0 {
		logCmd, _ := exec.Command("journalctl", "-u", "badvpn-7300.service", "--no-pager", "-n", "10").Output()
		logs := string(logCmd)
		if logs == "" {
			logs = "No se pudieron obtener logs."
		}

		// Limpiar servicios fallidos
		for _, p := range badvpnPorts {
			svcName := "badvpn-" + p
			_ = exec.Command("systemctl", "stop", svcName+".service").Run()
			_ = os.Remove("/etc/systemd/system/" + svcName + ".service")
		}
		_ = exec.Command("systemctl", "daemon-reload").Run()
		return fmt.Errorf("badvpn no pudo mantenerse activo.\n\n📝 <b>LOGS:</b>\n<pre>%s</pre>", logs)
	}

	if len(failedPorts) > 0 {
		// Algunos puertos fallaron pero al menos uno funciona
		fmt.Printf("[WARN] BadVPN: puertos fallidos: %v\n", failedPorts)
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

	if err := exec.Command("cp", buildDir+"/udpgw/badvpn-udpgw", "/usr/bin/badvpn-udpgw").Run(); err != nil {
		return fmt.Errorf("fallo copiar binario badvpn: %v", err)
	}
	os.Chmod("/usr/bin/badvpn-udpgw", 0755)
	os.RemoveAll(buildDir)

	return InstallBadVPN(port)
}

// RemoveBadVPN detiene y elimina todos los servicios badvpn
func RemoveBadVPN() error {
	// Limpiar servicios individuales por puerto
	for _, p := range badvpnPorts {
		svcName := "badvpn-" + p
		exec.Command("systemctl", "stop", svcName+".service").Run()
		exec.Command("systemctl", "disable", svcName+".service").Run()
		os.Remove("/etc/systemd/system/" + svcName + ".service")
	}

	// Limpiar servicio viejo por si existe
	exec.Command("systemctl", "stop", "badvpn.service").Run()
	exec.Command("systemctl", "disable", "badvpn.service").Run()
	os.Remove("/etc/systemd/system/badvpn.service")

	os.Remove("/usr/bin/badvpn-udpgw")
	exec.Command("systemctl", "daemon-reload").Run()
	return nil
}
