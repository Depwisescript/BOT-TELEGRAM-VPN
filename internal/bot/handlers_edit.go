package bot

import (
	"fmt"
	"strconv"
	"strings"

	tele "gopkg.in/telebot.v3"
	"github.com/Depwisescript/BOT-TELEGRAM-VPN/internal/sys"
)

// Interceptar "Editar SSH" y renovar / cambiar pass
func handleMenuEditar(c tele.Context, b *tele.Bot) error {
	chatID := c.Chat().ID
	
	// Preguntar qué usuario editar
	userSteps[chatID] = "awaiting_edit_user"
	lastBotMsg[chatID] = c.Message()
	
	markup := &tele.ReplyMarkup{}
	markup.Inline(markup.Row(markup.Data("❌ Cancelar", "cancelar_accion")))
	
	return c.Edit("✏️ <b>Editar Usuario SSH</b>\n\n📝 <i>Dime el nombre del usuario a modificar:</i>", markup, tele.ModeHTML)
}

func handleEditMenu(c tele.Context, user string, b *tele.Bot, lastMsg *tele.Message) error {
	chatID := c.Chat().ID
	tempData[chatID] = make(map[string]string)
	tempData[chatID]["username"] = user
	
	markup := &tele.ReplyMarkup{}
	btnPass := markup.Data("🔑 Cambiar Contraseña", "edit_pass")
	btnRenew := markup.Data("⏳ Renovar Días", "edit_renew")
	btnCancel := markup.Data("❌ Cancelar", "cancelar_accion")
	
	markup.Inline(
		markup.Row(btnPass, btnRenew),
		markup.Row(btnCancel),
	)

	b.Edit(lastMsg, fmt.Sprintf("⚙️ <b>Opciones para:</b> <code>%s</code>", user), markup, tele.ModeHTML)
	return nil
}

func handleEditPass(c tele.Context, b *tele.Bot) error {
	chatID := c.Chat().ID
	user := tempData[chatID]["username"]
	
	userSteps[chatID] = "awaiting_edit_pass_val"
	lastBotMsg[chatID] = c.Message()
	
	markup := &tele.ReplyMarkup{}
	markup.Inline(markup.Row(markup.Data("❌ Cancelar", "cancelar_accion")))
	
	return c.Edit(fmt.Sprintf("🔑 <b>Cambiando contraseña de:</b> <code>%s</code>\n\n✏️ <i>Escribe la nueva contraseña:</i>", user), markup, tele.ModeHTML)
}

func handleEditRenew(c tele.Context, b *tele.Bot) error {
	chatID := c.Chat().ID
	user := tempData[chatID]["username"]
	
	userSteps[chatID] = "awaiting_edit_renew_val"
	lastBotMsg[chatID] = c.Message()
	
	markup := &tele.ReplyMarkup{}
	markup.Inline(markup.Row(markup.Data("❌ Cancelar", "cancelar_accion")))
	
	return c.Edit(fmt.Sprintf("⏳ <b>Renovando expiración de:</b> <code>%s</code>\n\n📆 <i>¿Cuántos días extra quieres agregarle? (ej: 30)</i>", user), markup, tele.ModeHTML)
}

// Interceptar las entradas de texto para edicion
func processEditSteps(step string, text string, chatID int64, c tele.Context, b *tele.Bot, lastMsg *tele.Message) error {
	user := tempData[chatID]["username"]
	
	markup := &tele.ReplyMarkup{}
	markup.Inline(markup.Row(markup.Data("🔙 Volver al Menú", "back_main")))

	switch step {
	case "awaiting_edit_user":
		userSteps[chatID] = "editing_menu_open"
		return handleEditMenu(c, text, b, lastMsg)

	case "awaiting_edit_pass_val":
		err := sys.UpdateSSHUserPassword(user, text)
		delete(userSteps, chatID)
		delete(lastBotMsg, chatID)
		if err != nil {
			b.Edit(lastMsg, fmt.Sprintf("❌ <b>Error cambiando contraseña:</b> %v", err), markup, tele.ModeHTML)
			return nil
		}
		b.Edit(lastMsg, fmt.Sprintf("✅ La contraseña de <code>%s</code> ha sido cambiada a: <code>%s</code>", user, text), markup, tele.ModeHTML)
		return nil

	case "awaiting_edit_renew_val":
		days, err := strconv.Atoi(strings.TrimSpace(text))
		if err != nil || days <= 0 {
			b.Edit(lastMsg, "⚠️ Debes enviar un número de días válido.", markup, tele.ModeHTML)
			return nil
		}
		
		err = sys.RenewSSHUser(user, days)
		delete(userSteps, chatID)
		delete(lastBotMsg, chatID)
		if err != nil {
			b.Edit(lastMsg, fmt.Sprintf("❌ <b>Error durante renovación:</b> %v", err), markup, tele.ModeHTML)
			return nil
		}
		b.Edit(lastMsg, fmt.Sprintf("✅ <b>Renovación exitosa:</b> Se agregaron %d días a <code>%s</code>", days, user), markup, tele.ModeHTML)
		return nil
	}
	return nil
}
