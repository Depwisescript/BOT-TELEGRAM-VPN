#!/bin/bash
set -euo pipefail

# =========================================================
# INSTALADOR UNIVERSAL V7.1: BOT TELEGRAM DEPWISE SSH 💎 (GO EDITION)
# =========================================================

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
NC='\033[0m'

log_info() { echo -e "${GREEN}[INFO]${NC} $1"; }
log_warn() { echo -e "${YELLOW}[WARN]${NC} $1"; }
log_error() { echo -e "${RED}[ERROR]${NC} $1" >&2; }

if [ "$EUID" -ne 0 ]; then
  log_error "Por favor, ejecuta este script como root"
  exit 1
fi

PROJECT_DIR="/opt/depwise_bot"
ENV_FILE="$PROJECT_DIR/.env"

install_bot() {
    echo -e "${GREEN}=================================================="
    echo -e "       CONFIGURACION BOT DEPWISE V7.1 (GO)"
    echo -e "==================================================${NC}"

    read -p "Introduce el TOKEN: " BOT_TOKEN
    read -p "Introduce tu Chat ID de Telegram: " ADMIN_ID

    if [ -z "$BOT_TOKEN" ] || [ -z "$ADMIN_ID" ]; then
        log_error "Error: Datos incompletos."
        exit 1
    fi

    # 1. Preparar Entorno
    mkdir -p "$PROJECT_DIR"
    echo "BOT_TOKEN=$BOT_TOKEN" > "$ENV_FILE"
    echo "SUPER_ADMIN=$ADMIN_ID" >> "$ENV_FILE"
    chmod 600 "$ENV_FILE"

    log_info "Instalando dependencias base..."
    apt update -y && apt install -y curl git make wget

    # 2. Instalar Go si no existe
    export PATH=$PATH:/usr/local/go/bin
    if ! command -v go &> /dev/null; then
        log_info "Instalando GoLang..."
        wget -q https://go.dev/dl/go1.21.0.linux-amd64.tar.gz
        rm -rf /usr/local/go && tar -C /usr/local -xzf go1.21.0.linux-amd64.tar.gz
        rm go1.21.0.linux-amd64.tar.gz
    fi

    # 3. Clonar y Compilar Proyecto Repo
    log_info "Descargando y compilando el Bot en Go..."
    cd /tmp
    rm -rf BOT-TELEGRAM-VPN
    git clone https://github.com/Depwisescript/BOT-TELEGRAM-VPN.git || { log_error "Error al descargar el bot."; exit 1; }
    cd BOT-TELEGRAM-VPN

    log_info "Descargando módulos necesarios..."
    go mod tidy

    go build -o /usr/local/bin/depwise-bot cmd/depwise/main.go
    chmod +x /usr/local/bin/depwise-bot
    rm -rf /tmp/BOT-TELEGRAM-VPN

    # 4. Servicio Systemd
    log_info "Generando sistema daemon SystemD..."
    cat << EOF > /etc/systemd/system/depwise.service
[Unit]
Description=Depwise Telegram Bot (Go Edition)
After=network.target

[Service]
Type=simple
User=root
EnvironmentFile=$ENV_FILE
ExecStart=/usr/local/bin/depwise-bot
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
EOF

    systemctl daemon-reload
    systemctl enable depwise.service
    systemctl restart depwise.service

    echo -e "${GREEN}=================================================="
    echo -e "       INSTALACION V7.1 COMPLETADA 💎"
    echo -e "=================================================="
    echo -e "El bot de Go está escuchando. Puedes enviar /start en Telegram.${NC}"
}

uninstall_all() {
    echo -e "${RED}=================================================="
    echo -e "       ⚠️ ADVERTENCIA: DESINSTALACIÓN TOTAL ⚠️"
    echo -e "==================================================${NC}"
    echo -e "Esto eliminará:"
    echo -e "- El Bot de Telegram y sus configuraciones"
    echo -e "- Todos los servicios VPN instalados por el bot (SlowDNS, ProxyDT, SSL, etc.)"
    echo -e "- Los binarios descargados"
    echo -e "- La base de datos de usuarios (bot_data.json)"
    
    read -p "¿Estás completamente seguro de continuar? (escribe 'si' para confirmar): " confirm
    if [ "$confirm" != "si" ]; then
        log_info "Desinstalación cancelada."
        return
    fi

    log_info "1/4 Deteniendo servicios..."
    systemctl stop depwise.service 2>/dev/null || true
    systemctl disable depwise.service 2>/dev/null || true
    
    # Detener proxies y vpns
    local services=("badvpn" "proxydt" "stunnel4" "dropbear" "falconproxy" "udpcustom" "zivpn" "nsd")
    for svc in "${services[@]}"; do
        systemctl stop "$svc" 2>/dev/null || true
        systemctl disable "$svc" 2>/dev/null || true
        rm -f "/etc/systemd/system/${svc}.service"
    done

    log_info "2/4 Eliminando archivos y binarios..."
    rm -f /usr/local/bin/depwise-bot
    rm -f /etc/systemd/system/depwise.service
    rm -rf "$PROJECT_DIR"
    rm -f /root/bot_data.json
    
    # VPN Binaries & Configs
    rm -f /usr/local/bin/badvpn-udpgw
    rm -f /usr/local/bin/proxydt
    rm -f /usr/local/bin/falconproxy
    rm -f /usr/local/bin/udpcustom
    rm -rf /etc/zivpn
    rm -f /usr/local/bin/zivpn
    rm -f /etc/falconproxy.conf
    rm -rf /etc/slowdns

    log_info "3/4 Limpiando GoLang..."
    rm -rf /usr/local/go
    # Remover del PATH si está en bashrc (opcional/precaución)
    sed -i '/\/usr\/local\/go\/bin/d' /root/.bashrc || true

    log_info "4/4 Recargando demonios de sistema..."
    systemctl daemon-reload

    echo -e "${GREEN}=================================================="
    echo -e "   ✅ DESINSTALACIÓN COMPLETADA EXITOSAMENTE  "
    echo -e "==================================================${NC}"
}

show_menu() {
    clear
    echo -e "${CYAN}=================================================="
    echo -e "       DEPWISE BOT INSTALLER (GO EDITION)"
    echo -e "==================================================${NC}"
    echo -e "  1. ${GREEN}Instalar / Actualizar Bot${NC}"
    echo -e "  2. ${RED}Desinstalar Todo (Bot + VPNs)${NC}"
    echo -e "  3. Salir"
    echo -e "${CYAN}==================================================${NC}"
    read -p "Selecciona una opción [1-3]: " opt

    case $opt in
        1) install_bot ;;
        2) uninstall_all ;;
        3) exit 0 ;;
        *) log_error "Opción inválida"; sleep 2; show_menu ;;
    esac
}

show_menu
