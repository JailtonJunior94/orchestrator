package embedded

import (
	"embed"
	"io/fs"
	"os"
	"path/filepath"
)

//go:embed all:assets
var Assets embed.FS

// ExtractToTempDir extrai os assets embutidos para um diretorio temporario
// e retorna o caminho e uma funcao de limpeza.
func ExtractToTempDir() (string, func(), error) {
	dir, err := os.MkdirTemp("", "ai-spec-harness-embedded-*")
	if err != nil {
		return "", nil, err
	}
	cleanup := func() { os.RemoveAll(dir) }

	if err := copyFS(Assets, "assets", dir); err != nil {
		cleanup()
		return "", nil, err
	}
	return dir, cleanup, nil
}

// copyFS copia o conteudo de um embed.FS para um diretorio de destino no OS.
func copyFS(fsys embed.FS, root, dst string) error {
	return fs.WalkDir(fsys, root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}

		target := filepath.Join(dst, rel)

		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}

		data, err := fsys.ReadFile(path)
		if err != nil {
			return err
		}

		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}

		info, err := d.Info()
		if err != nil {
			return err
		}

		return os.WriteFile(target, data, info.Mode())
	})
}
