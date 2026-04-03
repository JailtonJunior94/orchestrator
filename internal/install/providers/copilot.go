package providers

import (
	"context"
	"fmt"
	"path/filepath"

	install "github.com/jailtonjunior/orchestrator/internal/install/domain"
	"github.com/jailtonjunior/orchestrator/internal/platform"
)

// CopilotAdapter manages install targets for GitHub Copilot.
type CopilotAdapter struct {
	managed *managedAdapter
}

// NewCopilotAdapter creates a Copilot target adapter.
func NewCopilotAdapter(
	fileSystem platform.FileSystem,
	projectRoot string,
	globalRoot string,
) *CopilotAdapter {
	verifier := structuralVerifier{
		provider:          install.ProviderCopilot,
		fs:                fileSystem,
		functionalMessage: "functional verification is not supported for GitHub Copilot; structural verification only",
	}

	return &CopilotAdapter{
		managed: &managedAdapter{
			provider:        install.ProviderCopilot,
			projectRoot:     projectRoot,
			globalRoot:      globalRoot,
			fs:              fileSystem,
			verifier:        verifier,
			backupQualifier: "copilot",
		},
	}
}

// Provider returns the target provider handled by this adapter.
func (a *CopilotAdapter) Provider() install.Provider {
	return a.managed.Provider()
}

// ResolveTarget returns the Copilot destination for a supported asset/scope pair.
func (a *CopilotAdapter) ResolveTarget(_ context.Context, scope install.Scope, asset install.Asset) (ResolvedTarget, error) {
	if err := install.ValidateScope(scope); err != nil {
		return ResolvedTarget{}, err
	}
	if !asset.Supports(install.ProviderCopilot) {
		return ResolvedTarget{}, fmt.Errorf("asset %q does not support provider %q", asset.ID(), install.ProviderCopilot)
	}

	baseRoot, err := a.managed.baseRoot(scope)
	if err != nil {
		return ResolvedTarget{}, err
	}

	var targetPath string
	switch scope {
	case install.ScopeProject:
		switch asset.Kind() {
		case install.AssetKindCommand:
			targetPath = filepath.Join(baseRoot, ".claude", "commands", filepath.Base(asset.SourcePath()))
		case install.AssetKindSkill:
			targetPath = filepath.Join(baseRoot, ".claude", "skills", filepath.Base(asset.SourcePath()))
		case install.AssetKindInstruction:
			if !isAgentsAsset(asset) {
				return ResolvedTarget{}, fmt.Errorf("provider %q does not support project-scoped instruction asset %q", install.ProviderCopilot, asset.ID())
			}
			targetPath = filepath.Join(baseRoot, "AGENTS.md")
		default:
			return ResolvedTarget{}, fmt.Errorf("provider %q does not support asset kind %q", install.ProviderCopilot, asset.Kind())
		}
	case install.ScopeGlobal:
		switch asset.Kind() {
		case install.AssetKindSkill:
			targetPath = filepath.Join(baseRoot, ".copilot", "skills", filepath.Base(asset.SourcePath()))
		case install.AssetKindInstruction:
			if !isCopilotInstructionsAsset(asset) {
				return ResolvedTarget{}, fmt.Errorf("provider %q does not support global instruction asset %q", install.ProviderCopilot, asset.ID())
			}
			targetPath = filepath.Join(baseRoot, ".copilot", "copilot-instructions.md")
		default:
			return ResolvedTarget{}, fmt.Errorf("provider %q does not support global asset kind %q", install.ProviderCopilot, asset.Kind())
		}
	default:
		return ResolvedTarget{}, fmt.Errorf("unsupported scope %q", scope)
	}

	targetPath = filepath.Clean(targetPath)
	return ResolvedTarget{
		TargetPath:   targetPath,
		ManagedPath:  targetPath,
		Verification: install.VerificationStatusPartial,
	}, nil
}

// Apply materializes Copilot assets and protects unmanaged content on remove.
func (a *CopilotAdapter) Apply(ctx context.Context, changes []install.PlannedChange) ([]ApplyResult, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	providerChanges, err := a.managed.validateChanges(changes)
	if err != nil {
		return nil, err
	}

	results := make([]ApplyResult, 0, len(providerChanges))
	for _, change := range providerChanges {
		if err := validateCopilotChange(change); err != nil {
			return nil, err
		}
		if change.Action() == install.ActionSkip {
			continue
		}

		result, err := a.applyChange(change, len(results))
		if err != nil {
			rollbackErr := a.managed.rollback(results)
			if rollbackErr != nil {
				return nil, fmt.Errorf("apply change for provider %q failed: %w; rollback failed: %v", install.ProviderCopilot, err, rollbackErr)
			}
			return nil, fmt.Errorf("apply change for provider %q asset %q target %q: %w", install.ProviderCopilot, change.AssetID(), change.TargetPath(), err)
		}

		results = append(results, result)
	}

	if err := a.managed.cleanupBackups(results); err != nil {
		return nil, err
	}

	return results, nil
}

// Verify checks Copilot targets structurally.
func (a *CopilotAdapter) Verify(ctx context.Context, changes []install.PlannedChange) (VerificationReport, error) {
	return a.managed.Verify(ctx, changes)
}

func (a *CopilotAdapter) applyChange(change install.PlannedChange, index int) (ApplyResult, error) {
	operationTarget := change.TargetPath()
	if change.Action() == install.ActionRemove {
		operationTarget = filepath.Clean(change.ManagedPath())
	}
	backupPath := buildBackupPath(operationTarget, a.managed.backupQualifier, index)

	switch change.Action() {
	case install.ActionInstall, install.ActionUpdate:
		if err := a.managed.replaceTarget(change.SourcePath(), operationTarget, backupPath); err != nil {
			return ApplyResult{}, err
		}
	case install.ActionRemove:
		if filepath.Clean(change.TargetPath()) != filepath.Clean(change.ManagedPath()) {
			exists, err := pathExists(a.managed.fs, change.TargetPath())
			if err != nil {
				return ApplyResult{}, fmt.Errorf("stat resolved remove target %q: %w", change.TargetPath(), err)
			}
			if exists {
				return ApplyResult{}, fmt.Errorf("refusing to remove unmanaged target %q; managed asset is tracked at %q", change.TargetPath(), change.ManagedPath())
			}
		}

		if err := a.managed.removeTarget(operationTarget, backupPath); err != nil {
			return ApplyResult{}, err
		}
	default:
		return ApplyResult{}, fmt.Errorf("unsupported action %q", change.Action())
	}

	resultChange := change
	if operationTarget != change.TargetPath() {
		updatedChange, err := install.NewPlannedChange(
			change.Provider(),
			change.Scope(),
			change.AssetID(),
			change.Action(),
			change.SourcePath(),
			operationTarget,
			change.ManagedPath(),
			change.Conflict(),
			change.Verification(),
		)
		if err != nil {
			return ApplyResult{}, err
		}
		resultChange = updatedChange
	}

	return ApplyResult{
		Change:     resultChange,
		BackupPath: backupPath,
	}, nil
}

func validateCopilotChange(change install.PlannedChange) error {
	targetPath := filepath.Clean(change.TargetPath())
	managedPath := filepath.Clean(change.ManagedPath())

	switch change.Scope() {
	case install.ScopeProject:
		switch {
		case isPathWithin(targetPath, filepath.Join(".claude", "commands")):
			return nil
		case isPathWithin(targetPath, filepath.Join(".claude", "skills")):
			return nil
		case filepath.Base(targetPath) == "AGENTS.md":
			return nil
		case change.Action() == install.ActionRemove && filepath.Base(managedPath) == "AGENTS.md":
			return nil
		default:
			return fmt.Errorf("provider %q only supports project targets under .claude/* or AGENTS.md: %q", install.ProviderCopilot, change.TargetPath())
		}
	case install.ScopeGlobal:
		switch {
		case isPathWithin(targetPath, filepath.Join(".copilot", "skills")):
			return nil
		case filepath.Base(targetPath) == "copilot-instructions.md":
			return nil
		case change.Action() == install.ActionRemove && filepath.Base(managedPath) == "copilot-instructions.md":
			return nil
		default:
			return fmt.Errorf("provider %q only supports global targets under .copilot/skills or .copilot/copilot-instructions.md: %q", install.ProviderCopilot, change.TargetPath())
		}
	default:
		return fmt.Errorf("unsupported scope %q", change.Scope())
	}
}

func isAgentsAsset(asset install.Asset) bool {
	metadata := asset.Metadata()
	return filepath.Base(metadata.RelativePath) == "AGENTS.md" || asset.Name() == "agents"
}

func isCopilotInstructionsAsset(asset install.Asset) bool {
	metadata := asset.Metadata()
	return filepath.Base(metadata.RelativePath) == "copilot-instructions.md" || asset.Name() == "copilot-instructions"
}

func isPathWithin(path string, segment string) bool {
	cleanPath := filepath.Clean(path)
	cleanSegment := filepath.Clean(segment) + string(filepath.Separator)
	return filepath.Base(cleanPath) != cleanSegment && containsTokenCaseInsensitive(cleanPath, cleanSegment)
}
