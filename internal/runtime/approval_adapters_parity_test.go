package runtime_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/JailtonJunior94/ai-spec-harness/internal/approval"
	airuntime "github.com/JailtonJunior94/ai-spec-harness/internal/runtime"
	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/acpfake"
)

func TestTaskStatusWriterDetectsStatusColumnDynamically(t *testing.T) {
	t.Parallel()

	tasksDir := t.TempDir()
	taskFile := filepath.Join(tasksDir, "4.2_task.md")
	if err := os.WriteFile(taskFile, []byte("# Task\n\n**Status:** done\n"), 0o644); err != nil {
		t.Fatalf("write task file: %v", err)
	}
	table := "| # | Status | Título | Dependências |\n" +
		"|---|--------|--------|--------------|\n" +
		"| 4.2 | done | T 4.2 | — |\n"
	tasksFile := filepath.Join(tasksDir, "tasks.md")
	if err := os.WriteFile(tasksFile, []byte(table), 0o644); err != nil {
		t.Fatalf("write tasks.md: %v", err)
	}

	if err := airuntime.NewTaskStatusWriter(tasksDir, "4.2_task.md").Force("blocked"); err != nil {
		t.Fatalf("Force: %v", err)
	}

	content, err := os.ReadFile(tasksFile)
	if err != nil {
		t.Fatalf("read tasks.md: %v", err)
	}
	if !strings.Contains(string(content), "| 4.2 | blocked | T 4.2 |") {
		t.Fatalf("status escrito na coluna errada:\n%s", content)
	}
}

func TestTaskStatusWriterFailsLoudWhenTaskFileIsMissing(t *testing.T) {
	t.Parallel()

	tasksDir := t.TempDir()
	if err := airuntime.NewTaskStatusWriter(tasksDir, "4.2_task.md").Force("blocked"); err == nil {
		t.Fatal("Force silenciou a ausencia do arquivo de task (RF-36)")
	}
}

func TestTaskStatusWriterFailsLoudWhenStatusFieldIsAbsent(t *testing.T) {
	t.Parallel()

	tasksDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(tasksDir, "4.2_task.md"), []byte("# Task sem campo de status\n"), 0o644); err != nil {
		t.Fatalf("write task file: %v", err)
	}
	if err := airuntime.NewTaskStatusWriter(tasksDir, "4.2_task.md").Force("blocked"); err == nil {
		t.Fatal("Force silenciou a ausencia do campo **Status:** (RF-36)")
	}
}

func TestRepositoryAdapterFullTargetSeesCommittedWork(t *testing.T) {
	t.Parallel()

	dir := gitRepoWithUpstreamBaseline(t)

	target, err := airuntime.NewRepositoryAdapter(dir).FullTarget(context.Background())
	if err != nil {
		t.Fatalf("FullTarget: %v", err)
	}
	if target.Empty() {
		t.Fatal("FullTarget cego a commits: se o executor commitou, a rodada 1 revisa nada")
	}
	if !strings.Contains(target.String(), "committed.go") {
		t.Fatalf("FullTarget nao inclui o trabalho commitado desde o ponto de corte:\n%s", target.String())
	}
}

func gitRepoWithUpstreamBaseline(t *testing.T) string {
	t.Helper()
	dir := gitInitRepoForAdapter(t)
	gitRunForCycle(t, dir, "branch", "baseline")
	if err := os.WriteFile(filepath.Join(dir, "committed.go"), []byte("package main\n"), 0o644); err != nil {
		t.Fatalf("write committed.go: %v", err)
	}
	gitRunForCycle(t, dir, "add", ".")
	gitRunForCycle(t, dir, "commit", "-m", "trabalho do executor")
	branch := gitRunForCycle(t, dir, "symbolic-ref", "--short", "HEAD")
	gitRunForCycle(t, dir, "config", "branch."+branch+".remote", ".")
	gitRunForCycle(t, dir, "config", "branch."+branch+".merge", "refs/heads/baseline")
	return dir
}

func TestApprovalCycleFailureForcesBlockedTaskStatus(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	script := acpfake.NewScript().
		AppendAgentMessage("implementando a tarefa").
		AppendSessionEnd()
	reviewFn := func(_ context.Context, _ airuntime.Job) (string, error) {
		return approvedCycleReview(), nil
	}
	runner := buildRunnerWithReviewFn(t, ctx, script, reviewFn)

	workDir := gitWorkDirWithAgentsMDForCycle(t)
	tasksDir := t.TempDir()
	taskFile := filepath.Join(tasksDir, "4.2_task.md")
	if err := os.WriteFile(taskFile, []byte("# Task 4.2\n\n**Status:** done\n\nSem criterios de aceite declarados.\n"), 0o644); err != nil {
		t.Fatalf("write task file: %v", err)
	}
	table := "| # | Título | Status | Dependências |\n|---|--------|--------|--------------|\n| 4.2 | T 4.2 | done | — |\n"
	if err := os.WriteFile(filepath.Join(tasksDir, "tasks.md"), []byte(table), 0o644); err != nil {
		t.Fatalf("write tasks.md: %v", err)
	}

	summary, err := runner.Run(ctx, airuntime.Job{
		Prompt:       "implementar tarefa 4.2",
		WorkDir:      workDir,
		EvidenceDir:  t.TempDir(),
		Quiet:        true,
		AutoReview:   true,
		TasksDir:     tasksDir,
		TaskFileName: "4.2_task.md",
	})
	if err != nil {
		t.Fatalf("Run falhou: %v", err)
	}
	if summary.ReviewStatus != "blocked" {
		t.Errorf("ReviewStatus = %q, quero blocked", summary.ReviewStatus)
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

type emptyTargetRepository struct{}

func (emptyTargetRepository) Checkpoint(context.Context) (approval.Checkpoint, error) {
	return approval.NewCheckpoint("deadbeef")
}

func (emptyTargetRepository) FullTarget(context.Context) (approval.ReviewTarget, error) {
	return approval.NewReviewTarget(""), nil
}

func (emptyTargetRepository) Delta(context.Context, approval.Checkpoint) (approval.ReviewTarget, error) {
	return approval.NewReviewTarget(""), nil
}

type unusedReviewer struct{ t *testing.T }

func (r unusedReviewer) Review(context.Context, approval.ReviewRequest) (approval.ReviewerOutput, error) {
	r.t.Fatal("rodada 1 com alvo vazio nao pode chegar ao revisor")
	return approval.ReviewerOutput{}, nil
}

type unusedFixer struct{}

func (unusedFixer) Fix(context.Context, approval.FixRequest) error { return nil }

func TestCycleFailsClosedWhenRoundOneTargetIsEmpty(t *testing.T) {
	t.Parallel()

	task, err := approval.NewTaskIdentity("task-1.0")
	if err != nil {
		t.Fatalf("NewTaskIdentity: %v", err)
	}
	agent, err := approval.NewAgentIdentity("claude")
	if err != nil {
		t.Fatalf("NewAgentIdentity: %v", err)
	}
	policy, err := approval.NewApprovalPolicy()
	if err != nil {
		t.Fatalf("NewApprovalPolicy: %v", err)
	}
	criterion, err := approval.NewAcceptanceCriterion("build green")
	if err != nil {
		t.Fatalf("NewAcceptanceCriterion: %v", err)
	}

	cycle, err := approval.NewCycle(task, agent, policy, []approval.AcceptanceCriterion{criterion},
		unusedReviewer{t: t}, unusedFixer{}, emptyTargetRepository{})
	if err != nil {
		t.Fatalf("NewCycle: %v", err)
	}

	if _, runErr := cycle.Run(context.Background()); runErr == nil {
		t.Fatal("Cycle.Run aprovou uma rodada 1 sem nada para revisar")
	}
}
