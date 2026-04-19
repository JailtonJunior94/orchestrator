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
	"github.com/JailtonJunior94/ai-spec-harness/internal/uninstall"
	"github.com/JailtonJunior94/ai-spec-harness/internal/upgrade"
)

func mustWriteFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func mustWriteExecFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(content), 0o755); err != nil {
		t.Fatalf("write exec %s: %v", path, err)
	}
	// os.WriteFile nao atualiza permissoes em arquivo existente; garantir 0o755 explicitamente
	if err := os.Chmod(path, 0o755); err != nil {
		t.Fatalf("chmod %s: %v", path, err)
	}
}

// setupSourceDir cria uma estrutura minima de source dir para os testes E2E.
func setupSourceDir(t *testing.T, dir string) {
	t.Helper()
	mustWriteFile(t, filepath.Join(dir, ".agents/skills/review/SKILL.md"),
		"---\nname: review\nversion: 1.0.0\ndescription: Revisa codigo.\n---\n")
	mustWriteFile(t, filepath.Join(dir, ".agents/skills/analyze-project/assets/agents-template.md"),
		"<!-- governance-schema: 1.0.0 -->\n")
	mustWriteFile(t, filepath.Join(dir, ".claude/hooks/validate-governance.sh"), "#!/usr/bin/env bash\n")
	mustWriteFile(t, filepath.Join(dir, ".claude/hooks/validate-preload.sh"), "#!/usr/bin/env bash\n")
	mustWriteFile(t, filepath.Join(dir, "scripts/lib/parse-hook-input.sh"), "#!/usr/bin/env bash\n")
	mustWriteFile(t, filepath.Join(dir, "AGENTS.md"), "<!-- governance-schema: 1.0.0 -->\n# AGENTS\n")
}

func newInstallSvc(fsys fs.FileSystem) *install.Service {
	p := output.New(false)
	return install.NewService(fsys, p, manifest.NewStore(fsys), adapters.NewGenerator(fsys, p), contextgen.NewGenerator(fsys, p))
}

func newUpgradeSvc(fsys fs.FileSystem) *upgrade.Service {
	p := output.New(false)
	return upgrade.NewService(fsys, p, manifest.NewStore(fsys), adapters.NewGenerator(fsys, p), contextgen.NewGenerator(fsys, p))
}

func newUninstallSvc(fsys fs.FileSystem) *uninstall.Service {
	return uninstall.NewService(fsys, output.New(false))
}

func TestE2E_InstallUpgradeSchemaChange(t *testing.T) {
	sourceDir := t.TempDir()
	projectDir := t.TempDir()
	setupSourceDir(t, sourceDir)

	fsys := fs.NewOSFileSystem()

	// Install
	err := newInstallSvc(fsys).Execute(config.InstallOptions{
		ProjectDir:  projectDir,
		SourceDir:   sourceDir,
		Tools:       []skills.Tool{skills.ToolClaude},
		LinkMode:    skills.LinkCopy,
		GenerateCtx: true,
	})
	if err != nil {
		t.Fatalf("install: %v", err)
	}

	agentsPath := filepath.Join(projectDir, "AGENTS.md")
	data, err := os.ReadFile(agentsPath)
	if err != nil {
		t.Fatalf("AGENTS.md not found after install: %v", err)
	}
	if !strings.Contains(string(data), "governance-schema:") {
		t.Fatalf("AGENTS.md missing governance-schema after install, got: %q", string(data))
	}

	// Downgrade schema version to simulate an outdated project
	modified := strings.ReplaceAll(string(data), "governance-schema: 1.0.0", "governance-schema: 0.9.0")
	if err := os.WriteFile(agentsPath, []byte(modified), 0o644); err != nil {
		t.Fatalf("write AGENTS.md: %v", err)
	}

	// Upgrade
	err = newUpgradeSvc(fsys).Execute(config.UpgradeOptions{
		ProjectDir: projectDir,
		SourceDir:  sourceDir,
	})
	if err != nil {
		t.Fatalf("upgrade: %v", err)
	}

	data, err = os.ReadFile(agentsPath)
	if err != nil {
		t.Fatalf("AGENTS.md not found after upgrade: %v", err)
	}
	if !strings.Contains(string(data), "governance-schema: 1.0.0") {
		t.Fatalf("AGENTS.md schema not regenerated after upgrade, got: %q", string(data))
	}
}

func TestE2E_InstallGeneratesHooks(t *testing.T) {
	sourceDir := t.TempDir()
	projectDir := t.TempDir()
	setupSourceDir(t, sourceDir)

	fsys := fs.NewOSFileSystem()

	err := newInstallSvc(fsys).Execute(config.InstallOptions{
		ProjectDir: projectDir,
		SourceDir:  sourceDir,
		Tools:      []skills.Tool{skills.ToolClaude},
		LinkMode:   skills.LinkCopy,
	})
	if err != nil {
		t.Fatalf("install: %v", err)
	}

	expectedPaths := []string{
		filepath.Join(projectDir, ".claude/hooks/validate-governance.sh"),
		filepath.Join(projectDir, ".claude/hooks/validate-preload.sh"),
		filepath.Join(projectDir, "scripts/lib/parse-hook-input.sh"),
	}
	for _, p := range expectedPaths {
		if _, err := os.Stat(p); err != nil {
			t.Errorf("expected %s to exist: %v", p, err)
		}
	}

	settingsData, err := os.ReadFile(filepath.Join(projectDir, ".claude/settings.local.json"))
	if err != nil {
		t.Fatalf("settings.local.json not found: %v", err)
	}
	content := string(settingsData)
	if !strings.Contains(content, "validate-governance.sh") {
		t.Error("settings.local.json missing validate-governance.sh hook reference")
	}
	if !strings.Contains(content, "validate-preload.sh") {
		t.Error("settings.local.json missing validate-preload.sh hook reference")
	}
}

func TestE2E_UninstallRemovesAll(t *testing.T) {
	sourceDir := t.TempDir()
	projectDir := t.TempDir()
	setupSourceDir(t, sourceDir)

	fsys := fs.NewOSFileSystem()

	// Install with claude + codex
	err := newInstallSvc(fsys).Execute(config.InstallOptions{
		ProjectDir: projectDir,
		SourceDir:  sourceDir,
		Tools:      []skills.Tool{skills.ToolClaude, skills.ToolCodex},
		LinkMode:   skills.LinkCopy,
	})
	if err != nil {
		t.Fatalf("install: %v", err)
	}

	// Create AGENTS.local.md — must be preserved after uninstall
	localMD := filepath.Join(projectDir, "AGENTS.local.md")
	mustWriteFile(t, localMD, "# Custom user content\n")

	// Uninstall
	err = newUninstallSvc(fsys).Execute(projectDir, false)
	if err != nil {
		t.Fatalf("uninstall: %v", err)
	}

	// Governance artifacts must be removed
	removed := []string{
		filepath.Join(projectDir, ".agents/skills/review"),
		filepath.Join(projectDir, "AGENTS.md"),
		filepath.Join(projectDir, ".ai_spec_harness.json"),
	}
	for _, p := range removed {
		if _, err := os.Stat(p); err == nil {
			t.Errorf("expected %s to be removed after uninstall", p)
		}
	}

	// AGENTS.local.md must be preserved
	data, err := os.ReadFile(localMD)
	if err != nil {
		t.Fatalf("AGENTS.local.md should be preserved after uninstall: %v", err)
	}
	if string(data) != "# Custom user content\n" {
		t.Fatalf("AGENTS.local.md content changed: got %q", string(data))
	}
}

// setupSourceDirFull configura um source dir com hooks funcionais para testes de T10.
func setupSourceDirFull(t *testing.T, dir string) {
	t.Helper()
	setupSourceDir(t, dir)

	govHook := `#!/usr/bin/env bash
INPUT=$(cat)
if echo "$INPUT" | grep -qiE '"(AGENTS|CLAUDE|GEMINI)\.md"'; then
    MODE="${GOVERNANCE_HOOK_MODE:-warn}"
    if [[ "$MODE" == "fail" ]]; then
        echo "ERRO: Modificacao de governanca bloqueada" >&2
        exit 1
    fi
    echo "AVISO: Arquivo de governanca modificado" >&2
fi
exit 0
`
	preloadHook := `#!/usr/bin/env bash
INPUT=$(cat)
if echo "$INPUT" | grep -qiE '\.(go|ts|js|py|java)'; then
    MODE="${GOVERNANCE_PRELOAD_MODE:-warn}"
    if [[ "$MODE" == "fail" ]]; then
        echo "ERRO: Preload nao realizado" >&2
        exit 1
    fi
    echo "LEMBRETE: Carregar AGENTS.md antes de editar codigo" >&2
fi
exit 0
`
	mustWriteExecFile(t, filepath.Join(dir, ".claude/hooks/validate-governance.sh"), govHook)
	mustWriteExecFile(t, filepath.Join(dir, ".claude/hooks/validate-preload.sh"), preloadHook)
	mustWriteExecFile(t, filepath.Join(dir, "scripts/lib/parse-hook-input.sh"), "#!/usr/bin/env bash\n")
}

// T09 — Cross-Tool Upgrade Tests

func TestE2E_CrossToolUpgrade_CodexToClaude(t *testing.T) {
	sourceDir := t.TempDir()
	projectDir := t.TempDir()
	setupSourceDir(t, sourceDir)

	fsys := fs.NewOSFileSystem()

	// Primeira instalacao: apenas Codex
	err := newInstallSvc(fsys).Execute(config.InstallOptions{
		ProjectDir: projectDir,
		SourceDir:  sourceDir,
		Tools:      []skills.Tool{skills.ToolCodex},
		LinkMode:   skills.LinkCopy,
	})
	if err != nil {
		t.Fatalf("install codex: %v", err)
	}

	codexConfig := filepath.Join(projectDir, ".codex", "config.toml")
	claudeMD := filepath.Join(projectDir, "CLAUDE.md")

	if _, err := os.Stat(codexConfig); err != nil {
		t.Errorf(".codex/config.toml deve existir apos install codex: %v", err)
	}
	if _, err := os.Stat(claudeMD); err == nil {
		t.Error("CLAUDE.md nao deve existir apos install apenas codex")
	}

	// Segunda instalacao: Claude + Codex
	err = newInstallSvc(fsys).Execute(config.InstallOptions{
		ProjectDir:  projectDir,
		SourceDir:   sourceDir,
		Tools:       []skills.Tool{skills.ToolClaude, skills.ToolCodex},
		LinkMode:    skills.LinkCopy,
		GenerateCtx: true,
	})
	if err != nil {
		t.Fatalf("install claude+codex: %v", err)
	}

	if _, err := os.Stat(codexConfig); err != nil {
		t.Errorf(".codex/config.toml deve ser preservado apos upgrade: %v", err)
	}
	if _, err := os.Stat(claudeMD); err != nil {
		t.Errorf("CLAUDE.md deve existir apos install claude+codex: %v", err)
	}

	hooksDir := filepath.Join(projectDir, ".claude", "hooks")
	if info, err := os.Stat(hooksDir); err != nil || !info.IsDir() {
		t.Errorf(".claude/hooks/ deve existir apos install claude: %v", err)
	}
}

func TestE2E_CrossToolUpgrade_CopilotToGemini(t *testing.T) {
	sourceDir := t.TempDir()
	projectDir := t.TempDir()
	setupSourceDir(t, sourceDir)

	fsys := fs.NewOSFileSystem()

	// Primeira instalacao: apenas Copilot
	err := newInstallSvc(fsys).Execute(config.InstallOptions{
		ProjectDir: projectDir,
		SourceDir:  sourceDir,
		Tools:      []skills.Tool{skills.ToolCopilot},
		LinkMode:   skills.LinkCopy,
	})
	if err != nil {
		t.Fatalf("install copilot: %v", err)
	}

	githubAgents := filepath.Join(projectDir, ".github", "agents")
	geminiCmds := filepath.Join(projectDir, ".gemini", "commands")

	if info, err := os.Stat(githubAgents); err != nil || !info.IsDir() {
		t.Errorf(".github/agents/ deve existir apos install copilot: %v", err)
	}
	if _, err := os.Stat(geminiCmds); err == nil {
		t.Error(".gemini/commands/ nao deve existir antes de instalar gemini")
	}

	// Segunda instalacao: Copilot + Gemini
	err = newInstallSvc(fsys).Execute(config.InstallOptions{
		ProjectDir: projectDir,
		SourceDir:  sourceDir,
		Tools:      []skills.Tool{skills.ToolCopilot, skills.ToolGemini},
		LinkMode:   skills.LinkCopy,
	})
	if err != nil {
		t.Fatalf("install copilot+gemini: %v", err)
	}

	if info, err := os.Stat(githubAgents); err != nil || !info.IsDir() {
		t.Errorf(".github/agents/ deve ser preservado apos upgrade para copilot+gemini: %v", err)
	}
	if info, err := os.Stat(geminiCmds); err != nil || !info.IsDir() {
		t.Errorf(".gemini/commands/ deve existir apos install gemini: %v", err)
	}
}

// T10 — Hook Fail-Mode E2E Tests

func TestE2E_HookScriptsInstalled(t *testing.T) {
	sourceDir := t.TempDir()
	projectDir := t.TempDir()
	setupSourceDirFull(t, sourceDir)

	fsys := fs.NewOSFileSystem()
	err := newInstallSvc(fsys).Execute(config.InstallOptions{
		ProjectDir: projectDir,
		SourceDir:  sourceDir,
		Tools:      []skills.Tool{skills.ToolClaude},
		LinkMode:   skills.LinkCopy,
	})
	if err != nil {
		t.Fatalf("install: %v", err)
	}

	govHookPath := filepath.Join(projectDir, ".claude/hooks/validate-governance.sh")
	preloadHookPath := filepath.Join(projectDir, ".claude/hooks/validate-preload.sh")
	parseLibPath := filepath.Join(projectDir, "scripts/lib/parse-hook-input.sh")

	for _, p := range []string{govHookPath, preloadHookPath, parseLibPath} {
		info, err := os.Stat(p)
		if err != nil {
			t.Errorf("esperado %s existir: %v", p, err)
			continue
		}
		if info.Mode()&0o111 == 0 {
			t.Errorf("%s deve ser executavel, modo=%v", p, info.Mode())
		}
	}

	settingsData, err := os.ReadFile(filepath.Join(projectDir, ".claude/settings.local.json"))
	if err != nil {
		t.Fatalf("settings.local.json nao encontrado: %v", err)
	}
	content := string(settingsData)
	if !strings.Contains(content, "validate-governance.sh") {
		t.Error("settings.local.json deve conter referencia a validate-governance.sh")
	}
	if !strings.Contains(content, "validate-preload.sh") {
		t.Error("settings.local.json deve conter referencia a validate-preload.sh")
	}
}

func runHook(t *testing.T, scriptPath, stdinJSON string, env []string) (exitCode int, stderr string) {
	t.Helper()
	cmd := exec.Command("bash", scriptPath)
	cmd.Stdin = bytes.NewBufferString(stdinJSON)
	if env != nil {
		cmd.Env = append(os.Environ(), env...)
	}
	var errBuf bytes.Buffer
	cmd.Stderr = &errBuf
	err := cmd.Run()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			t.Fatalf("exec hook %s: %v", scriptPath, err)
		}
	}
	return exitCode, errBuf.String()
}

func installWithFunctionalHooks(t *testing.T) (projectDir string) {
	t.Helper()
	sourceDir := t.TempDir()
	projectDir = t.TempDir()
	setupSourceDirFull(t, sourceDir)
	fsys := fs.NewOSFileSystem()
	if err := newInstallSvc(fsys).Execute(config.InstallOptions{
		ProjectDir: projectDir,
		SourceDir:  sourceDir,
		Tools:      []skills.Tool{skills.ToolClaude},
		LinkMode:   skills.LinkCopy,
	}); err != nil {
		t.Fatalf("install: %v", err)
	}
	return projectDir
}

func TestE2E_GovernanceHookWarnMode(t *testing.T) {
	projectDir := installWithFunctionalHooks(t)
	hookPath := filepath.Join(projectDir, ".claude/hooks/validate-governance.sh")
	stdin := `{"tool":"Write","tool_input":{"file_path":"AGENTS.md"}}`

	code, stderr := runHook(t, hookPath, stdin, nil)
	if code != 0 {
		t.Errorf("warn mode: esperado exit 0, got %d", code)
	}
	if !strings.Contains(stderr, "AVISO") {
		t.Errorf("warn mode: stderr deve conter 'AVISO', got: %q", stderr)
	}
}

func TestE2E_GovernanceHookFailMode(t *testing.T) {
	projectDir := installWithFunctionalHooks(t)
	hookPath := filepath.Join(projectDir, ".claude/hooks/validate-governance.sh")
	stdin := `{"tool":"Write","tool_input":{"file_path":"AGENTS.md"}}`

	code, _ := runHook(t, hookPath, stdin, []string{"GOVERNANCE_HOOK_MODE=fail"})
	if code != 1 {
		t.Errorf("fail mode: esperado exit 1, got %d", code)
	}
}

func TestE2E_GovernanceHook_NonGovernanceFile(t *testing.T) {
	projectDir := installWithFunctionalHooks(t)
	hookPath := filepath.Join(projectDir, ".claude/hooks/validate-governance.sh")
	stdin := `{"tool":"Edit","tool_input":{"file_path":"internal/service.go"}}`

	code, stderr := runHook(t, hookPath, stdin, nil)
	if code != 0 {
		t.Errorf("arquivo nao-governanca: esperado exit 0, got %d", code)
	}
	if strings.Contains(stderr, "AVISO") {
		t.Errorf("arquivo nao-governanca nao deve disparar aviso de governanca, got: %q", stderr)
	}
}

func TestE2E_PreloadHookWarnMode(t *testing.T) {
	projectDir := installWithFunctionalHooks(t)
	hookPath := filepath.Join(projectDir, ".claude/hooks/validate-preload.sh")
	stdin := `{"tool":"Edit","tool_input":{"file_path":"internal/service.go"}}`

	code, stderr := runHook(t, hookPath, stdin, nil)
	if code != 0 {
		t.Errorf("warn mode: esperado exit 0, got %d", code)
	}
	if !strings.Contains(stderr, "LEMBRETE") {
		t.Errorf("warn mode: stderr deve conter 'LEMBRETE', got: %q", stderr)
	}
}

func TestE2E_PreloadHookFailMode(t *testing.T) {
	projectDir := installWithFunctionalHooks(t)
	hookPath := filepath.Join(projectDir, ".claude/hooks/validate-preload.sh")
	stdin := `{"tool":"Edit","tool_input":{"file_path":"internal/service.go"}}`

	code, _ := runHook(t, hookPath, stdin, []string{"GOVERNANCE_PRELOAD_MODE=fail"})
	if code != 1 {
		t.Errorf("fail mode: esperado exit 1, got %d", code)
	}
}

func TestE2E_PreloadHook_NonCodeFile(t *testing.T) {
	projectDir := installWithFunctionalHooks(t)
	hookPath := filepath.Join(projectDir, ".claude/hooks/validate-preload.sh")
	stdin := `{"tool":"Read","tool_input":{"file_path":"README.md"}}`

	code, stderr := runHook(t, hookPath, stdin, nil)
	if code != 0 {
		t.Errorf("arquivo nao-codigo: esperado exit 0, got %d", code)
	}
	if strings.Contains(stderr, "LEMBRETE") {
		t.Errorf("arquivo nao-codigo nao deve disparar lembrete de preload, got: %q", stderr)
	}
}
