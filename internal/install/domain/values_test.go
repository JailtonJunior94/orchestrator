package domain

import (
	"errors"
	"path/filepath"
	"testing"
	"time"
)

func TestValidateProvider(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		provider Provider
		wantErr  bool
	}{
		{name: "claude", provider: ProviderClaude},
		{name: "gemini", provider: ProviderGemini},
		{name: "codex", provider: ProviderCodex},
		{name: "copilot", provider: ProviderCopilot},
		{name: "invalid", provider: Provider("unknown"), wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := ValidateProvider(tt.provider)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ValidateProvider(%q) error = %v, wantErr %v", tt.provider, err, tt.wantErr)
			}
		})
	}
}

func TestNormalizeProviders(t *testing.T) {
	t.Parallel()

	normalized, err := NormalizeProviders([]Provider{ProviderCopilot, ProviderClaude})
	if err != nil {
		t.Fatalf("NormalizeProviders() error = %v", err)
	}

	want := []Provider{ProviderClaude, ProviderCopilot}
	for index := range want {
		if normalized[index] != want[index] {
			t.Fatalf("NormalizeProviders() = %v, want %v", normalized, want)
		}
	}

	_, err = NormalizeProviders([]Provider{ProviderClaude, ProviderClaude})
	if !errors.Is(err, ErrDuplicateProvider) {
		t.Fatalf("NormalizeProviders() duplicate error = %v", err)
	}
}

func TestNewAsset(t *testing.T) {
	t.Parallel()

	asset, err := NewAsset(
		"catalog:skill:reviewer",
		"reviewer",
		AssetKindSkill,
		filepath.Join("repo", ".claude", "skills", "reviewer"),
		[]Provider{ProviderCopilot, ProviderClaude},
		AssetMetadata{
			EntryPath:     filepath.Join("repo", ".claude", "skills", "reviewer", "SKILL.md"),
			SourceRoot:    filepath.Join("repo", ".claude", "skills"),
			RelativePath:  filepath.Join(".claude", "skills", "reviewer"),
			ProviderHints: []Provider{ProviderCopilot, ProviderClaude},
		},
	)
	if err != nil {
		t.Fatalf("NewAsset() error = %v", err)
	}

	if asset.Name() != "reviewer" {
		t.Fatalf("asset.Name() = %q", asset.Name())
	}
	if !asset.Supports(ProviderClaude) || !asset.Supports(ProviderCopilot) {
		t.Fatalf("asset providers = %v", asset.Providers())
	}
	if asset.Metadata().EntryPath == "" {
		t.Fatal("asset metadata entry path must not be empty")
	}

	_, err = NewAsset("", "reviewer", AssetKindSkill, "path", []Provider{ProviderClaude}, AssetMetadata{})
	if !errors.Is(err, ErrEmptyAssetID) {
		t.Fatalf("NewAsset() empty id error = %v", err)
	}

	asset, err = NewAsset("asset:id", "name", AssetKindCommand, "path", []Provider{ProviderClaude}, AssetMetadata{})
	if err != nil {
		t.Fatalf("NewAsset() empty metadata error = %v", err)
	}
	if asset.Metadata().EntryPath != "" || asset.Metadata().SourceRoot != "" || asset.Metadata().RelativePath != "" {
		t.Fatalf("NewAsset() normalized empty metadata = %+v", asset.Metadata())
	}
}

func TestNewPlannedChange(t *testing.T) {
	t.Parallel()

	change, err := NewPlannedChange(
		ProviderCodex,
		ScopeProject,
		"codex:skill:reviewer",
		ActionInstall,
		filepath.Join("repo", ".codex", "skills", "reviewer"),
		filepath.Join("target", ".codex", "skills", "reviewer"),
		filepath.Join("target", ".codex", "skills", "reviewer"),
		&Conflict{
			Provider:   ProviderCodex,
			AssetID:    "codex:skill:reviewer",
			TargetPath: filepath.Join("target", ".codex", "skills", "reviewer"),
			Reason:     "already exists",
		},
		VerificationStatusUnknown,
	)
	if err != nil {
		t.Fatalf("NewPlannedChange() error = %v", err)
	}
	if change.Provider() != ProviderCodex {
		t.Fatalf("change.Provider() = %q", change.Provider())
	}
	if change.Conflict() == nil {
		t.Fatal("change.Conflict() must not be nil")
	}

	_, err = NewPlannedChange(
		ProviderCodex,
		ScopeProject,
		"codex:skill:reviewer",
		ActionInstall,
		"source",
		"",
		"managed",
		nil,
		VerificationStatusUnknown,
	)
	if !errors.Is(err, ErrEmptyTargetPath) {
		t.Fatalf("NewPlannedChange() empty target error = %v", err)
	}
}

func TestValidateOperationAndConflictPolicy(t *testing.T) {
	t.Parallel()

	for _, operation := range []Operation{
		OperationInstall,
		OperationUpdate,
		OperationRemove,
		OperationList,
		OperationVerify,
	} {
		if err := ValidateOperation(operation); err != nil {
			t.Fatalf("ValidateOperation(%q) error = %v", operation, err)
		}
	}

	if err := ValidateOperation(Operation("unknown")); !errors.Is(err, ErrInvalidOperation) {
		t.Fatalf("ValidateOperation(invalid) error = %v", err)
	}

	for _, policy := range []ConflictPolicy{
		ConflictPolicyAbort,
		ConflictPolicySkip,
		ConflictPolicyOverwrite,
	} {
		if err := ValidateConflictPolicy(policy); err != nil {
			t.Fatalf("ValidateConflictPolicy(%q) error = %v", policy, err)
		}
	}

	if err := ValidateConflictPolicy(ConflictPolicy("unknown")); !errors.Is(err, ErrInvalidConflictPolicy) {
		t.Fatalf("ValidateConflictPolicy(invalid) error = %v", err)
	}
}

func TestNewConflictDecision(t *testing.T) {
	t.Parallel()

	decision, err := NewConflictDecision(
		ProviderClaude,
		"claude:command:review",
		filepath.Join("repo", ".claude", "commands", "review.md"),
		ConflictPolicyOverwrite,
	)
	if err != nil {
		t.Fatalf("NewConflictDecision() error = %v", err)
	}
	if decision.Policy() != ConflictPolicyOverwrite {
		t.Fatalf("decision.Policy() = %q", decision.Policy())
	}

	_, err = NewConflictDecision(ProviderClaude, "", "target", ConflictPolicyAbort)
	if !errors.Is(err, ErrEmptyAssetID) {
		t.Fatalf("NewConflictDecision() empty asset id error = %v", err)
	}
}

func TestNewInventory(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC()
	entry, err := NewInventoryEntry(
		ProviderClaude,
		"claude:command:review",
		AssetKindCommand,
		filepath.Join("repo", ".claude", "commands", "review.md"),
		"sha256:abc",
		VerificationStatusComplete,
		&now,
	)
	if err != nil {
		t.Fatalf("NewInventoryEntry() error = %v", err)
	}

	inventory, err := NewInventory(1, ScopeProject, filepath.Join(".orq", "install", "inventory.json"), []InventoryEntry{entry}, now)
	if err != nil {
		t.Fatalf("NewInventory() error = %v", err)
	}
	if inventory.Scope() != ScopeProject {
		t.Fatalf("inventory.Scope() = %q", inventory.Scope())
	}
	if len(inventory.Entries()) != 1 {
		t.Fatalf("inventory.Entries() len = %d", len(inventory.Entries()))
	}
}
