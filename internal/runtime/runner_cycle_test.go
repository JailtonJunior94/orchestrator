package runtime_test

import (
	"context"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"testing"
	"time"

	airuntime "github.com/JailtonJunior94/ai-spec-harness/internal/runtime"
	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/acpfake"
)

func gitWorkDirWithAgentsMDForCycle(t *testing.T) string {
	t.Helper()
	dir := gitInitRepoForAdapter(t)
	if err := os.WriteFile(filepath.Join(dir, "AGENTS.md"), []byte("# Agents\n"), 0o644); err != nil {
		t.Fatalf("write AGENTS.md: %v", err)
	}
	skillDir := filepath.Join(dir, ".agents", "skills", "review")
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatalf("mkdir skill dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte("# Review Skill\n"), 0o644); err != nil {
		t.Fatalf("write SKILL.md: %v", err)
	}
	return dir
}

func writeCycleTaskFile(t *testing.T, tasksDir, name string) {
	t.Helper()
	if err := os.MkdirAll(tasksDir, 0o755); err != nil {
		t.Fatalf("mkdir tasksDir: %v", err)
	}
	content := "# Task\n\n## Critérios de Sucesso\n\n- [ ] Faz X\n"
	if err := os.WriteFile(filepath.Join(tasksDir, name), []byte(content), 0o644); err != nil {
		t.Fatalf("write task file: %v", err)
	}
}

func TestACPRunnerConductsCycleApprovedFirstRound(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	script := acpfake.NewScript().
		AppendAgentMessage("implementando a tarefa").
		AppendSessionEnd()

	reviewFn := func(_ context.Context, _ airuntime.Job) (string, error) {
		return "Verdict: APPROVED\n", nil
	}

	runner := buildRunnerWithReviewFn(t, ctx, script, reviewFn)

	workDir := gitWorkDirWithAgentsMDForCycle(t)
	tasksDir := t.TempDir()
	writeCycleTaskFile(t, tasksDir, "task-x.md")

	job := airuntime.Job{
		Prompt:       "implementar tarefa x",
		WorkDir:      workDir,
		EvidenceDir:  t.TempDir(),
		Quiet:        true,
		AutoReview:   true,
		TasksDir:     tasksDir,
		TaskFileName: "task-x.md",
	}

	summary, err := runner.Run(ctx, job)
	if err != nil {
		t.Fatalf("Run falhou: %v", err)
	}

	if summary.ReviewStatus != "ok" {
		t.Errorf("ReviewStatus = %q, quero ok", summary.ReviewStatus)
	}
	if summary.CycleStopReason != "approved" {
		t.Errorf("CycleStopReason = %q, quero approved", summary.CycleStopReason)
	}
	if len(summary.CycleRounds) != 1 {
		t.Fatalf("CycleRounds len = %d, quero 1", len(summary.CycleRounds))
	}
	if summary.CycleRounds[0].Number != 1 {
		t.Errorf("CycleRounds[0].Number = %d, quero 1", summary.CycleRounds[0].Number)
	}
	if summary.CycleRounds[0].Verdict != "APPROVED" {
		t.Errorf("CycleRounds[0].Verdict = %q, quero APPROVED", summary.CycleRounds[0].Verdict)
	}
}

func TestACPRunnerCycleFeedsRemarksBackToFix(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	script := acpfake.NewScript().
		AppendAgentMessage("sessão").
		AppendSessionEnd()

	var mu sync.Mutex
	calls := 0
	reviewFn := func(_ context.Context, _ airuntime.Job) (string, error) {
		mu.Lock()
		defer mu.Unlock()
		calls++
		if calls == 1 {
			return "Verdict: APPROVED_WITH_REMARKS\n[HIGH] fix.go: precisa de ajuste\n", nil
		}
		return "Verdict: APPROVED\n", nil
	}

	runner := buildRunnerWithReviewFn(t, ctx, script, reviewFn)

	workDir := gitWorkDirWithAgentsMDForCycle(t)
	tasksDir := t.TempDir()
	writeCycleTaskFile(t, tasksDir, "task-x.md")

	job := airuntime.Job{
		Prompt:       "implementar tarefa x",
		WorkDir:      workDir,
		EvidenceDir:  t.TempDir(),
		Quiet:        true,
		AutoReview:   true,
		TasksDir:     tasksDir,
		TaskFileName: "task-x.md",
	}

	summary, err := runner.Run(ctx, job)
	if err != nil {
		t.Fatalf("Run falhou: %v", err)
	}

	if summary.CycleStopReason != "approved" {
		t.Errorf("CycleStopReason = %q, quero approved", summary.CycleStopReason)
	}
	if len(summary.CycleRounds) != 2 {
		t.Fatalf("CycleRounds len = %d, quero 2 (ressalva realimentou a correção)", len(summary.CycleRounds))
	}
	if summary.CycleRounds[0].Verdict != "APPROVED_WITH_REMARKS" {
		t.Errorf("CycleRounds[0].Verdict = %q, quero APPROVED_WITH_REMARKS", summary.CycleRounds[0].Verdict)
	}
	if summary.CycleRounds[1].Verdict != "APPROVED" {
		t.Errorf("CycleRounds[1].Verdict = %q, quero APPROVED", summary.CycleRounds[1].Verdict)
	}
	if summary.CycleRounds[0].FindingsBySeverity["high"] != 1 {
		t.Errorf("CycleRounds[0].FindingsBySeverity[high] = %d, quero 1", summary.CycleRounds[0].FindingsBySeverity["high"])
	}
}

func TestACPRunnerCycleAbortsOnNoConvergenceWithoutSpendingNextRound(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	script := acpfake.NewScript().
		AppendAgentMessage("sessão").
		AppendSessionEnd()

	reviewFn := func(_ context.Context, _ airuntime.Job) (string, error) {
		return "Verdict: REJECTED\n[HIGH] fix.go: mesmo problema\n", nil
	}

	runner := buildRunnerWithReviewFn(t, ctx, script, reviewFn)

	workDir := gitWorkDirWithAgentsMDForCycle(t)
	tasksDir := t.TempDir()
	writeCycleTaskFile(t, tasksDir, "task-x.md")

	job := airuntime.Job{
		Prompt:       "implementar tarefa x",
		WorkDir:      workDir,
		EvidenceDir:  t.TempDir(),
		Quiet:        true,
		AutoReview:   true,
		TasksDir:     tasksDir,
		TaskFileName: "task-x.md",
	}

	summary, err := runner.Run(ctx, job)
	if err != nil {
		t.Fatalf("Run falhou (estado terminal não é erro de infraestrutura, RF-45): %v", err)
	}

	if summary.CycleStopReason != "no_convergence" {
		t.Errorf("CycleStopReason = %q, quero no_convergence", summary.CycleStopReason)
	}
	if len(summary.CycleRounds) != 2 {
		t.Fatalf("CycleRounds len = %d, quero 2 (fingerprint repetida aborta sem gastar a rodada seguinte)", len(summary.CycleRounds))
	}
}

func TestACPRunnerCycleAbortsOnEmptyDiff(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	script := acpfake.NewScript().
		AppendAgentMessage("sessão").
		AppendSessionEnd()

	var mu sync.Mutex
	calls := 0
	reviewFn := func(_ context.Context, _ airuntime.Job) (string, error) {
		mu.Lock()
		defer mu.Unlock()
		calls++
		return "Verdict: REJECTED\n[HIGH] fix.go:" + strconv.Itoa(calls) + " problema distinto\n", nil
	}

	runner := buildRunnerWithReviewFn(t, ctx, script, reviewFn)

	workDir := gitWorkDirWithAgentsMDForCycle(t)
	tasksDir := t.TempDir()
	writeCycleTaskFile(t, tasksDir, "task-x.md")

	job := airuntime.Job{
		Prompt:       "implementar tarefa x",
		WorkDir:      workDir,
		EvidenceDir:  t.TempDir(),
		Quiet:        true,
		AutoReview:   true,
		TasksDir:     tasksDir,
		TaskFileName: "task-x.md",
	}

	summary, err := runner.Run(ctx, job)
	if err != nil {
		t.Fatalf("Run falhou: %v", err)
	}

	if summary.CycleStopReason != "empty_diff" {
		t.Errorf("CycleStopReason = %q, quero empty_diff (correção sem diff aborta)", summary.CycleStopReason)
	}
	if len(summary.CycleRounds) != 2 {
		t.Fatalf("CycleRounds len = %d, quero 2", len(summary.CycleRounds))
	}
}

func TestACPRunnerCycleHonorsConfiguredMaxRounds(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	script := acpfake.NewScript().
		AppendAgentMessage("sessão").
		AppendSessionEnd()

	var mu sync.Mutex
	calls := 0
	reviewFn := func(_ context.Context, _ airuntime.Job) (string, error) {
		mu.Lock()
		defer mu.Unlock()
		calls++
		return "Verdict: REJECTED\n[HIGH] fix.go: problema\n", nil
	}

	runner := buildRunnerWithReviewFn(t, ctx, script, reviewFn)

	workDir := gitWorkDirWithAgentsMDForCycle(t)
	tasksDir := t.TempDir()
	writeCycleTaskFile(t, tasksDir, "task-x.md")

	job := airuntime.Job{
		Prompt:        "implementar tarefa x",
		WorkDir:       workDir,
		EvidenceDir:   t.TempDir(),
		Quiet:         true,
		AutoReview:    true,
		TasksDir:      tasksDir,
		TaskFileName:  "task-x.md",
		RuntimeConfig: airuntime.RuntimeConfig{MaxBugfixIterations: 1},
	}

	summary, err := runner.Run(ctx, job)
	if err != nil {
		t.Fatalf("Run falhou (estado terminal não é erro de infraestrutura, RF-45): %v", err)
	}

	if summary.ReviewStatus == "ok" {
		t.Fatalf("ReviewStatus = ok, nunca deveria aprovar sem veredito APPROVED (RF-56)")
	}
	if summary.CycleStopReason != "max_rounds" {
		t.Errorf("CycleStopReason = %q, quero max_rounds (teto configurado para 1)", summary.CycleStopReason)
	}
	if len(summary.CycleRounds) != 1 {
		t.Fatalf("CycleRounds len = %d, quero 1 (teto de 1 rodada esgotado na primeira review)", len(summary.CycleRounds))
	}

	mu.Lock()
	defer mu.Unlock()
	if calls != 1 {
		t.Errorf("reviewFn chamada %d vezes, quero 1 (teto de 1 rodada não deve acionar correção)", calls)
	}
}

func TestACPRunnerCycleDefaultMaxRoundsIsFiveWithoutConfiguration(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	script := acpfake.NewScript().
		AppendAgentMessage("sessão").
		AppendSessionEnd()

	var mu sync.Mutex
	calls := 0
	reviewFn := func(_ context.Context, _ airuntime.Job) (string, error) {
		mu.Lock()
		defer mu.Unlock()
		calls++
		return "Verdict: REJECTED\n[HIGH] fix.go:" + strconv.Itoa(calls) + " problema distinto\n", nil
	}

	runner := buildRunnerWithReviewFn(t, ctx, script, reviewFn)

	workDir := gitWorkDirWithAgentsMDForCycle(t)
	tasksDir := t.TempDir()
	writeCycleTaskFile(t, tasksDir, "task-x.md")

	job := airuntime.Job{
		Prompt:       "implementar tarefa x",
		WorkDir:      workDir,
		EvidenceDir:  t.TempDir(),
		Quiet:        true,
		AutoReview:   true,
		TasksDir:     tasksDir,
		TaskFileName: "task-x.md",
	}

	summary, err := runner.Run(ctx, job)
	if err != nil {
		t.Fatalf("Run falhou: %v", err)
	}

	if summary.CycleStopReason != "empty_diff" && summary.CycleStopReason != "max_rounds" {
		t.Errorf("CycleStopReason = %q, quero empty_diff ou max_rounds", summary.CycleStopReason)
	}
	if len(summary.CycleRounds) > 5 {
		t.Fatalf("CycleRounds len = %d, nunca deveria exceder o default de 5 rodadas", len(summary.CycleRounds))
	}
}

func TestACPRunnerFallsBackToOneShotWithoutTaskFile(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	script := acpfake.NewScript().
		AppendAgentMessage("sessão sem contexto de task").
		AppendSessionEnd()

	var mu sync.Mutex
	calls := 0
	reviewFn := func(_ context.Context, _ airuntime.Job) (string, error) {
		mu.Lock()
		defer mu.Unlock()
		calls++
		return "Verdict: APPROVED\n", nil
	}

	runner := buildRunnerWithReviewFn(t, ctx, script, reviewFn)

	job := airuntime.Job{
		Prompt:      "tarefa interativa sem task file",
		WorkDir:     workDirWithAgentsMDForReview(t),
		EvidenceDir: t.TempDir(),
		Quiet:       true,
		AutoReview:  true,
	}

	summary, err := runner.Run(ctx, job)
	if err != nil {
		t.Fatalf("Run falhou: %v", err)
	}

	if summary.CycleRounds != nil {
		t.Errorf("CycleRounds = %v, quero nil (fallback one-shot sem Cycle)", summary.CycleRounds)
	}
	if summary.CycleStopReason != "" {
		t.Errorf("CycleStopReason = %q, quero vazio", summary.CycleStopReason)
	}

	mu.Lock()
	defer mu.Unlock()
	if calls != 1 {
		t.Errorf("reviewOutputFn chamada %d vezes, quero 1 (one-shot, sem rodadas)", calls)
	}
}
