package bot

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/Depwisescript/BOT-TELEGRAM-VPN/internal/db"
	"github.com/Depwisescript/BOT-TELEGRAM-VPN/internal/sys"
	tele "gopkg.in/telebot.v3"
)

// Interceptar "Editar SSH" y renovar / cambiar pass
func handleMenuEditar(c tele.Context, b *tele.Bot) error {
	chatID := c.Chat().ID
	data, _ := db.Load()

	markup := &tele.ReplyMarkup{}
	var rows []tele.Row

	// Filtrar usuarios
	sa, _ := strconv.ParseInt(superAdmin, 10, 64)
	isSA := chatID == sa

	count := 0
	for user, ownerID := range data.SSHOwners {
		if isSA || ownerID == fmt.Sprintf("%d", chatID) {
			rows = append(rows, markup.Row(markup.Data("👤 "+user, "ed_user:"+user)))
			count++
		}
	}

	rows = append(rows, markup.Row(markup.Data("🔙 Volver", "back_main")))
	markup.Inline(rows...)

	if count == 0 {
		return c.Edit("❌ <b>No hay usuarios para editar.</b>", markup, tele.ModeHTML)
	}

	return c.Edit("✏️ <b>Editar Usuario SSH</b>\n\nSelecciona el usuario que deseas modificar:", markup, tele.ModeHTML)
}

func handleEditSelection(c tele.Context, b *tele.Bot) error {
	user := strings.TrimPrefix(c.Callback().Data, "ed_user:")
	return handleEditMenu(c, user, b, c.Message())
}

func handleEditMenu(c tele.Context, user string, b *tele.Bot, lastMsg *tele.Message) error {
	chatID := c.Chat().ID
	tempData[chatID] = make(map[string]string)
	tempData[chatID]["username"] = user

	markup := &tele.ReplyMarkup{}
	btnPass := markup.Data("🔑 Pass", "edit_pass")
	btnRenew := markup.Data("⏳ Días", "edit_renew")
	btnLimit := markup.Data("💻 Límite Conex.", "edit_limit")
	btnQuota := markup.Data("📊 Cuota GB", "edit_quota")
	btnCancel := markup.Data("❌ Cancelar", "cancelar_accion")

	markup.Inline(
		markup.Row(btnPass, btnRenew),
		markup.Row(btnLimit, btnQuota),
		markup.Row(btnCancel),
	)

	b.Edit(lastMsg, fmt.Sprintf("⚙️ <b>Opciones para:</b> <code>%s</code>", user), markup, tele.ModeHTML)
	return nil
}

func handleEditLimit(c tele.Context, b *tele.Bot) error {
	chatID := c.Chat().ID
	user := tempData[chatID]["username"]
	userSteps[chatID] = "awaiting_edit_limit_val"
	lastBotMsg[chatID] = c.Message()
	markup := &tele.ReplyMarkup{}
	markup.Inline(markup.Row(markup.Data("❌ Cancelar", "cancelar_accion")))
	return c.Edit(fmt.Sprintf("💻 <b>Límite de conexiones para:</b> <code>%s</code>\n\n✏️ <i>Escribe el nuevo límite (0 = ilimitado):</i>", user), markup, tele.ModeHTML)
}

func handleEditQuota(c tele.Context, b *tele.Bot) error {
	chatID := c.Chat().ID
	user := tempData[chatID]["username"]
	userSteps[chatID] = "awaiting_edit_quota_val"
	lastBotMsg[chatID] = c.Message()
	markup := &tele.ReplyMarkup{}
	markup.Inline(markup.Row(markup.Data("❌ Cancelar", "cancelar_accion")))
	return c.Edit(fmt.Sprintf("📊 <b>Cuota de datos para:</b> <code>%s</code>\n\n✏️ <i>Escribe la nueva cuota en GB (ej: 10.5, 0 = ilimitado):</i>", user), markup, tele.ModeHTML)
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
