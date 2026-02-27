package sys

import (
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/Depwisescript/BOT-TELEGRAM-VPN/internal/db"
	"github.com/Depwisescript/BOT-TELEGRAM-VPN/internal/vpn"
	tele "gopkg.in/telebot.v3"
)

// CountZivpnActive returns true if any UDP session exists for zivpn
func CountZivpnActive() bool {
	out, err := exec.Command("sh", "-c", "ss -u -n -p | grep 'zivpn' | wc -l").Output()
	if err != nil {
		return false
	}
	count := strings.TrimSpace(string(out))
	return count != "" && count != "0"
}

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
						delete(data.SSHLastActive, user)
					}
				}

				// Revisar ZiVPN - auto-expiración por fecha
				for pass, expire := range data.ZivpnUsers {
					if now > expire {
						vpn.RemoveZivpnUser(pass)
						delete(data.ZivpnUsers, pass)
						delete(data.ZivpnOwners, pass)
						delete(data.ZivpnLastActive, pass)
					}
				}

				// 3. Limpieza de Usuarios Zombi SSH (12 Horas de inactividad)
				online, _ := CountOnlineConnections()
				saID := os.Getenv("SUPER_ADMIN")
				for user, ownerID := range data.SSHOwners {
					// Obviar si es del SuperAdmin
					if ownerID == saID {
						continue
					}

					// Si está online, actualizar rastro
					if _, isOnline := online[user]; isOnline {
						data.SSHLastActive[user] = time.Now().Format(time.RFC3339)
						continue
					}

					// Si no está online, ver cuándo fue la última vez
					lastStr, exists := data.SSHLastActive[user]
					if !exists {
						// Si no existe rastro, inicializar con ahora (dar 12h de gracia)
						data.SSHLastActive[user] = time.Now().Format(time.RFC3339)
						continue
					}

					lastTime, err := time.Parse(time.RFC3339, lastStr)
					if err == nil {
						if time.Since(lastTime) > 12*time.Hour {
							// Borrar Zombi
							DeleteSSHUser(user)
							delete(data.SSHOwners, user)
							delete(data.SSHTimeUsers, user)
							delete(data.SSHLastActive, user)
						}
					}
				}

				// 4. Limpieza de Usuarios Zombi ZiVPN (12 Horas de inactividad)
				zivpnHasTraffic := CountZivpnActive()
				for pass, ownerID := range data.ZivpnOwners {
					// Obviar si es del SuperAdmin
					if ownerID == saID {
						continue
					}

					if zivpnHasTraffic {
						// Si hay tráfico activo, actualizar rastro para todas las contraseñas activas
						data.ZivpnLastActive[pass] = time.Now().Format(time.RFC3339)
						continue
					}

					// Si no hay tráfico, revisar antigüedad
					lastStr, exists := data.ZivpnLastActive[pass]
					if !exists {
						data.ZivpnLastActive[pass] = time.Now().Format(time.RFC3339)
						continue
					}

					lastTime, err := time.Parse(time.RFC3339, lastStr)
					if err == nil {
						if time.Since(lastTime) > 12*time.Hour {
							vpn.RemoveZivpnUser(pass)
							delete(data.ZivpnUsers, pass)
							delete(data.ZivpnOwners, pass)
							delete(data.ZivpnLastActive, pass)
						}
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
