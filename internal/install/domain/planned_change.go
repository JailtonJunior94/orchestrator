package domain

import "path/filepath"

// Conflict describes a target collision detected during planning.
type Conflict struct {
	Provider   Provider
	AssetID    string
	TargetPath string
	Managed    bool
	Reason     string
}

// ConflictDecision captures the selected resolution for a conflict.
type ConflictDecision struct {
	provider   Provider
	assetID    string
	targetPath string
	policy     ConflictPolicy
}

// NewConflictDecision creates a validated conflict decision.
func NewConflictDecision(provider Provider, assetID string, targetPath string, policy ConflictPolicy) (ConflictDecision, error) {
	if err := ValidateProvider(provider); err != nil {
		return ConflictDecision{}, err
	}
	if assetID == "" {
		return ConflictDecision{}, ErrEmptyAssetID
	}
	if targetPath == "" {
		return ConflictDecision{}, ErrEmptyConflictTargetPath
	}
	if err := ValidateConflictPolicy(policy); err != nil {
		return ConflictDecision{}, err
	}

	return ConflictDecision{
		provider:   provider,
		assetID:    assetID,
		targetPath: filepath.Clean(targetPath),
		policy:     policy,
	}, nil
}

// Provider returns the decision provider.
func (d ConflictDecision) Provider() Provider { return d.provider }

// AssetID returns the conflicted asset id.
func (d ConflictDecision) AssetID() string { return d.assetID }

// TargetPath returns the conflicted target path.
func (d ConflictDecision) TargetPath() string { return d.targetPath }

// Policy returns the selected policy.
func (d ConflictDecision) Policy() ConflictPolicy { return d.policy }

// PlannedChange represents a single deterministic change to be applied.
type PlannedChange struct {
	provider     Provider
	scope        Scope
	assetID      string
	action       Action
	sourcePath   string
	targetPath   string
	managedPath  string
	conflict     *Conflict
	verification VerificationStatus
}

// NewPlannedChange creates a validated planned change.
func NewPlannedChange(
	provider Provider,
	scope Scope,
	assetID string,
	action Action,
	sourcePath string,
	targetPath string,
	managedPath string,
	conflict *Conflict,
	verification VerificationStatus,
) (PlannedChange, error) {
	if err := ValidateProvider(provider); err != nil {
		return PlannedChange{}, err
	}
	if err := ValidateScope(scope); err != nil {
		return PlannedChange{}, err
	}
	if assetID == "" {
		return PlannedChange{}, ErrEmptyAssetID
	}
	if err := ValidateAction(action); err != nil {
		return PlannedChange{}, err
	}
	if sourcePath == "" {
		return PlannedChange{}, ErrEmptyAssetSourcePath
	}
	if targetPath == "" {
		return PlannedChange{}, ErrEmptyTargetPath
	}
	if managedPath == "" {
		return PlannedChange{}, ErrEmptyManagedPath
	}
	if err := ValidateVerificationStatus(verification); err != nil {
		return PlannedChange{}, err
	}

	change := PlannedChange{
		provider:     provider,
		scope:        scope,
		assetID:      assetID,
		action:       action,
		sourcePath:   filepath.Clean(sourcePath),
		targetPath:   filepath.Clean(targetPath),
		managedPath:  filepath.Clean(managedPath),
		verification: verification,
	}
	if conflict != nil {
		copyConflict := *conflict
		copyConflict.TargetPath = filepath.Clean(copyConflict.TargetPath)
		change.conflict = &copyConflict
	}

	return change, nil
}

// Provider returns the target provider.
func (c PlannedChange) Provider() Provider { return c.provider }

// Scope returns the target scope.
func (c PlannedChange) Scope() Scope { return c.scope }

// AssetID returns the asset identifier.
func (c PlannedChange) AssetID() string { return c.assetID }

// Action returns the operation to be performed.
func (c PlannedChange) Action() Action { return c.action }

// SourcePath returns the source path.
func (c PlannedChange) SourcePath() string { return c.sourcePath }

// TargetPath returns the final destination path.
func (c PlannedChange) TargetPath() string { return c.targetPath }

// ManagedPath returns the ORQ-managed path for the change.
func (c PlannedChange) ManagedPath() string { return c.managedPath }

// Conflict returns a defensive copy of the conflict metadata.
func (c PlannedChange) Conflict() *Conflict {
	if c.conflict == nil {
		return nil
	}
	copyConflict := *c.conflict
	return &copyConflict
}

// Verification returns the expected verification status.
func (c PlannedChange) Verification() VerificationStatus { return c.verification }
