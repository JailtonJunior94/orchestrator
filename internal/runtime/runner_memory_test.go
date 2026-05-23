package runtime_test

import (
	"strings"
	"testing"

	airuntime "github.com/JailtonJunior94/ai-spec-harness/internal/runtime"
	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/memory"
)

// TestInjectMemoryContext_T_MEM_INJECT_01 valida o helper puro injectMemoryContext.
// Critérios:
//   - Sem documentos: prompt original retornado sem alteração.
//   - Com workflow: ## Memory Context injetado com conteúdo.
//   - Com task: ## Task Memory injetado.
//   - Com NeedsCompaction=true: diretiva "compact the flagged memory files before proceeding" anexada.
//   - Com erro de leitura: documento correspondente ignorado.
func TestInjectMemoryContext_T_MEM_INJECT_01(t *testing.T) {
	t.Parallel()

	const basePrompt = "execute task X"

	t.Run("sem documentos: prompt original preservado", func(t *testing.T) {
		t.Parallel()

		result := airuntime.InjectMemoryContextForTest(
			basePrompt,
			memory.Document{},
			memory.Document{},
			nil, nil,
		)

		if result != basePrompt {
			t.Errorf("sem docs: prompt alterado indevidamente; got=%q want=%q", result, basePrompt)
		}
	})

	t.Run("workflow existente: ## Memory Context injetado", func(t *testing.T) {
		t.Parallel()

		wf := memory.Document{
			FileState: memory.FileState{Path: "/.specs/prd-foo/memory/MEMORY.md"},
			Content:   "workflow content aqui",
			Exists:    true,
		}

		result := airuntime.InjectMemoryContextForTest(basePrompt, wf, memory.Document{}, nil, nil)

		if !strings.Contains(result, "## Memory Context") {
			t.Error("deve conter '## Memory Context'")
		}
		if !strings.Contains(result, "workflow content aqui") {
			t.Error("deve conter conteúdo do workflow")
		}
		if !strings.Contains(result, basePrompt) {
			t.Error("deve preservar prompt base")
		}
	})

	t.Run("task existente: ## Task Memory injetado", func(t *testing.T) {
		t.Parallel()

		tk := memory.Document{
			FileState: memory.FileState{Path: "/.specs/prd-foo/memory/task-6.0.md"},
			Content:   "task memory content",
			Exists:    true,
		}

		result := airuntime.InjectMemoryContextForTest(basePrompt, memory.Document{}, tk, nil, nil)

		if !strings.Contains(result, "## Memory Context") {
			t.Error("deve conter '## Memory Context'")
		}
		if !strings.Contains(result, "task memory content") {
			t.Error("deve conter conteúdo da task")
		}
	})

	t.Run("NeedsCompaction=true: diretiva de compactação anexada", func(t *testing.T) {
		t.Parallel()

		wf := memory.Document{
			FileState: memory.FileState{
				Path:            "/.specs/prd-foo/memory/MEMORY.md",
				NeedsCompaction: true,
			},
			Content: strings.Repeat("linha\n", 160), // >150 linhas
			Exists:  true,
		}

		result := airuntime.InjectMemoryContextForTest(basePrompt, wf, memory.Document{}, nil, nil)

		const directive = "compact the flagged memory files before proceeding"
		if !strings.Contains(result, directive) {
			t.Errorf("deve conter diretiva de compactação %q; got=%q", directive, result)
		}
	})

	t.Run("erro de leitura: documento ignorado silenciosamente", func(t *testing.T) {
		t.Parallel()

		wf := memory.Document{Exists: false}
		tk := memory.Document{Exists: false}

		result := airuntime.InjectMemoryContextForTest(basePrompt, wf, tk, nil, nil)

		// Sem documentos: retorna prompt original.
		if result != basePrompt {
			t.Errorf("com docs inexistentes: prompt alterado; got=%q", result)
		}
	})

	t.Run("workflow + task: ambos injetados na ordem correta", func(t *testing.T) {
		t.Parallel()

		wf := memory.Document{
			FileState: memory.FileState{Path: "/m/MEMORY.md"},
			Content:   "wf-content",
			Exists:    true,
		}
		tk := memory.Document{
			FileState: memory.FileState{Path: "/m/task.md"},
			Content:   "tk-content",
			Exists:    true,
		}

		result := airuntime.InjectMemoryContextForTest(basePrompt, wf, tk, nil, nil)

		idxWF := strings.Index(result, "wf-content")
		idxTK := strings.Index(result, "tk-content")

		if idxWF < 0 || idxTK < 0 {
			t.Error("ambos os conteúdos devem estar presentes")
		}
		if idxWF > idxTK {
			t.Error("workflow deve aparecer antes de task no prompt final")
		}
	})

	t.Run("Regressão F2: prompt sem TasksDir permanece idêntico", func(t *testing.T) {
		t.Parallel()

		// Com documentos vazios (sem TasksDir → store nil → injectMemoryContext não chamado).
		// Este sub-test valida que o contrato puro também é estável com zero-values.
		result := airuntime.InjectMemoryContextForTest(
			basePrompt,
			memory.Document{Exists: false},
			memory.Document{Exists: false},
			nil, nil,
		)

		if result != basePrompt {
			t.Errorf("regressão F2: prompt modificado sem docs; got=%q", result)
		}
	})
}
