//go:build integration

package integration

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/JailtonJunior94/ai-spec-harness/internal/embedded"
)

const opencodeDispatchHarness = `
import { pathToFileURL } from "node:url"

const pluginPath = process.argv[2]
const directory = process.argv[3]
const tool = process.argv[4]
const filePath = process.argv[5] || ""

const mod = await import(pathToFileURL(pluginPath).href)
const hooks = await mod.default({ directory })

const input = { tool, sessionID: "test-session", callID: "test-call" }
const output = { args: filePath ? { filePath } : {} }

try {
  await hooks["tool.execute.before"](input, output)
  console.log("ALLOWED")
  process.exit(0)
} catch (err) {
  console.log("BLOCKED:" + err.message)
  process.exit(1)
}
`

func detectNode(t *testing.T) string {
	t.Helper()
	path, err := exec.LookPath("node")
	if err != nil {
		t.Skip("t.Skip: node ausente do PATH — instale node para rodar o disparo simulado do plugin OpenCode")
	}
	return path
}

func writeCanonicalScripts(t *testing.T, root string) {
	t.Helper()
	for _, name := range []string{
		"hook-prereq-gate.sh",
		"validate-skill-prerequisites.sh",
		"resolve-references.sh",
	} {
		data, err := embedded.Assets.ReadFile("assets/.agents/scripts/" + name)
		if err != nil {
			t.Fatalf("read embedded script %s: %v", name, err)
		}
		dst := filepath.Join(root, ".agents", "scripts", name)
		if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
		if err := os.WriteFile(dst, data, 0o755); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}
}

func copyGoImplementationSkill(t *testing.T, root string) {
	t.Helper()
	for _, rel := range []string{
		".agents/skills/go-implementation/SKILL.md",
		".agents/skills/go-implementation/references/INDEX.yaml",
	} {
		data, err := embedded.Assets.ReadFile("assets/" + rel)
		if err != nil {
			t.Fatalf("read embedded %s: %v", rel, err)
		}
		dst := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
		if err := os.WriteFile(dst, data, 0o644); err != nil {
			t.Fatalf("write %s: %v", rel, err)
		}
	}
}

func runDispatchHarness(t *testing.T, nodePath, pluginPath, directory, tool, filePath string, extraEnv ...string) (string, int) {
	t.Helper()
	harnessPath := filepath.Join(t.TempDir(), "dispatch-harness.mjs")
	if err := os.WriteFile(harnessPath, []byte(opencodeDispatchHarness), 0o644); err != nil {
		t.Fatalf("write harness: %v", err)
	}

	cmd := exec.Command(nodePath, harnessPath, pluginPath, directory, tool, filePath)
	cmd.Env = append(os.Environ(), extraEnv...)
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

func pluginAssetPath(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	data, err := embedded.Assets.ReadFile("assets/.opencode/plugin/governance.js")
	if err != nil {
		t.Fatalf("read embedded plugin: %v", err)
	}
	path := filepath.Join(dir, "governance.js")
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("write plugin: %v", err)
	}
	return path
}

func TestOpenCodeGovernancePluginBlocksWhenSkillPrerequisiteMissing(t *testing.T) {
	nodePath := detectNode(t)
	pluginPath := pluginAssetPath(t)
	dir := t.TempDir()
	writeCanonicalScripts(t, dir)

	out, exitCode := runDispatchHarness(t, nodePath, pluginPath, dir, "edit", "main.go")
	if exitCode == 0 {
		t.Fatalf("expected non-zero exit for edit without go-implementation skill; output=%s", out)
	}
	if !strings.Contains(out, "BLOCKED:") {
		t.Fatalf("expected corrective BLOCKED message; output=%s", out)
	}
	if !strings.Contains(out, "GOVERNANCE BLOCKED") {
		t.Fatalf("blocking message is not phrased as a corrective instruction; output=%s", out)
	}
}

func TestOpenCodeGovernancePluginAllowsWhenPrerequisiteSatisfied(t *testing.T) {
	nodePath := detectNode(t)
	pluginPath := pluginAssetPath(t)
	dir := t.TempDir()
	writeCanonicalScripts(t, dir)
	copyGoImplementationSkill(t, dir)

	out, exitCode := runDispatchHarness(t, nodePath, pluginPath, dir, "edit", "main.go")
	if exitCode != 0 {
		t.Fatalf("expected zero exit once go-implementation skill is present; output=%s", out)
	}
	if !strings.Contains(out, "ALLOWED") {
		t.Fatalf("expected ALLOWED; output=%s", out)
	}
}

func TestOpenCodeGovernancePluginIgnoresNonMutatingTools(t *testing.T) {
	nodePath := detectNode(t)
	pluginPath := pluginAssetPath(t)
	dir := t.TempDir()

	out, exitCode := runDispatchHarness(t, nodePath, pluginPath, dir, "read", "main.go")
	if exitCode != 0 {
		t.Fatalf("read tool must never be blocked (only pre-tool blocks; read never mutates); output=%s", out)
	}
	if !strings.Contains(out, "ALLOWED") {
		t.Fatalf("expected ALLOWED for non-mutating tool; output=%s", out)
	}
}

func TestOpenCodeGovernancePluginWarnsOnceForUnrecognizedTool(t *testing.T) {
	nodePath := detectNode(t)
	pluginPath := pluginAssetPath(t)
	dir := t.TempDir()

	harnessPath := filepath.Join(t.TempDir(), "dispatch-harness-unrecognized.mjs")
	multi := fmt.Sprintf(`
import { pathToFileURL } from "node:url"
const mod = await import(pathToFileURL(%q).href)
const hooks = await mod.default({ directory: %q })
let warnings = 0
const originalWarn = console.warn
console.warn = (...args) => { warnings++; originalWarn(...args) }
for (let i = 0; i < 2; i++) {
  await hooks["tool.execute.before"]({ tool: "future_mutating_tool", sessionID: "s", callID: "c" + i }, { args: { filePath: "file" + i + ".go" } })
}
console.log("WARNINGS=" + warnings)
`, pluginPath, dir)
	if err := os.WriteFile(harnessPath, []byte(multi), 0o644); err != nil {
		t.Fatalf("write unrecognized-tool harness: %v", err)
	}

	cmd := exec.Command(nodePath, harnessPath)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("unrecognized tool must never block (only known mutating tools are validated): %v (output=%s)", err, out)
	}
	if !strings.Contains(string(out), "not in the known mutating-tools set") {
		t.Fatalf("expected visibility warning for unrecognized tool bypassing validation; output=%s", out)
	}
	if !strings.Contains(string(out), "WARNINGS=1") {
		t.Fatalf("expected exactly one warning across two calls with the same unrecognized tool; output=%s", out)
	}
}

func TestOpenCodeGovernancePluginTimeoutIsDenial(t *testing.T) {
	nodePath := detectNode(t)
	pluginPath := pluginAssetPath(t)
	dir := t.TempDir()

	slowScript := "#!/usr/bin/env bash\nsleep 5\nexit 0\n"
	dst := filepath.Join(dir, ".agents", "scripts", "hook-prereq-gate.sh")
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(dst, []byte(slowScript), 0o755); err != nil {
		t.Fatalf("write slow script: %v", err)
	}

	out, exitCode := runDispatchHarness(t, nodePath, pluginPath, dir, "edit", "main.go",
		"AISPEC_OPENCODE_VALIDATOR_TIMEOUT_MS=200")
	if exitCode == 0 {
		t.Fatalf("timeout must be treated as denial, never approval; output=%s", out)
	}
	if !strings.Contains(out, "timed out") {
		t.Fatalf("expected timeout-as-denial message; output=%s", out)
	}
}

func TestOpenCodeGovernancePluginOrchestratedFailsClosedWithoutValidator(t *testing.T) {
	nodePath := detectNode(t)
	pluginPath := pluginAssetPath(t)
	dir := t.TempDir()

	out, exitCode := runDispatchHarness(t, nodePath, pluginPath, dir, "edit", "main.go",
		"AISPEC_OPENCODE_ORCHESTRATED=1")
	if exitCode == 0 {
		t.Fatalf("orchestrated mode without validator must fail closed; output=%s", out)
	}
	if !strings.Contains(out, "GOVERNANCE BLOCKED") {
		t.Fatalf("expected fail-closed corrective message; output=%s", out)
	}
}

func TestOpenCodeGovernancePluginMemoizesByToolAndFiles(t *testing.T) {
	nodePath := detectNode(t)
	pluginPath := pluginAssetPath(t)
	dir := t.TempDir()

	counterPath := filepath.Join(dir, "invocations.txt")
	script := "#!/usr/bin/env bash\n" +
		"echo x >> " + counterPath + "\n" +
		"exit 0\n"
	dst := filepath.Join(dir, ".agents", "scripts", "hook-prereq-gate.sh")
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(dst, []byte(script), 0o755); err != nil {
		t.Fatalf("write script: %v", err)
	}

	harnessPath := filepath.Join(t.TempDir(), "dispatch-harness-memo.mjs")
	memo := fmt.Sprintf(`
import { pathToFileURL } from "node:url"
const mod = await import(pathToFileURL(%q).href)
const hooks = await mod.default({ directory: %q })
for (let i = 0; i < 3; i++) {
  await hooks["tool.execute.before"]({ tool: "edit", sessionID: "s", callID: "c" + i }, { args: { filePath: "main.go" } })
}
console.log("DONE")
`, pluginPath, dir)
	if err := os.WriteFile(harnessPath, []byte(memo), 0o644); err != nil {
		t.Fatalf("write memo harness: %v", err)
	}

	cmd := exec.Command(nodePath, harnessPath)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("memoized calls must not error: %v (output=%s)", err, out)
	}

	data, err := os.ReadFile(counterPath)
	if err != nil {
		t.Fatalf("counter file not created: %v", err)
	}
	lines := strings.Count(string(data), "x")
	if lines != 1 {
		t.Fatalf("validator invoked %d times for the same (tool, files) key across 3 calls; want 1 (memoization)", lines)
	}
}

func TestOpenCodeGovernancePluginStatsValidatorOnceAcrossMultipleTools(t *testing.T) {
	nodePath := detectNode(t)
	pluginPath := pluginAssetPath(t)
	dir := t.TempDir()
	writeCanonicalScripts(t, dir)
	copyGoImplementationSkill(t, dir)

	harnessPath := filepath.Join(t.TempDir(), "dispatch-harness-stat.mjs")
	multiFile := fmt.Sprintf(`
import { pathToFileURL } from "node:url"
const mod = await import(pathToFileURL(%q).href)
const hooks = await mod.default({ directory: %q })
await hooks["tool.execute.before"]({ tool: "edit", sessionID: "s", callID: "c1" }, { args: { filePath: "a.go" } })
await hooks["tool.execute.before"]({ tool: "edit", sessionID: "s", callID: "c2" }, { args: { filePath: "b.go" } })
console.log("DONE")
`, pluginPath, dir)
	if err := os.WriteFile(harnessPath, []byte(multiFile), 0o644); err != nil {
		t.Fatalf("write stat harness: %v", err)
	}

	cmd := exec.Command(nodePath, harnessPath)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("distinct files must each be checked but never error: %v (output=%s)", err, out)
	}
	if !strings.Contains(string(out), "DONE") {
		t.Fatalf("harness did not complete: %s", out)
	}
}

func TestOpenCodeGovernancePluginInteractiveWarnsOnceWithoutValidator(t *testing.T) {
	nodePath := detectNode(t)
	pluginPath := pluginAssetPath(t)
	dir := t.TempDir()

	harnessPath := filepath.Join(t.TempDir(), "dispatch-harness-multi.mjs")
	multi := fmt.Sprintf(`
import { pathToFileURL } from "node:url"
const mod = await import(pathToFileURL(%q).href)
const hooks = await mod.default({ directory: %q })
let warnings = 0
const originalWarn = console.warn
console.warn = (...args) => { warnings++; originalWarn(...args) }
for (let i = 0; i < 2; i++) {
  await hooks["tool.execute.before"]({ tool: "edit", sessionID: "s", callID: "c" + i }, { args: { filePath: "file" + i + ".go" } })
}
console.log("WARNINGS=" + warnings)
`, pluginPath, dir)
	if err := os.WriteFile(harnessPath, []byte(multi), 0o644); err != nil {
		t.Fatalf("write multi harness: %v", err)
	}

	cmd := exec.Command(nodePath, harnessPath)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("interactive mode without validator must never block: %v (output=%s)", err, out)
	}
	if !strings.Contains(string(out), "WARNINGS=1") {
		t.Fatalf("expected exactly one warning across two calls; output=%s", out)
	}
}
