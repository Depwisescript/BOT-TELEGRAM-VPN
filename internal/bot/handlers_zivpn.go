package bot

import (
	"fmt"
	"html"
	"strconv"
	"strings"
	"time"

	"github.com/Depwisescript/BOT-TELEGRAM-VPN/internal/db"
	"github.com/Depwisescript/BOT-TELEGRAM-VPN/internal/sys"
	"github.com/Depwisescript/BOT-TELEGRAM-VPN/internal/vpn"
	tele "gopkg.in/telebot.v3"
)

func handleCrearZivpn(c tele.Context, b *tele.Bot) error {
	chatID := c.Chat().ID

	// Solo admins o si es publico
	data, _ := db.Load()
	if !data.PublicAccess && !isAdmin(chatID) {
		return c.Edit("⛔ <b>ACCESO DENEGADO</b>", tele.ModeHTML)
	}

	UserSteps[chatID] = "awaiting_zivpn_pass"
	TempData[chatID] = make(map[string]string)
	lastMsg, _ := LastBotMsg[chatID]

	markup := &tele.ReplyMarkup{}
	markup.Inline(markup.Row(markup.Data("❌ Cancelar", "cancelar_accion")))

	_, err := SafeEdit(chatID, b, lastMsg, "🛰️ <b>Crear Acceso ZiVPN</b>\n\n🔑 <i>Escribe la contraseña (Password) para el nuevo acceso:</i>", markup)
	return err
}

func finishZivpnCreation(c tele.Context, password string, days int, chatID int64, b *tele.Bot, lastMsg *tele.Message) error {
	// Bloquear estado inmediatamente para evitar spam/carreras
	delete(UserSteps, chatID)
	delete(LastBotMsg, chatID)

	SafeEdit(chatID, b, lastMsg, "⏳ <i>Registrando acceso en ZiVPN...</i>", nil)

	err := vpn.AddZivpnUser(password)
	if err != nil {
		markup := &tele.ReplyMarkup{}
		markup.Inline(markup.Row(markup.Data("🔙 Volver", "back_main")))
		_, errEdit := SafeEdit(chatID, b, lastMsg, fmt.Sprintf("❌ <b>Error al crear acceso ZiVPN:</b>\n<pre>%v</pre>", err), markup)
		return errEdit
	}

	// Guardar en DB con fecha de expiración
	expireDate := time.Now().AddDate(0, 0, days).Format("2006-01-02")

	db.Update(func(data *db.ConfigData) error {
		if data.ZivpnUsers == nil {
			data.ZivpnUsers = make(map[string]string)
		}
		if data.ZivpnOwners == nil {
			data.ZivpnOwners = make(map[string]string)
		}
		data.ZivpnUsers[password] = expireDate
		data.ZivpnOwners[password] = fmt.Sprintf("%d", chatID)
		// Guardar @handle
		if c != nil && c.Sender() != nil && c.Sender().Username != "" {
			data.ZivpnHandles[password] = "@" + c.Sender().Username
		}
		// Inicializar actividad
		data.ZivpnLastActive[password] = time.Now().Format(time.RFC3339)
		return nil
	})

	// Construir mensaje de éxito con toda la info
	data, _ := db.Load()

	res := "✅ <b>Acceso ZiVPN Creado</b>\n"
	res += "━━━━━━━━━━━━━━\n"
	res += fmt.Sprintf("🔑 <b>Password:</b> <code>%s</code>\n", html.EscapeString(password))
	res += fmt.Sprintf("⏳ <b>Días:</b> %d\n", days)
	res += fmt.Sprintf("📅 <b>Expira:</b> <code>%s</code>\n", expireDate)
	res += "━━━━━━━━━━━━━━\n"
	res += fmt.Sprintf("🌐 <b>IP:</b> <code>%s</code>\n", sys.GetPublicIP())

	if data.CloudflareDomain != "" {
		res += fmt.Sprintf("☁️ <b>Cloudflare:</b> <code>%s</code>\n", data.CloudflareDomain)
	}
	if data.CloudfrontDomain != "" {
		res += fmt.Sprintf("🚀 <b>Cloudfront:</b> <code>%s</code>\n", data.CloudfrontDomain)
	}

	res += "━━━━━━━━━━━━━━\n"
	res += "📢 <b>Canal:</b> @Depwise2\n"
	res += "👨‍💻 <b>Dev:</b> @Dan3651\n"

	markup := &tele.ReplyMarkup{}
	markup.Inline(markup.Row(markup.Data("🔙 Volver", "menu_protocols")))

	_, err = SafeEdit(chatID, b, lastMsg, res, markup)
	return err
}

// processZivpnSteps maneja los pasos de creación de ZiVPN
func processZivpnSteps(step string, text string, chatID int64, c tele.Context, b *tele.Bot, lastMsg *tele.Message) error {
	markupCancel := &tele.ReplyMarkup{}
	markupCancel.Inline(markupCancel.Row(markupCancel.Data("❌ Cancelar", "cancelar_accion")))

	switch step {
	case "awaiting_zivpn_pass":
		password := strings.TrimSpace(text)
		if len(password) < 1 {
			_, err := SafeEdit(chatID, b, lastMsg, "⚠️ La contraseña no puede estar vacía.\n\n🔑 <i>Escribe la contraseña:</i>", markupCancel)
			return err
		}

		TempData[chatID]["zivpn_pass"] = password

		// Determinar días según rol
		if isSuperAdminID(chatID) {
			// SuperAdmin: pedir días libremente
			UserSteps[chatID] = "awaiting_zivpn_days"
			_, err := SafeEdit(chatID, b, lastMsg, fmt.Sprintf("✅ Password <code>%s</code> guardada.\n\n⏳ <i>¿Cuántos días de duración? (ej: 30):</i>", html.EscapeString(password)), markupCancel)
			return err
		}

		// Admin: 7 días automático | Público: 3 días automático
		days := 3
		if isAdmin(chatID) {
			days = 7
		}

		return finishZivpnCreation(c, password, days, chatID, b, lastMsg)

	case "awaiting_zivpn_days":
		days, err := strconv.Atoi(strings.TrimSpace(text))
		if err != nil || days <= 0 {
			_, err := SafeEdit(chatID, b, lastMsg, "⚠️ Por favor envía un número válido mayor a 0.\n\n⏳ <i>¿Cuántos días de duración?</i>", markupCancel)
			return err
		}

		password := TempData[chatID]["zivpn_pass"]
		return finishZivpnCreation(c, password, days, chatID, b, lastMsg)
	}

	return nil
}
