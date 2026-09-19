//go:build integration

package integration_test

import (
	"os"
	"os/exec"
	"strconv"
	"strings"
	"testing"

	"github.com/JailtonJunior94/ai-spec-harness/internal/adapters"
	"github.com/JailtonJunior94/ai-spec-harness/internal/config"
	"github.com/JailtonJunior94/ai-spec-harness/internal/contextgen"
	"github.com/JailtonJunior94/ai-spec-harness/internal/doctor"
	"github.com/JailtonJunior94/ai-spec-harness/internal/fs"
	"github.com/JailtonJunior94/ai-spec-harness/internal/git"
	"github.com/JailtonJunior94/ai-spec-harness/internal/install"
	"github.com/JailtonJunior94/ai-spec-harness/internal/manifest"
	"github.com/JailtonJunior94/ai-spec-harness/internal/output"
	"github.com/JailtonJunior94/ai-spec-harness/internal/skills"
	"github.com/JailtonJunior94/ai-spec-harness/internal/upgrade"
)

const legacyManifestVersion = "2.0.1"

func buildLegacyV201Project(t *testing.T) (string, *manifest.Store) {
	t.Helper()
	projectDir := t.TempDir()

	fsys := fs.NewOSFileSystem()
	printer := output.New(false)
	mfst := manifest.NewStore(fsys)
	adpt := adapters.NewGenerator(fsys, printer)
	ctxg := contextgen.NewGenerator(fsys, printer)
	installSvc := install.NewService(fsys, printer, mfst, adpt, ctxg)

	if err := installSvc.Execute(config.InstallOptions{
		ProjectDir:  projectDir,
		Tools:       []skills.Tool{skills.ToolClaude},
		LinkMode:    skills.LinkCopy,
		GenerateCtx: false,
	}); err != nil {
		t.Fatalf("instalacao real (fixture base) falhou: %v", err)
	}

	mf, err := mfst.Load(projectDir)
	if err != nil {
		t.Fatalf("carregar manifesto recem instalado: %v", err)
	}
	if mf.FileChecksums == nil {
		t.Fatalf("instalacao atual deveria gravar file_checksums (RF-24); fixture nao reflete o binario novo")
	}

	mf.Version = legacyManifestVersion
	mf.FileChecksums = nil
	if err := mfst.Save(projectDir, mf); err != nil {
		t.Fatalf("regravar manifesto envelhecido para v2.0.1: %v", err)
	}

	aged, err := mfst.Load(projectDir)
	if err != nil {
		t.Fatalf("recarregar manifesto envelhecido: %v", err)
	}
	if aged.FileChecksums != nil {
		t.Fatalf("manifesto envelhecido nao deveria ter file_checksums; got %v", aged.FileChecksums)
	}
	if aged.Version != legacyManifestVersion {
		t.Fatalf("manifesto envelhecido deveria declarar version=%s; got %s", legacyManifestVersion, aged.Version)
	}

	return projectDir, mfst
}

func TestLegacyV201Project_DoctorWorksWithoutAnyUserAction(t *testing.T) {
	projectDir, mfst := buildLegacyV201Project(t)

	fsys := fs.NewOSFileSystem()
	printer := output.New(false)
	doctorSvc := doctor.NewService(fsys, printer, mfst, git.NewCLIRepository())

	if err := doctorSvc.Execute(projectDir); err != nil {
		t.Fatalf("doctor deveria funcionar sem nenhuma acao do usuario num projeto v2.0.1; err=%v", err)
	}
}

func TestLegacyV201Project_UpgradeSucceedsWithZeroFlags(t *testing.T) {
	projectDir, mfst := buildLegacyV201Project(t)

	fsys := fs.NewOSFileSystem()
	printer := output.New(false)
	adpt := adapters.NewGenerator(fsys, printer)
	ctxg := contextgen.NewGenerator(fsys, printer)
	upgradeSvc := upgrade.NewService(fsys, printer, mfst, adpt, ctxg)

	if err := upgradeSvc.Execute(config.UpgradeOptions{ProjectDir: projectDir}); err != nil {
		t.Fatalf("upgrade com zero flags deveria bastar para migrar um projeto v2.0.1; err=%v", err)
	}

	after, err := mfst.Load(projectDir)
	if err != nil {
		t.Fatalf("carregar manifesto pos-upgrade: %v", err)
	}
	if after.FileChecksums == nil || len(after.FileChecksums) == 0 {
		t.Fatalf("upgrade deveria backfillar file_checksums; got %v", after.FileChecksums)
	}
	if after.Version == legacyManifestVersion {
		t.Fatalf("upgrade deveria ter atualizado version alem de %s", legacyManifestVersion)
	}
}

func runGit(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s falhou: %v\n%s", strconv.Quote(strings.Join(args, " ")), err, out)
	}
	return string(out)
}

func commitCount(t *testing.T, dir string) int {
	t.Helper()
	cmd := exec.Command("git", "rev-list", "--count", "HEAD")
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git rev-list --count HEAD falhou: %v\n%s", err, out)
	}
	n, err := strconv.Atoi(strings.TrimSpace(string(out)))
	if err != nil {
		t.Fatalf("parse contagem de commits %q: %v", out, err)
	}
	return n
}

func TestNoImplicitGitOperations_InstallUpgradeDoctor(t *testing.T) {
	projectDir := t.TempDir()

	runGit(t, projectDir, "init", "--initial-branch=main")
	runGit(t, projectDir, "config", "user.email", "test@example.com")
	runGit(t, projectDir, "config", "user.name", "Test")
	runGit(t, projectDir, "config", "commit.gpgsign", "false")
	writeFile(t, projectDir+"/README.md", "fixture inicial")
	runGit(t, projectDir, "add", "README.md")
	runGit(t, projectDir, "commit", "-m", "initial")

	baseline := commitCount(t, projectDir)
	if baseline != 1 {
		t.Fatalf("setup do fixture deveria ter exatamente 1 commit; got %d", baseline)
	}

	fsys := fs.NewOSFileSystem()
	printer := output.New(false)
	mfst := manifest.NewStore(fsys)
	adpt := adapters.NewGenerator(fsys, printer)
	ctxg := contextgen.NewGenerator(fsys, printer)
	installSvc := install.NewService(fsys, printer, mfst, adpt, ctxg)

	if err := installSvc.Execute(config.InstallOptions{
		ProjectDir:  projectDir,
		Tools:       []skills.Tool{skills.ToolClaude},
		LinkMode:    skills.LinkCopy,
		GenerateCtx: false,
	}); err != nil {
		t.Fatalf("instalacao falhou: %v", err)
	}
	if got := commitCount(t, projectDir); got != baseline {
		t.Fatalf("install criou commit(s) implicitamente: baseline=%d, apos install=%d", baseline, got)
	}

	upgradeSvc := upgrade.NewService(fsys, printer, mfst, adpt, ctxg)
	if err := upgradeSvc.Execute(config.UpgradeOptions{ProjectDir: projectDir}); err != nil {
		t.Fatalf("upgrade (sincronizacao) falhou: %v", err)
	}
	if got := commitCount(t, projectDir); got != baseline {
		t.Fatalf("upgrade criou commit(s) implicitamente: baseline=%d, apos upgrade=%d", baseline, got)
	}

	doctorSvc := doctor.NewService(fsys, printer, mfst, git.NewCLIRepository())
	_ = doctorSvc.Execute(projectDir)
	if got := commitCount(t, projectDir); got != baseline {
		t.Fatalf("doctor criou commit(s) implicitamente: baseline=%d, apos doctor=%d", baseline, got)
	}

	remotes := runGit(t, projectDir, "remote")
	if strings.TrimSpace(remotes) != "" {
		t.Fatalf("nenhum dos fluxos deveria ter adicionado remote; got %q", remotes)
	}

	status := runGit(t, projectDir, "status", "--porcelain")
	if strings.TrimSpace(status) == "" {
		t.Fatalf("instalacao deveria ter gravado arquivos nao commitados no working tree; git status veio vazio")
	}
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("escrever %s: %v", path, err)
	}
}
