# 💎 DEPWISE BOT GO EDITION

<p align="center">
  <img src="https://img.shields.io/badge/Language-Go-00ADD8?style=for-the-badge&logo=go" alt="Go">
  <img src="https://img.shields.io/badge/Platform-Linux-FCC624?style=for-the-badge&logo=linux" alt="Linux">
  <img src="https://img.shields.io/badge/Status-Stable-success?style=for-the-badge" alt="Status">
  <img src="https://img.shields.io/badge/Version-7.0-blue?style=for-the-badge" alt="Version">
</p>

---

## 🚀 ¿Qué es Depwise Bot?

**Depwise Bot Go Edition** es una solución integral y de alto rendimiento para la gestión de servidores VPN y cuentas SSH a través de Telegram. Reesclito completamente en **Go** para garantizar velocidad, estabilidad y bajo consumo de recursos, este bot transforma tu VPS en un panel de control profesional y automatizado.

---

## ✨ Características Principales

### 🛠️ Gestión de Protocolos (All-in-One)
- **SSH/Dropbear/SSL Tunnel:** Gestión completa de cuentas con límites de conexión.
- **SlowDNS:** Instalación y configuración de túneles DNS.
- **ZiVPN & UDP Custom:** Soporte para protocolos de gaming y bypass robusto.
- **ProxyDT:** Integración con ProxyDT Cracked para túneles HTTP estables.
- **Falcon Proxy:** Proxy de alto rendimiento integrado.

### 🛡️ Administración Pro (Panel de Control)
- **Ajustes Pro:** Panel administrativo avanzado con control de acceso público/privado.
- **Mensaje Global (Broadcast):** Envía anuncios a todos tus usuarios con reportes en tiempo real.
- **Gestión de Admins:** Agrega o quita administradores directamente desde el bot.
- **Dominios CF/Cloudfront:** Integración nativa con dominios de CDN.

### 🧹 Mantenimiento Inteligente
- **Limpieza de Zombis:** Eliminación automática de usuarios inactivos por más de 12 horas (excluyendo al SuperAdmin).
- **Deep System Cleanup:** Botón de un solo clic para liberar espacio en el SSD (Apt, Logs, Caché de Go).
- **Auto-Cleanup Loop:** Monitoreo constante de expiraciones y limpieza de sistema.

---

## 📥 Instalación Rápida (Universal)

Ejecuta el siguiente comando en tu terminal como usuario **root**:

```bash
apt update && apt install -y git && git clone https://github.com/Depwisescript/BOT-TELEGRAM-VPN.git && cd BOT-TELEGRAM-VPN && chmod +x install_go.sh && ./install_go.sh
```

> [!IMPORTANT]
> El instalador configurará automáticamente el entorno Go, compilará el bot y lo registrará como un servicio de sistema (SystemD) para que siempre esté activo.

---

## 🛠️ Comandos de Terminal Útiles

| Comando | Descripción |
| :--- | :--- |
| `systemctl restart depwise` | Reiniciar el servicio del bot |
| `systemctl status depwise` | Ver el estado actual del bot |
| `journalctl -u depwise -f` | Ver los logs en tiempo real |
| `df -h` | Verificar espacio en SSD |

---

## 💎 Créditos y Soporte

Este proyecto es desarrollado y mantenido con pasión por:

- **👨‍💻 Desarrollador:** [Depwisescript](https://github.com/Depwisescript)
- **📢 Canal Oficial:** [Depwise Channel](https://t.me/Depwise2)

---

<p align="center">
  <i>"Potenciando tu VPS con la velocidad de Go."</i><br>
  <b>© 2026 Depwise Project</b>
</p>
