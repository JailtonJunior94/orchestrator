package aispecharness

import (
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/JailtonJunior94/ai-spec-harness/internal/skills"
)

func runTaskLoopArgs(t *testing.T, args ...string) error {
	t.Helper()
	cmd := newTaskLoopCmd()
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)
	cmd.SetArgs(args)
	return cmd.Execute()
}

func TestTaskLoopFallbackToolRemovedAgentTypedError(t *testing.T) {
	err := runTaskLoopArgs(t, "--tool", "claude", "--fallback-tool", "gemini", t.TempDir())
	if err == nil {
		t.Fatal("esperava erro para --fallback-tool gemini, obteve nil")
	}

	var removedErr *skills.RemovedAgentError
	if !errors.As(err, &removedErr) {
		t.Fatalf("erro deveria ser RemovedAgentError (errors.As), obteve: %v", err)
	}
	if removedErr.Agent != "gemini" {
		t.Errorf("RemovedAgentError.Agent = %q, want gemini", removedErr.Agent)
	}
	if !strings.Contains(err.Error(), "migracao-legacy-acp.md") {
		t.Errorf("mensagem deveria citar o guia de migracao, got: %q", err.Error())
	}
}

func TestTaskLoopFallbackToolUnknownIsNotRemovedAgent(t *testing.T) {
	err := runTaskLoopArgs(t, "--tool", "claude", "--fallback-tool", "not-a-real-tool", t.TempDir())
	if err == nil {
		t.Fatal("esperava erro para --fallback-tool desconhecido, obteve nil")
	}
	var removedErr *skills.RemovedAgentError
	if errors.As(err, &removedErr) {
		t.Error("ferramenta desconhecida generica nao deveria ser RemovedAgentError")
	}
}
