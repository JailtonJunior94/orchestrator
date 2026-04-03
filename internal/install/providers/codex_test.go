package providers

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	install "github.com/jailtonjunior/orchestrator/internal/install/domain"
	"github.com/jailtonjunior/orchestrator/internal/platform"
)

func TestCodexAdapterResolveTarget(t *testing.T) {
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
	adapter := NewCodexAdapter(fileSystem, projectRoot, globalRoot)

	skill := mustAsset(t, "codex:skill:reviewer", "reviewer", install.AssetKindSkill, filepath.Join(projectRoot, ".codex", "skills", "reviewer"), install.ProviderCodex)

	projectTarget, err := adapter.ResolveTarget(context.Background(), install.ScopeProject, skill)
	if err != nil {
		t.Fatalf("ResolveTarget(project) error = %v", err)
	}
	if got, want := projectTarget.TargetPath, filepath.Join(projectRoot, ".codex", "skills", "reviewer"); got != want {
		t.Fatalf("project target = %q, want %q", got, want)
	}
	if projectTarget.Verification != install.VerificationStatusPartial {
		t.Fatalf("project verification = %q, want %q", projectTarget.Verification, install.VerificationStatusPartial)
	}

	globalTarget, err := adapter.ResolveTarget(context.Background(), install.ScopeGlobal, skill)
	if err != nil {
		t.Fatalf("ResolveTarget(global) error = %v", err)
	}
	if got, want := globalTarget.TargetPath, filepath.Join(globalRoot, ".codex", "skills", "reviewer"); got != want {
		t.Fatalf("global target = %q, want %q", got, want)
	}
}

func TestCodexAdapterApplyPreservesExistingConfigAndRemovesManagedEntry(t *testing.T) {
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
	adapter := NewCodexAdapter(fileSystem, projectRoot, globalRoot)

	sourcePath := filepath.Join(projectRoot, "assets", "reviewer")
	targetPath := filepath.Join(projectRoot, ".codex", "skills", "reviewer")
	configPath := filepath.Join(projectRoot, ".codex", "config.toml")
	externalPath := filepath.Join(projectRoot, "custom", "external-skill")

	if err := fileSystem.WriteFile(filepath.Join(sourcePath, "SKILL.md"), []byte("updated skill"), 0o644); err != nil {
		t.Fatalf("WriteFile(source) error = %v", err)
	}
	if err := fileSystem.WriteFile(filepath.Join(targetPath, "SKILL.md"), []byte("old skill"), 0o644); err != nil {
		t.Fatalf("WriteFile(target) error = %v", err)
	}
	if err := fileSystem.WriteFile(configPath, []byte(fmt.Sprintf("model = \"gpt-5-codex\"\n\n[[skills.config]]\npath = %q\nenabled = false\n\n[[skills.config]]\npath = %q\nenabled = true\n", targetPath, externalPath)), 0o644); err != nil {
		t.Fatalf("WriteFile(config) error = %v", err)
	}

	updateChange := mustChange(t, install.ProviderCodex, install.AssetKindSkill, install.ActionUpdate, sourcePath, targetPath)
	results, err := adapter.Apply(context.Background(), []install.PlannedChange{updateChange})
	if err != nil {
		t.Fatalf("Apply(update) error = %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("Apply(update) results = %d, want 1", len(results))
	}

	skillBytes, err := fileSystem.ReadFile(filepath.Join(targetPath, "SKILL.md"))
	if err != nil {
		t.Fatalf("ReadFile(updated skill) error = %v", err)
	}
	if got, want := string(skillBytes), "updated skill"; got != want {
		t.Fatalf("updated skill body = %q, want %q", got, want)
	}

	config, _, err := adapter.readConfig(configPath)
	if err != nil {
		t.Fatalf("readConfig(updated) error = %v", err)
	}
	skillPaths, err := codexConfiguredSkillPaths(config)
	if err != nil {
		t.Fatalf("codexConfiguredSkillPaths(updated) error = %v", err)
	}
	if !slices.Contains(skillPaths, targetPath) {
		t.Fatalf("updated config paths = %v, want managed path %q", skillPaths, targetPath)
	}
	if !slices.Contains(skillPaths, externalPath) {
		t.Fatalf("updated config paths = %v, want external path %q", skillPaths, externalPath)
	}

	configBytes, err := fileSystem.ReadFile(configPath)
	if err != nil {
		t.Fatalf("ReadFile(config after update) error = %v", err)
	}
	if !strings.Contains(string(configBytes), "model = 'gpt-5-codex'") && !strings.Contains(string(configBytes), "model = \"gpt-5-codex\"") {
		t.Fatalf("updated config must preserve model key, got %q", string(configBytes))
	}

	removeChange := mustChange(t, install.ProviderCodex, install.AssetKindSkill, install.ActionRemove, sourcePath, targetPath)
	if _, err := adapter.Apply(context.Background(), []install.PlannedChange{removeChange}); err != nil {
		t.Fatalf("Apply(remove) error = %v", err)
	}

	if _, err := fileSystem.Stat(targetPath); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("Stat(removed skill) error = %v, want not exist", err)
	}

	config, _, err = adapter.readConfig(configPath)
	if err != nil {
		t.Fatalf("readConfig(removed) error = %v", err)
	}
	skillPaths, err = codexConfiguredSkillPaths(config)
	if err != nil {
		t.Fatalf("codexConfiguredSkillPaths(removed) error = %v", err)
	}
	if slices.Contains(skillPaths, targetPath) {
		t.Fatalf("removed config paths still contain managed path %q: %v", targetPath, skillPaths)
	}
	if !slices.Contains(skillPaths, externalPath) {
		t.Fatalf("removed config paths = %v, want external path %q", skillPaths, externalPath)
	}
}

func TestCodexAdapterApplyRejectsInvalidConfig(t *testing.T) {
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
	adapter := NewCodexAdapter(fileSystem, projectRoot, globalRoot)

	sourcePath := filepath.Join(projectRoot, "assets", "reviewer")
	targetPath := filepath.Join(projectRoot, ".codex", "skills", "reviewer")
	configPath := filepath.Join(projectRoot, ".codex", "config.toml")

	if err := fileSystem.WriteFile(filepath.Join(sourcePath, "SKILL.md"), []byte("skill"), 0o644); err != nil {
		t.Fatalf("WriteFile(source) error = %v", err)
	}
	if err := fileSystem.WriteFile(configPath, []byte("[skills\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(config) error = %v", err)
	}

	change := mustChange(t, install.ProviderCodex, install.AssetKindSkill, install.ActionInstall, sourcePath, targetPath)
	if _, err := adapter.Apply(context.Background(), []install.PlannedChange{change}); err == nil {
		t.Fatal("Apply() error = nil, want invalid config error")
	} else if !strings.Contains(err.Error(), configPath) {
		t.Fatalf("Apply() error = %v, want path %q", err, configPath)
	}

	if _, err := fileSystem.Stat(targetPath); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("Stat(target after invalid config) error = %v, want not exist", err)
	}
}

func TestCodexAdapterApplyRollsBackSkillWhenConfigWriteFails(t *testing.T) {
	t.Parallel()

	baseFileSystem, err := platform.NewFakeFileSystem()
	if err != nil {
		t.Fatalf("NewFakeFileSystem() error = %v", err)
	}
	t.Cleanup(func() {
		if closeErr := baseFileSystem.Close(); closeErr != nil {
			t.Fatalf("Close() error = %v", closeErr)
		}
	})

	projectRoot := filepath.Join(baseFileSystem.Root(), "repo")
	globalRoot := filepath.Join(baseFileSystem.Root(), "home")
	configPath := filepath.Join(projectRoot, ".codex", "config.toml")
	fileSystem := failingRenameFileSystem{
		FileSystem:  baseFileSystem,
		failOldPath: buildTempPath(configPath, codexConfigQualifier),
		failNewPath: configPath,
	}
	adapter := NewCodexAdapter(fileSystem, projectRoot, globalRoot)

	sourcePath := filepath.Join(projectRoot, "assets", "reviewer")
	targetPath := filepath.Join(projectRoot, ".codex", "skills", "reviewer")

	if err := baseFileSystem.WriteFile(filepath.Join(sourcePath, "SKILL.md"), []byte("updated skill"), 0o644); err != nil {
		t.Fatalf("WriteFile(source) error = %v", err)
	}
	if err := baseFileSystem.WriteFile(filepath.Join(targetPath, "SKILL.md"), []byte("old skill"), 0o644); err != nil {
		t.Fatalf("WriteFile(target) error = %v", err)
	}

	change := mustChange(t, install.ProviderCodex, install.AssetKindSkill, install.ActionUpdate, sourcePath, targetPath)
	if _, err := adapter.Apply(context.Background(), []install.PlannedChange{change}); err == nil {
		t.Fatal("Apply() error = nil, want config write failure")
	}

	targetBytes, err := baseFileSystem.ReadFile(filepath.Join(targetPath, "SKILL.md"))
	if err != nil {
		t.Fatalf("ReadFile(rolled back target) error = %v", err)
	}
	if got, want := string(targetBytes), "old skill"; got != want {
		t.Fatalf("rolled back target body = %q, want %q", got, want)
	}

	if _, err := baseFileSystem.Stat(configPath); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("Stat(config after rollback) error = %v, want not exist", err)
	}
}

func TestCodexAdapterVerifyStatuses(t *testing.T) {
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
	adapter := NewCodexAdapter(fileSystem, projectRoot, globalRoot)

	globalSkillPath := filepath.Join(globalRoot, ".codex", "skills", "reviewer")
	if err := fileSystem.WriteFile(filepath.Join(globalSkillPath, "SKILL.md"), []byte("skill"), 0o644); err != nil {
		t.Fatalf("WriteFile(global skill) error = %v", err)
	}
	if err := fileSystem.WriteFile(filepath.Join(globalRoot, ".codex", "config.toml"), []byte(fmt.Sprintf("[[skills.config]]\npath = %q\nenabled = true\n", globalSkillPath)), 0o644); err != nil {
		t.Fatalf("WriteFile(global config) error = %v", err)
	}

	globalChange, err := install.NewPlannedChange(
		install.ProviderCodex,
		install.ScopeGlobal,
		"codex:skill:reviewer",
		install.ActionInstall,
		filepath.Join(projectRoot, "assets", "reviewer"),
		globalSkillPath,
		globalSkillPath,
		nil,
		install.VerificationStatusPartial,
	)
	if err != nil {
		t.Fatalf("NewPlannedChange(global) error = %v", err)
	}

	report, err := adapter.Verify(context.Background(), []install.PlannedChange{globalChange})
	if err != nil {
		t.Fatalf("Verify(global) error = %v", err)
	}
	if report.Status != install.VerificationStatusPartial {
		t.Fatalf("Verify(global) status = %q, want %q", report.Status, install.VerificationStatusPartial)
	}

	projectSkillPath := filepath.Join(projectRoot, ".codex", "skills", "reviewer")
	if err := fileSystem.WriteFile(filepath.Join(projectSkillPath, "SKILL.md"), []byte("skill"), 0o644); err != nil {
		t.Fatalf("WriteFile(project skill) error = %v", err)
	}
	if err := fileSystem.WriteFile(filepath.Join(projectRoot, ".codex", "config.toml"), []byte(fmt.Sprintf("[[skills.config]]\npath = %q\nenabled = true\n", projectSkillPath)), 0o644); err != nil {
		t.Fatalf("WriteFile(project config) error = %v", err)
	}
	if err := fileSystem.WriteFile(filepath.Join(globalRoot, ".codex", "config.toml"), []byte(fmt.Sprintf("[projects.%q]\ntrust_level = \"untrusted\"\n", projectRoot)), 0o644); err != nil {
		t.Fatalf("WriteFile(user config trust) error = %v", err)
	}

	projectChange, err := install.NewPlannedChange(
		install.ProviderCodex,
		install.ScopeProject,
		"codex:skill:reviewer",
		install.ActionInstall,
		filepath.Join(projectRoot, "assets", "reviewer"),
		projectSkillPath,
		projectSkillPath,
		nil,
		install.VerificationStatusPartial,
	)
	if err != nil {
		t.Fatalf("NewPlannedChange(project) error = %v", err)
	}

	report, err = adapter.Verify(context.Background(), []install.PlannedChange{projectChange})
	if err != nil {
		t.Fatalf("Verify(project) error = %v", err)
	}
	if report.Status != install.VerificationStatusFailed {
		t.Fatalf("Verify(project) status = %q, want %q", report.Status, install.VerificationStatusFailed)
	}
	if !containsTokenCaseInsensitive(strings.Join(report.Details, "\n"), "untrusted") {
		t.Fatalf("Verify(project) details = %v, want untrusted note", report.Details)
	}
}

type failingRenameFileSystem struct {
	platform.FileSystem
	failOldPath string
	failNewPath string
}

func (f failingRenameFileSystem) Rename(oldPath string, newPath string) error {
	if filepath.Clean(oldPath) == filepath.Clean(f.failOldPath) && filepath.Clean(newPath) == filepath.Clean(f.failNewPath) {
		return errors.New("forced rename failure")
	}
	return f.FileSystem.Rename(oldPath, newPath)
}
