package bot

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/Depwisescript/BOT-TELEGRAM-VPN/internal/db"
	"github.com/Depwisescript/BOT-TELEGRAM-VPN/internal/sys"
	"github.com/Depwisescript/BOT-TELEGRAM-VPN/internal/vpn"
	tele "gopkg.in/telebot.v3"
)

func handleInfo(c tele.Context, b *tele.Bot) error {
	data, _ := db.Load()
	stats := sys.GetSystemStats()

	info := "🌐 <b>INFORMACIÓN DEL SERVIDOR</b>\n"
	info += "━━━━━━━━━━━━━━\n"
	info += fmt.Sprintf("🌍 <b>IP:</b> <code>%s</code>\n", sys.GetPublicIP())
	info += fmt.Sprintf("💻 <b>CPU:</b> %s (%d cores)\n", stats.CPUModel, stats.Cores)
	info += fmt.Sprintf("🔥 <b>Uso:</b> <code>%.1f%%</code>\n", stats.CPUUsage)
	info += fmt.Sprintf("📟 <b>RAM:</b> %dMB / %dMB\n", stats.RAMUsed, stats.RAMTotal)
	info += fmt.Sprintf("💿 <b>Disco:</b> %dGB / %dGB\n", stats.DiskUsed, stats.DiskTotal)
	info += "━━━━━━━━━━━━━━\n"

	// Protocolos
	info += "🛰️ <b>PROTOCOLOS ACTIVOS</b>\n"
	active := false
	if data.SlowDNS.NS != "" {
		info += fmt.Sprintf("🐢 <b>SlowDNS NS:</b> <code>%s</code>\n", data.SlowDNS.NS)
		if data.SlowDNS.Key != "" {
			info += fmt.Sprintf("🔑 <b>SlowDNS Key:</b> <code>%s</code>\n", data.SlowDNS.Key)
		}
		active = true
	}
	if data.Zivpn {
		info += "🛰️ <b>ZiVPN UDP:</b> <code>activo</code>\n"
		active = true
	}
	if data.BadVPN {
		info += "🎮 <b>BadVPN UDPGW:</b> <code>activo (7300)</code>\n"
		active = true
	}
	if data.Falcon != "" {
		info += fmt.Sprintf("🦅 <b>Falcon Proxy:</b> puerto <code>%s</code>\n", data.Falcon)
		active = true
	}
	if data.Dropbear != "" {
		info += fmt.Sprintf("🐻 <b>Dropbear:</b> puerto <code>%s</code>\n", data.Dropbear)
		active = true
	}
	if data.SSLTunnel != "" {
		info += fmt.Sprintf("📜 <b>SSL Tunnel:</b> puerto <code>%s</code>\n", data.SSLTunnel)
		active = true
	}
	if len(data.ProxyDT.Ports) > 0 {
		var ports []string
		for p := range data.ProxyDT.Ports {
			ports = append(ports, "<code>"+p+"</code>")
		}
		info += fmt.Sprintf("🌐 <b>ProxyDT:</b> puertos %s\n", strings.Join(ports, ", "))
		active = true
	}
	if data.CloudflareDomain != "" {
		info += fmt.Sprintf("☁️ <b>Cloudflare DNS:</b> <code>%s</code>\n", data.CloudflareDomain)
	}
	if data.CloudfrontDomain != "" {
		info += fmt.Sprintf("🚀 <b>Cloudfront DNS:</b> <code>%s</code>\n", data.CloudfrontDomain)
	}

	if !active {
		info += "<i>Ningún protocolo instalado.</i>\n"
	}
	info += "━━━━━━━━━━━━━━\n"

	// Solo SuperAdmin ve trafico global
	sa, _ := strconv.ParseInt(superAdmin, 10, 64)
	if c.Chat().ID == sa {
		rx, tx := sys.GetGlobalTraffic()
		info += "📊 <b>TRÁFICO GLOBAL VPS</b>\n"
		info += fmt.Sprintf("📥 <b>Download:</b> <code>%.2f GB</code>\n", rx)
		info += fmt.Sprintf("📤 <b>Upload:</b> <code>%.2f GB</code>\n", tx)
		info += "━━━━━━━━━━━━━━\n"
	}

	info += "\nℹ️ <i>Extrainfo:</i>\n" + data.ExtraInfo

	markup := &tele.ReplyMarkup{}
	markup.Inline(markup.Row(markup.Data("🔙 Volver", "back_main")))

	return SafeEditCtx(c, b, info, markup)
}

func handleMenuOnline(c tele.Context, b *tele.Bot) error {
	sshOnline := sys.GetOnlineUsers()
	zivpnOnline := sys.GetZivpnOnline()

	res := "📊 <b>MONITOR DE CONEXIONES</b>\n\n"

	res += "🔒 <b>SSH / Dropbear:</b>\n"
	if len(sshOnline) > 0 {
		for _, line := range sshOnline {
			res += line + "\n"
		}
	} else {
		res += "<i>Sin conexiones activas.</i>\n"
	}

	res += "\n🛰️ <b>ZIVPN UDP:</b>\n"
	if len(zivpnOnline) > 0 {
		for _, line := range zivpnOnline {
			res += line + "\n"
		}
	} else {
		res += "<i>Sin sesiones activas.</i>\n"
	}

	markup := &tele.ReplyMarkup{}
	markup.Inline(markup.Row(markup.Data("🔙 Volver", "back_main")))

	return SafeEditCtx(c, b, res, markup)
}

// Interceptamos opciones administrativas de borrado
func handleMenuEliminar(c tele.Context, b *tele.Bot) error {
	chatID := c.Chat().ID
	data, _ := db.Load()

	// Filtrar usuarios permitidos para este chatID (o todos si es SuperAdmin)
	sa, _ := strconv.ParseInt(superAdmin, 10, 64)
	isSA := chatID == sa

	res := "🗑️ <b>ELIMINAR CUENTA (SSH/ZiVPN)</b>\n"
	res += "━━━━━━━━━━━━━━\n"

	count := 0
	// Listar SSH
	res += "🔒 <b>Cuentas SSH:</b>\n"
	for user, ownerID := range data.SSHOwners {
		if isSA || ownerID == fmt.Sprintf("%d", chatID) {
			handle := data.SSHHandles[user]
			if handle != "" {
				res += fmt.Sprintf("👤 <code>%s</code> (%s)\n", user, handle)
			} else {
				res += fmt.Sprintf("👤 <code>%s</code>\n", user)
			}
			count++
		}
	}

	// Listar ZiVPN
	res += "\n🛰️ <b>Accesos ZiVPN:</b>\n"
	for pass, ownerID := range data.ZivpnOwners {
		if isSA || ownerID == fmt.Sprintf("%d", chatID) {
			handle := data.ZivpnHandles[pass]
			if handle != "" {
				res += fmt.Sprintf("🔑 <code>%s</code> (%s)\n", pass, handle)
			} else {
				res += fmt.Sprintf("🔑 <code>%s</code>\n", pass)
			}
			count++
		}
	}

	markup := &tele.ReplyMarkup{}
	markup.Inline(markup.Row(markup.Data("🔙 Volver", "back_main")))

	if count == 0 {
		return c.Edit("❌ <b>No hay cuentas para eliminar.</b>", markup, tele.ModeHTML)
	}

	res += "━━━━━━━━━━━━━━\n"
	res += "✏️ <b>Escribe el Nombre o Password</b> de la cuenta que deseas eliminar exactamente como aparece arriba:"

	// Cambiar estado a espera de texto
	UserSteps[chatID] = "awaiting_delete_user_selection"
	LastBotMsg[chatID] = c.Message()

	return SafeEditCtx(c, b, res, markup)
}

func processDeleteSteps(text string, chatID int64, c tele.Context, b *tele.Bot) error {
	data, _ := db.Load()
	target := strings.TrimSpace(text)
	sa, _ := strconv.ParseInt(superAdmin, 10, 64)
	isSA := chatID == sa

	lastMsg := LastBotMsg[chatID]

	// 1. Identificar si es SSH
	if ownerID, exists := data.SSHOwners[target]; exists {
		if !isSA && ownerID != fmt.Sprintf("%d", chatID) {
			_, err := SafeEdit(chatID, b, lastMsg, "⛔ <b>No tienes permiso para eliminar este usuario SSH.</b>", nil)
			return err
		}

		_ = sys.DeleteSSHUser(target)
		db.Update(func(d *db.ConfigData) error {
			delete(d.SSHOwners, target)
			delete(d.SSHHandles, target)
			return nil
		})

		_ = c.Respond(&tele.CallbackResponse{Text: "Usuario SSH eliminado.", ShowAlert: false})
		return handleMenuEliminar(c, b)
	}

	// 2. Identificar si es ZiVPN (usamos el password como id)
	if ownerID, exists := data.ZivpnOwners[target]; exists {
		if !isSA && ownerID != fmt.Sprintf("%d", chatID) {
			_, err := SafeEdit(chatID, b, lastMsg, "⛔ <b>No tienes permiso para eliminar este acceso ZiVPN.</b>", nil)
			return err
		}

		_ = vpn.RemoveZivpnUser(target)
		db.Update(func(d *db.ConfigData) error {
			delete(d.ZivpnUsers, target)
			delete(d.ZivpnOwners, target)
			delete(d.ZivpnHandles, target)
			delete(d.ZivpnLastActive, target)
			return nil
		})

		_ = c.Respond(&tele.CallbackResponse{Text: "Acceso ZiVPN eliminado.", ShowAlert: false})
		return handleMenuEliminar(c, b)
	}

	// 3. No encontrado
	_, err := SafeEdit(chatID, b, lastMsg, "❌ <b>Cuenta no encontrada o no tienes acceso.</b>\n\nVerifica el nombre o password e intenta de nuevo:", nil)
	return err
}
