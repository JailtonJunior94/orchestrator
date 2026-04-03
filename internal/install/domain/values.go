package domain

import (
	"fmt"
	"slices"
)

// Provider identifies a supported install target.
type Provider string

const (
	ProviderClaude  Provider = "claude"
	ProviderGemini  Provider = "gemini"
	ProviderCodex   Provider = "codex"
	ProviderCopilot Provider = "copilot"
)

var validProviders = map[Provider]bool{
	ProviderClaude:  true,
	ProviderGemini:  true,
	ProviderCodex:   true,
	ProviderCopilot: true,
}

// ValidateProvider checks whether a provider is supported.
func ValidateProvider(provider Provider) error {
	if !validProviders[provider] {
		return fmt.Errorf("%w: %q", ErrInvalidProvider, provider)
	}
	return nil
}

// Scope identifies where an install is applied.
type Scope string

const (
	ScopeProject Scope = "project"
	ScopeGlobal  Scope = "global"
)

var validScopes = map[Scope]bool{
	ScopeProject: true,
	ScopeGlobal:  true,
}

// ValidateScope checks whether a scope is supported.
func ValidateScope(scope Scope) error {
	if !validScopes[scope] {
		return fmt.Errorf("%w: %q", ErrInvalidScope, scope)
	}
	return nil
}

// AssetKind identifies the normalized asset type.
type AssetKind string

const (
	AssetKindSkill       AssetKind = "skill"
	AssetKindCommand     AssetKind = "command"
	AssetKindInstruction AssetKind = "instruction"
)

var validAssetKinds = map[AssetKind]bool{
	AssetKindSkill:       true,
	AssetKindCommand:     true,
	AssetKindInstruction: true,
}

// ValidateAssetKind checks whether an asset kind is supported.
func ValidateAssetKind(kind AssetKind) error {
	if !validAssetKinds[kind] {
		return fmt.Errorf("%w: %q", ErrInvalidAssetKind, kind)
	}
	return nil
}

// Action identifies an install operation on a planned change.
type Action string

const (
	ActionInstall Action = "install"
	ActionUpdate  Action = "update"
	ActionRemove  Action = "remove"
	ActionSkip    Action = "skip"
)

var validActions = map[Action]bool{
	ActionInstall: true,
	ActionUpdate:  true,
	ActionRemove:  true,
	ActionSkip:    true,
}

// ValidateAction checks whether an action is supported.
func ValidateAction(action Action) error {
	if !validActions[action] {
		return fmt.Errorf("%w: %q", ErrInvalidAction, action)
	}
	return nil
}

// Operation identifies a top-level install workflow request.
type Operation string

const (
	OperationInstall Operation = "install"
	OperationUpdate  Operation = "update"
	OperationRemove  Operation = "remove"
	OperationList    Operation = "list"
	OperationVerify  Operation = "verify"
)

var validOperations = map[Operation]bool{
	OperationInstall: true,
	OperationUpdate:  true,
	OperationRemove:  true,
	OperationList:    true,
	OperationVerify:  true,
}

// ValidateOperation checks whether an operation is supported.
func ValidateOperation(operation Operation) error {
	if !validOperations[operation] {
		return fmt.Errorf("%w: %q", ErrInvalidOperation, operation)
	}
	return nil
}

// ConflictPolicy identifies how a conflict should be handled.
type ConflictPolicy string

const (
	ConflictPolicyAbort     ConflictPolicy = "abort"
	ConflictPolicySkip      ConflictPolicy = "skip"
	ConflictPolicyOverwrite ConflictPolicy = "overwrite"
)

var validConflictPolicies = map[ConflictPolicy]bool{
	ConflictPolicyAbort:     true,
	ConflictPolicySkip:      true,
	ConflictPolicyOverwrite: true,
}

// ValidateConflictPolicy checks whether a conflict policy is supported.
func ValidateConflictPolicy(policy ConflictPolicy) error {
	if !validConflictPolicies[policy] {
		return fmt.Errorf("%w: %q", ErrInvalidConflictPolicy, policy)
	}
	return nil
}

// VerificationStatus identifies the outcome of a verification step.
type VerificationStatus string

const (
	VerificationStatusComplete VerificationStatus = "complete"
	VerificationStatusPartial  VerificationStatus = "partial"
	VerificationStatusFailed   VerificationStatus = "failed"
	VerificationStatusUnknown  VerificationStatus = "unknown"
)

var validVerificationStatuses = map[VerificationStatus]bool{
	VerificationStatusComplete: true,
	VerificationStatusPartial:  true,
	VerificationStatusFailed:   true,
	VerificationStatusUnknown:  true,
}

// ValidateVerificationStatus checks whether a verification status is supported.
func ValidateVerificationStatus(status VerificationStatus) error {
	if !validVerificationStatuses[status] {
		return fmt.Errorf("%w: %q", ErrInvalidVerificationStatus, status)
	}
	return nil
}

// NormalizeProviders validates, deduplicates and orders a provider set.
func NormalizeProviders(providers []Provider) ([]Provider, error) {
	if len(providers) == 0 {
		return nil, ErrEmptyProviderSet
	}

	seen := make(map[Provider]struct{}, len(providers))
	normalized := make([]Provider, 0, len(providers))
	for _, provider := range providers {
		if err := ValidateProvider(provider); err != nil {
			return nil, err
		}
		if _, ok := seen[provider]; ok {
			return nil, fmt.Errorf("%w: %q", ErrDuplicateProvider, provider)
		}
		seen[provider] = struct{}{}
		normalized = append(normalized, provider)
	}

	slices.Sort(normalized)
	return normalized, nil
}
