package doctor

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/JailtonJunior94/ai-spec-harness/internal/adapters"
	"github.com/JailtonJunior94/ai-spec-harness/internal/config"
	"github.com/JailtonJunior94/ai-spec-harness/internal/contextgen"
	"github.com/JailtonJunior94/ai-spec-harness/internal/embedded"
	"github.com/JailtonJunior94/ai-spec-harness/internal/fs"
	"github.com/JailtonJunior94/ai-spec-harness/internal/install"
	"github.com/JailtonJunior94/ai-spec-harness/internal/manifest"
	"github.com/JailtonJunior94/ai-spec-harness/internal/output"
	"github.com/JailtonJunior94/ai-spec-harness/internal/skills"
)

func installRealProjectForHookTests(t *testing.T, tools []skills.Tool) (projectDir string, items []install.VerifyItem) {
	t.Helper()

	sourceDir, cleanup, err := embedded.NewExtractor().ExtractToTempDir()
	if err != nil {
		t.Fatalf("extract embedded assets: %v", err)
	}
	t.Cleanup(cleanup)

	projectDir = t.TempDir()
	fsys := fs.NewOSFileSystem()
	printer := output.New(false)
	mfst := manifest.NewStore(fsys)
	adpt := adapters.NewGenerator(fsys, printer)
	ctxg := contextgen.NewGenerator(fsys, printer)
	svc := install.NewService(fsys, printer, mfst, adpt, ctxg)

	if err := svc.Execute(config.InstallOptions{
		ProjectDir: projectDir,
		SourceDir:  sourceDir,
		Tools:      tools,
		Langs:      []skills.Lang{skills.LangNode},
		LinkMode:   skills.LinkCopy,
	}); err != nil {
		t.Fatalf("install: %v", err)
	}

	items, err = svc.Verify(config.InstallOptions{ProjectDir: projectDir})
	if err != nil {
		t.Fatalf("verify: %v", err)
	}
	return projectDir, items
}

func TestRunHookChecks_FreshInstallHasNoFailures(t *testing.T) {
	svc := setupService(fs.NewFakeFileSystem(), true)
	projectDir, items := installRealProjectForHookTests(t, []skills.Tool{skills.ToolClaude, skills.ToolCodex, skills.ToolCopilot})

	checks := svc.runHookChecks(projectDir, items)
	for _, c := range checks {
		if c.Status == "fail" {
			t.Errorf("check %q unexpectedly failed on a fresh install: %s", c.Name, c.Detail)
		}
	}
}

func TestRunHookChecks_NoInstalledProviderWarnsAndSkips(t *testing.T) {
	svc := setupService(fs.NewFakeFileSystem(), true)
	checks := svc.runHookChecks("/project", nil)
	if len(checks) != 1 {
		t.Fatalf("expected exactly 1 check when no provider is installed, got %d", len(checks))
	}
	if checks[0].Status != "warn" {
		t.Errorf("Status = %q, want warn", checks[0].Status)
	}
}

func TestCheckHookAdapterMissing_DetectsMissingArtifact(t *testing.T) {
	svc := setupService(fs.NewFakeFileSystem(), true)
	projectDir, items := installRealProjectForHookTests(t, []skills.Tool{skills.ToolClaude})

	if err := os.Remove(filepath.Join(projectDir, ".claude", "hooks", "validate-preload.sh")); err != nil {
		t.Fatalf("remove installed artifact: %v", err)
	}

	checks := svc.runHookChecks(projectDir, items)
	check := findCheck(t, checks, "Adapter ausente")
	if check.Status != "fail" {
		t.Fatalf("Status = %q, want fail after deleting an installed artifact; detail=%s", check.Status, check.Detail)
	}
}

func TestCheckHookExecutable_DetectsNonExecutableArtifact(t *testing.T) {
	svc := setupService(fs.NewFakeFileSystem(), true)
	projectDir, items := installRealProjectForHookTests(t, []skills.Tool{skills.ToolClaude})

	target := filepath.Join(projectDir, ".claude", "hooks", "validate-preload.sh")
	if err := os.Chmod(target, 0o644); err != nil {
		t.Fatalf("chmod: %v", err)
	}

	checks := svc.runHookChecks(projectDir, items)
	check := findCheck(t, checks, "Hook executavel")
	if check.Status != "fail" {
		t.Fatalf("Status = %q, want fail after removing the execute bit; detail=%s", check.Status, check.Detail)
	}
}

func TestCheckHookAdapterDivergence_DetectsCorruptedArtifactContent(t *testing.T) {
	svc := setupService(fs.NewFakeFileSystem(), true)
	projectDir, items := installRealProjectForHookTests(t, []skills.Tool{skills.ToolClaude})

	target := filepath.Join(projectDir, ".claude", "hooks", "validate-preload.sh")
	if err := os.WriteFile(target, []byte("#!/usr/bin/env bash\necho corrupted\n"), 0o755); err != nil {
		t.Fatalf("corrupt installed artifact: %v", err)
	}

	checks := svc.runHookChecks(projectDir, items)
	check := findCheck(t, checks, "Divergencia entre contrato e adapter")
	if check.Status != "fail" {
		t.Fatalf("Status = %q, want fail after corrupting the installed artifact's content; detail=%s", check.Status, check.Detail)
	}
}

func TestCheckHookNativeConfigInvalid_DetectsMissingKey(t *testing.T) {
	svc := setupService(fs.NewFakeFileSystem(), true)
	projectDir, items := installRealProjectForHookTests(t, []skills.Tool{skills.ToolClaude})

	settingsPath := filepath.Join(projectDir, ".claude", "settings.json")
	if err := os.WriteFile(settingsPath, []byte(`{"hooks":{}}`), 0o644); err != nil {
		t.Fatalf("corrupt native config: %v", err)
	}

	checks := svc.runHookChecks(projectDir, items)
	check := findCheck(t, checks, "Configuracao do hook invalida")
	if check.Status != "fail" {
		t.Fatalf("Status = %q, want fail after emptying the native config's hooks map; detail=%s", check.Status, check.Detail)
	}
}

func TestCheckHookContractVersion_DetectsUnknownProvider(t *testing.T) {
	svc := setupService(fs.NewFakeFileSystem(), true)
	items := []install.VerifyItem{
		{Tool: skills.Tool("unknown-provider"), Skill: "x", State: install.VerifyStateCurrent, Kind: install.VerifyKindSkill},
	}
	check := svc.checkHookContractVersion(installedToolsFrom(items))
	if check.Status != "fail" {
		t.Fatalf("Status = %q, want fail for a provider absent from the current hook contract registry", check.Status)
	}
}

func TestCheckHookContractVersion_OkForFullyCoveredProvider(t *testing.T) {
	svc := setupService(fs.NewFakeFileSystem(), true)
	check := svc.checkHookContractVersion([]skills.Tool{skills.ToolClaude})
	if check.Status != "ok" {
		t.Fatalf("Status = %q, want ok; detail=%s", check.Status, check.Detail)
	}
}

func TestExecuteWithOptions_ReturnsErrorExitWhenHookCheckFails(t *testing.T) {
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
	installSvc := install.NewService(fsys, printer, mfst, adpt, ctxg)
	if err := installSvc.Execute(config.InstallOptions{
		ProjectDir: projectDir,
		SourceDir:  sourceDir,
		Tools:      []skills.Tool{skills.ToolClaude},
		Langs:      []skills.Lang{skills.LangNode},
		LinkMode:   skills.LinkCopy,
	}); err != nil {
		t.Fatalf("install: %v", err)
	}
	if err := os.Remove(filepath.Join(projectDir, ".claude", "hooks", "validate-preload.sh")); err != nil {
		t.Fatalf("remove installed artifact: %v", err)
	}

	doctorSvc := setupService(fs.NewFakeFileSystem(), true)
	doctorSvc.fs = fsys
	doctorSvc.manifest = mfst
	doctorSvc.install = installSvc

	err = doctorSvc.Execute(projectDir)
	if err == nil {
		t.Fatal("expected a non-nil error (non-zero exit code) when a hook adapter is missing")
	}
}

func findCheck(t *testing.T, checks []Check, name string) Check {
	t.Helper()
	for _, c := range checks {
		if c.Name == name {
			return c
		}
	}
	t.Fatalf("check %q not found among %d checks", name, len(checks))
	return Check{}
}
