package vpn

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"time"
)

const (
	sshWsBin    = "/usr/bin/ssh-ws"
	sshWsProBin = "/usr/bin/ssh-ws-pro"
	sshWsSvc    = "ssh-ws"
	sshWsProSvc = "ssh-ws-pro"
)

// InstallSSHWebSocket instala los binarios nativos ssh-ws y ssh-ws-pro.
// ssh-ws: puerto 10015 → 127.0.0.1:22 (para HAProxy backend)
// ssh-ws-pro: puerto 2082 → 127.0.0.1:22
func InstallSSHWebSocket() error {
	// 1. Dependencias
	_ = exec.Command("apt-get", "update", "-qq").Run()
	_ = exec.Command("apt-get", "install", "-y", "-qq", "curl", "openssh-server").Run()

	arch := runtime.GOARCH

	// 2. Descargar binarios ssh-ws y ssh-ws-pro
	var sshWsURL, sshWsProURL string
	if arch == "amd64" {
		sshWsURL = "https://github.com/firewallfalcons/FirewallFalcon-Manager/raw/main/ws/ssh-ws-amd64"
		sshWsProURL = "https://github.com/firewallfalcons/FirewallFalcon-Manager/raw/main/ws/ssh-ws-pro-amd64"
	} else if arch == "arm64" {
		sshWsURL = "https://github.com/firewallfalcons/FirewallFalcon-Manager/raw/main/ws/ssh-ws-arm64"
		sshWsProURL = "https://github.com/firewallfalcons/FirewallFalcon-Manager/raw/main/ws/ssh-ws-pro-arm64"
	} else {
		// Fallback: usar script Python como última opción
		return installSSHWebSocketPython()
	}

	// Descargar ssh-ws
	if err := exec.Command("curl", "-L", "-s", "-f", "-o", sshWsBin, sshWsURL).Run(); err != nil {
		return installSSHWebSocketPython() // Fallback
	}
	os.Chmod(sshWsBin, 0755)

	// Descargar ssh-ws-pro
	if err := exec.Command("curl", "-L", "-s", "-f", "-o", sshWsProBin, sshWsProURL).Run(); err != nil {
		// Continuar sin ssh-ws-pro (no es crítico)
		fmt.Println("[WARN] No se pudo descargar ssh-ws-pro, continuando sin él")
	} else {
		os.Chmod(sshWsProBin, 0755)
	}

	// 3. Servicio ssh-ws (puerto 10015 → SSH)
	svcWS := `[Unit]
Description=SSH WebSocket Proxy (Puerto 10015)
After=network.target sshd.service
Wants=sshd.service

[Service]
Type=simple
ExecStart=` + sshWsBin + ` -b 127.0.0.1 -p 10015 -t 127.0.0.1:22 -l /var/log/ssh-ws.log
Restart=always
RestartSec=3
StandardOutput=journal
StandardError=journal

[Install]
WantedBy=multi-user.target`

	if err := os.WriteFile("/etc/systemd/system/"+sshWsSvc+".service", []byte(svcWS), 0644); err != nil {
		return fmt.Errorf("fallo escribir ssh-ws.service: %v", err)
	}

	// 4. Servicio ssh-ws-pro (puerto 2082 → SSH)
	if _, err := os.Stat(sshWsProBin); err == nil {
		svcWSPro := `[Unit]
Description=SSH WebSocket Pro Proxy (Puerto 2082)
After=network.target sshd.service
Wants=sshd.service

[Service]
Type=simple
ExecStart=` + sshWsProBin + ` -p 2082 -t 127.0.0.1:22
Restart=always
RestartSec=3
StandardOutput=journal
StandardError=journal

[Install]
WantedBy=multi-target.target`

		if err := os.WriteFile("/etc/systemd/system/"+sshWsProSvc+".service", []byte(svcWSPro), 0644); err != nil {
			return fmt.Errorf("fallo escribir ssh-ws-pro.service: %v", err)
		}
	}

	// 5. Iniciar servicios
	exec.Command("systemctl", "daemon-reload").Run()
	exec.Command("systemctl", "enable", sshWsSvc+".service").Run()
	if err := exec.Command("systemctl", "restart", sshWsSvc+".service").Run(); err != nil {
		return fmt.Errorf("fallo iniciar ssh-ws: %v", err)
	}

	// ssh-ws-pro es opcional
	if _, err := os.Stat(sshWsProBin); err == nil {
		exec.Command("systemctl", "enable", sshWsProSvc+".service").Run()
		_ = exec.Command("systemctl", "restart", sshWsProSvc+".service").Run()
	}

	// 6. Verificar
	time.Sleep(2 * time.Second)
	wsOK := exec.Command("systemctl", "is-active", "--quiet", sshWsSvc+".service").Run() == nil

	if !wsOK {
		logCmd, _ := exec.Command("journalctl", "-u", sshWsSvc+".service", "--no-pager", "-n", "10").Output()
		return fmt.Errorf("ssh-ws no pudo activarse.\n\n📝 <b>LOGS:</b>\n<pre>%s</pre>", string(logCmd))
	}

	return nil
}

// installSSHWebSocketPython es el fallback usando script Python (versión anterior)
func installSSHWebSocketPython() error {
	_ = exec.Command("apt-get", "install", "-y", "-qq", "python3", "openssl").Run()

	os.MkdirAll("/etc/ssh-ws/certs", 0755)
	certFile := "/etc/ssh-ws/certs/cert.pem"
	keyFile := "/etc/ssh-ws/certs/key.pem"

	if _, err := os.Stat(certFile); os.IsNotExist(err) {
		exec.Command("openssl", "req", "-x509", "-newkey", "rsa:2048",
			"-keyout", keyFile, "-out", certFile,
			"-days", "3650", "-nodes",
			"-subj", "/C=US/ST=Cloud/L=VPS/O=SSH-WS/CN=ssh-websocket").Run()
	}

	proxyCode := `#!/usr/bin/env python3
"""SSH WebSocket Proxy v2.0 (Raw TCP)"""
import asyncio, sys, ssl, signal, os
BUFFER_SIZE = 65536
SSH_HOST = "127.0.0.1"
SSH_PORT = 22
RESPONSE_101 = (b"HTTP/1.1 101 Switching Protocols\r\n"
    b"Upgrade: websocket\r\nConnection: Upgrade\r\n\r\n")
RESPONSE_200 = b"HTTP/1.1 200 Connection established\r\n\r\n"
active = 0
async def pipe(r, w):
    try:
        while True:
            d = await r.read(BUFFER_SIZE)
            if not d: break
            w.write(d); await w.drain()
    except: pass
    finally:
        try: w.close()
        except: pass
async def handle(cr, cw):
    global active; active += 1
    sw = None
    try:
        try: payload = await asyncio.wait_for(cr.read(BUFFER_SIZE), timeout=10)
        except asyncio.TimeoutError: cw.close(); active -= 1; return
        if not payload: cw.close(); active -= 1; return
        ps = payload.decode("utf-8", errors="ignore").upper()
        if "UPGRADE" in ps or "WEBSOCKET" in ps: cw.write(RESPONSE_101)
        else: cw.write(RESPONSE_200)
        await cw.drain()
        try: sr, sw = await asyncio.open_connection(SSH_HOST, SSH_PORT)
        except: cw.close(); active -= 1; return
        await asyncio.gather(pipe(cr, sw), pipe(sr, cw))
    except: pass
    finally:
        active -= 1
        try: cw.close()
        except: pass
        if sw:
            try: sw.close()
            except: pass
async def start(port, ctx=None):
    srv = await asyncio.start_server(handle, "0.0.0.0", port, ssl=ctx)
    async with srv: await srv.serve_forever()
def main():
    port = int(sys.argv[1]); ctx = None
    if len(sys.argv) >= 3:
        cd = sys.argv[2]; ctx = ssl.SSLContext(ssl.PROTOCOL_TLS_SERVER)
        ctx.load_cert_chain(os.path.join(cd,"cert.pem"), os.path.join(cd,"key.pem"))
    loop = asyncio.new_event_loop(); asyncio.set_event_loop(loop)
    try: loop.run_until_complete(start(port, ctx))
    except KeyboardInterrupt: pass
    finally: loop.close()
if __name__ == "__main__": main()
`
	proxyScript := "/usr/local/bin/ssh-ws-proxy.py"
	os.WriteFile(proxyScript, []byte(proxyCode), 0755)

	svcWS := `[Unit]
Description=SSH WebSocket Python Proxy (Puerto 80)
After=network.target sshd.service
[Service]
Type=simple
ExecStart=/usr/bin/python3 ` + proxyScript + ` 80
Restart=always
RestartSec=3
[Install]
WantedBy=multi-user.target`

	os.WriteFile("/etc/systemd/system/ssh-ws.service", []byte(svcWS), 0644)
	exec.Command("systemctl", "daemon-reload").Run()
	exec.Command("systemctl", "enable", "ssh-ws.service").Run()
	if err := exec.Command("systemctl", "restart", "ssh-ws.service").Run(); err != nil {
		return fmt.Errorf("fallo iniciar ssh-ws (python fallback): %v", err)
	}

	return nil
}

// RemoveSSHWebSocket detiene y elimina los servicios SSH WebSocket
func RemoveSSHWebSocket() error {
	exec.Command("systemctl", "stop", sshWsSvc+".service").Run()
	exec.Command("systemctl", "stop", sshWsProSvc+".service").Run()
	exec.Command("systemctl", "disable", sshWsSvc+".service").Run()
	exec.Command("systemctl", "disable", sshWsProSvc+".service").Run()

	os.Remove("/etc/systemd/system/" + sshWsSvc + ".service")
	os.Remove("/etc/systemd/system/" + sshWsProSvc + ".service")
	os.Remove(sshWsBin)
	os.Remove(sshWsProBin)
	os.Remove("/usr/local/bin/ssh-ws-proxy.py")
	os.RemoveAll("/etc/ssh-ws")

	exec.Command("systemctl", "daemon-reload").Run()
	return nil
}

// IsSSHWebSocketActive verifica si los servicios están activos
func IsSSHWebSocketActive() (wsActive bool, wssActive bool) {
	wsActive = exec.Command("systemctl", "is-active", "--quiet", sshWsSvc+".service").Run() == nil
	wssActive = exec.Command("systemctl", "is-active", "--quiet", sshWsProSvc+".service").Run() == nil
	return
}
