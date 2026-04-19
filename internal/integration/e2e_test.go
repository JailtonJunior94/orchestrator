//go:build integration

package integration

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/JailtonJunior94/ai-spec-harness/internal/adapters"
	"github.com/JailtonJunior94/ai-spec-harness/internal/config"
	"github.com/JailtonJunior94/ai-spec-harness/internal/contextgen"
	"github.com/JailtonJunior94/ai-spec-harness/internal/evidence"
	"github.com/JailtonJunior94/ai-spec-harness/internal/fs"
	"github.com/JailtonJunior94/ai-spec-harness/internal/install"
	"github.com/JailtonJunior94/ai-spec-harness/internal/manifest"
	"github.com/JailtonJunior94/ai-spec-harness/internal/output"
	"github.com/JailtonJunior94/ai-spec-harness/internal/skills"
	"github.com/JailtonJunior94/ai-spec-harness/internal/specdrift"
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

// T13 — Testes E2E para novos fluxos

// setupSourceDirWithEvidenceArtifacts estende setupSourceDir com scripts de evidencia,
// check-invocation-depth.sh e hook Gemini para os cenarios T13.
func setupSourceDirWithEvidenceArtifacts(t *testing.T, dir string) {
	t.Helper()
	setupSourceDir(t, dir)
	mustWriteFile(t, filepath.Join(dir, ".claude/scripts/validate-task-evidence.sh"), "#!/usr/bin/env bash\n")
	mustWriteFile(t, filepath.Join(dir, ".claude/scripts/validate-bugfix-evidence.sh"), "#!/usr/bin/env bash\n")
	mustWriteFile(t, filepath.Join(dir, ".claude/scripts/validate-refactor-evidence.sh"), "#!/usr/bin/env bash\n")
	mustWriteFile(t, filepath.Join(dir, "scripts/lib/check-invocation-depth.sh"), "#!/usr/bin/env bash\n")
	mustWriteExecFile(t, filepath.Join(dir, ".gemini/hooks/validate-preload.sh"), "#!/usr/bin/env bash\nexit 0\n")
}

func TestInstallCopiesAllEvidenceScripts(t *testing.T) {
	sourceDir := t.TempDir()
	projectDir := t.TempDir()
	setupSourceDirWithEvidenceArtifacts(t, sourceDir)

	fsys := fs.NewOSFileSystem()
	if err := newInstallSvc(fsys).Execute(config.InstallOptions{
		ProjectDir: projectDir,
		SourceDir:  sourceDir,
		Tools:      []skills.Tool{skills.ToolClaude},
		LinkMode:   skills.LinkCopy,
	}); err != nil {
		t.Fatalf("install: %v", err)
	}

	scripts := []string{
		filepath.Join(projectDir, ".claude/scripts/validate-task-evidence.sh"),
		filepath.Join(projectDir, ".claude/scripts/validate-bugfix-evidence.sh"),
		filepath.Join(projectDir, ".claude/scripts/validate-refactor-evidence.sh"),
	}
	for _, p := range scripts {
		if _, err := os.Stat(p); err != nil {
			t.Errorf("expected %s to exist after install: %v", p, err)
		}
	}

	if err := newUninstallSvc(fsys).Execute(projectDir, false); err != nil {
		t.Fatalf("uninstall: %v", err)
	}

	for _, p := range scripts {
		if _, err := os.Stat(p); err == nil {
			t.Errorf("expected %s to be removed after uninstall", p)
		}
	}
}

func TestInstallCopiesInvocationDepthGuard(t *testing.T) {
	sourceDir := t.TempDir()
	projectDir := t.TempDir()
	setupSourceDir(t, sourceDir)
	mustWriteFile(t, filepath.Join(sourceDir, "scripts/lib/check-invocation-depth.sh"), "#!/usr/bin/env bash\n")

	fsys := fs.NewOSFileSystem()
	if err := newInstallSvc(fsys).Execute(config.InstallOptions{
		ProjectDir: projectDir,
		SourceDir:  sourceDir,
		Tools:      []skills.Tool{skills.ToolClaude},
		LinkMode:   skills.LinkCopy,
	}); err != nil {
		t.Fatalf("install: %v", err)
	}

	depthGuardPath := filepath.Join(projectDir, "scripts/lib/check-invocation-depth.sh")
	if _, err := os.Stat(depthGuardPath); err != nil {
		t.Errorf("expected check-invocation-depth.sh to exist after install: %v", err)
	}

	if err := newUninstallSvc(fsys).Execute(projectDir, false); err != nil {
		t.Fatalf("uninstall: %v", err)
	}

	if _, err := os.Stat(depthGuardPath); err == nil {
		t.Error("expected check-invocation-depth.sh to be removed after uninstall")
	}
}

func TestGeminiHookInstalled(t *testing.T) {
	sourceDir := t.TempDir()
	projectDir := t.TempDir()
	setupSourceDir(t, sourceDir)
	mustWriteExecFile(t, filepath.Join(sourceDir, ".gemini/hooks/validate-preload.sh"), "#!/usr/bin/env bash\nexit 0\n")

	fsys := fs.NewOSFileSystem()
	if err := newInstallSvc(fsys).Execute(config.InstallOptions{
		ProjectDir: projectDir,
		SourceDir:  sourceDir,
		Tools:      []skills.Tool{skills.ToolGemini},
		LinkMode:   skills.LinkCopy,
	}); err != nil {
		t.Fatalf("install: %v", err)
	}

	hookPath := filepath.Join(projectDir, ".gemini/hooks/validate-preload.sh")
	info, err := os.Stat(hookPath)
	if err != nil {
		t.Fatalf("expected .gemini/hooks/validate-preload.sh to exist after install: %v", err)
	}
	if info.Mode()&0o111 == 0 {
		t.Errorf("validate-preload.sh deve ser executavel, modo=%v", info.Mode())
	}

	if err := newUninstallSvc(fsys).Execute(projectDir, false); err != nil {
		t.Fatalf("uninstall: %v", err)
	}

	if _, err := os.Stat(hookPath); err == nil {
		t.Error("expected .gemini/hooks/validate-preload.sh to be removed after uninstall")
	}
}

// completeTaskReport e um relatorio de tarefa com todas as secoes e padroes obrigatorios.
const completeTaskReport = `# Relatorio de Execucao

PRD: docs/prd.md
TechSpec: docs/techspec.md
RF-01: requisito implementado

## Contexto Carregado

Contexto carregado do projeto.

## Comandos Executados

- go test ./...

## Arquivos Alterados

- internal/service.go

## Resultados de Validacao

estado: done
testes: pass
lint: pass
veredito do revisor: APPROVED

## Suposicoes

Nenhuma.

## Riscos Residuais

Nenhum.
`

// TestValidateEvidenceCommand verifica a logica do subcomando validate-evidence:
// relatorio completo → Pass=true (exit 0); relatorio incompleto → Pass=false (exit 1).
func TestValidateEvidenceCommand(t *testing.T) {
	result := evidence.Validate([]byte(completeTaskReport), evidence.KindTask, nil)
	if !result.Pass {
		var labels []string
		for _, f := range result.Findings {
			labels = append(labels, f.Label)
		}
		t.Errorf("relatorio completo deveria ser aprovado, findings: %v", labels)
	}

	incomplete := strings.ReplaceAll(completeTaskReport, "## Riscos Residuais\n\nNenhum.\n", "")
	result = evidence.Validate([]byte(incomplete), evidence.KindTask, nil)
	if result.Pass {
		t.Error("relatorio incompleto deveria ser reprovado")
	}
	foundFinding := false
	for _, f := range result.Findings {
		if strings.Contains(f.Label, "Riscos Residuais") {
			foundFinding = true
			break
		}
	}
	if !foundFinding {
		t.Errorf("finding 'Riscos Residuais' esperado, got: %v", result.Findings)
	}
}

// TestCheckSpecDriftCommand verifica a logica do subcomando check-spec-drift:
// todos os IDs cobertos → Pass=true (exit 0); ID faltante → Pass=false + DRIFT (exit 1).
func TestCheckSpecDriftCommand(t *testing.T) {
	tmpDir := t.TempDir()

	prdContent := "# PRD\n\nRF-01: Feature A\nRF-02: Feature B\n"
	sum := sha256.Sum256([]byte(prdContent))
	prdHash := fmt.Sprintf("%x", sum)

	mustWriteFile(t, filepath.Join(tmpDir, "prd.md"), prdContent)
	tasksOK := fmt.Sprintf("## Tasks\n\nRF-01: implementado\nRF-02: implementado\n<!-- spec-hash-prd: %s -->\n", prdHash)
	mustWriteFile(t, filepath.Join(tmpDir, "tasks.md"), tasksOK)

	report, err := specdrift.CheckDrift(tmpDir)
	if err != nil {
		t.Fatalf("CheckDrift: %v", err)
	}
	if !report.Pass {
		t.Errorf("esperado sem drift quando todos os IDs estao cobertos: coverage=%v hashes=%v", report.Coverage, report.Hashes)
	}

	// Remover RF-02 das tasks: drift esperado
	tasksWithDrift := fmt.Sprintf("## Tasks\n\nRF-01: implementado\n<!-- spec-hash-prd: %s -->\n", prdHash)
	mustWriteFile(t, filepath.Join(tmpDir, "tasks.md"), tasksWithDrift)

	report, err = specdrift.CheckDrift(tmpDir)
	if err != nil {
		t.Fatalf("CheckDrift com drift: %v", err)
	}
	if report.Pass {
		t.Error("esperado drift detectado quando RF-02 esta ausente das tasks")
	}
	foundDrift := false
	for _, cov := range report.Coverage {
		for _, missing := range cov.MissingIDs {
			if missing == "RF-02" {
				foundDrift = true
			}
		}
	}
	if !foundDrift {
		t.Errorf("esperado RF-02 como ID faltante, got: %v", report.Coverage)
	}
}

func TestE2E_CopilotInstall_GeneratesEightAgents(t *testing.T) {
	sourceDir := t.TempDir()
	projectDir := t.TempDir()
	setupSourceDir(t, sourceDir)

	// Seed all 8 processual skills in source
	processualSkills := []struct {
		name string
		desc string
	}{
		{"bugfix", "Corrects bugs automatically."},
		{"create-prd", "Creates product requirements documents."},
		{"analyze-project", "Analyzes project architecture and stack."},
		{"refactor", "Refactors code preserving behavior."},
		{"review", "Reviews code and pull requests."},
		{"execute-task", "Executes eligible tasks with validation."},
		{"create-tasks", "Plans and creates task breakdowns."},
		{"create-technical-specification", "Writes technical specifications and ADRs."},
	}
	for _, s := range processualSkills {
		mustWriteFile(t, filepath.Join(sourceDir, ".agents/skills/"+s.name+"/SKILL.md"),
			"---\nname: "+s.name+"\nversion: 1.0.0\ndescription: "+s.desc+"\n---\n")
	}

	fsys := fs.NewOSFileSystem()
	err := newInstallSvc(fsys).Execute(config.InstallOptions{
		ProjectDir: projectDir,
		SourceDir:  sourceDir,
		Tools:      []skills.Tool{skills.ToolCopilot},
		LinkMode:   skills.LinkCopy,
	})
	if err != nil {
		t.Fatalf("install copilot: %v", err)
	}

	agentsDir := filepath.Join(projectDir, ".github", "agents")
	info, err := os.Stat(agentsDir)
	if err != nil || !info.IsDir() {
		t.Fatalf(".github/agents/ should exist after copilot install: %v", err)
	}

	entries, err := os.ReadDir(agentsDir)
	if err != nil {
		t.Fatalf("failed to read .github/agents/: %v", err)
	}

	agentMDFiles := 0
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".agent.md") {
			agentMDFiles++
		}
	}
	if agentMDFiles != 8 {
		t.Errorf("expected 8 .agent.md files in .github/agents/, got %d", agentMDFiles)
		for _, e := range entries {
			t.Logf("  found: %s", e.Name())
		}
	}
}

func TestE2E_GeminiInstall_TomlContent(t *testing.T) {
	sourceDir := t.TempDir()
	projectDir := t.TempDir()
	setupSourceDir(t, sourceDir)

	fsys := fs.NewOSFileSystem()

	err := newInstallSvc(fsys).Execute(config.InstallOptions{
		ProjectDir: projectDir,
		SourceDir:  sourceDir,
		Tools:      []skills.Tool{skills.ToolGemini},
		LinkMode:   skills.LinkCopy,
	})
	if err != nil {
		t.Fatalf("install: %v", err)
	}

	// setupSourceDir seeds a "review" skill; verify its TOML was generated
	tomlPath := filepath.Join(projectDir, ".gemini", "commands", "review.toml")
	data, err := os.ReadFile(tomlPath)
	if err != nil {
		t.Fatalf("expected .gemini/commands/review.toml to exist after install: %v", err)
	}

	content := string(data)
	if !strings.Contains(content, `description =`) {
		t.Errorf("review.toml should have 'description =' field, got: %q", content)
	}
	if !strings.Contains(content, `prompt =`) {
		t.Errorf("review.toml should have 'prompt =' field, got: %q", content)
	}
	if !strings.Contains(content, "SKILL.md") {
		t.Errorf("review.toml prompt should reference SKILL.md, got: %q", content)
	}
	if !strings.Contains(content, "{{args}}") {
		t.Errorf("review.toml prompt should contain {{args}} placeholder, got: %q", content)
	}
}

func TestPythonMonorepoSnapshot(t *testing.T) {
	projectDir := t.TempDir()
	fixtureDir := filepath.Join("..", "..", "testdata", "python-monorepo")

	fsys := fs.NewOSFileSystem()
	g := contextgen.NewGenerator(fsys, output.New(false))

	if err := g.Generate(fixtureDir, projectDir, []skills.Tool{skills.ToolClaude}, nil, "full", false); err != nil {
		t.Fatalf("Generate python-monorepo: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(projectDir, "AGENTS.md"))
	if err != nil {
		t.Fatalf("AGENTS.md nao gerado: %v", err)
	}

	snapshotPath := filepath.Join("..", "..", "testdata", "snapshots", "python-monorepo.agents.md")
	expected, err := os.ReadFile(snapshotPath)
	if err != nil {
		t.Fatalf("snapshot nao encontrado: %s", snapshotPath)
	}

	if string(expected) != string(data) {
		t.Errorf("output diverge do snapshot: %s\nExecte UPDATE_SNAPSHOTS=1 no contextgen_test para atualizar.", snapshotPath)
	}

	// Verificacoes de sanidade para python-monorepo
	content := string(data)
	if !strings.Contains(content, "governance-schema:") {
		t.Error("AGENTS.md deve conter governance-schema")
	}
	if !strings.Contains(content, "agent-governance") {
		t.Error("AGENTS.md deve mencionar agent-governance skill")
	}
}

func TestCodexOnlySnapshot(t *testing.T) {
	projectDir := t.TempDir()
	fixtureDir := filepath.Join("..", "..", "testdata", "fixtures", "codex-only")

	// Copy fixture files to projectDir so architecture detection works correctly
	goModData, err := os.ReadFile(filepath.Join(fixtureDir, "go.mod"))
	if err != nil {
		t.Fatalf("fixture go.mod nao encontrado: %v", err)
	}
	if err := os.WriteFile(filepath.Join(projectDir, "go.mod"), goModData, 0o644); err != nil {
		t.Fatalf("escrever go.mod: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(projectDir, ".codex"), 0o755); err != nil {
		t.Fatalf("mkdir .codex: %v", err)
	}
	codexCfgData, err := os.ReadFile(filepath.Join(fixtureDir, ".codex", "config.toml"))
	if err != nil {
		t.Fatalf("fixture .codex/config.toml nao encontrado: %v", err)
	}
	if err := os.WriteFile(filepath.Join(projectDir, ".codex", "config.toml"), codexCfgData, 0o644); err != nil {
		t.Fatalf("escrever .codex/config.toml: %v", err)
	}

	fsys := fs.NewOSFileSystem()
	g := contextgen.NewGenerator(fsys, output.New(false))

	// tools=[Codex] only → compact profile auto-detected
	if err := g.Generate(projectDir, projectDir, []skills.Tool{skills.ToolCodex}, nil, "full", false); err != nil {
		t.Fatalf("Generate codex-only: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(projectDir, "AGENTS.md"))
	if err != nil {
		t.Fatalf("AGENTS.md nao gerado: %v", err)
	}

	snapshotPath := filepath.Join("..", "..", "testdata", "snapshots", "codex-only.agents.md")

	if os.Getenv("UPDATE_SNAPSHOTS") == "1" {
		if err := os.MkdirAll(filepath.Dir(snapshotPath), 0o755); err != nil {
			t.Fatalf("criar diretorio de snapshots: %v", err)
		}
		if err := os.WriteFile(snapshotPath, data, 0o644); err != nil {
			t.Fatalf("escrever snapshot: %v", err)
		}
		t.Logf("snapshot atualizado: %s", snapshotPath)
		return
	}

	expected, err := os.ReadFile(snapshotPath)
	if err != nil {
		t.Fatalf("snapshot nao encontrado: %s (execute com UPDATE_SNAPSHOTS=1 para criar)", snapshotPath)
	}

	if string(expected) != string(data) {
		t.Errorf("output diverge do snapshot: %s\nExecte UPDATE_SNAPSHOTS=1 para atualizar.", snapshotPath)
	}

	// Sanity checks: compact profile must NOT contain verbose sections
	content := string(data)
	if strings.Contains(content, "## Diretrizes de Estrutura") {
		t.Error("snapshot codex-only nao deve conter '## Diretrizes de Estrutura' (compact profile)")
	}
	if strings.Contains(content, "### Composicao Multi-Linguagem") {
		t.Error("snapshot codex-only nao deve conter '### Composicao Multi-Linguagem' (compact profile)")
	}
	if !strings.Contains(content, "governance-schema:") {
		t.Error("AGENTS.md deve conter governance-schema")
	}
}

func TestUpgradeRecopiesNewArtifacts(t *testing.T) {
	sourceDir := t.TempDir()
	projectDir := t.TempDir()
	setupSourceDirWithEvidenceArtifacts(t, sourceDir)

	fsys := fs.NewOSFileSystem()
	if err := newInstallSvc(fsys).Execute(config.InstallOptions{
		ProjectDir: projectDir,
		SourceDir:  sourceDir,
		Tools:      []skills.Tool{skills.ToolClaude, skills.ToolGemini},
		LinkMode:   skills.LinkCopy,
	}); err != nil {
		t.Fatalf("install: %v", err)
	}

	// Simular divergencia: corromper validate-bugfix-evidence.sh no target
	bugfixScript := filepath.Join(projectDir, ".claude/scripts/validate-bugfix-evidence.sh")
	mustWriteFile(t, bugfixScript, "#!/usr/bin/env bash\n# CORROMPIDO\n")

	// Bump skill version para que o upgrade detecte skill desatualizada e regenere adaptadores
	mustWriteFile(t, filepath.Join(sourceDir, ".agents/skills/review/SKILL.md"),
		"---\nname: review\nversion: 2.0.0\ndescription: Revisa codigo.\n---\n")

	if err := newUpgradeSvc(fsys).Execute(config.UpgradeOptions{
		ProjectDir: projectDir,
		SourceDir:  sourceDir,
	}); err != nil {
		t.Fatalf("upgrade: %v", err)
	}

	// validate-bugfix-evidence.sh deve ser restaurado pela re-sincronizacao de adaptadores
	data, err := os.ReadFile(bugfixScript)
	if err != nil {
		t.Fatalf("validate-bugfix-evidence.sh nao encontrado apos upgrade: %v", err)
	}
	if strings.Contains(string(data), "CORROMPIDO") {
		t.Error("validate-bugfix-evidence.sh deveria ser restaurado pelo upgrade")
	}

	// .gemini/hooks/validate-preload.sh deve ser re-copiado pelo upgrade
	geminiHook := filepath.Join(projectDir, ".gemini/hooks/validate-preload.sh")
	if _, err := os.Stat(geminiHook); err != nil {
		t.Errorf(".gemini/hooks/validate-preload.sh deve existir apos upgrade: %v", err)
	}
}
