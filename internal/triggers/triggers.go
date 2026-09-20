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

var _ Loader = (*embeddedLoader)(nil)

func NewLoader(fsys fs.FileSystem, baseDir string) Loader {
	return &embeddedLoader{fs: fsys, baseDir: baseDir}
}

func (l *embeddedLoader) Load(lang string) ([]Trigger, error) {
	normalized, ok := l.normalizeLang(lang)
	if !ok {
		return nil, fmt.Errorf("carregar gatilhos: linguagem desconhecida %q sem trigger.yaml correspondente", lang)
	}
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

func (l *embeddedLoader) normalizeLang(lang string) (string, bool) {
	switch lang {
	case "go", "node", "python", "dotnet", "java":
		return lang, true
	case "":
		return "go", true
	default:
		return "", false
	}
}

// Detector detecta linguagem a partir de caminhos de arquivo.
type Detector struct{}

// NewDetector cria um Detector stateless.
func NewDetector() *Detector {
	return &Detector{}
}

// DetectLang retorna a linguagem majoritaria de um conjunto de caminhos de arquivo
// com base na extensao dominante. Retorna "" quando nenhuma extensao conhecida domina.
func (d *Detector) DetectLang(files []string) string {
	counts := make(map[string]int, 5)
	for _, f := range files {
		switch filepath.Ext(f) {
		case ".go":
			counts["go"]++
		case ".ts", ".tsx", ".js", ".jsx", ".mjs", ".cjs":
			counts["node"]++
		case ".py":
			counts["python"]++
		case ".cs":
			counts["dotnet"]++
		case ".java":
			counts["java"]++
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
