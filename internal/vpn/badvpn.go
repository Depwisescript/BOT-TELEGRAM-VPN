package vpn

import (
	"fmt"
	"os"
	"os/exec"
	"time"
)

// badvpnPorts son los puertos donde escucha BadVPN (como en producción)
var badvpnPorts = []string{"7100", "7200", "7300"}

// badvpnBin es la ruta del binario custom de BadVPN (soporta multi-listen-addr y RakNet/Minecraft)
const badvpnBin = "/usr/bin/badvpn"

// badvpnDownloadURL es la URL del binario custom hospedado en el repo
const badvpnDownloadURL = "https://github.com/Depwisescript/BOT-TELEGRAM-VPN/raw/main/bin/badvpn"

// InstallBadVPN descarga el binario custom de badvpn y lo configura
// en múltiples puertos (7100, 7200, 7300) con un solo servicio.
// Este binario soporta multi-listen-addr y maneja mejor juegos como Minecraft Bedrock.
func InstallBadVPN(port string) error {
	// 1. Dependencias
	_ = exec.Command("apt-get", "update").Run()
	_ = exec.Command("apt-get", "install", "-y", "curl", "screen").Run()

	// 2. Descargar binario custom desde el repo
	if _, err := os.Stat(badvpnBin); os.IsNotExist(err) {
		cmd := exec.Command("curl", "-L", "-s", "-f", "-o", badvpnBin, badvpnDownloadURL)
		if err := cmd.Run(); err != nil {
			// Fallback: intentar el estándar badvpn-udpgw
			return installBadVPNFallback()
		}
		os.Chmod(badvpnBin, 0755)
	}

	// 3. Verificar que el binario es ejecutable
	if err := exec.Command(badvpnBin, "--help").Run(); err != nil {
		// Si falla, puede ser arquitectura incorrecta, intentar fallback
		os.Remove(badvpnBin)
		return installBadVPNFallback()
	}

	// 4. Limpiar servicios viejos (por-puerto)
	for _, p := range badvpnPorts {
		exec.Command("systemctl", "stop", "badvpn-"+p+".service").Run()
		exec.Command("systemctl", "disable", "badvpn-"+p+".service").Run()
		os.Remove("/etc/systemd/system/badvpn-" + p + ".service")
	}

	// 5. Crear servicio único con multi-listen-addr (como el servidor de producción)
	svc := `[Unit]
Description=BadVPN UDP Gateway (Multi-Port)
Documentation=https://t.me/gerhanatunnel
After=syslog.target network-online.target

[Service]
User=root
NoNewPrivileges=true
ExecStart=/usr/bin/badvpn --listen-addr 127.0.0.1:7100 --listen-addr 127.0.0.1:7200 --listen-addr 127.0.0.1:7300 --max-clients 500
Restart=on-failure
RestartPreventExitStatus=23
LimitNPROC=10000
LimitNOFILE=1000000

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

	// 6. Verificación
	time.Sleep(2 * time.Second)
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

// installBadVPNFallback usa servicios separados con badvpn-udpgw estándar
func installBadVPNFallback() error {
	// Descargar badvpn-udpgw estándar
	binURL := "https://github.com/firewallfalcons/FirewallFalcon-Manager/raw/main/udp/badvpn-udpgw64"
	stdBin := "/usr/bin/badvpn-udpgw"

	if _, err := os.Stat(stdBin); os.IsNotExist(err) {
		if err := exec.Command("curl", "-L", "-s", "-f", "-o", stdBin, binURL).Run(); err != nil {
			return fmt.Errorf("fallo descargar badvpn-udpgw: %v", err)
		}
		os.Chmod(stdBin, 0755)
	}

	// Crear servicios separados (un servicio por puerto)
	for _, p := range badvpnPorts {
		svcName := "badvpn-" + p
		svc := `[Unit]
Description=BadVPN UDP Gateway (Puerto ` + p + `)
After=network.target

[Service]
ExecStart=` + stdBin + ` --listen-addr 127.0.0.1:` + p + ` --max-clients 500
User=root
Restart=always
RestartSec=3

[Install]
WantedBy=multi-user.target`

		svcFile := "/etc/systemd/system/" + svcName + ".service"
		os.WriteFile(svcFile, []byte(svc), 0644)
	}

	exec.Command("systemctl", "daemon-reload").Run()
	for _, p := range badvpnPorts {
		svcName := "badvpn-" + p
		exec.Command("systemctl", "enable", svcName+".service").Run()
		exec.Command("systemctl", "restart", svcName+".service").Run()
	}

	time.Sleep(2 * time.Second)

	activeCount := 0
	for _, p := range badvpnPorts {
		if exec.Command("systemctl", "is-active", "--quiet", "badvpn-"+p+".service").Run() == nil {
			activeCount++
		}
	}

	if activeCount == 0 {
		return fmt.Errorf("ningún servicio badvpn pudo mantenerse activo")
	}

	return nil
}

// RemoveBadVPN detiene y elimina todos los servicios badvpn
func RemoveBadVPN() error {
	// Servicio único (custom binary)
	exec.Command("systemctl", "stop", "badvpn.service").Run()
	exec.Command("systemctl", "disable", "badvpn.service").Run()
	os.Remove("/etc/systemd/system/badvpn.service")

	// Servicios por-puerto (fallback)
	for _, p := range badvpnPorts {
		svcName := "badvpn-" + p
		exec.Command("systemctl", "stop", svcName+".service").Run()
		exec.Command("systemctl", "disable", svcName+".service").Run()
		os.Remove("/etc/systemd/system/" + svcName + ".service")
	}

	os.Remove("/usr/bin/badvpn")
	os.Remove("/usr/bin/badvpn-udpgw")
	exec.Command("systemctl", "daemon-reload").Run()
	return nil
}
