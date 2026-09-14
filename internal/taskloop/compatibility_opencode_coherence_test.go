package taskloop

import (
	"slices"
	"testing"

	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/specs"
)

func TestOpenCodeCompatibilityAcceptsModelWithResolvedWindow(t *testing.T) {
	const model = "anthropic/claude-sonnet-4-5"

	window := specs.NewCatalog().OpenCode().ResolveWindow(model)
	if window.MaxTokens != 200_000 {
		t.Fatalf("janela resolvida = %d, quero 200000 (sonda invalida)", window.MaxTokens)
	}

	if err := NewCompatibilityTable().ValidateCombination(openCodeToolID, model); err != nil {
		t.Errorf("ValidateCombination rejeitou %q com janela resolvida %d: %v", model, window.MaxTokens, err)
	}
}

func TestOpenCodeCompatibilityRejectsModelWithoutResolvedWindow(t *testing.T) {
	const model = "anthropic/modelo-inexistente"

	if _, matched := specs.NewCatalog().MatchOpenCodeModel(model); matched {
		t.Fatalf("modelo %q nao deveria casar na tabela de janela (sonda invalida)", model)
	}
	if err := NewCompatibilityTable().ValidateCombination(openCodeToolID, model); err == nil {
		t.Errorf("ValidateCombination aceitou %q sem janela conhecida", model)
	}
}

func TestOpenCodeTablesAreCoherent(t *testing.T) {
	windowPrefixes := specs.OpenCodeModelPrefixes()
	compatModels := NewCompatibilityTable().Models(openCodeToolID)

	for _, prefix := range windowPrefixes {
		if !slices.Contains(compatModels, prefix) {
			t.Errorf("prefixo %q da tabela de janela ausente na tabela de compatibilidade", prefix)
		}
		if _, matched := specs.NewCatalog().MatchOpenCodeModel(prefix); !matched {
			t.Errorf("prefixo %q nao casa na propria tabela de janela", prefix)
		}
		if !NewCompatibilityTable().IsSupported(openCodeToolID, prefix) {
			t.Errorf("prefixo %q da tabela de janela rejeitado por IsSupported", prefix)
		}
	}
	for _, model := range compatModels {
		if !slices.Contains(windowPrefixes, model) {
			t.Errorf("modelo %q da tabela de compatibilidade sem entrada na tabela de janela", model)
		}
	}
}
