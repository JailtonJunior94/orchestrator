package taskloop

import (
	"context"
	"testing"

	"github.com/JailtonJunior94/ai-spec-harness/internal/fs"
)

func TestEvidenceRecorder_Append(t *testing.T) {
	report := AcceptanceReport{
		TaskID:        "task-3.0",
		Passed:        true,
		CriteriaMet:   3,
		CriteriaTotal: 3,
		GoTestOutput:  "ok  github.com/JailtonJunior94/ai-spec-harness/internal/taskloop",
		GoVetOutput:   "",
		LintOutput:    "",
	}

	tests := []struct {
		name          string
		initialContent string
		wantContains  []string
		wantNotLost   string
	}{
		{
			name:           "cria bloco quando ausente",
			initialContent: "# Task\n\nConteúdo da task.\n",
			wantContains:   []string{evidenceStart, evidenceEnd, "task-3.0", "3/3"},
			wantNotLost:    "Conteúdo da task.",
		},
		{
			name: "substitui bloco existente",
			initialContent: "# Task\n\n" + evidenceStart + "\n## Evidência\n\n**Task:** old\n" + evidenceEnd + "\n",
			wantContains:   []string{evidenceStart, evidenceEnd, "task-3.0"},
			wantNotLost:    "# Task",
		},
		{
			name:           "preserva conteudo apos o bloco",
			initialContent: "# Task\n\n" + evidenceStart + "\nold\n" + evidenceEnd + "\n\n## Proxima Secao\n",
			wantContains:   []string{"## Proxima Secao", "task-3.0"},
			wantNotLost:    "## Proxima Secao",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			fakeFS := fs.NewFakeFileSystem()
			fakeFS.Files["task.md"] = []byte(tc.initialContent)

			rec := NewEvidenceRecorder(fakeFS)
			ctx := context.Background()

			if err := rec.Append(ctx, "task.md", report); err != nil {
				t.Fatalf("Append: %v", err)
			}

			result := string(fakeFS.Files["task.md"])
			for _, want := range tc.wantContains {
				if !contains(result, want) {
					t.Errorf("resultado nao contem %q\ngot:\n%s", want, result)
				}
			}
			if tc.wantNotLost != "" && !contains(result, tc.wantNotLost) {
				t.Errorf("conteudo perdido: %q\ngot:\n%s", tc.wantNotLost, result)
			}
		})
	}
}

func TestEvidenceRecorder_Idempotency(t *testing.T) {
	report := AcceptanceReport{
		TaskID:        "task-idem",
		Passed:        true,
		CriteriaMet:   2,
		CriteriaTotal: 2,
		GoTestOutput:  "PASS",
	}

	fakeFS := fs.NewFakeFileSystem()
	fakeFS.Files["task.md"] = []byte("# Task\n\nTexto inicial.\n")

	rec := NewEvidenceRecorder(fakeFS)
	ctx := context.Background()

	if err := rec.Append(ctx, "task.md", report); err != nil {
		t.Fatalf("primeira chamada: %v", err)
	}
	first := string(fakeFS.Files["task.md"])

	if err := rec.Append(ctx, "task.md", report); err != nil {
		t.Fatalf("segunda chamada: %v", err)
	}
	second := string(fakeFS.Files["task.md"])

	if first != second {
		t.Errorf("idempotência violada\nprimeira:\n%s\nsegunda:\n%s", first, second)
	}
}

func TestEvidenceRecorder_FileNotFound(t *testing.T) {
	fakeFS := fs.NewFakeFileSystem()
	rec := NewEvidenceRecorder(fakeFS)

	err := rec.Append(context.Background(), "inexistente.md", AcceptanceReport{})
	if err == nil {
		t.Fatal("esperava erro para arquivo inexistente")
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(sub) == 0 ||
		func() bool {
			for i := 0; i <= len(s)-len(sub); i++ {
				if s[i:i+len(sub)] == sub {
					return true
				}
			}
			return false
		}())
}
