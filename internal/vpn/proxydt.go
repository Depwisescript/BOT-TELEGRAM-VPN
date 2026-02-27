package vpn

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

// InstallProxyDT download the cracked proxydt binary based on architecture.
func InstallProxyDT() error {
	// 0. Install system dependencies
	_ = exec.Command("apt-get", "update").Run()
	_ = exec.Command("apt-get", "install", "-y", "curl", "psmisc").Run()

	arch := runtime.GOARCH

	mirrors := []string{
		"https://raw.githubusercontent.com/Depwisescript/PROXY-DT/928bb1af4211b874361bc65c210189a5922ccaa8/DT%201.2.3/x86/proxy",
		"https://raw.githubusercontent.com/Depwisescript/PROXY-DT/928bb1af4211b874361bc65c210189a5922ccaa8/DT%201.2.3/proxydt",
	}

	// Prioritize arm64 if architecture matches
	if arch == "arm64" || arch == "aarch64" {
		mirrors = append([]string{"https://raw.githubusercontent.com/Depwisescript/PROXY-DT/928bb1af4211b874361bc65c210189a5922ccaa8/DT%201.2.3/arm64/proxy"}, mirrors...)
	}

	os.Remove("/usr/bin/proxydt")
	os.Remove("/usr/bin/proxy")

	var lastErr error
	success := false
	for _, url := range mirrors {
		// Use -f to return error code on 404/500
		cmd := exec.Command("curl", "-L", "-s", "-f", "-o", "/usr/bin/proxydt", url)
		if err := cmd.Run(); err == nil {
			success = true
			break
		} else {
			lastErr = err
		}
	}

	if !success {
		return fmt.Errorf("fallo la descarga de todos los mirrors de ProxyDT: %v", lastErr)
	}

	if err := os.Chmod("/usr/bin/proxydt", 0755); err != nil {
		return fmt.Errorf("error al dar permisos a proxydt: %v", err)
	}

	// Create symlink
	_ = os.Remove("/usr/bin/proxy")
	if err := os.Symlink("/usr/bin/proxydt", "/usr/bin/proxy"); err != nil {
		// Not critical if symlink fails but log it maybe?
	}
	return nil
}

// OpenProxyDTPort creates and starts a systemd service running ProxyDT.
func OpenProxyDTPort(port string) error {
	// 1. Ensure binary exists
	if _, err := os.Stat("/usr/bin/proxydt"); os.IsNotExist(err) {
		if err := InstallProxyDT(); err != nil {
			return err
		}
	}

	// 2. Kill legacy processes on this port
	_ = exec.Command("fuser", "-k", "-n", "tcp", port).Run()

	svcName := "proxydt-" + port
	svcFile := "/etc/systemd/system/" + svcName + ".service"

	svc := `[Unit]
Description=ProxyDT (Cracked) on port ` + port + `
After=network.target

[Service]
Type=simple
ExecStart=/usr/bin/proxydt --port ` + port + ` --response SSHTFREE
Restart=always
RestartSec=3

[Install]
WantedBy=multi-user.target`

	if err := os.WriteFile(svcFile, []byte(svc), 0644); err != nil {
		return fmt.Errorf("error al escribir servicio %s: %v", svcName, err)
	}

	_ = exec.Command("systemctl", "daemon-reload").Run()
	_ = exec.Command("systemctl", "enable", svcName).Run()

	if err := exec.Command("systemctl", "restart", svcName).Run(); err != nil {
		return fmt.Errorf("error al iniciar servicio %s: %v", svcName, err)
	}

	// 3. Simple verification
	time.Sleep(1 * time.Second)
	if err := exec.Command("systemctl", "is-active", "--quiet", svcName).Run(); err != nil {
		_ = exec.Command("systemctl", "stop", svcName).Run()
		_ = os.Remove(svcFile)
		_ = exec.Command("systemctl", "daemon-reload").Run()
		return fmt.Errorf("el servicio ProxyDT no pudo mantenerse activo en el puerto %s", port)
	}

	return nil
}

// CloseProxyDTPort stops and removes a proxy service running on a given port.
func CloseProxyDTPort(port string) error {
	svcName := "proxydt-" + port
	exec.Command("systemctl", "stop", svcName).Run()
	exec.Command("systemctl", "disable", svcName).Run()
	os.Remove("/etc/systemd/system/" + svcName + ".service")
	exec.Command("systemctl", "daemon-reload").Run()
	return nil
}

// RemoveProxyDT stops every running ProxyDT ports and uninstalls to binary.
func RemoveProxyDT() error {
	out, _ := exec.Command("bash", "-c", "systemctl list-units --all | grep proxydt- | awk '{print $1}'").Output()
	services := strings.Split(strings.TrimSpace(string(out)), "\n")

	for _, svc := range services {
		if svc != "" {
			exec.Command("systemctl", "stop", svc).Run()
			exec.Command("systemctl", "disable", svc).Run()
			os.Remove("/etc/systemd/system/" + svc)
		}
	}

	os.Remove("/usr/bin/proxydt")
	exec.Command("systemctl", "daemon-reload").Run()
	return nil
}
