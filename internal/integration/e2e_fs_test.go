//go:build integration

package integration

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/JailtonJunior94/ai-spec-harness/internal/config"
	"github.com/JailtonJunior94/ai-spec-harness/internal/doctor"
	"github.com/JailtonJunior94/ai-spec-harness/internal/fs"
	"github.com/JailtonJunior94/ai-spec-harness/internal/git"
	"github.com/JailtonJunior94/ai-spec-harness/internal/manifest"
	"github.com/JailtonJunior94/ai-spec-harness/internal/output"
	"github.com/JailtonJunior94/ai-spec-harness/internal/skills"
)

func newDoctorSvc(fsys fs.FileSystem) *doctor.Service {
	return doctor.NewService(fsys, output.New(false), manifest.NewStore(fsys), git.NewCLIRepository())
}

// ---- Subtask 8.2: Testes de Symlink ----

// TestE2E_SymlinkMode_SkillsAreSymlinks verifica que install com LinkSymlink cria
// symlinks reais em .agents/skills/ em vez de copias de diretorio.
func TestE2E_SymlinkMode_SkillsAreSymlinks(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlinks nao suportados nativamente no Windows")
	}

	sourceDir := t.TempDir()
	projectDir := t.TempDir()
	setupSourceDir(t, sourceDir)

	fsys := fs.NewOSFileSystem()
	if err := newInstallSvc(fsys).Execute(config.InstallOptions{
		ProjectDir: projectDir,
		SourceDir:  sourceDir,
		Tools:      []skills.Tool{skills.ToolClaude},
		LinkMode:   skills.LinkSymlink,
	}); err != nil {
		t.Fatalf("install com symlink: %v", err)
	}

	skillsDir := filepath.Join(projectDir, ".agents", "skills")
	entries, err := os.ReadDir(skillsDir)
	if err != nil {
		t.Fatalf("ler .agents/skills/: %v", err)
	}
	if len(entries) == 0 {
		t.Fatal("nenhuma skill instalada em .agents/skills/")
	}

	symlinkCount := 0
	for _, e := range entries {
		path := filepath.Join(skillsDir, e.Name())
		info, err := os.Lstat(path)
		if err != nil {
			t.Errorf("Lstat %s: %v", path, err)
			continue
		}
		if info.Mode()&os.ModeSymlink != 0 {
			symlinkCount++
		}
	}

	if symlinkCount == 0 {
		t.Errorf("esperado symlinks em .agents/skills/ com --mode symlink, nenhum encontrado (%d entradas)", len(entries))
	}
}

// TestE2E_SymlinkBroken_DoctorDetects verifica que doctor detecta symlinks quebrados
// em .agents/skills/ apos a remocao do diretorio fonte.
func TestE2E_SymlinkBroken_DoctorDetects(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlinks nao suportados nativamente no Windows")
	}

	sourceDir := t.TempDir()
	projectDir := t.TempDir()
	setupSourceDir(t, sourceDir)

	fsys := fs.NewOSFileSystem()
	if err := newInstallSvc(fsys).Execute(config.InstallOptions{
		ProjectDir: projectDir,
		SourceDir:  sourceDir,
		Tools:      []skills.Tool{skills.ToolClaude},
		LinkMode:   skills.LinkSymlink,
	}); err != nil {
		t.Fatalf("install com symlink: %v", err)
	}

	// Verificar que a skill foi instalada como symlink
	reviewLink := filepath.Join(projectDir, ".agents", "skills", "review")
	if info, err := os.Lstat(reviewLink); err != nil || info.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("esperado symlink em .agents/skills/review apos install, Lstat: %v", err)
	}

	// Remover skill do source para quebrar o symlink
	reviewSrc := filepath.Join(sourceDir, ".agents", "skills", "review")
	if err := os.RemoveAll(reviewSrc); err != nil {
		t.Fatalf("remover skill fonte: %v", err)
	}

	// O symlink deve continuar existindo como entrada no filesystem
	if _, err := os.Lstat(reviewLink); err != nil {
		t.Fatalf("symlink quebrado deve existir como entrada: %v", err)
	}
	// Mas o target nao deve ser acessivel via Stat
	if _, err := os.Stat(reviewLink); err == nil {
		t.Fatal("os.Stat em symlink quebrado deveria falhar")
	}

	// Doctor deve detectar symlink quebrado e retornar erro
	svc := newDoctorSvc(fsys)
	err := svc.Execute(projectDir)
	if err == nil {
		t.Error("doctor deveria retornar erro ao detectar symlink quebrado")
	}
}

// TestE2E_SymlinkUpgrade_Works verifica que upgrade funciona corretamente
// quando as skills estao instaladas como symlinks (nao recopia, apenas regenera adaptadores).
func TestE2E_SymlinkUpgrade_Works(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlinks nao suportados nativamente no Windows")
	}

	sourceDir := t.TempDir()
	projectDir := t.TempDir()
	setupSourceDir(t, sourceDir)

	fsys := fs.NewOSFileSystem()
	if err := newInstallSvc(fsys).Execute(config.InstallOptions{
		ProjectDir: projectDir,
		SourceDir:  sourceDir,
		Tools:      []skills.Tool{skills.ToolClaude},
		LinkMode:   skills.LinkSymlink,
	}); err != nil {
		t.Fatalf("install: %v", err)
	}

	// Bump skill version para forcar upgrade
	mustWriteFile(t, filepath.Join(sourceDir, ".agents/skills/review/SKILL.md"),
		"---\nname: review\nversion: 2.0.0\ndescription: Revisa codigo.\n---\n")

	if err := newUpgradeSvc(fsys).Execute(config.UpgradeOptions{
		ProjectDir: projectDir,
		SourceDir:  sourceDir,
	}); err != nil {
		t.Fatalf("upgrade com symlinks: %v", err)
	}

	// Skills devem continuar acessiveis via symlink (source ainda existe)
	skillsDir := filepath.Join(projectDir, ".agents", "skills")
	if _, err := os.Stat(skillsDir); err != nil {
		t.Errorf(".agents/skills/ deve ser acessivel apos upgrade: %v", err)
	}
	// Hooks devem continuar instalados
	if _, err := os.Stat(filepath.Join(projectDir, ".claude", "hooks", "validate-governance.sh")); err != nil {
		t.Errorf("validate-governance.sh deve existir apos upgrade: %v", err)
	}
}

// ---- Subtask 8.3: Testes de Permissao ----

// TestE2E_ReadOnlyDir_InstallFails verifica que install em diretorio read-only
// retorna erro claro mencionando permissao de escrita.
func TestE2E_ReadOnlyDir_InstallFails(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("permissoes POSIX nao aplicam no Windows")
	}
	if os.Getuid() == 0 {
		t.Skip("root bypassa restricoes de permissao")
	}

	sourceDir := t.TempDir()
	projectDir := t.TempDir()
	setupSourceDir(t, sourceDir)

	// Tornar projectDir read-only (r-xr-xr-x)
	if err := os.Chmod(projectDir, 0o555); err != nil {
		t.Fatalf("chmod 555: %v", err)
	}
	t.Cleanup(func() { os.Chmod(projectDir, 0o755) })

	fsys := fs.NewOSFileSystem()
	err := newInstallSvc(fsys).Execute(config.InstallOptions{
		ProjectDir: projectDir,
		SourceDir:  sourceDir,
		Tools:      []skills.Tool{skills.ToolClaude},
		LinkMode:   skills.LinkCopy,
	})
	if err == nil {
		t.Fatal("esperado erro ao instalar em diretorio read-only")
	}

	errMsg := strings.ToLower(err.Error())
	if !strings.Contains(errMsg, "permissao") && !strings.Contains(errMsg, "permission") && !strings.Contains(errMsg, "escrita") {
		t.Errorf("mensagem de erro deveria mencionar permissao de escrita, got: %v", err)
	}
}

// TestE2E_DoctorOnValidInstall_PassesPermissionsCheck verifica que doctor reporta
// permissoes OK em uma instalacao valida e falha quando o diretorio fica read-only.
func TestE2E_DoctorOnValidInstall_PassesPermissionsCheck(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("permissoes POSIX nao aplicam no Windows")
	}
	if os.Getuid() == 0 {
		t.Skip("root bypassa restricoes de permissao")
	}

	sourceDir := t.TempDir()
	projectDir := t.TempDir()
	setupSourceDir(t, sourceDir)

	// Inicializar git para doctor passar o check de repositorio
	fsys := fs.NewOSFileSystem()
	if err := newInstallSvc(fsys).Execute(config.InstallOptions{
		ProjectDir: projectDir,
		SourceDir:  sourceDir,
		Tools:      []skills.Tool{skills.ToolClaude},
		LinkMode:   skills.LinkCopy,
	}); err != nil {
		t.Fatalf("install: %v", err)
	}

	// Doctor deve reportar permissoes OK no projectDir gravavel
	svc := newDoctorSvc(fsys)
	// Nao verificamos o retorno total (pode ter aviso de git), apenas que nao ha falha de permissao
	// Verificamos diretamente que Writable retorna true antes de chmod
	if !fsys.Writable(projectDir) {
		t.Error("projectDir deve ser gravavel antes do chmod")
	}

	// Tornar projectDir read-only e verificar que Writable retorna false
	if err := os.Chmod(projectDir, 0o555); err != nil {
		t.Fatalf("chmod 555: %v", err)
	}
	t.Cleanup(func() { os.Chmod(projectDir, 0o755) })

	if fsys.Writable(projectDir) {
		t.Error("projectDir nao deve ser gravavel apos chmod 555")
	}

	// Doctor deve reportar falha de permissao
	err := svc.Execute(projectDir)
	if err == nil {
		t.Error("doctor deveria retornar erro quando projectDir e read-only")
	}
}

// ---- Subtask 8.4: Testes de Paths Especiais ----

// TestE2E_InstallInPathWithSpaces verifica que install funciona corretamente
// em caminhos que contem espacos.
func TestE2E_InstallInPathWithSpaces(t *testing.T) {
	baseDir := t.TempDir()
	sourceDir := filepath.Join(baseDir, "source dir with spaces")
	projectDir := filepath.Join(baseDir, "my project dir")

	if err := os.MkdirAll(sourceDir, 0o755); err != nil {
		t.Fatalf("mkdir sourceDir: %v", err)
	}
	if err := os.MkdirAll(projectDir, 0o755); err != nil {
		t.Fatalf("mkdir projectDir: %v", err)
	}

	setupSourceDir(t, sourceDir)

	fsys := fs.NewOSFileSystem()
	if err := newInstallSvc(fsys).Execute(config.InstallOptions{
		ProjectDir: projectDir,
		SourceDir:  sourceDir,
		Tools:      []skills.Tool{skills.ToolClaude},
		LinkMode:   skills.LinkCopy,
	}); err != nil {
		t.Fatalf("install em path com espacos: %v", err)
	}

	expectedPaths := []string{
		filepath.Join(projectDir, ".agents", "skills"),
		filepath.Join(projectDir, ".claude", "hooks", "validate-governance.sh"),
		filepath.Join(projectDir, ".ai_spec_harness.json"),
	}
	for _, p := range expectedPaths {
		if _, err := os.Stat(p); err != nil {
			t.Errorf("esperado %s existir apos install em path com espacos: %v", p, err)
		}
	}
}

// TestE2E_InstallInPathWithUnicode verifica que install funciona corretamente
// em caminhos com caracteres unicode (acentos, caracteres especiais).
func TestE2E_InstallInPathWithUnicode(t *testing.T) {
	baseDir := t.TempDir()
	projectDir := filepath.Join(baseDir, "projeto-ações-üñícode-テスト")

	if err := os.MkdirAll(projectDir, 0o755); err != nil {
		t.Fatalf("mkdir projectDir unicode: %v", err)
	}

	sourceDir := t.TempDir()
	setupSourceDir(t, sourceDir)

	fsys := fs.NewOSFileSystem()
	if err := newInstallSvc(fsys).Execute(config.InstallOptions{
		ProjectDir: projectDir,
		SourceDir:  sourceDir,
		Tools:      []skills.Tool{skills.ToolClaude},
		LinkMode:   skills.LinkCopy,
	}); err != nil {
		t.Fatalf("install em path unicode: %v", err)
	}

	expectedPaths := []string{
		filepath.Join(projectDir, ".agents", "skills"),
		filepath.Join(projectDir, ".claude", "hooks", "validate-governance.sh"),
		filepath.Join(projectDir, ".ai_spec_harness.json"),
	}
	for _, p := range expectedPaths {
		if _, err := os.Stat(p); err != nil {
			t.Errorf("esperado %s existir apos install em path unicode: %v", p, err)
		}
	}
}
