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
  await hooks["session.idle"]()
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

await hooks["tool.execute.after"]()
console.log("DONE")
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

func TestOpenCodeGovernancePluginSessionIdleRecordsActiveTaskWithoutApprovedVerdictWithoutBlocking(t *testing.T) {
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
	if exitCode != 0 {
		t.Fatalf("session.idle must never block the session end (RF-19/8.7 — only tool.execute.before blocks); output=%s", out)
	}
	if !strings.Contains(out, "ALLOWED") {
		t.Fatalf("expected ALLOWED; output=%s", out)
	}
	if !strings.Contains(out, "GOVERNANCE ADVISORY at session.idle") {
		t.Fatalf("expected the session-end gate rejection to be recorded as an observational advisory; output=%s", out)
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
	if strings.Contains(out, "GOVERNANCE ADVISORY") {
		t.Fatalf("no advisory expected when the session-end validator does not reject; output=%s", out)
	}
}

func TestOpenCodeGovernancePluginToolExecuteAfterIsSafeNoOp(t *testing.T) {
	nodePath := detectNode(t)
	pluginPath := pluginAssetPath(t)
	dir := t.TempDir()

	out, exitCode := runNodeHarness(t, nodePath, opencodePostToolHarness, pluginPath, dir)
	if exitCode != 0 {
		t.Fatalf("tool.execute.after must never throw (documented non-blocking placeholder, ADR-004); output=%s", out)
	}
	if !strings.Contains(out, "DONE") {
		t.Fatalf("harness did not complete; output=%s", out)
	}
}
