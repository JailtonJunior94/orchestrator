package runtime

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/JailtonJunior94/ai-spec-harness/internal/approval"
	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/specs"
)

func workDirWithReviewSkill(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	skillDir := filepath.Join(dir, ".agents", "skills", "review")
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatalf("mkdir skill: %v", err)
	}
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte("# Review\n"), 0o644); err != nil {
		t.Fatalf("write skill: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "AGENTS.md"), []byte("# Agents\n"), 0o644); err != nil {
		t.Fatalf("write agents: %v", err)
	}
	return dir
}

func gitRepoWithReviewSkill(t *testing.T) string {
	t.Helper()
	dir := workDirWithReviewSkill(t)
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
	run("config", "commit.gpgsign", "false")
	run("add", ".")
	run("commit", "-m", "base")
	return dir
}

func runnerWithReviewFn(t *testing.T, fn ReviewOutputFn) *ACPRunner {
	t.Helper()
	return NewACPRunner(specs.NewCatalog().Claude(), NewCatalog().WithReviewOutputFn(fn))
}

func TestEvidenceRoundAddressesAreDistinctAndImmutable(t *testing.T) {
	dir := t.TempDir()
	catalog := NewCatalog()

	first := catalog.roundReviewEvidenceDir(dir, 1)
	second := catalog.roundReviewEvidenceDir(dir, 2)
	if first == second {
		t.Fatalf("round addresses collide: %q", first)
	}

	pathOne, err := catalog.writeRoundReviewEvidence(first, "round one output")
	if err != nil {
		t.Fatalf("write round 1: %v", err)
	}
	pathTwo, err := catalog.writeRoundReviewEvidence(second, "round two output")
	if err != nil {
		t.Fatalf("write round 2: %v", err)
	}
	if pathOne == pathTwo {
		t.Fatalf("evidence files collide: %q", pathOne)
	}

	for want, path := range map[string]string{"round one output": pathOne, "round two output": pathTwo} {
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			t.Fatalf("read %q: %v", path, readErr)
		}
		if string(data) != want {
			t.Fatalf("content of %q = %q, want %q", path, data, want)
		}
	}

	if _, err := catalog.writeRoundReviewEvidence(first, "overwrite attempt"); err == nil {
		t.Fatal("rewriting an existing round address must be an error")
	}
	data, _ := os.ReadFile(pathOne)
	if string(data) != "round one output" {
		t.Fatalf("round 1 evidence was mutated: %q", data)
	}
}

func TestEvidenceRunAutoReviewRoundIsNumberedAndExclusive(t *testing.T) {
	ctx := context.Background()
	workDir := workDirWithReviewSkill(t)
	job := Job{WorkDir: workDir, EvidenceDir: t.TempDir(), Quiet: true}

	runner := runnerWithReviewFn(t, func(_ context.Context, _ Job) (string, error) {
		return "## Review\n\nverdict: APPROVED\n", nil
	})

	roundOne, err := runner.runAutoReviewRound(ctx, job, 1, "")
	if err != nil {
		t.Fatalf("round 1: %v", err)
	}
	roundTwo, err := runner.runAutoReviewRound(ctx, job, 2, "sha-prev")
	if err != nil {
		t.Fatalf("round 2: %v", err)
	}
	if roundOne.Path == roundTwo.Path {
		t.Fatalf("round evidence addresses collide: %q", roundOne.Path)
	}
	for _, path := range []string{roundOne.Path, roundTwo.Path} {
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			t.Fatalf("read %q: %v", path, readErr)
		}
		if !strings.Contains(string(data), "verdict: APPROVED") {
			t.Fatalf("evidence %q missing reviewer output: %q", path, data)
		}
	}

	if _, err := runner.runAutoReviewRound(ctx, job, 1, ""); err == nil {
		t.Fatal("second write to round 1 address must fail")
	}
}

func TestReviewPriorSHAEnvIsScopedToRoundAndRestored(t *testing.T) {
	os.Unsetenv(envReviewPriorSHA)
	ctx := context.Background()
	workDir := workDirWithReviewSkill(t)
	job := Job{WorkDir: workDir, EvidenceDir: t.TempDir(), Quiet: true}

	var seen string
	var present bool
	capture := func(_ context.Context, _ Job) (string, error) {
		seen, present = os.LookupEnv(envReviewPriorSHA)
		return "verdict: APPROVED", nil
	}
	runner := runnerWithReviewFn(t, capture)

	if _, err := runner.runAutoReviewRound(ctx, job, 1, "sha-ignored"); err != nil {
		t.Fatalf("round 1: %v", err)
	}
	if present {
		t.Fatalf("round 1 must not export %s, got %q", envReviewPriorSHA, seen)
	}

	if _, err := runner.runAutoReviewRound(ctx, job, 2, "sha-cutpoint"); err != nil {
		t.Fatalf("round 2: %v", err)
	}
	if !present || seen != "sha-cutpoint" {
		t.Fatalf("round 2 env = (%q, %v), want (\"sha-cutpoint\", true)", seen, present)
	}

	if _, ok := os.LookupEnv(envReviewPriorSHA); ok {
		t.Fatal("prior SHA env leaked past the review session")
	}
}

func TestReviewPriorSHAValueComesFromRepositoryCutPoint(t *testing.T) {
	os.Unsetenv(envReviewPriorSHA)
	ctx := context.Background()
	repoDir := gitRepoWithReviewSkill(t)

	cutPoint, err := NewRepositoryAdapter(repoDir).Checkpoint(ctx)
	if err != nil {
		t.Fatalf("checkpoint: %v", err)
	}

	var seen string
	runner := runnerWithReviewFn(t, func(_ context.Context, _ Job) (string, error) {
		seen = os.Getenv(envReviewPriorSHA)
		return "verdict: APPROVED", nil
	})
	adapter := NewReviewerAdapter(runner, Job{WorkDir: repoDir, EvidenceDir: t.TempDir(), Quiet: true})

	criterion, err := approval.NewAcceptanceCriterion("builds green")
	if err != nil {
		t.Fatalf("criterion: %v", err)
	}
	task, err := approval.NewTaskIdentity("task-4.2")
	if err != nil {
		t.Fatalf("task: %v", err)
	}
	agent, err := approval.NewAgentIdentity("agent-x")
	if err != nil {
		t.Fatalf("agent: %v", err)
	}
	criteria := []approval.AcceptanceCriterion{criterion}

	firstRound, err := approval.NewReviewRequest(task, agent, 1, approval.NewReviewTarget("d1"), criteria)
	if err != nil {
		t.Fatalf("request 1: %v", err)
	}
	if _, err := adapter.Review(ctx, firstRound); err != nil {
		t.Fatalf("review 1: %v", err)
	}
	if seen != "" {
		t.Fatalf("round 1 exported prior SHA %q", seen)
	}

	secondRound, err := approval.NewReviewRequest(task, agent, 2, approval.NewReviewTarget("d2"), criteria)
	if err != nil {
		t.Fatalf("request 2: %v", err)
	}
	if _, err := adapter.Review(ctx, secondRound); err != nil {
		t.Fatalf("review 2: %v", err)
	}
	if seen != cutPoint.String() {
		t.Fatalf("round 2 prior SHA = %q, want %q", seen, cutPoint.String())
	}
}

func TestReviewRoundResetsInvocationDepthWithoutRaisingLimit(t *testing.T) {
	os.Setenv(invocationDepthEnvKey, "1")
	defer os.Unsetenv(invocationDepthEnvKey)
	ctx := context.Background()
	workDir := workDirWithReviewSkill(t)
	job := Job{WorkDir: workDir, EvidenceDir: t.TempDir(), Quiet: true}

	var depthDuringRound string
	var childAutoReview bool
	runner := runnerWithReviewFn(t, func(_ context.Context, child Job) (string, error) {
		depthDuringRound = os.Getenv(invocationDepthEnvKey)
		childAutoReview = child.AutoReview
		return "verdict: APPROVED", nil
	})

	for round := 1; round <= 3; round++ {
		prior := ""
		if round > 1 {
			prior = "sha-prev"
		}
		if _, err := runner.runAutoReviewRound(ctx, job, round, prior); err != nil {
			t.Fatalf("round %d: %v", round, err)
		}
		if depthDuringRound != "0" {
			t.Fatalf("round %d opened with depth %q, want reset to 0", round, depthDuringRound)
		}
		if childAutoReview {
			t.Fatalf("round %d child session has AutoReview enabled (RF-44)", round)
		}
	}

	if got := os.Getenv(invocationDepthEnvKey); got != "1" {
		t.Fatalf("invocation depth not restored after rounds: %q", got)
	}
}

const invocationDepthEnvKey = "AI_INVOCATION_DEPTH"
