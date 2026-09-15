package taskloop

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/JailtonJunior94/ai-spec-harness/internal/approval"
)

type fixedDiffCapturer struct{ diff string }

func (c *fixedDiffCapturer) CaptureDiff(context.Context) (string, error) {
	return c.diff, nil
}

func initGitRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	for _, args := range [][]string{
		{"init", "-q"},
		{"config", "user.email", "cycle@example.invalid"},
		{"config", "user.name", "Cycle Test"},
		{"config", "commit.gpgsign", "false"},
	} {
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v (%s)", args, err, out)
		}
	}
	if err := os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main\n"), 0o600); err != nil {
		t.Fatalf("write baseline: %v", err)
	}
	for _, args := range [][]string{{"add", "."}, {"commit", "-qm", "baseline"}} {
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v (%s)", args, err, out)
		}
	}
	return dir
}

func TestRepositoryPortDeltaUsesCheckpoint(t *testing.T) {
	dir := initGitRepo(t)
	port := newRepositoryPort(&fixedDiffCapturer{diff: "fallback-diff"}, dir)
	ctx := context.Background()

	if err := os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main\n\nfunc first() {}\n"), 0o600); err != nil {
		t.Fatalf("write round 1: %v", err)
	}
	for _, args := range [][]string{{"add", "."}, {"commit", "-qm", "round-1"}} {
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v (%s)", args, err, out)
		}
	}

	checkpoint, err := port.Checkpoint(ctx)
	if err != nil {
		t.Fatalf("Checkpoint: %v", err)
	}

	if err := os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main\n\nfunc first() {}\n\nfunc second() {}\n"), 0o600); err != nil {
		t.Fatalf("write round 2: %v", err)
	}

	delta, err := port.Delta(ctx, checkpoint)
	if err != nil {
		t.Fatalf("Delta: %v", err)
	}
	if strings.Contains(delta.String(), "+func first") {
		t.Errorf("Delta reincluiu a mudanca da rodada anterior:\n%s", delta.String())
	}
	if !strings.Contains(delta.String(), "+func second") {
		t.Errorf("Delta nao contem a mudanca da rodada corrente:\n%s", delta.String())
	}

	full, err := port.FullTarget(ctx)
	if err != nil {
		t.Fatalf("FullTarget: %v", err)
	}
	if full.String() == delta.String() {
		t.Error("Delta e FullTarget devolveram o mesmo alvo; o checkpoint foi ignorado (RF-39)")
	}
}

func TestRepositoryPortDeltaEmptyWhenNothingChanged(t *testing.T) {
	dir := initGitRepo(t)
	port := newRepositoryPort(&fixedDiffCapturer{diff: "fallback-diff"}, dir)
	ctx := context.Background()

	checkpoint, err := port.Checkpoint(ctx)
	if err != nil {
		t.Fatalf("Checkpoint: %v", err)
	}
	delta, err := port.Delta(ctx, checkpoint)
	if err != nil {
		t.Fatalf("Delta: %v", err)
	}
	if !delta.Empty() {
		t.Errorf("Delta deveria ser vazio sem mudanca, obteve:\n%s", delta.String())
	}
}

func TestRepositoryPortDeltaFallsBackWithoutGit(t *testing.T) {
	dir := t.TempDir()
	port := newRepositoryPort(&fixedDiffCapturer{diff: "fallback-diff"}, dir)

	checkpoint, err := approval.NewCheckpoint("0123456789abcdef0123456789abcdef01234567")
	if err != nil {
		t.Fatalf("NewCheckpoint: %v", err)
	}
	delta, err := port.Delta(context.Background(), checkpoint)
	if err != nil {
		t.Fatalf("Delta: %v", err)
	}
	if delta.String() != "fallback-diff" {
		t.Errorf("fallback de conteudo perdido, obteve %q", delta.String())
	}
}

func TestCycleApprovalPolicyDefaultCeiling(t *testing.T) {
	policy, err := cycleApprovalPolicy(0)
	if err != nil {
		t.Fatalf("cycleApprovalPolicy(0): %v", err)
	}
	reference, err := approval.NewApprovalPolicy()
	if err != nil {
		t.Fatalf("NewApprovalPolicy: %v", err)
	}
	if policy.MaxRounds() != reference.MaxRounds() {
		t.Errorf("MaxRounds=%d, want %d (default do dominio)", policy.MaxRounds(), reference.MaxRounds())
	}

	configured, err := cycleApprovalPolicy(3)
	if err != nil {
		t.Fatalf("cycleApprovalPolicy(3): %v", err)
	}
	if configured.MaxRounds() != 3 {
		t.Errorf("MaxRounds=%d, want 3 (sem +1)", configured.MaxRounds())
	}
}

func TestRunLoopCycleIsIterativeNotRecursive(t *testing.T) {
	source, err := os.ReadFile("runloop.go")
	if err != nil {
		t.Fatalf("os.ReadFile(runloop.go): %v", err)
	}
	if regexp.MustCompile(`applyImplementDecisions`).Match(source) {
		t.Error("runloop.go ainda contem a reentrada recursiva applyImplementDecisions (RF-38)")
	}
	if regexp.MustCompile(`NewBugfixLoop\(`).Match(source) {
		t.Error("runloop.go ainda alcanca o BugfixLoop legado fora do Ciclo (RF-31/RF-47)")
	}
}
