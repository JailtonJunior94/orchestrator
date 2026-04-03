package application

import (
	"context"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"github.com/jailtonjunior/orchestrator/internal/install/catalog"
	install "github.com/jailtonjunior/orchestrator/internal/install/domain"
	installinventory "github.com/jailtonjunior/orchestrator/internal/install/inventory"
	installproviders "github.com/jailtonjunior/orchestrator/internal/install/providers"
	"github.com/jailtonjunior/orchestrator/internal/platform"
)

func TestInstallServiceInstallCoordinatesProvidersAndPersistsInventory(t *testing.T) {
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
	projectInventoryPath := filepath.Join(projectRoot, ".orq", "install", "inventory.json")
	globalInventoryPath := filepath.Join(fileSystem.Root(), "state", "orq", "install", "inventory.json")
	clock := platform.NewFakeClock(time.Date(2026, 4, 3, 12, 0, 0, 0, time.UTC))

	claudeAsset := mustServiceAsset(
		t,
		"claude:command:review",
		"review",
		install.AssetKindCommand,
		filepath.Join(projectRoot, ".claude", "commands", "review.md"),
		install.ProviderClaude,
	)
	geminiAsset := mustServiceAsset(
		t,
		"gemini:skill:reviewer",
		"reviewer",
		install.AssetKindSkill,
		filepath.Join(projectRoot, ".gemini", "skills", "reviewer"),
		install.ProviderGemini,
	)
	writeServiceAsset(t, fileSystem, claudeAsset)
	writeServiceAsset(t, fileSystem, geminiAsset)

	registry, err := installproviders.NewRegistry(
		newFakeAdapter(install.ProviderClaude, projectRoot, install.VerificationStatusPartial),
		newFakeAdapter(install.ProviderGemini, projectRoot, install.VerificationStatusComplete),
	)
	if err != nil {
		t.Fatalf("NewRegistry() error = %v", err)
	}

	store := &stubInventoryStore{
		clock: clock,
		loadInventory: mustServiceInventory(
			t,
			install.ScopeProject,
			projectInventoryPath,
			clock.Now(),
			nil,
		),
	}
	planner := NewPlanner(fileSystem, registryTargetResolver{registry: registry}, nil)
	service := NewService(
		projectRoot,
		projectInventoryPath,
		globalInventoryPath,
		staticCatalog{assets: []install.Asset{claudeAsset, geminiAsset}},
		planner,
		store,
		registry,
		fileSystem,
		clock,
		nil,
	)

	result, err := service.Install(context.Background(), InstallRequest{
		OperationRequest: OperationRequest{
			Scope:          install.ScopeProject,
			Providers:      []install.Provider{install.ProviderClaude, install.ProviderGemini},
			ConflictPolicy: install.ConflictPolicyOverwrite,
		},
	})
	if err != nil {
		t.Fatalf("Install() error = %v", err)
	}
	if len(result.Providers) != 2 {
		t.Fatalf("Install() providers = %d, want 2", len(result.Providers))
	}
	if got := len(store.savedInventories); got != 2 {
		t.Fatalf("inventory saves = %d, want 2", got)
	}

	lastInventory := store.savedInventories[len(store.savedInventories)-1]
	if got := len(lastInventory.Entries()); got != 2 {
		t.Fatalf("final inventory entries = %d, want 2", got)
	}
}

func TestInstallServiceInstallStopsAfterProviderVerificationFailure(t *testing.T) {
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
	homeRoot := filepath.Join(fileSystem.Root(), "home")
	projectInventoryPath := filepath.Join(projectRoot, ".orq", "install", "inventory.json")
	globalInventoryPath := filepath.Join(fileSystem.Root(), "state", "orq", "install", "inventory.json")
	clock := platform.NewFakeClock(time.Date(2026, 4, 3, 12, 0, 0, 0, time.UTC))

	if err := fileSystem.WriteFile(filepath.Join(projectRoot, ".claude", "commands", "review.md"), []byte("review"), 0o644); err != nil {
		t.Fatalf("WriteFile(claude source) error = %v", err)
	}
	if err := fileSystem.WriteFile(filepath.Join(projectRoot, ".gemini", "skills", "reviewer", "SKILL.md"), []byte("reviewer"), 0o644); err != nil {
		t.Fatalf("WriteFile(gemini source) error = %v", err)
	}

	registry, err := installproviders.NewRegistry(
		installproviders.NewClaudeAdapter(fileSystem, projectRoot, homeRoot),
		installproviders.NewGeminiAdapter(
			fileSystem,
			platform.FakeCommandRunner{
				RunFunc: func(ctx context.Context, name string, args []string, stdin string) (platform.CommandResult, error) {
					return platform.CommandResult{Stdout: "other-skill\n", ExitCode: 0}, nil
				},
			},
			projectRoot,
			homeRoot,
		),
	)
	if err != nil {
		t.Fatalf("NewRegistry() error = %v", err)
	}

	store := installinventory.NewStore(fileSystem, clock)
	planner := NewPlanner(fileSystem, registryTargetResolver{registry: registry}, nil)
	service := NewService(
		projectRoot,
		projectInventoryPath,
		globalInventoryPath,
		catalog.New(),
		planner,
		store,
		registry,
		fileSystem,
		clock,
		nil,
	)

	result, err := service.Install(context.Background(), InstallRequest{
		OperationRequest: OperationRequest{
			Scope:          install.ScopeProject,
			Providers:      []install.Provider{install.ProviderClaude, install.ProviderGemini},
			ConflictPolicy: install.ConflictPolicyOverwrite,
		},
	})
	if err == nil {
		t.Fatal("Install() error = nil, want non-nil")
	}
	if len(result.Providers) != 2 {
		t.Fatalf("Install() providers = %d, want 2", len(result.Providers))
	}
	if result.Providers[0].InventorySaved != true {
		t.Fatal("first provider inventory must be saved")
	}
	if result.Providers[1].InventorySaved {
		t.Fatal("failing provider inventory must not be saved")
	}

	inventorySnapshot, err := store.Load(context.Background(), install.ScopeProject, projectInventoryPath)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if got := len(inventorySnapshot.Entries()); got != 1 {
		t.Fatalf("saved inventory entries = %d, want 1", got)
	}
	if got := inventorySnapshot.Entries()[0].Provider(); got != install.ProviderClaude {
		t.Fatalf("saved provider = %q, want %q", got, install.ProviderClaude)
	}
}

type staticCatalog struct {
	assets []install.Asset
	err    error
}

func (c staticCatalog) Discover(ctx context.Context, root string) ([]install.Asset, error) {
	if c.err != nil {
		return nil, c.err
	}
	return append([]install.Asset(nil), c.assets...), nil
}

type stubInventoryStore struct {
	clock            platform.Clock
	loadInventory    *install.Inventory
	savedInventories []*install.Inventory
}

func (s *stubInventoryStore) Load(ctx context.Context, scope install.Scope, location string) (*install.Inventory, error) {
	if s.loadInventory != nil {
		return s.loadInventory, nil
	}
	return install.NewInventory(1, scope, location, nil, s.clock.Now().UTC())
}

func (s *stubInventoryStore) Save(ctx context.Context, scope install.Scope, location string, inv *install.Inventory) error {
	snapshot, err := install.NewInventory(inv.SchemaVersion(), scope, location, inv.Entries(), inv.UpdatedAt())
	if err != nil {
		return err
	}
	s.savedInventories = append(s.savedInventories, snapshot)
	s.loadInventory = snapshot
	return nil
}

type fakeAdapter struct {
	provider install.Provider
	root     string
	status   install.VerificationStatus
}

func newFakeAdapter(provider install.Provider, root string, status install.VerificationStatus) *fakeAdapter {
	return &fakeAdapter{provider: provider, root: root, status: status}
}

func (a *fakeAdapter) Provider() install.Provider {
	return a.provider
}

func (a *fakeAdapter) ResolveTarget(ctx context.Context, scope install.Scope, asset install.Asset) (installproviders.ResolvedTarget, error) {
	targetPath := filepath.Join(a.root, "."+string(a.provider), string(asset.Kind())+"s", filepath.Base(asset.SourcePath()))
	if asset.Kind() == install.AssetKindSkill {
		targetPath = filepath.Join(a.root, "."+string(a.provider), "skills", filepath.Base(asset.SourcePath()))
	}
	return installproviders.ResolvedTarget{
		TargetPath:   targetPath,
		ManagedPath:  targetPath,
		Verification: a.status,
	}, nil
}

func (a *fakeAdapter) Apply(ctx context.Context, changes []install.PlannedChange) ([]installproviders.ApplyResult, error) {
	results := make([]installproviders.ApplyResult, 0, len(changes))
	for _, change := range changes {
		if change.Action() == install.ActionSkip {
			continue
		}
		results = append(results, installproviders.ApplyResult{Change: change})
	}
	return results, nil
}

func (a *fakeAdapter) Verify(ctx context.Context, changes []install.PlannedChange) (installproviders.VerificationReport, error) {
	return installproviders.VerificationReport{
		Provider: a.provider,
		Status:   a.status,
		Details:  []string{fmt.Sprintf("status=%s", a.status)},
	}, nil
}

func mustServiceAsset(
	t *testing.T,
	id string,
	name string,
	kind install.AssetKind,
	sourcePath string,
	provider install.Provider,
) install.Asset {
	t.Helper()

	asset, err := install.NewAsset(
		id,
		name,
		kind,
		sourcePath,
		[]install.Provider{provider},
		install.AssetMetadata{
			EntryPath:     sourcePath,
			SourceRoot:    filepath.Dir(sourcePath),
			RelativePath:  filepath.Base(sourcePath),
			ProviderHints: []install.Provider{provider},
		},
	)
	if err != nil {
		t.Fatalf("NewAsset() error = %v", err)
	}
	return asset
}

func writeServiceAsset(t *testing.T, fileSystem platform.FileSystem, asset install.Asset) {
	t.Helper()

	payload := []byte("content-" + asset.ID())
	target := asset.SourcePath()
	if asset.Kind() == install.AssetKindSkill {
		target = filepath.Join(asset.SourcePath(), "SKILL.md")
	}
	if err := fileSystem.WriteFile(target, payload, 0o644); err != nil {
		t.Fatalf("WriteFile(%q) error = %v", target, err)
	}
}

func mustServiceInventory(
	t *testing.T,
	scope install.Scope,
	location string,
	updatedAt time.Time,
	entries []install.InventoryEntry,
) *install.Inventory {
	t.Helper()

	inventorySnapshot, err := install.NewInventory(1, scope, location, entries, updatedAt)
	if err != nil {
		t.Fatalf("NewInventory() error = %v", err)
	}
	return inventorySnapshot
}

func TestInstallServiceVerifySkipsInventorySaveWhenProviderFails(t *testing.T) {
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
	projectInventoryPath := filepath.Join(projectRoot, ".orq", "install", "inventory.json")
	globalInventoryPath := filepath.Join(fileSystem.Root(), "state", "orq", "install", "inventory.json")
	clock := platform.NewFakeClock(time.Date(2026, 4, 3, 12, 0, 0, 0, time.UTC))

	asset := mustServiceAsset(
		t,
		"claude:command:review",
		"review",
		install.AssetKindCommand,
		filepath.Join(projectRoot, ".claude", "commands", "review.md"),
		install.ProviderClaude,
	)
	writeServiceAsset(t, fileSystem, asset)

	entry := mustServiceInventoryEntry(
		t,
		install.ProviderClaude,
		asset.ID(),
		asset.Kind(),
		filepath.Join(projectRoot, ".claude", "commands", "review.md"),
		install.VerificationStatusPartial,
	)

	registry, err := installproviders.NewRegistry(newFakeAdapter(install.ProviderClaude, projectRoot, install.VerificationStatusFailed))
	if err != nil {
		t.Fatalf("NewRegistry() error = %v", err)
	}
	store := &stubInventoryStore{
		clock:         clock,
		loadInventory: mustServiceInventory(t, install.ScopeProject, projectInventoryPath, clock.Now(), []install.InventoryEntry{entry}),
	}
	planner := NewPlanner(fileSystem, registryTargetResolver{registry: registry}, nil)
	service := NewService(
		projectRoot,
		projectInventoryPath,
		globalInventoryPath,
		staticCatalog{assets: []install.Asset{asset}},
		planner,
		store,
		registry,
		fileSystem,
		clock,
		nil,
	)

	_, err = service.Verify(context.Background(), VerifyRequest{
		Scope:     install.ScopeProject,
		Providers: []install.Provider{install.ProviderClaude},
	})
	if err == nil {
		t.Fatal("Verify() error = nil, want non-nil")
	}
	if got := len(store.savedInventories); got != 0 {
		t.Fatalf("inventory saves = %d, want 0", got)
	}
}

func TestInstallServiceRemoveDeletesManagedEntriesFromInventory(t *testing.T) {
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
	projectInventoryPath := filepath.Join(projectRoot, ".orq", "install", "inventory.json")
	globalInventoryPath := filepath.Join(fileSystem.Root(), "state", "orq", "install", "inventory.json")
	clock := platform.NewFakeClock(time.Date(2026, 4, 3, 12, 0, 0, 0, time.UTC))

	asset := mustServiceAsset(
		t,
		"claude:command:review",
		"review",
		install.AssetKindCommand,
		filepath.Join(projectRoot, ".claude", "commands", "review.md"),
		install.ProviderClaude,
	)
	writeServiceAsset(t, fileSystem, asset)

	entry := mustServiceInventoryEntry(
		t,
		install.ProviderClaude,
		asset.ID(),
		asset.Kind(),
		filepath.Join(projectRoot, ".claude", "commands", "review.md"),
		install.VerificationStatusComplete,
	)

	registry, err := installproviders.NewRegistry(newFakeAdapter(install.ProviderClaude, projectRoot, install.VerificationStatusPartial))
	if err != nil {
		t.Fatalf("NewRegistry() error = %v", err)
	}
	store := &stubInventoryStore{
		clock:         clock,
		loadInventory: mustServiceInventory(t, install.ScopeProject, projectInventoryPath, clock.Now(), []install.InventoryEntry{entry}),
	}
	planner := NewPlanner(fileSystem, registryTargetResolver{registry: registry}, nil)
	service := NewService(
		projectRoot,
		projectInventoryPath,
		globalInventoryPath,
		staticCatalog{assets: []install.Asset{asset}},
		planner,
		store,
		registry,
		fileSystem,
		clock,
		nil,
	)

	result, err := service.Remove(context.Background(), RemoveRequest{
		OperationRequest: OperationRequest{
			Scope:          install.ScopeProject,
			Providers:      []install.Provider{install.ProviderClaude},
			ConflictPolicy: install.ConflictPolicyOverwrite,
		},
	})
	if err != nil {
		t.Fatalf("Remove() error = %v", err)
	}
	if len(result.Providers) != 1 {
		t.Fatalf("Remove() providers = %d, want 1", len(result.Providers))
	}
	if !result.Providers[0].InventorySaved {
		t.Fatal("Remove() inventory must be saved")
	}
	if got := len(store.savedInventories); got != 1 {
		t.Fatalf("inventory saves = %d, want 1", got)
	}
	if got := len(store.savedInventories[0].Entries()); got != 0 {
		t.Fatalf("saved inventory entries = %d, want 0", got)
	}
}

func mustServiceInventoryEntry(
	t *testing.T,
	provider install.Provider,
	assetID string,
	kind install.AssetKind,
	managedPath string,
	verification install.VerificationStatus,
) install.InventoryEntry {
	t.Helper()

	entry, err := install.NewInventoryEntry(provider, assetID, kind, managedPath, "fingerprint", verification, nil)
	if err != nil {
		t.Fatalf("NewInventoryEntry() error = %v", err)
	}
	return entry
}

var _ installproviders.TargetAdapter = (*fakeAdapter)(nil)
var _ installinventory.Store = (*stubInventoryStore)(nil)
var _ catalog.Catalog = staticCatalog{}

func TestStubInventoryStoreLoadPropagatesScope(t *testing.T) {
	t.Parallel()

	store := &stubInventoryStore{
		clock: platform.NewFakeClock(time.Date(2026, 4, 3, 12, 0, 0, 0, time.UTC)),
	}
	inventorySnapshot, err := store.Load(context.Background(), install.ScopeGlobal, filepath.Join("state", "inventory.json"))
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if inventorySnapshot.Scope() != install.ScopeGlobal {
		t.Fatalf("Load() scope = %q, want %q", inventorySnapshot.Scope(), install.ScopeGlobal)
	}
}

func TestInstallServicePreparePlanRequiresDependencies(t *testing.T) {
	t.Parallel()

	service := &InstallService{}
	_, err := service.preparePlan(context.Background(), install.OperationInstall, OperationRequest{Scope: install.ScopeProject})
	if err == nil {
		t.Fatal("preparePlan() error = nil, want non-nil")
	}
	if err.Error() == "" {
		t.Fatalf("preparePlan() error = %v", err)
	}
}
