package bot

import (
	"fmt"

	"github.com/Depwisescript/BOT-TELEGRAM-VPN/internal/db"
	"github.com/Depwisescript/BOT-TELEGRAM-VPN/internal/vpn"
	tele "gopkg.in/telebot.v3"
)

func handleInstallUDPCustom(c tele.Context, b *tele.Bot) error {
	data, _ := db.Load()
	if data.Zivpn {
		markup := &tele.ReplyMarkup{}
		markup.Inline(markup.Row(markup.Data("🔙 Volver", "menu_protocols")))
		return c.Edit("⚠️ <b>Conflicto de Protocolo</b>\n\nNo puedes instalar <b>UDP Custom</b> mientras <b>ZiVPN</b> esté activo. Por favor, desinstala ZiVPN primero.", markup, tele.ModeHTML)
	}

	c.Edit("⏳ <b>Instalando UDP Custom...</b>\n\nPor favor espera, configurando el servidor para HTTP Custom.", tele.ModeHTML)

	// Puerto de escucha UDP (2100 como en producción)
	port := "2100"

	err := vpn.InstallUDPCustom(port)
	markup := &tele.ReplyMarkup{}
	markup.Inline(markup.Row(markup.Data("🔙 Volver", "menu_protocols")))

	if err != nil {
		return c.Edit(fmt.Sprintf("❌ <b>Error al instalar UDP Custom:</b>\n%v", err), markup, tele.ModeHTML)
	}

	db.Update(func(data *db.ConfigData) error {
		data.UDPCustom = true
		return nil
	})

	return c.Edit("✅ <b>¡UDP Custom instalado con éxito!</b>\n\n🚀 <b>Puerto:</b> 2100 (NAT 1:65535)\n🔑 <b>Auth:</b> Usuarios SSH del sistema\n⚠️ <b>Exclusiones:</b> 323, 2200, 7100-7300, 10004, 10008\n\nYa puedes conectar desde la app <b>HTTP Custom</b> usando el método <b>UDP Custom</b>.", markup, tele.ModeHTML)
}

func handleUninstallUDPCustom(c tele.Context, b *tele.Bot) error {
	c.Edit("⏳ <b>Desinstalando UDP Custom...</b>", tele.ModeHTML)

	err := vpn.RemoveUDPCustom()
	markup := &tele.ReplyMarkup{}
	markup.Inline(markup.Row(markup.Data("🔙 Volver", "menu_protocols")))

	if err != nil {
		return c.Edit(fmt.Sprintf("❌ <b>Error al desinstalar:</b>\n%v", err), markup, tele.ModeHTML)
	}

	db.Update(func(data *db.ConfigData) error {
		data.UDPCustom = false
		return nil
	})

	return c.Edit("🗑️ <b>UDP Custom desinstalado correctamente.</b>", markup, tele.ModeHTML)
}
