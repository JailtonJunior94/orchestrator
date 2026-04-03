package providers

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	install "github.com/jailtonjunior/orchestrator/internal/install/domain"
	"github.com/jailtonjunior/orchestrator/internal/platform"
)

func TestCopilotAdapterResolveTarget(t *testing.T) {
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
	adapter := NewCopilotAdapter(fileSystem, projectRoot, globalRoot)

	command := mustAsset(t, "claude:command:review", "review", install.AssetKindCommand, filepath.Join(projectRoot, ".claude", "commands", "review.md"), install.ProviderCopilot)
	skill := mustAsset(t, "claude:skill:reviewer", "reviewer", install.AssetKindSkill, filepath.Join(projectRoot, ".claude", "skills", "reviewer"), install.ProviderCopilot)
	agents := mustInstructionAsset(t, "agents", filepath.Join(projectRoot, "AGENTS.md"))
	copilotInstructions := mustInstructionAsset(t, "copilot-instructions", filepath.Join(projectRoot, ".github", "copilot-instructions.md"))

	commandTarget, err := adapter.ResolveTarget(context.Background(), install.ScopeProject, command)
	if err != nil {
		t.Fatalf("ResolveTarget(project command) error = %v", err)
	}
	if got, want := commandTarget.TargetPath, filepath.Join(projectRoot, ".claude", "commands", "review.md"); got != want {
		t.Fatalf("project command target = %q, want %q", got, want)
	}

	skillTarget, err := adapter.ResolveTarget(context.Background(), install.ScopeGlobal, skill)
	if err != nil {
		t.Fatalf("ResolveTarget(global skill) error = %v", err)
	}
	if got, want := skillTarget.TargetPath, filepath.Join(globalRoot, ".copilot", "skills", "reviewer"); got != want {
		t.Fatalf("global skill target = %q, want %q", got, want)
	}

	agentsTarget, err := adapter.ResolveTarget(context.Background(), install.ScopeProject, agents)
	if err != nil {
		t.Fatalf("ResolveTarget(project AGENTS) error = %v", err)
	}
	if got, want := agentsTarget.TargetPath, filepath.Join(projectRoot, "AGENTS.md"); got != want {
		t.Fatalf("project AGENTS target = %q, want %q", got, want)
	}

	instructionsTarget, err := adapter.ResolveTarget(context.Background(), install.ScopeGlobal, copilotInstructions)
	if err != nil {
		t.Fatalf("ResolveTarget(global instructions) error = %v", err)
	}
	if got, want := instructionsTarget.TargetPath, filepath.Join(globalRoot, ".copilot", "copilot-instructions.md"); got != want {
		t.Fatalf("global instructions target = %q, want %q", got, want)
	}

	if _, err := adapter.ResolveTarget(context.Background(), install.ScopeGlobal, command); err == nil {
		t.Fatal("ResolveTarget(global command) error = nil, want unsupported error")
	}
	if _, err := adapter.ResolveTarget(context.Background(), install.ScopeProject, copilotInstructions); err == nil {
		t.Fatal("ResolveTarget(project copilot-instructions) error = nil, want unsupported error")
	}
}

func TestCopilotAdapterApplyRemoveUsesManagedPath(t *testing.T) {
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
	adapter := NewCopilotAdapter(fileSystem, projectRoot, globalRoot)

	sourcePath := filepath.Join(projectRoot, "assets", "reviewer")
	managedPath := filepath.Join(projectRoot, ".claude", "skills", "reviewer")
	resolvedTarget := filepath.Join(projectRoot, ".claude", "skills", "renamed-reviewer")

	if err := fileSystem.WriteFile(filepath.Join(sourcePath, "SKILL.md"), []byte("skill"), 0o644); err != nil {
		t.Fatalf("WriteFile(source) error = %v", err)
	}
	if err := fileSystem.WriteFile(filepath.Join(managedPath, "SKILL.md"), []byte("managed"), 0o644); err != nil {
		t.Fatalf("WriteFile(managed target) error = %v", err)
	}

	change, err := install.NewPlannedChange(
		install.ProviderCopilot,
		install.ScopeProject,
		"claude:skill:reviewer",
		install.ActionRemove,
		sourcePath,
		resolvedTarget,
		managedPath,
		nil,
		install.VerificationStatusUnknown,
	)
	if err != nil {
		t.Fatalf("NewPlannedChange() error = %v", err)
	}

	results, err := adapter.Apply(context.Background(), []install.PlannedChange{change})
	if err != nil {
		t.Fatalf("Apply(remove) error = %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("Apply(remove) results = %d, want 1", len(results))
	}
	if got, want := results[0].Change.TargetPath(), managedPath; got != want {
		t.Fatalf("Apply(remove) target = %q, want %q", got, want)
	}

	if _, err := fileSystem.Stat(managedPath); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("Stat(removed managed path) error = %v, want not exist", err)
	}
	if _, err := fileSystem.Stat(resolvedTarget); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("Stat(unused resolved target) error = %v, want not exist", err)
	}
}

func TestCopilotAdapterApplyRejectsRemovalWhenResolvedTargetExists(t *testing.T) {
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
	adapter := NewCopilotAdapter(fileSystem, projectRoot, globalRoot)

	sourcePath := filepath.Join(projectRoot, "assets", "reviewer")
	managedPath := filepath.Join(projectRoot, ".claude", "skills", "reviewer")
	resolvedTarget := filepath.Join(projectRoot, ".claude", "skills", "renamed-reviewer")

	if err := fileSystem.WriteFile(filepath.Join(sourcePath, "SKILL.md"), []byte("skill"), 0o644); err != nil {
		t.Fatalf("WriteFile(source) error = %v", err)
	}
	if err := fileSystem.WriteFile(filepath.Join(managedPath, "SKILL.md"), []byte("managed"), 0o644); err != nil {
		t.Fatalf("WriteFile(managed target) error = %v", err)
	}
	if err := fileSystem.WriteFile(filepath.Join(resolvedTarget, "SKILL.md"), []byte("external"), 0o644); err != nil {
		t.Fatalf("WriteFile(resolved target) error = %v", err)
	}

	change, err := install.NewPlannedChange(
		install.ProviderCopilot,
		install.ScopeProject,
		"claude:skill:reviewer",
		install.ActionRemove,
		sourcePath,
		resolvedTarget,
		managedPath,
		nil,
		install.VerificationStatusUnknown,
	)
	if err != nil {
		t.Fatalf("NewPlannedChange() error = %v", err)
	}

	if _, err := adapter.Apply(context.Background(), []install.PlannedChange{change}); err == nil {
		t.Fatal("Apply(remove) error = nil, want unmanaged target protection")
	}

	if _, err := fileSystem.Stat(managedPath); err != nil {
		t.Fatalf("Stat(managed path) error = %v, want file still present", err)
	}
	if _, err := fileSystem.Stat(resolvedTarget); err != nil {
		t.Fatalf("Stat(external resolved target) error = %v, want file still present", err)
	}
}

func TestCopilotAdapterVerify(t *testing.T) {
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
	adapter := NewCopilotAdapter(fileSystem, projectRoot, globalRoot)

	skillSourcePath := filepath.Join(projectRoot, "assets", "reviewer")
	skillTargetPath := filepath.Join(globalRoot, ".copilot", "skills", "reviewer")
	if err := fileSystem.WriteFile(filepath.Join(skillTargetPath, "SKILL.md"), []byte("skill"), 0o644); err != nil {
		t.Fatalf("WriteFile(skill target) error = %v", err)
	}

	skillChange, err := install.NewPlannedChange(
		install.ProviderCopilot,
		install.ScopeGlobal,
		"claude:skill:reviewer",
		install.ActionInstall,
		skillSourcePath,
		skillTargetPath,
		skillTargetPath,
		nil,
		install.VerificationStatusPartial,
	)
	if err != nil {
		t.Fatalf("NewPlannedChange(skill) error = %v", err)
	}

	report, err := adapter.Verify(context.Background(), []install.PlannedChange{skillChange})
	if err != nil {
		t.Fatalf("Verify() error = %v", err)
	}
	if report.Status != install.VerificationStatusPartial {
		t.Fatalf("Verify() status = %q, want %q", report.Status, install.VerificationStatusPartial)
	}

	missingInstructionChange, err := install.NewPlannedChange(
		install.ProviderCopilot,
		install.ScopeGlobal,
		".github:instruction:copilot-instructions",
		install.ActionInstall,
		filepath.Join(projectRoot, ".github", "copilot-instructions.md"),
		filepath.Join(globalRoot, ".copilot", "copilot-instructions.md"),
		filepath.Join(globalRoot, ".copilot", "copilot-instructions.md"),
		nil,
		install.VerificationStatusPartial,
	)
	if err != nil {
		t.Fatalf("NewPlannedChange(instruction) error = %v", err)
	}

	report, err = adapter.Verify(context.Background(), []install.PlannedChange{skillChange, missingInstructionChange})
	if err != nil {
		t.Fatalf("Verify(missing) error = %v", err)
	}
	if report.Status != install.VerificationStatusFailed {
		t.Fatalf("Verify(missing) status = %q, want %q", report.Status, install.VerificationStatusFailed)
	}
}

func mustInstructionAsset(t *testing.T, name string, sourcePath string) install.Asset {
	t.Helper()

	asset, err := install.NewAsset(
		filepath.Dir(sourcePath)+":instruction:"+name,
		name,
		install.AssetKindInstruction,
		sourcePath,
		[]install.Provider{install.ProviderCopilot},
		install.AssetMetadata{
			EntryPath:     sourcePath,
			SourceRoot:    filepath.Dir(sourcePath),
			RelativePath:  filepath.Clean(sourcePath),
			ProviderHints: []install.Provider{install.ProviderCopilot},
		},
	)
	if err != nil {
		t.Fatalf("NewAsset(instruction) error = %v", err)
	}
	return asset
}
