//go:build integration

package integration

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/JailtonJunior94/ai-spec-harness/internal/adapters"
	"github.com/JailtonJunior94/ai-spec-harness/internal/config"
	"github.com/JailtonJunior94/ai-spec-harness/internal/contextgen"
	"github.com/JailtonJunior94/ai-spec-harness/internal/fs"
	"github.com/JailtonJunior94/ai-spec-harness/internal/install"
	"github.com/JailtonJunior94/ai-spec-harness/internal/manifest"
	"github.com/JailtonJunior94/ai-spec-harness/internal/output"
	"github.com/JailtonJunior94/ai-spec-harness/internal/skills"
)

var preToolHookRelPath = map[skills.Tool]string{
	skills.ToolClaude:  filepath.Join(".claude", "hooks", "validate-preload.sh"),
	skills.ToolCodex:   filepath.Join(".codex", "hooks", "validate-preload.sh"),
	skills.ToolCopilot: filepath.Join(".github", "hooks", "validate-preload.sh"),
}

var postToolHookRelPath = map[skills.Tool]string{
	skills.ToolClaude:  filepath.Join(".claude", "hooks", "validate-governance.sh"),
	skills.ToolCodex:   filepath.Join(".codex", "hooks", "validate-governance.sh"),
	skills.ToolCopilot: filepath.Join(".github", "hooks", "validate-governance.sh"),
}

func repoRootForDispatch(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	for {
		if _, statErr := os.Stat(filepath.Join(dir, ".agents", "scripts", "hook-prereq-gate.sh")); statErr == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatalf("could not locate repo root walking up from %s", dir)
		}
		dir = parent
	}
}

func installMandatoryAgentsForDispatch(t *testing.T) string {
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
		Tools:      []skills.Tool{skills.ToolClaude, skills.ToolCodex, skills.ToolCopilot},
		Langs:      []skills.Lang{skills.LangNode},
		LinkMode:   skills.LinkCopy,
	}); err != nil {
		t.Fatalf("install: %v", err)
	}
	return projectDir
}

func runHookScript(t *testing.T, projectDir, relPath string, stdin []byte, env ...string) (string, int) {
	t.Helper()
	return runHookScriptWithArgs(t, projectDir, relPath, stdin, nil, env...)
}

func runHookScriptWithArgs(t *testing.T, projectDir, relPath string, stdin []byte, args []string, env ...string) (string, int) {
	t.Helper()
	scriptPath := filepath.Join(projectDir, relPath)
	cmd := exec.Command("bash", append([]string{scriptPath}, args...)...)
	cmd.Dir = projectDir
	cmd.Stdin = bytes.NewReader(stdin)
	cmd.Env = append(os.Environ(), env...)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	err := cmd.Run()
	exitCode := 0
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			exitCode = ee.ExitCode()
		} else {
			t.Fatalf("run %s: %v", relPath, err)
		}
	}
	return out.String(), exitCode
}

func postToolHookInput(tool skills.Tool, filePath string) ([]byte, []string) {
	return []byte(`{"tool_input":{"file_path":"` + filePath + `"}}`), nil
}

const preToolBlockExitCode = 2

const brokenChainMarker = "validador canonico"

func TestPreToolHookDispatchBlocksWhenSkillPrerequisiteMissing(t *testing.T) {
	t.Parallel()
	projectDir := installMandatoryAgentsForDispatch(t)

	for tool, relPath := range preToolHookRelPath {
		t.Run(string(tool), func(t *testing.T) {
			stdin := []byte(`{"tool_input":{"file_path":"main.go"}}`)
			out, exitCode := runHookScript(t, projectDir, relPath, stdin, "GOVERNANCE_PRELOAD_CONFIRMED=1")
			if exitCode != preToolBlockExitCode {
				t.Fatalf("tool=%s: expected exit %d for .go edit without go-implementation skill; got exit=%d output=%s", tool, preToolBlockExitCode, exitCode, out)
			}
			if strings.Contains(out, brokenChainMarker) {
				t.Fatalf("tool=%s: the wrapper refused because the canonical validator is missing, not because the gate ran; a broken delegation chain must never read as a working gate; output=%s", tool, out)
			}
			if !strings.Contains(out, "BLOQUEIO: tarefa toca arquivos cuja skill obrigatoria nao esta acessivel.") {
				t.Fatalf("tool=%s: the refusal must carry the verdict of the canonical prerequisite gate, not merely a non-zero exit; output=%s", tool, out)
			}
			if !strings.Contains(out, "go-implementation") {
				t.Fatalf("tool=%s: the verdict must name the missing skill for the edited target; output=%s", tool, out)
			}
		})
	}
}

func TestPreToolHookDispatchAllowsNonCodeFiles(t *testing.T) {
	t.Parallel()
	projectDir := installMandatoryAgentsForDispatch(t)

	for tool, relPath := range preToolHookRelPath {
		t.Run(string(tool), func(t *testing.T) {
			stdin := []byte(`{"tool_input":{"file_path":"README.md"}}`)
			out, exitCode := runHookScript(t, projectDir, relPath, stdin)
			if exitCode != 0 {
				t.Fatalf("tool=%s: expected zero exit for non-code file; output=%s", tool, out)
			}
		})
	}
}

func TestPostToolHookDispatchBlocksGovernanceFileEdit(t *testing.T) {
	t.Parallel()
	projectDir := installMandatoryAgentsForDispatch(t)

	for tool, relPath := range postToolHookRelPath {
		t.Run(string(tool), func(t *testing.T) {
			governanceFile := "./.agents/skills/go-implementation/SKILL.md"
			stdin, args := postToolHookInput(tool, governanceFile)
			out, exitCode := runHookScriptWithArgs(t, projectDir, relPath, stdin, args)
			if exitCode == 0 {
				t.Fatalf("tool=%s: expected non-zero exit for governance file edit; output=%s", tool, out)
			}
		})
	}
}

func TestPostToolHookDispatchAllowsRegularFileEdit(t *testing.T) {
	t.Parallel()
	projectDir := installMandatoryAgentsForDispatch(t)

	for tool, relPath := range postToolHookRelPath {
		t.Run(string(tool), func(t *testing.T) {
			stdin, args := postToolHookInput(tool, "main.go")
			out, exitCode := runHookScriptWithArgs(t, projectDir, relPath, stdin, args)
			if exitCode != 0 {
				t.Fatalf("tool=%s: expected zero exit for regular file edit; output=%s", tool, out)
			}
		})
	}
}

func TestToolHooksDispatchNativeApplyPatchPayload(t *testing.T) {
	t.Parallel()
	projectDir := installMandatoryAgentsForDispatch(t)

	for tool, relPath := range preToolHookRelPath {
		t.Run(string(tool)+"/pre", func(t *testing.T) {
			stdin := []byte(`{"tool_input":{"patch":"*** Add File: main.go\n+package main\n"}}`)
			out, exitCode := runHookScript(t, projectDir, relPath, stdin, "GOVERNANCE_PRELOAD_CONFIRMED=1")
			if exitCode == 0 {
				t.Fatalf("tool=%s: expected non-zero exit for native apply_patch Go payload; output=%s", tool, out)
			}
		})
	}

	for tool, relPath := range postToolHookRelPath {
		t.Run(string(tool)+"/post", func(t *testing.T) {
			stdin := []byte(`{"tool_input":{"patch":"*** Update File: AGENTS.md\n@@\n"}}`)
			out, exitCode := runHookScript(t, projectDir, relPath, stdin)
			if exitCode == 0 {
				t.Fatalf("tool=%s: expected non-zero exit for native apply_patch governance payload; output=%s", tool, out)
			}
		})
	}
}

var sessionEndHookRelPath = map[skills.Tool]string{
	skills.ToolClaude:  filepath.Join(".claude", "hooks", "validate-session-end.sh"),
	skills.ToolCodex:   filepath.Join(".codex", "hooks", "validate-session-end.sh"),
	skills.ToolCopilot: filepath.Join(".github", "hooks", "validate-session-end.sh"),
}

const sessionEndBlockExitCode = 2

var sessionEndCanonicalScriptExitCode = map[skills.Tool]int{
	skills.ToolClaude:  sessionEndBlockExitCode,
	skills.ToolCodex:   sessionEndBlockExitCode,
	skills.ToolCopilot: sessionEndBlockExitCode,
}

var sessionEndNativePayload = map[skills.Tool][]byte{
	skills.ToolClaude:  []byte(`{"hook_event_name":"Stop","session_id":"s1"}`),
	skills.ToolCodex:   []byte(`{"event":"Stop"}`),
	skills.ToolCopilot: []byte(`{"type":"agentStop"}`),
}

func writeSessionEndTasksFixture(t *testing.T, projectDir, status, verdict string) {
	t.Helper()
	prdDir := filepath.Join(projectDir, ".specs", "prd-dispatch-demo")
	if err := os.MkdirAll(prdDir, 0o755); err != nil {
		t.Fatalf("mkdir prd dir: %v", err)
	}
	tasksMD := "| # | Título | Status | Dependências | Paralelizável | Skills |\n" +
		"|---|--------|--------|-------------|---------------|--------|\n" +
		"| 1.0 | Demo | " + status + " | — | — | — |\n"
	if err := os.WriteFile(filepath.Join(prdDir, "tasks.md"), []byte(tasksMD), 0o644); err != nil {
		t.Fatalf("write tasks.md: %v", err)
	}
	reportPath := filepath.Join(prdDir, "1.0_execution_report.md")
	if verdict == "" {
		_ = os.Remove(reportPath)
		return
	}
	if err := os.WriteFile(reportPath, []byte("# Report\nveredito: "+verdict+"\n"), 0o644); err != nil {
		t.Fatalf("write report: %v", err)
	}
}

func TestSessionEndHookDispatchBlocksActiveTaskWithoutApprovedVerdict(t *testing.T) {
	t.Parallel()

	for tool, relPath := range sessionEndHookRelPath {
		t.Run(string(tool), func(t *testing.T) {
			t.Parallel()
			projectDir := installMandatoryAgentsForDispatch(t)
			writeSessionEndTasksFixture(t, projectDir, "in_progress", "")

			out, exitCode := runHookScript(t, projectDir, relPath, sessionEndNativePayload[tool])
			want := sessionEndCanonicalScriptExitCode[tool]
			if exitCode != want {
				t.Fatalf("tool=%s: the canonical session-end script must reject an active task without APPROVED verdict with exit %d; this asserts the SCRIPT contract, not the CLI blocking contract (see TestSessionEndCliBlockingContractIsMeasuredNotAssumed); got exit=%d output=%s",
					tool, want, exitCode, out)
			}
		})
	}
}

func TestSessionEndHookDispatchAllowsApprovedVerdict(t *testing.T) {
	t.Parallel()

	for tool, relPath := range sessionEndHookRelPath {
		t.Run(string(tool), func(t *testing.T) {
			t.Parallel()
			projectDir := installMandatoryAgentsForDispatch(t)
			writeSessionEndTasksFixture(t, projectDir, "in_progress", "APPROVED")

			out, exitCode := runHookScript(t, projectDir, relPath, sessionEndNativePayload[tool])
			if exitCode != 0 {
				t.Fatalf("tool=%s: session-end hook must allow an APPROVED active task; output=%s", tool, out)
			}
		})
	}
}

func TestPreToolHookDecisionIsIdenticalAcrossAgents(t *testing.T) {
	t.Parallel()
	projectDir := installMandatoryAgentsForDispatch(t)

	cases := []struct {
		name      string
		stdin     string
		env       []string
		wantBlock bool
	}{
		{name: "empty-payload", stdin: `{}`, wantBlock: true},
		{name: "empty-payload-confirmed", stdin: `{}`, env: []string{"GOVERNANCE_PRELOAD_CONFIRMED=1"}, wantBlock: false},
		{name: "go-file-unconfirmed", stdin: `{"tool_input":{"file_path":"main.go"}}`, wantBlock: true},
		{name: "go-file-confirmed", stdin: `{"tool_input":{"file_path":"main.go"}}`, env: []string{"GOVERNANCE_PRELOAD_CONFIRMED=1"}, wantBlock: true},
		{name: "markdown-file", stdin: `{"tool_input":{"file_path":"README.md"}}`, wantBlock: false},
		{name: "apply-patch-go", stdin: `{"tool_input":{"patch":"*** Add File: main.go\n+package main\n"}}`, env: []string{"GOVERNANCE_PRELOAD_CONFIRMED=1"}, wantBlock: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			exitByTool := make(map[skills.Tool]int, len(preToolHookRelPath))
			outputByTool := make(map[skills.Tool]string, len(preToolHookRelPath))
			for tool, relPath := range preToolHookRelPath {
				out, exitCode := runHookScript(t, projectDir, relPath, []byte(tc.stdin), tc.env...)
				exitByTool[tool] = exitCode
				outputByTool[tool] = out
			}

			var reference skills.Tool
			for tool := range exitByTool {
				reference = tool
				break
			}
			for tool, exitCode := range exitByTool {
				if exitCode != exitByTool[reference] {
					t.Fatalf("RF-22: pre-tool decision diverges for %q: %s exited %d but %s exited %d\n%s=%s\n%s=%s",
						tc.name, tool, exitCode, reference, exitByTool[reference],
						tool, outputByTool[tool], reference, outputByTool[reference])
				}
			}

			blocked := exitByTool[reference] != 0
			if blocked != tc.wantBlock {
				t.Fatalf("case %q: blocked=%v want=%v (exit=%d) output=%s",
					tc.name, blocked, tc.wantBlock, exitByTool[reference], outputByTool[reference])
			}
		})
	}
}

func TestPreToolHooksDelegateToCanonicalScript(t *testing.T) {
	t.Parallel()
	projectDir := installMandatoryAgentsForDispatch(t)

	canonical := filepath.Join(projectDir, ".agents", "hooks", "validate-preload.sh")
	if _, err := os.Stat(canonical); err != nil {
		t.Fatalf("canonical pre-tool gate .agents/hooks/validate-preload.sh must be installed: %v", err)
	}

	var reference []byte
	for tool, relPath := range preToolHookRelPath {
		data, err := os.ReadFile(filepath.Join(projectDir, relPath))
		if err != nil {
			t.Fatalf("read %s hook: %v", tool, err)
		}
		if !bytes.Contains(data, []byte(".agents/hooks/validate-preload.sh")) {
			t.Fatalf("tool=%s: pre-tool hook must delegate to the canonical script, not reimplement the gate:\n%s", tool, data)
		}
		if reference == nil {
			reference = data
			continue
		}
		if !bytes.Equal(reference, data) {
			t.Fatalf("tool=%s: pre-tool wrappers must be byte-identical across agents", tool)
		}
	}
}

func TestPreToolHookDispatchValidatesShellCommandTargetsWithPreloadConfirmed(t *testing.T) {
	t.Parallel()
	projectDir := installMandatoryAgentsForDispatch(t)

	for tool, relPath := range preToolHookRelPath {
		t.Run(string(tool), func(t *testing.T) {
			stdin := []byte(`{"tool_input":{"command":"gofmt -w main.go"}}`)
			out, exitCode := runHookScript(t, projectDir, relPath, stdin, "GOVERNANCE_PRELOAD_CONFIRMED=1")
			if exitCode == 0 {
				t.Fatalf("tool=%s: a shell command that rewrites main.go must reach the prerequisite gate even with the preload confirmed — confirming the preload must never disable target validation; output=%s", tool, out)
			}
			if !bytes.Contains([]byte(out), []byte("go-implementation")) {
				t.Fatalf("tool=%s: the refusal must name the missing skill for the target extracted from the command; output=%s", tool, out)
			}
		})
	}
}

func TestPreToolHookDispatchAllowsShellCommandWithoutSourceTargets(t *testing.T) {
	t.Parallel()
	projectDir := installMandatoryAgentsForDispatch(t)

	for tool, relPath := range preToolHookRelPath {
		t.Run(string(tool), func(t *testing.T) {
			stdin := []byte(`{"tool_input":{"command":"ls -la"}}`)
			out, exitCode := runHookScript(t, projectDir, relPath, stdin, "GOVERNANCE_PRELOAD_CONFIRMED=1")
			if exitCode != 0 {
				t.Fatalf("tool=%s: a command with no source-file target must not be turned into a false block; output=%s", tool, out)
			}
		})
	}
}

func TestPreToolHookDispatchBlocksUnsolicitedGitCommitAndPushEvenWithPreloadConfirmed(t *testing.T) {
	t.Parallel()
	projectDir := installMandatoryAgentsForDispatch(t)

	for _, command := range []string{"git commit -m x", "git push origin main"} {
		for tool, relPath := range preToolHookRelPath {
			t.Run(string(tool)+"/"+command, func(t *testing.T) {
				stdin := []byte(`{"tool_input":{"command":"` + command + `"}}`)
				out, exitCode := runHookScript(t, projectDir, relPath, stdin, "GOVERNANCE_PRELOAD_CONFIRMED=1")
				if exitCode != preToolBlockExitCode {
					t.Fatalf("tool=%s: RF-40.1 requires exit %d for unsolicited %q even with preload confirmed; got exit=%d output=%s", tool, preToolBlockExitCode, command, exitCode, out)
				}
				if strings.Contains(out, brokenChainMarker) {
					t.Fatalf("tool=%s: the wrapper refused because the canonical validator is missing, not because the gate ran; output=%s", tool, out)
				}
				if !strings.Contains(out, "operacao git nao solicitada") {
					t.Fatalf("tool=%s: the refusal must carry the verdict of the canonical git-operation gate; output=%s", tool, out)
				}
			})
		}
	}
}

func TestPreToolHookDispatchAllowsGitCommitWithBothEscapesConfirmed(t *testing.T) {
	t.Parallel()
	projectDir := installMandatoryAgentsForDispatch(t)

	for tool, relPath := range preToolHookRelPath {
		t.Run(string(tool), func(t *testing.T) {
			stdin := []byte(`{"tool_input":{"command":"git commit -m x"}}`)
			out, exitCode := runHookScript(t, projectDir, relPath, stdin,
				"GOVERNANCE_PRELOAD_CONFIRMED=1", "GOVERNANCE_GIT_OPERATION_CONFIRMED=1")
			if exitCode != 0 {
				t.Fatalf("tool=%s: a git commit explicitly confirmed via its own dedicated escape must be released; output=%s", tool, out)
			}
		})
	}
}

func TestPreToolHookDispatchAllowsReadOnlyGitCommandsWithPreloadConfirmed(t *testing.T) {
	t.Parallel()
	projectDir := installMandatoryAgentsForDispatch(t)

	for _, command := range []string{"git status", "git diff", "git log --oneline -5"} {
		for tool, relPath := range preToolHookRelPath {
			t.Run(string(tool)+"/"+command, func(t *testing.T) {
				stdin := []byte(`{"tool_input":{"command":"` + command + `"}}`)
				out, exitCode := runHookScript(t, projectDir, relPath, stdin, "GOVERNANCE_PRELOAD_CONFIRMED=1")
				if exitCode != 0 {
					t.Fatalf("tool=%s: the git-operation gate must never turn into a blind block of read-only git commands; output=%s", tool, out)
				}
			})
		}
	}
}

func TestPreToolHookDispatchBlocksDestructiveRemovalEvenWithPreloadConfirmed(t *testing.T) {
	t.Parallel()
	projectDir := installMandatoryAgentsForDispatch(t)

	for tool, relPath := range preToolHookRelPath {
		t.Run(string(tool), func(t *testing.T) {
			stdin := []byte(`{"tool_input":{"command":"rm -rf /"}}`)
			out, exitCode := runHookScript(t, projectDir, relPath, stdin, "GOVERNANCE_PRELOAD_CONFIRMED=1")
			if exitCode != preToolBlockExitCode {
				t.Fatalf("tool=%s: RF-40.2 requires destructiveness to be evaluated independently of preload; got exit=%d output=%s", tool, exitCode, out)
			}
			if strings.Contains(out, brokenChainMarker) {
				t.Fatalf("tool=%s: the wrapper refused because the canonical validator is missing, not because the gate ran; output=%s", tool, out)
			}
			if !strings.Contains(out, "comando destrutivo detectado") {
				t.Fatalf("tool=%s: the refusal must carry the verdict of the canonical destructive-operation criterion; output=%s", tool, out)
			}
		})
	}
}
