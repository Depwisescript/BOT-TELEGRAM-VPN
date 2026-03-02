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

	// 1. Iptables (primero borramos por si acaso la vieja, ignoramos error borrado)
	exec.Command("iptables", "-D", "OUTPUT", "-m", "owner", "--uid-owner", username, "-j", "ACCEPT").Run()

	err := exec.Command("iptables", "-I", "OUTPUT", "-m", "owner", "--uid-owner", username, "-j", "ACCEPT").Run()
	if err != nil {
		return fmt.Errorf("fallo al crear regla iptables: %v", err)
	}

	// 2. Archivo de límite (para referencia)
	os.MkdirAll("/etc/ssh_limits", 0755)
	return ioutil.WriteFile(fmt.Sprintf("/etc/ssh_limits/%s.limit", username), []byte(fmt.Sprintf("%f", gb)), 0644)
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

// EnforceConnectionLimits revisa las conexiones activas y mata procesos si exceden el límite
func EnforceConnectionLimits() {
	connections, err := CountOnlineConnections()
	if err != nil {
		return
	}

	for user, activeCount := range connections {
		// Obtener límite del usuario
		maxLogins := GetUserMaxLogins(user)
		if maxLogins > 0 && activeCount > maxLogins {
			// El usuario excedió el límite. Matar todas sus conexiones SSH/Dropbear iterativamente o la más reciente.
			// Para simplificar y asegurar que se respeta, matamos todos sus procesos SSH/Dropbear.
			// Cuando intenten reconectar, el límite de PAM actuará, y este script sirve como barredora activa.
			exec.Command("killall", "-u", user, "sshd").Run()
			exec.Command("killall", "-u", user, "dropbear").Run()
		}
	}
}
