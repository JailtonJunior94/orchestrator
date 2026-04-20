package config

import (
	"encoding/json"
	"testing"
)

// configJSON e um struct intermediario para testar parsing de configuracao
// que poderia ser passada via arquivo ou stdin.
type configJSON struct {
	ProjectDir   string   `json:"project_dir"`
	SourceDir    string   `json:"source_dir"`
	Tools        []string `json:"tools"`
	Langs        []string `json:"langs"`
	LinkMode     string   `json:"link_mode"`
	DryRun       bool     `json:"dry_run"`
	CodexProfile string   `json:"codex_profile"`
	FocusPaths   []string `json:"focus_paths"`
}

// FuzzParseConfig verifica que inputs malformados nao causam panic no parsing
// de configuracao de instalacao/upgrade. Qualquer input deve retornar erro ou
// sucesso sem panic.
func FuzzParseConfig(f *testing.F) {
	// Corpus: JSON valido de InstallOptions
	f.Add([]byte(`{
		"project_dir": "/tmp/myproject",
		"source_dir": "/home/user/.agents",
		"tools": ["claude", "gemini"],
		"langs": ["go"],
		"link_mode": "symlink",
		"dry_run": false,
		"codex_profile": "full"
	}`))

	// Corpus: configuracao minima
	f.Add([]byte(`{"project_dir":"/tmp/proj","tools":["claude"]}`))

	// Corpus: JSON vazio
	f.Add([]byte(`{}`))

	// Corpus: array no lugar de objeto
	f.Add([]byte(`["claude","gemini"]`))

	// Corpus: string arbitraria
	f.Add([]byte(`"nao e json de config"`))

	// Corpus: null
	f.Add([]byte(`null`))

	// Corpus: ferramenta invalida
	f.Add([]byte(`{"tools":["ferramenta_invalida_xpto"]}`))

	// Corpus: path com null byte
	f.Add([]byte(`{"project_dir":"/tmp/proj\u0000etc"}`))

	// Corpus: campo desconhecido
	f.Add([]byte(`{"project_dir":"/tmp","unknown":true,"injection":"'; DROP TABLE users; --"}`))

	f.Fuzz(func(t *testing.T, data []byte) {
		var cfg configJSON
		// Parsing nao deve causar panic
		_ = json.Unmarshal(data, &cfg)
	})
}
