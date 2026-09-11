//go:build integration

package integration

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

const sessionEndGateRelPath = ".agents/scripts/validate-session-end.sh"

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
			if exitCode == 0 {
				t.Fatalf("cli=%s: expected non-zero exit for active task without approved verdict; output=%s", cli, out)
			}
		})
	}
}

func TestSessionEndGateAllowsWhenNoActiveTaskOrApproved(t *testing.T) {
	t.Parallel()

	for cli, stdin := range cliSessionEndContracts {
		t.Run(cli+"/done", func(t *testing.T) {
			t.Parallel()
			dir := newGitProjectDir(t)
			writeTasksFixture(t, dir, "done", "")

			out, exitCode := runSessionEndGate(t, dir, stdin)
			if exitCode != 0 {
				t.Fatalf("cli=%s: expected zero exit when no active task; output=%s", cli, out)
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
