package taskloop

import (
	"errors"
	"fmt"
)

// ErrModeloIncompativel indica que a combinacao ferramenta+modelo nao e suportada.
var ErrModeloIncompativel = errors.New("modelo incompativel com a ferramenta")

// CompatibilityTable mapeia ferramentas aos modelos suportados.
// Tipo concreto (nao interface) — conforme ADR-004.
type CompatibilityTable struct {
	entries map[string][]string // tool -> []model
}

// NewCompatibilityTable cria a tabela com os modelos catalogados (RF-04 do PRD).
func NewCompatibilityTable() *CompatibilityTable {
	return &CompatibilityTable{
		entries: map[string][]string{
			"claude": {
				"claude-opus-4-7",
				"claude-opus-4-6",
				"claude-sonnet-4-6",
				"claude-haiku-4-5",
			},
			"codex": {
				"gpt-5.4",
				"gpt-5.4-mini",
				"gpt-5.3-codex",
				"gpt-5.3-codex-spark",
			},
			"gemini": {
				"auto",
				"pro",
				"flash",
				"flash-lite",
				"gemini-2.5-pro",
				"gemini-2.5-flash",
				"gemini-2.5-flash-lite",
				"gemini-3-pro-preview",
			},
			"copilot": {
				"claude-sonnet-4.5",
				"claude-sonnet-4.6",
				"gpt-5.4",
				"gpt-5.4-mini",
				"haiku-4.5",
				"auto",
			},
		},
	}
}

// IsSupported verifica se a combinacao tool+model e conhecida.
// Retorna true quando model e vazio (usa default da ferramenta).
// Retorna false para ferramenta desconhecida.
func (t *CompatibilityTable) IsSupported(tool, model string) bool {
	if model == "" {
		// model vazio e sempre valido; basta que a ferramenta exista na tabela
		_, known := t.entries[tool]
		return known
	}

	models, ok := t.entries[tool]
	if !ok {
		return false
	}

	for _, m := range models {
		if m == model {
			return true
		}
	}

	return false
}

// Models retorna a lista de modelos suportados para a ferramenta informada.
// Retorna nil para ferramenta desconhecida.
func (t *CompatibilityTable) Models(tool string) []string {
	return t.entries[tool]
}

// ValidateCombination verifica se a combinacao e suportada e retorna ErrModeloIncompativel
// com contexto quando nao for.
func (t *CompatibilityTable) ValidateCombination(tool, model string) error {
	if t.IsSupported(tool, model) {
		return nil
	}

	if model == "" {
		return fmt.Errorf("%w: ferramenta %q nao reconhecida", ErrModeloIncompativel, tool)
	}

	return fmt.Errorf("%w: %q nao suportado pela ferramenta %q", ErrModeloIncompativel, model, tool)
}
