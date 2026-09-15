//go:build integration

package integration

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

const sessionEndGateRelPath = ".agents/scripts/validate-session-end.sh"

const sessionEndCanonicalBlockExitCode = 2

func sessionEndGateAbsPath(t *testing.T) string {
	t.Helper()
	repoRoot, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	for {
		candidate := filepath.Join(repoRoot, sessionEndGateRelPath)
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
		parent := filepath.Dir(repoRoot)
		if parent == repoRoot {
			t.Fatalf("could not locate %s walking up from %s", sessionEndGateRelPath, repoRoot)
		}
		repoRoot = parent
	}
}

func writeTasksFixture(t *testing.T, projectDir, status, verdict string) {
	t.Helper()
	prdDir := filepath.Join(projectDir, ".specs", "prd-demo")
	if err := os.MkdirAll(prdDir, 0o755); err != nil {
		t.Fatalf("mkdir prd dir: %v", err)
	}
	tasksMD := "| # | Título | Status | Dependências | Paralelizável | Skills |\n" +
		"|---|--------|--------|-------------|---------------|--------|\n" +
		"| 1.0 | Demo | " + status + " | — | — | — |\n"
	if err := os.WriteFile(filepath.Join(prdDir, "tasks.md"), []byte(tasksMD), 0o644); err != nil {
		t.Fatalf("write tasks.md: %v", err)
	}
	if verdict != "" {
		report := "# Report\nveredito: " + verdict + "\n"
		if err := os.WriteFile(filepath.Join(prdDir, "1.0_execution_report.md"), []byte(report), 0o644); err != nil {
			t.Fatalf("write report: %v", err)
		}
	}
}

func runSessionEndGate(t *testing.T, projectDir string, stdin []byte) (string, int) {
	t.Helper()
	scriptPath := sessionEndGateAbsPath(t)
	cmd := exec.Command("bash", scriptPath)
	cmd.Dir = projectDir
	cmd.Stdin = bytes.NewReader(stdin)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	err := cmd.Run()
	exitCode := 0
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			exitCode = ee.ExitCode()
		} else {
			t.Fatalf("run session-end gate: %v", err)
		}
	}
	return out.String(), exitCode
}

func newGitProjectDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	cmd := exec.Command("git", "init", "-q", dir)
	if err := cmd.Run(); err != nil {
		t.Fatalf("git init: %v", err)
	}
	return dir
}

var cliSessionEndContracts = map[string][]byte{
	"claude":   []byte(`{"hook_event_name":"Stop","session_id":"s1"}`),
	"codex":    []byte(`{"event":"Stop"}`),
	"copilot":  []byte(`{"type":"agentStop"}`),
	"opencode": []byte(``),
}

func TestSessionEndGateBlocksActiveTaskWithoutApprovedVerdict(t *testing.T) {
	t.Parallel()

	for cli, stdin := range cliSessionEndContracts {
		t.Run(cli, func(t *testing.T) {
			t.Parallel()
			dir := newGitProjectDir(t)
			writeTasksFixture(t, dir, "in_progress", "")

			out, exitCode := runSessionEndGate(t, dir, stdin)
			if exitCode != sessionEndCanonicalBlockExitCode {
				t.Fatalf("cli=%s: expected exit %d for active task without approved verdict; got exit=%d output=%s",
					cli, sessionEndCanonicalBlockExitCode, exitCode, out)
			}
		})
	}
}

func TestSessionEndGateAllowsWhenNoActiveTaskOrApproved(t *testing.T) {
	t.Parallel()

	for cli, stdin := range cliSessionEndContracts {
		t.Run(cli+"/pending", func(t *testing.T) {
			t.Parallel()
			dir := newGitProjectDir(t)
			writeTasksFixture(t, dir, "pending", "")

			out, exitCode := runSessionEndGate(t, dir, stdin)
			if exitCode != 0 {
				t.Fatalf("cli=%s: expected zero exit when no task was started; output=%s", cli, out)
			}
		})

		t.Run(cli+"/done-approved", func(t *testing.T) {
			t.Parallel()
			dir := newGitProjectDir(t)
			writeTasksFixture(t, dir, "done", "APPROVED")

			out, exitCode := runSessionEndGate(t, dir, stdin)
			if exitCode != 0 {
				t.Fatalf("cli=%s: a task closed with an APPROVED verdict must not be blocked; output=%s", cli, out)
			}
		})

		t.Run(cli+"/approved", func(t *testing.T) {
			t.Parallel()
			dir := newGitProjectDir(t)
			writeTasksFixture(t, dir, "in_progress", "APPROVED")

			out, exitCode := runSessionEndGate(t, dir, stdin)
			if exitCode != 0 {
				t.Fatalf("cli=%s: expected zero exit when active task has APPROVED verdict; output=%s", cli, out)
			}
		})
	}
}

func TestSessionEndGateBlocksApprovedWithRemarks(t *testing.T) {
	dir := newGitProjectDir(t)
	writeTasksFixture(t, dir, "in_progress", "APPROVED_WITH_REMARKS")
	_, exitCode := runSessionEndGate(t, dir, cliSessionEndContracts["opencode"])
	if exitCode != sessionEndCanonicalBlockExitCode {
		t.Fatalf("expected exit %d for APPROVED_WITH_REMARKS; got %d", sessionEndCanonicalBlockExitCode, exitCode)
	}
}

func runSessionEndGateSplit(t *testing.T, projectDir string, env []string) (string, string, int) {
	t.Helper()
	scriptPath := sessionEndGateAbsPath(t)
	cmd := exec.Command("bash", scriptPath)
	cmd.Dir = projectDir
	cmd.Stdin = bytes.NewReader([]byte(``))
	cmd.Env = append(os.Environ(), env...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	exitCode := 0
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			exitCode = ee.ExitCode()
		} else {
			t.Fatalf("run session-end gate: %v", err)
		}
	}
	return stdout.String(), stderr.String(), exitCode
}

func TestSessionEndGateDefaultModeKeepsStdoutEmpty(t *testing.T) {
	t.Parallel()
	dir := newGitProjectDir(t)
	writeTasksFixture(t, dir, "in_progress", "")

	stdout, stderr, exitCode := runSessionEndGateSplit(t, dir, nil)
	if exitCode != sessionEndCanonicalBlockExitCode {
		t.Fatalf("default mode must keep blocking with exit %d; got %d", sessionEndCanonicalBlockExitCode, exitCode)
	}
	if stdout != "" {
		t.Fatalf("Claude and Codex read the exit code and stderr; emitting anything on stdout by default changes a contract they already honour: %q", stdout)
	}
	if !strings.Contains(stderr, "GATE DE ENCERRAMENTO BLOQUEADO") {
		t.Fatalf("the refusal reason must stay on stderr in the default mode; got %q", stderr)
	}
}

func TestSessionEndGateJSONModeEmitsBlockDecisionOnStdout(t *testing.T) {
	t.Parallel()
	dir := newGitProjectDir(t)
	writeTasksFixture(t, dir, "in_progress", "")

	stdout, stderr, exitCode := runSessionEndGateSplit(t, dir, []string{"AISPEC_HOOK_DECISION_OUTPUT=json"})

	var decision struct {
		Decision string `json:"decision"`
		Reason   string `json:"reason"`
	}
	if err := json.Unmarshal([]byte(stdout), &decision); err != nil {
		t.Fatalf("Copilot parses agentStop stdout as a JSON object; got %q: %v", stdout, err)
	}
	if decision.Decision != "block" {
		t.Fatalf("decision block is the only value Copilot honours to keep the agent running; got %q", decision.Decision)
	}
	if decision.Reason == "" {
		t.Fatalf("Copilot enqueues reason as a follow-up user message; an empty reason blocks without telling the agent why: %q", stdout)
	}
	if exitCode != sessionEndCanonicalBlockExitCode {
		t.Fatalf("Copilot keeps stdout parsed on exit 0 and exit 2 but discards it on any other non-zero exit, so the gate must still exit %d; got %d", sessionEndCanonicalBlockExitCode, exitCode)
	}
	if !strings.Contains(stderr, "GATE DE ENCERRAMENTO BLOQUEADO") {
		t.Fatalf("the JSON mode must add a stdout channel without removing the stderr diagnostic; got %q", stderr)
	}
}

func TestSessionEndGateJSONModeEmitsNoDecisionWhenApproved(t *testing.T) {
	t.Parallel()
	dir := newGitProjectDir(t)
	writeTasksFixture(t, dir, "in_progress", "APPROVED")

	stdout, _, exitCode := runSessionEndGateSplit(t, dir, []string{"AISPEC_HOOK_DECISION_OUTPUT=json"})
	if exitCode != 0 {
		t.Fatalf("an approved verdict must let the session close; got exit %d stdout=%q", exitCode, stdout)
	}
	var decision map[string]any
	if err := json.Unmarshal([]byte(stdout), &decision); err != nil {
		t.Fatalf("the JSON mode must always emit a JSON object; got %q: %v", stdout, err)
	}
	if _, present := decision["decision"]; present {
		t.Fatalf("omitting decision is what lets Copilot stop normally; got %q", stdout)
	}
}

func writeCanonicalVerdictReport(t *testing.T, projectDir, body string) {
	t.Helper()
	path := filepath.Join(projectDir, ".specs", "prd-demo", "1.0_execution_report.md")
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatalf("write report: %v", err)
	}
}

func TestSessionEndGateBlocksTaskClosedAsDoneWithoutApprovedVerdict(t *testing.T) {
	t.Parallel()

	for cli, stdin := range cliSessionEndContracts {
		t.Run(cli+"/no-report", func(t *testing.T) {
			t.Parallel()
			dir := newGitProjectDir(t)
			writeTasksFixture(t, dir, "done", "")

			out, exitCode := runSessionEndGate(t, dir, stdin)
			if exitCode != sessionEndCanonicalBlockExitCode {
				t.Fatalf("cli=%s: writing done in tasks.md must not be enough to close the session — RF-36 forbids done without APPROVED; got exit=%d output=%s",
					cli, exitCode, out)
			}
			if !strings.Contains(out, "done") {
				t.Fatalf("cli=%s: the refusal must name the status that triggered it; output=%s", cli, out)
			}
		})

		t.Run(cli+"/approved-with-remarks", func(t *testing.T) {
			t.Parallel()
			dir := newGitProjectDir(t)
			writeTasksFixture(t, dir, "done", "APPROVED_WITH_REMARKS")

			out, exitCode := runSessionEndGate(t, dir, stdin)
			if exitCode != sessionEndCanonicalBlockExitCode {
				t.Fatalf("cli=%s: APPROVED_WITH_REMARKS does not close the approval cycle (RF-33); got exit=%d output=%s",
					cli, exitCode, out)
			}
		})
	}
}

func TestSessionEndGateReadsTheCanonicalVerdictBlockOfTheExecutionReport(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		body     string
		wantExit int
	}{
		{"diff-reviewed block approved", "# Report\n\n```\nverdict=APPROVED\ntool=claude\n```\n", 0},
		{"diff-reviewed block with remarks", "# Report\n\n```\nverdict=APPROVED_WITH_REMARKS\ntool=claude\n```\n", sessionEndCanonicalBlockExitCode},
		{"prose reviewer verdict approved", "# Report\n\n- Veredito do Revisor: APPROVED (sem achados)\n", 0},
		{"prose reviewer verdict with remarks", "# Report\n\n- Veredito do Revisor: APPROVED_WITH_REMARKS (low)\n", sessionEndCanonicalBlockExitCode},
		{"prose reviewer verdict rejected", "# Report\n\n- Veredito do Revisor: REJECTED\n", sessionEndCanonicalBlockExitCode},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			dir := newGitProjectDir(t)
			writeTasksFixture(t, dir, "done", "")
			writeCanonicalVerdictReport(t, dir, tc.body)

			out, exitCode := runSessionEndGate(t, dir, cliSessionEndContracts["opencode"])
			if exitCode != tc.wantExit {
				t.Fatalf("report body %q: exit=%d want %d; the gate must read the same verdict locator that validate-task-evidence.sh enforces; output=%s",
					tc.body, exitCode, tc.wantExit, out)
			}
		})
	}
}
