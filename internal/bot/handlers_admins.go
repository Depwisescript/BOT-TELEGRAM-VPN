package bot

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/Depwisescript/BOT-TELEGRAM-VPN/internal/db"
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
	LastBotMsg[chatID] = c.Message()

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
	LastBotMsg[chatID] = c.Message()

	markup := &tele.ReplyMarkup{}
	markup.Inline(markup.Row(markup.Data("❌ Cancelar", "menu_admins")))

	return SafeEditCtx(c, b, "📝 <b>Editar Información Extra</b>\n\nEsta información aparecerá en el menú /info.\n\n✏️ <i>Escribe el nuevo texto (soporta HTML):</i>", markup)
}

func handleEditCloudflarePrompt(c tele.Context, b *tele.Bot) error {
	chatID := c.Chat().ID
	UserSteps[chatID] = "awaiting_vpn_cloudflare"
	LastBotMsg[chatID] = c.Message()
	markup := &tele.ReplyMarkup{}
	markup.Inline(markup.Row(markup.Data("❌ Cancelar", "menu_admins")))
	return SafeEditCtx(c, b, "☁️ <b>Configurar Dominio Cloudflare</b>\n\n✏️ <i>Escribe el dominio :</i>\n\nEjemplo: <code>mi.host.com</code>", markup)
}

func handleEditCloudfrontPrompt(c tele.Context, b *tele.Bot) error {
	chatID := c.Chat().ID
	UserSteps[chatID] = "awaiting_vpn_cloudfront"
	LastBotMsg[chatID] = c.Message()
	markup := &tele.ReplyMarkup{}
	markup.Inline(markup.Row(markup.Data("❌ Cancelar", "menu_admins")))
	return SafeEditCtx(c, b, "🚀 <b>Configurar Dominio Cloudfront</b>\n\n✏️ <i>Escribe el dominio:</i>\n\nEjemplo: <code>xyz123.cloudfront.net</code>", markup)
}

func handleEditBannerPrompt(c tele.Context, b *tele.Bot) error {
	chatID := c.Chat().ID
	UserSteps[chatID] = "awaiting_vpn_ssh_banner"
	LastBotMsg[chatID] = c.Message()
	markup := &tele.ReplyMarkup{}
	markup.Inline(markup.Row(markup.Data("❌ Cancelar", "menu_admins")))
	return SafeEditCtx(c, b, "📜 <b>Configurar Banner SSH</b>\n\n✏️ <i>Escribe el texto del banner (admite HTML básico):</i>\n\nEsto se mostrará al conectar por SSH.", markup)
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
