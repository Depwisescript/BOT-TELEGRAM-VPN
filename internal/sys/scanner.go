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

	tools := map[string]string{
		"assetfinder": "github.com/tomnomnom/assetfinder@latest",
		"httpx":       "github.com/projectdiscovery/httpx/cmd/httpx@latest",
	}

	for name, pkg := range tools {
		if !isToolAvailable(name) {
			// Try to find it in common go bin paths
			binPath := findGoBinary(name)
			if binPath != "" {
				// Link it to /usr/local/bin
				_ = exec.Command("ln", "-sf", binPath, "/usr/local/bin/"+name).Run()
			} else {
				// Install it
				_ = exec.Command("go", "install", "-v", pkg).Run()
				// Try finding again after install
				binPath = findGoBinary(name)
				if binPath != "" {
					_ = exec.Command("ln", "-sf", binPath, "/usr/local/bin/"+name).Run()
				}
			}
		}
	}

	return nil
}

func isToolAvailable(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}

func findGoBinary(name string) string {
	paths := []string{
		"/root/go/bin/" + name,
		"/usr/local/go/bin/" + name,
		"~/go/bin/" + name,
	}

	// Expand ~ manual
	for _, p := range paths {
		checkPath := p
		if strings.HasPrefix(p, "~") {
			// Placeholder for home if needed, but usually bot runs as root or specific user
			// For simplicity in VPS context, root is standard
		}
		if _, err := exec.LookPath(checkPath); err == nil {
			return checkPath
		}
		// Direct check if LookPath fails due to not being in PATH
		if err := exec.Command("ls", checkPath).Run(); err == nil {
			return checkPath
		}
	}
	return ""
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
