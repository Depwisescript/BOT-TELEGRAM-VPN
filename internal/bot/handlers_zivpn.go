package bot

import (
	"fmt"

	"github.com/Depwisescript/BOT-TELEGRAM-VPN/internal/db"
	"github.com/Depwisescript/BOT-TELEGRAM-VPN/internal/vpn"
	tele "gopkg.in/telebot.v3"
)

func handleCrearZivpn(c tele.Context, b *tele.Bot) error {
	chatID := c.Chat().ID

	// Solo admins o si es publico (aunque zivpn suele ser mas restrictivo)
	data, _ := db.Load()
	if !data.PublicAccess && !isAdmin(chatID) {
		return c.Edit("⛔ <b>ACCESO DENEGADO</b>", tele.ModeHTML)
	}

	userSteps[chatID] = "awaiting_zivpn_pass"
	lastBotMsg[chatID] = c.Message()

	markup := &tele.ReplyMarkup{}
	markup.Inline(markup.Row(markup.Data("❌ Cancelar", "cancelar_accion")))

	return c.Edit("🛰️ <b>Crear Acceso ZiVPN</b>\n\n🔑 <i>Escribe la contraseña (Password) para el nuevo acceso:</i>", markup, tele.ModeHTML)
}

func finishZivpnCreation(password string, chatID int64, b *tele.Bot, lastMsg *tele.Message) error {
	b.Edit(lastMsg, "⏳ <i>Registrando acceso en ZiVPN...</i>", tele.ModeHTML)

	err := vpn.AddZivpnUser(password)
	if err != nil {
		markup := &tele.ReplyMarkup{}
		markup.Inline(markup.Row(markup.Data("🔙 Volver", "back_main")))
		_, errEdit := b.Edit(lastMsg, fmt.Sprintf("❌ <b>Error al crear acceso ZiVPN:</b>\n<pre>%v</pre>", err), markup, tele.ModeHTML)
		return errEdit
	}

	// Guardar en DB para seguimiento
	data, _ := db.Load()
	if data.ZivpnUsers == nil {
		data.ZivpnUsers = make(map[string]string)
	}
	if data.ZivpnOwners == nil {
		data.ZivpnOwners = make(map[string]string)
	}

	data.ZivpnUsers[password] = "Activado"
	data.ZivpnOwners[password] = fmt.Sprintf("%d", chatID)
	db.Save(data)

	res := "✅ <b>Acceso ZiVPN Creado</b>\n"
	res += "━━━━━━━━━━━━━━\n"
	res += fmt.Sprintf("🔑 <b>Password:</b> <code>%s</code>\n", password)
	res += "━━━━━━━━━━━━━━\n"
	res += "📢 <b>Canal:</b> @Depwise2\n"
	res += "👨‍💻 <b>Dev:</b> @Dan3651\n"
	res += "━━━━━━━━━━━━━━\n"
	res += "<i>El servidor UDP se ha reiniciado satisfactoriamente.</i>"

	markup := &tele.ReplyMarkup{}
	markup.Inline(markup.Row(markup.Data("🔙 Volver", "menu_protocols")))

	_, err = b.Edit(lastMsg, res, markup, tele.ModeHTML)
	return err
}
