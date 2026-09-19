package install_test

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/JailtonJunior94/ai-spec-harness/internal/adapters"
	"github.com/JailtonJunior94/ai-spec-harness/internal/config"
	"github.com/JailtonJunior94/ai-spec-harness/internal/contextgen"
	"github.com/JailtonJunior94/ai-spec-harness/internal/embedded"
	"github.com/JailtonJunior94/ai-spec-harness/internal/fs"
	"github.com/JailtonJunior94/ai-spec-harness/internal/install"
	"github.com/JailtonJunior94/ai-spec-harness/internal/manifest"
	"github.com/JailtonJunior94/ai-spec-harness/internal/output"
	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/specs"
	"github.com/JailtonJunior94/ai-spec-harness/internal/skills"
)

var mandatoryMatrixAgents = []skills.Tool{
	skills.ToolClaude,
	skills.ToolCodex,
	skills.ToolCopilot,
	skills.ToolOpenCode,
}

var jsConstStringPattern = regexp.MustCompile(`const\s+([A-Za-z0-9_]+)\s*=\s*"([^"]*)"`)

func extractJSConstStrings(content string) map[string]string {
	out := map[string]string{}
	for _, m := range jsConstStringPattern.FindAllStringSubmatch(content, -1) {
		out[m[1]] = m[2]
	}
	return out
}

func installedProjectScriptResolver(projectDir string) specs.ScriptResolver {
	return func(relPath string) ([]byte, error) {
		return os.ReadFile(filepath.Join(projectDir, filepath.FromSlash(relPath)))
	}
}

func allMandatoryMatrixCellsDispatchProven(string, specs.CanonicalPoint) bool { return true }

func mandatoryMatrixCells(t *testing.T) []specs.AgentEnforcement {
	t.Helper()
	catalog := specs.NewCatalog()
	cells := make([]specs.AgentEnforcement, 0, len(mandatoryMatrixAgents))
	for _, tool := range mandatoryMatrixAgents {
		agent, err := catalog.AgentByID(string(tool))
		if err != nil {
			t.Fatalf("agent %q not in registry: %v", tool, err)
		}
		cells = append(cells, specs.AgentEnforcement{Agent: string(tool), Enforcement: agent.Enforcement()})
	}
	return cells
}

func installMandatoryMatrixProject(t *testing.T, generateCtx bool) string {
	t.Helper()

	sourceDir, cleanup, err := embedded.NewExtractor().ExtractToTempDir()
	if err != nil {
		t.Fatalf("extract embedded assets: %v", err)
	}
	t.Cleanup(cleanup)

	projectDir := t.TempDir()
	fsys := fs.NewOSFileSystem()
	printer := output.New(false)
	mfst := manifest.NewStore(fsys)
	adpt := adapters.NewGenerator(fsys, printer)
	ctxg := contextgen.NewGenerator(fsys, printer)
	svc := install.NewService(fsys, printer, mfst, adpt, ctxg)

	if err := svc.Execute(config.InstallOptions{
		ProjectDir:  projectDir,
		SourceDir:   sourceDir,
		Tools:       mandatoryMatrixAgents,
		LinkMode:    skills.LinkCopy,
		GenerateCtx: generateCtx,
	}); err != nil {
		t.Fatalf("install (GenerateCtx=%v): %v", generateCtx, err)
	}
	return projectDir
}

func TestMandatoryMatrixWrittenConfigContainsNativeKeyAndValidator(t *testing.T) {
	t.Parallel()

	points := specs.NewCatalog().CanonicalPoints()
	if len(points) != 3 {
		t.Fatalf("expected 3 canonical points, got %d", len(points))
	}
	if len(mandatoryMatrixAgents) != 4 {
		t.Fatalf("expected 4 mandatory agents, got %d", len(mandatoryMatrixAgents))
	}
	expectedCells := len(mandatoryMatrixAgents) * len(points)
	if expectedCells != 12 {
		t.Fatalf("mandatory matrix must cover 4 agents x 3 canonical points = 12 cells; got %d", expectedCells)
	}

	requiredAgents := make([]string, 0, len(mandatoryMatrixAgents))
	for _, tool := range mandatoryMatrixAgents {
		requiredAgents = append(requiredAgents, string(tool))
	}

	for _, generateCtx := range []bool{false, true} {
		t.Run(fmt.Sprintf("GenerateCtx=%v", generateCtx), func(t *testing.T) {
			t.Parallel()
			projectDir := installMandatoryMatrixProject(t, generateCtx)

			violations := specs.ValidateParityMatrix(
				mandatoryMatrixCells(t),
				requiredAgents,
				allMandatoryMatrixCellsDispatchProven,
				installedProjectScriptResolver(projectDir),
			)
			if len(violations) != 0 {
				t.Fatalf("installed native config must wire every mandatory cell to its canonical validator by execution, not by textual mention; got %v", violations)
			}
		})
	}
}

func TestCodexGovernanceRootKeysSurviveContextGeneration(t *testing.T) {
	t.Parallel()

	for _, generateCtx := range []bool{false, true} {
		t.Run(fmt.Sprintf("GenerateCtx=%v", generateCtx), func(t *testing.T) {
			t.Parallel()
			projectDir := installMandatoryMatrixProject(t, generateCtx)

			data, err := os.ReadFile(filepath.Join(projectDir, ".codex", "config.toml"))
			if err != nil {
				t.Fatalf("read .codex/config.toml: %v", err)
			}
			content := string(data)

			required := []string{
				"sandbox_mode",
				"approval_policy",
				"[[hooks.PreToolUse]]",
				"[[hooks.PostToolUse]]",
				"[[hooks.Stop]]",
				"[[skills.config]]",
			}
			for _, want := range required {
				if !strings.Contains(content, want) {
					t.Errorf("GenerateCtx=%v: .codex/config.toml lost %q; content=\n%s", generateCtx, want, content)
				}
			}

			firstTable := strings.Index(content, "[")
			for _, rootKey := range []string{"sandbox_mode", "approval_policy"} {
				idx := strings.Index(content, rootKey)
				if idx < 0 {
					continue
				}
				if firstTable >= 0 && idx > firstTable {
					t.Errorf("GenerateCtx=%v: root key %q must precede every TOML table to stay top-level", generateCtx, rootKey)
				}
			}
		})
	}
}

func TestCanonicalValidatorScriptsAreInstalledAndExecutable(t *testing.T) {
	t.Parallel()

	for _, generateCtx := range []bool{false, true} {
		t.Run(fmt.Sprintf("GenerateCtx=%v", generateCtx), func(t *testing.T) {
			t.Parallel()
			projectDir := installMandatoryMatrixProject(t, generateCtx)

			catalog := specs.NewCatalog()
			for _, tool := range mandatoryMatrixAgents {
				agent, err := catalog.AgentByID(string(tool))
				if err != nil {
					t.Fatalf("agent %q not in registry: %v", tool, err)
				}
				for _, cov := range agent.Enforcement().Coverage() {
					assertInstalledExecutable(t, projectDir, cov.ScriptPath())
				}
			}
		})
	}
}

func TestOpenCodePluginDeclaredScriptsExistAfterInstall(t *testing.T) {
	t.Parallel()

	projectDir := installMandatoryMatrixProject(t, true)

	pluginPath := filepath.Join(projectDir, ".opencode", "plugin", "governance.js")
	data, err := os.ReadFile(pluginPath)
	if err != nil {
		t.Fatalf("read installed governance plugin: %v", err)
	}

	consts := extractJSConstStrings(string(data))
	for _, name := range []string{"CANONICAL_PRE_TOOL_SCRIPT", "CANONICAL_POST_TOOL_SCRIPT", "CANONICAL_SESSION_END_SCRIPT"} {
		declared, ok := consts[name]
		if !ok {
			t.Fatalf("governance plugin does not declare %s", name)
		}
		assertInstalledExecutable(t, projectDir, declared)
	}
}

func assertInstalledExecutable(t *testing.T, projectDir, relPath string) {
	t.Helper()
	full := filepath.Join(projectDir, filepath.FromSlash(relPath))
	info, err := os.Stat(full)
	if err != nil {
		t.Errorf("validator %q was not installed: %v", relPath, err)
		return
	}
	if info.Mode().Perm()&0o111 == 0 {
		t.Errorf("validator %q is installed but not executable (mode %v)", relPath, info.Mode().Perm())
	}
}
