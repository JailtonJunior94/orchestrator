package memory_test

import (
	"testing"

	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/memory"
	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/specs"
)

// TestWindowPolicy_Standard verifica que WindowStandard retorna limites F1.
func TestWindowPolicy_Standard(t *testing.T) {
	t.Parallel()

	p := memory.WindowPolicy{}
	base := memory.Limits{} // zero-value → defaults F1 via applyDefaultsToLimits

	got := p.LimitsFor(specs.WindowStandard, base)

	if got.WorkflowLines != memory.DefaultWorkflowLineLimit {
		t.Errorf("WorkflowLines = %d; want %d (F1)", got.WorkflowLines, memory.DefaultWorkflowLineLimit)
	}
	if got.WorkflowBytes != memory.DefaultWorkflowByteLimit {
		t.Errorf("WorkflowBytes = %d; want %d (F1)", got.WorkflowBytes, memory.DefaultWorkflowByteLimit)
	}
	if got.TaskLines != memory.DefaultTaskLineLimit {
		t.Errorf("TaskLines = %d; want %d (F1)", got.TaskLines, memory.DefaultTaskLineLimit)
	}
	if got.TaskBytes != memory.DefaultTaskByteLimit {
		t.Errorf("TaskBytes = %d; want %d (F1)", got.TaskBytes, memory.DefaultTaskByteLimit)
	}
}

// TestWindowPolicy_Large verifica que WindowLarge retorna limites ampliados.
func TestWindowPolicy_Large(t *testing.T) {
	t.Parallel()

	p := memory.WindowPolicy{}
	base := memory.Limits{} // zero-value; deve ser substituído pelos limites ampliados

	got := p.LimitsFor(specs.WindowLarge, base)

	if got.WorkflowLines != memory.LargeWorkflowLineLimit {
		t.Errorf("WorkflowLines = %d; want %d (Large)", got.WorkflowLines, memory.LargeWorkflowLineLimit)
	}
	if got.WorkflowBytes != memory.LargeWorkflowByteLimit {
		t.Errorf("WorkflowBytes = %d; want %d (Large)", got.WorkflowBytes, memory.LargeWorkflowByteLimit)
	}
	if got.TaskLines != memory.LargeTaskLineLimit {
		t.Errorf("TaskLines = %d; want %d (Large)", got.TaskLines, memory.LargeTaskLineLimit)
	}
	if got.TaskBytes != memory.LargeTaskByteLimit {
		t.Errorf("TaskBytes = %d; want %d (Large)", got.TaskBytes, memory.LargeTaskByteLimit)
	}
}

// TestWindowPolicy_LargeExceedsStandard garante que limites Large > Standard.
func TestWindowPolicy_LargeExceedsStandard(t *testing.T) {
	t.Parallel()

	p := memory.WindowPolicy{}
	standard := p.LimitsFor(specs.WindowStandard, memory.Limits{})
	large := p.LimitsFor(specs.WindowLarge, memory.Limits{})

	if large.WorkflowLines <= standard.WorkflowLines {
		t.Errorf("Large.WorkflowLines (%d) deve exceder Standard (%d)", large.WorkflowLines, standard.WorkflowLines)
	}
	if large.WorkflowBytes <= standard.WorkflowBytes {
		t.Errorf("Large.WorkflowBytes (%d) deve exceder Standard (%d)", large.WorkflowBytes, standard.WorkflowBytes)
	}
	if large.TaskLines <= standard.TaskLines {
		t.Errorf("Large.TaskLines (%d) deve exceder Standard (%d)", large.TaskLines, standard.TaskLines)
	}
	if large.TaskBytes <= standard.TaskBytes {
		t.Errorf("Large.TaskBytes (%d) deve exceder Standard (%d)", large.TaskBytes, standard.TaskBytes)
	}
}

// TestWindowPolicy_ZeroValueIsStandard garante zero-value de WindowClass ⇒ WindowStandard ⇒ F1.
func TestWindowPolicy_ZeroValueIsStandard(t *testing.T) {
	t.Parallel()

	var class specs.WindowClass // zero-value = WindowStandard

	p := memory.WindowPolicy{}
	got := p.LimitsFor(class, memory.Limits{})
	f1 := p.LimitsFor(specs.WindowStandard, memory.Limits{})

	if got != f1 {
		t.Errorf("zero-value WindowClass deve produzir mesmos limites que WindowStandard; got=%+v want=%+v", got, f1)
	}
}

// TestWindowPolicy_TableDriven cobre todos os casos via table.
func TestWindowPolicy_TableDriven(t *testing.T) {
	t.Parallel()

	p := memory.WindowPolicy{}

	tests := []struct {
		name      string
		class     specs.WindowClass
		wantLines int // WorkflowLines esperado
	}{
		{"standard_zero", specs.WindowStandard, memory.DefaultWorkflowLineLimit},
		{"large", specs.WindowLarge, memory.LargeWorkflowLineLimit},
		{"zero_value", 0, memory.DefaultWorkflowLineLimit}, // 0 == WindowStandard
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := p.LimitsFor(tc.class, memory.Limits{})
			if got.WorkflowLines != tc.wantLines {
				t.Errorf("WorkflowLines = %d; want %d", got.WorkflowLines, tc.wantLines)
			}
		})
	}
}

// TestWindowPolicy_BasePreservedForStandard verifica que base customizado é aplicado para Standard.
func TestWindowPolicy_BasePreservedForStandard(t *testing.T) {
	t.Parallel()

	p := memory.WindowPolicy{}
	base := memory.Limits{
		WorkflowLines: 300, // custom override
		WorkflowBytes: 24 * 1024,
		TaskLines:     400,
		TaskBytes:     32 * 1024,
	}

	got := p.LimitsFor(specs.WindowStandard, base)

	// Standard deve retornar base customizado (sem substituição).
	if got.WorkflowLines != 300 {
		t.Errorf("WorkflowLines = %d; want 300 (base custom)", got.WorkflowLines)
	}
}

// TestWindowPolicy_LargePreservesExplicitBase verifica regressão: WindowLarge não pode
// descartar limites --memory-* explicitamente informados pelo usuário.
func TestWindowPolicy_LargePreservesExplicitBase(t *testing.T) {
	t.Parallel()

	p := memory.WindowPolicy{}
	base := memory.Limits{
		WorkflowLines: 123,
		WorkflowBytes: 456,
		TaskLines:     789,
		TaskBytes:     1024,
	}

	got := p.LimitsForWithOverride(specs.WindowLarge, base, true)

	if got != base {
		t.Fatalf("WindowLarge com override explicito deve preservar base; got=%+v want=%+v", got, base)
	}
}
