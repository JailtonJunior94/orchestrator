package runtime_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/JailtonJunior94/ai-spec-harness/internal/approval"
	airuntime "github.com/JailtonJunior94/ai-spec-harness/internal/runtime"
	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/specs"
)

func TestReviewPromptInstructsTheCanonicalSeverityTaxonomy(t *testing.T) {
	t.Parallel()

	prompt := airuntime.NewCatalog().BuildReviewPromptForTest("# Review Skill", "diff --git a/x b/x")

	for _, marker := range []string{"[CRITICAL]", "[HIGH]", "[MEDIUM]", "[LOW]"} {
		if !strings.Contains(prompt, marker) {
			t.Errorf("prompt does not instruct canonical marker %s", marker)
		}
	}
}

func TestReviewPromptMarkersAreParsedIntoFindings(t *testing.T) {
	t.Parallel()

	prompt := airuntime.NewCatalog().BuildReviewPromptForTest("# Review Skill", "diff")

	for _, marker := range []string{"[CRITICAL]", "[HIGH]", "[MEDIUM]", "[LOW]"} {
		if !strings.Contains(prompt, marker) {
			t.Fatalf("prompt does not instruct marker %s", marker)
		}
		findings := approval.ParseReviewFindings(marker + " internal/x/a.go:12 broken invariant\n")
		if len(findings) != 1 {
			t.Errorf("marker %s produced %d findings, want 1", marker, len(findings))
		}
	}
}

func TestReviewOutputWithoutAnyMarkerStaysFailClosed(t *testing.T) {
	t.Parallel()

	raw := "A implementacao tem um problema serio no handler, mas nao vou marcar severidade.\nverdict: REJECTED\n"
	if findings := approval.ParseReviewFindings(raw); len(findings) != 0 {
		t.Errorf("unmarked review text produced %d findings, want 0", len(findings))
	}
}

func TestReviewerAdapterAddressesEvidenceDirPerRound(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	const rawText = "## Review\n\n[HIGH] internal/x/a.go:12 broken invariant\n\nverdict: REJECTED\n\n## Mapa de Critérios de Aceite\n- [atendido] builds green -> go test ./... -> PASS\n"

	var seen []string
	reviewFn := func(_ context.Context, job airuntime.Job) (string, error) {
		seen = append(seen, job.EvidenceDir)
		return rawText, nil
	}

	runner := airuntime.NewACPRunner(specs.NewCatalog().Claude(),
		airuntime.NewCatalog().WithProber(&fakeProberForReview{}),
		airuntime.NewCatalog().WithReviewOutputFn(reviewFn),
	)

	baseDir := t.TempDir()
	adapter := airuntime.NewReviewerAdapter(runner, airuntime.Job{
		WorkDir:     workDirWithAgentsMDForReview(t),
		EvidenceDir: baseDir,
		Quiet:       true,
	})

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

	for _, round := range []int{1, 2} {
		request, err := approval.NewReviewRequest(task, agent, round, approval.NewReviewTarget("diff --git a/x b/x"), []approval.AcceptanceCriterion{criterion})
		if err != nil {
			t.Fatalf("request round %d: %v", round, err)
		}
		if _, err := adapter.Review(ctx, request); err != nil {
			t.Fatalf("Review round %d: %v", round, err)
		}
	}

	want := []string{
		baseDir + "/review/round-1",
		baseDir + "/review/round-2",
	}
	if len(seen) != len(want) {
		t.Fatalf("captured %d evidence dirs, want %d", len(seen), len(want))
	}
	for i, dir := range want {
		if seen[i] != dir {
			t.Errorf("round %d evidence dir = %q, want %q", i+1, seen[i], dir)
		}
	}
}

func TestExtractHardIssuesCoversTheCanonicalBlockingMarkers(t *testing.T) {
	t.Parallel()

	raw := "[CRITICAL] internal/x/a.go:12 eval detectado\n" +
		"[HIGH] internal/x/b.go:7 validação ausente\n" +
		"[MEDIUM] internal/x/c.go:3 nome confuso\n"

	issues := airuntime.NewCatalog().ExtractHardIssuesForTest(raw)

	if len(issues) != 2 {
		t.Fatalf("issues bloqueantes = %d (%v), quero 2", len(issues), issues)
	}
	if !strings.Contains(issues[0], "[CRITICAL]") || !strings.Contains(issues[1], "[HIGH]") {
		t.Errorf("issues = %v, quero a linha [CRITICAL] e a linha [HIGH]", issues)
	}
}
