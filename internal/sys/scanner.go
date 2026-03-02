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
		if findGoBinary(name) == "" {
			// Install it and capture output for debugging if needed
			out, err := exec.Command("go", "install", "-v", pkg).CombinedOutput()
			if err != nil {
				return fmt.Errorf("error instalando %s: %v\nOutput: %s", name, err, string(out))
			}
		}
	}

	return nil
}

func findGoBinary(name string) string {
	// 1. Try PATH
	if p, err := exec.LookPath(name); err == nil {
		return p
	}

	// 2. Try to get GOPATH bin
	if out, err := exec.Command("go", "env", "GOPATH").Output(); err == nil {
		gopath := strings.TrimSpace(string(out))
		if gopath != "" {
			p := fmt.Sprintf("%s/bin/%s", gopath, name)
			if err := exec.Command("ls", p).Run(); err == nil {
				return p
			}
		}
	}

	// 3. Try common VPS Go bin paths
	paths := []string{
		"/usr/local/bin/" + name,
		"/usr/bin/" + name,
		"/bin/" + name,
		"/root/go/bin/" + name,
		"/usr/local/go/bin/" + name,
		"/home/ubuntu/go/bin/" + name,
		"/home/debian/go/bin/" + name,
	}

	for _, p := range paths {
		if err := exec.Command("ls", p).Run(); err == nil {
			return p
		}
	}
	return ""
}

// RunScanner runs assetfinder and httpx on a domain
func RunScanner(domain string) (string, error) {
	// Resolve paths
	assetPath := findGoBinary("assetfinder")
	if assetPath == "" {
		return "", fmt.Errorf("assetfinder no encontrado. Intenta ejecutar de nuevo para instalarlo.")
	}

	httpxPath := findGoBinary("httpx")
	if httpxPath == "" {
		return "", fmt.Errorf("httpx no encontrado. Intenta ejecutar de nuevo para instalarlo.")
	}

	// 1. Assetfinder
	cmdAsset := exec.Command(assetPath, "--subs-only", domain)
	outAsset, err := cmdAsset.Output()
	if err != nil {
		return "", fmt.Errorf("error en assetfinder (%s): %v", assetPath, err)
	}

	subs := strings.TrimSpace(string(outAsset))
	if subs == "" {
		return "❌ No se encontraron subdominios.", nil
	}

	// 2. HTTPX (using stdin)
	cmdHttpx := exec.Command(httpxPath, "-silent", "-status-code", "-title", "-tech-detect", "-ip")
	cmdHttpx.Stdin = strings.NewReader(subs)
	outHttpx, err := cmdHttpx.Output()
	if err != nil {
		return "", fmt.Errorf("error en httpx (%s): %v", httpxPath, err)
	}

	result := string(outHttpx)
	if result == "" {
		return "🔍 Subdominios encontrados, pero ninguno respondió a HTTP/HTTPS.", nil
	}

	return result, nil
}
