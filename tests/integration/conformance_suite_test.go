//go:build integration

package integration

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/JailtonJunior94/ai-spec-harness/internal/adapters"
	"github.com/JailtonJunior94/ai-spec-harness/internal/config"
	"github.com/JailtonJunior94/ai-spec-harness/internal/conformance"
	"github.com/JailtonJunior94/ai-spec-harness/internal/contextgen"
	"github.com/JailtonJunior94/ai-spec-harness/internal/fs"
	"github.com/JailtonJunior94/ai-spec-harness/internal/install"
	"github.com/JailtonJunior94/ai-spec-harness/internal/manifest"
	"github.com/JailtonJunior94/ai-spec-harness/internal/output"
	"github.com/JailtonJunior94/ai-spec-harness/internal/skills"
)

const conformanceOpenCodeHarness = `
import { pathToFileURL } from "node:url"
const pluginPath = process.argv[2]
const directory = process.argv[3]
const command = process.argv[4]
const mod = await import(pathToFileURL(pluginPath).href)
const hooks = await mod.default({ directory })
try {
  await hooks["tool.execute.before"]({ tool: "bash", sessionID: "conformance-suite", callID: "conformance-suite" }, { args: { command } })
  console.log("ALLOWED")
  process.exit(0)
} catch (err) {
  console.log("BLOCKED:" + err.message)
  process.exit(2)
}
`

func runConformanceOpenCodeHarness(t *testing.T, pluginPath, directory, command string, extraEnv ...string) (string, int) {
	t.Helper()
	nodePath := detectNode(t)
	harnessPath := filepath.Join(t.TempDir(), "conformance-suite-harness.mjs")
	if err := os.WriteFile(harnessPath, []byte(conformanceOpenCodeHarness), 0o644); err != nil {
		t.Fatalf("write conformance suite node harness: %v", err)
	}

	cmd := exec.Command(nodePath, harnessPath, pluginPath, directory, command)
	cmd.Env = append(os.Environ(), extraEnv...)
	out, err := cmd.CombinedOutput()
	exitCode := 0
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			exitCode = ee.ExitCode()
		} else {
			t.Fatalf("run conformance suite node harness: %v (output=%s)", err, out)
		}
	}
	return string(out), exitCode
}

var conformanceCanonicalTools = []skills.Tool{
	skills.ToolClaude,
	skills.ToolCodex,
	skills.ToolCopilot,
	skills.ToolOpenCode,
}

func installConformanceFixture(t *testing.T) string {
	t.Helper()
	sourceDir := repoRootForDispatch(t)

	projectDir := t.TempDir()
	fsys := fs.NewOSFileSystem()
	printer := output.New(false)
	mfst := manifest.NewStore(fsys)
	adpt := adapters.NewGenerator(fsys, printer)
	ctxg := contextgen.NewGenerator(fsys, printer)
	svc := install.NewService(fsys, printer, mfst, adpt, ctxg)

	if err := svc.Execute(config.InstallOptions{
		ProjectDir: projectDir,
		SourceDir:  sourceDir,
		Tools:      conformanceCanonicalTools,
		Langs:      []skills.Lang{skills.LangNode},
		LinkMode:   skills.LinkCopy,
	}); err != nil {
		t.Fatalf("install conformance fixture: %v", err)
	}
	return projectDir
}

func dispatchGitOperationCommand(t *testing.T, projectDir string, tool skills.Tool, command string, extraEnv ...string) (string, int) {
	t.Helper()

	if tool == skills.ToolOpenCode {
		pluginPath := filepath.Join(projectDir, ".opencode", "plugin", "governance.js")
		return runConformanceOpenCodeHarness(t, pluginPath, projectDir, command, extraEnv...)
	}

	relPath, ok := preToolHookRelPath[tool]
	if !ok {
		t.Fatalf("no bash pre-tool hook registered for provider %q", tool)
	}
	stdin := []byte(`{"tool_input":{"command":"` + command + `"}}`)
	return runHookScript(t, projectDir, relPath, stdin, extraEnv...)
}

func assertConformanceGateBlocks(t *testing.T, tool skills.Tool, out string, exitCode int, wantExit int, wantVerdict string) {
	t.Helper()
	if exitCode != wantExit {
		t.Fatalf("provider=%s: expected exit %d, got exit=%d output=%s", tool, wantExit, exitCode, out)
	}
	if strings.Contains(out, brokenChainMarker) {
		t.Fatalf("provider=%s: the wrapper refused because the canonical validator is missing, not because the gate ran; a broken delegation chain must never read as a working gate; output=%s", tool, out)
	}
	if !strings.Contains(out, wantVerdict) {
		t.Fatalf("provider=%s: the refusal must carry the verdict of the canonical validator (%q); output=%s", tool, wantVerdict, out)
	}
}

func TestConformanceSuite_ScenariosSixAndSevenAreDeterministicPerManifest(t *testing.T) {
	byID := make(map[int]conformance.Scenario, 14)
	for _, s := range conformance.Manifest() {
		byID[s.ID] = s
	}
	for _, id := range []int{6, 7} {
		s, ok := byID[id]
		if !ok || !s.Determinist {
			t.Fatalf("scenario #%d must be classified as deterministic in the conformance manifest before its gate can bind CI", id)
		}
	}
}

func TestConformanceSuite_GitOperationGateBlocksAcrossFourProvidersWithSingleRunner(t *testing.T) {
	projectDir := installConformanceFixture(t)

	for _, tool := range conformanceCanonicalTools {
		tool := tool
		t.Run(string(tool), func(t *testing.T) {
			out, exitCode := dispatchGitOperationCommand(t, projectDir, tool, "git commit -m x", "GOVERNANCE_PRELOAD_CONFIRMED=1")
			assertConformanceGateBlocks(t, tool, out, exitCode, preToolBlockExitCode, "operacao git nao solicitada")
		})
	}
}

func TestConformanceSuite_DestructiveOperationGateBlocksAcrossFourProvidersWithSingleRunner(t *testing.T) {
	projectDir := installConformanceFixture(t)

	for _, tool := range conformanceCanonicalTools {
		tool := tool
		t.Run(string(tool), func(t *testing.T) {
			out, exitCode := dispatchGitOperationCommand(t, projectDir, tool, "rm -rf /", "GOVERNANCE_PRELOAD_CONFIRMED=1")
			assertConformanceGateBlocks(t, tool, out, exitCode, preToolBlockExitCode, "comando destrutivo detectado")
		})
	}
}

func TestConformanceSuite_SuiteFailsWhenCanonicalGitOperationGateIsRemoved(t *testing.T) {
	projectDir := installConformanceFixture(t)

	canonicalGate := filepath.Join(projectDir, ".agents", "scripts", "git-operation-gate.sh")
	if _, err := os.Stat(canonicalGate); err != nil {
		t.Fatalf("canonical git-operation-gate.sh must be installed before it can be removed: %v", err)
	}
	if err := os.Remove(canonicalGate); err != nil {
		t.Fatalf("remove canonical git-operation-gate.sh: %v", err)
	}

	for _, tool := range conformanceCanonicalTools {
		tool := tool
		t.Run(string(tool), func(t *testing.T) {
			out, exitCode := dispatchGitOperationCommand(t, projectDir, tool, "git commit -m x", "GOVERNANCE_PRELOAD_CONFIRMED=1")
			gateStillProvesTheDenial := exitCode == preToolBlockExitCode &&
				!strings.Contains(out, brokenChainMarker) &&
				strings.Contains(out, "operacao git nao solicitada")
			if gateStillProvesTheDenial {
				t.Fatalf("provider=%s: removing the canonical git-operation-gate.sh must make the suite fail (V-31 gate-of-the-gate) — it stayed green instead; output=%s", tool, out)
			}
		})
	}
}

func TestConformanceSuite_FourProvidersReturnSameExitCodeForSameInput(t *testing.T) {
	projectDir := installConformanceFixture(t)

	cases := []struct {
		name    string
		command string
		env     []string
	}{
		{name: "unsolicited-commit", command: "git commit -m x", env: []string{"GOVERNANCE_PRELOAD_CONFIRMED=1"}},
		{name: "unsolicited-push", command: "git push origin main", env: []string{"GOVERNANCE_PRELOAD_CONFIRMED=1"}},
		{name: "destructive-removal", command: "rm -rf /", env: []string{"GOVERNANCE_PRELOAD_CONFIRMED=1"}},
		{name: "read-only-status", command: "git status", env: []string{"GOVERNANCE_PRELOAD_CONFIRMED=1"}},
		{name: "confirmed-commit", command: "git commit -m x", env: []string{"GOVERNANCE_PRELOAD_CONFIRMED=1", "GOVERNANCE_GIT_OPERATION_CONFIRMED=1"}},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			exitByTool := make(map[skills.Tool]int, len(conformanceCanonicalTools))
			outputByTool := make(map[skills.Tool]string, len(conformanceCanonicalTools))
			for _, tool := range conformanceCanonicalTools {
				out, exitCode := dispatchGitOperationCommand(t, projectDir, tool, tc.command, tc.env...)
				exitByTool[tool] = exitCode
				outputByTool[tool] = out
			}

			reference := conformanceCanonicalTools[0]
			diverging := make([]skills.Tool, 0, len(conformanceCanonicalTools))
			for _, tool := range conformanceCanonicalTools {
				if exitByTool[tool] != exitByTool[reference] {
					diverging = append(diverging, tool)
				}
			}
			if len(diverging) == 0 {
				return
			}

			attribution := conformance.AttributionCore
			if len(diverging) < len(conformanceCanonicalTools)-1 {
				attribution = conformance.AttributionAdapter
			}
			t.Fatalf(
				"case %q: exit code diverges across providers (attribution=%s): reference=%s exit=%d diverging=%v\nreference output=%s",
				tc.name, attribution, reference, exitByTool[reference], diverging, outputByTool[reference],
			)
		})
	}
}
