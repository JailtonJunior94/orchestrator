//go:build integration

package integration

import (
	"encoding/json"
	"os"
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
	"github.com/JailtonJunior94/ai-spec-harness/internal/upgrade"
)

const legacyCopilotGovernanceJSON = `{
  "version": 1,
  "hooks": {
    "preToolUse": [
      {
        "type": "command",
        "bash": "bash .github/hooks/validate-preload.sh"
      }
    ],
    "postToolUse": [
      {
        "type": "command",
        "bash": "bash .github/hooks/validate-governance.sh"
      }
    ],
    "stop": [
      {
        "type": "command",
        "bash": "bash .github/hooks/subagent-stop-wrapper.sh"
      }
    ]
  }
}
`

func seedLegacyCopilotGovernance(t *testing.T, projectDir string) string {
	t.Helper()
	path := filepath.Join(projectDir, ".github", "hooks", "governance.json")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, []byte(legacyCopilotGovernanceJSON), 0o644); err != nil {
		t.Fatalf("write legacy governance.json: %v", err)
	}
	return path
}

func readCopilotSessionEndCommands(t *testing.T, path string) ([]string, bool) {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read governance.json: %v", err)
	}
	var doc struct {
		Hooks map[string][]struct {
			Bash string `json:"bash"`
		} `json:"hooks"`
	}
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatalf("decode governance.json: %v", err)
	}
	_, obsolete := doc.Hooks["stop"]
	commands := make([]string, 0, len(doc.Hooks[upgrade.CopilotSessionEndHookKey]))
	for _, entry := range doc.Hooks[upgrade.CopilotSessionEndHookKey] {
		commands = append(commands, entry.Bash)
	}
	return commands, obsolete
}

func TestInstallRepairsPreexistingObsoleteCopilotGovernance(t *testing.T) {
	t.Parallel()
	sourceDir := repoRootForDispatch(t)
	projectDir := t.TempDir()
	path := seedLegacyCopilotGovernance(t, projectDir)

	fsys := fs.NewOSFileSystem()
	printer := output.New(false)
	svc := install.NewService(fsys, printer, manifest.NewStore(fsys),
		adapters.NewGenerator(fsys, printer), contextgen.NewGenerator(fsys, printer))
	if err := svc.Execute(config.InstallOptions{
		ProjectDir: projectDir,
		SourceDir:  sourceDir,
		Tools:      []skills.Tool{skills.ToolCopilot},
		Langs:      []skills.Lang{skills.LangNode},
		LinkMode:   skills.LinkCopy,
	}); err != nil {
		t.Fatalf("install: %v", err)
	}

	commands, obsolete := readCopilotSessionEndCommands(t, path)
	if obsolete {
		t.Fatalf("install must migrate the dead stop key; it survived")
	}
	if !strings.Contains(strings.Join(commands, "\n"), "validate-session-end.sh") {
		t.Fatalf("install must wire the session-end gate under agentStop; got %v", commands)
	}
}

func TestUpgradeRepairsPreexistingObsoleteCopilotGovernance(t *testing.T) {
	t.Parallel()
	sourceDir := repoRootForDispatch(t)
	projectDir := installMandatoryAgentsForDispatch(t)
	path := seedLegacyCopilotGovernance(t, projectDir)

	fsys := fs.NewOSFileSystem()
	printer := output.New(false)
	upgradeSvc := upgrade.NewService(fsys, printer, manifest.NewStore(fsys),
		adapters.NewGenerator(fsys, printer), contextgen.NewGenerator(fsys, printer))
	if err := upgradeSvc.Execute(config.UpgradeOptions{
		ProjectDir: projectDir,
		SourceDir:  sourceDir,
		Langs:      []skills.Lang{skills.LangNode},
	}); err != nil {
		t.Fatalf("upgrade: %v", err)
	}

	commands, obsolete := readCopilotSessionEndCommands(t, path)
	if obsolete {
		t.Fatalf("upgrade must offer a repair route for the dead stop key; it survived")
	}
	if !strings.Contains(strings.Join(commands, "\n"), "validate-session-end.sh") {
		t.Fatalf("upgrade must wire the session-end gate under agentStop; got %v", commands)
	}
	if !strings.Contains(strings.Join(commands, "\n"), "subagent-stop-wrapper.sh") {
		t.Fatalf("upgrade must preserve the migrated entry; got %v", commands)
	}
}

func TestUpgradeCheckOnlyReportsObsoleteCopilotGovernance(t *testing.T) {
	t.Parallel()
	sourceDir := repoRootForDispatch(t)
	projectDir := installMandatoryAgentsForDispatch(t)
	path := seedLegacyCopilotGovernance(t, projectDir)

	fsys := fs.NewOSFileSystem()
	printer := output.New(false)
	upgradeSvc := upgrade.NewService(fsys, printer, manifest.NewStore(fsys),
		adapters.NewGenerator(fsys, printer), contextgen.NewGenerator(fsys, printer))
	err := upgradeSvc.Execute(config.UpgradeOptions{
		ProjectDir: projectDir,
		SourceDir:  sourceDir,
		Langs:      []skills.Lang{skills.LangNode},
		CheckOnly:  true,
	})
	if err == nil {
		t.Fatalf("--check must fail while the dead stop key is still installed")
	}

	if _, obsolete := readCopilotSessionEndCommands(t, path); !obsolete {
		t.Fatalf("--check must not mutate the project")
	}
}

func TestRepoCopilotGovernanceAppliesItsOwnFix(t *testing.T) {
	t.Parallel()
	repo := repoRootForDispatch(t)
	commands, obsolete := readCopilotSessionEndCommands(t, filepath.Join(repo, ".github", "hooks", "governance.json"))
	if obsolete {
		t.Fatalf("this repository must not ship the dead stop key it claims to have fixed")
	}
	if !strings.Contains(strings.Join(commands, "\n"), "validate-session-end.sh") {
		t.Fatalf("this repository must wire its own session-end gate under agentStop; got %v", commands)
	}
}
