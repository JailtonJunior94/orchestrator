package providers

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"

	install "github.com/jailtonjunior/orchestrator/internal/install/domain"
	"github.com/jailtonjunior/orchestrator/internal/platform"
)

// ResolvedTarget stores the provider-specific destination for an asset.
type ResolvedTarget struct {
	TargetPath   string
	ManagedPath  string
	Verification install.VerificationStatus
}

// ApplyResult records a successfully applied change.
type ApplyResult struct {
	Change     install.PlannedChange
	BackupPath string
}

// VerificationReport captures the provider verification outcome.
type VerificationReport struct {
	Provider install.Provider
	Status   install.VerificationStatus
	Details  []string
}

// TargetAdapter encapsulates planning, apply and verify for a provider.
type TargetAdapter interface {
	Provider() install.Provider
	ResolveTarget(ctx context.Context, scope install.Scope, asset install.Asset) (ResolvedTarget, error)
	Apply(ctx context.Context, changes []install.PlannedChange) ([]ApplyResult, error)
	Verify(ctx context.Context, changes []install.PlannedChange) (VerificationReport, error)
}

// Registry indexes provider adapters by provider.
type Registry struct {
	adapters map[install.Provider]TargetAdapter
}

// NewRegistry builds a provider adapter registry.
func NewRegistry(adapters ...TargetAdapter) (*Registry, error) {
	index := make(map[install.Provider]TargetAdapter, len(adapters))
	for _, adapter := range adapters {
		if adapter == nil {
			return nil, errors.New("target adapter must not be nil")
		}

		provider := adapter.Provider()
		if err := install.ValidateProvider(provider); err != nil {
			return nil, err
		}
		if _, exists := index[provider]; exists {
			return nil, fmt.Errorf("duplicate target adapter for provider %q", provider)
		}

		index[provider] = adapter
	}

	return &Registry{adapters: index}, nil
}

// Adapter returns the adapter for the requested provider.
func (r *Registry) Adapter(provider install.Provider) (TargetAdapter, error) {
	if r == nil {
		return nil, errors.New("registry must not be nil")
	}
	if err := install.ValidateProvider(provider); err != nil {
		return nil, err
	}

	adapter, ok := r.adapters[provider]
	if !ok {
		return nil, fmt.Errorf("target adapter for provider %q not configured", provider)
	}
	return adapter, nil
}

type assetVerifier interface {
	verify(ctx context.Context, changes []install.PlannedChange) (VerificationReport, error)
}

type managedAdapter struct {
	provider        install.Provider
	toolDir         string
	projectRoot     string
	globalRoot      string
	fs              platform.FileSystem
	runner          platform.CommandRunner
	verifier        assetVerifier
	backupQualifier string
}

func (a *managedAdapter) Provider() install.Provider {
	return a.provider
}

func (a *managedAdapter) ResolveTarget(_ context.Context, scope install.Scope, asset install.Asset) (ResolvedTarget, error) {
	if err := install.ValidateScope(scope); err != nil {
		return ResolvedTarget{}, err
	}
	if err := install.ValidateProvider(a.provider); err != nil {
		return ResolvedTarget{}, err
	}
	if !asset.Supports(a.provider) {
		return ResolvedTarget{}, fmt.Errorf("asset %q does not support provider %q", asset.ID(), a.provider)
	}

	baseRoot, err := a.baseRoot(scope)
	if err != nil {
		return ResolvedTarget{}, err
	}

	targetRoot := filepath.Join(baseRoot, a.toolDir)
	var (
		targetPath   string
		verification install.VerificationStatus
	)

	switch asset.Kind() {
	case install.AssetKindCommand:
		targetPath = filepath.Join(targetRoot, "commands", filepath.Base(asset.SourcePath()))
		verification = install.VerificationStatusPartial
	case install.AssetKindSkill:
		targetPath = filepath.Join(targetRoot, "skills", filepath.Base(asset.SourcePath()))
		verification = install.VerificationStatusPartial
	default:
		return ResolvedTarget{}, fmt.Errorf("provider %q does not support asset kind %q", a.provider, asset.Kind())
	}

	return ResolvedTarget{
		TargetPath:   filepath.Clean(targetPath),
		ManagedPath:  filepath.Clean(targetPath),
		Verification: verification,
	}, nil
}

func (a *managedAdapter) Apply(ctx context.Context, changes []install.PlannedChange) ([]ApplyResult, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if a.fs == nil {
		return nil, errors.New("filesystem must not be nil")
	}

	providerChanges, err := a.validateChanges(changes)
	if err != nil {
		return nil, err
	}

	results := make([]ApplyResult, 0, len(providerChanges))
	for _, change := range providerChanges {
		if change.Action() == install.ActionSkip {
			continue
		}

		result, err := a.applyChange(change, len(results))
		if err != nil {
			rollbackErr := a.rollback(results)
			if rollbackErr != nil {
				return nil, fmt.Errorf("apply change for provider %q failed: %w; rollback failed: %v", a.provider, err, rollbackErr)
			}
			return nil, fmt.Errorf("apply change for provider %q asset %q target %q: %w", a.provider, change.AssetID(), change.TargetPath(), err)
		}

		results = append(results, result)
	}

	if err := a.cleanupBackups(results); err != nil {
		return nil, err
	}

	return results, nil
}

func (a *managedAdapter) Verify(ctx context.Context, changes []install.PlannedChange) (VerificationReport, error) {
	if err := ctx.Err(); err != nil {
		return VerificationReport{}, err
	}
	if a.verifier == nil {
		return VerificationReport{
			Provider: a.provider,
			Status:   install.VerificationStatusPartial,
			Details:  []string{"functional verification is not implemented for this provider"},
		}, nil
	}

	return a.verifier.verify(ctx, a.validateProviderOnly(changes))
}

func (a *managedAdapter) baseRoot(scope install.Scope) (string, error) {
	switch scope {
	case install.ScopeProject:
		if a.projectRoot == "" {
			return "", errors.New("project root must not be empty")
		}
		return filepath.Clean(a.projectRoot), nil
	case install.ScopeGlobal:
		if a.globalRoot == "" {
			return "", errors.New("global root must not be empty")
		}
		return filepath.Clean(a.globalRoot), nil
	default:
		return "", fmt.Errorf("unsupported scope %q", scope)
	}
}

func (a *managedAdapter) validateChanges(changes []install.PlannedChange) ([]install.PlannedChange, error) {
	providerChanges := a.validateProviderOnly(changes)
	for _, change := range providerChanges {
		if change.TargetPath() == "" {
			return nil, errors.New("planned change target path must not be empty")
		}
		if change.Action() != install.ActionRemove && change.SourcePath() == "" {
			return nil, errors.New("planned change source path must not be empty")
		}
	}
	return providerChanges, nil
}

func (a *managedAdapter) validateProviderOnly(changes []install.PlannedChange) []install.PlannedChange {
	providerChanges := make([]install.PlannedChange, 0, len(changes))
	for _, change := range changes {
		if change.Provider() != a.provider {
			continue
		}
		providerChanges = append(providerChanges, change)
	}
	return providerChanges
}

func (a *managedAdapter) applyChange(change install.PlannedChange, index int) (ApplyResult, error) {
	backupPath := buildBackupPath(change.TargetPath(), a.backupQualifier, index)

	switch change.Action() {
	case install.ActionInstall, install.ActionUpdate:
		if err := a.replaceTarget(change.SourcePath(), change.TargetPath(), backupPath); err != nil {
			return ApplyResult{}, err
		}
		return ApplyResult{Change: change, BackupPath: backupPath}, nil
	case install.ActionRemove:
		if err := a.removeTarget(change.TargetPath(), backupPath); err != nil {
			return ApplyResult{}, err
		}
		return ApplyResult{Change: change, BackupPath: backupPath}, nil
	default:
		return ApplyResult{}, fmt.Errorf("unsupported action %q", change.Action())
	}
}

func (a *managedAdapter) replaceTarget(sourcePath string, targetPath string, backupPath string) error {
	if err := a.fs.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
		return fmt.Errorf("create target parent directory %q: %w", filepath.Dir(targetPath), err)
	}

	tempPath := buildTempPath(targetPath, a.backupQualifier)
	if err := a.copyPath(sourcePath, tempPath); err != nil {
		_ = a.fs.RemoveAll(tempPath)
		return err
	}

	exists, err := pathExists(a.fs, targetPath)
	if err != nil {
		_ = a.fs.RemoveAll(tempPath)
		return fmt.Errorf("stat existing target %q: %w", targetPath, err)
	}

	if exists {
		if err := a.fs.Rename(targetPath, backupPath); err != nil {
			_ = a.fs.RemoveAll(tempPath)
			return fmt.Errorf("backup existing target %q to %q: %w", targetPath, backupPath, err)
		}
	}

	if err := a.fs.Rename(tempPath, targetPath); err != nil {
		_ = a.fs.RemoveAll(tempPath)
		if exists {
			_ = a.fs.Rename(backupPath, targetPath)
		}
		return fmt.Errorf("activate target %q: %w", targetPath, err)
	}

	return nil
}

func (a *managedAdapter) removeTarget(targetPath string, backupPath string) error {
	exists, err := pathExists(a.fs, targetPath)
	if err != nil {
		return fmt.Errorf("stat target %q: %w", targetPath, err)
	}
	if !exists {
		return nil
	}

	if err := a.fs.Rename(targetPath, backupPath); err != nil {
		return fmt.Errorf("backup target for removal %q: %w", targetPath, err)
	}
	return nil
}

func (a *managedAdapter) rollback(results []ApplyResult) error {
	var rollbackErrs []error
	for index := len(results) - 1; index >= 0; index-- {
		result := results[index]
		change := result.Change

		if err := a.fs.RemoveAll(change.TargetPath()); err != nil && !errors.Is(err, os.ErrNotExist) {
			rollbackErrs = append(rollbackErrs, fmt.Errorf("remove target %q during rollback: %w", change.TargetPath(), err))
			continue
		}

		backupExists, err := pathExists(a.fs, result.BackupPath)
		if err != nil {
			rollbackErrs = append(rollbackErrs, fmt.Errorf("stat backup %q during rollback: %w", result.BackupPath, err))
			continue
		}
		if !backupExists {
			continue
		}

		if err := a.fs.Rename(result.BackupPath, change.TargetPath()); err != nil {
			rollbackErrs = append(rollbackErrs, fmt.Errorf("restore backup %q: %w", result.BackupPath, err))
		}
	}

	return errors.Join(rollbackErrs...)
}

func (a *managedAdapter) cleanupBackups(results []ApplyResult) error {
	for _, result := range results {
		exists, err := pathExists(a.fs, result.BackupPath)
		if err != nil {
			return fmt.Errorf("stat backup %q: %w", result.BackupPath, err)
		}
		if !exists {
			continue
		}

		if err := a.fs.RemoveAll(result.BackupPath); err != nil {
			return fmt.Errorf("remove backup %q: %w", result.BackupPath, err)
		}
	}
	return nil
}

func (a *managedAdapter) copyPath(sourcePath string, destinationPath string) error {
	info, err := a.fs.Stat(sourcePath)
	if err != nil {
		return fmt.Errorf("stat source %q: %w", sourcePath, err)
	}

	if info.IsDir() {
		if err := a.fs.MkdirAll(destinationPath, info.Mode().Perm()); err != nil {
			return fmt.Errorf("create destination directory %q: %w", destinationPath, err)
		}

		entries, err := a.fs.ReadDir(sourcePath)
		if err != nil {
			return fmt.Errorf("read source directory %q: %w", sourcePath, err)
		}

		for _, entry := range entries {
			childSource := filepath.Join(sourcePath, entry.Name())
			childDestination := filepath.Join(destinationPath, entry.Name())
			if err := a.copyPath(childSource, childDestination); err != nil {
				return err
			}
		}
		return nil
	}

	if info.Mode()&fs.ModeSymlink != 0 {
		return fmt.Errorf("symlink source %q is not supported", sourcePath)
	}

	data, err := a.fs.ReadFile(sourcePath)
	if err != nil {
		return fmt.Errorf("read source file %q: %w", sourcePath, err)
	}

	if err := a.fs.MkdirAll(filepath.Dir(destinationPath), 0o755); err != nil {
		return fmt.Errorf("create destination parent %q: %w", filepath.Dir(destinationPath), err)
	}
	if err := a.fs.WriteFile(destinationPath, data, info.Mode().Perm()); err != nil {
		return fmt.Errorf("write destination file %q: %w", destinationPath, err)
	}

	return nil
}

type structuralVerifier struct {
	provider          install.Provider
	fs                platform.FileSystem
	functionalMessage string
	functionalCheck   func(ctx context.Context, changes []install.PlannedChange, details []string) (install.VerificationStatus, []string, error)
}

func (v structuralVerifier) verify(ctx context.Context, changes []install.PlannedChange) (VerificationReport, error) {
	if err := ctx.Err(); err != nil {
		return VerificationReport{}, err
	}

	details := make([]string, 0, len(changes)+1)
	status := install.VerificationStatusComplete
	for _, change := range changes {
		if change.Action() == install.ActionSkip && change.TargetPath() == "" {
			continue
		}

		expectedPresent := change.Action() != install.ActionRemove
		exists, err := pathExists(v.fs, change.TargetPath())
		if err != nil {
			return VerificationReport{}, fmt.Errorf("stat target %q during verification: %w", change.TargetPath(), err)
		}

		if expectedPresent && !exists {
			status = install.VerificationStatusFailed
			details = append(details, fmt.Sprintf("missing target: %s", change.TargetPath()))
			continue
		}
		if !expectedPresent && exists {
			status = install.VerificationStatusFailed
			details = append(details, fmt.Sprintf("target still present after removal: %s", change.TargetPath()))
			continue
		}

		if expectedPresent {
			details = append(details, fmt.Sprintf("target present: %s", change.TargetPath()))
		} else {
			details = append(details, fmt.Sprintf("target removed: %s", change.TargetPath()))
		}
	}

	if status == install.VerificationStatusFailed {
		return VerificationReport{Provider: v.provider, Status: status, Details: details}, nil
	}
	if v.functionalCheck != nil {
		functionalStatus, functionalDetails, err := v.functionalCheck(ctx, changes, details)
		if err != nil {
			return VerificationReport{}, err
		}
		return VerificationReport{Provider: v.provider, Status: functionalStatus, Details: functionalDetails}, nil
	}

	status = install.VerificationStatusPartial
	if v.functionalMessage != "" {
		details = append(details, v.functionalMessage)
	}
	return VerificationReport{Provider: v.provider, Status: status, Details: details}, nil
}

func buildBackupPath(targetPath string, qualifier string, index int) string {
	return fmt.Sprintf("%s.orq-%s-backup-%03d", filepath.Clean(targetPath), qualifier, index)
}

func buildTempPath(targetPath string, qualifier string) string {
	return fmt.Sprintf("%s.orq-%s-tmp", filepath.Clean(targetPath), qualifier)
}

func pathExists(fileSystem platform.FileSystem, path string) (bool, error) {
	_, err := fileSystem.Stat(path)
	if err == nil {
		return true, nil
	}
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	return false, err
}

func aggregateVerificationStatus(statuses ...install.VerificationStatus) install.VerificationStatus {
	if slices.Contains(statuses, install.VerificationStatusFailed) {
		return install.VerificationStatusFailed
	}
	if slices.Contains(statuses, install.VerificationStatusPartial) {
		return install.VerificationStatusPartial
	}
	if slices.Contains(statuses, install.VerificationStatusUnknown) {
		return install.VerificationStatusUnknown
	}
	return install.VerificationStatusComplete
}

func containsTokenCaseInsensitive(content string, token string) bool {
	return strings.Contains(strings.ToLower(content), strings.ToLower(token))
}
