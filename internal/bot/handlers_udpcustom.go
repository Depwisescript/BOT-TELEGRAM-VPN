package bot

import (
	"fmt"

	"github.com/Depwisescript/BOT-TELEGRAM-VPN/internal/db"
	"github.com/Depwisescript/BOT-TELEGRAM-VPN/internal/vpn"
	tele "gopkg.in/telebot.v3"
)

func handleInstallUDPCustom(c tele.Context, b *tele.Bot) error {
	c.Edit("⏳ <b>Instalando UDP Custom...</b>\n\nPor favor espera, configurando el servidor para HTTP Custom.", tele.ModeHTML)

	// Usamos puerto por defecto 36712 (común en UDP Custom) o automatizamos
	// Para seguir la lógica de "un clic" que pidió el usuario:
	port := "36712"

	err := vpn.InstallUDPCustom(port)
	if err != nil {
		return c.Send(fmt.Sprintf("❌ <b>Error al instalar UDP Custom:</b>\n%v", err), tele.ModeHTML)
	}

	db.Update(func(data *db.ConfigData) error {
		data.UDPCustom = true
		return nil
	})

	return c.Edit("✅ <b>¡UDP Custom instalado con éxito!</b>\n\n🚀 <b>Puerto:</b> 1-65535 (Redireccionado)\n🔑 <b>Auth:</b> Usuarios SSH del sistema\n\nYa puedes conectar desde la app <b>HTTP Custom</b> usando el método <b>UDP Custom</b>.", tele.ModeHTML)
}

func handleUninstallUDPCustom(c tele.Context, b *tele.Bot) error {
	c.Edit("⏳ <b>Desinstalando UDP Custom...</b>", tele.ModeHTML)

	err := vpn.RemoveUDPCustom()
	if err != nil {
		return c.Send(fmt.Sprintf("❌ <b>Error al desinstalar:</b>\n%v", err), tele.ModeHTML)
	}

	db.Update(func(data *db.ConfigData) error {
		data.UDPCustom = false
		return nil
	})

	return c.Edit("🗑️ <b>UDP Custom desinstalado correctamente.</b>", tele.ModeHTML)
}
