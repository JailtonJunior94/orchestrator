package qualitygate

import (
	"context"
	"testing"

	"github.com/JailtonJunior94/ai-spec-harness/internal/detect"
	"github.com/JailtonJunior94/ai-spec-harness/internal/fs"
	"github.com/JailtonJunior94/ai-spec-harness/internal/hookcontract"
)

type fakeToolchainProvider struct {
	result detect.ToolchainResult
}

func (p fakeToolchainProvider) Detect(string) detect.ToolchainResult {
	return p.result
}

type fakeExecutor struct {
	byCommand map[string]ExecutionResult
}

func (e fakeExecutor) Execute(_ context.Context, _ string, command string) (ExecutionResult, error) {
	if result, ok := e.byCommand[command]; ok {
		return result, nil
	}
	return ExecutionResult{Command: command, ExitCode: 0}, nil
}

func newGoToolchain() detect.ToolchainResult {
	return detect.ToolchainResult{
		"go": detect.ToolchainEntry{Fmt: "gofmt -l .", Test: "go test ./...", Lint: "golangci-lint run"},
	}
}

func TestGate_Evaluate_AllowsWhenChecksPass(t *testing.T) {
	fakeFS := fs.NewFakeFileSystem()
	gate := NewGate(
		NewDefaultPolicyLoader(fakeFS),
		fakeToolchainProvider{result: newGoToolchain()},
		fakeExecutor{byCommand: map[string]ExecutionResult{}},
		NewFileCache(fakeFS, "/project/.agents/generated/quality-gate-cache.json"),
		NewFileEvidenceWriter(fakeFS, "/project/.agents/generated/quality-gate-evidence"),
	)

	result := gate.Evaluate(context.Background(), EvaluationInput{
		ProjectDir: "/project",
		TaskID:     "10.0",
		TaskType:   DefaultTaskType,
		Risk:       DefaultRisk,
	})

	if result.Decision() != hookcontract.DecisionAllow {
		t.Fatalf("Decision() = %v, want ALLOW; reason=%q", result.Decision(), result.Reason())
	}
}

func TestGate_Evaluate_RequiredCheckFailureBlocksWithPolicyAndGateID(t *testing.T) {
	fakeFS := fs.NewFakeFileSystem()
	gate := NewGate(
		NewDefaultPolicyLoader(fakeFS),
		fakeToolchainProvider{result: newGoToolchain()},
		fakeExecutor{byCommand: map[string]ExecutionResult{
			"go test ./...": {Command: "go test ./...", ExitCode: 1, Output: "FAIL"},
		}},
		NewFileCache(fakeFS, "/project/.agents/generated/quality-gate-cache.json"),
		NewFileEvidenceWriter(fakeFS, "/project/.agents/generated/quality-gate-evidence"),
	)

	result := gate.Evaluate(context.Background(), EvaluationInput{
		ProjectDir: "/project",
		TaskID:     "10.0",
		TaskType:   DefaultTaskType,
		Risk:       DefaultRisk,
	})

	if result.Decision() != hookcontract.DecisionBlock {
		t.Fatalf("Decision() = %v, want BLOCK", result.Decision())
	}
	if result.PolicyID() == "" {
		t.Fatalf("PolicyID must be filled on BLOCK")
	}
	if result.GateID() != GateID {
		t.Fatalf("GateID() = %q, want %q", result.GateID(), GateID)
	}
}

func TestQualityGate_SkipsWhenFingerprintUnchanged(t *testing.T) {
	fakeFS := fs.NewFakeFileSystem()
	executor := fakeExecutor{byCommand: map[string]ExecutionResult{}}
	gate := NewGate(
		NewDefaultPolicyLoader(fakeFS),
		fakeToolchainProvider{result: newGoToolchain()},
		executor,
		NewFileCache(fakeFS, "/project/.agents/generated/quality-gate-cache.json"),
		NewFileEvidenceWriter(fakeFS, "/project/.agents/generated/quality-gate-evidence"),
	)

	input := EvaluationInput{ProjectDir: "/project", TaskID: "10.0", TaskType: DefaultTaskType, Risk: DefaultRisk}

	first := gate.Evaluate(context.Background(), input)
	if first.Decision() != hookcontract.DecisionAllow {
		t.Fatalf("first Decision() = %v, want ALLOW", first.Decision())
	}

	second := gate.Evaluate(context.Background(), input)
	if second.Decision() != hookcontract.DecisionNotApplicable {
		t.Fatalf("second Decision() = %v, want NOT_APPLICABLE (dedup skip)", second.Decision())
	}
}

func TestQualityGate_ReexecutesWhenFingerprintChanges(t *testing.T) {
	fakeFS := fs.NewFakeFileSystem()
	gate := NewGate(
		NewDefaultPolicyLoader(fakeFS),
		fakeToolchainProvider{result: newGoToolchain()},
		fakeExecutor{byCommand: map[string]ExecutionResult{}},
		NewFileCache(fakeFS, "/project/.agents/generated/quality-gate-cache.json"),
		NewFileEvidenceWriter(fakeFS, "/project/.agents/generated/quality-gate-evidence"),
	)

	first := gate.Evaluate(context.Background(), EvaluationInput{
		ProjectDir: "/project", TaskID: "10.0", TaskType: DefaultTaskType, Risk: DefaultRisk, StateDigest: "v1",
	})
	if first.Decision() != hookcontract.DecisionAllow {
		t.Fatalf("first Decision() = %v, want ALLOW", first.Decision())
	}

	second := gate.Evaluate(context.Background(), EvaluationInput{
		ProjectDir: "/project", TaskID: "10.0", TaskType: DefaultTaskType, Risk: DefaultRisk, StateDigest: "v2",
	})
	if second.Decision() != hookcontract.DecisionAllow {
		t.Fatalf("second Decision() = %v, want ALLOW (re-executed, not skipped)", second.Decision())
	}
	if _, isSkipReason := second.Metadata()["previous_decision"]; isSkipReason {
		t.Fatalf("re-executed result must not carry dedup metadata")
	}
}

func TestGate_Evaluate_UndeclaredPolicyCombinationProducesError(t *testing.T) {
	fakeFS := fs.NewFakeFileSystem()
	gate := NewGate(
		NewDefaultPolicyLoader(fakeFS),
		fakeToolchainProvider{result: newGoToolchain()},
		fakeExecutor{byCommand: map[string]ExecutionResult{}},
		NewFileCache(fakeFS, "/project/.agents/generated/quality-gate-cache.json"),
		NewFileEvidenceWriter(fakeFS, "/project/.agents/generated/quality-gate-evidence"),
	)

	undeclaredTaskType, err := NewTaskType("never-declared-task-type")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	result := gate.Evaluate(context.Background(), EvaluationInput{
		ProjectDir: "/project", TaskID: "10.0", TaskType: undeclaredTaskType, Risk: RiskLow,
	})

	if result.Decision() != hookcontract.DecisionError {
		t.Fatalf("Decision() = %v, want ERROR for undeclared policy combination (no runtime inference)", result.Decision())
	}
}

func TestGate_Evaluate_OptionalCheckFailureWarnsButDoesNotBlock(t *testing.T) {
	fakeFS := fs.NewFakeFileSystem()
	taskType, err := NewTaskType("feature")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	gate := NewGate(
		NewDefaultPolicyLoader(fakeFS),
		fakeToolchainProvider{result: newGoToolchain()},
		fakeExecutor{byCommand: map[string]ExecutionResult{
			"golangci-lint run": {Command: "golangci-lint run", ExitCode: 1, Output: "warnings"},
		}},
		NewFileCache(fakeFS, "/project/.agents/generated/quality-gate-cache.json"),
		NewFileEvidenceWriter(fakeFS, "/project/.agents/generated/quality-gate-evidence"),
	)

	result := gate.Evaluate(context.Background(), EvaluationInput{
		ProjectDir: "/project", TaskID: "10.0", TaskType: taskType, Risk: RiskLow,
	})

	if result.Decision() != hookcontract.DecisionWarn {
		t.Fatalf("Decision() = %v, want WARN (optional check failure never blocks)", result.Decision())
	}
}
