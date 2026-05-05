package taskloop

import (
	"context"
	"errors"
	"testing"

	"github.com/JailtonJunior94/ai-spec-harness/internal/fs"
)

const tasksHeader = "| # | Titulo | Status | Dependencias | Paralelizavel |\n|---|--------|--------|-------------|---------------|\n"

func buildTasksMD(rows string) []byte {
	return []byte("# Tasks\n\n## Tarefas\n\n" + tasksHeader + rows)
}

func TestTaskSelectorNext(t *testing.T) {
	tests := []struct {
		name         string
		tasksContent []byte
		wantID       string
		wantErr      error
	}{
		{
			name:         "lista vazia retorna ErrNoEligibleTask",
			tasksContent: buildTasksMD("| 1.0 | Task A | done | — | — |\n"),
			wantErr:      ErrNoEligibleTask,
		},
		{
			name:         "unica task pending sem deps retorna ela",
			tasksContent: buildTasksMD("| 1.0 | Task A | pending | — | — |\n"),
			wantID:       "1.0",
		},
		{
			name: "deps nao satisfeitas — nenhuma elegivel",
			tasksContent: buildTasksMD(
				"| 1.0 | Task A | pending | — | — |\n" +
					"| 2.0 | Task B | pending | 1.0 | — |\n",
			),
			wantID: "1.0",
		},
		{
			name: "deps satisfeitas — retorna a dependente",
			tasksContent: buildTasksMD(
				"| 1.0 | Task A | done | — | — |\n" +
					"| 2.0 | Task B | pending | 1.0 | — |\n",
			),
			wantID: "2.0",
		},
		{
			name: "task in_progress retomada antes de pending",
			tasksContent: buildTasksMD(
				"| 1.0 | Task A | in_progress | — | — |\n" +
					"| 2.0 | Task B | pending | — | — |\n",
			),
			wantID: "1.0",
		},
		{
			name: "task done ignorada",
			tasksContent: buildTasksMD(
				"| 1.0 | Task A | done | — | — |\n" +
					"| 2.0 | Task B | done | — | — |\n",
			),
			wantErr: ErrNoEligibleTask,
		},
		{
			name: "ordem estavel — menor ID retornado primeiro",
			tasksContent: buildTasksMD(
				"| 3.0 | Task C | pending | — | — |\n" +
					"| 1.0 | Task A | pending | — | — |\n" +
					"| 2.0 | Task B | pending | — | — |\n",
			),
			wantID: "1.0",
		},
		{
			name: "ciclo de dependencias retorna ErrDependencyCycle",
			tasksContent: buildTasksMD(
				"| 1.0 | Task A | pending | 2.0 | — |\n" +
					"| 2.0 | Task B | pending | 1.0 | — |\n",
			),
			wantErr: ErrDependencyCycle,
		},
		{
			name: "multiplas deps, todas done — elegivel",
			tasksContent: buildTasksMD(
				"| 1.0 | Task A | done | — | — |\n" +
					"| 2.0 | Task B | done | — | — |\n" +
					"| 3.0 | Task C | pending | 1.0, 2.0 | — |\n",
			),
			wantID: "3.0",
		},
		{
			name: "dep parcialmente done — nao elegivel",
			tasksContent: buildTasksMD(
				"| 1.0 | Task A | done | — | — |\n" +
					"| 2.0 | Task B | pending | — | — |\n" +
					"| 3.0 | Task C | pending | 1.0, 2.0 | — |\n",
			),
			wantID: "2.0",
		},
		{
			name:         "tasks.md sem tabela retorna erro de parse",
			tasksContent: []byte("# Sem tabela aqui\n"),
			wantErr:      errors.New("taskloop: erro ao parsear tasks.md"),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			fsys := fs.NewFakeFileSystem()
			fsys.Files["/prd/tasks.md"] = tc.tasksContent

			sel := NewTaskSelector(fsys)
			got, err := sel.Next(context.Background(), "/prd")

			if tc.wantErr != nil {
				if err == nil {
					t.Fatalf("esperava erro mas nao obteve nenhum")
				}
				if errors.Is(tc.wantErr, ErrNoEligibleTask) || errors.Is(tc.wantErr, ErrDependencyCycle) {
					if !errors.Is(err, tc.wantErr) {
						t.Fatalf("erro esperado %v, obteve %v", tc.wantErr, err)
					}
				}
				return
			}

			if err != nil {
				t.Fatalf("erro inesperado: %v", err)
			}
			if got.ID != tc.wantID {
				t.Fatalf("ID esperado %q, obteve %q", tc.wantID, got.ID)
			}
		})
	}
}

func TestTaskSelectorNextDeterminism(t *testing.T) {
	content := buildTasksMD(
		"| 2.0 | Task B | pending | — | — |\n" +
			"| 1.0 | Task A | pending | — | — |\n",
	)

	fsys := fs.NewFakeFileSystem()
	fsys.Files["/prd/tasks.md"] = content
	sel := NewTaskSelector(fsys)

	var ids []string
	for i := 0; i < 5; i++ {
		got, err := sel.Next(context.Background(), "/prd")
		if err != nil {
			t.Fatalf("iteracao %d: erro inesperado: %v", i, err)
		}
		ids = append(ids, got.ID)
	}

	for i, id := range ids {
		if id != "1.0" {
			t.Fatalf("iteracao %d: esperava 1.0, obteve %s", i, id)
		}
	}
}

func TestTaskSelectorNextContextCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	fsys := fs.NewFakeFileSystem()
	fsys.Files["/prd/tasks.md"] = buildTasksMD("| 1.0 | Task A | pending | — | — |\n")

	sel := NewTaskSelector(fsys)
	_, err := sel.Next(ctx, "/prd")
	if err == nil {
		t.Fatal("esperava erro com contexto cancelado")
	}
}
