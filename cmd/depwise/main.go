package main

import (
	"log"

	"github.com/Depwisescript/BOT-TELEGRAM-VPN/internal/bot"
)

func main() {
	log.Println("Iniciando Depwise SSH VPN Manager...")

	// Iniciar servidor del bot (bloqueante)
	bot.StartBot()
}
