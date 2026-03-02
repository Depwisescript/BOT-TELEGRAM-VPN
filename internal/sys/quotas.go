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

// SetDataQuota define el límite en GB habilitando la regla de Iptables
func SetDataQuota(username string, gb float64) error {
	// 1. Limpiar cualquier regla previa (ACCEPT o REJECT)
	comment := "QUOTA_" + username
	blockComment := "BLOCKED_" + username

	exec.Command("iptables", "-D", "OUTPUT", "-m", "owner", "--uid-owner", username, "-m", "comment", "--comment", comment, "-j", "ACCEPT").Run()
	exec.Command("iptables", "-D", "OUTPUT", "-m", "owner", "--uid-owner", username, "-m", "comment", "--comment", blockComment, "-j", "REJECT").Run()
	// Borrar genérica por si acaso
	exec.Command("iptables", "-D", "OUTPUT", "-m", "owner", "--uid-owner", username, "-j", "REJECT").Run()

	if gb <= 0 {
		// Opcional: borrar archivo de limite
		os.Remove(fmt.Sprintf("/etc/ssh_limits/%s.limit", username))
		return nil
	}

	// 2. Insertar regla de conteo
	err := exec.Command("iptables", "-I", "OUTPUT", "-m", "owner", "--uid-owner", username, "-m", "comment", "--comment", comment, "-j", "ACCEPT").Run()
	if err != nil {
		return fmt.Errorf("fallo al crear regla iptables: %v", err)
	}

	// 3. Archivo de límite (para referencia)
	os.MkdirAll("/etc/ssh_limits", 0755)
	return ioutil.WriteFile(fmt.Sprintf("/etc/ssh_limits/%s.limit", username), []byte(fmt.Sprintf("%f", gb)), 0644)
}

// EnforceDataQuotas escanea iptables una sola vez y aplica bloqueos a quienes excedan su cuota
func EnforceDataQuotas() {
	// 1. Obtener todos los límites configurados
	limits := make(map[string]float64)
	files, err := ioutil.ReadDir("/etc/ssh_limits")
	if err == nil {
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
	}

	if len(limits) == 0 {
		return
	}

	// 2. Leer iptables una sola vez (-nvx para bytes exactos)
	out, err := exec.Command("iptables", "-nvx", "-L", "OUTPUT").Output()
	if err != nil {
		return
	}

	lines := strings.Split(string(out), "\n")
	for _, line := range lines {
		// Buscamos nuestras reglas de cuota
		if !strings.Contains(line, "QUOTA_") {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}

		bytesUsed, _ := strconv.ParseUint(fields[1], 10, 64)
		gbUsed := float64(bytesUsed) / 1024 / 1024 / 1024

		// Extraer el nombre de usuario del comentario de la regla
		parts := strings.Split(line, "QUOTA_")
		if len(parts) < 2 {
			continue
		}

		userRaw := strings.Fields(parts[1])[0]
		user := strings.Trim(userRaw, "*/ ")

		limit, exists := limits[user]
		if exists && gbUsed >= limit {
			// BLOQUEO INSTANTÁNEO
			// 1. Matar procesos quirúrgicamente
			pids, _ := GetUserProcesses(user)
			for _, pid := range pids {
				exec.Command("kill", "-9", pid).Run()
			}

			// 2. Bloquear tráfico definitivamente
			blockComment := "BLOCKED_" + user
			exec.Command("iptables", "-D", "OUTPUT", "-m", "owner", "--uid-owner", user, "-m", "comment", "--comment", "QUOTA_"+user, "-j", "ACCEPT").Run()

			check := exec.Command("iptables", "-C", "OUTPUT", "-m", "owner", "--uid-owner", user, "-j", "REJECT")
			if err := check.Run(); err != nil {
				exec.Command("iptables", "-I", "OUTPUT", "-m", "owner", "--uid-owner", user, "-m", "comment", "--comment", blockComment, "-j", "REJECT").Run()
			}
		}
	}
}

// ResetDataQuota limpia los contadores del usuario borrando e insertando la regla de Iptables
func ResetDataQuota(username string) error {
	limit := 0.0
	if b, err := ioutil.ReadFile(fmt.Sprintf("/etc/ssh_limits/%s.limit", username)); err == nil {
		limit, _ = strconv.ParseFloat(strings.TrimSpace(string(b)), 64)
	}
	return SetDataQuota(username, limit)
}

// GetUserConsumption lee `iptables -nvx -L OUTPUT` y retorna los GB gastados y el límite
func GetUserConsumption(username string) (float64, string, error) {
	limitStr := "Infinito"
	if b, err := ioutil.ReadFile(fmt.Sprintf("/etc/ssh_limits/%s.limit", username)); err == nil {
		limitStr = strings.TrimSpace(string(b))
	}

	outUID, err := exec.Command("id", "-u", username).Output()
	if err != nil {
		return 0, "0", fmt.Errorf("el usuario %s no existe", username)
	}
	uid := strings.TrimSpace(string(outUID))

	cmdOutput, err := exec.Command("iptables", "-nvx", "-L", "OUTPUT").Output()
	if err != nil {
		return 0, limitStr, nil
	}

	bytesUsed := uint64(0)
	lines := strings.Split(string(cmdOutput), "\n")
	for _, line := range lines {
		if strings.Contains(line, "owner") && (strings.Contains(line, uid) || strings.Contains(line, username)) {
			fields := strings.Fields(line)
			if len(fields) >= 2 {
				b, errConv := strconv.ParseUint(fields[1], 10, 64)
				if errConv == nil {
					bytesUsed = b
					break
				}
			}
		}
	}

	gbUsed := float64(bytesUsed) / 1024 / 1024 / 1024
	return gbUsed, limitStr, nil
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

// GetUserProcesses devuelve una lista de PIDs de procesos SSH/Dropbear de un usuario, ordenados por fecha de inicio (antiguos primero)
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
