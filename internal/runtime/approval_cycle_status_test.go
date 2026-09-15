package runtime_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	airuntime "github.com/JailtonJunior94/ai-spec-harness/internal/runtime"
)

func writeTaskFileWithStatus(t *testing.T, tasksDir, name, status string) {
	t.Helper()
	if err := os.MkdirAll(tasksDir, 0o755); err != nil {
		t.Fatalf("mkdir tasksDir: %v", err)
	}
	content := "# Task 4.2\n\n**Status:** " + status + "\n\n## Critérios de Sucesso\n\n- [ ] Faz X\n"
	if err := os.WriteFile(filepath.Join(tasksDir, name), []byte(content), 0o644); err != nil {
		t.Fatalf("write task file: %v", err)
	}
	table := "| # | Título | Status | Dependências |\n|---|--------|--------|--------------|\n| 4.2 | T 4.2 | " + status + " | — |\n"
	if err := os.WriteFile(filepath.Join(tasksDir, "tasks.md"), []byte(table), 0o644); err != nil {
		t.Fatalf("write tasks.md: %v", err)
	}
}

func readFileForStatus(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %q: %v", path, err)
	}
	return string(data)
}

func TestApprovalCycleClosedWithoutApprovalRewritesTaskStatusToBlocked(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	workDir := gitWorkDirWithAgentsMDForCycle(t)
	tasksDir := t.TempDir()
	writeTaskFileWithStatus(t, tasksDir, "4.2_task.md", "done")

	factory := newSequencedClientFactory(t, ctx,
		sequencedCall{script: reviewScript("main session")},
		sequencedCall{script: reviewScript("[high] fix.go:1 issue\n\nVerdict: REJECTED\n")},
		sequencedCall{script: reviewScript("fix attempted without touching any file")},
		sequencedCall{script: reviewScript("[high] fix.go:2 distinct issue\n\nVerdict: REJECTED\n")},
	)
	runner := buildSequencedRunner(t, factory)

	job := airuntime.Job{
		Prompt:       "implement task 4.2",
		WorkDir:      workDir,
		EvidenceDir:  t.TempDir(),
		Quiet:        true,
		AutoReview:   true,
		TasksDir:     tasksDir,
		TaskFileName: "4.2_task.md",
	}

	summary, err := runner.Run(ctx, job)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}
	if summary.CycleStopReason == "approved" {
		t.Fatalf("test setup approved the cycle; expected a closed-without-approval cycle")
	}

	taskFile := readFileForStatus(t, filepath.Join(tasksDir, "4.2_task.md"))
	if !strings.Contains(taskFile, "**Status:** blocked") {
		t.Errorf("task file status was not rewritten to blocked:\n%s", taskFile)
	}
	tasksTable := readFileForStatus(t, filepath.Join(tasksDir, "tasks.md"))
	if !strings.Contains(tasksTable, "| 4.2 | T 4.2 | blocked |") {
		t.Errorf("tasks.md row was not rewritten to blocked:\n%s", tasksTable)
	}
}

func TestApprovalCycleApprovedKeepsTaskStatusUntouched(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	workDir := gitWorkDirWithAgentsMDForCycle(t)
	tasksDir := t.TempDir()
	writeTaskFileWithStatus(t, tasksDir, "4.2_task.md", "done")

	factory := newSequencedClientFactory(t, ctx,
		sequencedCall{script: reviewScript("main session")},
		sequencedCall{script: reviewScript(approvedReviewOutput())},
	)
	runner := buildSequencedRunner(t, factory)

	summary, err := runner.Run(ctx, airuntime.Job{
		Prompt:       "implement task 4.2",
		WorkDir:      workDir,
		EvidenceDir:  t.TempDir(),
		Quiet:        true,
		AutoReview:   true,
		TasksDir:     tasksDir,
		TaskFileName: "4.2_task.md",
	})
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}
	if summary.CycleStopReason != "approved" {
		t.Fatalf("CycleStopReason = %q, want approved", summary.CycleStopReason)
	}
	if !strings.Contains(readFileForStatus(t, filepath.Join(tasksDir, "4.2_task.md")), "**Status:** done") {
		t.Error("approved cycle must not rewrite the task status")
	}
}
