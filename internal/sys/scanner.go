package sys

import (
	"fmt"
	"os/exec"
	"strings"
)

// EnsureScannerDeps checks and installs assetfinder and httpx if missing
func EnsureScannerDeps() error {
	// Check Go
	if _, err := exec.LookPath("go"); err != nil {
		return fmt.Errorf("Go no está instalado. Por favor instala Go primero")
	}

	// Check assetfinder
	if _, err := exec.LookPath("assetfinder"); err != nil {
		// Install assetfinder
		_ = exec.Command("go", "install", "github.com/tomnomnom/assetfinder@latest").Run()
		// Add to path if not there (usually in /root/go/bin)
	}

	// Check httpx
	if _, err := exec.LookPath("httpx"); err != nil {
		// Install httpx
		_ = exec.Command("go", "install", "-v", "github.com/projectdiscovery/httpx/cmd/httpx@latest").Run()
	}

	return nil
}

// RunScanner runs assetfinder and httpx on a domain
func RunScanner(domain string) (string, error) {
	// 1. Assetfinder
	cmdAsset := exec.Command("assetfinder", "--subs-only", domain)
	outAsset, err := cmdAsset.Output()
	if err != nil {
		return "", fmt.Errorf("error en assetfinder: %v", err)
	}

	subs := strings.TrimSpace(string(outAsset))
	if subs == "" {
		return "❌ No se encontraron subdominios.", nil
	}

	// 2. HTTPX (using stdin)
	cmdHttpx := exec.Command("httpx", "-silent", "-status-code", "-title", "-tech-detect", "-ip")
	cmdHttpx.Stdin = strings.NewReader(subs)
	outHttpx, err := cmdHttpx.Output()
	if err != nil {
		return "", fmt.Errorf("error en httpx: %v", err)
	}

	result := string(outHttpx)
	if result == "" {
		return "🔍 Subdominios encontrados, pero ninguno respondió a HTTP/HTTPS.", nil
	}

	return result, nil
}
