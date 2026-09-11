//go:build integration

package integration

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
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
	switch tool {
	case skills.ToolClaude:
		return []byte(`{"tool_input":{"file_path":"` + filePath + `"}}`), nil
	default:
		return nil, []string{filePath}
	}
}

func TestPreToolHookDispatchBlocksWhenSkillPrerequisiteMissing(t *testing.T) {
	t.Parallel()
	projectDir := installMandatoryAgentsForDispatch(t)

	for tool, relPath := range preToolHookRelPath {
		t.Run(string(tool), func(t *testing.T) {
			stdin := []byte(`{"tool_input":{"file_path":"main.go"}}`)
			out, exitCode := runHookScript(t, projectDir, relPath, stdin, "GOVERNANCE_PRELOAD_CONFIRMED=1")
			if exitCode == 0 {
				t.Fatalf("tool=%s: expected non-zero exit for .go edit without go-implementation skill; output=%s", tool, out)
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
