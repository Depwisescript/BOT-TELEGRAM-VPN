package bot

import (
	"fmt"
	"os"
	"strings"

	"github.com/Depwisescript/BOT-TELEGRAM-VPN/internal/sys"
	tele "gopkg.in/telebot.v3"
)

func handleMenuScanner(c tele.Context, b *tele.Bot) error {
	markup := &tele.ReplyMarkup{}
	btnStart := markup.Data("🔍 Iniciar Escaneo", "start_scanner_prompt")
	btnBack := markup.Data("🔙 Volver", "back_main")
	markup.Inline(markup.Row(btnStart), markup.Row(btnBack))

	texto := "🔍 <b>Depwise Scanner 🌐</b>\n\n"
	texto += "Esta herramienta permite enumerar subdominios y detectar tecnologías (Cloudflare, Cloudfront, etc.) de forma automática.\n\n"
	texto += "🚀 <b>Funciones:</b>\n"
	texto += "- Enumeración pasiva (Assetfinder)\n"
	texto += "- Detección de Tech/CDN (httpx)\n"
	texto += "- Reporte en tiempo real\n\n"
	texto += "<i>Disponible para todos los usuarios.</i>"

	return c.Edit(texto, markup, tele.ModeHTML)
}

func handleStartScanPrompt(c tele.Context, b *tele.Bot) error {
	chatID := c.Chat().ID
	userSteps[chatID] = "awaiting_scanner_domain"
	lastBotMsg[chatID] = c.Message()

	markup := &tele.ReplyMarkup{}
	markup.Inline(markup.Row(markup.Data("❌ Cancelar", "menu_scanner")))

	return c.Edit("🌐 <b>Escaner:</b>\n\n✏️ <i>Escribe el dominio que deseas escanear (ej: google.com):</i>", markup, tele.ModeHTML)
}

func processScannerSteps(step string, text string, chatID int64, c tele.Context, b *tele.Bot, lastMsg *tele.Message) error {
	if step != "awaiting_scanner_domain" {
		return nil
	}

	domain := strings.TrimSpace(text)
	delete(userSteps, chatID)
	delete(lastBotMsg, chatID)

	b.Edit(lastMsg, fmt.Sprintf("⏳ <b>Escaneando:</b> <code>%s</code>\n\n<i>Esto puede tardar unos segundos, por favor espera...</i>", domain), tele.ModeHTML)

	go func() {
		// Asegurar dependencias
		_ = sys.EnsureScannerDeps()

		result, err := sys.RunScanner(domain)
		markup := &tele.ReplyMarkup{}
		markup.Inline(markup.Row(markup.Data("🔙 Volver al Escaner", "menu_scanner")))

		if err != nil {
			b.Edit(lastMsg, fmt.Sprintf("❌ <b>Error en el Escaneo:</b>\n<pre>%v</pre>", err), markup, tele.ModeHTML)
			return
		}

		// Limitar el resultado si es muy largo para Telegram (4096 chars)
		if len(result) > 3500 {
			// Enviar primero un adelanto
			preview := result[:3000] + "\n\n... (Reporte completo en el archivo adjunto)"
			header := fmt.Sprintf("✅ <b>Resultados (Previsualización):</b> <code>%s</code>\n", domain)
			header += "━━━━━━━━━━━━━━\n"
			b.Edit(lastMsg, header+preview, markup, tele.ModeHTML)

			// Crear archivo temporal para el reporte completo
			tmpFile := fmt.Sprintf("/tmp/scan_%s.txt", domain)
			_ = os.WriteFile(tmpFile, []byte(result), 0644)
			defer os.Remove(tmpFile)

			doc := &tele.Document{
				File:     tele.FromDisk(tmpFile),
				FileName: fmt.Sprintf("reporte_escaneo_%s.txt", domain),
				Caption:  fmt.Sprintf("📄 Reporte completo de escaneo para %s", domain),
			}
			b.Send(c.Sender(), doc)
			return
		}

		header := fmt.Sprintf("✅ <b>Resultados para:</b> <code>%s</code>\n", domain)
		header += "━━━━━━━━━━━━━━\n"
		b.Edit(lastMsg, header+result, markup, tele.ModeHTML)
	}()

	return nil
}
