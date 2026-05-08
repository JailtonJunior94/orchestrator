package triggers

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/JailtonJunior94/ai-spec-harness/internal/fs"
)

const triggersBase = "/base/triggers"

// goYAML, nodeYAML, pythonYAML sao representacoes minimas validas para os testes.
var goYAML = []byte(`
triggers:
  - ref: agent-governance/references/security.md
    patterns: ["os/exec", "auth", "password"]
  - ref: agent-governance/references/error-handling.md
    patterns: ["panic(", "recover(", "errors.New"]
  - ref: agent-governance/references/ddd.md
    patterns: ["internal/domain"]
`)

var nodeYAML = []byte(`
triggers:
  - ref: agent-governance/references/security.md
    patterns: ["eval(", "child_process", "JSON.parse", "crypto.createCipher", "req.body"]
  - ref: agent-governance/references/error-handling.md
    patterns: ["catch (e)", "unhandledRejection"]
  - ref: agent-governance/references/testing.md
    patterns: [".test.ts", ".spec.ts", ".test.js", ".spec.js", "coverage"]
`)

var pythonYAML = []byte(`
triggers:
  - ref: agent-governance/references/security.md
    patterns: ["exec(", "pickle.loads", "subprocess.shell=True", "eval(", "yaml.load"]
  - ref: agent-governance/references/error-handling.md
    patterns: ["except:", "except Exception"]
  - ref: agent-governance/references/testing.md
    patterns: ["test_", "pytest", "coverage"]
`)

func newFakeLoader() (Loader, *fs.FakeFileSystem) {
	ffs := fs.NewFakeFileSystem()
	_ = ffs.WriteFile(filepath.Join(triggersBase, "go.yaml"), goYAML)
	_ = ffs.WriteFile(filepath.Join(triggersBase, "node.yaml"), nodeYAML)
	_ = ffs.WriteFile(filepath.Join(triggersBase, "python.yaml"), pythonYAML)
	return NewLoader(ffs, triggersBase), ffs
}

// TestLoad_KnownLanguages verifica os criterios de pronto: go>=3, node>=5, python>=5.
func TestLoad_KnownLanguages(t *testing.T) {
	cases := []struct {
		lang    string
		minRefs int
	}{
		{"go", 3},
		{"node", 3},
		{"python", 3},
	}

	loader, _ := newFakeLoader()
	for _, tc := range cases {
		t.Run(tc.lang, func(t *testing.T) {
			triggers, err := loader.Load(tc.lang)
			if err != nil {
				t.Fatalf("Load(%q) erro inesperado: %v", tc.lang, err)
			}
			if len(triggers) < tc.minRefs {
				t.Errorf("Load(%q) retornou %d triggers, esperado >= %d", tc.lang, len(triggers), tc.minRefs)
			}
			for _, tr := range triggers {
				if tr.Ref == "" {
					t.Errorf("trigger sem ref em lang=%q", tc.lang)
				}
				if len(tr.Patterns) == 0 {
					t.Errorf("trigger %q sem patterns em lang=%q", tr.Ref, tc.lang)
				}
			}
		})
	}
}

// TestLoad_NodePatterns verifica que node.yaml contem os 5 padroes obrigatorios.
func TestLoad_NodePatterns(t *testing.T) {
	required := []string{"eval(", "child_process", "JSON.parse", "crypto.createCipher", "req.body"}
	loader, _ := newFakeLoader()
	triggers, err := loader.Load("node")
	if err != nil {
		t.Fatalf("Load(node) erro: %v", err)
	}
	all := collectPatterns(triggers)
	for _, p := range required {
		found := false
		for _, ap := range all {
			if ap == p {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("node.yaml nao contem padrao obrigatorio: %q", p)
		}
	}
}

// TestLoad_PythonPatterns verifica que python.yaml contem os 5 padroes obrigatorios.
func TestLoad_PythonPatterns(t *testing.T) {
	required := []string{"exec(", "pickle.loads", "subprocess.shell=True", "eval(", "yaml.load"}
	loader, _ := newFakeLoader()
	triggers, err := loader.Load("python")
	if err != nil {
		t.Fatalf("Load(python) erro: %v", err)
	}
	all := collectPatterns(triggers)
	for _, p := range required {
		found := false
		for _, ap := range all {
			if ap == p {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("python.yaml nao contem padrao obrigatorio: %q", p)
		}
	}
}

// TestLoad_Fallback verifica que linguagens desconhecidas e string vazia usam go.yaml.
func TestLoad_Fallback(t *testing.T) {
	cases := []string{"rust", "java", "", "cpp", "kotlin"}
	loader, _ := newFakeLoader()
	goTriggers, _ := loader.Load("go")

	for _, lang := range cases {
		t.Run(lang, func(t *testing.T) {
			got, err := loader.Load(lang)
			if err != nil {
				t.Fatalf("Load(%q) nao deve retornar erro no fallback: %v", lang, err)
			}
			if len(got) != len(goTriggers) {
				t.Errorf("Load(%q) retornou %d triggers, fallback go esperava %d", lang, len(got), len(goTriggers))
			}
		})
	}
}

// TestLoad_MalformedYAML verifica que YAML invalido retorna erro descritivo.
func TestLoad_MalformedYAML(t *testing.T) {
	ffs := fs.NewFakeFileSystem()
	_ = ffs.WriteFile(filepath.Join(triggersBase, "go.yaml"), []byte("triggers: [invalid: yaml: :::"))
	loader := NewLoader(ffs, triggersBase)

	_, err := loader.Load("go")
	if err == nil {
		t.Fatal("esperava erro para YAML malformado, got nil")
	}
	if !strings.Contains(err.Error(), "parsear") && !strings.Contains(err.Error(), "go") {
		t.Errorf("mensagem de erro deveria mencionar 'parsear' ou 'go', got: %q", err.Error())
	}
}

// TestLoad_MissingFile verifica que arquivo ausente retorna erro descritivo.
func TestLoad_MissingFile(t *testing.T) {
	ffs := fs.NewFakeFileSystem()
	loader := NewLoader(ffs, triggersBase)

	_, err := loader.Load("go")
	if err == nil {
		t.Fatal("esperava erro para arquivo ausente, got nil")
	}
}

// TestDetectLang verifica deteccao de linguagem por extensao dominante.
func TestDetectLang(t *testing.T) {
	cases := []struct {
		name  string
		files []string
		want  string
	}{
		{"go_majority", []string{"main.go", "service.go", "handler.go"}, "go"},
		{"node_majority", []string{"app.ts", "index.tsx", "utils.js"}, "node"},
		{"python_majority", []string{"main.py", "service.py", "test_service.py"}, "python"},
		{"mixed_go_dominant", []string{"main.go", "app.ts", "extra.go"}, "go"},
		{"empty", []string{}, ""},
		{"no_known_ext", []string{"README.md", "config.yaml"}, ""},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := DetectLang(tc.files)
			if got != tc.want {
				t.Errorf("DetectLang(%v) = %q, want %q", tc.files, got, tc.want)
			}
		})
	}
}

func collectPatterns(triggers []Trigger) []string {
	var out []string
	for _, t := range triggers {
		out = append(out, t.Patterns...)
	}
	return out
}
