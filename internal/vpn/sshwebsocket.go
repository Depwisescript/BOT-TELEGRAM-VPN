package vpn

import (
	"fmt"
	"os"
	"os/exec"
	"time"
)

const (
	wsProxyScript = "/usr/local/bin/ssh-ws-proxy.py"
	wsCertDir     = "/etc/ssh-ws/certs"
	wsServiceWS   = "ssh-ws"
	wsServiceWSS  = "ssh-wss"
)

// InstallSSHWebSocket installs the SSH WebSocket proxy (WS on port 80, WSS on port 443).
func InstallSSHWebSocket() error {
	// 1. Install dependencies
	if err := exec.Command("apt-get", "update", "-qq").Run(); err != nil {
		return fmt.Errorf("fallo apt update: %v", err)
	}
	for _, dep := range []string{"python3", "openssl", "openssh-server"} {
		if err := exec.Command("apt-get", "install", "-y", "-qq", dep).Run(); err != nil {
			return fmt.Errorf("fallo install %s: %v", dep, err)
		}
	}

	// 2. Generate SSL certificate
	os.MkdirAll(wsCertDir, 0755)
	certFile := wsCertDir + "/cert.pem"
	keyFile := wsCertDir + "/key.pem"
	if _, err := os.Stat(certFile); os.IsNotExist(err) {
		cmd := exec.Command("openssl", "req", "-x509", "-newkey", "rsa:2048",
			"-keyout", keyFile, "-out", certFile,
			"-days", "3650", "-nodes",
			"-subj", "/C=US/ST=Cloud/L=VPS/O=SSH-WS/CN=ssh-websocket")
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("fallo generar certificado SSL: %v", err)
		}
	}

	// 3. Create Python proxy script (Raw TCP pipe - compatible with HTTP Injector/Custom)
	proxyCode := `#!/usr/bin/env python3
"""SSH WebSocket Proxy v2.0 (Raw TCP) - compatible with HTTP Injector, HTTP Custom, HA Tunnel"""
import asyncio, sys, ssl, signal, os

BUFFER_SIZE = 65536
SSH_HOST = "127.0.0.1"
SSH_PORT = 22

RESPONSE_101 = (
    b"HTTP/1.1 101 Switching Protocols\r\n"
    b"Upgrade: websocket\r\n"
    b"Connection: Upgrade\r\n"
    b"\r\n"
)
RESPONSE_200 = b"HTTP/1.1 200 Connection established\r\n\r\n"

active = 0

async def pipe(r, w):
    try:
        while True:
            d = await r.read(BUFFER_SIZE)
            if not d: break
            w.write(d)
            await w.drain()
    except: pass
    finally:
        try: w.close()
        except: pass

async def handle(cr, cw):
    global active
    active += 1
    ip = "?"
    try:
        p = cw.get_extra_info("peername")
        if p: ip = p[0]
    except: pass
    sw = None
    try:
        try:
            payload = await asyncio.wait_for(cr.read(BUFFER_SIZE), timeout=10)
        except asyncio.TimeoutError:
            cw.close(); active -= 1; return
        if not payload:
            cw.close(); active -= 1; return
        ps = payload.decode("utf-8", errors="ignore").upper()
        if "UPGRADE" in ps or "WEBSOCKET" in ps:
            cw.write(RESPONSE_101)
        else:
            cw.write(RESPONSE_200)
        await cw.drain()
        try:
            sr, sw = await asyncio.open_connection(SSH_HOST, SSH_PORT)
        except Exception as e:
            print(f"[!] SSH error: {e}")
            cw.close(); active -= 1; return
        print(f"[+] {ip} -> SSH ({active})")
        await asyncio.gather(pipe(cr, sw), pipe(sr, cw))
    except: pass
    finally:
        active -= 1
        print(f"[-] {ip} ({active})")
        try: cw.close()
        except: pass
        if sw:
            try: sw.close()
            except: pass

async def start(port, ctx=None):
    m = "WSS" if ctx else "WS"
    srv = await asyncio.start_server(handle, "0.0.0.0", port, ssl=ctx)
    print(f"[*] SSH-WS Proxy ({m}) -> :{port}")
    async with srv: await srv.serve_forever()

def main():
    port = int(sys.argv[1])
    ctx = None
    if len(sys.argv) >= 3:
        cd = sys.argv[2]
        ctx = ssl.SSLContext(ssl.PROTOCOL_TLS_SERVER)
        ctx.load_cert_chain(os.path.join(cd,"cert.pem"), os.path.join(cd,"key.pem"))
    loop = asyncio.new_event_loop()
    asyncio.set_event_loop(loop)
    for s in (signal.SIGTERM, signal.SIGINT):
        try: loop.add_signal_handler(s, lambda: loop.stop())
        except: pass
    try: loop.run_until_complete(start(port, ctx))
    except KeyboardInterrupt: pass
    finally: loop.close()

if __name__ == "__main__": main()
`
	if err := os.WriteFile(wsProxyScript, []byte(proxyCode), 0755); err != nil {
		return fmt.Errorf("fallo escribir proxy script: %v", err)
	}

	// 4. Create systemd services
	svcWS := `[Unit]
Description=SSH WebSocket Proxy (WS Puerto 80)
After=network.target sshd.service
Wants=sshd.service

[Service]
Type=simple
ExecStart=/usr/bin/python3 ` + wsProxyScript + ` 80
Restart=always
RestartSec=3
StandardOutput=journal
StandardError=journal

[Install]
WantedBy=multi-user.target`

	svcWSS := `[Unit]
Description=SSH WebSocket Proxy SSL (WSS Puerto 443)
After=network.target sshd.service
Wants=sshd.service

[Service]
Type=simple
ExecStart=/usr/bin/python3 ` + wsProxyScript + ` 443 ` + wsCertDir + `
Restart=always
RestartSec=3
StandardOutput=journal
StandardError=journal

[Install]
WantedBy=multi-user.target`

	if err := os.WriteFile("/etc/systemd/system/"+wsServiceWS+".service", []byte(svcWS), 0644); err != nil {
		return fmt.Errorf("fallo escribir ssh-ws.service: %v", err)
	}
	if err := os.WriteFile("/etc/systemd/system/"+wsServiceWSS+".service", []byte(svcWSS), 0644); err != nil {
		return fmt.Errorf("fallo escribir ssh-wss.service: %v", err)
	}

	// 5. Kill any existing process on ports 80/443 and start services
	exec.Command("bash", "-c", "fuser -k 80/tcp 2>/dev/null || true").Run()
	exec.Command("bash", "-c", "fuser -k 443/tcp 2>/dev/null || true").Run()
	time.Sleep(500 * time.Millisecond)

	exec.Command("systemctl", "daemon-reload").Run()
	exec.Command("systemctl", "enable", wsServiceWS+".service").Run()
	exec.Command("systemctl", "enable", wsServiceWSS+".service").Run()

	if err := exec.Command("systemctl", "restart", wsServiceWS+".service").Run(); err != nil {
		return fmt.Errorf("fallo iniciar ssh-ws: %v", err)
	}
	if err := exec.Command("systemctl", "restart", wsServiceWSS+".service").Run(); err != nil {
		return fmt.Errorf("fallo iniciar ssh-wss: %v", err)
	}

	// 6. Verify
	time.Sleep(2 * time.Second)
	wsOK := exec.Command("systemctl", "is-active", "--quiet", wsServiceWS+".service").Run() == nil
	wssOK := exec.Command("systemctl", "is-active", "--quiet", wsServiceWSS+".service").Run() == nil

	if !wsOK && !wssOK {
		logCmd, _ := exec.Command("journalctl", "-u", wsServiceWS+".service", "--no-pager", "-n", "10").Output()
		return fmt.Errorf("ningún servicio WebSocket pudo activarse.\n\n📝 <b>LOGS:</b>\n<pre>%s</pre>", string(logCmd))
	}

	return nil
}

// RemoveSSHWebSocket stops and removes SSH WebSocket services.
func RemoveSSHWebSocket() error {
	exec.Command("systemctl", "stop", wsServiceWS+".service").Run()
	exec.Command("systemctl", "stop", wsServiceWSS+".service").Run()
	exec.Command("systemctl", "disable", wsServiceWS+".service").Run()
	exec.Command("systemctl", "disable", wsServiceWSS+".service").Run()

	os.Remove("/etc/systemd/system/" + wsServiceWS + ".service")
	os.Remove("/etc/systemd/system/" + wsServiceWSS + ".service")
	os.Remove(wsProxyScript)
	os.RemoveAll("/etc/ssh-ws")

	exec.Command("systemctl", "daemon-reload").Run()
	return nil
}

// IsSSHWebSocketActive checks if at least one WS service is running.
func IsSSHWebSocketActive() (wsActive bool, wssActive bool) {
	wsActive = exec.Command("systemctl", "is-active", "--quiet", wsServiceWS+".service").Run() == nil
	wssActive = exec.Command("systemctl", "is-active", "--quiet", wsServiceWSS+".service").Run() == nil
	return
}
