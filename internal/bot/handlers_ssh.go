package bot

import (
	"fmt"
	"html"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/Depwisescript/BOT-TELEGRAM-VPN/internal/db"
	"github.com/Depwisescript/BOT-TELEGRAM-VPN/internal/sys"
	tele "gopkg.in/telebot.v3"
)

func handleCrearSSH(c tele.Context, b *tele.Bot) error {
	chatID := c.Chat().ID

	// 1. Iniciar registro de estado
	UserSteps[chatID] = "awaiting_ssh_username"
	TempData[chatID] = make(map[string]string)

	markup := &tele.ReplyMarkup{}
	btnCancel := markup.Data("❌ Cancelar", "cancelar_accion")
	markup.Inline(markup.Row(btnCancel))

	lastMsg, _ := LastBotMsg[chatID]
	msg, _ := SafeEdit(chatID, b, lastMsg, "👤 <b>Crear Nuevo Usuario SSH</b>\n\n✏️ <i>Escribe el nombre de usuario que deseas (ej. pepito):</i>", markup)
	LastBotMsg[chatID] = msg
	return nil
}

func handleTextInputs(c tele.Context, b *tele.Bot) error {
	chatID := c.Chat().ID
	text := c.Text()
	step, ok := UserSteps[chatID]
	if !ok {
		return nil
	}

	// Borrar el mensaje del usuario de inmediato para mantener el chat limpio (Sink Global)
	_ = c.Delete()

	markupCancel := &tele.ReplyMarkup{}
	markupCancel.Inline(markupCancel.Row(markupCancel.Data("❌ Cancelar", "cancelar_accion")))

	// Dispatcher para otros protocolos
	if strings.HasPrefix(step, "awaiting_zivpn_") {
		lastMsg := LastBotMsg[chatID]
		return processZivpnSteps(step, text, chatID, c, b, lastMsg)
	}
	if strings.HasPrefix(step, "awaiting_vpn_") || strings.HasPrefix(step, "awaiting_quota_") {
		lastMsg := LastBotMsg[chatID]
		return processVPNSteps(step, text, chatID, c, b, lastMsg)
	}
	if strings.HasPrefix(step, "awaiting_scan_") {
		lastMsg, _ := LastBotMsg[chatID]
		return processScannerSteps(step, text, chatID, c, b, lastMsg)
	}

	lastMsg, _ := LastBotMsg[chatID]
	textLower := strings.ToLower(strings.TrimSpace(text))

	// Interceptar comandos de navegación para cancelar estado
	if strings.HasPrefix(text, "/") || textLower == "menu" || textLower == "salir" || textLower == "atrás" || textLower == "atras" || textLower == "cancelar" {
		delete(UserSteps, chatID)
		delete(TempData, chatID)
		return handleStart(c, b)
	}

	switch step {
	case "awaiting_ssh_username":
		if !regexp.MustCompile(`^[a-zA-Z0-9_]+$`).MatchString(text) {
			_, err := SafeEdit(chatID, b, lastMsg, "⚠️ El usuario solo puede contener letras, números y guiones bajos.\n✏️ <i>Intenta con otro:</i>", markupCancel)
			return err
		}
		TempData[chatID]["username"] = text
		UserSteps[chatID] = "awaiting_ssh_password"
		markupPass := &tele.ReplyMarkup{}
		btnRandom := markupPass.Data("🎲 Generar Aleatoria", "ssh_rnd_pass")
		btnCancel := markupPass.Data("❌ Cancelar", "cancelar_accion")
		markupPass.Inline(markupPass.Row(btnRandom), markupPass.Row(btnCancel))
		_, err := SafeEdit(chatID, b, lastMsg, fmt.Sprintf("✅ Usuario <code>%s</code> guardado.\n\n🔑 <i>Escribe la contraseña:</i>", html.EscapeString(text)), markupPass)
		return err

	case "awaiting_ssh_password":
		TempData[chatID]["password"] = text
		if !isSuperAdminID(chatID) {
			data, _ := db.Load()
			if isAdmin(chatID) {
				TempData[chatID]["days"] = strconv.Itoa(data.GetMaxDaysAdmin())
				TempData[chatID]["limit"] = strconv.Itoa(data.GetMaxLimitAdmin())
			} else {
				TempData[chatID]["days"] = strconv.Itoa(data.GetMaxDaysPublic())
				TempData[chatID]["limit"] = strconv.Itoa(data.GetMaxLimitPublic())
			}
			return finishSSHCreation(c, b, chatID, lastMsg)
		}
		UserSteps[chatID] = "awaiting_ssh_days"
		_, err := SafeEdit(chatID, b, lastMsg, "⏳ <i>¿Cuántos días de duración (ej: 30)?</i>", markupCancel)
		return err

	case "awaiting_ssh_days":
		days, err := strconv.Atoi(text)
		if err != nil || days <= 0 {
			_, err := SafeEdit(chatID, b, lastMsg, "⚠️ Valor inválido.\n⏳ <i>Días:</i>", markupCancel)
			return err
		}
		TempData[chatID]["days"] = text
		UserSteps[chatID] = "awaiting_ssh_limit"
		_, err = SafeEdit(chatID, b, lastMsg, "💻 <i>Límite de conexiones (0=infinito):</i>", markupCancel)
		return err

	case "awaiting_ssh_limit":
		limit, err := strconv.Atoi(text)
		if err != nil || limit < 0 {
			_, err := SafeEdit(chatID, b, lastMsg, "⚠️ Valor inválido.\n💻 <i>Límite:</i>", markupCancel)
			return err
		}
		TempData[chatID]["limit"] = text
		return finishSSHCreation(c, b, chatID, lastMsg)

	case "awaiting_broadcast":
		msg := "📢 <b>MENSAJE GLOBAL DE ADMIN:</b>\n\n" + text
		data, _ := db.Load()
		success := 0
		for _, id := range data.UserHistory {
			_, err := b.Send(tele.ChatID(id), msg, tele.ModeHTML)
			if err == nil {
				success++
			}
		}
		delete(UserSteps, chatID)
		return c.Send(fmt.Sprintf("✅ Broadcast enviado a %d usuarios.", success))

	case "awaiting_edit_user_selection":
		user := text
		userData, _ := db.Load()
		sa, _ := strconv.ParseInt(superAdmin, 10, 64)
		if chatID != sa {
			if ownerID, ok := userData.SSHOwners[user]; !ok || ownerID != fmt.Sprintf("%d", chatID) {
				SafeEdit(chatID, b, lastMsg, "❌ <b>No permitido o no existe.</b>\n✏️ <i>Intenta otro:</i>", markupCancel)
				return nil
			}
		} else if _, ok := userData.SSHOwners[user]; !ok {
			SafeEdit(chatID, b, lastMsg, "❌ <b>No existe.</b>\n✏️ <i>Intenta otro:</i>", markupCancel)
			return nil
		}
		TempData[chatID]["edit_target"] = user
		delete(UserSteps, chatID)
		return showEditUserMenu(c, b, user)

	case "awaiting_edit_pass_val":
		user := TempData[chatID]["edit_target"]
		err := sys.UpdateSSHUserPassword(user, text)
		delete(UserSteps, chatID)
		markup := &tele.ReplyMarkup{}
		markup.Inline(markup.Row(markup.Data("🔙 Menú Editar", "menu_editar")))
		if err != nil {
			SafeEdit(chatID, b, lastMsg, "❌ Error: "+err.Error(), markup)
		} else {
			SafeEdit(chatID, b, lastMsg, "✅ Pass cambiado para "+user, markup)
		}
		return nil

	case "awaiting_edit_renew_val":
		user := TempData[chatID]["edit_target"]
		days, _ := strconv.Atoi(text)
		err := sys.RenewSSHUser(user, days)
		delete(UserSteps, chatID)
		markup := &tele.ReplyMarkup{}
		markup.Inline(markup.Row(markup.Data("🔙 Menú Editar", "menu_editar")))
		if err != nil {
			SafeEdit(chatID, b, lastMsg, "❌ Error: "+err.Error(), markup)
		} else {
			SafeEdit(chatID, b, lastMsg, fmt.Sprintf("✅ Renovado %d días para %s", days, user), markup)
		}
		return nil

	case "awaiting_edit_limit_val":
		user := TempData[chatID]["edit_target"]
		limit, _ := strconv.Atoi(text)
		err := sys.SetConnectionLimit(user, limit)
		delete(UserSteps, chatID)
		markup := &tele.ReplyMarkup{}
		markup.Inline(markup.Row(markup.Data("🔙 Menú Editar", "menu_editar")))
		if err != nil {
			SafeEdit(chatID, b, lastMsg, "❌ Error: "+err.Error(), markup)
		} else {
			SafeEdit(chatID, b, lastMsg, fmt.Sprintf("✅ Límite cambiado para %s", user), markup)
		}
		return nil

	case "awaiting_delete_user_selection":
		return processDeleteSteps(text, chatID, c, b)
	}

	return nil
}

func handleMenuEditar(c tele.Context, b *tele.Bot) error {
	chatID := c.Chat().ID
	data, _ := db.Load()
	sa, _ := strconv.ParseInt(superAdmin, 10, 64)
	isSA := chatID == sa
	res := "✏️ <b>EDITAR USUARIO</b>\n━━━━━━━━━━━━━━\n"
	count := 0
	for user, ownerID := range data.SSHOwners {
		if isSA || ownerID == fmt.Sprintf("%d", chatID) {
			handle := data.SSHHandles[user]
			if handle != "" {
				res += fmt.Sprintf("👤 <code>%s</code> (%s)\n", user, handle)
			} else {
				res += "👤 <code>" + user + "</code>\n"
			}
			count++
		}
	}
	markup := &tele.ReplyMarkup{}
	markup.Inline(markup.Row(markup.Data("🔙 Volver", "back_main")))
	if count == 0 {
		return c.Edit("❌ No hay usuarios.", markup, tele.ModeHTML)
	}
	res += "━━━━━━━━━━━━━━\n✏️ Escribe el nombre del usuario:"
	UserSteps[chatID] = "awaiting_edit_user_selection"
	TempData[chatID] = make(map[string]string)
	return c.Edit(res, markup, tele.ModeHTML)
}

func showEditUserMenu(c tele.Context, b *tele.Bot, user string) error {
	markup := &tele.ReplyMarkup{}
	btnPass := markup.Data("🔑 Pass", "edit_pass")
	btnRenew := markup.Data("📅 Renov", "edit_renew")
	btnLimit := markup.Data("📱 Lim", "edit_limit")
	btnBack := markup.Data("🔙 Volver", "menu_editar")
	markup.Inline(markup.Row(btnPass, btnRenew), markup.Row(btnLimit), markup.Row(btnBack))
	texto := fmt.Sprintf("✏️ <b>EDITAR:</b> <code>%s</code>", user)

	lastMsg, _ := LastBotMsg[c.Chat().ID]
	_, err := SafeEdit(c.Chat().ID, b, lastMsg, texto, markup)
	return err
}

func finishSSHCreation(c tele.Context, b *tele.Bot, chatID int64, lastMsg *tele.Message) error {
	// Bloquear estado inmediatamente para evitar spam/carreras
	delete(UserSteps, chatID)
	delete(LastBotMsg, chatID)

	mData := TempData[chatID]
	user := mData["username"]
	pass := mData["password"]
	days, _ := strconv.Atoi(mData["days"])
	limit, _ := strconv.Atoi(mData["limit"])

	SafeEdit(chatID, b, lastMsg, "⏳ <i>Creando cuenta en el sistema...</i>", nil)

	// Crear usuario en el sistema
	err := sys.CreateSSHUser(user, pass, days)
	if err != nil {
		markup := &tele.ReplyMarkup{}
		markup.Inline(markup.Row(markup.Data("🔙 Volver", "back_main")))
		SafeEdit(chatID, b, lastMsg, fmt.Sprintf("❌ <b>ERROR:</b>\n<pre>%v</pre>", err), markup)
		return err
	}

	// Aplicar límite
	sys.SetConnectionLimit(user, limit)

	// Guardar en DB
	db.Update(func(data *db.ConfigData) error {
		if data.SSHOwners == nil {
			data.SSHOwners = make(map[string]string)
		}
		data.SSHOwners[user] = fmt.Sprintf("%d", chatID)
		if c.Sender() != nil && c.Sender().Username != "" {
			if data.SSHHandles == nil {
				data.SSHHandles = make(map[string]string)
			}
			data.SSHHandles[user] = "@" + c.Sender().Username
		}
		return nil
	})

	// Respuesta final
	ip := sys.GetPublicIP()
	dataFinal, _ := db.Load()
	res := "✅ <b>Usuario SSH Creado</b>\n"
	res += "━━━━━━━━━━━━━━\n"
	res += fmt.Sprintf("👤 <b>User:</b> <code>%s</code>\n", user)
	res += fmt.Sprintf("🔑 <b>Pass:</b> <code>%s</code>\n", html.EscapeString(pass))
	res += fmt.Sprintf("⏳ <b>Días:</b> %d\n", days)
	res += fmt.Sprintf("💻 <b>Límite:</b> %d\n", limit)
	res += "━━━━━━━━━━━━━━\n"
	res += fmt.Sprintf("🌐 <b>IP:</b> <code>%s</code>\n", ip)
	res += "🔌 <b>SSH Port:</b> <code>22</code>\n"
	if dataFinal.SSHWebSocket {
		res += "🌐 <b>WS:</b>  <code>ws://" + ip + ":80</code>\n"
		res += "🔒 <b>WSS:</b> <code>wss://" + ip + ":443</code>\n"
	}
	res += "━━━━━━━━━━━━━━\n"

	markup := &tele.ReplyMarkup{}
	markup.Inline(markup.Row(markup.Data("🔙 Volver", "back_main")))
	_, err = SafeEdit(chatID, b, lastMsg, res, markup)
	return err
}

func handleCancel(c tele.Context, b *tele.Bot) error {
	chatID := c.Chat().ID
	delete(UserSteps, chatID)
	delete(TempData, chatID)
	delete(LastBotMsg, chatID)
	return handleStart(c, b)
}

func handleRandomPass(c tele.Context, b *tele.Bot) error {
	chatID := c.Chat().ID
	pass := fmt.Sprintf("%06d", 100000+((time.Now().UnixNano()/1000)%900000))
	TempData[chatID]["password"] = pass
	lastMsg, _ := LastBotMsg[chatID]

	if !isSuperAdminID(chatID) {
		return finishSSHCreation(c, b, chatID, lastMsg)
	}

	UserSteps[chatID] = "awaiting_ssh_days"
	markupCancel := &tele.ReplyMarkup{}
	markupCancel.Inline(markupCancel.Row(markupCancel.Data("❌ Cancelar", "cancelar_accion")))

	_, err := SafeEdit(chatID, b, lastMsg, "✅ Pass: "+pass+"\n⏳ Días:", markupCancel)
	return err
}

func HandleEditPass(c tele.Context, b *tele.Bot) error {
	chatID := c.Chat().ID
	user := TempData[chatID]["edit_target"]
	UserSteps[chatID] = "awaiting_edit_pass_val"
	markupCancel := &tele.ReplyMarkup{}
	markupCancel.Inline(markupCancel.Row(markupCancel.Data("❌ Cancelar", "cancelar_accion")))
	return c.Edit(fmt.Sprintf("🔑 <b>Cambiando Pass:</b> <code>%s</code>\n✏️ Nueva pass:", user), markupCancel, tele.ModeHTML)
}

func HandleEditRenew(c tele.Context, b *tele.Bot) error {
	chatID := c.Chat().ID
	user := TempData[chatID]["edit_target"]
	UserSteps[chatID] = "awaiting_edit_renew_val"
	markupCancel := &tele.ReplyMarkup{}
	markupCancel.Inline(markupCancel.Row(markupCancel.Data("❌ Cancelar", "cancelar_accion")))
	return c.Edit(fmt.Sprintf("📅 <b>Renovando:</b> <code>%s</code>\n✏️ ¿Días extra?", user), markupCancel, tele.ModeHTML)
}

func HandleEditLimit(c tele.Context, b *tele.Bot) error {
	chatID := c.Chat().ID
	user := TempData[chatID]["edit_target"]
	UserSteps[chatID] = "awaiting_edit_limit_val"
	markupCancel := &tele.ReplyMarkup{}
	markupCancel.Inline(markupCancel.Row(markupCancel.Data("❌ Cancelar", "cancelar_accion")))
	return c.Edit(fmt.Sprintf("📱 <b>Límite:</b> <code>%s</code>\n✏️ Nuevo límite (0=inf):", user), markupCancel, tele.ModeHTML)
}

func handleEditSelection(c tele.Context, b *tele.Bot) error {
	return handleMenuEditar(c, b)
}

func handleDeleteSelection(c tele.Context, b *tele.Bot) error {
	return handleMenuEliminar(c, b)
}
