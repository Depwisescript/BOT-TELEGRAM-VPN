package bot

import (
	"fmt"
	"math/rand"
	"regexp"
	"strconv"
	"strings"

	"github.com/Depwisescript/BOT-TELEGRAM-VPN/internal/db"
	"github.com/Depwisescript/BOT-TELEGRAM-VPN/internal/sys"
	tele "gopkg.in/telebot.v3"
)

// Steps para la conversacion
var userSteps = make(map[int64]string)
var tempData = make(map[int64]map[string]string) // Guarda usuario, pass, limit temporalmente

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

var lastBotMsg = make(map[int64]*tele.Message)

func handleTextInputs(c tele.Context, b *tele.Bot) error {
	chatID := c.Chat().ID
	step, exists := userSteps[chatID]
	if !exists {
		// No esta en ninguna creacion, ignorar texto
		return nil
	}

	text := strings.TrimSpace(c.Text())

	// Anti-Spam: Borrar el mensaje de texto que el usuario acaba de enviar para mantener limpio el chat
	b.Delete(c.Message())

	markupCancel := &tele.ReplyMarkup{}
	markupCancel.Inline(markupCancel.Row(markupCancel.Data("❌ Cancelar", "cancelar_accion")))

	// Recuperar el ultimo mensaje enviado por el bot para editarlo
	lastMsg, ok := lastBotMsg[chatID]
	if !ok {
		// Si no existe, mandar uno nuevo y guardarlo
		lastMsg, _ = b.Send(c.Chat(), "⏳ Procesando...", tele.ModeHTML)
		lastBotMsg[chatID] = lastMsg
	}

	switch step {
	case "awaiting_ssh_username":
		if !regexp.MustCompile(`^[a-zA-Z0-9_]+$`).MatchString(text) {
			b.Edit(lastMsg, "⚠️ El usuario solo puede contener letras, números y guiones bajos (sin espacios).\n✏️ <i>Intenta con otro:</i>", markupCancel, tele.ModeHTML)
			return nil
		}
		tempData[chatID]["username"] = text
		userSteps[chatID] = "awaiting_ssh_password"

		markupPass := &tele.ReplyMarkup{}
		btnRandom := markupPass.Data("🎲 Generar Aleatoria", "ssh_rnd_pass")
		btnCancel := markupPass.Data("❌ Cancelar", "cancelar_accion")
		markupPass.Inline(markupPass.Row(btnRandom), markupPass.Row(btnCancel))

		b.Edit(lastMsg, fmt.Sprintf("✅ Usuario <code>%s</code> guardado.\n\n🔑 <i>Escribe la contraseña O presiona el botón para generarla:</i>", text), markupPass, tele.ModeHTML)
		return nil

	case "awaiting_ssh_password":
		tempData[chatID]["password"] = text
		userSteps[chatID] = "awaiting_ssh_days"
		b.Edit(lastMsg, "⏳ <i>¿Cuántos días de duración (ej: 30)?</i>", markupCancel, tele.ModeHTML)
		return nil

	case "awaiting_ssh_days":
		days, err := strconv.Atoi(text)
		if err != nil || days <= 0 {
			b.Edit(lastMsg, "⚠️ Por favor envía un número válido mayor a 0.\n⏳ <i>¿Cuántos días de duración (ej: 30)?</i>", markupCancel, tele.ModeHTML)
			return nil
		}
		tempData[chatID]["days"] = text
		userSteps[chatID] = "awaiting_ssh_limit"
		b.Edit(lastMsg, "💻 <i>Límite de conexiones simultáneas (ej: 1 o 2). Envía 0 para ilimitado:</i>", markupCancel, tele.ModeHTML)
		return nil

	case "awaiting_ssh_limit":
		limit, err := strconv.Atoi(text)
		if err != nil || limit < 0 {
			b.Edit(lastMsg, "⚠️ Envía un número válido (0 = ilimitadas).\n💻 <i>Límite de conexiones simultáneas:</i>", markupCancel, tele.ModeHTML)
			return nil
		}
		tempData[chatID]["limit"] = text
		userSteps[chatID] = "awaiting_ssh_quota"
		b.Edit(lastMsg, "📊 <i>Límite de datos en GB (ej: 10 o 5.5). Envía 0 para ilimitado:</i>", markupCancel, tele.ModeHTML)
		return nil

	case "awaiting_ssh_quota":
		quota, err := strconv.ParseFloat(text, 64)
		if err != nil || quota < 0 {
			b.Edit(lastMsg, "⚠️ Envía un número de GB válido (0 = Ilimitado).\n📊 <i>Límite de datos en GB:</i>", markupCancel, tele.ModeHTML)
			return nil
		}
		tempData[chatID]["quota"] = text

		return finishSSHCreation(c, b, chatID, lastMsg)
	case "awaiting_zivpn_pass":
		return finishZivpnCreation(text, chatID, b, lastMsg)

	case "awaiting_delete_user":
		userData, _ := db.Load()

		// Validar permisos
		sa, _ := strconv.ParseInt(superAdmin, 10, 64)
		if chatID != sa {
			if ownerID, ok := userData.SSHOwners[text]; !ok || ownerID != fmt.Sprintf("%d", chatID) {
				b.Edit(lastMsg, "❌ No tienes permisos para borrar este usuario o no existe.\n\n✏️ <i>Intenta con otro:</i>", markupCancel, tele.ModeHTML)
				return nil
			}
		}

		err := sys.DeleteSSHUser(text)
		if err != nil {
			b.Edit(lastMsg, fmt.Sprintf("❌ Error al borrar: %v\n\n✏️ <i>Intenta con otro:</i>", err), markupCancel, tele.ModeHTML)
			return nil
		}

		// Limpiar DB e Interfaz
		delete(userData.SSHOwners, text)
		db.Save(userData)

		delete(userSteps, chatID)
		delete(lastBotMsg, chatID)

		markup := &tele.ReplyMarkup{}
		markup.Inline(markup.Row(markup.Data("🔙 Volver", "menu_eliminar")))
		b.Edit(lastMsg, fmt.Sprintf("✅ Usuario <b>%s</b> eliminado correctamente del servidor.", text), markup, tele.ModeHTML)
		return nil

	default:
		if strings.HasPrefix(step, "awaiting_edit_") {
			return processEditSteps(step, text, chatID, c, b, lastMsg)
		} else if strings.HasPrefix(step, "awaiting_vpn_") {
			return processVPNSteps(step, text, chatID, c, b, lastMsg)
		}
	}

	return nil
}

func handleDeleteSelection(c tele.Context, b *tele.Bot) error {
	user := strings.TrimPrefix(c.Callback().Data, "del_confirm:")
	chatID := c.Chat().ID

	// Validar permisos
	userData, _ := db.Load()
	sa, _ := strconv.ParseInt(superAdmin, 10, 64)
	if chatID != sa {
		if ownerID, ok := userData.SSHOwners[user]; !ok || ownerID != fmt.Sprintf("%d", chatID) {
			return c.Edit("❌ <b>No tienes permisos para borrar este usuario.</b>", tele.ModeHTML)
		}
	}

	err := sys.DeleteSSHUser(user)
	if err != nil {
		return c.Edit(fmt.Sprintf("❌ <b>Error al borrar:</b> %v", err), tele.ModeHTML)
	}

	// Limpiar DB
	delete(userData.SSHOwners, user)
	db.Save(userData)

	markup := &tele.ReplyMarkup{}
	markup.Inline(markup.Row(markup.Data("🔙 Volver", "menu_eliminar")))

	return c.Edit(fmt.Sprintf("✅ Usuario <b>%s</b> eliminado correctamente.", user), markup, tele.ModeHTML)
}

func finishSSHCreation(c tele.Context, b *tele.Bot, chatID int64, lastMsg *tele.Message) error {
	// Limpiar el step
	delete(userSteps, chatID)
	delete(lastBotMsg, chatID)

	mData := tempData[chatID]
	user := mData["username"]
	pass := mData["password"]
	days, _ := strconv.Atoi(mData["days"])
	limit, _ := strconv.Atoi(mData["limit"])
	quota, _ := strconv.ParseFloat(mData["quota"], 64)

	// Avisar que está trabajando
	b.Edit(lastMsg, "⏳ <i>Creando cuenta en el sistema...</i>", tele.ModeHTML)

	// Llamada a nuestro módulo sys nativo en Go
	err := sys.CreateSSHUser(user, pass, days)
	if err != nil {
		b.Edit(lastMsg, fmt.Sprintf("❌ <b>ERROR al crear:</b>\n<pre>%v</pre>", err), tele.ModeHTML)
		return nil
	}

	// Limites
	sys.SetConnectionLimit(user, limit)
	sys.SetDataQuota(user, quota)

	// Guardar en la DB
	dbData, _ := db.Load()
	dbData.SSHOwners[user] = fmt.Sprintf("%d", chatID)
	db.Save(dbData)

	// Formatear Mensaje Éxito (Igualito a Bash V6.7)
	limitStr := mData["limit"]
	if limit == 0 {
		limitStr = "Ilimitado"
	}
	quotaStr := mData["quota"] + " GB"
	if quota == 0 {
		quotaStr = "Ilimitado"
	}

	exito := "✅ <b>NUEVO USUARIO CREADO</b>\n"
	exito += "━━━━━━━━━━━━━━\n"
	exito += fmt.Sprintf("👤 <b>Usuario:</b> <code>%s</code>\n", user)
	exito += fmt.Sprintf("🔑 <b>Pass:</b> <code>%s</code>\n", pass)
	exito += fmt.Sprintf("⏳ <b>Días:</b> %d\n", days)
	exito += fmt.Sprintf("📱 <b>Conexiones:</b> %s\n", limitStr)
	exito += fmt.Sprintf("📊 <b>Datos:</b> %s\n", quotaStr)
	exito += "━━━━━━━━━━━━━━\n"
	exito += "<code>IP: " + sys.GetPublicIP() + "</code>\n"

	// Agregar info de protocolos activos
	data, _ := db.Load()
	protoInfo := ""
	if data.SlowDNS.NS != "" {
		protoInfo += fmt.Sprintf("🐢 <b>SlowDNS NS:</b> <code>%s</code>\n", data.SlowDNS.NS)
		if data.SlowDNS.Key != "" {
			protoInfo += fmt.Sprintf("🔑 <b>SlowDNS Key:</b> <code>%s</code>\n", data.SlowDNS.Key)
		}
	}
	if data.Zivpn {
		protoInfo += "🛰️ <b>ZiVPN UDP:</b> <code>activo</code>\n"
	}
	if data.Falcon != "" {
		protoInfo += fmt.Sprintf("🦅 <b>Falcon Proxy:</b> <code>%s</code>\n", data.Falcon)
	}
	if data.Dropbear != "" {
		protoInfo += fmt.Sprintf("🐻 <b>Dropbear:</b> <code>%s</code>\n", data.Dropbear)
	}
	if data.CloudflareDomain != "" {
		protoInfo += fmt.Sprintf("☁️ <b>Cloudflare DNS:</b> <code>%s</code>\n", data.CloudflareDomain)
	}
	if data.CloudfrontDomain != "" {
		protoInfo += fmt.Sprintf("🚀 <b>Cloudfront DNS:</b> <code>%s</code>\n", data.CloudfrontDomain)
	}
	if data.SSLTunnel != "" {
		protoInfo += fmt.Sprintf("📜 <b>SSL Tunnel:</b> <code>%s</code>\n", data.SSLTunnel)
	}
	if len(data.ProxyDT.Ports) > 0 {
		var pts []string
		for p := range data.ProxyDT.Ports {
			pts = append(pts, "<code>"+p+"</code>")
		}
		protoInfo += fmt.Sprintf("🌐 <b>ProxyDT:</b> %s\n", strings.Join(pts, ", "))
	}

	if protoInfo != "" {
		exito += "━━━━━━━━━━━━━━\n"
		exito += protoInfo
		exito += "━━━━━━━━━━━━━━\n"
	}

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
	return handleStart(c, b) // Llama la funcion desde menu.go
}

func handleRandomPass(c tele.Context, b *tele.Bot) error {
	chatID := c.Chat().ID
	step, exists := userSteps[chatID]
	if !exists || step != "awaiting_ssh_password" {
		return nil
	}

	// Generar random de 6 digitos
	pass := fmt.Sprintf("%06d", rand.Intn(1000000))
	tempData[chatID]["password"] = pass
	userSteps[chatID] = "awaiting_ssh_days"

	markupCancel := &tele.ReplyMarkup{}
	markupCancel.Inline(markupCancel.Row(markupCancel.Data("❌ Cancelar", "cancelar_accion")))

	return c.Edit(fmt.Sprintf("✅ Contraseña <code>%s</code> generada.\n\n⏳ <i>¿Cuántos días de duración (ej: 30)?</i>", pass), markupCancel, tele.ModeHTML)
}
