package sys

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"time"
)

// ExecCmdRun es una función auxiliar para ejecutar comandos del sistema (bash) de manera limpia
func ExecCmdRun(command string, args ...string) (string, error) {
	cmd := exec.Command(command, args...)

	var out bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		return "", fmt.Errorf("cmd error: %v, stderr: %s", err, stderr.String())
	}

	return out.String(), nil
}

// CreateSSHUser crea un usuario en el sistema con expiración y contraseña.
func CreateSSHUser(username string, password string, days int) error {
	// 1. Calcular Fecha Vencimiento
	expireDate := time.Now().AddDate(0, 0, days).Format("2006-01-02")

	// 2. Ejecutar useradd -m -s /bin/bash -e "fecha" "usuario"
	_, err := ExecCmdRun("useradd", "-m", "-s", "/bin/bash", "-e", expireDate, username)
	if err != nil {
		return fmt.Errorf("fallo al crear usuario: %v", err)
	}

	// 3. chpasswd
	// En Go podemos usar la entrada estándar del comando para chpasswd
	cmd := exec.Command("chpasswd")
	cmd.Stdin = bytes.NewBufferString(fmt.Sprintf("%s:%s", username, password))
	if err := cmd.Run(); err != nil {
		// Rollback (borramos usuario si chpasswd falla)
		_ = DeleteSSHUser(username)
		return fmt.Errorf("fallo al asignar contraseña: %v", err)
	}

	return nil
}

// DeleteSSHUser borra el usuario, home y reglas asociadas de iptables
func DeleteSSHUser(username string) error {
	// 1. Matar procesos primero (para desocupar userdel)
	pids, _ := GetUserProcesses(username)
	for _, pid := range pids {
		exec.Command("kill", "-9", pid).Run()
	}

	// 2. Limpiar Iptables de inmediato (Módulo Quotas robusto)
	CleanUserRules(username)

	// 3. Borrar usuario con sed y userdel (con timeout manual para no congelar)
	// Eliminamos rastro en limits
	exec.Command("sed", "-i", fmt.Sprintf("/^%s hard maxlogins/d", username), "/etc/security/limits.conf").Run()

	// userdel -f -r (usamos Command directo para evitar bloqueos infinitos si algo sale mal)
	cmd := exec.Command("userdel", "-f", "-r", username)
	// Esperar max 10 segundos
	done := make(chan error, 1)
	go func() { done <- cmd.Run() }()

	select {
	case <-time.After(10 * time.Second):
		if cmd.Process != nil {
			cmd.Process.Kill()
		}
		fmt.Printf("Aviso: userdel para %s tardó demasiado y fue terminado.\n", username)
	case err := <-done:
		if err != nil {
			fmt.Printf("Aviso: error en userdel para %s: %v\n", username, err)
		}
	}

	// 4. Archivo limit
	os.Remove(fmt.Sprintf("/etc/ssh_limits/%s.limit", username))

	return nil
}

// UpdateSSHUserPassword cambia la contraseña de un usuario SSH
func UpdateSSHUserPassword(username, newPassword string) error {
	cmd := exec.Command("chpasswd")
	cmd.Stdin = bytes.NewBufferString(fmt.Sprintf("%s:%s", username, newPassword))
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("fallo al actualizar contraseña: %v", err)
	}
	return nil
}

// RenewSSHUser renueva un usuario sumando dias a la fecha actual y lo desbloquea.
func RenewSSHUser(username string, days int) error {
	expireDate := time.Now().AddDate(0, 0, days).Format("2006-01-02")

	// Cambiar expiracion
	_, err := ExecCmdRun("usermod", "-e", expireDate, username)
	if err != nil {
		return err
	}

	// Desbloquear por si estaba vencido
	ExecCmdRun("passwd", "-u", username)
	return nil
}

// SetSSHBanner configura el banner de bienvenida de SSH
func SetSSHBanner(text string) error {
	// Guardar en /etc/sshd_banner
	err := exec.Command("sh", "-c", fmt.Sprintf("echo '%s' > /etc/sshd_banner", text)).Run()
	if err != nil {
		return err
	}

	// 2. Asegurar que sshd_config tiene el banner activado
	_, _ = ExecCmdRun("sed", "-i", "/^Banner/d", "/etc/ssh/sshd_config")
	_, _ = ExecCmdRun("sh", "-c", "echo 'Banner /etc/sshd_banner' >> /etc/ssh/sshd_config")

	// 3. Reiniciar SSH
	ExecCmdRun("systemctl", "reload", "ssh")

	return nil
}
