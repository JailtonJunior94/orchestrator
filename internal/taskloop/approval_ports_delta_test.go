package taskloop

import (
	"context"
	"os/exec"
	"strings"
	"testing"

	"github.com/JailtonJunior94/ai-spec-harness/internal/approval"
)

type sequencedDiffCapturer struct {
	diffs   []string
	current int
}

func (c *sequencedDiffCapturer) CaptureDiff(context.Context) (string, error) {
	diff := c.diffs[c.current]
	if c.current < len(c.diffs)-1 {
		c.current++
	}
	return diff, nil
}

func TestRepositoryPortContentCheckpointDetectsUnchangedTree(t *testing.T) {
	ctx := context.Background()
	port := newRepositoryPort(&sequencedDiffCapturer{diffs: []string{"same-diff"}}, t.TempDir())

	checkpoint, err := port.Checkpoint(ctx)
	if err != nil {
		t.Fatalf("checkpoint: %v", err)
	}
	delta, err := port.Delta(ctx, checkpoint)
	if err != nil {
		t.Fatalf("delta: %v", err)
	}
	if !delta.Empty() {
		t.Fatalf("delta = %q, want empty: a fix that changed nothing must abort the cycle (RF-31)", delta.String())
	}
}

func TestRepositoryPortContentCheckpointReportsChangedTree(t *testing.T) {
	ctx := context.Background()
	port := newRepositoryPort(&sequencedDiffCapturer{diffs: []string{"before-fix", "after-fix"}}, t.TempDir())

	checkpoint, err := port.Checkpoint(ctx)
	if err != nil {
		t.Fatalf("checkpoint: %v", err)
	}
	delta, err := port.Delta(ctx, checkpoint)
	if err != nil {
		t.Fatalf("delta: %v", err)
	}
	if delta.Empty() {
		t.Fatal("delta vazio apesar de o fix ter alterado a arvore")
	}
	if delta.String() != "after-fix" {
		t.Errorf("delta = %q, want after-fix", delta.String())
	}
}

func initGitRepoForPorts(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	commands := [][]string{
		{"init"},
		{"config", "user.email", "test@example.com"},
		{"config", "user.name", "Test"},
		{"config", "commit.gpgsign", "false"},
		{"add", "."},
		{"commit", "--allow-empty", "-m", "base"},
	}
	for _, args := range commands {
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, out)
		}
	}
	return dir
}

func TestRepositoryPortGitCheckpointStillUsesGitDelta(t *testing.T) {
	ctx := context.Background()
	dir := initGitRepoForPorts(t)
	port := newRepositoryPort(&sequencedDiffCapturer{diffs: []string{"fallback-diff"}}, dir)

	checkpoint, err := port.Checkpoint(ctx)
	if err != nil {
		t.Fatalf("checkpoint: %v", err)
	}
	if _, err := approval.NewCheckpoint(checkpoint.String()); err != nil {
		t.Fatalf("checkpoint invalido: %v", err)
	}
	delta, err := port.Delta(ctx, checkpoint)
	if err != nil {
		t.Fatalf("delta: %v", err)
	}
	if !delta.Empty() {
		t.Errorf("delta = %q, want empty para HEAD sem alteracoes", delta.String())
	}
}
