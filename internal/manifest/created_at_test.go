package manifest

import (
	"testing"
	"time"

	"github.com/JailtonJunior94/ai-spec-harness/internal/fs"
)

func TestSave_PreservesCreatedAtFromExistingManifest(t *testing.T) {
	t.Parallel()

	ffs := fs.NewFakeFileSystem()
	ffs.Dirs["/project"] = true
	store := NewStore(ffs)

	first := &Manifest{
		Version:   "1.0.0",
		CreatedAt: time.Date(2024, 1, 2, 3, 4, 5, 0, time.UTC),
		UpdatedAt: time.Date(2024, 1, 2, 3, 4, 5, 0, time.UTC),
	}
	if err := store.Save("/project", first); err != nil {
		t.Fatalf("primeira gravacao: %v", err)
	}

	second := &Manifest{
		Version:   "1.1.0",
		CreatedAt: time.Date(2026, 9, 14, 0, 0, 0, 0, time.UTC),
		UpdatedAt: time.Date(2026, 9, 14, 0, 0, 0, 0, time.UTC),
	}
	if err := store.Save("/project", second); err != nil {
		t.Fatalf("segunda gravacao: %v", err)
	}

	loaded, err := store.Load("/project")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if !loaded.CreatedAt.Equal(first.CreatedAt) {
		t.Errorf("created_at = %v; want %v (herdado do manifesto existente)", loaded.CreatedAt, first.CreatedAt)
	}
	if !loaded.UpdatedAt.Equal(second.UpdatedAt) {
		t.Errorf("updated_at = %v; want %v", loaded.UpdatedAt, second.UpdatedAt)
	}
}

func TestSave_SetsCreatedAtOnFirstInstall(t *testing.T) {
	t.Parallel()

	ffs := fs.NewFakeFileSystem()
	ffs.Dirs["/project"] = true
	store := NewStore(ffs)

	created := time.Date(2026, 9, 14, 10, 0, 0, 0, time.UTC)
	if err := store.Save("/project", &Manifest{Version: "1.0.0", CreatedAt: created}); err != nil {
		t.Fatalf("Save: %v", err)
	}

	loaded, err := store.Load("/project")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if !loaded.CreatedAt.Equal(created) {
		t.Errorf("created_at = %v; want %v", loaded.CreatedAt, created)
	}
}
