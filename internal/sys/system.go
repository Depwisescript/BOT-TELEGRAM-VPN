package sys

import (
	"fmt"
	"io/ioutil"
	"os/exec"
	"strconv"
	"strings"
)

// PerformFullCleanup realiza una limpieza profunda del SSD
func PerformFullCleanup() (string, error) {
	var report string

	// 1. Limpieza de APT
	report += "📦 <b>APT:</b> Limpiando caché y paquetes huérfanos...\n"
	_ = exec.Command("apt-get", "clean").Run()
	_ = exec.Command("apt-get", "autoremove", "-y").Run()

	// 2. Rotación de Logs (Journalctl)
	report += "📑 <b>Logs:</b> Reduciendo logs del sistema a 1 día...\n"
	_ = exec.Command("journalctl", "--vacuum-time=1d").Run()

	// 3. Limpiar temporales de compilación
	report += "🧹 <b>Temp:</b> Borrando carpetas de instalación temporales...\n"
	_ = exec.Command("rm", "-rf", "/tmp/BOT-TELEGRAM-VPN").Run()
	_ = exec.Command("rm", "-rf", "/root/go/pkg").Run()

	// 4. Limpiar caché de compilación de Go (si existe el binario)
	if _, err := exec.LookPath("go"); err == nil {
		report += "🐹 <b>Go:</b> Limpiando caché de módulos y build...\n"
		_ = exec.Command("go", "clean", "-cache", "-modcache").Run()
	}

	// 5. Borrar archivos de logs antiguos del bot (si los hay)
	_ = exec.Command("rm", "-f", "/var/log/depwise-bot.log*").Run()

	// Obtener espacio libre final
	freeSpace := "N/A"
	stats := GetSystemStats()
	freeSpace = fmt.Sprintf("%d GB", stats.DiskTotal-stats.DiskUsed)

	report += "\n✅ <b>¡LIMPIEZA COMPLETADA!</b>\n"
	report += fmt.Sprintf("💾 <b>Espacio Disponible:</b> <code>%s</code>", freeSpace)

	return report, nil
}

// GetGlobalTraffic lee /proc/net/dev y extrae el tráfico total (Subida/Bajada) en GB
func GetGlobalTraffic() (float64, float64) {
	data, err := ioutil.ReadFile("/proc/net/dev")
	if err != nil {
		return 0, 0
	}

	lines := strings.Split(string(data), "\n")
	var totalRX, totalTX uint64

	for _, line := range lines {
		if !strings.Contains(line, ":") {
			continue
		}
		parts := strings.Split(line, ":")
		if len(parts) < 2 {
			continue
		}
		iface := strings.TrimSpace(parts[0])
		// Ignorar interfaces virtuales/locales habituales
		if iface == "lo" || strings.HasPrefix(iface, "tun") || strings.HasPrefix(iface, "docker") || strings.HasPrefix(iface, "veth") {
			continue
		}

		fields := strings.Fields(parts[1])
		if len(fields) >= 9 {
			rx, _ := strconv.ParseUint(fields[0], 10, 64)
			tx, _ := strconv.ParseUint(fields[8], 10, 64)
			totalRX += rx
			totalTX += tx
		}
	}

	gbRX := float64(totalRX) / 1024 / 1024 / 1024
	gbTX := float64(totalTX) / 1024 / 1024 / 1024
	return gbRX, gbTX
}
