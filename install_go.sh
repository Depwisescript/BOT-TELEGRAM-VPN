#!/bin/bash
set -euo pipefail

# =========================================================
# INSTALADOR UNIVERSAL V7.0: BOT TELEGRAM DEPWISE SSH 💎 (GO EDITION)
# =========================================================

RED='\033[0;31m'
GREEN='\033[0;32m'
NC='\033[0m'

log_info() { echo -e "${GREEN}[INFO]${NC} $1"; }
log_error() { echo -e "${RED}[ERROR]${NC} $1" >&2; }

if [ "$EUID" -ne 0 ]; then
  log_error "Por favor, ejecuta este script como root"
  exit 1
fi

PROJECT_DIR="/opt/depwise_bot"
ENV_FILE="$PROJECT_DIR/.env"

echo -e "${GREEN}=================================================="
echo -e "       CONFIGURACION BOT DEPWISE V7.0 (GO)"
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
apt update && apt install -y curl git make wget

# 2. Instalar Go si no existe
export PATH=$PATH:/usr/local/go/bin
if ! command -v go &> /dev/null; then
    log_info "Instalando GoLang..."
    wget https://go.dev/dl/go1.21.0.linux-amd64.tar.gz
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
echo -e "       INSTALACION V7.0 COMPLETADA 💎"
echo -e "=================================================="
echo -e "El bot de Go está escuchando. Puedes enviar /start en Telegram.${NC}"
