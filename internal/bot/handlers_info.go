package bot

import (
	"fmt"
	"strings"

	"github.com/Depwisescript/BOT-TELEGRAM-VPN/internal/db"
	"github.com/Depwisescript/BOT-TELEGRAM-VPN/internal/sys"
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
		for p := range data.ProxyDT.Ports { ports = append(ports, "<code>"+p+"</code>")}
		info += fmt.Sprintf("🌐 <b>ProxyDT:</b> puertos %s\n", strings.Join(ports, ", "))
		active = true
	}
	
	if !active {
		info += "<i>Ningún protocolo instalado.</i>\n"
	}
	info += "━━━━━━━━━━━━━━\n"
	
    info += "\nℹ️ <i>Extrainfo:</i>\n" + data.ExtraInfo
    
	markup := &tele.ReplyMarkup{}
	markup.Inline(markup.Row(markup.Data("🔙 Volver", "back_main")))

	return c.Edit(info, markup, tele.ModeHTML)
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
	
	return c.Edit(res, markup, tele.ModeHTML)
}

func handleMyStats(c tele.Context, b *tele.Bot) error {
	chatID := c.Chat().ID
	data, _ := db.Load()
	
	// Buscar que usuarios le pertenecen a este chatID
	var myUsers []string
	for user, ownerID := range data.SSHOwners {
		if ownerID == fmt.Sprintf("%d", chatID) {
			myUsers = append(myUsers, user)
		}
	}
	
	if len(myUsers) == 0 {
		return c.Send("❌ No tienes ningún usuario asignado o creado.", tele.ModeHTML)
	}

	res := "📊 <b>Tus Usuarios SSH y Consumos</b>\n\n"
	for _, u := range myUsers {
		gb, limit, _ := sys.GetUserConsumption(u)
		res += fmt.Sprintf("👤 <code>%s</code> - <b>%.2f GB</b> / %s GB\n", u, gb, limit)
	}

	markup := &tele.ReplyMarkup{}
	markup.Inline(markup.Row(markup.Data("🔙 Volver", "back_main")))
	
	return c.Edit(res, markup, tele.ModeHTML)
}

// Interceptamos opciones administrativas de borrado
func handleMenuEliminar(c tele.Context, b *tele.Bot) error {
	chatID := c.Chat().ID
	
	userSteps[chatID] = "awaiting_delete_user"
	lastBotMsg[chatID] = c.Message()
	
	markup := &tele.ReplyMarkup{}
	markup.Inline(markup.Row(markup.Data("❌ Cancelar", "cancelar_accion")))
	
	return c.Edit("🗑️ <b>Eliminar Usuario SSH</b>\n\n✏️ <i>Dime el nombre del usuario a borrar:</i>", markup, tele.ModeHTML)
}
