//go:build integration

package integration

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/JailtonJunior94/ai-spec-harness/internal/config"
	"github.com/JailtonJunior94/ai-spec-harness/internal/fs"
	"github.com/JailtonJunior94/ai-spec-harness/internal/manifest"
	"github.com/JailtonJunior94/ai-spec-harness/internal/skills"
)

const authoredClaudeGovernance = `# Regras do time de pagamentos

Nunca alterar o esquema de ledger sem aprovacao do time de risco.
`

const authoredOpenCodeLayout = "{\n" +
	"    \"$schema\": \"https://opencode.ai/config.json\",\n" +
	"\t\"model\": \"anthropic/claude-sonnet-4\",\n" +
	"    \"theme\":     \"opencode\"\n" +
	"}\n"

func TestE2E_InstallPreservesAuthoredClaudeMDAndUninstallKeepsIt(t *testing.T) {
	sourceDir := t.TempDir()
	projectDir := t.TempDir()
	setupSourceDir(t, sourceDir)

	claudePath := filepath.Join(projectDir, "CLAUDE.md")
	mustWriteFile(t, claudePath, authoredClaudeGovernance)

	fsys := fs.NewOSFileSystem()
	if err := newInstallSvc(fsys).Execute(config.InstallOptions{
		ProjectDir:  projectDir,
		SourceDir:   sourceDir,
		Tools:       []skills.Tool{skills.ToolClaude},
		LinkMode:    skills.LinkCopy,
		GenerateCtx: true,
	}); err != nil {
		t.Fatalf("install: %v", err)
	}

	afterInstall, err := os.ReadFile(claudePath)
	if err != nil {
		t.Fatalf("read CLAUDE.md after install: %v", err)
	}
	if !strings.Contains(string(afterInstall), "# Regras do time de pagamentos") {
		t.Fatalf("install destroyed the authored CLAUDE.md:\n%s", afterInstall)
	}

	mf, err := manifest.NewStore(fsys).Load(projectDir)
	if err != nil {
		t.Fatalf("load manifest: %v", err)
	}
	if !containsPath(mf.MergedFiles, "CLAUDE.md") {
		t.Errorf("CLAUDE.md must be tracked as merged, got merged=%v", mf.MergedFiles)
	}
	if containsPath(mf.InstalledFiles, "CLAUDE.md") {
		t.Errorf("a pre-existing CLAUDE.md must never be tracked as created, got installed=%v", mf.InstalledFiles)
	}

	if err := newUninstallSvc(fsys).Execute(projectDir, false); err != nil {
		t.Fatalf("uninstall: %v", err)
	}

	afterUninstall, err := os.ReadFile(claudePath)
	if err != nil {
		t.Fatalf("uninstall removed an authored CLAUDE.md: %v", err)
	}
	if !strings.Contains(string(afterUninstall), "Nunca alterar o esquema de ledger sem aprovacao do time de risco.") {
		t.Errorf("authored content lost after uninstall:\n%s", afterUninstall)
	}
}

func TestE2E_OpenCodeConfigSurvivesInstallUninstallByteIdentical(t *testing.T) {
	sourceDir := t.TempDir()
	projectDir := t.TempDir()
	setupSourceDir(t, sourceDir)
	mustWriteFile(t, filepath.Join(sourceDir, ".opencode/plugin/governance.js"), "export const Plugin = async () => ({})\n")

	configPath := filepath.Join(projectDir, "opencode.json")
	mustWriteFile(t, configPath, authoredOpenCodeLayout)

	fsys := fs.NewOSFileSystem()
	if err := newInstallSvc(fsys).Execute(config.InstallOptions{
		ProjectDir:  projectDir,
		SourceDir:   sourceDir,
		Tools:       []skills.Tool{skills.ToolOpenCode},
		LinkMode:    skills.LinkCopy,
		GenerateCtx: true,
	}); err != nil {
		t.Fatalf("install: %v", err)
	}

	afterInstall, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("read opencode.json after install: %v", err)
	}
	if !strings.Contains(string(afterInstall), "\t\"model\": \"anthropic/claude-sonnet-4\",\n") {
		t.Errorf("install reformatted authored lines:\n%s", afterInstall)
	}
	if !strings.Contains(string(afterInstall), "\"permission\"") {
		t.Fatalf("install did not add the permission block:\n%s", afterInstall)
	}

	if err := newUninstallSvc(fsys).Execute(projectDir, false); err != nil {
		t.Fatalf("uninstall: %v", err)
	}

	afterUninstall, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("uninstall removed an authored opencode.json: %v", err)
	}
	if string(afterUninstall) != authoredOpenCodeLayout {
		t.Errorf("install+uninstall round trip is not byte identical\nwant:\n%q\ngot:\n%q",
			authoredOpenCodeLayout, string(afterUninstall))
	}
}

func containsPath(paths []string, want string) bool {
	for _, path := range paths {
		if filepath.ToSlash(path) == want {
			return true
		}
	}
	return false
}
