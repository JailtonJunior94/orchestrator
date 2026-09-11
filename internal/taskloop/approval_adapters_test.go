package taskloop

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/JailtonJunior94/ai-spec-harness/internal/approval"
)

func newReviewRequestForPort(t *testing.T, target string, descriptions ...string) approval.ReviewRequest {
	t.Helper()
	task, err := approval.NewTaskIdentity("task-4.3")
	if err != nil {
		t.Fatalf("task identity: %v", err)
	}
	agent, err := approval.NewAgentIdentity("agent-x")
	if err != nil {
		t.Fatalf("agent identity: %v", err)
	}
	var criteria []approval.AcceptanceCriterion
	for _, d := range descriptions {
		c, cErr := approval.NewAcceptanceCriterion(d)
		if cErr != nil {
			t.Fatalf("criterion %q: %v", d, cErr)
		}
		criteria = append(criteria, c)
	}
	request, err := approval.NewReviewRequest(task, agent, 1, approval.NewReviewTarget(target), criteria)
	if err != nil {
		t.Fatalf("review request: %v", err)
	}
	return request
}

func newFixRequestForPort(t *testing.T, target string, findings ...approval.Finding) approval.FixRequest {
	t.Helper()
	task, err := approval.NewTaskIdentity("task-4.3")
	if err != nil {
		t.Fatalf("task identity: %v", err)
	}
	agent, err := approval.NewAgentIdentity("agent-x")
	if err != nil {
		t.Fatalf("agent identity: %v", err)
	}
	request, err := approval.NewFixRequest(task, agent, 1, findings, approval.NewReviewTarget(target))
	if err != nil {
		t.Fatalf("fix request: %v", err)
	}
	return request
}

func TestReviewerPortTranslatesCallAndReturnsRawText(t *testing.T) {
	const raw = "## Review\n\nAll good.\n\nVerdict: REJECTED\n"
	reviewer := &stubFinalReviewer{results: []FinalReviewResult{{
		Verdict:   VerdictApproved,
		RawOutput: raw,
		Findings:  []Finding{{Severity: SeverityCritical, File: "a.go", Line: 10, Message: "boom"}},
	}}}

	port := newReviewerPort(reviewer)
	request := newReviewRequestForPort(t, "diff --git a/a.go b/a.go", "builds green", "tests pass")

	output, err := port.Review(context.Background(), request)
	if err != nil {
		t.Fatalf("Review: %v", err)
	}

	if output.RawText() != raw {
		t.Errorf("RawText = %q, want %q", output.RawText(), raw)
	}
	if reviewer.diffs[0] != "diff --git a/a.go b/a.go" {
		t.Errorf("target passed through = %q", reviewer.diffs[0])
	}

	findings := output.FindingsList()
	if len(findings) != 1 || findings[0].Severity() != approval.SeverityCritical || findings[0].File() != "a.go" {
		t.Fatalf("translated findings = %+v", findings)
	}
	if findings[0].Rule() != defaultFindingRule {
		t.Errorf("rule = %q, want %q", findings[0].Rule(), defaultFindingRule)
	}
}

func TestReviewerPortCriteriaMapReachesComplete(t *testing.T) {
	reviewer := &stubFinalReviewer{results: []FinalReviewResult{{RawOutput: "Verdict: APPROVED\n"}}}
	port := newReviewerPort(reviewer)
	request := newReviewRequestForPort(t, "diff", "criterion one", "criterion two")

	output, err := port.Review(context.Background(), request)
	if err != nil {
		t.Fatalf("Review: %v", err)
	}

	criteriaMap := output.CriteriaMap()
	if !criteriaMap.Complete() {
		t.Fatalf("criteria map not complete: %d/%d", criteriaMap.Bound(), criteriaMap.Total())
	}

	verdict := approval.NewTranslator().Translate(output.RawText())
	if _, proofErr := approval.NewApprovalProof(verdict, criteriaMap); proofErr != nil {
		t.Fatalf("approval proof from parity output: %v", proofErr)
	}
}

func TestReviewerPortDefaultsMissingFindingFile(t *testing.T) {
	reviewer := &stubFinalReviewer{results: []FinalReviewResult{{
		RawOutput: "Verdict: REJECTED\n",
		Findings:  []Finding{{Severity: SeverityImportant, Message: "no file"}},
	}}}
	port := newReviewerPort(reviewer)

	output, err := port.Review(context.Background(), newReviewRequestForPort(t, "diff", "c1"))
	if err != nil {
		t.Fatalf("Review: %v", err)
	}
	findings := output.FindingsList()
	if len(findings) != 1 || findings[0].File() != defaultFindingFile {
		t.Fatalf("expected default file, got %+v", findings)
	}
	if findings[0].Severity() != approval.SeverityMedium {
		t.Errorf("severity = %v, want medium", findings[0].Severity())
	}
}

func TestFixerPortRecordsEvidenceSideChannel(t *testing.T) {
	invoker := &stubBugfixInvoker{outputs: []string{bugfixEvidenceOutput("root cause: nil deref")}}
	recorder := newBugfixEvidenceRecorder()
	port := newFixerPort(invoker, recorder)

	finding, err := approval.NewFinding(approval.SeverityCritical, "a.go", "R-1", "boom")
	if err != nil {
		t.Fatalf("finding: %v", err)
	}
	request := newFixRequestForPort(t, "pure diff", finding)

	if fixErr := port.Fix(context.Background(), request); fixErr != nil {
		t.Fatalf("Fix: %v", fixErr)
	}
	if invoker.diffs[0] != "pure diff" {
		t.Errorf("diff passed through = %q", invoker.diffs[0])
	}
	entries := recorder.Entries()
	if len(entries) != 1 || entries[0].FailBefore == "" || entries[0].PassAfter == "" {
		t.Fatalf("recorder entries = %+v", entries)
	}
}

func TestFixerPortFailsClosedWithoutEvidenceMarkers(t *testing.T) {
	invoker := &stubBugfixInvoker{outputs: []string{"fixed it, trust me"}}
	recorder := newBugfixEvidenceRecorder()
	port := newFixerPort(invoker, recorder)

	finding, err := approval.NewFinding(approval.SeverityCritical, "a.go", "R-1", "boom")
	if err != nil {
		t.Fatalf("finding: %v", err)
	}

	fixErr := port.Fix(context.Background(), newFixRequestForPort(t, "diff", finding))
	if fixErr == nil {
		t.Fatal("expected fail-closed error, got nil")
	}
	if len(recorder.Entries()) != 0 {
		t.Errorf("recorder should stay empty on failure, got %+v", recorder.Entries())
	}
}

func TestRepositoryPortCaptureTargets(t *testing.T) {
	capturer := &stubDiffCapturer{diffs: []string{"full diff", "delta diff"}}
	port := newRepositoryPort(capturer, t.TempDir())

	full, err := port.FullTarget(context.Background())
	if err != nil {
		t.Fatalf("FullTarget: %v", err)
	}
	if full.String() != "full diff" {
		t.Errorf("FullTarget = %q", full.String())
	}

	delta, err := port.Delta(context.Background(), approval.Checkpoint{})
	if err != nil {
		t.Fatalf("Delta: %v", err)
	}
	if delta.String() != "delta diff" {
		t.Errorf("Delta = %q", delta.String())
	}
}

func TestRepositoryPortCheckpointFallsBackToDiffHashWithoutGit(t *testing.T) {
	dir := t.TempDir()
	capturer := &stubDiffCapturer{diffs: []string{"diff without a git repository"}}

	port := newRepositoryPort(capturer, dir)
	checkpoint, err := port.Checkpoint(context.Background())
	if err != nil {
		t.Fatalf("Checkpoint: %v", err)
	}
	if checkpoint.Zero() {
		t.Fatal("checkpoint should not be zero-valued in the fallback path")
	}

	want := sha256.Sum256([]byte("diff without a git repository"))
	if got := checkpoint.String(); got != hex.EncodeToString(want[:]) {
		t.Errorf("checkpoint = %q, want sha256 of the captured diff", got)
	}
}

func TestRepositoryPortCheckpointFromGitRevParse(t *testing.T) {
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
	run("config", "user.email", "t@example.com")
	run("config", "user.name", "T")
	run("config", "commit.gpgsign", "false")
	if err := os.WriteFile(filepath.Join(dir, "f.txt"), []byte("x\n"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	run("add", ".")
	run("commit", "-m", "base")

	port := newRepositoryPort(&stubDiffCapturer{}, dir)
	checkpoint, err := port.Checkpoint(context.Background())
	if err != nil {
		t.Fatalf("Checkpoint: %v", err)
	}
	if checkpoint.Zero() || len(checkpoint.String()) < 7 {
		t.Fatalf("checkpoint = %q", checkpoint.String())
	}
}
