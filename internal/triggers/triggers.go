// Package triggers carrega gatilhos de revisao por linguagem a partir de YAMLs externos.
package triggers

import (
	"fmt"
	"path/filepath"

	"gopkg.in/yaml.v3"

	"github.com/JailtonJunior94/ai-spec-harness/internal/fs"
)

// Trigger associa uma referencia de governanca aos padroes que a disparam no diff.
type Trigger struct {
	Ref      string   `yaml:"ref"`
	Patterns []string `yaml:"patterns"`
}

// Loader carrega gatilhos de revisao por linguagem.
type Loader interface {
	Load(lang string) ([]Trigger, error)
}

type embeddedLoader struct {
	fs      fs.FileSystem
	baseDir string
}

// NewLoader retorna um Loader que le YAMLs a partir de baseDir usando fsys.
// Linguagens suportadas: "go", "node", "python". Qualquer outra (inclusive "")
// usa fallback para "go".
func NewLoader(fsys fs.FileSystem, baseDir string) Loader {
	return &embeddedLoader{fs: fsys, baseDir: baseDir}
}

// Load retorna os gatilhos para a linguagem indicada.
// Linguagem desconhecida ou vazia faz fallback para go.yaml.
func (l *embeddedLoader) Load(lang string) ([]Trigger, error) {
	normalized := normalizeLang(lang)
	path := filepath.Join(l.baseDir, normalized+".yaml")

	data, err := l.fs.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("carregar gatilhos %q: %w", normalized, err)
	}

	var doc struct {
		Triggers []Trigger `yaml:"triggers"`
	}
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("parsear gatilhos %q: %w", normalized, err)
	}
	return doc.Triggers, nil
}

// normalizeLang mapeia a linguagem para o nome de arquivo suportado; fallback go.
func normalizeLang(lang string) string {
	switch lang {
	case "go", "node", "python":
		return lang
	default:
		return "go"
	}
}

// DetectLang retorna a linguagem majoritaria de um conjunto de caminhos de arquivo
// com base na extensao dominante. Retorna "" quando nenhuma extensao conhecida domina.
func DetectLang(files []string) string {
	counts := make(map[string]int, 3)
	for _, f := range files {
		switch filepath.Ext(f) {
		case ".go":
			counts["go"]++
		case ".ts", ".tsx", ".js", ".jsx", ".mjs", ".cjs":
			counts["node"]++
		case ".py":
			counts["python"]++
		}
	}

	best, max := "", 0
	for lang, n := range counts {
		if n > max {
			best, max = lang, n
		}
	}
	return best
}
