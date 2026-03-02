package sys

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

// SetConnectionLimit añade una regla a limits.conf para el usuario
func SetConnectionLimit(username string, maxLogins int) error {
	// Limpiar previos
	exec.Command("sed", "-i", fmt.Sprintf("/^%s hard maxlogins/d", username), "/etc/security/limits.conf").Run()

	if maxLogins <= 0 {
		return nil // Sin límite
	}

	// Abrimos en modo append
	f, err := os.OpenFile("/etc/security/limits.conf", os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	line := fmt.Sprintf("%s hard maxlogins %d\n", username, maxLogins)
	_, err = f.WriteString(line)
	return err
}

// SetDataQuota define el límite en GB habilitando la regla de Iptables (v4 y v6) con seguimiento bidireccional
func SetDataQuota(username string, gb float64) error {
	// 1. Obtener UID
	outUID, err := exec.Command("id", "-u", username).Output()
	if err != nil {
		return fmt.Errorf("usuario %s no existe", username)
	}
	uidStr := strings.TrimSpace(string(outUID))
	uid, _ := strconv.Atoi(uidStr)

	// 1. Limpiar cualquier regla previa de forma agresiva
	CleanUserRules(username)

	if gb <= 0 {
		os.Remove(fmt.Sprintf("/etc/ssh_limits/%s.limit", username))
		return nil
	}

	// 3. Crear reglas de MARCADO y CONTEO
	// Usamos el UID como marca hexadecimal para evitar colisiones
	mark := fmt.Sprintf("0x%x", uid)
	comment := "QUOTA_" + username

	// --- IPv4 Global (Solo si no existe) ---
	// Restaurar marca en paquetes entrantes para poder contarlos (Global para todos los usuarios)
	_ = exec.Command("iptables", "-t", "mangle", "-C", "PREROUTING", "-j", "CONNMARK", "--restore-mark").Run()
	if checkErr := exec.Command("iptables", "-t", "mangle", "-C", "PREROUTING", "-j", "CONNMARK", "--restore-mark").Run(); checkErr != nil {
		exec.Command("iptables", "-t", "mangle", "-I", "PREROUTING", "1", "-j", "CONNMARK", "--restore-mark").Run()
	}

	// --- IPv4 User Specific ---
	// Marcar conexiones salientes por UID
	exec.Command("iptables", "-t", "mangle", "-I", "OUTPUT", "1", "-m", "owner", "--uid-owner", username, "-j", "CONNMARK", "--set-mark", mark).Run()

	// Reglas de ACUMULACIÓN (Conteo) con prioridad -I 1
	// Salida (Upload desde el servidor / Download para el usuario)
	exec.Command("iptables", "-I", "OUTPUT", "1", "-m", "owner", "--uid-owner", username, "-m", "comment", "--comment", comment, "-j", "ACCEPT").Run()
	// Entrada (Download desde el servidor / Upload para el usuario)
	exec.Command("iptables", "-I", "INPUT", "1", "-m", "mark", "--mark", mark, "-m", "comment", "--comment", "IN_"+comment, "-j", "ACCEPT").Run()

	// --- IPv6 Global (Solo si no existe) ---
	if checkErr := exec.Command("ip6tables", "-t", "mangle", "-C", "PREROUTING", "-j", "CONNMARK", "--restore-mark").Run(); checkErr != nil {
		exec.Command("ip6tables", "-t", "mangle", "-I", "PREROUTING", "1", "-j", "CONNMARK", "--restore-mark").Run()
	}

	// --- IPv6 User Specific ---
	exec.Command("ip6tables", "-t", "mangle", "-I", "OUTPUT", "1", "-m", "owner", "--uid-owner", username, "-j", "CONNMARK", "--set-mark", mark).Run()

	exec.Command("ip6tables", "-I", "OUTPUT", "1", "-m", "owner", "--uid-owner", username, "-m", "comment", "--comment", comment, "-j", "ACCEPT").Run()
	exec.Command("ip6tables", "-I", "INPUT", "1", "-m", "mark", "--mark", mark, "-m", "comment", "--comment", "IN_"+comment, "-j", "ACCEPT").Run()

	// 4. Guardar límite
	os.MkdirAll("/etc/ssh_limits", 0755)
	return ioutil.WriteFile(fmt.Sprintf("/etc/ssh_limits/%s.limit", username), []byte(fmt.Sprintf("%f", gb)), 0644)
}

// CleanUserRules borra todas las posibles variaciones de reglas de un usuario (Exportada para sys)
func CleanUserRules(username string) {
	outUID, _ := exec.Command("id", "-u", username).Output()
	uidStr := strings.TrimSpace(string(outUID))
	mark := ""
	if uidStr != "" {
		u, _ := strconv.Atoi(uidStr)
		mark = fmt.Sprintf("0x%x", u)
	}

	comment := "QUOTA_" + username
	inComment := "IN_QUOTA_" + username
	blockComment := "BLOCKED_" + username

	tables := []string{"iptables", "ip6tables"}
	for _, ipt := range tables {
		// Borrar reglas de conteo
		exec.Command(ipt, "-D", "OUTPUT", "-m", "owner", "--uid-owner", username, "-m", "comment", "--comment", comment, "-j", "ACCEPT").Run()
		exec.Command(ipt, "-D", "INPUT", "-m", "mark", "--mark", mark, "-m", "comment", "--comment", inComment, "-j", "ACCEPT").Run()

		// Borrar regla de bloqueo
		exec.Command(ipt, "-D", "OUTPUT", "-m", "owner", "--uid-owner", username, "-m", "comment", "--comment", blockComment, "-j", "REJECT").Run()
		exec.Command(ipt, "-D", "OUTPUT", "-m", "owner", "--uid-owner", username, "-j", "REJECT").Run()

		// Borrar reglas de mangle (marcado)
		if mark != "" {
			exec.Command(ipt, "-t", "mangle", "-D", "OUTPUT", "-m", "owner", "--uid-owner", username, "-j", "CONNMARK", "--set-mark", mark).Run()
		}
	}
}

// EnforceDataQuotas escanea iptables y aplica bloqueos
func EnforceDataQuotas() {
	limits := make(map[string]float64)
	files, err := ioutil.ReadDir("/etc/ssh_limits")
	if err != nil {
		return
	}
	for _, f := range files {
		if strings.HasSuffix(f.Name(), ".limit") {
			user := strings.TrimSuffix(f.Name(), ".limit")
			if b, err := ioutil.ReadFile("/etc/ssh_limits/" + f.Name()); err == nil {
				if val, err := strconv.ParseFloat(strings.TrimSpace(string(b)), 64); err == nil {
					limits[user] = val
				}
			}
		}
	}

	if len(limits) == 0 {
		return
	}

	usageData := make(map[string]float64)
	collectAllUsage("iptables", usageData)
	collectAllUsage("ip6tables", usageData)

	for user, gbUsed := range usageData {
		limit, exists := limits[user]
		if exists && gbUsed >= limit {
			// BLOQUEO
			pids, _ := GetUserProcesses(user)
			for _, pid := range pids {
				exec.Command("kill", "-9", pid).Run()
			}
			// Re-aplicar limpieza y bloqueo REJECT
			CleanUserRules(user)
			blockComment := "BLOCKED_" + user
			exec.Command("iptables", "-I", "OUTPUT", "-m", "owner", "--uid-owner", user, "-m", "comment", "--comment", blockComment, "-j", "REJECT").Run()
			exec.Command("ip6tables", "-I", "OUTPUT", "-m", "owner", "--uid-owner", user, "-m", "comment", "--comment", blockComment, "-j", "REJECT").Run()
		}
	}
}

func collectAllUsage(cmd string, data map[string]float64) {
	// Recolectar de OUTPUT e INPUT
	chains := []string{"OUTPUT", "INPUT"}
	for _, chain := range chains {
		out, err := exec.Command(cmd, "-nvx", "-L", chain).Output()
		if err != nil {
			continue
		}
		lines := strings.Split(string(out), "\n")
		for _, line := range lines {
			if !strings.Contains(line, "QUOTA_") {
				continue
			}
			fields := strings.Fields(line)
			if len(fields) < 2 {
				continue
			}
			bytesUsed, _ := strconv.ParseUint(fields[1], 10, 64)

			// Determinar usuario
			user := ""
			if strings.Contains(line, "IN_QUOTA_") {
				parts := strings.Split(line, "IN_QUOTA_")
				user = strings.Fields(parts[1])[0]
			} else {
				parts := strings.Split(line, "QUOTA_")
				user = strings.Fields(parts[1])[0]
			}
			user = strings.Trim(user, "*/ ")
			data[user] += float64(bytesUsed) / 1024 / 1024 / 1024
		}
	}
}

func GetUserConsumption(username string) (float64, string, error) {
	limitStr := "Infinito"
	if b, err := ioutil.ReadFile(fmt.Sprintf("/etc/ssh_limits/%s.limit", username)); err == nil {
		limitStr = strings.TrimSpace(string(b))
	}

	usageData := make(map[string]float64)
	collectAllUsage("iptables", usageData)
	collectAllUsage("ip6tables", usageData)

	return usageData[username], limitStr, nil
}

// CountOnlineConnections devuelve el número total de conexiones SSH y Dropbear por usuario
func CountOnlineConnections() (map[string]int, error) {
	connections := make(map[string]int)
	out, err := exec.Command("ps", "aux").Output()
	if err != nil {
		return connections, err
	}

	lines := strings.Split(string(out), "\n")
	for _, line := range lines {
		if strings.Contains(line, "root") || strings.Contains(line, "grep") {
			continue
		}
		if strings.Contains(line, "sshd:") || strings.Contains(line, "dropbear") {
			fields := strings.Fields(line)
			if len(fields) > 0 {
				user := fields[0]
				connections[user]++
			}
		}
	}
	return connections, nil
}

// GetUserMaxLogins lee el límite de conexiones configurado en limits.conf para un usuario dado
func GetUserMaxLogins(username string) int {
	out, err := exec.Command("grep", fmt.Sprintf("^%s hard maxlogins", username), "/etc/security/limits.conf").Output()
	if err != nil {
		return 0
	}
	fields := strings.Fields(string(out))
	if len(fields) >= 4 {
		lim, _ := strconv.Atoi(fields[3])
		return lim
	}
	return 0
}

// GetUserProcesses devuelve una lista de PIDs de procesos SSH/Dropbear de un usuario (Exportado para sys)
func GetUserProcesses(username string) ([]string, error) {
	out, err := exec.Command("ps", "-u", username, "-o", "pid,cmd", "--no-headers", "--sort=start_time").Output()
	if err != nil {
		return nil, err
	}

	var pids []string
	lines := strings.Split(string(out), "\n")
	for _, line := range lines {
		if strings.Contains(line, "sshd:") || strings.Contains(line, "dropbear") {
			fields := strings.Fields(line)
			if len(fields) > 0 {
				pids = append(pids, fields[0])
			}
		}
	}
	return pids, nil
}

// SyncAllIptables sincroniza todas las reglas de iptables con los límites actuales
func SyncAllIptables() {
	files, err := ioutil.ReadDir("/etc/ssh_limits")
	if err != nil {
		return
	}

	for _, f := range files {
		if strings.HasSuffix(f.Name(), ".limit") {
			user := strings.TrimSuffix(f.Name(), ".limit")
			if b, err := ioutil.ReadFile("/etc/ssh_limits/" + f.Name()); err == nil {
				if val, err := strconv.ParseFloat(strings.TrimSpace(string(b)), 64); err == nil {
					SetDataQuota(user, val)
				}
			}
		}
	}
}

// EnforceConnectionLimits revisa las conexiones activas y mata procesos quirúrgicamente si exceden el límite
func EnforceConnectionLimits() {
	connections, err := CountOnlineConnections()
	if err != nil {
		return
	}

	for user, activeCount := range connections {
		maxLogins := GetUserMaxLogins(user)
		if maxLogins > 0 && activeCount > maxLogins {
			pids, err := GetUserProcesses(user)
			if err != nil || len(pids) <= maxLogins {
				continue
			}
			for i := maxLogins; i < len(pids); i++ {
				exec.Command("kill", "-9", pids[i]).Run()
			}
		}
	}
}
