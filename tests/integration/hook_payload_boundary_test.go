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

const gitGateBlockExitCode = 2

func gitOperationGatePath(t *testing.T) string {
	t.Helper()
	root := repoRootForDispatch(t)
	return filepath.Join(root, ".agents", "scripts", "git-operation-gate.sh")
}

func runGitOperationGate(t *testing.T, stdin []byte, env ...string) (string, int) {
	t.Helper()
	gate := gitOperationGatePath(t)
	cmd := exec.Command("bash", gate)
	cmd.Dir = filepath.Dir(gate)
	cmd.Stdin = bytes.NewReader(stdin)
	if env != nil {
		cmd.Env = append(os.Environ(), env...)
	}
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	err := cmd.Run()
	exitCode := 0
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			exitCode = ee.ExitCode()
		} else {
			t.Fatalf("run git-operation-gate.sh: %v", err)
		}
	}
	return out.String(), exitCode
}

func gitGatePayload(t *testing.T, sizeBytes int, malicious bool) []byte {
	t.Helper()
	padding := strings.Repeat("A", sizeBytes)
	command := "echo " + padding
	if malicious {
		command = padding + "; git push origin main"
	}
	payload := map[string]any{
		"tool_input": map[string]any{
			"command": command,
		},
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}
	return raw
}

func TestHookPayloadBoundaryMatrixBlocksGitPushRegardlessOfSize(t *testing.T) {
	t.Parallel()

	sizes := map[string]int{
		"1KiB":  1024,
		"64KiB": 65536,
		"65KiB": 66560,
		"1MiB":  1048576,
	}

	for label, size := range sizes {
		t.Run(label+"/benign", func(t *testing.T) {
			t.Parallel()
			out, exitCode := runGitOperationGate(t, gitGatePayload(t, size, false))
			if exitCode != 0 {
				t.Fatalf("benign payload of size %s must pass (exit 0); got exit=%d output=%s", label, exitCode, out)
			}
		})

		t.Run(label+"/malicious", func(t *testing.T) {
			t.Parallel()
			out, exitCode := runGitOperationGate(t, gitGatePayload(t, size, true))
			if exitCode != gitGateBlockExitCode {
				t.Fatalf("payload of size %s carrying an unsolicited git push must block (exit %d); got exit=%d output=%s", label, gitGateBlockExitCode, exitCode, out)
			}
		})
	}
}

func TestHookPayloadBoundaryReproducesADR003ExploitAndControl(t *testing.T) {
	t.Parallel()

	padding := strings.Repeat("A", 70000)
	exploitPayload := []byte(`{"tool_input":{"command":"` + padding + `; echo hi ; git push origin main"}}`)
	out, exitCode := runGitOperationGate(t, exploitPayload)
	if exitCode != gitGateBlockExitCode {
		t.Fatalf("ADR-003 exploit payload (~70KB padding + git push) must block with exit %d — head -c 65536 truncation must never resurrect the exit-0 bypass; got exit=%d output=%s", gitGateBlockExitCode, exitCode, out)
	}

	controlPayload := []byte(`{"tool_input":{"command":"git push origin main"}}`)
	out, exitCode = runGitOperationGate(t, controlPayload)
	if exitCode != gitGateBlockExitCode {
		t.Fatalf("control payload without padding must also block with exit %d; got exit=%d output=%s", gitGateBlockExitCode, exitCode, out)
	}
}

func TestHookPayloadBoundaryFailsClosedOnInvalidJSON(t *testing.T) {
	t.Parallel()

	out, exitCode := runGitOperationGate(t, []byte("this is not json"))
	if exitCode != gitGateBlockExitCode {
		t.Fatalf("invalid JSON must fail closed with exit %d (RF-57), never exit 0; got exit=%d output=%s", gitGateBlockExitCode, exitCode, out)
	}
}

func TestHookPayloadBoundaryDeniesByAbsenceOfTarget(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name  string
		stdin []byte
	}{
		{name: "empty-object", stdin: []byte(`{}`)},
		{name: "empty-stdin", stdin: []byte("")},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			out, exitCode := runGitOperationGate(t, tc.stdin)
			if exitCode != gitGateBlockExitCode {
				t.Fatalf("RF-68: absence of an extractable command must deny (exit %d), never approve; got exit=%d output=%s", gitGateBlockExitCode, exitCode, out)
			}
		})
	}
}

func TestHookPayloadBoundaryDoesNotExtractNestedHomonymCommandField(t *testing.T) {
	t.Parallel()

	nestedMaliciousOnly := []byte(`{"tool_input":{"nested":{"tool_input":{"command":"git push origin main"}}}}`)
	out, exitCode := runGitOperationGate(t, nestedMaliciousOnly)
	if exitCode != gitGateBlockExitCode {
		t.Fatalf("a nested homonymous 'command' field must never be extracted as the effective command; absence of an extractable top-level command must deny (exit %d) per RF-68; got exit=%d output=%s", gitGateBlockExitCode, exitCode, out)
	}

	directBenignWithHomonymMetadata := []byte(`{"tool_input":{"command":"echo hi","metadata":{"command":"git push origin main"}}}`)
	out, exitCode = runGitOperationGate(t, directBenignWithHomonymMetadata)
	if exitCode != 0 {
		t.Fatalf("the direct top-level command must be evaluated on its own merits, unaffected by a homonymous field elsewhere in the payload; got exit=%d output=%s", exitCode, out)
	}
}

func fakeInterpreterlessPath(t *testing.T) string {
	t.Helper()
	required := []string{"bash", "dirname", "grep", "tr", "date", "cat", "mkdir", "sed"}
	binDir := t.TempDir()
	for _, name := range required {
		realPath, err := exec.LookPath(name)
		if err != nil {
			t.Fatalf("locate real %s in PATH: %v", name, err)
		}
		if err := os.Symlink(realPath, filepath.Join(binDir, name)); err != nil {
			t.Fatalf("symlink %s: %v", name, err)
		}
	}
	return binDir
}

func TestHookPayloadBoundaryFailsClosedWhenNoInterpreterAvailable(t *testing.T) {
	t.Parallel()

	binDir := fakeInterpreterlessPath(t)
	gate := gitOperationGatePath(t)
	cmd := exec.Command("bash", gate)
	cmd.Dir = filepath.Dir(gate)
	cmd.Stdin = bytes.NewReader([]byte(`{"tool_input":{"command":"git push origin main"}}`))
	cmd.Env = []string{"PATH=" + binDir, "HOME=" + os.Getenv("HOME")}
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	err := cmd.Run()
	exitCode := 0
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			exitCode = ee.ExitCode()
		} else {
			t.Fatalf("run git-operation-gate.sh without interpreter: %v", err)
		}
	}
	if exitCode != gitGateBlockExitCode {
		t.Fatalf("with neither python3 nor jq on PATH, hook-payload.sh must fail closed with exit %d, never silently approve; got exit=%d output=%s", gitGateBlockExitCode, exitCode, out.String())
	}
	if !strings.Contains(out.String(), "python3 or jq is required") {
		t.Fatalf("the block must carry a stderr message identifying the missing-interpreter cause; output=%s", out.String())
	}
}
