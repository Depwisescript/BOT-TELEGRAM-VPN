package bot

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/Depwisescript/BOT-TELEGRAM-VPN/internal/db"
	"github.com/Depwisescript/BOT-TELEGRAM-VPN/internal/sys"
	tele "gopkg.in/telebot.v3"
)

var (
	botToken   = os.Getenv("BOT_TOKEN")
	superAdmin = os.Getenv("SUPER_ADMIN")
)

// StartBot inicializa el bot de Telegram y registra los handlers
func StartBot() {
	if botToken == "" || superAdmin == "" {
		log.Fatal("Variables BOT_TOKEN y SUPER_ADMIN son requeridas")
	}

	pref := tele.Settings{
		Token:  botToken,
		Poller: &tele.LongPoller{Timeout: 10 * time.Second},
	}

	b, err := tele.NewBot(pref)
	if err != nil {
		log.Fatal(err)
		return
	}

	// Handlers
	b.Handle("/start", func(c tele.Context) error {
		return handleStart(c, b)
	})

	b.Handle("/menu", func(c tele.Context) error {
		return handleStart(c, b)
	})

	// Text Interceptor para conversacion
	b.Handle(tele.OnText, func(c tele.Context) error {
		return handleTextInputs(c, b)
	})

	// Opciones del Menú Principal
	b.Handle(&tele.Btn{Unique: "menu_crear"}, func(c tele.Context) error {
		return c.Edit(menuCrearText(), menuCrearMarkup())
	})
	b.Handle(&tele.Btn{Unique: "menu_info"}, func(c tele.Context) error {
		return handleInfo(c, b)
	})
	b.Handle(&tele.Btn{Unique: "my_stats"}, func(c tele.Context) error {
		return handleMyStats(c, b)
	})
	b.Handle(&tele.Btn{Unique: "menu_broadcast"}, func(c tele.Context) error {
		return handleMenuBroadcast(c, b)
	})
	b.Handle(&tele.Btn{Unique: "menu_eliminar"}, func(c tele.Context) error {
		return handleMenuEliminar(c, b)
	})

	// Opciones de Configuración Avanzada
	b.Handle(&tele.Btn{Unique: "menu_editar"}, func(c tele.Context) error {
		return handleMenuEditar(c, b)
	})
	b.Handle(&tele.Btn{Unique: "edit_pass"}, func(c tele.Context) error {
		return handleEditPass(c, b)
	})
	b.Handle(&tele.Btn{Unique: "edit_renew"}, func(c tele.Context) error {
		return handleEditRenew(c, b)
	})

	b.Handle(&tele.Btn{Unique: "menu_protocols"}, func(c tele.Context) error {
		return handleMenuProtocols(c, b)
	})
	b.Handle(&tele.Btn{Unique: "menu_admins"}, func(c tele.Context) error {
		return handleMenuAdmins(c, b)
	})
	b.Handle(&tele.Btn{Unique: "menu_online"}, func(c tele.Context) error {
		return handleMenuOnline(c, b)
	})

	// VPNs
	b.Handle(&tele.Btn{Unique: "install_slowdns"}, func(c tele.Context) error {
		return handleInstallSlowDNS(c, b, c.Message())
	})
	b.Handle(&tele.Btn{Unique: "install_zivpn"}, func(c tele.Context) error {
		return handleInstallZivpn(c, b, c.Message())
	})
	b.Handle(&tele.Btn{Unique: "install_badvpn"}, func(c tele.Context) error {
		return handleInstallBadVPN(c, b, c.Message())
	})
	b.Handle(&tele.Btn{Unique: "install_falcon"}, func(c tele.Context) error {
		return handleInstallFalcon(c, b, c.Message())
	})
	b.Handle(&tele.Btn{Unique: "install_ssl"}, func(c tele.Context) error {
		return handleInstallSSL(c, b, c.Message())
	})
	b.Handle(&tele.Btn{Unique: "install_dropbear"}, func(c tele.Context) error {
		return handleInstallDropbear(c, b, c.Message())
	})
	b.Handle(&tele.Btn{Unique: "install_proxydt"}, func(c tele.Context) error {
		return handleInstallProxyDT(c, b, c.Message())
	})
	b.Handle(&tele.Btn{Unique: "install_udpcustom"}, func(c tele.Context) error {
		return handleInstallUDPCustom(c, b)
	})

	// Sub-Menús de Protocolos
	b.Handle(&tele.Btn{Unique: "submenu_slowdns"}, func(c tele.Context) error { return handleSubMenuSlowDNS(c, b) })
	b.Handle(&tele.Btn{Unique: "submenu_zivpn"}, func(c tele.Context) error { return handleSubMenuZiVPN(c, b) })
	b.Handle(&tele.Btn{Unique: "submenu_badvpn"}, func(c tele.Context) error { return handleSubMenuBadVPN(c, b) })
	b.Handle(&tele.Btn{Unique: "submenu_falcon"}, func(c tele.Context) error { return handleSubMenuFalcon(c, b) })
	b.Handle(&tele.Btn{Unique: "submenu_ssl"}, func(c tele.Context) error { return handleSubMenuSSL(c, b) })
	b.Handle(&tele.Btn{Unique: "submenu_dropbear"}, func(c tele.Context) error { return handleSubMenuDropbear(c, b) })
	b.Handle(&tele.Btn{Unique: "submenu_proxydt"}, func(c tele.Context) error { return handleSubMenuProxyDT(c, b) })
	b.Handle(&tele.Btn{Unique: "submenu_udpcustom"}, func(c tele.Context) error { return handleSubMenuUDPCustom(c, b) })
	b.Handle(&tele.Btn{Unique: "protocol_diag"}, func(c tele.Context) error { return handleProtocolDiag(c, b) })
	b.Handle(&tele.Btn{Unique: "menu_protocols"}, func(c tele.Context) error { return handleMenuProtocols(c, b) })

	// Desinstaladores
	b.Handle(&tele.Btn{Unique: "uninstall_slowdns"}, func(c tele.Context) error { return handleUninstallProtocol(c, b, "SlowDNS") })
	b.Handle(&tele.Btn{Unique: "uninstall_zivpn"}, func(c tele.Context) error { return handleUninstallProtocol(c, b, "ZiVPN") })
	b.Handle(&tele.Btn{Unique: "uninstall_badvpn"}, func(c tele.Context) error { return handleUninstallProtocol(c, b, "BadVPN") })
	b.Handle(&tele.Btn{Unique: "uninstall_falcon"}, func(c tele.Context) error { return handleUninstallProtocol(c, b, "Falcon") })
	b.Handle(&tele.Btn{Unique: "uninstall_ssl"}, func(c tele.Context) error { return handleUninstallProtocol(c, b, "SSL Tunnel") })
	b.Handle(&tele.Btn{Unique: "uninstall_dropbear"}, func(c tele.Context) error { return handleUninstallProtocol(c, b, "Dropbear") })
	b.Handle(&tele.Btn{Unique: "uninstall_proxydt"}, func(c tele.Context) error { return handleUninstallProtocol(c, b, "ProxyDT") })
	b.Handle(&tele.Btn{Unique: "uninstall_udpcustom"}, func(c tele.Context) error { return handleUninstallUDPCustom(c, b) })

	// Callbacks Dinámicos (One-Tap Selection)
	b.Handle("\fed_user:", func(c tele.Context) error { return handleEditSelection(c, b) })
	b.Handle("\fdel_confirm:", func(c tele.Context) error { return handleDeleteSelection(c, b) })
	b.Handle("\fdel_adm_exec:", func(c tele.Context) error { return handleDelAdminExec(c, b) })

	// Ajustes Pro
	b.Handle(&tele.Btn{Unique: "toggle_public_access"}, func(c tele.Context) error { return handleTogglePublicAccess(c, b) })
	b.Handle(&tele.Btn{Unique: "list_admins"}, func(c tele.Context) error { return handleListAdmins(c, b) })
	b.Handle(&tele.Btn{Unique: "add_admin"}, func(c tele.Context) error { return handleAddAdminPrompt(c, b) })
	b.Handle(&tele.Btn{Unique: "del_admin_menu"}, func(c tele.Context) error { return handleDelAdminMenu(c, b) })
	b.Handle(&tele.Btn{Unique: "edit_extrainfo"}, func(c tele.Context) error { return handleEditExtraInfoPrompt(c, b) })
	b.Handle(&tele.Btn{Unique: "edit_cloudflare"}, func(c tele.Context) error { return handleEditCloudflarePrompt(c, b) })
	b.Handle(&tele.Btn{Unique: "edit_cloudfront"}, func(c tele.Context) error { return handleEditCloudfrontPrompt(c, b) })
	b.Handle(&tele.Btn{Unique: "edit_banner"}, func(c tele.Context) error { return handleEditBannerPrompt(c, b) })
	b.Handle(&tele.Btn{Unique: "reset_history"}, func(c tele.Context) error { return handleResetHistoryConfirm(c, b) })
	b.Handle(&tele.Btn{Unique: "reset_history_exec"}, func(c tele.Context) error { return handleResetHistoryExec(c, b) })
	b.Handle(&tele.Btn{Unique: "reboot_vps_confirm"}, func(c tele.Context) error { return handleServerRebootConfirm(c, b) })
	b.Handle(&tele.Btn{Unique: "reboot_vps_exec"}, func(c tele.Context) error { return handleServerRebootExec(c, b) })
	b.Handle(&tele.Btn{Unique: "clean_db_confirm"}, func(c tele.Context) error { return handleCleanDBConfirm(c, b) })
	b.Handle(&tele.Btn{Unique: "clean_db_exec"}, func(c tele.Context) error { return handleCleanDBExec(c, b) })
	b.Handle(&tele.Btn{Unique: "deep_cleanup"}, func(c tele.Context) error { return handleDeepCleanup(c, b) })

	// Generar Usuario SSH / ZIVPN Handler
	b.Handle(&tele.Btn{Unique: "crear_ssh"}, func(c tele.Context) error {
		return handleCrearSSH(c, b)
	})
	b.Handle(&tele.Btn{Unique: "crear_zivpn"}, func(c tele.Context) error {
		return handleCrearZivpn(c, b)
	})
	b.Handle(&tele.Btn{Unique: "ssh_rnd_pass"}, func(c tele.Context) error {
		return handleRandomPass(c, b)
	})
	b.Handle(&tele.Btn{Unique: "cancelar_accion"}, func(c tele.Context) error {
		return handleCancel(c, b)
	})

	b.Handle(&tele.Btn{Unique: "back_main"}, func(c tele.Context) error {
		return handleStart(c, b) // Vuelve al inicio redibujando o editando
	})

	// Iniciar hilo de auto-limpieza (Rutina concurrente)
	go sys.AutoCleanupLoop(b)

	log.Println("Bot iniciado correctamente...")
	b.Start()
}

func isAdmin(chatID int64) bool {
	sa, _ := strconv.ParseInt(superAdmin, 10, 64)
	if chatID == sa {
		return true
	}
	data, _ := db.Load()
	_, exists := data.Admins[fmt.Sprintf("%d", chatID)]
	return exists
}

func handleStart(c tele.Context, b *tele.Bot) error {
	chatID := c.Chat().ID
	data, _ := db.Load()

	// Registrar historial
	found := false
	for _, id := range data.UserHistory {
		if id == chatID {
			found = true
			break
		}
	}
	if !found {
		data.UserHistory = append(data.UserHistory, chatID)
		db.Save(data)
	}

	// Comprobar Acceso Público
	if !data.PublicAccess && !isAdmin(chatID) {
		textoDenegado := "⛔ <b>ACCESO DENEGADO</b>\n\nEste Bot es privado."
		if c.Callback() != nil {
			return c.Edit(textoDenegado, tele.ModeHTML)
		}
		return c.Send(textoDenegado, tele.ModeHTML)
	}

	// Mostrar Menú Principal
	textoMenu := buildMainMenuText(data)
	markup := buildMainMenuMarkup(chatID)

	if c.Callback() != nil {
		return c.Edit(textoMenu, markup, tele.ModeHTML)
	}
	return c.Send(textoMenu, markup, tele.ModeHTML)
}

func buildMainMenuText(data *db.ConfigData) string {
	texto := "💎 <b>BOT TELEGRAM DEPWISE V6.7 (GO EDITION)</b>\n"
	texto += "<i>Panel de Control Avanzado</i>\n\n"

	stats := sys.GetSystemStats()

	// CPU Formatter
	barraCPU := sys.GenerarBarra(stats.CPUUsage, 100.0, 10)
	texto += fmt.Sprintf("🧠 <b>CPU:</b> [%s] <code>%.1f%%</code> (%d Cores)\n", barraCPU, stats.CPUUsage, stats.Cores)

	// RAM Formatter
	barraRAM := sys.GenerarBarra(float64(stats.RAMUsed), float64(stats.RAMTotal), 10)
	texto += fmt.Sprintf("💾 <b>RAM:</b> [%s] <code>%dMB / %dMB</code>\n", barraRAM, stats.RAMUsed, stats.RAMTotal)

	// Disco
	barraDisk := sys.GenerarBarra(float64(stats.DiskUsed), float64(stats.DiskTotal), 10)
	texto += fmt.Sprintf("💽 <b>DISCO:</b> [%s] <code>%dGB / %dGB</code>\n", barraDisk, stats.DiskUsed, stats.DiskTotal)

	texto += fmt.Sprintf("⏱️ <b>Uptime:</b> <code>%s</code>\n\n", stats.UptimeStr)

	if !data.PublicAccess {
		texto += "🔒 <i>Acceso Público: Desactivado</i>\n"
	}
	return texto
}

func buildMainMenuMarkup(chatID int64) *tele.ReplyMarkup {
	menu := &tele.ReplyMarkup{}

	btnCrear := menu.Data("👤 Crear SSH", "menu_crear")
	btnInfo := menu.Data("📡 Info Servidor", "menu_info")
	btnStats := menu.Data("📊 Mis Consumos", "my_stats")
	btnEditar := menu.Data("✏️ Editar SSH", "menu_editar")
	btnDelete := menu.Data("🗑️ Eliminar SSH", "menu_eliminar")
	btnGlobal := menu.Data("📢 Mensaje Global", "menu_broadcast")
	btnOnline := menu.Data("⚙️ Monitor Online", "menu_online")
	btnProtocols := menu.Data("⚙️ Protocolos", "menu_protocols")
	btnSettings := menu.Data("⚙️ Ajustes Pro", "menu_admins")

	sa, _ := strconv.ParseInt(superAdmin, 10, 64)
	isSA := chatID == sa
	isAdm := isAdmin(chatID)

	// Construir filas dinámicamente usando Slices
	var rows []tele.Row

	// Fila 1 (Compartida por todos)
	rows = append(rows, menu.Row(btnCrear, btnInfo))

	// Fila 2
	if isSA || isAdm {
		rows = append(rows, menu.Row(btnStats, btnEditar))
	} else {
		rows = append(rows, menu.Row(btnStats))
	}

	// Fila 3
	if isSA || isAdm {
		rows = append(rows, menu.Row(btnDelete, btnOnline))
	} else {
		rows = append(rows, menu.Row(btnDelete))
	}

	// Filas de SuperAdmin
	if isSA {
		rows = append(rows, menu.Row(btnGlobal, btnProtocols))
		rows = append(rows, menu.Row(btnSettings))
	}

	// Asignar filas al menú
	menu.Inline(rows...)

	return menu
}

func menuCrearText() string {
	return "📝 <b>¿Qué deseas crear?</b>"
}

func menuCrearMarkup() *tele.ReplyMarkup {
	menu := &tele.ReplyMarkup{}
	btnSSH := menu.Data("👤 Cliente SSH", "crear_ssh")
	btnZivpn := menu.Data("🛰️ Acceso ZIVPN", "crear_zivpn")
	btnBack := menu.Data("🔙 Volver", "back_main")

	menu.Inline(
		menu.Row(btnSSH),
		menu.Row(btnZivpn),
		menu.Row(btnBack),
	)
	return menu
}
