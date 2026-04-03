package application

import (
	"context"
	"errors"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	install "github.com/jailtonjunior/orchestrator/internal/install/domain"
	"github.com/jailtonjunior/orchestrator/internal/platform"
)

func TestPlannerPlanOperations(t *testing.T) {
	fileSystem, err := platform.NewFakeFileSystem()
	if err != nil {
		t.Fatalf("NewFakeFileSystem() error = %v", err)
	}
	t.Cleanup(func() {
		if err := fileSystem.Close(); err != nil {
			t.Fatalf("Close() error = %v", err)
		}
	})

	clock := platform.NewFakeClock(time.Date(2026, 4, 3, 12, 0, 0, 0, time.UTC))
	asset := mustAsset(t, "claude:command:review", "review", install.AssetKindCommand, fileSystem.Root(), []install.Provider{install.ProviderClaude})
	managedTarget := filepath.Join(fileSystem.Root(), ".claude", "commands", "review.md")
	inventory := mustInventory(t, install.ScopeProject, filepath.Join(fileSystem.Root(), ".orq", "install", "inventory.json"), clock.Now(), mustInventoryEntry(
		t,
		install.ProviderClaude,
		asset.ID(),
		asset.Kind(),
		managedTarget,
		install.VerificationStatusComplete,
	))

	planner := NewPlanner(fileSystem, staticTargetResolver{
		targets: map[string]Target{
			targetKey(install.ProviderClaude, asset.ID()): {
				TargetPath:   managedTarget,
				ManagedPath:  managedTarget,
				Verification: install.VerificationStatusPartial,
			},
		},
	}, nil)

	tests := []struct {
		name          string
		input         PlanInput
		wantActions   []install.Action
		wantConflicts int
	}{
		{
			name: "install without inventory produces install",
			input: PlanInput{
				Operation:      install.OperationInstall,
				Scope:          install.ScopeProject,
				Providers:      []install.Provider{install.ProviderClaude},
				Assets:         []install.Asset{asset},
				ConflictPolicy: install.ConflictPolicyOverwrite,
			},
			wantActions:   []install.Action{install.ActionInstall},
			wantConflicts: 0,
		},
		{
			name: "update managed asset produces update",
			input: PlanInput{
				Operation:      install.OperationUpdate,
				Scope:          install.ScopeProject,
				Providers:      []install.Provider{install.ProviderClaude},
				Assets:         []install.Asset{asset},
				Inventory:      inventory,
				ConflictPolicy: install.ConflictPolicyOverwrite,
			},
			wantActions:   []install.Action{install.ActionUpdate},
			wantConflicts: 0,
		},
		{
			name: "remove managed asset produces remove",
			input: PlanInput{
				Operation:      install.OperationRemove,
				Scope:          install.ScopeProject,
				Providers:      []install.Provider{install.ProviderClaude},
				Assets:         []install.Asset{asset},
				Inventory:      inventory,
				ConflictPolicy: install.ConflictPolicyOverwrite,
			},
			wantActions:   []install.Action{install.ActionRemove},
			wantConflicts: 0,
		},
		{
			name: "list produces preview-only plan",
			input: PlanInput{
				Operation:      install.OperationList,
				Scope:          install.ScopeProject,
				Providers:      []install.Provider{install.ProviderClaude},
				Assets:         []install.Asset{asset},
				Inventory:      inventory,
				ConflictPolicy: install.ConflictPolicyOverwrite,
			},
			wantActions:   []install.Action{install.ActionSkip},
			wantConflicts: 0,
		},
		{
			name: "verify produces preview-only plan",
			input: PlanInput{
				Operation:      install.OperationVerify,
				Scope:          install.ScopeProject,
				Providers:      []install.Provider{install.ProviderClaude},
				Assets:         []install.Asset{asset},
				Inventory:      inventory,
				ConflictPolicy: install.ConflictPolicyOverwrite,
			},
			wantActions:   []install.Action{install.ActionSkip},
			wantConflicts: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			plan, err := planner.Plan(context.Background(), tt.input)
			if err != nil {
				t.Fatalf("Plan() error = %v", err)
			}

			gotActions := make([]install.Action, 0, len(plan.Changes))
			conflictCount := 0
			for _, change := range plan.Changes {
				gotActions = append(gotActions, change.Action())
				if change.Conflict() != nil {
					conflictCount++
				}
			}

			if !reflect.DeepEqual(gotActions, tt.wantActions) {
				t.Fatalf("Plan() actions = %v, want %v", gotActions, tt.wantActions)
			}
			if conflictCount != tt.wantConflicts {
				t.Fatalf("Plan() conflicts = %d, want %d", conflictCount, tt.wantConflicts)
			}
		})
	}
}

func TestPlannerConflictPolicies(t *testing.T) {
	fileSystem, err := platform.NewFakeFileSystem()
	if err != nil {
		t.Fatalf("NewFakeFileSystem() error = %v", err)
	}
	t.Cleanup(func() {
		if err := fileSystem.Close(); err != nil {
			t.Fatalf("Close() error = %v", err)
		}
	})

	asset := mustAsset(t, "claude:command:review", "review", install.AssetKindCommand, fileSystem.Root(), []install.Provider{install.ProviderClaude})
	targetPath := filepath.Join(fileSystem.Root(), ".claude", "commands", "review.md")
	if err := fileSystem.WriteFile(targetPath, []byte("local"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	planner := NewPlanner(fileSystem, staticTargetResolver{
		targets: map[string]Target{
			targetKey(install.ProviderClaude, asset.ID()): {
				TargetPath:   targetPath,
				ManagedPath:  targetPath,
				Verification: install.VerificationStatusUnknown,
			},
		},
	}, nil)

	baseInput := PlanInput{
		Operation: install.OperationInstall,
		Scope:     install.ScopeProject,
		Providers: []install.Provider{install.ProviderClaude},
		Assets:    []install.Asset{asset},
	}

	t.Run("abort is default in non interactive mode", func(t *testing.T) {
		_, err := planner.Plan(context.Background(), baseInput)
		if !errors.Is(err, install.ErrConflictAborted) {
			t.Fatalf("Plan() error = %v, want conflict aborted", err)
		}
	})

	t.Run("skip converts conflicting change to skip", func(t *testing.T) {
		plan, err := planner.Plan(context.Background(), PlanInput{
			Operation:      baseInput.Operation,
			Scope:          baseInput.Scope,
			Providers:      baseInput.Providers,
			Assets:         baseInput.Assets,
			ConflictPolicy: install.ConflictPolicySkip,
		})
		if err != nil {
			t.Fatalf("Plan() error = %v", err)
		}
		if got := plan.Changes[0].Action(); got != install.ActionSkip {
			t.Fatalf("Plan() action = %q, want %q", got, install.ActionSkip)
		}
		if plan.Changes[0].Conflict() == nil {
			t.Fatal("Plan() conflict must be preserved for audit")
		}
	})

	t.Run("overwrite keeps install action", func(t *testing.T) {
		plan, err := planner.Plan(context.Background(), PlanInput{
			Operation:      baseInput.Operation,
			Scope:          baseInput.Scope,
			Providers:      baseInput.Providers,
			Assets:         baseInput.Assets,
			ConflictPolicy: install.ConflictPolicyOverwrite,
		})
		if err != nil {
			t.Fatalf("Plan() error = %v", err)
		}
		if got := plan.Changes[0].Action(); got != install.ActionInstall {
			t.Fatalf("Plan() action = %q, want %q", got, install.ActionInstall)
		}
	})
}

func TestConflictPolicyResolverInteractiveDelegation(t *testing.T) {
	t.Parallel()

	conflicts := []install.Conflict{{
		Provider:   install.ProviderClaude,
		AssetID:    "claude:command:review",
		TargetPath: filepath.Join("repo", ".claude", "commands", "review.md"),
		Reason:     "target already exists",
	}}

	resolver := NewConflictPolicyResolver(install.ConflictPolicyAbort, true, staticPrompter{
		decisions: []install.ConflictDecision{
			mustDecision(t, install.ProviderClaude, "claude:command:review", filepath.Join("repo", ".claude", "commands", "review.md"), install.ConflictPolicyOverwrite),
		},
	})

	decisions, err := resolver.Resolve(context.Background(), conflicts)
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if len(decisions) != 1 || decisions[0].Policy() != install.ConflictPolicyOverwrite {
		t.Fatalf("Resolve() = %+v", decisions)
	}
}

func TestPlannerRemoveDetectsExternalContent(t *testing.T) {
	t.Parallel()

	fileSystem, err := platform.NewFakeFileSystem()
	if err != nil {
		t.Fatalf("NewFakeFileSystem() error = %v", err)
	}
	t.Cleanup(func() {
		if err := fileSystem.Close(); err != nil {
			t.Fatalf("Close() error = %v", err)
		}
	})

	asset := mustAsset(t, "claude:command:review", "review", install.AssetKindCommand, fileSystem.Root(), []install.Provider{install.ProviderClaude})
	targetPath := filepath.Join(fileSystem.Root(), ".claude", "commands", "review.md")
	if err := fileSystem.WriteFile(targetPath, []byte("external"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	planner := NewPlanner(fileSystem, staticTargetResolver{
		targets: map[string]Target{
			targetKey(install.ProviderClaude, asset.ID()): {
				TargetPath:   targetPath,
				ManagedPath:  targetPath,
				Verification: install.VerificationStatusUnknown,
			},
		},
	}, nil)

	plan, err := planner.Plan(context.Background(), PlanInput{
		Operation:      install.OperationRemove,
		Scope:          install.ScopeProject,
		Providers:      []install.Provider{install.ProviderClaude},
		Assets:         []install.Asset{asset},
		ConflictPolicy: install.ConflictPolicySkip,
	})
	if err != nil {
		t.Fatalf("Plan() error = %v", err)
	}

	if got := plan.Changes[0].Action(); got != install.ActionSkip {
		t.Fatalf("Plan() action = %q, want %q", got, install.ActionSkip)
	}
	if plan.Changes[0].Conflict() == nil {
		t.Fatal("Plan() expected conflict for unmanaged target")
	}
}

type staticTargetResolver struct {
	targets map[string]Target
}

func (r staticTargetResolver) ResolveTarget(_ context.Context, provider install.Provider, _ install.Scope, asset install.Asset) (Target, error) {
	target, ok := r.targets[targetKey(provider, asset.ID())]
	if !ok {
		return Target{}, errors.New("target not found")
	}
	return target, nil
}

type staticPrompter struct {
	decisions []install.ConflictDecision
}

func (p staticPrompter) ResolveConflicts(_ context.Context, _ []install.Conflict) ([]install.ConflictDecision, error) {
	return p.decisions, nil
}

func mustAsset(t *testing.T, id string, name string, kind install.AssetKind, root string, providers []install.Provider) install.Asset {
	t.Helper()

	sourcePath := filepath.Join(root, "catalog", name)
	if kind == install.AssetKindCommand {
		sourcePath += ".md"
	}

	asset, err := install.NewAsset(id, name, kind, sourcePath, providers, install.AssetMetadata{
		EntryPath:    sourcePath,
		SourceRoot:   filepath.Dir(sourcePath),
		RelativePath: filepath.Base(sourcePath),
	})
	if err != nil {
		t.Fatalf("NewAsset() error = %v", err)
	}
	return asset
}

func mustInventory(t *testing.T, scope install.Scope, location string, now time.Time, entries ...install.InventoryEntry) *install.Inventory {
	t.Helper()

	inventory, err := install.NewInventory(1, scope, location, entries, now)
	if err != nil {
		t.Fatalf("NewInventory() error = %v", err)
	}
	return inventory
}

func mustInventoryEntry(
	t *testing.T,
	provider install.Provider,
	assetID string,
	kind install.AssetKind,
	managedPath string,
	verification install.VerificationStatus,
) install.InventoryEntry {
	t.Helper()

	entry, err := install.NewInventoryEntry(provider, assetID, kind, managedPath, "sha256:abc", verification, nil)
	if err != nil {
		t.Fatalf("NewInventoryEntry() error = %v", err)
	}
	return entry
}

func mustDecision(
	t *testing.T,
	provider install.Provider,
	assetID string,
	targetPath string,
	policy install.ConflictPolicy,
) install.ConflictDecision {
	t.Helper()

	decision, err := install.NewConflictDecision(provider, assetID, targetPath, policy)
	if err != nil {
		t.Fatalf("NewConflictDecision() error = %v", err)
	}
	return decision
}

func targetKey(provider install.Provider, assetID string) string {
	return string(provider) + "::" + assetID
}
