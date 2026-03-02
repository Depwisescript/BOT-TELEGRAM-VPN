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
	if gb <= 0 {
		return nil // Sin límite
	}

	// Usamos un comentario en la regla de iptables para identificarla fácilmente
	comment := fmt.Sprintf("QUOTA_%s", username)

	// 1. Iptables (primero borramos por si acaso la vieja, ignoramos error borrado)
	exec.Command("iptables", "-D", "OUTPUT", "-m", "owner", "--uid-owner", username, "-m", "comment", "--comment", comment, "-j", "ACCEPT").Run()

	err := exec.Command("iptables", "-I", "OUTPUT", "-m", "owner", "--uid-owner", username, "-m", "comment", "--comment", comment, "-j", "ACCEPT").Run()
	if err != nil {
		return fmt.Errorf("fallo al crear regla iptables: %v", err)
	}

	// 2. Archivo de límite (para referencia)
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
		// La regla suele contener "/* QUOTA_username */"
		parts := strings.Split(line, "QUOTA_")
		if len(parts) < 2 {
			continue
		}
		user := strings.Fields(parts[1])[0]
		user = strings.Trim(user, "*/ ")

		limit, exists := limits[user]
		if exists && gbUsed >= limit {
			// BLOQUEO INSTANTÁNEO
			// 1. Matar procesos quirúrgicamente
			pids, _ := GetUserProcesses(user)
			for _, pid := range pids {
				exec.Command("kill", "-9", pid).Run()
			}

			// 2. Bloquear tráfico definitivamente reemplazando ACCEPT por REJECT para este usuario
			comment := "QUOTA_" + user
			exec.Command("iptables", "-D", "OUTPUT", "-m", "owner", "--uid-owner", user, "-m", "comment", "--comment", comment, "-j", "ACCEPT").Run()
			// Insertar REJECT (solo si no existe ya)
			check := exec.Command("iptables", "-C", "OUTPUT", "-m", "owner", "--uid-owner", user, "-j", "REJECT")
			if err := check.Run(); err != nil {
				exec.Command("iptables", "-I", "OUTPUT", "-m", "owner", "--uid-owner", user, "-j", "REJECT").Run()
			}
		}
	}
}

// ResetDataQuota limpia los contadores del usuario borrando e insertando la regla de Iptables
func ResetDataQuota(username string) error {
	exec.Command("iptables", "-D", "OUTPUT", "-m", "owner", "--uid-owner", username, "-j", "ACCEPT").Run()
	return exec.Command("iptables", "-I", "OUTPUT", "-m", "owner", "--uid-owner", username, "-j", "ACCEPT").Run()
}

// GetUserConsumption lee `iptables -nvx -L OUTPUT` y retorna los GB gastados y el límite
func GetUserConsumption(username string) (float64, string, error) {
	// Límite configurado
	limitStr := "Infinito"
	if b, err := ioutil.ReadFile(fmt.Sprintf("/etc/ssh_limits/%s.limit", username)); err == nil {
		limitStr = strings.TrimSpace(string(b))
	}

	// UID del usuario
	outUID, err := exec.Command("id", "-u", username).Output()
	if err != nil {
		return 0, "0", fmt.Errorf("el usuario %s no existe", username)
	}
	uid := strings.TrimSpace(string(outUID))

	// Iptables
	cmdOutput, err := exec.Command("iptables", "-nvx", "-L", "OUTPUT").Output()
	if err != nil {
		// Tolerante a fallos de IPTables si no existe
		return 0, limitStr, nil
	}

	bytesUsed := uint64(0)

	// Parsear respuesta IPTables
	// Formato típico: "pkts      bytes target     prot opt in     out     source               destination         "
	// Lleno de reglas. Buscamos owner UID match `uid` o `username`
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
		// Ignorar root y grep
		if strings.Contains(line, "root") || strings.Contains(line, "grep") {
			continue
		}

		// OpenSSH: "sshd: username [priv]" o "sshd: username@pts"
		// Dropbear: "dropbear -R" ejecutado bajo el usuario
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
		return 0 // Sin límite aparente o error
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
	// ps -u user -o pid,cmd --no-headers --sort=start_time
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
			// El usuario excedió el límite.
			// Obtenemos sus PIDs ordenados por antigüedad (los primeros son los más viejos).
			pids, err := GetUserProcesses(user)
			if err != nil || len(pids) <= maxLogins {
				continue
			}

			// Matamos los que sobran (los más recientes)
			// Los PIDs a partir del índice maxLogins son los "extras".
			for i := maxLogins; i < len(pids); i++ {
				exec.Command("kill", "-9", pids[i]).Run()
			}
		}
	}
}
