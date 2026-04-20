package manifest

import (
	"encoding/json"
	"testing"
)

// FuzzParseManifest verifica que inputs malformados nao causam panic no parsing de manifesto.
// O manifesto e armazenado como JSON — inputs binarios, Unicode e estruturas invalidas
// devem ser tratados graciosamente (retornar erro, nunca panic).
func FuzzParseManifest(f *testing.F) {
	// Corpus: JSON valido do manifesto
	f.Add([]byte(`{
		"version": "0.9.0",
		"source_dir": "/home/user/.agents",
		"link_mode": "symlink",
		"tools": ["claude"],
		"langs": ["go"],
		"skills": ["agent-governance", "go-implementation"]
	}`))

	// Corpus: JSON vazio
	f.Add([]byte(`{}`))

	// Corpus: campos extras inesperados
	f.Add([]byte(`{"version":"1.0","unknown_field":true,"nested":{"a":1}}`))

	// Corpus: array no lugar de objeto
	f.Add([]byte(`[]`))

	// Corpus: string simples
	f.Add([]byte(`"apenas uma string"`))

	// Corpus: null
	f.Add([]byte(`null`))

	// Corpus: unicode e caracteres especiais
	f.Add([]byte(`{"version":"1.0\u0000\u0001","source_dir":"\xff\xfe"}`))

	// Corpus: JSON extremamente aninhado
	f.Add([]byte(`{"checksums":{"k1":"v1","k2":"v2","k3":"v3"}}`))

	f.Fuzz(func(t *testing.T, data []byte) {
		var m Manifest
		// Apenas verificar que nao causa panic — erro e aceitavel
		_ = json.Unmarshal(data, &m)
	})
}
