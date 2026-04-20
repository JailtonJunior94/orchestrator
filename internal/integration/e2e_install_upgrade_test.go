//go:build integration

package integration

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/JailtonJunior94/ai-spec-harness/internal/config"
	"github.com/JailtonJunior94/ai-spec-harness/internal/fs"
	"github.com/JailtonJunior94/ai-spec-harness/internal/manifest"
	"github.com/JailtonJunior94/ai-spec-harness/internal/skills"
)

// setupSourceDirMultiTool estende setupSourceDir com artefatos Gemini e Codex
// para testes que exigem instalacao de multiplas ferramentas.
func setupSourceDirMultiTool(t *testing.T, dir string) {
	t.Helper()
	setupSourceDir(t, dir)
	mustWriteFile(t, filepath.Join(dir, "GEMINI.md"), "# Gemini CLI\n")
	mustWriteExecFile(t, filepath.Join(dir, ".gemini/hooks/validate-preload.sh"), "#!/usr/bin/env bash\nif [[ \"${GOVERNANCE_PRELOAD_CONFIRMED:-}\" != \"1\" ]]; then exit 1; fi\n")
	mustWriteExecFile(t, filepath.Join(dir, ".codex/hooks/validate-preload.sh"), "#!/usr/bin/env bash\nif [[ \"${GOVERNANCE_PRELOAD_CONFIRMED:-}\" != \"1\" ]]; then exit 1; fi\n")
}

// readManifest e um helper que le e desserializa o manifesto do projeto.
func readManifest(t *testing.T, projectDir string) manifest.Manifest {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(projectDir, ".ai_spec_harness.json"))
	if err != nil {
		t.Fatalf("nao foi possivel ler .ai_spec_harness.json: %v", err)
	}
	var mf manifest.Manifest
	if err := json.Unmarshal(data, &mf); err != nil {
		t.Fatalf("manifesto invalido: %v", err)
	}
	return mf
}

// ---- Subtask 14.1: Testes de install em cenarios reais ----

// TestE2E14_Install_AllTools_AllArtifactsPresent verifica que um install completo
// com Claude, Gemini e Codex cria todos os diretorios e artefatos esperados.
func TestE2E14_Install_AllTools_AllArtifactsPresent(t *testing.T) {
	sourceDir := t.TempDir()
	projectDir := t.TempDir()
	setupSourceDirMultiTool(t, sourceDir)

	fsys := fs.NewOSFileSystem()
	err := newInstallSvc(fsys).Execute(config.InstallOptions{
		ProjectDir: projectDir,
		SourceDir:  sourceDir,
		Tools:      []skills.Tool{skills.ToolClaude, skills.ToolGemini, skills.ToolCodex},
		LinkMode:   skills.LinkCopy,
	})
	if err != nil {
		t.Fatalf("install com todas as ferramentas: %v", err)
	}

	expectedDirs := []string{
		filepath.Join(projectDir, ".agents"),
		filepath.Join(projectDir, ".claude"),
		filepath.Join(projectDir, ".gemini"),
		filepath.Join(projectDir, ".codex"),
	}
	for _, d := range expectedDirs {
		info, err := os.Stat(d)
		if err != nil || !info.IsDir() {
			t.Errorf("esperado diretorio %s existir apos install: %v", d, err)
		}
	}

	expectedFiles := []string{
		filepath.Join(projectDir, "AGENTS.md"),
		filepath.Join(projectDir, ".ai_spec_harness.json"),
	}
	for _, f := range expectedFiles {
		if _, err := os.Stat(f); err != nil {
			t.Errorf("esperado arquivo %s existir apos install: %v", f, err)
		}
	}
}

// TestE2E_InstallHooks_GeminiAndCodexHooksInstalledAndBlocking verifica que os hooks
// de preload do Gemini e do Codex sao instalados e bloqueiam execucao sem a variavel.
func TestE2E_InstallHooks_GeminiAndCodexHooksInstalledAndBlocking(t *testing.T) {
	sourceDir := t.TempDir()
	projectDir := t.TempDir()
	setupSourceDirMultiTool(t, sourceDir)

	fsys := fs.NewOSFileSystem()
	err := newInstallSvc(fsys).Execute(config.InstallOptions{
		ProjectDir: projectDir,
		SourceDir:  sourceDir,
		Tools:      []skills.Tool{skills.ToolGemini, skills.ToolCodex},
		LinkMode:   skills.LinkCopy,
	})
	if err != nil {
		t.Fatalf("install com Gemini e Codex: %v", err)
	}

	geminiHook := filepath.Join(projectDir, ".gemini", "hooks", "validate-preload.sh")
	codexHook := filepath.Join(projectDir, ".codex", "hooks", "validate-preload.sh")

	for _, hookPath := range []string{geminiHook, codexHook} {
		info, err := os.Stat(hookPath)
		if err != nil {
			t.Errorf("hook nao instalado: %s: %v", hookPath, err)
			continue
		}
		if info.Mode()&0o100 == 0 {
			t.Errorf("hook deve ser executavel: %s (mode: %o)", hookPath, info.Mode())
		}
	}

	// Verificar que o hook Gemini retorna exit 1 sem GOVERNANCE_PRELOAD_CONFIRMED
	cmd := exec.Command("bash", geminiHook)
	cmd.Env = []string{}
	if err := cmd.Run(); err == nil {
		t.Error("hook Gemini deveria retornar exit 1 sem GOVERNANCE_PRELOAD_CONFIRMED")
	}

	// Verificar que o hook Gemini retorna exit 0 com GOVERNANCE_PRELOAD_CONFIRMED=1
	cmd = exec.Command("bash", geminiHook)
	cmd.Env = []string{"GOVERNANCE_PRELOAD_CONFIRMED=1"}
	if err := cmd.Run(); err != nil {
		t.Errorf("hook Gemini deveria retornar exit 0 com GOVERNANCE_PRELOAD_CONFIRMED=1: %v", err)
	}

	// Verificar que o hook Codex retorna exit 1 sem GOVERNANCE_PRELOAD_CONFIRMED
	cmd = exec.Command("bash", codexHook)
	cmd.Env = []string{}
	if err := cmd.Run(); err == nil {
		t.Error("hook Codex deveria retornar exit 1 sem GOVERNANCE_PRELOAD_CONFIRMED")
	}

	// Verificar que o hook Codex retorna exit 0 com GOVERNANCE_PRELOAD_CONFIRMED=1
	cmd = exec.Command("bash", codexHook)
	cmd.Env = []string{"GOVERNANCE_PRELOAD_CONFIRMED=1"}
	if err := cmd.Run(); err != nil {
		t.Errorf("hook Codex deveria retornar exit 0 com GOVERNANCE_PRELOAD_CONFIRMED=1: %v", err)
	}
}

// TestE2E14_Install_ClaudeOnly_NoGeminiNoCodex verifica que install com apenas Claude
// nao cria diretorios ou artefatos de Gemini e Codex.
func TestE2E14_Install_ClaudeOnly_NoGeminiNoCodex(t *testing.T) {
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
		t.Fatalf("install claude: %v", err)
	}

	// Artefatos Claude devem existir
	if _, err := os.Stat(filepath.Join(projectDir, ".claude")); err != nil {
		t.Errorf(".claude/ deve existir apos install claude: %v", err)
	}
	if _, err := os.Stat(filepath.Join(projectDir, ".agents")); err != nil {
		t.Errorf(".agents/ deve existir apos install claude: %v", err)
	}

	// Artefatos Gemini e Codex NAO devem existir
	if _, err := os.Stat(filepath.Join(projectDir, ".gemini")); err == nil {
		t.Error(".gemini/ nao deve existir apos install apenas claude")
	}
	if _, err := os.Stat(filepath.Join(projectDir, ".codex")); err == nil {
		t.Error(".codex/ nao deve existir apos install apenas claude")
	}
}

// TestE2E14_Install_Idempotent_RealFS verifica que instalar duas vezes sobre o mesmo
// diretorio nao duplica entradas em settings.local.json nem corrompe o manifesto.
func TestE2E14_Install_Idempotent_RealFS(t *testing.T) {
	sourceDir := t.TempDir()
	projectDir := t.TempDir()
	setupSourceDir(t, sourceDir)

	fsys := fs.NewOSFileSystem()
	opts := config.InstallOptions{
		ProjectDir: projectDir,
		SourceDir:  sourceDir,
		Tools:      []skills.Tool{skills.ToolClaude},
		LinkMode:   skills.LinkCopy,
	}

	if err := newInstallSvc(fsys).Execute(opts); err != nil {
		t.Fatalf("primeira execucao: %v", err)
	}

	// Capturar conteudo de settings antes do segundo install
	settingsPath := filepath.Join(projectDir, ".claude", "settings.local.json")
	settingsBefore, err := os.ReadFile(settingsPath)
	if err != nil {
		t.Fatalf("settings.local.json nao encontrado apos primeira execucao: %v", err)
	}

	if err := newInstallSvc(fsys).Execute(opts); err != nil {
		t.Fatalf("segunda execucao: %v", err)
	}

	// settings.local.json nao deve ser sobrescrito na segunda execucao
	settingsAfter, err := os.ReadFile(settingsPath)
	if err != nil {
		t.Fatalf("settings.local.json nao encontrado apos segunda execucao: %v", err)
	}
	if string(settingsBefore) != string(settingsAfter) {
		t.Error("settings.local.json foi modificado pela segunda execucao (install nao e idempotente)")
	}

	// Sem entradas duplicadas de hooks
	content := string(settingsAfter)
	if strings.Count(content, "validate-governance.sh") > 1 {
		t.Error("settings.local.json tem entradas duplicadas de validate-governance.sh")
	}
	if strings.Count(content, "validate-preload.sh") > 1 {
		t.Error("settings.local.json tem entradas duplicadas de validate-preload.sh")
	}

	// Manifesto deve ser JSON valido apos segunda execucao
	mf := readManifest(t, projectDir)
	if mf.Version == "" {
		t.Error("manifesto corrompido: campo version ausente apos segunda execucao")
	}
	if len(mf.Tools) == 0 {
		t.Error("manifesto corrompido: campo tools vazio apos segunda execucao")
	}
}

// ---- Subtask 14.2: Testes de upgrade em cenarios reais ----

// TestE2E14_Upgrade_UpdatesExistingSkill verifica que upgrade atualiza uma skill existente
// quando a versao na fonte e superior a versao instalada no projeto.
func TestE2E14_Upgrade_UpdatesExistingSkill(t *testing.T) {
	sourceDir := t.TempDir()
	projectDir := t.TempDir()
	setupSourceDir(t, sourceDir)

	fsys := fs.NewOSFileSystem()

	if err := newInstallSvc(fsys).Execute(config.InstallOptions{
		ProjectDir: projectDir,
		SourceDir:  sourceDir,
		Tools:      []skills.Tool{skills.ToolClaude},
		LinkMode:   skills.LinkCopy,
	}); err != nil {
		t.Fatalf("install: %v", err)
	}

	// Verificar versao instalada (v1.0.0)
	reviewPath := filepath.Join(projectDir, ".agents", "skills", "review", "SKILL.md")
	dataBefore, err := os.ReadFile(reviewPath)
	if err != nil {
		t.Fatalf("review/SKILL.md nao encontrado apos install: %v", err)
	}
	if !strings.Contains(string(dataBefore), "1.0.0") {
		t.Fatalf("versao 1.0.0 esperada antes do upgrade, conteudo: %q", string(dataBefore))
	}

	// Bump de versao na fonte
	mustWriteFile(t, filepath.Join(sourceDir, ".agents/skills/review/SKILL.md"),
		"---\nname: review\nversion: 2.0.0\ndescription: Revisa codigo (v2).\n---\n")

	if err := newUpgradeSvc(fsys).Execute(config.UpgradeOptions{
		ProjectDir: projectDir,
		SourceDir:  sourceDir,
	}); err != nil {
		t.Fatalf("upgrade: %v", err)
	}

	dataAfter, err := os.ReadFile(reviewPath)
	if err != nil {
		t.Fatalf("review/SKILL.md nao encontrado apos upgrade: %v", err)
	}
	if !strings.Contains(string(dataAfter), "2.0.0") {
		t.Errorf("skill review deveria estar na versao 2.0.0 apos upgrade, conteudo: %q", string(dataAfter))
	}
}

// TestE2E14_Upgrade_WithoutInstall_ErrorClear verifica que upgrade sem instalacao previa
// retorna um erro com mensagem que orienta o usuario a executar install primeiro.
func TestE2E14_Upgrade_WithoutInstall_ErrorClear(t *testing.T) {
	sourceDir := t.TempDir()
	projectDir := t.TempDir()
	setupSourceDir(t, sourceDir)

	fsys := fs.NewOSFileSystem()
	err := newUpgradeSvc(fsys).Execute(config.UpgradeOptions{
		ProjectDir: projectDir,
		SourceDir:  sourceDir,
	})
	if err == nil {
		t.Fatal("esperado erro ao executar upgrade sem instalacao previa")
	}

	errMsg := err.Error()
	if !strings.Contains(errMsg, ".agents/skills") || !strings.Contains(errMsg, "install") {
		t.Errorf("mensagem de erro deveria mencionar .agents/skills/ e install, got: %q", errMsg)
	}
}

// TestE2E14_Upgrade_PreservesLocalCustomizations verifica que upgrade nao remove nem
// sobrescreve arquivos locais que o usuario criou manualmente.
func TestE2E14_Upgrade_PreservesLocalCustomizations(t *testing.T) {
	sourceDir := t.TempDir()
	projectDir := t.TempDir()
	setupSourceDir(t, sourceDir)

	fsys := fs.NewOSFileSystem()

	if err := newInstallSvc(fsys).Execute(config.InstallOptions{
		ProjectDir: projectDir,
		SourceDir:  sourceDir,
		Tools:      []skills.Tool{skills.ToolClaude},
		LinkMode:   skills.LinkCopy,
	}); err != nil {
		t.Fatalf("install: %v", err)
	}

	// Usuario cria arquivo local customizado
	customFile := filepath.Join(projectDir, "AGENTS.local.md")
	customContent := "# Customizacoes Locais\nConteudo do usuario.\n"
	mustWriteFile(t, customFile, customContent)

	// Bump de versao para forcar upgrade (e re-geracao de adaptadores)
	mustWriteFile(t, filepath.Join(sourceDir, ".agents/skills/review/SKILL.md"),
		"---\nname: review\nversion: 2.0.0\ndescription: Revisa codigo.\n---\n")

	if err := newUpgradeSvc(fsys).Execute(config.UpgradeOptions{
		ProjectDir: projectDir,
		SourceDir:  sourceDir,
	}); err != nil {
		t.Fatalf("upgrade: %v", err)
	}

	// Arquivo customizado deve ser preservado intacto
	data, err := os.ReadFile(customFile)
	if err != nil {
		t.Fatalf("AGENTS.local.md deveria ser preservado apos upgrade: %v", err)
	}
	if string(data) != customContent {
		t.Errorf("conteudo de AGENTS.local.md foi alterado pelo upgrade: got %q", string(data))
	}
}

// TestE2E14_Upgrade_SkillVersionChange_ChecksumUpdated verifica que apos upgrade com
// mudanca de versao de skill, o checksum correspondente no manifesto e atualizado.
func TestE2E14_Upgrade_SkillVersionChange_ChecksumUpdated(t *testing.T) {
	sourceDir := t.TempDir()
	projectDir := t.TempDir()
	setupSourceDir(t, sourceDir)

	fsys := fs.NewOSFileSystem()

	if err := newInstallSvc(fsys).Execute(config.InstallOptions{
		ProjectDir: projectDir,
		SourceDir:  sourceDir,
		Tools:      []skills.Tool{skills.ToolClaude},
		LinkMode:   skills.LinkCopy,
	}); err != nil {
		t.Fatalf("install: %v", err)
	}

	mfBefore := readManifest(t, projectDir)
	oldChecksum := mfBefore.Checksums["review"]

	// Bump de versao e conteudo da skill para alterar o hash
	mustWriteFile(t, filepath.Join(sourceDir, ".agents/skills/review/SKILL.md"),
		"---\nname: review\nversion: 2.0.0\ndescription: Revisa codigo (v2 com conteudo diferente).\n---\n# Review v2\n")

	if err := newUpgradeSvc(fsys).Execute(config.UpgradeOptions{
		ProjectDir: projectDir,
		SourceDir:  sourceDir,
	}); err != nil {
		t.Fatalf("upgrade: %v", err)
	}

	mfAfter := readManifest(t, projectDir)
	newChecksum := mfAfter.Checksums["review"]
	if newChecksum == "" {
		t.Fatal("checksum de 'review' ausente no manifesto apos upgrade")
	}
	if newChecksum == oldChecksum {
		t.Errorf("checksum de 'review' deveria ter mudado apos upgrade com novo conteudo, checksum: %q", newChecksum)
	}
}

// ---- Subtask 14.3: Testes de manifest ----

// TestE2E14_Manifest_CreatedAfterInstall verifica que .ai_spec_harness.json e criado
// apos install e contém campos obrigatorios (version, created_at).
func TestE2E14_Manifest_CreatedAfterInstall(t *testing.T) {
	sourceDir := t.TempDir()
	projectDir := t.TempDir()
	setupSourceDir(t, sourceDir)

	fsys := fs.NewOSFileSystem()
	if err := newInstallSvc(fsys).Execute(config.InstallOptions{
		ProjectDir: projectDir,
		SourceDir:  sourceDir,
		Tools:      []skills.Tool{skills.ToolClaude},
		LinkMode:   skills.LinkCopy,
	}); err != nil {
		t.Fatalf("install: %v", err)
	}

	manifestPath := filepath.Join(projectDir, ".ai_spec_harness.json")
	if _, err := os.Stat(manifestPath); err != nil {
		t.Fatalf(".ai_spec_harness.json nao criado apos install: %v", err)
	}

	mf := readManifest(t, projectDir)
	if mf.Version == "" {
		t.Error("manifesto deve conter campo version")
	}
	if mf.CreatedAt.IsZero() {
		t.Error("manifesto deve conter created_at nao-zero")
	}
	if mf.UpdatedAt.IsZero() {
		t.Error("manifesto deve conter updated_at nao-zero")
	}
}

// TestE2E14_Manifest_ReflectsToolsModeSkills verifica que o manifesto reflete corretamente
// as ferramentas instaladas, o modo de link e a lista de skills.
func TestE2E14_Manifest_ReflectsToolsModeSkills(t *testing.T) {
	sourceDir := t.TempDir()
	projectDir := t.TempDir()
	setupSourceDir(t, sourceDir)

	fsys := fs.NewOSFileSystem()
	if err := newInstallSvc(fsys).Execute(config.InstallOptions{
		ProjectDir: projectDir,
		SourceDir:  sourceDir,
		Tools:      []skills.Tool{skills.ToolClaude},
		LinkMode:   skills.LinkCopy,
	}); err != nil {
		t.Fatalf("install: %v", err)
	}

	mf := readManifest(t, projectDir)

	// Verificar ferramentas
	foundClaude := false
	for _, tool := range mf.Tools {
		if tool == skills.ToolClaude {
			foundClaude = true
		}
	}
	if !foundClaude {
		t.Errorf("manifesto deveria listar 'claude' como ferramenta, tools: %v", mf.Tools)
	}

	// Verificar modo de link
	if mf.LinkMode != skills.LinkCopy {
		t.Errorf("manifesto deveria refletir modo 'copy', got: %q", mf.LinkMode)
	}

	// Verificar lista de skills nao-vazia
	if len(mf.Skills) == 0 {
		t.Error("manifesto deveria listar skills instaladas (campo skills nao-vazio)")
	}

	// Verificar source_dir registrado
	if mf.SourceDir == "" {
		t.Error("manifesto deveria registrar source_dir")
	}
}

// TestE2E14_Manifest_IntegrityAfterUpgrade verifica que o manifesto permanece integro e
// consistente (JSON valido, campos presentes) apos um ciclo de install + upgrade.
func TestE2E14_Manifest_IntegrityAfterUpgrade(t *testing.T) {
	sourceDir := t.TempDir()
	projectDir := t.TempDir()
	setupSourceDir(t, sourceDir)

	fsys := fs.NewOSFileSystem()

	if err := newInstallSvc(fsys).Execute(config.InstallOptions{
		ProjectDir: projectDir,
		SourceDir:  sourceDir,
		Tools:      []skills.Tool{skills.ToolClaude},
		LinkMode:   skills.LinkCopy,
	}); err != nil {
		t.Fatalf("install: %v", err)
	}

	mfBefore := readManifest(t, projectDir)

	// Bump de versao para acionar atualizacao do manifesto no upgrade
	mustWriteFile(t, filepath.Join(sourceDir, ".agents/skills/review/SKILL.md"),
		"---\nname: review\nversion: 2.0.0\ndescription: Revisa codigo.\n---\n")

	if err := newUpgradeSvc(fsys).Execute(config.UpgradeOptions{
		ProjectDir: projectDir,
		SourceDir:  sourceDir,
	}); err != nil {
		t.Fatalf("upgrade: %v", err)
	}

	mfAfter := readManifest(t, projectDir)

	// Campos obrigatorios devem estar presentes
	if mfAfter.Version == "" {
		t.Error("manifesto corrompido apos upgrade: campo version ausente")
	}
	if len(mfAfter.Tools) == 0 {
		t.Error("manifesto corrompido apos upgrade: campo tools vazio")
	}
	if len(mfAfter.Skills) == 0 {
		t.Error("manifesto corrompido apos upgrade: campo skills vazio")
	}

	// CreatedAt deve ser preservado (nao zerado pelo upgrade)
	if mfAfter.CreatedAt.IsZero() {
		t.Error("manifesto corrompido apos upgrade: created_at zerado")
	}
	if mfAfter.CreatedAt != mfBefore.CreatedAt {
		t.Errorf("created_at nao deve ser alterado pelo upgrade: antes=%v, depois=%v",
			mfBefore.CreatedAt, mfAfter.CreatedAt)
	}

	// UpdatedAt deve ser nao-zero e >= CreatedAt
	if mfAfter.UpdatedAt.IsZero() {
		t.Error("manifesto corrompido apos upgrade: updated_at zerado")
	}
	if mfAfter.UpdatedAt.Before(mfAfter.CreatedAt) {
		t.Errorf("updated_at nao pode ser anterior a created_at: updated=%v, created=%v",
			mfAfter.UpdatedAt, mfAfter.CreatedAt)
	}
}
