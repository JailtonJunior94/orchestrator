package aispecharness

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/JailtonJunior94/ai-spec-harness/internal/taskloop"
)

func TestTaskLoopOpenCodeLegacyReturnsACPOnlyError(t *testing.T) {
	cmd := newTaskLoopCmd()
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"--tool", "opencode", t.TempDir()})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("esperado erro para --tool opencode sem --runtime acp")
	}
	var acpOnly *taskloop.ACPOnlyToolError
	if !errors.As(err, &acpOnly) {
		t.Fatalf("erro deve ser *taskloop.ACPOnlyToolError, obteve %T: %v", err, err)
	}
	if !strings.Contains(err.Error(), "--runtime acp") {
		t.Errorf("mensagem deve orientar --runtime acp, obteve: %v", err)
	}
}

func TestTaskLoopHelpDescribesOpenCodeACPOnly(t *testing.T) {
	f := newTaskLoopCmd().Flags().Lookup("tool")
	if f == nil {
		t.Fatal("flag --tool nao registrada")
	}
	if !strings.Contains(f.Usage, "opencode exige --runtime acp") {
		t.Errorf("usage de --tool deve alinhar com o comportamento, obteve: %q", f.Usage)
	}
}
