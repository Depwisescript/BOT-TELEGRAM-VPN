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
	btnDeepClean := markup.Data("🧹 Limpiar Basura SSD", "deep_cleanup")
	btnClean := markup.Data("⚠️ Reset DB", "clean_db_confirm")
	btnReboot := markup.Data("🔄 Reiniciar VPS", "reboot_vps_confirm")
	btnBack := markup.Data("🔙 Volver", "back_main")

	markup.Inline(
		markup.Row(btnToggle),
		markup.Row(btnList, btnAdd),
		markup.Row(btnDel, btnInfo),
		markup.Row(btnCloudflare, btnCloudfront),
		markup.Row(btnBanner),
		markup.Row(btnReset, btnDeepClean),
		markup.Row(btnClean, btnReboot),
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
	userSteps[chatID] = "awaiting_admin_id"
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
	userSteps[chatID] = "awaiting_extrainfo"
	lastBotMsg[chatID] = c.Message()

	markup := &tele.ReplyMarkup{}
	markup.Inline(markup.Row(markup.Data("❌ Cancelar", "menu_admins")))

	return c.Edit("📝 <b>Editar Información Extra</b>\n\nEsta información aparecerá en el menú /info.\n\n✏️ <i>Escribe el nuevo texto (soporta HTML):</i>", markup, tele.ModeHTML)
}

func handleEditCloudflarePrompt(c tele.Context, b *tele.Bot) error {
	chatID := c.Chat().ID
	userSteps[chatID] = "awaiting_cloudflare"
	lastBotMsg[chatID] = c.Message()
	markup := &tele.ReplyMarkup{}
	markup.Inline(markup.Row(markup.Data("❌ Cancelar", "menu_admins")))
	return c.Edit("☁️ <b>Configurar Dominio Cloudflare</b>\n\n✏️ <i>Escribe el dominio :</i>\n\nEjemplo: <code>mi.host.com</code>", markup, tele.ModeHTML)
}

func handleEditCloudfrontPrompt(c tele.Context, b *tele.Bot) error {
	chatID := c.Chat().ID
	userSteps[chatID] = "awaiting_cloudfront"
	lastBotMsg[chatID] = c.Message()
	markup := &tele.ReplyMarkup{}
	markup.Inline(markup.Row(markup.Data("❌ Cancelar", "menu_admins")))
	return c.Edit("🚀 <b>Configurar Dominio Cloudfront</b>\n\n✏️ <i>Escribe el dominio:</i>\n\nEjemplo: <code>xyz123.cloudfront.net</code>", markup, tele.ModeHTML)
}

func handleEditBannerPrompt(c tele.Context, b *tele.Bot) error {
	chatID := c.Chat().ID
	userSteps[chatID] = "awaiting_ssh_banner"
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

func handleCleanDBConfirm(c tele.Context, b *tele.Bot) error {
	markup := &tele.ReplyMarkup{}
	btnYes := markup.Data("🧨 FORMATEAR DB", "clean_db_exec")
	btnNo := markup.Data("🔙 Cancelar", "menu_admins")
	markup.Inline(markup.Row(btnYes, btnNo))

	return c.Edit("🧨 <b>BORRADO TOTAL DE DATOS</b>\n\n¿Estás seguro de resetear la base de datos? Se perderán configuraciones de puertos, protocolos y registros (Se mantienen SuperAdmin y Admins).", markup, tele.ModeHTML)
}

func handleCleanDBExec(c tele.Context, b *tele.Bot) error {
	db.Update(func(data *db.ConfigData) error {
		admins := data.Admins
		// Resetear casi todo
		*data = *defaultData()
		data.Admins = admins
		return nil
	})
	return handleMenuAdmins(c, b)
}

func handleDeepCleanup(c tele.Context, b *tele.Bot) error {
	c.Edit("🧹 <b>Iniciando limpieza profunda...</b>\nEsto puede tardar unos segundos.", tele.ModeHTML)

	report, err := sys.PerformFullCleanup()

	markup := &tele.ReplyMarkup{}
	markup.Inline(markup.Row(markup.Data("🔙 Volver", "menu_admins")))

	if err != nil {
		return c.Edit("❌ <b>Error durante la limpieza:</b>\n"+err.Error(), markup, tele.ModeHTML)
	}

	return c.Edit(report, markup, tele.ModeHTML)
}

// Replicamos defaultData si no es exportado, o lo exportamos en db/data.go
func defaultData() *db.ConfigData {
	return &db.ConfigData{
		Admins:       make(map[string]db.AdminInfo),
		ExtraInfo:    "Servidor Depwise Optimizado",
		PublicAccess: true,
		SSHOwners:    make(map[string]string),
		SSHTimeUsers: make(map[string]string),
		ZivpnUsers:   make(map[string]string),
		ZivpnOwners:  make(map[string]string),
		ProxyDT: db.ProxyDTConfig{
			Ports: make(map[string]string),
			Token: "dummy",
		},
	}
}
