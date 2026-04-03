package application

import (
	"context"

	install "github.com/jailtonjunior/orchestrator/internal/install/domain"
)

// Service exposes the install module application use cases.
type Service interface {
	Preview(ctx context.Context, req PreviewRequest) (*OperationPreview, error)
	Install(ctx context.Context, req InstallRequest) (*OperationResult, error)
	Update(ctx context.Context, req UpdateRequest) (*OperationResult, error)
	Remove(ctx context.Context, req RemoveRequest) (*OperationResult, error)
	List(ctx context.Context, req ListRequest) (*InventoryView, error)
	Verify(ctx context.Context, req VerifyRequest) (*VerificationReport, error)
}

// OperationRequest contains the common request filters for install operations.
type OperationRequest struct {
	Scope          install.Scope
	Providers      []install.Provider
	AssetNames     []string
	AssetKinds     []install.AssetKind
	ConflictPolicy install.ConflictPolicy
	Interactive    bool
}

// PreviewRequest requests a deterministic plan preview for a mutating operation.
type PreviewRequest struct {
	OperationRequest
	Operation install.Operation
}

// InstallRequest requests a new installation.
type InstallRequest struct {
	OperationRequest
}

// UpdateRequest requests a managed installation update.
type UpdateRequest struct {
	OperationRequest
}

// RemoveRequest requests the removal of managed assets.
type RemoveRequest struct {
	OperationRequest
}

// ListRequest requests a filtered inventory view.
type ListRequest struct {
	Scope      install.Scope
	Providers  []install.Provider
	AssetNames []string
	AssetKinds []install.AssetKind
}

// VerifyRequest requests a post-install verification pass.
type VerifyRequest struct {
	Scope      install.Scope
	Providers  []install.Provider
	AssetNames []string
	AssetKinds []install.AssetKind
}

// ProviderResult reports the execution outcome for a single provider.
type ProviderResult struct {
	Provider           install.Provider
	PlannedChangeCount int
	AppliedChangeCount int
	Verification       install.VerificationStatus
	Details            []string
	InventorySaved     bool
}

// OperationResult reports the execution outcome for a mutating operation.
type OperationResult struct {
	Operation     install.Operation
	Scope         install.Scope
	InventoryPath string
	Plan          *Plan
	Providers     []ProviderResult
}

// OperationPreview reports the deterministic plan for a mutating operation.
type OperationPreview struct {
	Operation     install.Operation
	Scope         install.Scope
	InventoryPath string
	Plan          *Plan
}

// InventoryItem is a rendered view of one managed or eligible asset.
type InventoryItem struct {
	Provider     install.Provider
	AssetID      string
	Name         string
	Kind         install.AssetKind
	TargetPath   string
	ManagedPath  string
	Managed      bool
	Verification install.VerificationStatus
}

// InventoryView reports the current install state for a scope.
type InventoryView struct {
	Scope         install.Scope
	InventoryPath string
	Plan          *Plan
	Items         []InventoryItem
}

// VerificationProviderReport reports one provider verification outcome.
type VerificationProviderReport struct {
	Provider       install.Provider
	Status         install.VerificationStatus
	Details        []string
	InventorySaved bool
}

// VerificationReport reports the consolidated verification results.
type VerificationReport struct {
	Scope         install.Scope
	InventoryPath string
	Providers     []VerificationProviderReport
}
