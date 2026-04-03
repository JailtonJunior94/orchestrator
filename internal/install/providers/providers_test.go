package providers

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	install "github.com/jailtonjunior/orchestrator/internal/install/domain"
	"github.com/jailtonjunior/orchestrator/internal/platform"
)

func TestClaudeAdapterResolveTarget(t *testing.T) {
	t.Parallel()

	fileSystem, err := platform.NewFakeFileSystem()
	if err != nil {
		t.Fatalf("NewFakeFileSystem() error = %v", err)
	}
	t.Cleanup(func() {
		if closeErr := fileSystem.Close(); closeErr != nil {
			t.Fatalf("Close() error = %v", closeErr)
		}
	})

	projectRoot := filepath.Join(fileSystem.Root(), "repo")
	globalRoot := filepath.Join(fileSystem.Root(), "home")
	adapter := NewClaudeAdapter(fileSystem, projectRoot, globalRoot)

	command := mustAsset(t, "claude:command:review", "review", install.AssetKindCommand, filepath.Join(projectRoot, ".claude", "commands", "review.md"), install.ProviderClaude)
	skill := mustAsset(t, "claude:skill:reviewer", "reviewer", install.AssetKindSkill, filepath.Join(projectRoot, ".claude", "skills", "reviewer"), install.ProviderClaude)

	commandTarget, err := adapter.ResolveTarget(context.Background(), install.ScopeProject, command)
	if err != nil {
		t.Fatalf("ResolveTarget(command) error = %v", err)
	}
	if got, want := commandTarget.TargetPath, filepath.Join(projectRoot, ".claude", "commands", "review.md"); got != want {
		t.Fatalf("command target = %q, want %q", got, want)
	}
	if commandTarget.Verification != install.VerificationStatusPartial {
		t.Fatalf("command verification = %q, want %q", commandTarget.Verification, install.VerificationStatusPartial)
	}

	skillTarget, err := adapter.ResolveTarget(context.Background(), install.ScopeGlobal, skill)
	if err != nil {
		t.Fatalf("ResolveTarget(skill) error = %v", err)
	}
	if got, want := skillTarget.TargetPath, filepath.Join(globalRoot, ".claude", "skills", "reviewer"); got != want {
		t.Fatalf("skill target = %q, want %q", got, want)
	}
}

func TestGeminiAdapterResolveTarget(t *testing.T) {
	t.Parallel()

	fileSystem, err := platform.NewFakeFileSystem()
	if err != nil {
		t.Fatalf("NewFakeFileSystem() error = %v", err)
	}
	t.Cleanup(func() {
		if closeErr := fileSystem.Close(); closeErr != nil {
			t.Fatalf("Close() error = %v", closeErr)
		}
	})

	projectRoot := filepath.Join(fileSystem.Root(), "repo")
	globalRoot := filepath.Join(fileSystem.Root(), "home")
	adapter := NewGeminiAdapter(fileSystem, platform.FakeCommandRunner{}, projectRoot, globalRoot)

	command := mustAsset(t, "gemini:command:review", "review", install.AssetKindCommand, filepath.Join(projectRoot, ".gemini", "commands", "review.md"), install.ProviderGemini)
	skill := mustAsset(t, "gemini:skill:reviewer", "reviewer", install.AssetKindSkill, filepath.Join(projectRoot, ".gemini", "skills", "reviewer"), install.ProviderGemini)

	commandTarget, err := adapter.ResolveTarget(context.Background(), install.ScopeProject, command)
	if err != nil {
		t.Fatalf("ResolveTarget(command) error = %v", err)
	}
	if commandTarget.Verification != install.VerificationStatusPartial {
		t.Fatalf("command verification = %q, want %q", commandTarget.Verification, install.VerificationStatusPartial)
	}

	skillTarget, err := adapter.ResolveTarget(context.Background(), install.ScopeGlobal, skill)
	if err != nil {
		t.Fatalf("ResolveTarget(skill) error = %v", err)
	}
	if skillTarget.Verification != install.VerificationStatusComplete {
		t.Fatalf("skill verification = %q, want %q", skillTarget.Verification, install.VerificationStatusComplete)
	}
	if got, want := skillTarget.TargetPath, filepath.Join(globalRoot, ".gemini", "skills", "reviewer"); got != want {
		t.Fatalf("skill target = %q, want %q", got, want)
	}
}

func TestClaudeAdapterApplyInstallRemoveAndRollback(t *testing.T) {
	t.Parallel()

	fileSystem, err := platform.NewFakeFileSystem()
	if err != nil {
		t.Fatalf("NewFakeFileSystem() error = %v", err)
	}
	t.Cleanup(func() {
		if closeErr := fileSystem.Close(); closeErr != nil {
			t.Fatalf("Close() error = %v", closeErr)
		}
	})

	projectRoot := filepath.Join(fileSystem.Root(), "repo")
	globalRoot := filepath.Join(fileSystem.Root(), "home")
	adapter := NewClaudeAdapter(fileSystem, projectRoot, globalRoot)

	commandSourcePath := filepath.Join(projectRoot, "assets", "review.md")
	if err := fileSystem.WriteFile(commandSourcePath, []byte("updated command"), 0o644); err != nil {
		t.Fatalf("WriteFile(command source) error = %v", err)
	}

	skillSourcePath := filepath.Join(projectRoot, "assets", "reviewer", "SKILL.md")
	if err := fileSystem.WriteFile(skillSourcePath, []byte("skill body"), 0o644); err != nil {
		t.Fatalf("WriteFile(skill source) error = %v", err)
	}

	commandTargetPath := filepath.Join(projectRoot, ".claude", "commands", "review.md")
	if err := fileSystem.WriteFile(commandTargetPath, []byte("old command"), 0o644); err != nil {
		t.Fatalf("WriteFile(existing command target) error = %v", err)
	}

	skillTargetPath := filepath.Join(projectRoot, ".claude", "skills", "reviewer")
	if err := fileSystem.WriteFile(filepath.Join(skillTargetPath, "SKILL.md"), []byte("old skill"), 0o644); err != nil {
		t.Fatalf("WriteFile(existing skill target) error = %v", err)
	}

	updateChange := mustChange(t, install.ProviderClaude, install.AssetKindCommand, install.ActionUpdate, commandSourcePath, commandTargetPath)
	removeChange := mustChange(t, install.ProviderClaude, install.AssetKindSkill, install.ActionRemove, skillSourcePath, skillTargetPath)

	results, err := adapter.Apply(context.Background(), []install.PlannedChange{updateChange, removeChange})
	if err != nil {
		t.Fatalf("Apply() error = %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("Apply() results = %d, want 2", len(results))
	}

	commandBytes, err := fileSystem.ReadFile(commandTargetPath)
	if err != nil {
		t.Fatalf("ReadFile(updated command target) error = %v", err)
	}
	if got, want := string(commandBytes), "updated command"; got != want {
		t.Fatalf("updated command target = %q, want %q", got, want)
	}

	if _, err := fileSystem.Stat(skillTargetPath); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("Stat(removed skill target) error = %v, want not exist", err)
	}

	failingChange := mustChange(t, install.ProviderClaude, install.AssetKindCommand, install.ActionInstall, filepath.Join(projectRoot, "assets", "missing.md"), filepath.Join(projectRoot, ".claude", "commands", "missing.md"))
	if _, err := adapter.Apply(context.Background(), []install.PlannedChange{updateChange, failingChange}); err == nil {
		t.Fatal("Apply() rollback case error = nil, want non-nil")
	}

	commandBytes, err = fileSystem.ReadFile(commandTargetPath)
	if err != nil {
		t.Fatalf("ReadFile(rolled back command target) error = %v", err)
	}
	if got, want := string(commandBytes), "updated command"; got != want {
		t.Fatalf("rolled back command target = %q, want %q", got, want)
	}
}

func TestGeminiAdapterVerify(t *testing.T) {
	t.Parallel()

	fileSystem, err := platform.NewFakeFileSystem()
	if err != nil {
		t.Fatalf("NewFakeFileSystem() error = %v", err)
	}
	t.Cleanup(func() {
		if closeErr := fileSystem.Close(); closeErr != nil {
			t.Fatalf("Close() error = %v", closeErr)
		}
	})

	projectRoot := filepath.Join(fileSystem.Root(), "repo")
	globalRoot := filepath.Join(fileSystem.Root(), "home")

	skillTargetPath := filepath.Join(projectRoot, ".gemini", "skills", "reviewer")
	commandTargetPath := filepath.Join(projectRoot, ".gemini", "commands", "review.md")
	if err := fileSystem.WriteFile(filepath.Join(skillTargetPath, "SKILL.md"), []byte("skill"), 0o644); err != nil {
		t.Fatalf("WriteFile(skill target) error = %v", err)
	}
	if err := fileSystem.WriteFile(commandTargetPath, []byte("command"), 0o644); err != nil {
		t.Fatalf("WriteFile(command target) error = %v", err)
	}

	runner := platform.FakeCommandRunner{
		RunFunc: func(ctx context.Context, name string, args []string, stdin string) (platform.CommandResult, error) {
			if name != "gemini" {
				t.Fatalf("runner name = %q, want gemini", name)
			}
			if strings.Join(args, " ") != "skills list" {
				t.Fatalf("runner args = %v, want [skills list]", args)
			}
			return platform.CommandResult{Stdout: "reviewer\nwriter\n", ExitCode: 0}, nil
		},
	}

	adapter := NewGeminiAdapter(fileSystem, runner, projectRoot, globalRoot)
	skillChange := mustChange(t, install.ProviderGemini, install.AssetKindSkill, install.ActionInstall, filepath.Join(projectRoot, "assets", "reviewer"), skillTargetPath)
	commandChange := mustChange(t, install.ProviderGemini, install.AssetKindCommand, install.ActionInstall, filepath.Join(projectRoot, "assets", "review.md"), commandTargetPath)

	report, err := adapter.Verify(context.Background(), []install.PlannedChange{skillChange})
	if err != nil {
		t.Fatalf("Verify(skill) error = %v", err)
	}
	if report.Status != install.VerificationStatusComplete {
		t.Fatalf("Verify(skill) status = %q, want %q", report.Status, install.VerificationStatusComplete)
	}

	report, err = adapter.Verify(context.Background(), []install.PlannedChange{skillChange, commandChange})
	if err != nil {
		t.Fatalf("Verify(skill+command) error = %v", err)
	}
	if report.Status != install.VerificationStatusPartial {
		t.Fatalf("Verify(skill+command) status = %q, want %q", report.Status, install.VerificationStatusPartial)
	}
}

func TestGeminiAdapterVerifyWithFakeBinaryOnPath(t *testing.T) {
	root := t.TempDir()
	projectRoot := filepath.Join(root, "repo")
	globalRoot := filepath.Join(root, "home")
	skillTargetPath := filepath.Join(projectRoot, ".gemini", "skills", "reviewer")
	if err := os.MkdirAll(skillTargetPath, 0o755); err != nil {
		t.Fatalf("MkdirAll(skill target) error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(skillTargetPath, "SKILL.md"), []byte("skill"), 0o644); err != nil {
		t.Fatalf("WriteFile(skill target) error = %v", err)
	}

	binDir := filepath.Join(root, "bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatalf("MkdirAll(bin dir) error = %v", err)
	}
	createFakeGeminiBinary(t, binDir, "reviewer\nwriter\n")
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	adapter := NewGeminiAdapter(platform.OSFileSystem{}, platform.NewCommandRunner(), projectRoot, globalRoot)
	skillChange := mustChange(t, install.ProviderGemini, install.AssetKindSkill, install.ActionInstall, filepath.Join(projectRoot, "assets", "reviewer"), skillTargetPath)

	report, err := adapter.Verify(context.Background(), []install.PlannedChange{skillChange})
	if err != nil {
		t.Fatalf("Verify() error = %v", err)
	}
	if report.Status != install.VerificationStatusComplete {
		t.Fatalf("Verify() status = %q, want %q", report.Status, install.VerificationStatusComplete)
	}
}

func createFakeGeminiBinary(t *testing.T, dir string, output string) {
	t.Helper()

	name := "gemini"
	body := "#!/bin/sh\nprintf '%s' " + shellSingleQuote(output) + "\n"
	if runtime.GOOS == "windows" {
		name = "gemini.cmd"
		body = "@echo off\r\n<nul set /p =" + output + "\r\n"
	}

	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(body), 0o755); err != nil {
		t.Fatalf("WriteFile(%q) error = %v", path, err)
	}
	if runtime.GOOS != "windows" {
		if err := os.Chmod(path, 0o755); err != nil {
			t.Fatalf("Chmod(%q) error = %v", path, err)
		}
	}
}

func shellSingleQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "'\"'\"'") + "'"
}

func mustAsset(t *testing.T, id string, name string, kind install.AssetKind, sourcePath string, provider install.Provider) install.Asset {
	t.Helper()

	asset, err := install.NewAsset(
		id,
		name,
		kind,
		sourcePath,
		[]install.Provider{provider},
		install.AssetMetadata{
			EntryPath:  sourcePath,
			SourceRoot: filepath.Dir(sourcePath),
		},
	)
	if err != nil {
		t.Fatalf("NewAsset() error = %v", err)
	}
	return asset
}

func mustChange(t *testing.T, provider install.Provider, kind install.AssetKind, action install.Action, sourcePath string, targetPath string) install.PlannedChange {
	t.Helper()

	verification := install.VerificationStatusPartial
	if provider == install.ProviderGemini && kind == install.AssetKindSkill {
		verification = install.VerificationStatusComplete
	}

	change, err := install.NewPlannedChange(
		provider,
		install.ScopeProject,
		string(provider)+":"+string(kind)+":"+filepath.Base(targetPath),
		action,
		sourcePath,
		targetPath,
		targetPath,
		nil,
		verification,
	)
	if err != nil {
		t.Fatalf("NewPlannedChange() error = %v", err)
	}
	return change
}
