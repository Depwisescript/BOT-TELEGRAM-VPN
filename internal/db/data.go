package db

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
)

// ConfigData representa el archivo bot_data.json
type ConfigData struct {
	Admins           map[string]AdminInfo `json:"admins"`
	ExtraInfo        string               `json:"extra_info"`
	UserHistory      []int64              `json:"user_history"`
	PublicAccess     bool                 `json:"public_access"`
	SSHOwners        map[string]string    `json:"ssh_owners"`
	SSHTimeUsers     map[string]string    `json:"ssh_time_users"` // user -> expire date
	CloudflareDomain string               `json:"cloudflare_domain"`
	CloudfrontDomain string               `json:"cloudfront_domain"`
	ProxyDT          ProxyDTConfig        `json:"proxydt"`
	SlowDNS          SlowDNSConfig        `json:"slowdns"`
	Zivpn            bool                 `json:"zivpn"`
	ZivpnUsers       map[string]string    `json:"zivpn_users"`  // password -> expire
	ZivpnOwners      map[string]string    `json:"zivpn_owners"` // password -> owner chat ID
	BadVPN           bool                 `json:"badvpn"`
	UDPCustom        bool                 `json:"udp_custom"`
	Falcon           string               `json:"falcon"`     // Port as string for compatibility
	Dropbear         string               `json:"dropbear"`   // Port as string for compatibility
	SSLTunnel        string               `json:"ssl_tunnel"` // Port as string for compatibility
	SSHBanner        string               `json:"ssh_banner"`
	SSHLastActive    map[string]string    `json:"ssh_last_active"` // user -> last active RFC3339
}

type AdminInfo struct {
	Alias string `json:"alias"`
}

type ProxyDTConfig struct {
	Ports map[string]string `json:"ports"`
	Token string            `json:"token"`
}

type SlowDNSConfig struct {
	NS   string `json:"ns"`
	Port string `json:"port"`
	Key  string `json:"key"`
}

var (
	mutex sync.Mutex
	dir   = "/opt/depwise_bot"
)

// SetDir permite cambiar el directorio del DB (util para testing local)
func SetDir(newDir string) {
	dir = newDir
}

// GetDataPath retorna la ruta absoluta del bot_data.json
func GetDataPath() string {
	return filepath.Join(dir, "bot_data.json")
}

// Load lee el archivo bot_data.json o retorna una data por defecto
func Load() (*ConfigData, error) {
	mutex.Lock()
	defer mutex.Unlock()
	return loadUnlocked()
}

func loadUnlocked() (*ConfigData, error) {
	path := GetDataPath()
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return defaultData(), nil
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		return defaultData(), err
	}

	var data ConfigData
	err = json.Unmarshal(raw, &data)
	if err != nil {
		return defaultData(), err // Archivo corrupto, reset fallback (en un caso real, haríamos backup)
	}

	// Inicializaciones de seguridad para mapas nulos
	if data.Admins == nil {
		data.Admins = make(map[string]AdminInfo)
	}
	if data.SSHOwners == nil {
		data.SSHOwners = make(map[string]string)
	}
	if data.SSHTimeUsers == nil {
		data.SSHTimeUsers = make(map[string]string)
	}
	if data.ZivpnUsers == nil {
		data.ZivpnUsers = make(map[string]string)
	}
	if data.ZivpnOwners == nil {
		data.ZivpnOwners = make(map[string]string)
	}
	if data.ProxyDT.Ports == nil {
		data.ProxyDT.Ports = make(map[string]string)
	}
	if data.SSHLastActive == nil {
		data.SSHLastActive = make(map[string]string)
	}

	return &data, nil
}

// Save guarda la memoria en el archivo bot_data.json
func Save(data *ConfigData) error {
	mutex.Lock()
	defer mutex.Unlock()
	return saveUnlocked(data)
}

// Update encierra una operacion de lectura y escritura en un solo bloqueo concurrente
func Update(fn func(*ConfigData) error) error {
	mutex.Lock()
	defer mutex.Unlock()

	data, err := loadUnlocked()
	if err != nil {
		return err
	}

	if err := fn(data); err != nil {
		return err
	}

	return saveUnlocked(data)
}

func saveUnlocked(data *ConfigData) error {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	raw, err := json.MarshalIndent(data, "", "    ")
	if err != nil {
		return err
	}

	return os.WriteFile(GetDataPath(), raw, 0644)
}

func defaultData() *ConfigData {
	return &ConfigData{
		Admins:       make(map[string]AdminInfo),
		ExtraInfo:    "Puertos: 22, 80, 443",
		PublicAccess: true,
		SSHOwners:    make(map[string]string),
		SSHTimeUsers: make(map[string]string),
		ZivpnUsers:   make(map[string]string),
		ZivpnOwners:  make(map[string]string),
		ProxyDT: ProxyDTConfig{
			Ports: make(map[string]string),
			Token: "dummy",
		},
		SSHLastActive: make(map[string]string),
	}
}
