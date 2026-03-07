package bot

import (
	"fmt"
	"strconv"

	"github.com/Depwisescript/BOT-TELEGRAM-VPN/internal/db"
	"github.com/Depwisescript/BOT-TELEGRAM-VPN/internal/sys"
	"github.com/Depwisescript/BOT-TELEGRAM-VPN/internal/vpn"
	tele "gopkg.in/telebot.v3"
)

func handleProtocolDiag(c tele.Context, b *tele.Bot) error {
	report := vpn.GetSystemReport()
	markup := &tele.ReplyMarkup{}
	markup.Inline(markup.Row(markup.Data("🔙 Volver", "menu_protocols")))
	return SafeEditCtx(c, b, report, markup)
}

// Interceptar "Protocolos" para ver e Iniciar SlowDNS, Zivpn o BadVPN
func handleMenuProtocols(c tele.Context, b *tele.Bot) error {
	markup := &tele.ReplyMarkup{}

	btnSlowDNS := markup.Data("🐢 SlowDNS", "submenu_slowdns")
	btnZiVPN := markup.Data("🛰️ ZiVPN", "submenu_zivpn")
	btnBadVPN := markup.Data("🎮 BadVPN", "submenu_badvpn")
	btnUDPCustom := markup.Data("📡 UDP Custom", "submenu_udpcustom")
	btnProxy := markup.Data("🌐 ProxyDT", "submenu_proxydt")
	btnFalcon := markup.Data("🦅 Falcon", "submenu_falcon")
	btnSSL := markup.Data("📜 SSL Tunnel", "submenu_ssl")
	btnDropbear := markup.Data("🐻 Dropbear", "submenu_dropbear")
	btnScannerDeps := markup.Data("🛠️ Instalar Herramientas Escaner", "install_scanner_deps")
	btnCancel := markup.Data("🔙 Volver", "back_main")

	markup.Inline(
		markup.Row(btnSlowDNS, btnZiVPN),
		markup.Row(btnBadVPN, btnUDPCustom),
		markup.Row(btnProxy, btnFalcon),
		markup.Row(btnSSL, btnDropbear),
		markup.Row(markup.Data("🛡️ Diagnóstico de Red", "protocol_diag")),
		markup.Row(btnScannerDeps),
		markup.Row(btnCancel),
	)

	texto := "⚙️ <b>Gestor de Protocolos VPN</b>\n\n"
	texto += "<i>Selecciona un protocolo para ver las opciones de instalación o desinstalación.</i>"

	return SafeEditCtx(c, b, texto, markup)
}

// Mover handleMenuAdmins a handlers_admins.go

func handleMenuBroadcast(c tele.Context, b *tele.Bot) error {
	chatID := c.Chat().ID
	if !isAdmin(chatID) {
		return c.Send("⛔ Solo administradores pueden usar esta función.", tele.ModeHTML)
	}

	UserSteps[chatID] = "awaiting_vpn_broadcast"

	markup := &tele.ReplyMarkup{}
	markup.Inline(markup.Row(markup.Data("❌ Cancelar", "back_main")))

	return SafeEditCtx(c, b, "📢 <b>Mensaje Global (Broadcast)</b>\n\n✏️ <i>Escribe el mensaje que deseas enviar a todos los usuarios:</i>\n\nPuedes usar etiquetas HTML básicas como &lt;b&gt;, &lt;i&gt;, etc.", markup)
}

func handleInstallScannerDeps(c tele.Context, b *tele.Bot) error {
	chatID := c.Chat().ID
	if !isSuperAdminID(chatID) {
		return c.Send("⛔ Solo el SuperAdmin puede realizar esta instalación manual.", tele.ModeHTML)
	}

	SafeEditCtx(c, b, "⏳ <b>Instalando Herramientas de Escaneo...</b>\n\n<i>Esto instalará assetfinder y httpx. Por favor espera...</i>", nil)

	err := sys.EnsureScannerDeps()
	markup := &tele.ReplyMarkup{}
	markup.Inline(markup.Row(markup.Data("🔙 Volver", "menu_protocols")))

	if err != nil {
		return SafeEditCtx(c, b, fmt.Sprintf("❌ <b>Error en la instalación:</b>\n<pre>%v</pre>", err), markup)
	}

	return SafeEditCtx(c, b, "✅ <b>Herramientas de Escaneo Instaladas y Vinculadas Correctamente.</b>\n\nYa puedes usar el botón 🔍 <b>Escaner</b> del menú principal.", markup)
}

// Sub-Menús de Protocolos
func handleSubMenuSlowDNS(c tele.Context, b *tele.Bot) error {
	data, _ := db.Load()
	status := "❌ Desinstalado"
	if data.SlowDNS.NS != "" {
		status = "✅ Instalado"
	}

	markup := &tele.ReplyMarkup{}
	btnInst := markup.Data("📥 Instalar / Reconfigurar", "install_slowdns")
	btnUninst := markup.Data("🗑️ Desinstalar", "uninstall_slowdns")
	btnBack := markup.Data("🔙 Volver", "menu_protocols")

	markup.Inline(markup.Row(btnInst), markup.Row(btnUninst), markup.Row(btnBack))

	texto := fmt.Sprintf("🐢 <b>Gestión de SlowDNS</b>\n\n📊 <b>Estado:</b> %s\n🌍 <b>NS:</b> %s\n\n¿Qué deseas hacer?", status, data.SlowDNS.NS)
	return SafeEditCtx(c, b, texto, markup)
}

func handleSubMenuZiVPN(c tele.Context, b *tele.Bot) error {
	data, _ := db.Load()
	status := "❌ Desinstalado"
	if data.Zivpn {
		status = "✅ Instalado"
	}

	markup := &tele.ReplyMarkup{}
	btnInst := markup.Data("📥 Instalar", "install_zivpn")
	btnUninst := markup.Data("🗑️ Desinstalar", "uninstall_zivpn")
	btnBack := markup.Data("🔙 Volver", "menu_protocols")

	markup.Inline(markup.Row(btnInst), markup.Row(btnUninst), markup.Row(btnBack))

	texto := fmt.Sprintf("🛰️ <b>Gestión de ZiVPN</b>\n\n📊 <b>Estado:</b> %s\n\n¿Qué deseas hacer?", status)
	return SafeEditCtx(c, b, texto, markup)
}

func handleSubMenuUDPCustom(c tele.Context, b *tele.Bot) error {
	data, _ := db.Load()
	status := "❌ Desinstalado"
	if data.UDPCustom {
		status = "✅ Instalado"
	}

	markup := &tele.ReplyMarkup{}
	btnInst := markup.Data("📥 Instalar", "install_udpcustom")
	btnUninst := markup.Data("🗑️ Desinstalación Completa", "uninstall_udpcustom")
	btnBack := markup.Data("🔙 Volver", "menu_protocols")

	markup.Inline(markup.Row(btnInst), markup.Row(btnUninst), markup.Row(btnBack))

	texto := fmt.Sprintf("📡 <b>Gestión de UDP Custom (HTTP Custom)</b>\n\n📊 <b>Estado:</b> %s\n\nEste protocolo es el que utiliza específicamente la aplicación <b>HTTP Custom</b> en su opción 'UDP Custom'.\n\n¿Qué deseas hacer?", status)
	return SafeEditCtx(c, b, texto, markup)
}

func handleSubMenuBadVPN(c tele.Context, b *tele.Bot) error {
	data, _ := db.Load()
	status := "❌ Desinstalado"
	if data.BadVPN {
		status = "✅ Instalado"
	}

	markup := &tele.ReplyMarkup{}
	btnInst := markup.Data("📥 Instalar", "install_badvpn")
	btnUninst := markup.Data("🗑️ Desinstalar", "uninstall_badvpn")
	btnBack := markup.Data("🔙 Volver", "menu_protocols")

	markup.Inline(markup.Row(btnInst), markup.Row(btnUninst), markup.Row(btnBack))

	texto := fmt.Sprintf("🎮 <b>Gestión de BadVPN</b>\n\n📊 <b>Estado:</b> %s\n\n¿Qué deseas hacer?", status)
	return SafeEditCtx(c, b, texto, markup)
}

func handleSubMenuFalcon(c tele.Context, b *tele.Bot) error {
	markup := &tele.ReplyMarkup{}
	markup.Inline(
		markup.Row(markup.Data("📥 Instalar", "install_falcon")),
		markup.Row(markup.Data("🗑️ Desinstall", "uninstall_falcon")),
		markup.Row(markup.Data("🔙 Volver", "menu_protocols")),
	)
	return SafeEditCtx(c, b, "🦅 <b>Gestión de Falcon Proxy</b>\n\n¿Qué deseas hacer?", markup)
}

func handleSubMenuSSL(c tele.Context, b *tele.Bot) error {
	markup := &tele.ReplyMarkup{}
	markup.Inline(
		markup.Row(markup.Data("📥 Instalar", "install_ssl")),
		markup.Row(markup.Data("🗑️ Desinstalar", "uninstall_ssl")),
		markup.Row(markup.Data("🔙 Volver", "menu_protocols")),
	)
	return SafeEditCtx(c, b, "📜 <b>Gestión de SSL Tunnel</b>\n\n¿Qué deseas hacer?", markup)
}

func handleSubMenuDropbear(c tele.Context, b *tele.Bot) error {
	markup := &tele.ReplyMarkup{}
	markup.Inline(
		markup.Row(markup.Data("📥 Instalar", "install_dropbear")),
		markup.Row(markup.Data("🗑️ Desinstalar", "uninstall_dropbear")),
		markup.Row(markup.Data("🔙 Volver", "menu_protocols")),
	)
	return SafeEditCtx(c, b, "🐻 <b>Gestión de Dropbear</b>\n\n¿Qué deseas hacer?", markup)
}

func handleSubMenuProxyDT(c tele.Context, b *tele.Bot) error {
	markup := &tele.ReplyMarkup{}
	markup.Inline(
		markup.Row(markup.Data("📥 Instalar", "install_proxydt")),
		markup.Row(markup.Data("🗑️ Desinstalar", "uninstall_proxydt")),
		markup.Row(markup.Data("🔙 Volver", "menu_protocols")),
	)
	return SafeEditCtx(c, b, "🌐 <b>Gestión de ProxyDT</b>\n\n¿Qué deseas hacer?", markup)
}

// Handlers de Desinstalación
func handleUninstallProtocol(c tele.Context, b *tele.Bot, proto string) error {
	SafeEditCtx(c, b, fmt.Sprintf("⏳ <i>Desinstalando %s...</i>", proto), nil)
	var err error
	data, _ := db.Load()

	switch proto {
	case "SlowDNS":
		err = vpn.RemoveSlowDNS()
		data.SlowDNS = db.SlowDNSConfig{}
	case "ZiVPN":
		err = vpn.RemoveZiVPN()
		data.Zivpn = false
	case "BadVPN":
		err = vpn.RemoveBadVPN()
		data.BadVPN = false
	case "Falcon":
		err = vpn.RemoveFalcon()
		data.Falcon = ""
	case "SSL Tunnel":
		err = vpn.RemoveSSLTunnel()
		data.SSLTunnel = ""
	case "Dropbear":
		err = vpn.RemoveDropbear()
		data.Dropbear = ""
	case "ProxyDT":
		err = vpn.RemoveProxyDT()
		data.ProxyDT.Ports = make(map[string]string)
	}

	if err != nil {
		return c.Edit(fmt.Sprintf("❌ <b>Error al desinstalar %s:</b>\n%v", proto, err), tele.ModeHTML)
	}

	db.Save(data)
	markup := &tele.ReplyMarkup{}
	markup.Inline(markup.Row(markup.Data("🔙 Volver", "menu_protocols")))
	return c.Edit(fmt.Sprintf("✅ <b>%s desinstalado correctamente.</b>", proto), markup, tele.ModeHTML)
}

// Instaladores (Interacciones base)
func handleInstallSlowDNS(c tele.Context, b *tele.Bot, lastMsg *tele.Message) error {
	chatID := c.Chat().ID
	UserSteps[chatID] = "awaiting_vpn_slowdns_domain"
	TempData[chatID] = make(map[string]string)

	markup := &tele.ReplyMarkup{}
	markup.Inline(markup.Row(markup.Data("❌ Cancelar", "cancelar_accion")))

	b.Edit(lastMsg, "🐢 <b>Instalador de SlowDNS</b>\n\n🌍 <i>Escribe el subdominio (NS) que ya tengas apuntado a este servidor:</i>", markup, tele.ModeHTML)
	return nil
}

func handleInstallZivpn(c tele.Context, b *tele.Bot, lastMsg *tele.Message) error {
	data, _ := db.Load()
	if data.UDPCustom {
		markup := &tele.ReplyMarkup{}
		markup.Inline(markup.Row(markup.Data("🔙 Volver", "menu_protocols")))
		return c.Edit("⚠️ <b>Conflicto de Protocolo</b>\n\nNo puedes instalar <b>ZiVPN</b> mientras <b>UDP Custom</b> esté activo. Por favor, desinstala UDP Custom primero.", markup, tele.ModeHTML)
	}

	chatID := c.Chat().ID
	delete(UserSteps, chatID)

	b.Edit(lastMsg, "⏳ <i>Instalando ZiVPN (UDP Custom) en puerto automático 5667...</i>", tele.ModeHTML)

	err := vpn.InstallZivpn("5667")
	if err != nil {
		markup := &tele.ReplyMarkup{}
		markup.Inline(markup.Row(markup.Data("🔙 Volver", "menu_protocols")))
		b.Edit(lastMsg, fmt.Sprintf("❌ <b>Error al instalar ZiVPN:</b>\n<pre>%v</pre>", err), markup, tele.ModeHTML)
		return nil
	}

	res := "✅ <b>ZiVPN Instalado Correctamente</b>\n"
	res += "━━━━━━━━━━━━━━\n"
	res += "⚙️ <b>Puerto UDP:</b> <code>5667</code>\n"
	res += "━━━━━━━━━━━━━━\n"
	res += "<i>El servicio udp-custom ya está activo.</i>"

	data, _ = db.Load()
	data.Zivpn = true
	db.Save(data)

	markup := &tele.ReplyMarkup{}
	markup.Inline(markup.Row(markup.Data("🔙 Volver", "menu_protocols")))
	b.Edit(lastMsg, res, markup, tele.ModeHTML)
	return nil
}

func handleInstallBadVPN(c tele.Context, b *tele.Bot, lastMsg *tele.Message) error {
	chatID := c.Chat().ID
	delete(UserSteps, chatID)

	b.Edit(lastMsg, "⏳ <i>Instalando BadVPN (UDPGW) en puerto automático 7300...</i>", tele.ModeHTML)

	err := vpn.InstallBadVPN("7300")
	if err != nil {
		markup := &tele.ReplyMarkup{}
		markup.Inline(markup.Row(markup.Data("🔙 Volver", "menu_protocols")))
		b.Edit(lastMsg, fmt.Sprintf("❌ <b>Error al instalar BadVPN:</b>\n<pre>%v</pre>", err), markup, tele.ModeHTML)
		return nil
	}

	res := "✅ <b>BadVPN Instalado Correctamente</b>\n"
	res += "━━━━━━━━━━━━━━\n"
	res += "⚙️ <b>Puerto TCP:</b> <code>7300</code>\n"
	res += "━━━━━━━━━━━━━━\n"
	res += "<i>El demonio udpgw ya está escuchando.</i>"

	data, _ := db.Load()
	data.BadVPN = true
	db.Save(data)

	markup := &tele.ReplyMarkup{}
	markup.Inline(markup.Row(markup.Data("🔙 Volver", "menu_protocols")))
	b.Edit(lastMsg, res, markup, tele.ModeHTML)
	return nil
}

func handleInstallFalcon(c tele.Context, b *tele.Bot, lastMsg *tele.Message) error {
	chatID := c.Chat().ID
	UserSteps[chatID] = "awaiting_vpn_falcon_port"

	markup := &tele.ReplyMarkup{}
	markup.Inline(markup.Row(markup.Data("❌ Cancelar", "cancelar_accion")))

	b.Edit(lastMsg, "🦅 <b>Instalador de Falcon Proxy</b>\n\n⚙️ <i>Escribe el puerto de escucha (Ej: 8080):</i>", markup, tele.ModeHTML)
	return nil
}

func handleInstallSSL(c tele.Context, b *tele.Bot, lastMsg *tele.Message) error {
	chatID := c.Chat().ID
	UserSteps[chatID] = "awaiting_vpn_ssl_port"

	markup := &tele.ReplyMarkup{}
	markup.Inline(markup.Row(markup.Data("❌ Cancelar", "cancelar_accion")))

	b.Edit(lastMsg, "📜 <b>Instalador de SSL Tunnel (HAProxy)</b>\n\n⚙️ <i>Escribe el puerto de escucha (Ej: 443):</i>", markup, tele.ModeHTML)
	return nil
}

func handleInstallDropbear(c tele.Context, b *tele.Bot, lastMsg *tele.Message) error {
	chatID := c.Chat().ID
	UserSteps[chatID] = "awaiting_vpn_dropbear_port"

	markup := &tele.ReplyMarkup{}
	markup.Inline(markup.Row(markup.Data("❌ Cancelar", "cancelar_accion")))

	b.Edit(lastMsg, "🐻 <b>Instalador de Dropbear</b>\n\n⚙️ <i>Escribe el puerto de escucha (Ej: 90):</i>", markup, tele.ModeHTML)
	return nil
}

func handleInstallProxyDT(c tele.Context, b *tele.Bot, lastMsg *tele.Message) error {
	chatID := c.Chat().ID
	UserSteps[chatID] = "awaiting_vpn_proxydt_port"

	markup := &tele.ReplyMarkup{}
	markup.Inline(markup.Row(markup.Data("❌ Cancelar", "cancelar_accion")))

	b.Edit(lastMsg, "🌐 <b>Instalador de ProxyDT (Cracked)</b>\n\n⚙️ <i>Escribe el puerto de escucha (Ej: 80 o 8080):</i>", markup, tele.ModeHTML)
	return nil
}

// Interceptor secuencial para los módulos VPN
func processVPNSteps(step string, text string, chatID int64, c tele.Context, b *tele.Bot, lastMsg *tele.Message) error {
	markup := &tele.ReplyMarkup{}
	markup.Inline(markup.Row(markup.Data("🔙 Volver", "menu_protocols")))

	switch step {
	case "awaiting_vpn_broadcast":
		delete(UserSteps, chatID)
		delete(LastBotMsg, chatID)

		data, _ := db.Load()
		total := len(data.UserHistory)
		success := 0
		failed := 0

		// Avisar al admin que empezó
		b.Edit(lastMsg, fmt.Sprintf("⏳ <i>Emitiendo mensaje a %d usuarios...</i>", total), tele.ModeHTML)

		for _, id := range data.UserHistory {
			_, err := b.Send(tele.ChatID(id), "📢 <b>MENSAJE GLOBAL DE ADMINISTRACIÓN</b>\n\n"+text, tele.ModeHTML)
			if err == nil {
				success++
			} else {
				failed++
			}
		}

		res := "✅ <b>Emisión Finalizada</b>\n"
		res += "━━━━━━━━━━━━━━\n"
		res += fmt.Sprintf("📤 <b>Enviados:</b> <code>%d</code>\n", success)
		res += fmt.Sprintf("❌ <b>Fallidos:</b> <code>%d</code>\n", failed)
		res += "━━━━━━━━━━━━━━\n"

		markup := &tele.ReplyMarkup{}
		markup.Inline(markup.Row(markup.Data("🔙 Volver", "back_main")))
		b.Edit(lastMsg, res, markup, tele.ModeHTML)
		return nil

	case "awaiting_vpn_admin_id":
		id := text
		delete(UserSteps, chatID)
		delete(LastBotMsg, chatID)

		// Solo numérico
		if _, err := strconv.ParseInt(id, 10, 64); err != nil {
			b.Edit(lastMsg, "❌ <b>ID Inválido:</b> Debe ser un número.", markup, tele.ModeHTML)
			return nil
		}

		db.Update(func(data *db.ConfigData) error {
			data.Admins[id] = db.AdminInfo{Alias: "Admin"}
			return nil
		})

		b.Edit(lastMsg, fmt.Sprintf("✅ <b>ID %s</b> ahora es administrador.", id), markup, tele.ModeHTML)
		return nil

	case "awaiting_vpn_extrainfo":
		info := text
		delete(UserSteps, chatID)
		delete(LastBotMsg, chatID)

		db.Update(func(data *db.ConfigData) error {
			data.ExtraInfo = info
			return nil
		})

		b.Edit(lastMsg, "✅ <b>Información extra actualizada correctamente.</b>", markup, tele.ModeHTML)
		return nil

	case "awaiting_vpn_cloudflare":
		domain := text
		delete(UserSteps, chatID)
		delete(LastBotMsg, chatID)
		db.Update(func(data *db.ConfigData) error {
			data.CloudflareDomain = domain
			return nil
		})
		b.Edit(lastMsg, fmt.Sprintf("✅ <b>Dominio Cloudflare actualizado:</b> <code>%s</code>", domain), markup, tele.ModeHTML)
		return nil

	case "awaiting_vpn_cloudfront":
		domain := text
		delete(UserSteps, chatID)
		delete(LastBotMsg, chatID)
		db.Update(func(data *db.ConfigData) error {
			data.CloudfrontDomain = domain
			return nil
		})
		b.Edit(lastMsg, fmt.Sprintf("✅ <b>Dominio Cloudfront actualizado:</b> <code>%s</code>", domain), markup, tele.ModeHTML)
		return nil

	case "awaiting_vpn_ssh_banner":
		banner := text
		delete(UserSteps, chatID)
		delete(LastBotMsg, chatID)
		db.Update(func(data *db.ConfigData) error {
			data.SSHBanner = banner
			return nil
		})
		// Aplicar al sistema
		err := sys.SetSSHBanner(banner)
		if err != nil {
			b.Edit(lastMsg, fmt.Sprintf("⚠️ <b>Banner guardado en DB pero error al aplicar al sistema:</b>\n%v", err), markup, tele.ModeHTML)
		} else {
			b.Edit(lastMsg, "✅ <b>Banner SSH actualizado y aplicado al sistema.</b>", markup, tele.ModeHTML)
		}
		return nil

	case "awaiting_vpn_slowdns_domain":
		TempData[chatID]["domain"] = text
		UserSteps[chatID] = "awaiting_vpn_slowdns_port"

		markupCancel := &tele.ReplyMarkup{}
		markupCancel.Inline(markupCancel.Row(markupCancel.Data("❌ Cancelar", "cancelar_accion")))
		b.Edit(lastMsg, "⚙️ <i>¿A qué puerto local quieres redirigir SlowDNS? (Ej: 110, 22 o 443):</i>", markupCancel, tele.ModeHTML)
		return nil

	case "awaiting_vpn_slowdns_port":
		domain := TempData[chatID]["domain"]
		port := text

		delete(UserSteps, chatID)
		delete(LastBotMsg, chatID)

		b.Edit(lastMsg, "⏳ <i>Descargando binarios e instalando SlowDNS... (Tomará unos segundos)</i>", tele.ModeHTML)

		pubKey, err := vpn.InstallSlowDNS(domain, port)
		if err != nil {
			b.Edit(lastMsg, fmt.Sprintf("❌ <b>Error al instalar SlowDNS:</b>\n<pre>%v</pre>", err), markup, tele.ModeHTML)
			return nil
		}

		res := "✅ <b>SlowDNS Instalado Correctamente</b>\n"
		res += "━━━━━━━━━━━━━━\n"
		res += fmt.Sprintf("🌍 <b>NS:</b> <code>%s</code>\n", domain)
		res += fmt.Sprintf("🔑 <b>Pub Key:</b> <code>%s</code>\n", pubKey)
		res += "━━━━━━━━━━━━━━\n"
		res += "<i>El servicio ya está activo en Systemd.</i>"

		// Guardar estado
		data, _ := db.Load()
		data.SlowDNS.NS = domain
		data.SlowDNS.Port = port
		data.SlowDNS.Key = pubKey
		db.Save(data)

		b.Edit(lastMsg, res, markup, tele.ModeHTML)
		return nil

	case "awaiting_vpn_zivpn_port":
		port := text
		if _, err := strconv.Atoi(port); err != nil {
			b.Edit(lastMsg, "❌ <b>Puerto inválido.</b> Por favor, ingresa solo números (Ej: 7300).", markup, tele.ModeHTML)
			return nil
		}
		delete(UserSteps, chatID)
		delete(LastBotMsg, chatID)

		b.Edit(lastMsg, "⏳ <i>Instalando ZiVPN (UDP Custom)...</i>", tele.ModeHTML)

		err := vpn.InstallZivpn(port)
		if err != nil {
			b.Edit(lastMsg, fmt.Sprintf("❌ <b>Error al instalar ZiVPN:</b>\n<pre>%v</pre>", err), markup, tele.ModeHTML)
			return nil
		}

		res := "✅ <b>ZiVPN Instalado Correctamente</b>\n"
		res += "━━━━━━━━━━━━━━\n"
		res += fmt.Sprintf("⚙️ <b>Puerto UDP:</b> <code>%s</code>\n", port)
		res += "━━━━━━━━━━━━━━\n"
		res += "<i>El servicio udp-custom ya está activo.</i>"

		// Guardar estado
		data, _ := db.Load()
		data.Zivpn = true
		db.Save(data)

		b.Edit(lastMsg, res, markup, tele.ModeHTML)
		return nil

	case "awaiting_vpn_badvpn_port":
		port := text
		if _, err := strconv.Atoi(port); err != nil {
			b.Edit(lastMsg, "❌ <b>Puerto inválido.</b> Por favor, ingresa solo números (Ej: 7200).", markup, tele.ModeHTML)
			return nil
		}
		delete(UserSteps, chatID)
		delete(LastBotMsg, chatID)

		b.Edit(lastMsg, "⏳ <i>Descargando e instalando BadVPN...</i>", tele.ModeHTML)

		err := vpn.InstallBadVPN(port)
		if err != nil {
			b.Edit(lastMsg, fmt.Sprintf("❌ <b>Error al instalar BadVPN:</b>\n<pre>%v</pre>", err), markup, tele.ModeHTML)
			return nil
		}

		res := "✅ <b>BadVPN Instalado Correctamente</b>\n"
		res += "━━━━━━━━━━━━━━\n"
		res += fmt.Sprintf("⚙️ <b>Puerto TCP:</b> <code>%s</code>\n", port)
		res += "━━━━━━━━━━━━━━\n"
		res += "<i>El demonio udpgw ya está escuchando.</i>"

		// Guardar estado
		data, _ := db.Load()
		data.BadVPN = true
		db.Save(data)

		b.Edit(lastMsg, res, markup, tele.ModeHTML)
		return nil

	case "awaiting_vpn_falcon_port":
		port := text
		delete(UserSteps, chatID)
		delete(LastBotMsg, chatID)

		b.Edit(lastMsg, "⏳ <i>Instalando Falcon Proxy...</i>", tele.ModeHTML)
		ver, err := vpn.InstallFalcon(port)
		if err != nil {
			b.Edit(lastMsg, fmt.Sprintf("❌ <b>Error al instalar Falcon:</b>\n<pre>%v</pre>", err), markup, tele.ModeHTML)
			return nil
		}

		res := "✅ <b>Falcon Proxy Instalado</b>\n"
		res += "━━━━━━━━━━━━━━\n"
		res += fmt.Sprintf("🦅 <b>Version:</b> <code>%s</code>\n", ver)
		res += fmt.Sprintf("⚙️ <b>Puerto:</b> <code>%s</code>\n", port)
		res += "━━━━━━━━━━━━━━\n"

		// Guardar estado
		data, _ := db.Load()
		data.Falcon = port
		db.Save(data)

		b.Edit(lastMsg, res, markup, tele.ModeHTML)
		return nil

	case "awaiting_vpn_ssl_port":
		port := text
		delete(UserSteps, chatID)
		delete(LastBotMsg, chatID)

		b.Edit(lastMsg, "⏳ <i>Configurando SSL Tunnel (HAProxy)...</i>", tele.ModeHTML)
		err := vpn.InstallSSLTunnel(port)
		if err != nil {
			b.Edit(lastMsg, fmt.Sprintf("❌ <b>Error al instalar SSL Tunnel:</b>\n<pre>%v</pre>", err), markup, tele.ModeHTML)
			return nil
		}

		res := "✅ <b>SSL Tunnel Instalado</b>\n"
		res += "━━━━━━━━━━━━━━\n"
		res += fmt.Sprintf("📜 <b>Puerto SSL:</b> <code>%s</code>\n", port)
		res += "━━━━━━━━━━━━━━\n"

		// Guardar estado
		data, _ := db.Load()
		data.SSLTunnel = port
		db.Save(data)

		b.Edit(lastMsg, res, markup, tele.ModeHTML)
		return nil

	case "awaiting_vpn_dropbear_port":
		port := text
		delete(UserSteps, chatID)
		delete(LastBotMsg, chatID)

		b.Edit(lastMsg, "⏳ <i>Configurando Dropbear...</i>", tele.ModeHTML)
		err := vpn.InstallDropbear(port)
		if err != nil {
			b.Edit(lastMsg, fmt.Sprintf("❌ <b>Error al instalar Dropbear:</b>\n<pre>%v</pre>", err), markup, tele.ModeHTML)
			return nil
		}

		res := "✅ <b>Dropbear Instalado</b>\n"
		res += "━━━━━━━━━━━━━━\n"
		res += fmt.Sprintf("🐻 <b>Puerto:</b> <code>%s</code>\n", port)
		res += "━━━━━━━━━━━━━━\n"

		// Guardar estado
		data, _ := db.Load()
		data.Dropbear = port
		db.Save(data)

		b.Edit(lastMsg, res, markup, tele.ModeHTML)
		return nil

	case "awaiting_vpn_proxydt_port":
		port := text
		if _, err := strconv.Atoi(port); err != nil {
			b.Edit(lastMsg, "❌ <b>Puerto inválido.</b> Por favor, ingresa solo números (Ej: 8080).", markup, tele.ModeHTML)
			return nil
		}
		delete(UserSteps, chatID)
		delete(LastBotMsg, chatID)

		b.Edit(lastMsg, "⏳ <i>Instalando y configurando ProxyDT...</i>", tele.ModeHTML)

		if err := vpn.InstallProxyDT(); err != nil {
			b.Edit(lastMsg, fmt.Sprintf("❌ <b>Error al instalar binario ProxyDT:</b>\n<pre>%v</pre>", err), markup, tele.ModeHTML)
			return nil
		}

		err := vpn.OpenProxyDTPort(port)
		if err != nil {
			b.Edit(lastMsg, fmt.Sprintf("❌ <b>Error al abrir puerto ProxyDT:</b>\n<pre>%v</pre>", err), markup, tele.ModeHTML)
			return nil
		}

		res := "✅ <b>ProxyDT Online</b>\n"
		res += "━━━━━━━━━━━━━━\n"
		res += fmt.Sprintf("🌐 <b>Puerto:</b> <code>%s</code>\n", port)
		res += "━━━━━━━━━━━━━━\n"

		// Guardar estado
		data, _ := db.Load()
		if data.ProxyDT.Ports == nil {
			data.ProxyDT.Ports = make(map[string]string)
		}
		data.ProxyDT.Ports[port] = "Online"
		db.Save(data)

		b.Edit(lastMsg, res, markup, tele.ModeHTML)
		return nil
	}
	return nil
}
