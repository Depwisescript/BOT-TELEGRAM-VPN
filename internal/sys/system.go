package sys

import (
	"fmt"
	"os/exec"
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
