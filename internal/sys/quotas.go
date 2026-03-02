package sys

import (
	"fmt"
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

// CleanUserRules borra las reglas de bloqueo de un usuario
func CleanUserRules(username string) {
	tables := []string{"iptables", "ip6tables"}
	blockComment := "BLOCKED_" + username

	for _, ipt := range tables {
		// Borrar regla de bloqueo
		exec.Command(ipt, "-D", "OUTPUT", "-m", "owner", "--uid-owner", username, "-m", "comment", "--comment", blockComment, "-j", "REJECT").Run()
		exec.Command(ipt, "-D", "OUTPUT", "-m", "owner", "--uid-owner", username, "-j", "REJECT").Run()
	}
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
