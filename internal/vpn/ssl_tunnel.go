package vpn

import (
	"fmt"
	"os"
	"os/exec"
)

// InstallSSLTunnel instala HAProxy y lo configura como tunel SSL
func InstallSSLTunnel(port string) error {
	// 1. Instalar HAProxy
	exec.Command("apt-get", "update").Run()
	if err := exec.Command("apt-get", "install", "-y", "haproxy").Run(); err != nil {
		return fmt.Errorf("fallo instalacion haproxy: %v", err)
	}

	certFile := "/etc/haproxy/haproxy.pem"
	configFile := "/etc/haproxy/haproxy.cfg"

	// 2. Generar Certificado si no existe
	if _, err := os.Stat(certFile); os.IsNotExist(err) {
		cmdCert := exec.Command("openssl", "req", "-x509", "-newkey", "rsa:2048", "-nodes", "-days", "3650",
			"-keyout", certFile, "-out", certFile, "-subj", "/CN=ssl-tunnel")
		cmdCert.Run()
	}

	// 3. Configuración
	config := fmt.Sprintf(`global
    log /dev/log    local0
    log /dev/log    local1 notice
    chroot /var/lib/haproxy
    stats socket /run/haproxy/admin.sock mode 660 level admin expose-fd listeners
    stats timeout 30s
    user haproxy
    group haproxy
    daemon
defaults
    log     global
    mode    tcp
    option  tcplog
    option  dontlognull
    timeout connect 5000
    timeout client  50000
    timeout server  50000
frontend ssh_ssl_in
    bind *:%s ssl crt %s
    mode tcp
    default_backend ssh_backend
backend ssh_backend
    mode tcp
    server ssh_server 127.0.0.1:22
`, port, certFile)

	os.WriteFile(configFile, []byte(config), 0644)

	// 4. Reiniciar
	exec.Command("systemctl", "daemon-reload").Run()
	if err := exec.Command("systemctl", "restart", "haproxy").Run(); err != nil {
		return fmt.Errorf("fallo reinicio haproxy: %v", err)
	}

	return nil
}

// RemoveSSLTunnel detiene y elimina HAProxy (o al menos la config)
func RemoveSSLTunnel() error {
	exec.Command("systemctl", "stop", "haproxy").Run()
	os.Remove("/etc/haproxy/haproxy.cfg")
	os.Remove("/etc/haproxy/haproxy.pem")
	return nil
}
