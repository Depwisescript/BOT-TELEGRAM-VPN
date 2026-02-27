package bot

import (
	"fmt"
	"strconv"

	"github.com/Depwisescript/BOT-TELEGRAM-VPN/internal/db"
	"github.com/Depwisescript/BOT-TELEGRAM-VPN/internal/vpn"
	tele "gopkg.in/telebot.v3"
)

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
	btnCancel := markup.Data("🔙 Volver", "back_main")

	markup.Inline(
		markup.Row(btnSlowDNS, btnZiVPN),
		markup.Row(btnBadVPN, btnUDPCustom),
		markup.Row(btnProxy, btnFalcon),
		markup.Row(btnSSL, btnDropbear),
		markup.Row(btnCancel),
	)

	texto := "⚙️ <b>Gestor de Protocolos VPN</b>\n\n"
	texto += "<i>Selecciona un protocolo para ver las opciones de instalación o desinstalación.</i>"

	return c.Edit(texto, markup, tele.ModeHTML)
}

// Interceptar "Ajustes Pro" para activar/desactivar el modo publico del bot
func handleMenuAdmins(c tele.Context, b *tele.Bot) error {
	markup := &tele.ReplyMarkup{}

	btnMode := markup.Data("🔓 Cambiar Acceso (Pub/Priv)", "toggle_public_access")
	btnList := markup.Data("📋 Ver Admins", "list_admins")
	btnCancel := markup.Data("🔙 Volver", "back_main")

	markup.Inline(
		markup.Row(btnMode),
		markup.Row(btnList),
		markup.Row(btnCancel),
	)

	return c.Edit("⚙️ <b>Ajustes de Administrador</b>\n\nConfigura la privacidad y opciones críticas del bot.", markup, tele.ModeHTML)
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
	return c.Edit(texto, markup, tele.ModeHTML)
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
	return c.Edit(texto, markup, tele.ModeHTML)
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
	return c.Edit(texto, markup, tele.ModeHTML)
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
	return c.Edit(texto, markup, tele.ModeHTML)
}

func handleSubMenuFalcon(c tele.Context, b *tele.Bot) error {
	// ... similar logic
	markup := &tele.ReplyMarkup{}
	markup.Inline(
		markup.Row(markup.Data("📥 Instalar", "install_falcon")),
		markup.Row(markup.Data("🗑️ Desinstalar", "uninstall_falcon")),
		markup.Row(markup.Data("🔙 Volver", "menu_protocols")),
	)
	return c.Edit("🦅 <b>Gestión de Falcon Proxy</b>\n\n¿Qué deseas hacer?", markup, tele.ModeHTML)
}

func handleSubMenuSSL(c tele.Context, b *tele.Bot) error {
	markup := &tele.ReplyMarkup{}
	markup.Inline(
		markup.Row(markup.Data("📥 Instalar", "install_ssl")),
		markup.Row(markup.Data("🗑️ Desinstalar", "uninstall_ssl")),
		markup.Row(markup.Data("🔙 Volver", "menu_protocols")),
	)
	return c.Edit("📜 <b>Gestión de SSL Tunnel</b>\n\n¿Qué deseas hacer?", markup, tele.ModeHTML)
}

func handleSubMenuDropbear(c tele.Context, b *tele.Bot) error {
	markup := &tele.ReplyMarkup{}
	markup.Inline(
		markup.Row(markup.Data("📥 Instalar", "install_dropbear")),
		markup.Row(markup.Data("🗑️ Desinstalar", "uninstall_dropbear")),
		markup.Row(markup.Data("🔙 Volver", "menu_protocols")),
	)
	return c.Edit("🐻 <b>Gestión de Dropbear</b>\n\n¿Qué deseas hacer?", markup, tele.ModeHTML)
}

func handleSubMenuProxyDT(c tele.Context, b *tele.Bot) error {
	markup := &tele.ReplyMarkup{}
	markup.Inline(
		markup.Row(markup.Data("📥 Instalar", "install_proxydt")),
		markup.Row(markup.Data("🗑️ Desinstalar", "uninstall_proxydt")),
		markup.Row(markup.Data("🔙 Volver", "menu_protocols")),
	)
	return c.Edit("🌐 <b>Gestión de ProxyDT</b>\n\n¿Qué deseas hacer?", markup, tele.ModeHTML)
}

// Handlers de Desinstalación
func handleUninstallProtocol(c tele.Context, b *tele.Bot, proto string) error {
	c.Edit(fmt.Sprintf("⏳ <i>Desinstalando %s...</i>", proto), tele.ModeHTML)
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
	userSteps[chatID] = "awaiting_vpn_slowdns_domain"
	tempData[chatID] = make(map[string]string)
	lastBotMsg[chatID] = lastMsg

	markup := &tele.ReplyMarkup{}
	markup.Inline(markup.Row(markup.Data("❌ Cancelar", "cancelar_accion")))

	b.Edit(lastMsg, "🐢 <b>Instalador de SlowDNS</b>\n\n🌍 <i>Escribe el subdominio (NS) que ya tengas apuntado a este servidor:</i>", markup, tele.ModeHTML)
	return nil
}

func handleInstallZivpn(c tele.Context, b *tele.Bot, lastMsg *tele.Message) error {
	chatID := c.Chat().ID
	delete(userSteps, chatID)

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

	data, _ := db.Load()
	data.Zivpn = true
	db.Save(data)

	markup := &tele.ReplyMarkup{}
	markup.Inline(markup.Row(markup.Data("🔙 Volver", "menu_protocols")))
	b.Edit(lastMsg, res, markup, tele.ModeHTML)
	return nil
}

func handleInstallBadVPN(c tele.Context, b *tele.Bot, lastMsg *tele.Message) error {
	chatID := c.Chat().ID
	delete(userSteps, chatID)

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
	userSteps[chatID] = "awaiting_vpn_falcon_port"
	lastBotMsg[chatID] = lastMsg

	markup := &tele.ReplyMarkup{}
	markup.Inline(markup.Row(markup.Data("❌ Cancelar", "cancelar_accion")))

	b.Edit(lastMsg, "🦅 <b>Instalador de Falcon Proxy</b>\n\n⚙️ <i>Escribe el puerto de escucha (Ej: 8080):</i>", markup, tele.ModeHTML)
	return nil
}

func handleInstallSSL(c tele.Context, b *tele.Bot, lastMsg *tele.Message) error {
	chatID := c.Chat().ID
	userSteps[chatID] = "awaiting_vpn_ssl_port"
	lastBotMsg[chatID] = lastMsg

	markup := &tele.ReplyMarkup{}
	markup.Inline(markup.Row(markup.Data("❌ Cancelar", "cancelar_accion")))

	b.Edit(lastMsg, "📜 <b>Instalador de SSL Tunnel (HAProxy)</b>\n\n⚙️ <i>Escribe el puerto de escucha (Ej: 443):</i>", markup, tele.ModeHTML)
	return nil
}

func handleInstallDropbear(c tele.Context, b *tele.Bot, lastMsg *tele.Message) error {
	chatID := c.Chat().ID
	userSteps[chatID] = "awaiting_vpn_dropbear_port"
	lastBotMsg[chatID] = lastMsg

	markup := &tele.ReplyMarkup{}
	markup.Inline(markup.Row(markup.Data("❌ Cancelar", "cancelar_accion")))

	b.Edit(lastMsg, "🐻 <b>Instalador de Dropbear</b>\n\n⚙️ <i>Escribe el puerto de escucha (Ej: 90):</i>", markup, tele.ModeHTML)
	return nil
}

func handleInstallProxyDT(c tele.Context, b *tele.Bot, lastMsg *tele.Message) error {
	chatID := c.Chat().ID
	userSteps[chatID] = "awaiting_vpn_proxydt_port"
	lastBotMsg[chatID] = lastMsg

	markup := &tele.ReplyMarkup{}
	markup.Inline(markup.Row(markup.Data("❌ Cancelar", "cancelar_accion")))

	b.Edit(lastMsg, "🌐 <b>Instalador de ProxyDT (Cracked)</b>\n\n⚙️ <i>Escribe el puerto de escucha (Ej: 80 o 8080):</i>", markup, tele.ModeHTML)
	return nil
}

// Interceptor secuencial para los módulos VPN
func processVPNSteps(step string, text string, chatID int64, c tele.Context, b *tele.Bot, lastMsg *tele.Message) error {
	markup := &tele.ReplyMarkup{}
	markup.Inline(markup.Row(markup.Data("🔙 Volver al Menú", "back_main")))

	switch step {
	case "awaiting_vpn_slowdns_domain":
		tempData[chatID]["domain"] = text
		userSteps[chatID] = "awaiting_vpn_slowdns_port"

		markupCancel := &tele.ReplyMarkup{}
		markupCancel.Inline(markupCancel.Row(markupCancel.Data("❌ Cancelar", "cancelar_accion")))
		b.Edit(lastMsg, "⚙️ <i>¿A qué puerto local quieres redirigir SlowDNS? (Ej: 110, 22 o 443):</i>", markupCancel, tele.ModeHTML)
		return nil

	case "awaiting_vpn_slowdns_port":
		domain := tempData[chatID]["domain"]
		port := text

		delete(userSteps, chatID)
		delete(lastBotMsg, chatID)

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
		delete(userSteps, chatID)
		delete(lastBotMsg, chatID)

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
		delete(userSteps, chatID)
		delete(lastBotMsg, chatID)

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
		delete(userSteps, chatID)
		delete(lastBotMsg, chatID)

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
		delete(userSteps, chatID)
		delete(lastBotMsg, chatID)

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
		delete(userSteps, chatID)
		delete(lastBotMsg, chatID)

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
		delete(userSteps, chatID)
		delete(lastBotMsg, chatID)

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
