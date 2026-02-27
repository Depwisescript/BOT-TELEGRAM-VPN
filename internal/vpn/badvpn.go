package vpn

import (
	"fmt"
	"os"
	"os/exec"
)

// InstallBadVPN installs badvpn-udpgw compiling it from the official GitHub repo using CMake.
func InstallBadVPN(port string) error {
	// 1. Install dependencies
	if err := exec.Command("apt-get", "update").Run(); err != nil {
		return fmt.Errorf("fallo apt update: %v", err)
	}
	deps := []string{"cmake", "g++", "make", "screen", "git", "build-essential", "libssl-dev", "libnspr4-dev", "libnss3-dev", "pkg-config"}
	for _, dep := range deps {
		if err := exec.Command("apt-get", "install", "-y", dep).Run(); err != nil {
			return fmt.Errorf("fallo install %s: %v", dep, err)
		}
	}

	// 2. Clone and Compile
	buildDir := "/tmp/badvpn_build"
	os.RemoveAll(buildDir)

	if _, err := os.Stat("/usr/bin/badvpn-udpgw"); os.IsNotExist(err) {
		if err := exec.Command("git", "clone", "https://github.com/ambrop72/badvpn.git", buildDir).Run(); err != nil {
			return fmt.Errorf("fallo clonar badvpn: %v", err)
		}

		cmdCmake := exec.Command("cmake", ".")
		cmdCmake.Dir = buildDir
		if err := cmdCmake.Run(); err != nil {
			return fmt.Errorf("fallo compilar cmake: %v", err)
		}

		cmdMake := exec.Command("make")
		cmdMake.Dir = buildDir
		if err := cmdMake.Run(); err != nil {
			return fmt.Errorf("fallo make: %v", err)
		}

		if err := exec.Command("cp", buildDir+"/udpgw/badvpn-udpgw", "/usr/bin/badvpn-udpgw").Run(); err != nil {
			return fmt.Errorf("fallo copiar binario badvpn: %v", err)
		}
		if err := os.Chmod("/usr/bin/badvpn-udpgw", 0755); err != nil {
			return fmt.Errorf("fallo chmod badvpn: %v", err)
		}
	}

	// 3. Service
	svc := `[Unit]
Description=BadVPN UDP Gateway
After=network.target

[Service]
ExecStart=/usr/bin/badvpn-udpgw --listen-addr 0.0.0.0:` + port + ` --max-clients 1000 --max-connections-for-client 8
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

	return nil
}

// RemoveBadVPN stops and removes badvpn
func RemoveBadVPN() error {
	exec.Command("systemctl", "stop", "badvpn.service").Run()
	exec.Command("systemctl", "disable", "badvpn.service").Run()
	os.Remove("/etc/systemd/system/badvpn.service")
	os.Remove("/usr/bin/badvpn-udpgw")
	exec.Command("systemctl", "daemon-reload").Run()
	return nil
}
