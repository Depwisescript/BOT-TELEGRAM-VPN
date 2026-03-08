package bot

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/Depwisescript/BOT-TELEGRAM-VPN/internal/db"
	"github.com/Depwisescript/BOT-TELEGRAM-VPN/internal/sys"
	tele "gopkg.in/telebot.v3"
)

func handleMenuAdmins(c tele.Context, b *tele.Bot) error {
	chatID := c.Chat().ID
	if !isAdmin(chatID) {
		return c.Send("⛔ Solo administradores pueden usar esta función.", tele.ModeHTML)
	}

	data, _ := db.Load()
	accStatus := "🔓 Público"
	if !data.PublicAccess {
		accStatus = "🔒 Privado"
	}

	markup := &tele.ReplyMarkup{}
	btnToggle := markup.Data("🔄 Acceso: "+accStatus, "toggle_public_access")
	btnList := markup.Data("📋 Listar Admins", "list_admins")
	btnAdd := markup.Data("➕ Agregar Admin", "add_admin")
	btnDel := markup.Data("➖ Quitar Admin", "del_admin_menu")
	btnInfo := markup.Data("📝 Editar Info Extra", "edit_extrainfo")
	btnCloudflare := markup.Data("☁️ Cloudflare Domain", "edit_cloudflare")
	btnCloudfront := markup.Data("🚀 Cloudfront Domain", "edit_cloudfront")
	btnBanner := markup.Data("📜 Banner SSH", "edit_banner")
	btnReset := markup.Data("🧹 Limpiar Historial", "reset_history")

	scanPubStatus := "🔓 ON"
	if !data.PublicScanner {
		scanPubStatus = "🔒 OFF"
	}
	btnScanToggle := markup.Data("🔍 Escaner Público: "+scanPubStatus, "toggle_public_scanner")

	btnReboot := markup.Data("🔄 Reiniciar VPS", "reboot_vps_confirm")
	btnBack := markup.Data("🔙 Volver", "back_main")

	markup.Inline(
		markup.Row(btnToggle),
		markup.Row(btnList, btnAdd),
		markup.Row(btnDel, btnInfo),
		markup.Row(btnCloudflare, btnCloudfront),
		markup.Row(btnBanner),
		markup.Row(btnReset, btnScanToggle),
		markup.Row(btnReboot),
		markup.Row(btnBack),
	)

	texto := "⚙️ <b>CONFIGURACIÓN PRO (ADMIN)</b>\n"
	texto += "━━━━━━━━━━━━━━\n"
	texto += fmt.Sprintf("🛡️ <b>Acceso:</b> %s\n", accStatus)
	texto += fmt.Sprintf("🔍 <b>Escaner Público:</b> %s\n", scanPubStatus)
	texto += fmt.Sprintf("👤 <b>Admins:</b> %d\n", len(data.Admins)+1) // +1 por SuperAdmin
	texto += fmt.Sprintf("👥 <b>Historial:</b> %d IDs\n", len(data.UserHistory))
	texto += "━━━━━━━━━━━━━━\n"
	texto += "<i>Selecciona una opción avanzada:</i>"

	return SafeEditCtx(c, b, texto, markup)
}

func handleTogglePublicAccess(c tele.Context, b *tele.Bot) error {
	db.Update(func(data *db.ConfigData) error {
		data.PublicAccess = !data.PublicAccess
		return nil
	})
	return handleMenuAdmins(c, b)
}

func handleListAdmins(c tele.Context, b *tele.Bot) error {
	data, _ := db.Load()
	res := "📋 <b>LISTADO DE ADMINISTRADORES</b>\n\n"
	res += fmt.Sprintf("⭐ <b>SuperAdmin (Root):</b> <code>%s</code>\n", superAdmin)

	if len(data.Admins) == 0 {
		res += "\n<i>No hay administradores adicionales.</i>"
	} else {
		for id, info := range data.Admins {
			res += fmt.Sprintf("👤 ID: <code>%s</code> - <b>%s</b>\n", id, info.Alias)
		}
	}

	markup := &tele.ReplyMarkup{}
	markup.Inline(markup.Row(markup.Data("🔙 Volver", "menu_admins")))
	return SafeEditCtx(c, b, res, markup)
}

func handleAddAdminPrompt(c tele.Context, b *tele.Bot) error {
	chatID := c.Chat().ID
	UserSteps[chatID] = "awaiting_vpn_admin_id"

	markup := &tele.ReplyMarkup{}
	markup.Inline(markup.Row(markup.Data("❌ Cancelar", "menu_admins")))

	return SafeEditCtx(c, b, "➕ <b>Agregar Nuevo Administrador</b>\n\n✏️ <i>Escribe el ID numérico del usuario de Telegram:</i>\n\nEjemplo: <code>123456789</code>", markup)
}

func handleDelAdminMenu(c tele.Context, b *tele.Bot) error {
	data, _ := db.Load()
	if len(data.Admins) == 0 {
		return c.Respond(&tele.CallbackResponse{Text: "No hay administradores para quitar.", ShowAlert: true})
	}

	markup := &tele.ReplyMarkup{}
	var rows []tele.Row
	for id, info := range data.Admins {
		rows = append(rows, markup.Row(markup.Data("❌ "+info.Alias+" ("+id+")", "del_adm_exec:"+id)))
	}
	rows = append(rows, markup.Row(markup.Data("🔙 Volver", "menu_admins")))
	markup.Inline(rows...)

	return SafeEditCtx(c, b, "➖ <b>Quitar Administrador</b>\n\nSelecciona a quién deseas retirar los permisos:", markup)
}

func handleDelAdminExec(c tele.Context, b *tele.Bot) error {
	id := strings.TrimPrefix(c.Callback().Data, "del_adm_exec:")
	db.Update(func(data *db.ConfigData) error {
		delete(data.Admins, id)
		return nil
	})
	return handleListAdmins(c, b)
}

func handleEditExtraInfoPrompt(c tele.Context, b *tele.Bot) error {
	chatID := c.Chat().ID
	UserSteps[chatID] = "awaiting_vpn_extrainfo"

	markup := &tele.ReplyMarkup{}
	markup.Inline(markup.Row(markup.Data("❌ Cancelar", "menu_admins")))

	return SafeEditCtx(c, b, "📝 <b>Editar Información Extra</b>\n\nEsta información aparecerá en el menú /info.\n\n✏️ <i>Escribe el nuevo texto (soporta HTML):</i>", markup)
}

func handleEditCloudflarePrompt(c tele.Context, b *tele.Bot) error {
	chatID := c.Chat().ID
	UserSteps[chatID] = "awaiting_vpn_cloudflare"
	markup := &tele.ReplyMarkup{}
	markup.Inline(markup.Row(markup.Data("❌ Cancelar", "menu_admins")))
	return SafeEditCtx(c, b, "☁️ <b>Configurar Dominio Cloudflare</b>\n\n✏️ <i>Escribe el dominio :</i>\n\nEjemplo: <code>mi.host.com</code>", markup)
}

func handleEditCloudfrontPrompt(c tele.Context, b *tele.Bot) error {
	chatID := c.Chat().ID
	UserSteps[chatID] = "awaiting_vpn_cloudfront"
	markup := &tele.ReplyMarkup{}
	markup.Inline(markup.Row(markup.Data("❌ Cancelar", "menu_admins")))
	return SafeEditCtx(c, b, "🚀 <b>Configurar Dominio Cloudfront</b>\n\n✏️ <i>Escribe el dominio:</i>\n\nEjemplo: <code>xyz123.cloudfront.net</code>", markup)
}

// Banner predeterminado de Depwise
const defaultBanner = `<html>
<h5 style="text-align:center;">
<font face="monospace" color="#00ff00">
⠀⠀⢀⣶⡆⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⢰⣶⡀⠀⠀
⠀⠀⢸⣿⡇⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⢸⣿⡇⠀⠀
⠀⠀⢸⣿⡇⠀⠀⠀⣠⣶⣄⠀⠀⠀⢸⣿⡇⠀⠀
⠀⠀⢸⣿⡇⠀⠀⢰⣿⣿⣿⡆⠀⠀⢸⣿⡇⠀⠀
⠀⠀⠈⣿⣿⡄⢀⣿⣿⠻⣿⣿⡀⢠⣿⣿⠁⠀⠀
⠀⠀⠀⠹⣿⣿⣾⣿⡏⠀⢹⣿⣷⣿⣿⠏⠀⠀⠀
⠀⠀⠀⠀⠙⢿⣿⡿⠀⠀⠀⢿⣿⡿⠋⠀⠀⠀⠀
</font>
</h5>
<h1 style="text-align:center;">
<font face="monospace" color="#00ff00"><b>DEPWISE</b></font>
</h1>
<h5 style="text-align:center;">
<font color='#29b6f6'>==============================</font>
<font color='#29b6f6'><b>✈ TELEGRAM ✈</b></font>
<font color='#29b6f6'>==============================</font>
</h5>
<h5 style="text-align:center;">
<font color='#ffffff'>Dev: </font><a href="https://t.me/Dan3651"><font color='#f1c40f'>@Dan3651</font></a>
<font color='#ffffff'>Canal: </font><a href="https://t.me/Depwise2"><font color='#f1c40f'>@Depwise2</font></a>
</h5>
<h4 style="text-align:center;">
<font color='#FF00FF'><b>🔥 ¡SE VENDEN SERVIDORES PREMIUM 35 DÍAS A 8.5 SOLES! 🔥</b></font>
</h4>
<h5 style="text-align:center;">
<font color='#ff0000'>==============================</font>
<font color='#ff0000'><b>⚡ SERVIDORES FREE ⚡</b></font>
<font color='#ff0000'>==============================</font>
</h5>
<h6 style="text-align:center;">
<font color='#ff9800'><b>⚠️ REGLAS DEL SERVIDOR ⚠️</b></font>
<font color='#ffffff'>🚫 NO Torrent / P2P</font>
<font color='#ffffff'>🚫 NO Spam / Fraude</font>
<font color='#ffffff'>🚫 NO Ataques DDoS</font>
<font color='#ff5252'><i>El incumplimiento genera ban automático</i></font>
</h6>
<h5 style="text-align:center;">
<font color='#00e676'><b>CREADO EN : @Depwise_bot</b></font>
</h5>
</html>`

func handleEditBannerPrompt(c tele.Context, b *tele.Bot) error {
	data, _ := db.Load()

	status := "❌ Sin banner"
	bannerType := ""
	if data.SSHBanner != "" {
		status = "✅ Activo"
		if data.SSHBanner == defaultBanner {
			bannerType = "\n📋 <b>Tipo:</b> Predeterminado (Depwise)"
		} else {
			bannerType = "\n📋 <b>Tipo:</b> Personalizado"
		}
	}

	markup := &tele.ReplyMarkup{}
	btnDefault := markup.Data("🎨 Activar Predeterminado", "banner_set_default")
	btnCustom := markup.Data("✏️ Personalizar", "banner_set_custom")
	btnDeactivate := markup.Data("🚫 Desactivar Banner", "banner_deactivate")
	btnBack := markup.Data("🔙 Volver", "menu_admins")

	markup.Inline(
		markup.Row(btnDefault),
		markup.Row(btnCustom),
		markup.Row(btnDeactivate),
		markup.Row(btnBack),
	)

	texto := fmt.Sprintf("📜 <b>Gestión de Banner SSH</b>\n\n📊 <b>Estado:</b> %s%s\n\n<i>El banner se muestra al usuario al conectar por SSH.</i>\n\n¿Qué deseas hacer?", status, bannerType)
	return SafeEditCtx(c, b, texto, markup)
}

func handleBannerSetDefault(c tele.Context, b *tele.Bot) error {
	SafeEditCtx(c, b, "⏳ <i>Activando banner predeterminado...</i>", nil)

	db.Update(func(data *db.ConfigData) error {
		data.SSHBanner = defaultBanner
		return nil
	})

	err := sys.SetSSHBanner(defaultBanner)
	markup := &tele.ReplyMarkup{}
	markup.Inline(markup.Row(markup.Data("🔙 Volver", "edit_banner")))

	if err != nil {
		return SafeEditCtx(c, b, fmt.Sprintf("⚠️ <b>Banner guardado pero error al aplicar:</b>\n%v", err), markup)
	}
	return SafeEditCtx(c, b, "✅ <b>Banner Depwise predeterminado activado y aplicado.</b>", markup)
}

func handleBannerSetCustom(c tele.Context, b *tele.Bot) error {
	chatID := c.Chat().ID
	UserSteps[chatID] = "awaiting_vpn_ssh_banner"
	markup := &tele.ReplyMarkup{}
	markup.Inline(markup.Row(markup.Data("❌ Cancelar", "edit_banner")))
	return SafeEditCtx(c, b, "📜 <b>Banner SSH Personalizado</b>\n\n✏️ <i>Escribe el texto del banner (admite HTML básico):</i>\n\nEsto se mostrará al conectar por SSH.", markup)
}

func handleBannerDeactivate(c tele.Context, b *tele.Bot) error {
	db.Update(func(data *db.ConfigData) error {
		data.SSHBanner = ""
		return nil
	})

	// Quitar banner del sistema
	exec.Command("sh", "-c", "rm -f /etc/sshd_banner").Run()
	exec.Command("sed", "-i", "/^Banner/d", "/etc/ssh/sshd_config").Run()
	exec.Command("systemctl", "reload", "ssh").Run()

	markup := &tele.ReplyMarkup{}
	markup.Inline(markup.Row(markup.Data("🔙 Volver", "edit_banner")))
	return SafeEditCtx(c, b, "✅ <b>Banner SSH desactivado.</b>\n\n<i>Ya no se mostrará ningún banner al conectar.</i>", markup)
}

func handleResetHistoryConfirm(c tele.Context, b *tele.Bot) error {
	markup := &tele.ReplyMarkup{}
	btnYes := markup.Data("✅ Sí, Limpiar", "reset_history_exec")
	btnNo := markup.Data("❌ No, Cancelar", "menu_admins")
	markup.Inline(markup.Row(btnYes, btnNo))

	return SafeEditCtx(c, b, "⚠️ <b>¿Estás seguro de limpiar el historial?</b>\n\nSe borrarán todos los IDs de usuarios registrados (el broadcast ya no les llegará hasta que vuelvan a iniciar el bot).", markup)
}

func handleResetHistoryExec(c tele.Context, b *tele.Bot) error {
	db.Update(func(data *db.ConfigData) error {
		data.UserHistory = []int64{}
		return nil
	})
	return c.Respond(&tele.CallbackResponse{Text: "Historial de IDs reseteado.", ShowAlert: true})
}

func handleServerRebootConfirm(c tele.Context, b *tele.Bot) error {
	markup := &tele.ReplyMarkup{}
	btnYes := markup.Data("🔄 Reiniciar AHORA", "reboot_vps_exec")
	btnNo := markup.Data("🔙 Cancelar", "menu_admins")
	markup.Inline(markup.Row(btnYes, btnNo))

	return SafeEditCtx(c, b, "🚨 <b>ADVERTENCIA: REINICIO DEL SERVIDOR</b>\n\n¿Estás seguro de que quieres reiniciar la VPS? Todas las conexiones actuales se cortarán.", markup)
}

func handleServerRebootExec(c tele.Context, b *tele.Bot) error {
	c.Edit("⏳ <b>Reiniciando VPS...</b> el bot estará offline unos minutos.", tele.ModeHTML)
	exec.Command("reboot").Run()
	return nil
}

func handleTogglePublicScanner(c tele.Context, b *tele.Bot) error {
	db.Update(func(data *db.ConfigData) error {
		data.PublicScanner = !data.PublicScanner
		return nil
	})
	return handleMenuAdmins(c, b)
}
