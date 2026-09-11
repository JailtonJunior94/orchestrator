package aispecharness

import (
	"os"
	"path/filepath"
	"testing"
)

func writeTraceabilityFixture(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
		t.Fatalf("write fixture %s: %v", name, err)
	}
}

func runCheckTraceability(t *testing.T, arg string) int {
	t.Helper()
	cmd := newCheckTraceabilityCmd()
	err := cmd.RunE(cmd, []string{arg})
	if err == nil {
		return 0
	}
	return NewExitResolver().CodeFor(err)
}

func TestCheckTraceabilityCmd(t *testing.T) {
	cases := []struct {
		name     string
		prepare  func(t *testing.T, root string)
		wantCode int
	}{
		{
			name: "cadeia fechada retorna zero",
			prepare: func(t *testing.T, root string) {
				writeTraceabilityFixture(t, root, "prd.md", "RF-01 deve existir.")
				writeTraceabilityFixture(t, root, "tasks.md", "# Tasks\n\n## Cobertura de Requisitos\n\n| Tarefa | Requisitos cobertos |\n|---|---|\n| 1.0 | RF-01 |\n")
				writeTraceabilityFixture(t, root, "1.0_execution_report.md", "# Report\n\n## Critérios de Aceite\n- criterio -> comprovado: log\n")
			},
			wantCode: 0,
		},
		{
			name: "requisito sem tarefa retorna um",
			prepare: func(t *testing.T, root string) {
				writeTraceabilityFixture(t, root, "prd.md", "RF-01 deve existir. RF-02 tambem.")
				writeTraceabilityFixture(t, root, "tasks.md", "# Tasks\n\n## Cobertura de Requisitos\n\n| Tarefa | Requisitos cobertos |\n|---|---|\n| 1.0 | RF-01 |\n")
				writeTraceabilityFixture(t, root, "1.0_execution_report.md", "# Report\n\n## Critérios de Aceite\n- criterio -> comprovado: log\n")
			},
			wantCode: 1,
		},
		{
			name: "criterio sem evidencia retorna um",
			prepare: func(t *testing.T, root string) {
				writeTraceabilityFixture(t, root, "prd.md", "RF-01 deve existir.")
				writeTraceabilityFixture(t, root, "tasks.md", "# Tasks\n\n## Cobertura de Requisitos\n\n| Tarefa | Requisitos cobertos |\n|---|---|\n| 1.0 | RF-01 |\n")
				writeTraceabilityFixture(t, root, "1.0_execution_report.md", "# Report\n\n## Critérios de Aceite\n- criterio sem seta\n")
			},
			wantCode: 1,
		},
		{
			name:     "diretorio inexistente retorna erro de uso",
			prepare:  func(t *testing.T, root string) {},
			wantCode: 2,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			tc.prepare(t, root)

			arg := root
			if tc.name == "diretorio inexistente retorna erro de uso" {
				arg = filepath.Join(root, "inexistente")
			}

			gotCode := runCheckTraceability(t, arg)
			if gotCode != tc.wantCode {
				t.Fatalf("codigo de saida = %d, quer %d", gotCode, tc.wantCode)
			}
		})
	}
}
