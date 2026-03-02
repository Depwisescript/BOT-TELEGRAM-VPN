package bot

import (
	"fmt"
	"math/rand"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/Depwisescript/BOT-TELEGRAM-VPN/internal/db"
	"github.com/Depwisescript/BOT-TELEGRAM-VPN/internal/sys"
	"github.com/Depwisescript/BOT-TELEGRAM-VPN/internal/vpn"
	tele "gopkg.in/telebot.v3"
)

// Steps para la conversacion
var userSteps = make(map[int64]string)
var tempData = make(map[int64]map[string]string) // Guarda usuario, pass, limit temporalmente
var lastBotMsg = make(map[int64]*tele.Message)

func handleCrearSSH(c tele.Context, b *tele.Bot) error {
	chatID := c.Chat().ID

	// Verificar permisos
	data, _ := db.Load()
	if !data.PublicAccess && !isAdmin(chatID) {
		return c.Send("⛔ <b>ACCESO DENEGADO</b>", tele.ModeHTML)
	}

	// 1. Iniciar registro de estado
	userSteps[chatID] = "awaiting_ssh_username"
	tempData[chatID] = make(map[string]string)
	lastBotMsg[chatID] = c.Message()

	markup := &tele.ReplyMarkup{}
	btnCancel := markup.Data("❌ Cancelar", "cancelar_accion")
	markup.Inline(markup.Row(btnCancel))

	return c.Edit("👤 <b>Crear Nuevo Usuario SSH</b>\n\n✏️ <i>Escribe el nombre de usuario que deseas (ej. pepito):</i>", markup, tele.ModeHTML)
}

func handleTextInputs(c tele.Context, b *tele.Bot) error {
	chatID := c.Chat().ID
	step, exists := userSteps[chatID]
	if !exists {
		return nil
	}

	text := strings.TrimSpace(c.Text())
	b.Delete(c.Message())

	markupCancel := &tele.ReplyMarkup{}
	markupCancel.Inline(markupCancel.Row(markupCancel.Data("❌ Cancelar", "cancelar_accion")))

	lastMsg, ok := lastBotMsg[chatID]
	if !ok {
		lastMsg, _ = b.Send(c.Chat(), "⏳ Procesando...", tele.ModeHTML)
		lastBotMsg[chatID] = lastMsg
	}

	switch step {
	case "awaiting_ssh_username":
		if !regexp.MustCompile(`^[a-zA-Z0-9_]+$`).MatchString(text) {
			b.Edit(lastMsg, "⚠️ El usuario solo puede contener letras, números y guiones bajos.\n✏️ <i>Intenta con otro:</i>", markupCancel, tele.ModeHTML)
			return nil
		}
		tempData[chatID]["username"] = text
		userSteps[chatID] = "awaiting_ssh_password"
		markupPass := &tele.ReplyMarkup{}
		btnRandom := markupPass.Data("🎲 Generar Aleatoria", "ssh_rnd_pass")
		btnCancel := markupPass.Data("❌ Cancelar", "cancelar_accion")
		markupPass.Inline(markupPass.Row(btnRandom), markupPass.Row(btnCancel))
		b.Edit(lastMsg, fmt.Sprintf("✅ Usuario <code>%s</code> guardado.\n\n🔑 <i>Escribe la contraseña:</i>", text), markupPass, tele.ModeHTML)
		return nil

	case "awaiting_ssh_password":
		tempData[chatID]["password"] = text
		if !isSuperAdminID(chatID) {
			if isAdmin(chatID) {
				tempData[chatID]["days"] = "7"
				tempData[chatID]["limit"] = "20"
				tempData[chatID]["quota"] = "30"
			} else {
				tempData[chatID]["days"] = "3"
				tempData[chatID]["limit"] = "1"
				tempData[chatID]["quota"] = "6"
			}
			return finishSSHCreation(c, b, chatID, lastMsg)
		}
		userSteps[chatID] = "awaiting_ssh_days"
		b.Edit(lastMsg, "⏳ <i>¿Cuántos días de duración (ej: 30)?</i>", markupCancel, tele.ModeHTML)
		return nil

	case "awaiting_ssh_days":
		days, err := strconv.Atoi(text)
		if err != nil || days <= 0 {
			b.Edit(lastMsg, "⚠️ Valor inválido.\n⏳ <i>Días:</i>", markupCancel, tele.ModeHTML)
			return nil
		}
		tempData[chatID]["days"] = text
		userSteps[chatID] = "awaiting_ssh_limit"
		b.Edit(lastMsg, "💻 <i>Límite de conexiones (0=infinito):</i>", markupCancel, tele.ModeHTML)
		return nil

	case "awaiting_ssh_limit":
		limit, err := strconv.Atoi(text)
		if err != nil || limit < 0 {
			b.Edit(lastMsg, "⚠️ Valor inválido.\n💻 <i>Límite:</i>", markupCancel, tele.ModeHTML)
			return nil
		}
		tempData[chatID]["limit"] = text
		return finishSSHCreation(c, b, chatID, lastMsg)

	case "awaiting_delete_user_selection":
		target := text
		data, _ := db.Load()
		saID := os.Getenv("SUPER_ADMIN")
		isSA := fmt.Sprintf("%d", chatID) == saID

		// 1. Verificar si es SSH
		if ownerID, exists := data.SSHOwners[target]; exists {
			if !isSA && ownerID != fmt.Sprintf("%d", chatID) {
				_, err := b.Edit(lastMsg, "❌ <b>No tienes permiso para borrar este SSH.</b>", markupCancel, tele.ModeHTML)
				return err
			}
			b.Edit(lastMsg, fmt.Sprintf("⏳ <b>Borrando SSH:</b> <code>%s</code>...", target), tele.ModeHTML)
			go func(u string, msg *tele.Message) {
				err := sys.DeleteSSHUser(u)
				db.Update(func(d *db.ConfigData) error {
					delete(d.SSHOwners, u)
					delete(d.SSHTimeUsers, u)
					delete(d.SSHLastActive, u)
					delete(d.SSHHandles, u)
					return nil
				})
				markup := &tele.ReplyMarkup{}
				markup.Inline(markup.Row(markup.Data("🔙 Volver", "menu_eliminar")))
				if err != nil {
					b.Edit(msg, fmt.Sprintf("❌ Error al borrar SSH %s: %v", u, err), markup, tele.ModeHTML)
				} else {
					b.Edit(msg, fmt.Sprintf("✅ Usuario SSH <b>%s</b> eliminado.", u), markup, tele.ModeHTML)
				}
			}(target, lastMsg)

		} else if ownerID, exists := data.ZivpnOwners[target]; exists {
			// 2. Verificar si es ZiVPN
			if !isSA && ownerID != fmt.Sprintf("%d", chatID) {
				_, err := b.Edit(lastMsg, "❌ <b>No tienes permiso para borrar este ZiVPN.</b>", markupCancel, tele.ModeHTML)
				return err
			}
			b.Edit(lastMsg, fmt.Sprintf("⏳ <b>Borrando ZiVPN:</b> <code>%s</code>...", target), tele.ModeHTML)
			go func(p string, msg *tele.Message) {
				err := vpn.RemoveZivpnUser(p)
				db.Update(func(d *db.ConfigData) error {
					delete(d.ZivpnOwners, p)
					delete(d.ZivpnUsers, p)
					delete(d.ZivpnLastActive, p)
					delete(d.ZivpnHandles, p)
					return nil
				})
				markup := &tele.ReplyMarkup{}
				markup.Inline(markup.Row(markup.Data("🔙 Volver", "menu_eliminar")))
				if err != nil {
					b.Edit(msg, fmt.Sprintf("❌ Error al borrar ZiVPN %s: %v", p, err), markup, tele.ModeHTML)
				} else {
					b.Edit(msg, fmt.Sprintf("✅ Acceso ZiVPN <b>%s</b> eliminado.", p), markup, tele.ModeHTML)
				}
			}(target, lastMsg)

		} else {
			_, err := b.Edit(lastMsg, "❌ <b>No existe esa cuenta.</b>\n✏️ <i>Intenta escribirla de nuevo exactamente igual:</i>", markupCancel, tele.ModeHTML)
			return err
		}

		delete(userSteps, chatID)
		delete(lastBotMsg, chatID)
		return nil

	case "awaiting_edit_user_selection":
		user := text
		userData, _ := db.Load()
		sa, _ := strconv.ParseInt(superAdmin, 10, 64)
		if chatID != sa {
			if ownerID, ok := userData.SSHOwners[user]; !ok || ownerID != fmt.Sprintf("%d", chatID) {
				b.Edit(lastMsg, "❌ <b>No permitido o no existe.</b>\n✏️ <i>Intenta otro:</i>", markupCancel, tele.ModeHTML)
				return nil
			}
		} else if _, ok := userData.SSHOwners[user]; !ok {
			b.Edit(lastMsg, "❌ <b>No existe.</b>\n✏️ <i>Intenta otro:</i>", markupCancel, tele.ModeHTML)
			return nil
		}
		tempData[chatID]["edit_target"] = user
		delete(userSteps, chatID)
		return showEditUserMenu(c, b, user)

	case "awaiting_edit_pass_val":
		user := tempData[chatID]["edit_target"]
		err := sys.UpdateSSHUserPassword(user, text)
		delete(userSteps, chatID)
		markup := &tele.ReplyMarkup{}
		markup.Inline(markup.Row(markup.Data("🔙 Menú Editar", "menu_editar")))
		if err != nil {
			b.Edit(lastMsg, "❌ Error: "+err.Error(), markup, tele.ModeHTML)
		} else {
			b.Edit(lastMsg, "✅ Pass cambiado para "+user, markup, tele.ModeHTML)
		}
		return nil

	case "awaiting_edit_renew_val":
		user := tempData[chatID]["edit_target"]
		days, _ := strconv.Atoi(text)
		err := sys.RenewSSHUser(user, days)
		delete(userSteps, chatID)
		markup := &tele.ReplyMarkup{}
		markup.Inline(markup.Row(markup.Data("🔙 Menú Editar", "menu_editar")))
		if err != nil {
			b.Edit(lastMsg, "❌ Error: "+err.Error(), markup, tele.ModeHTML)
		} else {
			b.Edit(lastMsg, fmt.Sprintf("✅ Renovado %d días para %s", days, user), markup, tele.ModeHTML)
		}
		return nil

	case "awaiting_edit_limit_val":
		user := tempData[chatID]["edit_target"]
		limit, _ := strconv.Atoi(text)
		err := sys.SetConnectionLimit(user, limit)
		delete(userSteps, chatID)
		markup := &tele.ReplyMarkup{}
		markup.Inline(markup.Row(markup.Data("🔙 Menú Editar", "menu_editar")))
		if err != nil {
			b.Edit(lastMsg, "❌ Error: "+err.Error(), markup, tele.ModeHTML)
		} else {
			b.Edit(lastMsg, fmt.Sprintf("✅ Límite cambiado para %s", user), markup, tele.ModeHTML)
		}
		return nil

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
	userSteps[chatID] = "awaiting_edit_user_selection"
	tempData[chatID] = make(map[string]string)
	lastBotMsg[chatID] = c.Message()
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
	if c.Callback() != nil {
		return c.Edit(texto, markup, tele.ModeHTML)
	}
	lastMsg := lastBotMsg[c.Chat().ID]
	if lastMsg != nil {
		b.Edit(lastMsg, texto, markup, tele.ModeHTML)
		return nil
	}
	return c.Send(texto, markup, tele.ModeHTML)
}

func handleEditSelection(c tele.Context, b *tele.Bot) error {
	return handleMenuEditar(c, b)
}

func handleDeleteSelection(c tele.Context, b *tele.Bot) error {
	return handleMenuEliminar(c, b)
}

func finishSSHCreation(c tele.Context, b *tele.Bot, chatID int64, lastMsg *tele.Message) error {
	delete(userSteps, chatID)
	delete(lastBotMsg, chatID)
	mData := tempData[chatID]
	user := mData["username"]
	pass := mData["password"]
	days, _ := strconv.Atoi(mData["days"])
	limit, _ := strconv.Atoi(mData["limit"])

	b.Edit(lastMsg, "⏳ <i>Creando cuenta en el sistema...</i>", tele.ModeHTML)

	// Crear usuario en el sistema
	err := sys.CreateSSHUser(user, pass, days)
	if err != nil {
		b.Edit(lastMsg, fmt.Sprintf("❌ <b>ERROR:</b>\n<pre>%v</pre>", err), tele.ModeHTML)
		return nil
	}

	// Aplicar limites
	sys.SetConnectionLimit(user, limit)

	// Guardar en DB
	dbData, _ := db.Load()
	dbData.SSHOwners[user] = fmt.Sprintf("%d", chatID)
	// Guardar @handle si existe
	if c.Sender() != nil && c.Sender().Username != "" {
		dbData.SSHHandles[user] = "@" + c.Sender().Username
	}
	db.Save(dbData)

	// Formatear Mensaje Éxito Detallado
	limitStr := mData["limit"]
	if limit == 0 {
		limitStr = "Ilimitado"
	}

	exito := "✅ <b>NUEVO USUARIO CREADO</b>\n"
	exito += "━━━━━━━━━━━━━━\n"
	exito += fmt.Sprintf("👤 <b>Usuario:</b> <code>%s</code>\n", user)
	exito += fmt.Sprintf("🔑 <b>Pass:</b> <code>%s</code>\n", pass)
	exito += fmt.Sprintf("⏳ <b>Días:</b> %d\n", days)
	exito += fmt.Sprintf("📱 <b>Conexiones:</b> %s\n", limitStr)
	exito += "━━━━━━━━━━━━━━\n"
	exito += "<code>IP: " + sys.GetPublicIP() + "</code>\n"

	// Info de protocolos
	protoInfo := ""
	if dbData.SlowDNS.NS != "" {
		protoInfo += fmt.Sprintf("🐢 <b>SlowDNS NS:</b> <code>%s</code>\n", dbData.SlowDNS.NS)
		if dbData.SlowDNS.Key != "" {
			protoInfo += fmt.Sprintf("🔑 <b>SlowDNS Key:</b> <code>%s</code>\n", dbData.SlowDNS.Key)
		}
	}
	if dbData.Zivpn {
		protoInfo += "🛰️ <b>ZiVPN UDP:</b> <code>activo</code>\n"
	}
	if dbData.Falcon != "" {
		protoInfo += fmt.Sprintf("🦅 <b>Falcon Proxy:</b> <code>%s</code>\n", dbData.Falcon)
	}
	if dbData.Dropbear != "" {
		protoInfo += fmt.Sprintf("🐻 <b>Dropbear:</b> <code>%s</code>\n", dbData.Dropbear)
	}
	if dbData.CloudflareDomain != "" {
		protoInfo += fmt.Sprintf("☁️ <b>Cloudflare DNS:</b> <code>%s</code>\n", dbData.CloudflareDomain)
	}
	if dbData.CloudfrontDomain != "" {
		protoInfo += fmt.Sprintf("🚀 <b>Cloudfront DNS:</b> <code>%s</code>\n", dbData.CloudfrontDomain)
	}
	if dbData.SSLTunnel != "" {
		protoInfo += fmt.Sprintf("📜 <b>SSL Tunnel:</b> <code>%s</code>\n", dbData.SSLTunnel)
	}
	if len(dbData.ProxyDT.Ports) > 0 {
		var pts []string
		for p := range dbData.ProxyDT.Ports {
			pts = append(pts, "<code>"+p+"</code>")
		}
		protoInfo += fmt.Sprintf("🌐 <b>ProxyDT:</b> %s\n", strings.Join(pts, ", "))
	}

	if protoInfo != "" {
		exito += "━━━━━━━━━━━━━━\n"
		exito += protoInfo
		exito += "━━━━━━━━━━━━━━\n"
	}

	exito += "📢 <b>Canal:</b> @Depwise2\n"
	exito += "👨‍💻 <b>Dev:</b> @Dan3651\n"

	markup := &tele.ReplyMarkup{}
	markup.Inline(markup.Row(markup.Data("🔙 Ir al Menú", "menu_crear")))

	b.Edit(lastMsg, exito, markup, tele.ModeHTML)
	return nil
}

func handleCancel(c tele.Context, b *tele.Bot) error {
	chatID := c.Chat().ID
	delete(userSteps, chatID)
	delete(tempData, chatID)
	delete(lastBotMsg, chatID)
	return handleStart(c, b)
}

func handleRandomPass(c tele.Context, b *tele.Bot) error {
	chatID := c.Chat().ID
	pass := fmt.Sprintf("%06d", rand.Intn(1000000))
	tempData[chatID]["password"] = pass
	lastMsg := lastBotMsg[chatID]
	if !isSuperAdminID(chatID) {
		return finishSSHCreation(c, b, chatID, lastMsg)
	}
	userSteps[chatID] = "awaiting_ssh_days"
	_, err := b.Edit(lastMsg, "✅ Pass: "+pass+"\n⏳ Días:", tele.ModeHTML)
	return err
}

func HandleEditPass(c tele.Context, b *tele.Bot) error {
	chatID := c.Chat().ID
	user := tempData[chatID]["edit_target"]
	userSteps[chatID] = "awaiting_edit_pass_val"
	lastBotMsg[chatID] = c.Message()
	markup := &tele.ReplyMarkup{}
	markup.Inline(markup.Row(markup.Data("❌ Cancelar", "cancelar_accion")))
	return c.Edit(fmt.Sprintf("🔑 <b>Cambiando Pass:</b> <code>%s</code>\n✏️ Nueva pass:", user), markup, tele.ModeHTML)
}

func HandleEditRenew(c tele.Context, b *tele.Bot) error {
	chatID := c.Chat().ID
	user := tempData[chatID]["edit_target"]
	userSteps[chatID] = "awaiting_edit_renew_val"
	lastBotMsg[chatID] = c.Message()
	markup := &tele.ReplyMarkup{}
	markup.Inline(markup.Row(markup.Data("❌ Cancelar", "cancelar_accion")))
	return c.Edit(fmt.Sprintf("📅 <b>Renovando:</b> <code>%s</code>\n✏️ ¿Días extra?", user), markup, tele.ModeHTML)
}

func HandleEditLimit(c tele.Context, b *tele.Bot) error {
	chatID := c.Chat().ID
	user := tempData[chatID]["edit_target"]
	userSteps[chatID] = "awaiting_edit_limit_val"
	lastBotMsg[chatID] = c.Message()
	markup := &tele.ReplyMarkup{}
	markup.Inline(markup.Row(markup.Data("❌ Cancelar", "cancelar_accion")))
	return c.Edit(fmt.Sprintf("📱 <b>Límite:</b> <code>%s</code>\n✏️ Nuevo límite (0=inf):", user), markup, tele.ModeHTML)
}
