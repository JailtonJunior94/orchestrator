//go:build integration

package integration

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

const opencodeSessionEndHarness = `
import { pathToFileURL } from "node:url"

const pluginPath = process.argv[2]
const directory = process.argv[3]

const mod = await import(pathToFileURL(pluginPath).href)
const hooks = await mod.default({ directory })

try {
  await hooks.event({ event: { type: "session.idle" } })
  console.log("ALLOWED")
  process.exit(0)
} catch (err) {
  console.log("BLOCKED:" + err.message)
  process.exit(1)
}
`

const opencodePostToolHarness = `
import { pathToFileURL } from "node:url"

const pluginPath = process.argv[2]
const directory = process.argv[3]

const mod = await import(pathToFileURL(pluginPath).href)
const hooks = await mod.default({ directory })

try {
  await hooks["tool.execute.after"]({ tool: "edit", sessionID: "s", callID: "c", args: { filePath: ".agents/skills/go-implementation/SKILL.md" } }, { title: "edit", output: "ok", metadata: {} })
  console.log("ALLOWED")
  process.exit(0)
} catch (err) {
  console.log("BLOCKED:" + err.message)
  process.exit(1)
}
`

func writeSessionEndGateFixture(t *testing.T, root string) {
	t.Helper()
	dst := filepath.Join(root, ".agents", "scripts", "validate-session-end.sh")
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	scriptPath := sessionEndGateAbsPath(t)
	data, err := os.ReadFile(scriptPath)
	if err != nil {
		t.Fatalf("read canonical session-end gate: %v", err)
	}
	if err := os.WriteFile(dst, data, 0o755); err != nil {
		t.Fatalf("write session-end gate fixture: %v", err)
	}
}

func runNodeHarness(t *testing.T, nodePath, source, pluginPath, directory string) (string, int) {
	t.Helper()
	harnessPath := filepath.Join(t.TempDir(), fmt.Sprintf("harness-%d.mjs", len(source)))
	if err := os.WriteFile(harnessPath, []byte(source), 0o644); err != nil {
		t.Fatalf("write harness: %v", err)
	}
	cmd := exec.Command(nodePath, harnessPath, pluginPath, directory)
	out, err := cmd.CombinedOutput()
	exitCode := 0
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			exitCode = ee.ExitCode()
		} else {
			t.Fatalf("run harness: %v (output=%s)", err, out)
		}
	}
	return string(out), exitCode
}

func TestOpenCodeGovernancePluginSessionIdleBlocksActiveTaskWithoutApprovedVerdict(t *testing.T) {
	nodePath := detectNode(t)
	pluginPath := pluginAssetPath(t)
	dir := t.TempDir()
	writeSessionEndGateFixture(t, dir)

	prdDir := filepath.Join(dir, ".specs", "prd-demo")
	if err := os.MkdirAll(prdDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	tasksMD := "| # | Título | Status | Dependências | Paralelizável | Skills |\n" +
		"|---|--------|--------|-------------|---------------|--------|\n" +
		"| 1.0 | Demo | in_progress | — | — | — |\n"
	if err := os.WriteFile(filepath.Join(prdDir, "tasks.md"), []byte(tasksMD), 0o644); err != nil {
		t.Fatalf("write tasks.md: %v", err)
	}

	out, exitCode := runNodeHarness(t, nodePath, opencodeSessionEndHarness, pluginPath, dir)
	if exitCode == 0 {
		t.Fatalf("session.idle must block an active task without APPROVED verdict; output=%s", out)
	}
	if !strings.Contains(out, "BLOCKED:GOVERNANCE BLOCKED for tool \"session.idle\"") {
		t.Fatalf("expected recognizable governance denial; output=%s", out)
	}
	if strings.Contains(out, "ALLOWED") {
		t.Fatalf("session end must terminate after rejection; output=%s", out)
	}
}

func TestOpenCodeGovernancePluginSessionIdleAllowsWhenNoActiveTask(t *testing.T) {
	nodePath := detectNode(t)
	pluginPath := pluginAssetPath(t)
	dir := t.TempDir()
	writeSessionEndGateFixture(t, dir)

	out, exitCode := runNodeHarness(t, nodePath, opencodeSessionEndHarness, pluginPath, dir)
	if exitCode != 0 {
		t.Fatalf("expected zero exit when no active task exists; output=%s", out)
	}
	if !strings.Contains(out, "ALLOWED") {
		t.Fatalf("expected ALLOWED; output=%s", out)
	}
	if strings.Contains(out, "GOVERNANCE BLOCKED") {
		t.Fatalf("no denial expected when the session-end validator does not reject; output=%s", out)
	}
}

func TestOpenCodeGovernancePluginToolExecuteAfterObservesValidatorWithoutBlocking(t *testing.T) {
	nodePath := detectNode(t)
	pluginPath := pluginAssetPath(t)
	dir := t.TempDir()
	writeCanonicalScripts(t, dir)

	out, exitCode := runNodeHarness(t, nodePath, opencodePostToolHarness, pluginPath, dir)
	if exitCode != 0 {
		t.Fatalf("tool.execute.after is observational and must not block; output=%s", out)
	}
	if !strings.Contains(out, "ALLOWED") {
		t.Fatalf("expected allowed post-tool observation; output=%s", out)
	}
	if !strings.Contains(out, "GOVERNANCE OBSERVED at tool.execute.after") {
		t.Fatalf("the canonical post-tool validator must receive the edited target and report it; invoking it with an empty payload makes the cell inert; output=%s", out)
	}
	if !strings.Contains(out, ".agents/skills/go-implementation/SKILL.md") {
		t.Fatalf("the observation must name the governance file the tool actually touched; output=%s", out)
	}
}

func TestOpenCodeGovernancePluginToolExecuteAfterStaysSilentForRegularFile(t *testing.T) {
	nodePath := detectNode(t)
	pluginPath := pluginAssetPath(t)
	dir := t.TempDir()
	writeCanonicalScripts(t, dir)

	harness := strings.Replace(opencodePostToolHarness,
		".agents/skills/go-implementation/SKILL.md", "main.go", 1)
	out, exitCode := runNodeHarness(t, nodePath, harness, pluginPath, dir)
	if exitCode != 0 {
		t.Fatalf("tool.execute.after must not block a regular file edit; output=%s", out)
	}
	if strings.Contains(out, "GOVERNANCE OBSERVED") {
		t.Fatalf("a regular file edit must not produce a governance observation; output=%s", out)
	}
}

const opencodePostToolResultShapeHarness = `
import { pathToFileURL } from "node:url"

const pluginPath = process.argv[2]
const directory = process.argv[3]

const mod = await import(pathToFileURL(pluginPath).href)
const hooks = await mod.default({ directory })

await hooks["tool.execute.after"]({ tool: "edit", sessionID: "s", callID: "c" }, { title: "edit", output: "ok", metadata: {}, args: { filePath: ".agents/skills/go-implementation/SKILL.md" } })
console.log("COMPLETED")
process.exit(0)
`

func TestOpenCodePostToolReadsArgsFromInputNotFromToolResult(t *testing.T) {
	nodePath := detectNode(t)
	pluginPath := pluginAssetPath(t)
	dir := t.TempDir()
	writeCanonicalScripts(t, dir)

	out, exitCode := runNodeHarness(t, nodePath, opencodePostToolResultShapeHarness, pluginPath, dir)
	if exitCode != 0 {
		t.Fatalf("tool.execute.after is observational and must not block; output=%s", out)
	}
	if strings.Contains(out, "GOVERNANCE OBSERVED at tool.execute.after") {
		t.Fatalf("OpenCode 1.18.30 delivers args in the first handler argument; reading them from the tool result is the wrong shape and must observe nothing; output=%s", out)
	}
	if !strings.Contains(out, "no post-tool target extracted") {
		t.Fatalf("expected the wrong-shape payload to yield no observable target; output=%s", out)
	}
}
