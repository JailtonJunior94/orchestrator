package wrapper_test

import (
	"errors"
	"testing"

	"github.com/JailtonJunior94/ai-spec-harness/internal/fs"
	"github.com/JailtonJunior94/ai-spec-harness/internal/skills"
	"github.com/JailtonJunior94/ai-spec-harness/internal/wrapper"
)

func TestExecute_RemovedAgentGemini(t *testing.T) {
	t.Parallel()
	_, err := wrapper.NewExecutor().Execute("gemini", "go-implementation", "/project", nil, fs.NewFakeFileSystem())
	if err == nil {
		t.Fatal("esperava erro para agente removido")
	}
	var removed *skills.RemovedAgentError
	if !errors.As(err, &removed) {
		t.Fatalf("RF-03: erro deve ser *skills.RemovedAgentError, got %T: %v", err, err)
	}
	if removed.Agent != "gemini" {
		t.Errorf("agent: got %q, want gemini", removed.Agent)
	}
}

func TestExecute_GenericInvalidToolStaysGeneric(t *testing.T) {
	t.Parallel()
	_, err := wrapper.NewExecutor().Execute("nao-existe", "go-implementation", "/project", nil, fs.NewFakeFileSystem())
	if err == nil {
		t.Fatal("esperava erro para ferramenta invalida")
	}
	var removed *skills.RemovedAgentError
	if errors.As(err, &removed) {
		t.Fatal("valor generico invalido nao deve virar RemovedAgentError")
	}
}
