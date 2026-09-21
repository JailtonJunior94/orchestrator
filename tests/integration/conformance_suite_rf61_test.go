//go:build integration

package integration

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/JailtonJunior94/ai-spec-harness/internal/capability"
	"github.com/JailtonJunior94/ai-spec-harness/internal/detect"
	"github.com/JailtonJunior94/ai-spec-harness/internal/fs"
	"github.com/JailtonJunior94/ai-spec-harness/internal/hookcontract"
	"github.com/JailtonJunior94/ai-spec-harness/internal/qualitygate"
	"github.com/JailtonJunior94/ai-spec-harness/internal/sdd"
	"github.com/JailtonJunior94/ai-spec-harness/internal/skills"
	"github.com/JailtonJunior94/ai-spec-harness/internal/telemetry"
)

type rf61FakeExecutor struct {
	byCommand map[string]qualitygate.ExecutionResult
}

func (e rf61FakeExecutor) Execute(_ context.Context, _ string, command string) (qualitygate.ExecutionResult, error) {
	if result, ok := e.byCommand[command]; ok {
		return result, nil
	}
	return qualitygate.ExecutionResult{Command: command, ExitCode: 0}, nil
}

type rf61FakeToolchain struct {
	result detect.ToolchainResult
}

func (p rf61FakeToolchain) Detect(string) detect.ToolchainResult {
	return p.result
}

func rf61GoToolchain() detect.ToolchainResult {
	return detect.ToolchainResult{
		"go": detect.ToolchainEntry{Fmt: "gofmt -l .", Test: "go test ./...", Lint: "golangci-lint run"},
	}
}

func TestConformanceSuite_RF61Scenario05_MandatoryTestFailingBlocksCoreValidatorInvariantPerProvider(t *testing.T) {
	for _, tool := range conformanceCanonicalTools {
		t.Run(string(tool), func(t *testing.T) {
			fakeFS := fs.NewFakeFileSystem()
			gate := qualitygate.NewGate(
				qualitygate.NewDefaultPolicyLoader(fakeFS),
				rf61FakeToolchain{result: rf61GoToolchain()},
				rf61FakeExecutor{byCommand: map[string]qualitygate.ExecutionResult{
					"go test ./...": {Command: "go test ./...", ExitCode: 1, Output: "FAIL"},
				}},
				qualitygate.NewFileCache(fakeFS, "/project/.agents/generated/quality-gate-cache.json"),
				qualitygate.NewFileEvidenceWriter(fakeFS, "/project/.agents/generated/quality-gate-evidence"),
			)

			blocked := gate.Evaluate(context.Background(), qualitygate.EvaluationInput{
				ProjectDir: "/project", TaskID: "rf61-5", TaskType: qualitygate.DefaultTaskType, Risk: qualitygate.DefaultRisk,
			})
			if blocked.Decision() != hookcontract.DecisionBlock {
				t.Fatalf("provider=%s: a mandatory test failure must BLOCK the quality gate; got decision=%v reason=%q", tool, blocked.Decision(), blocked.Reason())
			}
			if blocked.PolicyID() == "" || blocked.GateID() != qualitygate.GateID {
				t.Fatalf("provider=%s: BLOCK verdict must carry policy_id and gate_id; policy_id=%q gate_id=%q", tool, blocked.PolicyID(), blocked.GateID())
			}

			passingGate := qualitygate.NewGate(
				qualitygate.NewDefaultPolicyLoader(fakeFS),
				rf61FakeToolchain{result: rf61GoToolchain()},
				rf61FakeExecutor{byCommand: map[string]qualitygate.ExecutionResult{}},
				qualitygate.NewFileCache(fakeFS, "/project2/.agents/generated/quality-gate-cache.json"),
				qualitygate.NewFileEvidenceWriter(fakeFS, "/project2/.agents/generated/quality-gate-evidence"),
			)
			allowed := passingGate.Evaluate(context.Background(), qualitygate.EvaluationInput{
				ProjectDir: "/project2", TaskID: "rf61-5", TaskType: qualitygate.DefaultTaskType, Risk: qualitygate.DefaultRisk,
			})
			if allowed.Decision() != hookcontract.DecisionAllow {
				t.Fatalf("provider=%s: gate-of-the-gate — a passing suite must ALLOW, discriminating it from the failing case above; got decision=%v", tool, allowed.Decision())
			}
		})
	}
}

func TestConformanceSuite_RF61Scenario06_MissingEvidenceBlocksAcrossFourProviders(t *testing.T) {
	for tool, relPath := range sessionEndHookRelPath {
		t.Run(string(tool), func(t *testing.T) {
			projectDir := installMandatoryAgentsForDispatch(t)
			writeSessionEndTasksFixture(t, projectDir, "in_progress", "")

			out, exitCode := runHookScript(t, projectDir, relPath, sessionEndNativePayload[tool])
			if exitCode != sessionEndCanonicalScriptExitCode[tool] {
				t.Fatalf("provider=%s: an active task with no execution_report.md at all must BLOCK session end (evidence-gate dispatched for real through %s); got exit=%d output=%s", tool, relPath, exitCode, out)
			}

			writeSessionEndTasksFixture(t, projectDir, "in_progress", "APPROVED")
			allowedOut, allowedExit := runHookScript(t, projectDir, relPath, sessionEndNativePayload[tool])
			if allowedExit != 0 {
				t.Fatalf("provider=%s: gate-of-the-gate — writing an APPROVED report for the same task must flip the same dispatched script to ALLOW; got exit=%d output=%s", tool, allowedExit, allowedOut)
			}
		})
	}

	nodePath := detectNode(t)
	pluginPath := pluginAssetPath(t)
	t.Run(string(skills.ToolOpenCode), func(t *testing.T) {
		dir := t.TempDir()
		writeSessionEndGateFixture(t, dir)
		writeSessionEndTasksFixture(t, dir, "in_progress", "")

		out, exitCode := runNodeHarness(t, nodePath, opencodeSessionEndHarness, pluginPath, dir)
		if exitCode == 0 {
			t.Fatalf("opencode: session.idle must block an active task with no execution_report.md at all, dispatched through the real governance.js plugin; output=%s", out)
		}

		writeSessionEndTasksFixture(t, dir, "in_progress", "APPROVED")
		allowedOut, allowedExit := runNodeHarness(t, nodePath, opencodeSessionEndHarness, pluginPath, dir)
		if allowedExit != 0 {
			t.Fatalf("opencode: gate-of-the-gate — an APPROVED report for the same task must flip the same plugin dispatch to ALLOW; output=%s", allowedOut)
		}
	})
}

func TestConformanceSuite_RF61Scenario07_InvalidEvidenceBlocksAcrossFourProviders(t *testing.T) {
	for tool, relPath := range sessionEndHookRelPath {
		t.Run(string(tool), func(t *testing.T) {
			projectDir := installMandatoryAgentsForDispatch(t)
			writeSessionEndTasksFixture(t, projectDir, "in_progress", "needs_input")

			out, exitCode := runHookScript(t, projectDir, relPath, sessionEndNativePayload[tool])
			if exitCode != sessionEndCanonicalScriptExitCode[tool] {
				t.Fatalf("provider=%s: an execution_report.md present but without an APPROVED verdict is invalid evidence and must BLOCK, dispatched for real through %s; got exit=%d output=%s", tool, relPath, exitCode, out)
			}

			writeSessionEndTasksFixture(t, projectDir, "in_progress", "APPROVED")
			allowedOut, allowedExit := runHookScript(t, projectDir, relPath, sessionEndNativePayload[tool])
			if allowedExit != 0 {
				t.Fatalf("provider=%s: gate-of-the-gate — replacing the invalid verdict with APPROVED on the same dispatched script must ALLOW; got exit=%d output=%s", tool, allowedExit, allowedOut)
			}
		})
	}

	nodePath := detectNode(t)
	pluginPath := pluginAssetPath(t)
	t.Run(string(skills.ToolOpenCode), func(t *testing.T) {
		dir := t.TempDir()
		writeSessionEndGateFixture(t, dir)
		writeSessionEndTasksFixture(t, dir, "in_progress", "needs_input")

		out, exitCode := runNodeHarness(t, nodePath, opencodeSessionEndHarness, pluginPath, dir)
		if exitCode == 0 {
			t.Fatalf("opencode: an execution_report.md present but without an APPROVED verdict must block session.idle, dispatched through the real governance.js plugin; output=%s", out)
		}

		writeSessionEndTasksFixture(t, dir, "in_progress", "APPROVED")
		allowedOut, allowedExit := runNodeHarness(t, nodePath, opencodeSessionEndHarness, pluginPath, dir)
		if allowedExit != 0 {
			t.Fatalf("opencode: gate-of-the-gate — an APPROVED verdict on the same task must flip the same plugin dispatch to ALLOW; output=%s", allowedOut)
		}
	})
}

const rf61ValidCheckpointJSON = `{"schema_version":2,"run_id":"run-1","task_id":"13.0","attempt":1,"status":"blocked","base_sha":"0123456789012345678901234567890123456789","patch_sha256":"0123456789012345678901234567890123456789012345678901234567890123","final_state_sha256":"0123456789012345678901234567890123456789012345678901234567890123","coverage_regression":false,"tests":[{"command":"go test ./...","exit_code":1,"output_sha256":"0123456789012345678901234567890123456789012345678901234567890123"}],"criteria":[{"id":"AC-01","evidence_ref":"report.md#criterion"}],"evidence":["report.md"],"review_verdict":"needs_input"}`

const rf61CorruptedCheckpointJSON = `{"schema_version":2,"run_id":"run-1","task_id":"13.0","status":"done", this is not valid json`

func TestConformanceSuite_RF61Scenario08_ValidCheckpointPassesCoreValidatorInvariantPerProvider(t *testing.T) {
	for _, tool := range conformanceCanonicalTools {
		t.Run(string(tool), func(t *testing.T) {
			result, err := sdd.NewResultValidator().ValidateCheckpointJSON([]byte(rf61ValidCheckpointJSON))
			if err != nil {
				t.Fatalf("provider=%s: a well-formed schema_version 2 checkpoint must validate; got err=%v", tool, err)
			}
			if result.TaskID != "13.0" {
				t.Fatalf("provider=%s: validated checkpoint must preserve task_id; got %q", tool, result.TaskID)
			}
		})
	}
}

func TestConformanceSuite_RF61Scenario09_CorruptedCheckpointBlocksCoreValidatorInvariantPerProvider(t *testing.T) {
	for _, tool := range conformanceCanonicalTools {
		t.Run(string(tool), func(t *testing.T) {
			_, err := sdd.NewResultValidator().ValidateCheckpointJSON([]byte(rf61CorruptedCheckpointJSON))
			if err == nil {
				t.Fatalf("provider=%s: a syntactically corrupted checkpoint must never be silently accepted; gate-of-the-gate against the valid checkpoint of scenario 08", tool)
			}
		})
	}
}

func TestConformanceSuite_RF61Scenario10_TelemetryUnavailableTokenCountIsExplicitlyUnknownCoreValidatorInvariantPerProvider(t *testing.T) {
	for _, tool := range conformanceCanonicalTools {
		t.Run(string(tool), func(t *testing.T) {
			unavailable, err := telemetry.ResolveTokenMetric("", "")
			if err != nil {
				t.Fatalf("provider=%s: an absent token count must resolve to %q, never error; got err=%v", tool, telemetry.UnknownMetricValue, err)
			}
			if unavailable != telemetry.UnknownMetricValue {
				t.Fatalf("provider=%s: absent token count must resolve to explicit %q; got %q", tool, telemetry.UnknownMetricValue, unavailable)
			}

			available, err := telemetry.ResolveTokenMetric("1234", "acp-usage-reported")
			if err != nil {
				t.Fatalf("provider=%s: an available token count with a calculation method must not error; got err=%v", tool, err)
			}
			if available == telemetry.UnknownMetricValue {
				t.Fatalf("provider=%s: gate-of-the-gate — an available value must never collapse to %q", tool, telemetry.UnknownMetricValue)
			}
		})
	}
}

func TestConformanceSuite_RF61Scenario11_UnknownEventIsRejectedCoreValidatorInvariantPerProvider(t *testing.T) {
	for _, tool := range conformanceCanonicalTools {
		t.Run(string(tool), func(t *testing.T) {
			_, err := hookcontract.ParseEventKind("does_not_exist_event")
			if !errors.Is(err, hookcontract.ErrUnknownEvent) {
				t.Fatalf("provider=%s: an unrecognized event name must produce ErrUnknownEvent, never a silent zero-value; got err=%v", tool, err)
			}

			known, err := hookcontract.ParseEventKind("before_tool")
			if err != nil {
				t.Fatalf("provider=%s: gate-of-the-gate — a known event name must parse without error; got err=%v", tool, err)
			}
			if !known.Valid() {
				t.Fatalf("provider=%s: a successfully parsed known event must be Valid()", tool)
			}
		})
	}
}

func TestConformanceSuite_RF61Scenario12_UnsupportedCapabilityIsDeclaredExplicitlyPerProvider(t *testing.T) {
	matrix, err := capability.GenerateHooks()
	if err != nil {
		t.Fatalf("generate hook capability matrix: %v", err)
	}

	for _, tool := range conformanceCanonicalTools {
		t.Run(string(tool), func(t *testing.T) {
			var sawUnsupported, sawOtherState bool
			for _, cell := range matrix.Cells {
				if cell.Provider != string(tool) {
					continue
				}
				if cell.State == hookcontract.SupportUnsupported.String() {
					sawUnsupported = true
					if strings.TrimSpace(cell.Reason) == "" {
						t.Fatalf("provider=%s: cell event=%s family=%s is unsupported but carries no reason — unsupported must be a declared state, never a silent absence", tool, cell.Event, cell.Family)
					}
					continue
				}
				sawOtherState = true
			}
			if !sawUnsupported {
				t.Fatalf("provider=%s: expected at least one explicitly unsupported event/family cell in the generated hook capability matrix", tool)
			}
			if !sawOtherState {
				t.Fatalf("provider=%s: gate-of-the-gate — expected at least one non-unsupported cell too, discriminating the unsupported cells from the rest of the matrix", tool)
			}
		})
	}
}

var conformanceGitPreToolHookRelPath = map[skills.Tool]string{
	skills.ToolClaude:  filepath.Join(".claude", "hooks", "validate-preload.sh"),
	skills.ToolCodex:   filepath.Join(".codex", "hooks", "validate-preload.sh"),
	skills.ToolCopilot: filepath.Join(".github", "hooks", "validate-preload.sh"),
}

func TestConformanceSuite_RF61Scenario13_AdapterReturningErrorSurfacesAcrossFourProviders(t *testing.T) {
	for _, tool := range conformanceCanonicalTools {
		t.Run(string(tool), func(t *testing.T) {
			projectDir := installConformanceFixture(t)

			if tool == skills.ToolOpenCode {
				pluginPath := filepath.Join(projectDir, ".opencode", "plugin", "governance.js")
				if err := os.WriteFile(pluginPath, []byte("export default function( this is not valid javascript"), 0o644); err != nil {
					t.Fatalf("corrupt opencode plugin: %v", err)
				}
				out, exitCode := runConformanceOpenCodeHarness(t, pluginPath, projectDir, "git status")
				if exitCode == 0 {
					t.Fatalf("a syntactically broken adapter must surface as an error, never as a silent ALLOW; output=%s", out)
				}
				return
			}

			relPath := conformanceGitPreToolHookRelPath[tool]
			hookPath := filepath.Join(projectDir, relPath)
			if err := os.WriteFile(hookPath, []byte("#!/usr/bin/env bash\nthis is not a valid script (\n"), 0o644); err != nil {
				t.Fatalf("corrupt provider adapter hook: %v", err)
			}
			stdin := []byte(`{"tool_input":{"command":"git status"}}`)
			out, exitCode := runHookScript(t, projectDir, relPath, stdin, "GOVERNANCE_PRELOAD_CONFIRMED=1")
			if exitCode == 0 {
				t.Fatalf("provider=%s: a syntactically broken adapter hook must surface as an error, never as a silent ALLOW; output=%s", tool, out)
			}
			if strings.Contains(out, "operacao git nao solicitada") {
				t.Fatalf("provider=%s: a broken adapter must never produce a well-formed policy verdict; it must fail as a broken adapter, not impersonate the gate; output=%s", tool, out)
			}
		})
	}
}

func TestConformanceSuite_RF61Scenario14_CommandVariationBypassIsBlockedAcrossFourProviders(t *testing.T) {
	projectDir := installConformanceFixture(t)

	bypassCommands := []string{
		`$(echo git) push origin main`,
		`eval "git push origin main"`,
		`sh -c "git push origin main"`,
		`/usr/bin/git push origin main`,
		`git -c user.name=x commit -m x`,
		`git push --force`,
		`git push -f`,
	}

	for _, command := range bypassCommands {
		for _, tool := range conformanceCanonicalTools {
			t.Run(string(tool)+"/"+command, func(t *testing.T) {
				out, exitCode := dispatchGitOperationCommand(t, projectDir, tool, command, "GOVERNANCE_PRELOAD_CONFIRMED=1")
				if exitCode != preToolBlockExitCode {
					t.Fatalf("provider=%s: bypass attempt %q must be blocked with exit=%d; got exit=%d output=%s", tool, command, preToolBlockExitCode, exitCode, out)
				}
				if strings.Contains(out, brokenChainMarker) {
					t.Fatalf("provider=%s: the wrapper refused because the canonical validator is missing, not because the gate ran; output=%s", tool, out)
				}
			})
		}
	}

	safeCommand := "git status"
	for _, tool := range conformanceCanonicalTools {
		t.Run(string(tool)+"/"+safeCommand, func(t *testing.T) {
			out, exitCode := dispatchGitOperationCommand(t, projectDir, tool, safeCommand, "GOVERNANCE_PRELOAD_CONFIRMED=1")
			if exitCode != 0 {
				t.Fatalf("provider=%s: gate-of-the-gate — a genuinely read-only command must stay ALLOWED, discriminating it from the bypass attempts above; got exit=%d output=%s", tool, exitCode, out)
			}
		})
	}
}

func TestConformanceSuite_RF61Scenario14_SuiteFailsWhenCanonicalGitOperationGateIsRemoved(t *testing.T) {
	projectDir := installConformanceFixture(t)

	canonicalGate := filepath.Join(projectDir, ".agents", "scripts", "git-operation-gate.sh")
	if _, err := os.Stat(canonicalGate); err != nil {
		t.Fatalf("canonical git-operation-gate.sh must be installed before it can be removed: %v", err)
	}
	if err := os.Remove(canonicalGate); err != nil {
		t.Fatalf("remove canonical git-operation-gate.sh: %v", err)
	}

	for _, tool := range conformanceCanonicalTools {
		t.Run(string(tool), func(t *testing.T) {
			out, exitCode := dispatchGitOperationCommand(t, projectDir, tool, `$(echo git) push origin main`, "GOVERNANCE_PRELOAD_CONFIRMED=1")
			gateStillProvesTheDenial := exitCode == preToolBlockExitCode && !strings.Contains(out, brokenChainMarker)
			if gateStillProvesTheDenial {
				t.Fatalf("provider=%s: removing the canonical git-operation-gate.sh must make the bypass-variation suite fail (V-31 gate-of-the-gate) — it stayed green instead; output=%s", tool, out)
			}
		})
	}
}
