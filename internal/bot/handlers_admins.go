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
	btnButtons := markup.Data("🎮 Gestión de Botones", "menu_buttons")
	btnReboot := markup.Data("🔄 Reiniciar VPS", "reboot_vps_confirm")
	btnBack := markup.Data("🔙 Volver", "back_main")

	markup.Inline(
		markup.Row(btnToggle),
		markup.Row(btnList, btnAdd),
		markup.Row(btnDel, btnInfo),
		markup.Row(btnCloudflare, btnCloudfront),
		markup.Row(btnBanner),
		markup.Row(btnReset, btnButtons),
		markup.Row(btnReboot),
		markup.Row(btnBack),
	)

	texto := "⚙️ <b>CONFIGURACIÓN PRO (ADMIN)</b>\n"
	texto += "━━━━━━━━━━━━━━\n"
	texto += fmt.Sprintf("🛡️ <b>Acceso:</b> %s\n", accStatus)
	texto += fmt.Sprintf("👤 <b>Admins:</b> %d\n", len(data.Admins)+1) // +1 por SuperAdmin
	texto += fmt.Sprintf("👥 <b>Historial:</b> %d IDs\n", len(data.UserHistory))
	texto += "━━━━━━━━━━━━━━\n"
	texto += "<i>Selecciona una opción avanzada:</i>"

	return c.Edit(texto, markup, tele.ModeHTML)
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
	return c.Edit(res, markup, tele.ModeHTML)
}

func handleAddAdminPrompt(c tele.Context, b *tele.Bot) error {
	chatID := c.Chat().ID
	userSteps[chatID] = "awaiting_vpn_admin_id"
	lastBotMsg[chatID] = c.Message()

	markup := &tele.ReplyMarkup{}
	markup.Inline(markup.Row(markup.Data("❌ Cancelar", "menu_admins")))

	return c.Edit("➕ <b>Agregar Nuevo Administrador</b>\n\n✏️ <i>Escribe el ID numérico del usuario de Telegram:</i>\n\nEjemplo: <code>123456789</code>", markup, tele.ModeHTML)
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

	return c.Edit("➖ <b>Quitar Administrador</b>\n\nSelecciona a quién deseas retirar los permisos:", markup, tele.ModeHTML)
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
	userSteps[chatID] = "awaiting_vpn_extrainfo"
	lastBotMsg[chatID] = c.Message()

	markup := &tele.ReplyMarkup{}
	markup.Inline(markup.Row(markup.Data("❌ Cancelar", "menu_admins")))

	return c.Edit("📝 <b>Editar Información Extra</b>\n\nEsta información aparecerá en el menú /info.\n\n✏️ <i>Escribe el nuevo texto (soporta HTML):</i>", markup, tele.ModeHTML)
}

func handleEditCloudflarePrompt(c tele.Context, b *tele.Bot) error {
	chatID := c.Chat().ID
	userSteps[chatID] = "awaiting_vpn_cloudflare"
	lastBotMsg[chatID] = c.Message()
	markup := &tele.ReplyMarkup{}
	markup.Inline(markup.Row(markup.Data("❌ Cancelar", "menu_admins")))
	return c.Edit("☁️ <b>Configurar Dominio Cloudflare</b>\n\n✏️ <i>Escribe el dominio :</i>\n\nEjemplo: <code>mi.host.com</code>", markup, tele.ModeHTML)
}

func handleEditCloudfrontPrompt(c tele.Context, b *tele.Bot) error {
	chatID := c.Chat().ID
	userSteps[chatID] = "awaiting_vpn_cloudfront"
	lastBotMsg[chatID] = c.Message()
	markup := &tele.ReplyMarkup{}
	markup.Inline(markup.Row(markup.Data("❌ Cancelar", "menu_admins")))
	return c.Edit("🚀 <b>Configurar Dominio Cloudfront</b>\n\n✏️ <i>Escribe el dominio:</i>\n\nEjemplo: <code>xyz123.cloudfront.net</code>", markup, tele.ModeHTML)
}

func handleEditBannerPrompt(c tele.Context, b *tele.Bot) error {
	chatID := c.Chat().ID
	userSteps[chatID] = "awaiting_vpn_ssh_banner"
	lastBotMsg[chatID] = c.Message()
	markup := &tele.ReplyMarkup{}
	markup.Inline(markup.Row(markup.Data("❌ Cancelar", "menu_admins")))
	return c.Edit("📜 <b>Configurar Banner SSH</b>\n\n✏️ <i>Escribe el texto del banner (admite HTML básico):</i>\n\nEsto se mostrará al conectar por SSH.", markup, tele.ModeHTML)
}

func handleResetHistoryConfirm(c tele.Context, b *tele.Bot) error {
	markup := &tele.ReplyMarkup{}
	btnYes := markup.Data("✅ Sí, Limpiar", "reset_history_exec")
	btnNo := markup.Data("❌ No, Cancelar", "menu_admins")
	markup.Inline(markup.Row(btnYes, btnNo))

	return c.Edit("⚠️ <b>¿Estás seguro de limpiar el historial?</b>\n\nSe borrarán todos los IDs de usuarios registrados (el broadcast ya no les llegará hasta que vuelvan a iniciar el bot).", markup, tele.ModeHTML)
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

	return c.Edit("🚨 <b>ADVERTENCIA: REINICIO DEL SERVIDOR</b>\n\n¿Estás seguro de que quieres reiniciar la VPS? Todas las conexiones actuales se cortarán.", markup, tele.ModeHTML)
}

func handleServerRebootExec(c tele.Context, b *tele.Bot) error {
	c.Edit("⏳ <b>Reiniciando VPS...</b> el bot estará offline unos minutos.", tele.ModeHTML)
	exec.Command("reboot").Run()
	return nil
}

func handleMenuButtons(c tele.Context, b *tele.Bot) error {
	chatID := c.Chat().ID
	if !isSuperAdminID(chatID) {
		return c.Send("⛔ Solo el SuperAdmin puede gestionar la visibilidad de botones.", tele.ModeHTML)
	}

	data, _ := db.Load()
	markup := &tele.ReplyMarkup{}

	// ID de botones a gestionar
	btns := []struct {
		ID   string
		Name string
	}{
		{"menu_crear", "👤 Crear SSH"},
		{"menu_info", "📡 Info Servidor"},
		{"menu_editar", "✏️ Editar SSH"},
		{"menu_eliminar", "🗑️ Eliminar SSH"},
		{"menu_scanner", "🔍 Escaner"},
		{"menu_online", "⚙️ Monitor Online"},
		{"menu_protocols", "⚙️ Protocolos"},
		{"menu_admins", "⚙️ Ajustes Pro"},
	}

	var rows []tele.Row
	for _, btn := range btns {
		vis := data.ButtonVisibility[btn.ID]
		pubIcon := "❌"
		if vis.ShowPublic {
			pubIcon = "✅"
		}
		admIcon := "❌"
		if vis.ShowAdmin {
			admIcon = "✅"
		}

		label := fmt.Sprintf("%s | P:%s A:%s", btn.Name, pubIcon, admIcon)
		rows = append(rows, markup.Row(
			markup.Data(label, "toggle_btn_vis:"+btn.ID),
		))
	}

	rows = append(rows, markup.Row(markup.Data("🔙 Volver", "menu_admins")))
	markup.Inline(rows...)

	texto := "🎮 <b>GESTIÓN DE VISIBILIDAD</b>\n\n"
	texto += "Configura qué botones son visibles para cada rango.\n"
	texto += "P = Público | A = Administradores\n\n"
	texto += "<i>Toca un botón para rotar su visibilidad:</i>\n"
	texto += "1. ✅ P | ✅ A\n2. ❌ P | ✅ A\n3. ❌ P | ❌ A"

	return c.Edit(texto, markup, tele.ModeHTML)
}

func handleToggleButtonVisibility(c tele.Context, b *tele.Bot) error {
	btnID := strings.TrimPrefix(c.Callback().Data, "toggle_btn_vis:")

	db.Update(func(data *db.ConfigData) error {
		vis := data.ButtonVisibility[btnID]

		// Ciclo de visibilidad
		if vis.ShowPublic && vis.ShowAdmin {
			// Caso 1 -> Caso 2 (Ocultar a Publico)
			vis.ShowPublic = false
			vis.ShowAdmin = true
		} else if !vis.ShowPublic && vis.ShowAdmin {
			// Caso 2 -> Caso 3 (Ocultar a Admin también)
			vis.ShowPublic = false
			vis.ShowAdmin = false
		} else {
			// Caso 3 (o cualquier otro) -> Caso 1 (Mostrar a todos)
			vis.ShowPublic = true
			vis.ShowAdmin = true
		}

		data.ButtonVisibility[btnID] = vis
		return nil
	})

	return handleMenuButtons(c, b)
}
