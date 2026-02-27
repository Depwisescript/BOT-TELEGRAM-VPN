package vpn

import (
    "fmt"
    "os"
    "os/exec"
    "runtime"
    "strings"
)

// InstallSlowDNS replaces `instalar_slowdns()` downloading the dnstt-server binaries, generating keys and services.
func InstallSlowDNS(domain, port string) (string, error) {
    arch := runtime.GOARCH
    binName := "dnstt-server-linux-" + arch
    if arch == "386" {
        binName = "dnstt-server-linux-386"
    }

    mirrors := []string{
        "https://dnstt.network/" + binName,
        "https://github.com/bugfloyd/dnstt-deploy/raw/main/bin/" + binName,
        "https://raw.githubusercontent.com/Dan3651/scripts/main/slowdns-server",
    }

    // Attempt Download
    success := false
    for _, url := range mirrors {
        cmd := exec.Command("curl", "-L", "-k", "-s", "-f", "-o", "/usr/bin/slowdns-server", url)
        if err := cmd.Run(); err == nil {
            success = true
            break
        }
    }

    if !success {
        return "", fmt.Errorf("fallo al descargar binario para %s", arch)
    }
    os.Chmod("/usr/bin/slowdns-server", 0755)

    // Key Generation
    os.MkdirAll("/etc/slowdns", 0755)
    if _, err := os.Stat("/etc/slowdns/server.pub"); os.IsNotExist(err) {
        exec.Command("/usr/bin/slowdns-server", "-gen-key", "-privkey-file", "/etc/slowdns/server.key", "-pubkey-file", "/etc/slowdns/server.pub").Run()
    }

    // iptables
    exec.Command("iptables", "-t", "nat", "-D", "PREROUTING", "-p", "udp", "--dport", "53", "-j", "REDIRECT", "--to-ports", "5300").Run()
    exec.Command("iptables", "-t", "nat", "-A", "PREROUTING", "-p", "udp", "--dport", "53", "-j", "REDIRECT", "--to-ports", "5300").Run()

    // Service Creation
    svc := `[Unit]
Description=SlowDNS Depwise Service
After=network.target

[Service]
Type=simple
User=root
WorkingDirectory=/etc/slowdns
ExecStart=/usr/bin/slowdns-server -udp :5300 -privkey-file /etc/slowdns/server.key ` + domain + ` 127.0.0.1:` + port + `
Restart=always
RestartSec=3

[Install]
WantedBy=multi-user.target`

    os.WriteFile("/etc/systemd/system/slowdns.service", []byte(svc), 0644)
    exec.Command("systemctl", "daemon-reload").Run()
    exec.Command("systemctl", "enable", "slowdns").Run()
    exec.Command("systemctl", "restart", "slowdns").Run()

    pubBytes, _ := os.ReadFile("/etc/slowdns/server.pub")
    return strings.TrimSpace(string(pubBytes)), nil
}

// RemoveSlowDNS stops and removes the slowdns daemon and iptables rules.
func RemoveSlowDNS() error {
    exec.Command("systemctl", "stop", "slowdns").Run()
    exec.Command("systemctl", "disable", "slowdns").Run()
    os.Remove("/etc/systemd/system/slowdns.service")
    exec.Command("systemctl", "daemon-reload").Run()
    
    // Loop removal iptables
    for {
        err := exec.Command("iptables", "-t", "nat", "-D", "PREROUTING", "-p", "udp", "--dport", "53", "-j", "REDIRECT", "--to-ports", "5300").Run()
        if err != nil {
            break
        }
    }
    os.RemoveAll("/etc/slowdns")
    return nil
}
