package sys

import (
	"fmt"
	"math"
	"os/exec"
	"strconv"
	"strings"
)

// Info contiene la información parseada del VPS
type VPSInfo struct {
	CPUUsage   float64
	CPUModel   string
	Cores      int
	RAMTotal   int // in MB
	RAMUsed    int // in MB
	DiskTotal  int // in GB
	DiskUsed   int // in GB
	UptimeStr  string
}

func getBar(percentage float64, length int) string {
	filled := int(math.Round((percentage / 100.0) * float64(length)))
	if filled > length {
		filled = length
	}
	if filled < 0 {
		filled = 0
	}
	return strings.Repeat("■", filled) + strings.Repeat("□", length-filled)
}

// GenerarBarra crea la vista de texto visual para Telegram
func GenerarBarra(actual, total float64, length int) string {
    if total == 0 {
		return getBar(0, length)
	}
	porcentaje := (actual / total) * 100.0
	return getBar(porcentaje, length)
}

// GetSystemStats extrae métricas vitales usando utilidades base de Linux
func GetSystemStats() VPSInfo {
	var info VPSInfo

	// 1. CPU
	outTop, err := exec.Command("top", "-bn1").Output()
	if err == nil {
		lines := strings.Split(string(outTop), "\n")
		for _, line := range lines {
			if strings.Contains(line, "%Cpu(s):") {
				// Formato: %Cpu(s):  1.5 us,  0.5 sy,  0.0 ni, 97.5 id,  0.0 wa,
				parts := strings.Split(line, ",")
				for _, p := range parts {
					if strings.Contains(p, "id") {
						idleStr := strings.TrimSpace(strings.Replace(p, "id", "", -1))
						idle, _ := strconv.ParseFloat(idleStr, 64)
						info.CPUUsage = 100.0 - idle
					}
				}
				break
			}
		}
	}

	// 2. Cores y Modelo
	outCpu, err := exec.Command("lscpu").Output()
	if err == nil {
		lines := strings.Split(string(outCpu), "\n")
		for _, line := range lines {
			if strings.HasPrefix(line, "CPU(s):") {
				parts := strings.Fields(line)
				if len(parts) >= 2 {
					info.Cores, _ = strconv.Atoi(parts[1])
				}
			}
			if strings.HasPrefix(line, "Model name:") {
				info.CPUModel = strings.TrimSpace(strings.Split(line, ":")[1])
			}
		}
	}

	// 3. RAM (usando free -m)
	outFree, err := exec.Command("free", "-m").Output()
	if err == nil {
		lines := strings.Split(string(outFree), "\n")
		if len(lines) >= 2 {
			fields := strings.Fields(lines[1]) // Mem: line
			if len(fields) >= 3 {
				info.RAMTotal, _ = strconv.Atoi(fields[1])
				info.RAMUsed, _ = strconv.Atoi(fields[2])
			}
		}
	}

	// 4. Disco (usando df -h /)
	outDf, err := exec.Command("df", "-B1G", "/").Output() // Gigabytes
	if err == nil {
		lines := strings.Split(string(outDf), "\n")
		if len(lines) >= 2 {
			fields := strings.Fields(lines[1])
			if len(fields) >= 4 {
				// Remover la G final
				totalStr := strings.TrimRight(fields[1], "G")
				usedStr := strings.TrimRight(fields[2], "G")
				info.DiskTotal, _ = strconv.Atoi(totalStr)
				info.DiskUsed, _ = strconv.Atoi(usedStr)
			}
		}
	}

	return info
}

// GetOnlineUsers cuenta las conexiones SSH activas
func GetOnlineUsers() []string {
	// ps aux | grep sshd | grep -v root | grep -v grep | awk '{print $1}' | sort | uniq -c
	out, err := exec.Command("sh", "-c", "ps aux | grep sshd | grep -v root | grep -v grep | awk '{print $1}' | sort | uniq -c").Output()
	if err != nil {
		return []string{}
	}

	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	var result []string
	for _, line := range lines {
		if line == "" {
			continue
		}
		parts := strings.Fields(line)
		if len(parts) >= 2 {
			count := parts[0]
			user := parts[1]
			result = append(result, fmt.Sprintf("👤 %s: %s conexion(es)", user, count))
		}
	}
	return result
}

// GetZivpnOnline cuenta las conexiones UDP a Zivpn
func GetZivpnOnline() []string {
	// ss -u -n -p | grep "zivpn" | awk '{print $5}' | cut -d: -f1 | sort | uniq -c
	out, err := exec.Command("sh", "-c", "ss -u -n -p | grep 'zivpn' | awk '{print $5}' | cut -d: -f1 | sort | uniq -c").Output()
	if err != nil {
		return []string{}
	}

	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	var result []string
	for _, line := range lines {
		if line == "" {
			continue
		}
		parts := strings.Fields(line)
		if len(parts) >= 2 {
			count := parts[0]
			ip := parts[1]
			result = append(result, fmt.Sprintf("🌐 IP:%s (%s sesion)", ip, count))
		}
	}
	return result
}

// GetPublicIP retorna la IP pública del servidor con múltiples fallbacks (IPv4 garantizado)
func GetPublicIP() string {
	urls := []string{
		"http://ip4.icanhazip.com",
		"http://api.ipify.org",
		"http://ifconfig.me/ip",
		"http://checkip.amazonaws.com",
	}

	for _, url := range urls {
		// -4 fuerza IPv4, --connect-timeout evita bloqueos largos
		out, err := exec.Command("curl", "-4", "-s", "--connect-timeout", "5", url).Output()
		if err == nil && len(out) > 5 {
			ip := strings.TrimSpace(string(out))
			if strings.Count(ip, ".") == 3 { // Validar formato básico IPv4
				return ip
			}
		}
	}
	return "Desconocida"
}
