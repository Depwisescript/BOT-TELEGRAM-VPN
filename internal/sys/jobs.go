package sys

import (
	"time"

	"github.com/Depwisescript/BOT-TELEGRAM-VPN/internal/db"
	tele "gopkg.in/telebot.v3"
)

// AutoCleanupLoop corre en un hilo separado ejecutando la limpieza de Iptables
// y usuarios excedidos cada cierto tiempo.
func AutoCleanupLoop(b *tele.Bot) {
	tick := 0
	for {
		// 1. Limpieza de usuarios vencidos de forma periódica
		if tick >= 9 { // Cada 60-70 segundos aprox
			db.Update(func(data *db.ConfigData) error {
				now := time.Now().Format("2006-01-02")

				// Revisar SSH
				for user, expire := range data.SSHTimeUsers {
					if now > expire {
						DeleteSSHUser(user)
						delete(data.SSHTimeUsers, user)
						delete(data.SSHOwners, user)
					}
				}

				// Revisar ZiVPN
				for pass, expire := range data.ZivpnUsers {
					if now > expire {
						// Limpiar si hubiese funcion de borrado de ZiVPN
						delete(data.ZivpnUsers, pass)
						delete(data.ZivpnOwners, pass)
					}
				}
				return nil
			})

			// syncIptables() -> pendiente implementación detallada si aplica
			tick = 0
		}

		tick++
		time.Sleep(7 * time.Second)
	}
}
