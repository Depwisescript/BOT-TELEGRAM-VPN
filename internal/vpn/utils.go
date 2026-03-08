package vpn

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
)

func installLibSSL11() {
	// Check if already exists
	if _, err := os.Stat("/usr/lib/x86_64-linux-gnu/libssl.so.1.1"); err == nil {
		return
	}
	if _, err := os.Stat("/usr/lib/aarch64-linux-gnu/libssl.so.1.1"); err == nil {
		return
	}

	arch := runtime.GOARCH
	var url string
	if arch == "amd64" {
		url = "http://nz2.archive.ubuntu.com/ubuntu/pool/main/o/openssl/libssl1.1_1.1.1f-1ubuntu2_amd64.deb"
	} else if arch == "arm64" || arch == "aarch64" {
		url = "http://ports.ubuntu.com/ubuntu-ports/pool/main/o/openssl/libssl1.1_1.1.1f-1ubuntu2_arm64.deb"
	}

	if url != "" {
		_ = exec.Command("wget", "-q", "-O", "/tmp/libssl1.1.deb", url).Run()
		_ = exec.Command("dpkg", "-i", "/tmp/libssl1.1.deb").Run()
		_ = os.Remove("/tmp/libssl1.1.deb")
	}
}

// GetSystemReport returns a diagnostic string about network and services
func GetSystemReport() string {
	report := "🛡️ <b>REPORTE TÉCNICO DE RED</b>\n\n"

	// 1. IPTables NAT (Prerouting)
	iptNat, _ := exec.Command("bash", "-c", "iptables -t nat -L PREROUTING -n -v | head -15").Output()
	report += "🔌 <b>Redirecciones (NAT):</b>\n<pre>" + string(iptNat) + "</pre>\n"

	// 2. Status Servicios
	svcs := []string{
		"badvpn.service",
		"udp-custom.service",
		"ssh-ws.service",
		"ssh-ws-pro.service",
		"haproxy.service",
		"dropbear_custom.service",
		"zivpn.service",
		"falconproxy.service",
	}
	report += "⚙️ <b>Estado Servicios:</b>\n"
	for _, s := range svcs {
		active, _ := exec.Command("systemctl", "is-active", s).Output()
		status := strings.TrimSpace(string(active))
		if status == "" {
			status = "no encontrado"
		}
		report += fmt.Sprintf("• %s: <code>%s</code>\n", s, status)
	}

	// 3. RAM
	free, _ := exec.Command("free", "-m").Output()
	report += "\n💾 <b>Memoria RAM (MB):</b>\n<pre>" + string(free) + "</pre>"

	return report
}
