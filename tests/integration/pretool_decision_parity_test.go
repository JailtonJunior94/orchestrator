//go:build integration

package integration

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/JailtonJunior94/ai-spec-harness/internal/skills"
)

const preToolDenialExitCode = 2

const opencodeDecisionHarness = `
import { pathToFileURL } from "node:url"

const mod = await import(pathToFileURL(process.argv[2]).href)
const hooks = await mod.default({ directory: process.argv[3] })

try {
  await hooks["tool.execute.before"]({ tool: "edit", sessionID: "s", callID: "c" }, { args: { filePath: process.argv[4] } })
  console.log("ALLOWED")
  process.exit(0)
} catch (err) {
  console.log("BLOCKED:" + err.message)
  process.exit(%d)
}
`

func runOpenCodePreToolDecision(t *testing.T, nodePath, pluginPath, projectDir, filePath string, env ...string) (string, int) {
	t.Helper()
	harnessPath := filepath.Join(t.TempDir(), "decision-harness.mjs")
	source := fmt.Sprintf(opencodeDecisionHarness, preToolDenialExitCode)
	if err := os.WriteFile(harnessPath, []byte(source), 0o644); err != nil {
		t.Fatalf("write decision harness: %v", err)
	}

	cmd := exec.Command(nodePath, harnessPath, pluginPath, projectDir, filePath)
	cmd.Dir = projectDir
	cmd.Env = append(os.Environ(), env...)
	out, err := cmd.CombinedOutput()
	exitCode := 0
	if err != nil {
		ee, ok := err.(*exec.ExitError)
		if !ok {
			t.Fatalf("run decision harness: %v (output=%s)", err, out)
		}
		exitCode = ee.ExitCode()
	}
	return string(out), exitCode
}

func TestPreToolDecisionIsIdenticalAcrossAllFourAgents(t *testing.T) {
	t.Parallel()
	nodePath := detectNode(t)
	projectDir := installMandatoryAgentsForDispatch(t)
	pluginPath := pluginAssetPath(t)

	cases := []struct {
		name     string
		filePath string
		env      []string
		wantExit int
	}{
		{name: "go-file-unconfirmed", filePath: "main.go", wantExit: preToolDenialExitCode},
		{name: "go-file-confirmed-skill-missing", filePath: "main.go", env: []string{"GOVERNANCE_PRELOAD_CONFIRMED=1"}, wantExit: preToolDenialExitCode},
		{name: "markdown-file", filePath: "README.md", wantExit: 0},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			stdin := []byte(`{"tool_input":{"file_path":"` + tc.filePath + `"}}`)
			for tool, relPath := range preToolHookRelPath {
				out, exitCode := runHookScript(t, projectDir, relPath, stdin, tc.env...)
				if exitCode != tc.wantExit {
					t.Fatalf("RF-22 case %q: agent %s decided exit=%d; want %d\n%s", tc.name, tool, exitCode, tc.wantExit, out)
				}
			}

			out, exitCode := runOpenCodePreToolDecision(t, nodePath, pluginPath, projectDir, tc.filePath, tc.env...)
			if exitCode != tc.wantExit {
				t.Fatalf("RF-22 case %q: agent %s decided exit=%d; want %d (the OpenCode plugin must route through the same canonical entry point as the legacy agents)\n%s",
					tc.name, skills.ToolOpenCode, exitCode, tc.wantExit, out)
			}
		})
	}
}
