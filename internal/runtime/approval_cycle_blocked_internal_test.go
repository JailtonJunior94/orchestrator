package runtime

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunApprovalCycleForcesBlockedWhenCriteriaAreMissing(t *testing.T) {
	tasksDir := t.TempDir()
	taskFile := filepath.Join(tasksDir, "4.2_task.md")
	if err := os.WriteFile(taskFile, []byte("# Task 4.2\n\n**Status:** done\n\nSem criterios de aceite.\n"), 0o644); err != nil {
		t.Fatalf("write task file: %v", err)
	}
	table := "| # | Título | Status | Dependências |\n|---|--------|--------|--------------|\n| 4.2 | T 4.2 | done | — |\n"
	if err := os.WriteFile(filepath.Join(tasksDir, "tasks.md"), []byte(table), 0o644); err != nil {
		t.Fatalf("write tasks.md: %v", err)
	}

	runner := &ACPRunner{}
	job := Job{WorkDir: t.TempDir(), TasksDir: tasksDir, TaskFileName: "4.2_task.md"}

	if _, err := runner.runApprovalCycle(context.Background(), job); err == nil {
		t.Fatal("task sem criterios de aceite deve falhar o Ciclo")
	}

	content, err := os.ReadFile(taskFile)
	if err != nil {
		t.Fatalf("read task file: %v", err)
	}
	if !strings.Contains(string(content), "**Status:** blocked") {
		t.Errorf("erro do Ciclo nao forcou blocked no arquivo da task (RF-36):\n%s", content)
	}
	tasksContent, err := os.ReadFile(filepath.Join(tasksDir, "tasks.md"))
	if err != nil {
		t.Fatalf("read tasks.md: %v", err)
	}
	if !strings.Contains(string(tasksContent), "| 4.2 | T 4.2 | blocked |") {
		t.Errorf("erro do Ciclo nao forcou blocked na tabela (RF-36):\n%s", tasksContent)
	}
}

func TestTaskStatusWriterFailsLoudOnMissingTaskFileOnly(t *testing.T) {
	tasksDir := t.TempDir()
	table := "| # | Título | Status | Dependências |\n|---|--------|--------|--------------|\n| 4.2 | T 4.2 | done | — |\n"
	if err := os.WriteFile(filepath.Join(tasksDir, "tasks.md"), []byte(table), 0o644); err != nil {
		t.Fatalf("write tasks.md: %v", err)
	}
	if err := NewTaskStatusWriter(tasksDir, "4.2_task.md").writeTaskFileStatus("blocked"); err == nil {
		t.Fatal("writeTaskFileStatus silenciou a ausencia do arquivo de task (RF-36)")
	}
}

func TestTaskStatusWriterFailsLoudOnMissingTasksTableOnly(t *testing.T) {
	tasksDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(tasksDir, "4.2_task.md"), []byte("**Status:** done\n"), 0o644); err != nil {
		t.Fatalf("write task file: %v", err)
	}
	if err := NewTaskStatusWriter(tasksDir, "4.2_task.md").writeTasksTableStatus("blocked"); err == nil {
		t.Fatal("writeTasksTableStatus silenciou a ausencia de tasks.md (RF-36)")
	}
}

func TestTaskStatusWriterFailsLoudWhenTaskRowIsAbsent(t *testing.T) {
	tasksDir := t.TempDir()
	table := "| # | Título | Status | Dependências |\n|---|--------|--------|--------------|\n| 9.9 | Outra | done | — |\n"
	if err := os.WriteFile(filepath.Join(tasksDir, "tasks.md"), []byte(table), 0o644); err != nil {
		t.Fatalf("write tasks.md: %v", err)
	}
	if err := NewTaskStatusWriter(tasksDir, "4.2_task.md").writeTasksTableStatus("blocked"); err == nil {
		t.Fatal("writeTasksTableStatus silenciou a ausencia da linha da task (RF-36)")
	}
}
