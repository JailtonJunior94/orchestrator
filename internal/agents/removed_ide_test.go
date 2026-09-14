package agents

import (
	"errors"
	"testing"

	"github.com/JailtonJunior94/ai-spec-harness/internal/skills"
)

func TestValidateAgentFrontmatterRemovedIDEIsTyped(t *testing.T) {
	content := []byte("---\nname: foo\ndescription: agente de teste\nversion: 1.0.0\nruntime:\n  ide: gemini\n---\n\ncorpo\n")

	_, err := NewCatalog().ValidateAgentFrontmatter(content, "foo")
	if err == nil {
		t.Fatal("esperado erro para runtime.ide gemini")
	}
	var removed *skills.RemovedAgentError
	if !errors.As(err, &removed) {
		t.Fatalf("erro deve ser *skills.RemovedAgentError, obteve %T: %v", err, err)
	}
	if removed.Agent != "gemini" {
		t.Errorf("RemovedAgentError.Agent = %q; want gemini", removed.Agent)
	}
}

func TestValidateAgentFrontmatterUnknownIDEUsesErrIDEUnsupported(t *testing.T) {
	content := []byte("---\nname: foo\ndescription: agente de teste\nversion: 1.0.0\nruntime:\n  ide: ide-inexistente\n---\n\ncorpo\n")

	_, err := NewCatalog().ValidateAgentFrontmatter(content, "foo")
	if err == nil {
		t.Fatal("esperado erro para runtime.ide desconhecido")
	}
	if !errors.Is(err, ErrIDEUnsupported) {
		t.Fatalf("erro deve satisfazer errors.Is(err, ErrIDEUnsupported), obteve %T: %v", err, err)
	}
	if !errors.Is(err, ErrFrontmatterInvalid) {
		t.Errorf("erro deve preservar ErrFrontmatterInvalid, obteve: %v", err)
	}
	var removed *skills.RemovedAgentError
	if errors.As(err, &removed) {
		t.Error("ide desconhecido generico nao deve virar RemovedAgentError")
	}
}

func TestValidateAgentFrontmatterSupportedIDEStillValid(t *testing.T) {
	for _, ide := range []string{"claude", "codex", "copilot", "opencode"} {
		content := []byte("---\nname: foo\ndescription: agente de teste\nversion: 1.0.0\nruntime:\n  ide: " + ide + "\n---\n\ncorpo\n")
		if _, err := NewCatalog().ValidateAgentFrontmatter(content, "foo"); err != nil {
			t.Errorf("ide %q deveria ser valido, obteve: %v", ide, err)
		}
	}
}
