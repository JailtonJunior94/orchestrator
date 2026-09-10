package runtime_test

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/JailtonJunior94/ai-spec-harness/internal/approval"
	airuntime "github.com/JailtonJunior94/ai-spec-harness/internal/runtime"
	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/acpfake"
	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/specs"
)

func gitInitRepoForAdapter(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	run := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, out)
		}
	}
	run("init")
	run("config", "user.email", "test@example.com")
	run("config", "user.name", "Test")
	if err := os.WriteFile(filepath.Join(dir, "base.txt"), []byte("base\n"), 0o644); err != nil {
		t.Fatalf("write base: %v", err)
	}
	run("add", ".")
	run("commit", "-m", "base")
	return dir
}

func TestRepositoryAdapterCheckpointAndTargets(t *testing.T) {
	t.Parallel()

	dir := gitInitRepoForAdapter(t)
	adapter := airuntime.NewRepositoryAdapter(dir)
	ctx := context.Background()

	checkpoint, err := adapter.Checkpoint(ctx)
	if err != nil {
		t.Fatalf("Checkpoint: %v", err)
	}
	if checkpoint.Zero() || len(checkpoint.String()) < 7 {
		t.Fatalf("Checkpoint returned %q", checkpoint.String())
	}

	if err := os.WriteFile(filepath.Join(dir, "base.txt"), []byte("base\nchanged\n"), 0o644); err != nil {
		t.Fatalf("mutate: %v", err)
	}

	full, err := adapter.FullTarget(ctx)
	if err != nil {
		t.Fatalf("FullTarget: %v", err)
	}
	if !strings.Contains(full.String(), "changed") {
		t.Errorf("FullTarget = %q, want diff containing 'changed'", full.String())
	}

	delta, err := adapter.Delta(ctx, checkpoint)
	if err != nil {
		t.Fatalf("Delta: %v", err)
	}
	if !strings.Contains(delta.String(), "changed") {
		t.Errorf("Delta = %q, want diff containing 'changed'", delta.String())
	}
}

func TestReviewerAdapterReturnsRawReviewerText(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	const rawText = "## Review\n\nOne blocker remains.\n\nverdict: REJECTED\n"
	reviewFn := func(_ context.Context, _ airuntime.Job) (string, error) {
		return rawText, nil
	}

	runner := airuntime.NewACPRunner(specs.NewCatalog().Claude(),
		airuntime.NewCatalog().WithProber(&fakeProberForReview{}),
		airuntime.NewCatalog().WithReviewOutputFn(reviewFn),
	)

	baseJob := airuntime.Job{
		WorkDir:     workDirWithAgentsMDForReview(t),
		EvidenceDir: t.TempDir(),
		Quiet:       true,
	}
	adapter := airuntime.NewReviewerAdapter(runner, baseJob)

	criterion, err := approval.NewAcceptanceCriterion("builds green")
	if err != nil {
		t.Fatalf("criterion: %v", err)
	}
	task, err := approval.NewTaskIdentity("task-4.1")
	if err != nil {
		t.Fatalf("task: %v", err)
	}
	agent, err := approval.NewAgentIdentity("agent-x")
	if err != nil {
		t.Fatalf("agent: %v", err)
	}
	request, err := approval.NewReviewRequest(task, agent, 1, approval.NewReviewTarget("diff --git a/x b/x"), []approval.AcceptanceCriterion{criterion})
	if err != nil {
		t.Fatalf("request: %v", err)
	}

	output, err := adapter.Review(ctx, request)
	if err != nil {
		t.Fatalf("Review: %v", err)
	}
	if output.RawText() != rawText {
		t.Errorf("RawText() = %q, want %q", output.RawText(), rawText)
	}
	if approval.NewTranslator().Translate(output.RawText()) != approval.VerdictRejected {
		t.Errorf("translated verdict = %v, want REJECTED", approval.NewTranslator().Translate(output.RawText()))
	}
}

func TestFixerAdapterRunsSessionWithFindings(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	script := acpfake.NewScript().
		AppendAgentMessage("applied fix").
		AppendSessionEnd()

	runner := airuntime.NewACPRunner(specs.NewCatalog().Claude(),
		airuntime.NewCatalog().WithProber(&fakeProberForReview{}),
		airuntime.NewCatalog().WithClientFactory(&fakeClientFactoryForReview{script: script, ctx: ctx, t: t}),
		airuntime.NewCatalog().WithPersistenceFactory(&fakePersistenceFactoryForReview{}),
		airuntime.NewCatalog().WithRenderer(&fakeRendererForReview{}),
	)

	baseJob := airuntime.Job{
		WorkDir:     workDirWithAgentsMDForReview(t),
		EvidenceDir: t.TempDir(),
		Quiet:       true,
	}
	adapter := airuntime.NewFixerAdapter(runner, baseJob)

	finding, err := approval.NewFinding(approval.SeverityHigh, "internal/x/y.go", "R-STYLE-001", "comment on new line")
	if err != nil {
		t.Fatalf("finding: %v", err)
	}
	task, err := approval.NewTaskIdentity("task-4.1")
	if err != nil {
		t.Fatalf("task: %v", err)
	}
	agent, err := approval.NewAgentIdentity("agent-x")
	if err != nil {
		t.Fatalf("agent: %v", err)
	}
	request, err := approval.NewFixRequest(task, agent, 1, []approval.Finding{finding}, approval.NewReviewTarget("diff"))
	if err != nil {
		t.Fatalf("request: %v", err)
	}

	if err := adapter.Fix(ctx, request); err != nil {
		t.Fatalf("Fix: %v", err)
	}
}
