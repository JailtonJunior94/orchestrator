package taskloop

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/JailtonJunior94/ai-spec-harness/internal/approval"
	taskfs "github.com/JailtonJunior94/ai-spec-harness/internal/fs"
	airuntime "github.com/JailtonJunior94/ai-spec-harness/internal/runtime"
)

func fingerprintFromRuntimeAdapter(raw string) approval.Fingerprint {
	return approval.NewFingerprintCalculator().Compute(approval.ParseReviewFindings(raw))
}

func fingerprintFromTaskloopAdapter(t *testing.T, raw string) approval.Fingerprint {
	t.Helper()
	translated, err := translateReviewFindings(NewCatalog().parseFindings(raw))
	if err != nil {
		t.Fatalf("translateReviewFindings: %v", err)
	}
	return approval.NewFingerprintCalculator().Compute(translated)
}

func TestAdapterFingerprintParityBetweenProductionCallSites(t *testing.T) {
	cases := map[string]string{
		"high":       "- [high] [internal/x.go:10] validacao ausente\n",
		"critical":   "- [critical] [internal/x.go:10] validacao ausente\n",
		"medium":     "- [medium] [internal/x.go:10] validacao ausente\n",
		"low":        "- [low] [internal/x.go:10] validacao ausente\n",
		"mixed":      "- [high] [internal/x.go:10] a\n- [critical] [internal/y.go:2] b\n",
		"noLocation": "- [high] sem arquivo declarado\n",
	}

	for name, raw := range cases {
		t.Run(name, func(t *testing.T) {
			runtimeFP := fingerprintFromRuntimeAdapter(raw)
			taskloopFP := fingerprintFromTaskloopAdapter(t, raw)
			if !runtimeFP.Equal(taskloopFP) {
				t.Fatalf("fingerprint divergente entre call sites (RF-43): runtime=%s taskloop=%s", runtimeFP, taskloopFP)
			}
		})
	}
}

func TestTaskloopAdapterKeepsHighDistinctFromCritical(t *testing.T) {
	const high = "- [high] [internal/x.go:10] validacao ausente\n"
	const critical = "- [critical] [internal/x.go:10] validacao ausente\n"

	if fingerprintFromTaskloopAdapter(t, high).Equal(fingerprintFromTaskloopAdapter(t, critical)) {
		t.Fatal("taskloop colapsa high em critical: achado que evolui de high para critical geraria no_convergence prematura")
	}
	if fingerprintFromRuntimeAdapter(high).Equal(fingerprintFromRuntimeAdapter(critical)) {
		t.Fatal("runtime colapsa high em critical")
	}
}

func TestTranslateSeverityRoundTripIsLossless(t *testing.T) {
	for _, severity := range []approval.Severity{
		approval.SeverityLow,
		approval.SeverityMedium,
		approval.SeverityHigh,
		approval.SeverityCritical,
	} {
		if got := translateSeverity(reverseSeverity(severity)); got != severity {
			t.Errorf("round trip de %s devolveu %s", severity, got)
		}
	}
}

type envProbeReviewer struct {
	priorSHA []string
	present  []bool
	depth    []string
	calls    int
}

func (r *envProbeReviewer) ReviewConsolidated(context.Context, string) (FinalReviewResult, error) {
	value, ok := os.LookupEnv("AI_REVIEW_PRIOR_SHA")
	r.priorSHA = append(r.priorSHA, value)
	r.present = append(r.present, ok)
	r.depth = append(r.depth, os.Getenv("AI_INVOCATION_DEPTH"))
	r.calls++
	return FinalReviewResult{RawOutput: "Verdict: REJECTED\n"}, nil
}

type fixedCutPoints struct {
	values map[int]string
}

func (c fixedCutPoints) CheckpointAt(round int) (approval.Checkpoint, bool) {
	value, ok := c.values[round]
	if !ok {
		return approval.Checkpoint{}, false
	}
	checkpoint, err := approval.NewCheckpoint(value)
	if err != nil {
		return approval.Checkpoint{}, false
	}
	return checkpoint, true
}

func parityReviewRequestForRound(t *testing.T, round int, target string) approval.ReviewRequest {
	t.Helper()
	task, err := approval.NewTaskIdentity("task-1.0")
	if err != nil {
		t.Fatalf("NewTaskIdentity: %v", err)
	}
	agent, err := approval.NewAgentIdentity("claude")
	if err != nil {
		t.Fatalf("NewAgentIdentity: %v", err)
	}
	criterion, err := approval.NewAcceptanceCriterion("build green")
	if err != nil {
		t.Fatalf("NewAcceptanceCriterion: %v", err)
	}
	request, err := approval.NewReviewRequest(task, agent, round, approval.NewReviewTarget(target), []approval.AcceptanceCriterion{criterion})
	if err != nil {
		t.Fatalf("NewReviewRequest: %v", err)
	}
	return request
}

func TestTaskloopReviewerPortExportsPriorCutPointFromRoundTwo(t *testing.T) {
	t.Setenv("AI_REVIEW_PRIOR_SHA", "leftover")
	t.Setenv("AI_INVOCATION_DEPTH", "2")

	probe := &envProbeReviewer{}
	port := newReviewerPort(probe, airuntime.NewRoundEvidenceWriter(""), fixedCutPoints{values: map[int]string{1: "abc123"}})

	if _, err := port.Review(context.Background(), parityReviewRequestForRound(t, 1, "diff")); err != nil {
		t.Fatalf("Review round 1: %v", err)
	}
	if _, err := port.Review(context.Background(), parityReviewRequestForRound(t, 2, "diff")); err != nil {
		t.Fatalf("Review round 2: %v", err)
	}

	if probe.present[0] {
		t.Errorf("rodada 1 nao pode exportar AI_REVIEW_PRIOR_SHA, obteve %q", probe.priorSHA[0])
	}
	if probe.priorSHA[1] != "abc123" {
		t.Errorf("rodada 2 AI_REVIEW_PRIOR_SHA = %q, want abc123 (RF-39)", probe.priorSHA[1])
	}
	for round, depth := range probe.depth {
		if depth != "0" {
			t.Errorf("rodada %d abriu com AI_INVOCATION_DEPTH=%q, want 0 (RF-38)", round+1, depth)
		}
	}
	if got := os.Getenv("AI_REVIEW_PRIOR_SHA"); got != "leftover" {
		t.Errorf("AI_REVIEW_PRIOR_SHA nao restaurado: %q", got)
	}
	if got := os.Getenv("AI_INVOCATION_DEPTH"); got != "2" {
		t.Errorf("AI_INVOCATION_DEPTH nao restaurado: %q", got)
	}
}

type depthProbeBugfixInvoker struct {
	depth string
}

func (b *depthProbeBugfixInvoker) InvokeBugfix(context.Context, []Finding, string) (string, error) {
	b.depth = os.Getenv("AI_INVOCATION_DEPTH")
	return "root cause: x\nFail-before: go test ./... FAIL\nPass-after: go test ./... ok\n", nil
}

func TestTaskloopFixerPortResetsInvocationDepth(t *testing.T) {
	t.Setenv("AI_INVOCATION_DEPTH", "2")

	invoker := &depthProbeBugfixInvoker{}
	port := newFixerPort(invoker, newBugfixEvidenceRecorder())

	task, err := approval.NewTaskIdentity("task-1.0")
	if err != nil {
		t.Fatalf("NewTaskIdentity: %v", err)
	}
	agent, err := approval.NewAgentIdentity("claude")
	if err != nil {
		t.Fatalf("NewAgentIdentity: %v", err)
	}
	finding, err := approval.NewFinding(approval.SeverityHigh, "a.go:1", "R-1", "boom")
	if err != nil {
		t.Fatalf("NewFinding: %v", err)
	}
	request, err := approval.NewFixRequest(task, agent, 1, []approval.Finding{finding}, approval.NewReviewTarget("diff"))
	if err != nil {
		t.Fatalf("NewFixRequest: %v", err)
	}

	if err := port.Fix(context.Background(), request); err != nil {
		t.Fatalf("Fix: %v", err)
	}
	if invoker.depth != "0" {
		t.Errorf("bugfix rodou com AI_INVOCATION_DEPTH=%q, want 0 (RF-38)", invoker.depth)
	}
	if got := os.Getenv("AI_INVOCATION_DEPTH"); got != "2" {
		t.Errorf("AI_INVOCATION_DEPTH nao restaurado: %q", got)
	}
}

func TestPrimedReviewerValidatesCriteriaAgainstItsOwnTarget(t *testing.T) {
	primedTarget := "Contexto da revisao consolidada:\n" +
		"diff --git a/internal/x.go b/internal/x.go\n" +
		"--- a/internal/x.go\n+++ b/internal/x.go\n@@ -10,2 +10,3 @@\n+linha nova\n"
	repositoryTarget := "diff --git a/outro.go b/outro.go\n" +
		"--- a/outro.go\n+++ b/outro.go\n@@ -1,1 +1,2 @@\n+outra coisa\n"

	primed := FinalReviewResult{
		Verdict:   VerdictApproved,
		RawOutput: "Verdict: APPROVED\n\n## Mapa de Critérios de Aceite\n- [atendido] build green -> internal/x.go:10\n",
	}

	port := newPrimedReviewerPort(primed, primedTarget, &envProbeReviewer{}, airuntime.NewRoundEvidenceWriter(""), nil)
	output, err := port.Review(context.Background(), parityReviewRequestForRound(t, 1, repositoryTarget))
	if err != nil {
		t.Fatalf("Review: %v", err)
	}
	if _, err := approval.NewApprovalProof(approval.VerdictApproved, output.CriteriaMap()); err != nil {
		t.Fatalf("RF-48(b) avaliado contra o alvo errado: %v", err)
	}
}

func newTaskloopGitRepoWithUpstream(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	run := func(args ...string) string {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, out)
		}
		return strings.TrimSpace(string(out))
	}
	run("init")
	run("config", "user.email", "test@example.com")
	run("config", "user.name", "Test")
	run("config", "commit.gpgsign", "false")
	if err := os.WriteFile(filepath.Join(dir, "base.txt"), []byte("base\n"), 0o644); err != nil {
		t.Fatalf("write base.txt: %v", err)
	}
	run("add", ".")
	run("commit", "-m", "base")
	run("branch", "baseline")

	if err := os.WriteFile(filepath.Join(dir, "committed.go"), []byte("package main\n"), 0o644); err != nil {
		t.Fatalf("write committed.go: %v", err)
	}
	run("add", ".")
	run("commit", "-m", "trabalho do executor")

	branch := run("symbolic-ref", "--short", "HEAD")
	run("config", "branch."+branch+".remote", ".")
	run("config", "branch."+branch+".merge", "refs/heads/baseline")
	return dir
}

func TestCaptureGitDiffIncludesCommittedWorkSinceBaseRef(t *testing.T) {
	dir := newTaskloopGitRepoWithUpstream(t)

	diff := NewCatalog().captureGitDiff(context.Background(), dir)
	if diff == diffUnavailable {
		t.Fatalf("captureGitDiff cego a commits: se o executor commitou, a rodada 1 revisa nada")
	}
	if !strings.Contains(diff, "committed.go") {
		t.Fatalf("captureGitDiff nao inclui o trabalho commitado desde o ponto de corte:\n%s", diff)
	}
}

func TestForceTaskStatusFailsLoudWhenArtifactsAreMissing(t *testing.T) {
	fsys := taskfs.NewFakeFileSystem()

	if err := NewCatalog().writeTaskFileStatus("/fake/prd/task-1.0.md", statusBlocked, fsys); err == nil {
		t.Error("writeTaskFileStatus silenciou a ausencia do arquivo de task (RF-36)")
	}

	fsys.Files["/fake/prd/task-1.0.md"] = []byte("# Task sem campo de status\n")
	if err := NewCatalog().writeTaskFileStatus("/fake/prd/task-1.0.md", statusBlocked, fsys); err == nil {
		t.Error("writeTaskFileStatus silenciou a ausencia do campo **Status:** (RF-36)")
	}

	if err := NewCatalog().writeTasksTableStatus("/fake/prd/tasks.md", "1.0", statusBlocked, fsys); err == nil {
		t.Error("writeTasksTableStatus silenciou a ausencia de tasks.md (RF-36)")
	}

	fsys.Files["/fake/prd/tasks.md"] = []byte("| # | Título | Status | Dependências |\n|---|---|---|---|\n| 9.9 | Outra | done | — |\n")
	if err := NewCatalog().writeTasksTableStatus("/fake/prd/tasks.md", "1.0", statusBlocked, fsys); err == nil {
		t.Error("writeTasksTableStatus silenciou a ausencia da linha da task (RF-36)")
	}
}
