package inventory

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	install "github.com/jailtonjunior/orchestrator/internal/install/domain"
	"github.com/jailtonjunior/orchestrator/internal/platform"
)

type stubStateDirResolver struct {
	path string
	err  error
}

func (r stubStateDirResolver) UserStateDir() (string, error) {
	if r.err != nil {
		return "", r.err
	}
	return r.path, nil
}

func TestResolvePath(t *testing.T) {
	t.Parallel()

	resolver := stubStateDirResolver{path: filepath.Join("home", "tester", ".local", "state")}

	projectPath, err := ResolvePath(resolver, install.ScopeProject, filepath.Join("repo"))
	if err != nil {
		t.Fatalf("ResolvePath(project) error = %v", err)
	}
	if want := filepath.Join("repo", ".orq", "install", fileName); projectPath != want {
		t.Fatalf("ResolvePath(project) = %q, want %q", projectPath, want)
	}

	globalPath, err := ResolvePath(resolver, install.ScopeGlobal, "")
	if err != nil {
		t.Fatalf("ResolvePath(global) error = %v", err)
	}
	if want := filepath.Join("home", "tester", ".local", "state", "orq", "install", fileName); globalPath != want {
		t.Fatalf("ResolvePath(global) = %q, want %q", globalPath, want)
	}
}

func TestFileStoreLoadMissingReturnsEmptyInventory(t *testing.T) {
	t.Parallel()

	fileSystem, err := platform.NewFakeFileSystem()
	if err != nil {
		t.Fatalf("NewFakeFileSystem() error = %v", err)
	}
	defer closeFakeFileSystem(t, fileSystem)

	now := time.Date(2026, 4, 3, 12, 0, 0, 0, time.UTC)
	store := NewStore(fileSystem, platform.NewFakeClock(now))

	location := filepath.Join(fileSystem.Root(), ".orq", "install", fileName)
	inventory, err := store.Load(context.Background(), install.ScopeProject, location)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if inventory.Scope() != install.ScopeProject {
		t.Fatalf("inventory.Scope() = %q", inventory.Scope())
	}
	if inventory.SchemaVersion() != schemaVersion {
		t.Fatalf("inventory.SchemaVersion() = %d", inventory.SchemaVersion())
	}
	if len(inventory.Entries()) != 0 {
		t.Fatalf("inventory.Entries() len = %d, want 0", len(inventory.Entries()))
	}
}

func TestFileStoreSaveAndLoadRoundTrip(t *testing.T) {
	t.Parallel()

	fileSystem, err := platform.NewFakeFileSystem()
	if err != nil {
		t.Fatalf("NewFakeFileSystem() error = %v", err)
	}
	defer closeFakeFileSystem(t, fileSystem)

	now := time.Date(2026, 4, 3, 12, 30, 0, 0, time.UTC)
	clock := platform.NewFakeClock(now)
	store := NewStore(fileSystem, clock)
	location := filepath.Join(fileSystem.Root(), "state", "orq", "install", fileName)

	entry, err := install.NewInventoryEntry(
		install.ProviderCodex,
		"codex:skill:reviewer",
		install.AssetKindSkill,
		filepath.Join("home", "tester", ".codex", "skills", "reviewer"),
		"sha256:123",
		install.VerificationStatusPartial,
		&now,
	)
	if err != nil {
		t.Fatalf("NewInventoryEntry() error = %v", err)
	}

	inventory, err := install.NewInventory(schemaVersion, install.ScopeGlobal, location, []install.InventoryEntry{entry}, now)
	if err != nil {
		t.Fatalf("NewInventory() error = %v", err)
	}

	if err := store.Save(context.Background(), install.ScopeGlobal, location, inventory); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	loaded, err := store.Load(context.Background(), install.ScopeGlobal, location)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if loaded.Location() != location {
		t.Fatalf("loaded.Location() = %q, want %q", loaded.Location(), location)
	}
	if len(loaded.Entries()) != 1 {
		t.Fatalf("loaded.Entries() len = %d, want 1", len(loaded.Entries()))
	}

	got := loaded.Entries()[0]
	if got.Fingerprint() != "sha256:123" {
		t.Fatalf("loaded fingerprint = %q", got.Fingerprint())
	}
	if got.LastVerifiedAt() == nil || !got.LastVerifiedAt().Equal(now) {
		t.Fatalf("loaded last verified at = %v, want %v", got.LastVerifiedAt(), now)
	}
}

func TestFileStoreLoadInvalidJSON(t *testing.T) {
	t.Parallel()

	fileSystem, err := platform.NewFakeFileSystem()
	if err != nil {
		t.Fatalf("NewFakeFileSystem() error = %v", err)
	}
	defer closeFakeFileSystem(t, fileSystem)

	store := NewStore(fileSystem, platform.NewFakeClock(time.Now().UTC()))
	location := filepath.Join(fileSystem.Root(), ".orq", "install", fileName)
	if err := fileSystem.WriteFile(location, []byte("{invalid"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	if _, err := store.Load(context.Background(), install.ScopeProject, location); err == nil {
		t.Fatal("Load() expected error")
	}
}

func TestFileStoreSaveScopeMismatch(t *testing.T) {
	t.Parallel()

	fileSystem, err := platform.NewFakeFileSystem()
	if err != nil {
		t.Fatalf("NewFakeFileSystem() error = %v", err)
	}
	defer closeFakeFileSystem(t, fileSystem)

	now := time.Now().UTC()
	store := NewStore(fileSystem, platform.NewFakeClock(now))
	location := filepath.Join(fileSystem.Root(), ".orq", "install", fileName)
	inventory, err := install.NewInventory(schemaVersion, install.ScopeProject, location, nil, now)
	if err != nil {
		t.Fatalf("NewInventory() error = %v", err)
	}

	if err := store.Save(context.Background(), install.ScopeGlobal, location, inventory); err == nil {
		t.Fatal("Save() expected scope mismatch error")
	}
}

func TestFileStoreLoadPropagatesContextCancellation(t *testing.T) {
	t.Parallel()

	store := NewStore(platform.OSFileSystem{}, platform.NewFakeClock(time.Now().UTC()))
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if _, err := store.Load(ctx, install.ScopeProject, "inventory.json"); !errors.Is(err, context.Canceled) {
		t.Fatalf("Load() error = %v, want context.Canceled", err)
	}
}

func TestFileStoreSaveWritesAtomically(t *testing.T) {
	t.Parallel()

	fileSystem, err := platform.NewFakeFileSystem()
	if err != nil {
		t.Fatalf("NewFakeFileSystem() error = %v", err)
	}
	defer closeFakeFileSystem(t, fileSystem)

	now := time.Now().UTC()
	store := NewStore(fileSystem, platform.NewFakeClock(now))
	location := filepath.Join(fileSystem.Root(), ".orq", "install", fileName)
	inventory, err := install.NewInventory(schemaVersion, install.ScopeProject, location, nil, now)
	if err != nil {
		t.Fatalf("NewInventory() error = %v", err)
	}

	if err := store.Save(context.Background(), install.ScopeProject, location, inventory); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	if _, err := fileSystem.Stat(location + ".tmp"); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("temp file must be removed after atomic write, got err = %v", err)
	}
}

func closeFakeFileSystem(t *testing.T, fileSystem *platform.FakeFileSystem) {
	t.Helper()

	if err := fileSystem.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
}
