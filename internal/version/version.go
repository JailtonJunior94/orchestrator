package version

import (
	"os"
	"path/filepath"
	"strings"
)

// Set via ldflags at build time.
var (
	Version = "dev"
	Commit  = "none"
	Date    = "unknown"
)

// ReadVersionFile le o arquivo VERSION de um diretorio e retorna a versao.
// Retorna "unknown" se o arquivo nao existir ou nao puder ser lido.
func ReadVersionFile(dir string) string {
	data, err := os.ReadFile(filepath.Join(dir, "VERSION"))
	if err != nil {
		return "unknown"
	}
	return strings.TrimSpace(string(data))
}

// Resolve retorna a versao do binario. Prioridade:
//  1. Versao injetada via ldflags (releases via GoReleaser)
//  2. Arquivo VERSION no diretorio informado com sufixo "-dev" (builds locais)
//  3. "dev" como fallback final
func Resolve(dir string) string {
	if Version != "dev" {
		return Version
	}
	if v := ReadVersionFile(dir); v != "unknown" {
		return v + "-dev"
	}
	return "dev"
}
