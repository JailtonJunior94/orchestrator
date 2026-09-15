package runtime_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	airuntime "github.com/JailtonJunior94/ai-spec-harness/internal/runtime"
	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/acpfake"
)

func TestACPRunnerCycleErrorForcesBlockedStatusOnDisk(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	script := acpfake.NewScript().
		AppendAgentMessage("sessão").
		AppendSessionEnd()

	reviewFn := func(_ context.Context, _ airuntime.Job) (string, error) {
		return "", errors.New("reviewer indisponivel")
	}

	runner := buildRunnerWithReviewFn(t, ctx, script, reviewFn)

	workDir := gitWorkDirWithAgentsMDForCycle(t)
	tasksDir := t.TempDir()

	taskFile := filepath.Join(tasksDir, "task-1.0.md")
	taskContent := "# Task 1.0\n\n**Status:** done\n\n## Critérios de Sucesso\n\n- [ ] Faz X\n"
	if err := os.WriteFile(taskFile, []byte(taskContent), 0o644); err != nil {
		t.Fatalf("setup task file: %v", err)
	}
	tasksTable := filepath.Join(tasksDir, "tasks.md")
	if err := os.WriteFile(tasksTable, []byte("| ID | Titulo | Status |\n|---|---|---|\n| 1.0 | Task | done |\n"), 0o644); err != nil {
		t.Fatalf("setup tasks.md: %v", err)
	}

	summary, err := runner.Run(ctx, airuntime.Job{
		Prompt:       "implementar tarefa 1.0",
		WorkDir:      workDir,
		EvidenceDir:  t.TempDir(),
		Quiet:        true,
		AutoReview:   true,
		TasksDir:     tasksDir,
		TaskFileName: "task-1.0.md",
	})
	if err != nil {
		t.Fatalf("Run falhou: %v", err)
	}
	if summary.ReviewStatus != "blocked" {
		t.Fatalf("ReviewStatus=%q, quero blocked", summary.ReviewStatus)
	}

	gotTask := readFileForCycle(t, taskFile)
	if !strings.Contains(gotTask, "**Status:** blocked") {
		t.Errorf("task file nao foi forcado para blocked (RF-42):\n%s", gotTask)
	}
	gotTable := readFileForCycle(t, tasksTable)
	if !strings.Contains(gotTable, "| blocked |") {
		t.Errorf("tasks.md nao foi forcado para blocked (RF-42):\n%s", gotTable)
	}
}

func readFileForCycle(t *testing.T, path string) string {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ler %s: %v", path, err)
	}
	return string(raw)
}
