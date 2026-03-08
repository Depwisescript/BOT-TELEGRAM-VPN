package vpn

import (
	"fmt"
	"os"
	"os/exec"
)

// haproxyCfg es la configuración multi-protocolo de HAProxy.
// websocket_backend y default apuntan a ssh-ws en 10015 para soporte de juegos.
const haproxyCfg = `global
    stats socket /run/haproxy/admin.sock mode 660 level admin expose-fd listeners
    stats timeout 1d

    tune.bufsize 10485760
    tune.maxrewrite 3072
    tune.ssl.default-dh-param 2048

    pidfile /run/haproxy.pid
    chroot /var/lib/haproxy

    user haproxy
    group haproxy
    daemon

    ssl-default-bind-ciphers ECDHE-ECDSA-AES128-GCM-SHA256:ECDHE-RSA-AES128-GCM-SHA256:ECDHE-ECDSA-AES256-GCM-SHA384:ECDHE-RSA-AES256-GCM-SHA384:ECDHE-ECDSA-CHACHA20-POLY1305:ECDHE-RSA-CHACHA20-POLY1305:DHE-RSA-AES128-GCM-SHA256:DHE-RSA-AES256-GCM-SHA384
    ssl-default-bind-ciphersuites TLS_AES_128_GCM_SHA256:TLS_AES_256_GCM_SHA384:TLS_CHACHA20_POLY1305_SHA256
    ssl-default-bind-options no-sslv3 no-tlsv10 no-tlsv11

    ca-base /etc/ssl/certs
    crt-base /etc/ssl/private

defaults
    log global
    mode tcp
    option dontlognull
    option tcp-smart-connect
    timeout connect 5s
    timeout client  24h
    timeout server  24h

frontend multiport_frontend
    mode tcp
    bind *:443 tfo
    tcp-request inspect-delay 10ms
    tcp-request content accept if HTTP
    tcp-request content accept if { req.ssl_hello_type 1 }
    use_backend recir_http_backend if HTTP
    default_backend recir_https_backend

backend recir_https_backend
    mode tcp
    server recir_https_server abns@haproxy-https send-proxy-v2 check

backend recir_http_backend
    mode tcp
    server recir_http_server abns@haproxy-http send-proxy-v2 check

frontend multiports_frontend
    mode tcp
    bind abns@haproxy-http accept-proxy tfo
    default_backend recir_https_www_backend

backend recir_https_www_backend
    mode tcp
    server recir_https_www_server 127.0.0.1:2223 check

frontend ssl_frontend
    mode tcp
    bind *:80 tfo
    bind *:8080 tfo
    bind abns@haproxy-https accept-proxy ssl crt /etc/haproxy/yha.pem alpn h2,http/1.1 tfo

    tcp-request inspect-delay 200ms
    tcp-request content capture req.ssl_sni len 100
    tcp-request content accept if { req.ssl_hello_type 1 }

    acl acl_upgrade hdr(Connection) -i upgrade
    acl acl_websocket hdr(Upgrade) -i websocket
    acl acl_payload payload(0,7) -m bin 5353482d322e30
    acl acl_http2 ssl_fc_alpn -i h2
    acl acl_path_regex path_reg -i ^\/(.*)
    acl acl_path_vless path_reg -i ^\/vless.*
    acl acl_path_vmess path_reg -i ^\/vmess.*
    acl acl_path_trojan path_reg -i ^\/trojan-ws.*
    acl acl_path_grpc path_reg -i ^\/(vmess-grpc|trojan-grpc|ss-grpc).*
    acl acl_path_ssh path_reg -i ^\/fightertunnelssh.*

    use_backend grpc_backend if acl_http2
    use_backend websocket_backend if acl_upgrade acl_websocket
    use_backend websocket_backend if acl_path_regex
    use_backend bot_ftvpn_backend if acl_payload
    use_backend payload_backend if acl_path_vless
    use_backend payload_backend if acl_path_vmess
    use_backend payload_backend if acl_path_trojan
    use_backend payload_backend if acl_path_grpc
    use_backend ssh_backend if acl_path_ssh
    default_backend ssh_ws_default_backend

backend websocket_backend
    mode tcp
    server ssh_ws_server 127.0.0.1:10015 check

backend grpc_backend
    mode tcp
    server grpc_server 127.0.0.1:1013 check

backend ssh_ws_default_backend
    mode tcp
    balance roundrobin
    server ssh_ws_server 127.0.0.1:10015 check

backend bot_ftvpn_backend
    mode tcp
    server bot_ftvpn_server 127.0.0.1:2222 check

backend payload_backend
    mode tcp
    balance roundrobin
    server payload_server_vless 127.0.0.1:10001 check
    server payload_server_vmess 127.0.0.1:10002 check
    server payload_server_trojan 127.0.0.1:10003 check
    server payload_server_grpc 127.0.0.1:10004 check
    server payload_server_vless2 127.0.0.1:10005 check
    server payload_server_vmess2 127.0.0.1:10006 check
    server payload_server_trojan2 127.0.0.1:10007 check
    server payload_server_grpc2 127.0.0.1:10008 check
    server ssh_server 127.0.0.1:10015 check

backend ssh_backend
    mode tcp
    server ssh_server 127.0.0.1:10015 check
`

// InstallSSLTunnel instala HAProxy con configuración multi-protocolo.
// IMPORTANTE: Requiere que SSH WebSocket esté instalado en puerto 10015 primero.
func InstallSSLTunnel(port string) error {
	// 1. Instalar HAProxy
	exec.Command("apt-get", "update").Run()
	if err := exec.Command("apt-get", "install", "-y", "haproxy").Run(); err != nil {
		return fmt.Errorf("fallo instalacion haproxy: %v", err)
	}

	// 2. Crear directorio para el socket
	os.MkdirAll("/run/haproxy", 0755)

	certFile := "/etc/haproxy/yha.pem"
	configFile := "/etc/haproxy/haproxy.cfg"

	// 3. Generar Certificado PEM si no existe
	if _, err := os.Stat(certFile); os.IsNotExist(err) {
		cmdCert := exec.Command("openssl", "req", "-x509", "-newkey", "rsa:2048", "-nodes", "-days", "3650",
			"-keyout", "/tmp/haproxy_key.pem", "-out", "/tmp/haproxy_cert.pem",
			"-subj", "/CN=ssl-tunnel")
		if err := cmdCert.Run(); err != nil {
			return fmt.Errorf("fallo generar certificado: %v", err)
		}
		exec.Command("bash", "-c", "cat /tmp/haproxy_key.pem /tmp/haproxy_cert.pem > "+certFile).Run()
		os.Remove("/tmp/haproxy_key.pem")
		os.Remove("/tmp/haproxy_cert.pem")
	}

	// 4. Matar procesos en puertos que HAProxy necesita
	exec.Command("bash", "-c", "fuser -k 80/tcp 2>/dev/null || true").Run()
	exec.Command("bash", "-c", "fuser -k 443/tcp 2>/dev/null || true").Run()
	exec.Command("bash", "-c", "fuser -k 8080/tcp 2>/dev/null || true").Run()

	// 5. Detener servicios WS viejos que escuchan en 80/443
	exec.Command("systemctl", "stop", "ssh-ws.service").Run()
	exec.Command("systemctl", "stop", "ssh-wss.service").Run()
	exec.Command("systemctl", "disable", "ssh-ws.service").Run()
	exec.Command("systemctl", "disable", "ssh-wss.service").Run()
	os.Remove("/etc/systemd/system/ssh-ws.service")
	os.Remove("/etc/systemd/system/ssh-wss.service")

	// 6. Escribir configuración HAProxy
	if err := os.WriteFile(configFile, []byte(haproxyCfg), 0644); err != nil {
		return fmt.Errorf("fallo escribir haproxy.cfg: %v", err)
	}

	// 7. Crear servicio SSH WebSocket interno (puerto 10015) si no existe
	if exec.Command("systemctl", "is-active", "--quiet", "ssh-ws-internal.service").Run() != nil {
		installSSHWSInternal()
	}

	// 8. Validar y reiniciar HAProxy
	if out, err := exec.Command("haproxy", "-c", "-f", configFile).CombinedOutput(); err != nil {
		return fmt.Errorf("configuración haproxy inválida: %s", string(out))
	}

	exec.Command("systemctl", "daemon-reload").Run()
	exec.Command("systemctl", "enable", "haproxy").Run()
	if err := exec.Command("systemctl", "restart", "haproxy").Run(); err != nil {
		return fmt.Errorf("fallo reinicio haproxy: %v", err)
	}

	return nil
}

// installSSHWSInternal instala el proxy SSH WebSocket interno en puerto 10015
// Este es el servicio al que HAProxy redirige las conexiones WebSocket.
func installSSHWSInternal() {
	_ = exec.Command("apt-get", "install", "-y", "-qq", "python3").Run()

	proxyCode := `#!/usr/bin/env python3
"""SSH WebSocket Proxy (interno para HAProxy) - Puerto 10015"""
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
async def start(port):
    srv = await asyncio.start_server(handle, "127.0.0.1", port)
    async with srv: await srv.serve_forever()
def main():
    port = int(sys.argv[1]) if len(sys.argv) > 1 else 10015
    loop = asyncio.new_event_loop(); asyncio.set_event_loop(loop)
    for s in (signal.SIGTERM, signal.SIGINT):
        try: loop.add_signal_handler(s, lambda: loop.stop())
        except: pass
    try: loop.run_until_complete(start(port))
    except KeyboardInterrupt: pass
    finally: loop.close()
if __name__ == "__main__": main()
`
	proxyScript := "/usr/local/bin/ssh-ws-internal.py"
	os.WriteFile(proxyScript, []byte(proxyCode), 0755)

	svc := `[Unit]
Description=SSH WebSocket Proxy Internal (Puerto 10015 para HAProxy)
After=network.target sshd.service
Wants=sshd.service

[Service]
Type=simple
ExecStart=/usr/bin/python3 ` + proxyScript + ` 10015
Restart=always
RestartSec=3
StandardOutput=journal
StandardError=journal

[Install]
WantedBy=multi-user.target`

	os.WriteFile("/etc/systemd/system/ssh-ws-internal.service", []byte(svc), 0644)
	exec.Command("systemctl", "daemon-reload").Run()
	exec.Command("systemctl", "enable", "ssh-ws-internal.service").Run()
	exec.Command("systemctl", "restart", "ssh-ws-internal.service").Run()
}

// RemoveSSLTunnel detiene y elimina HAProxy y el proxy interno
func RemoveSSLTunnel() error {
	exec.Command("systemctl", "stop", "haproxy").Run()
	exec.Command("systemctl", "disable", "haproxy").Run()
	os.Remove("/etc/haproxy/haproxy.cfg")
	os.Remove("/etc/haproxy/yha.pem")

	// Limpiar proxy interno
	exec.Command("systemctl", "stop", "ssh-ws-internal.service").Run()
	exec.Command("systemctl", "disable", "ssh-ws-internal.service").Run()
	os.Remove("/etc/systemd/system/ssh-ws-internal.service")
	os.Remove("/usr/local/bin/ssh-ws-internal.py")

	exec.Command("systemctl", "daemon-reload").Run()
	return nil
}
