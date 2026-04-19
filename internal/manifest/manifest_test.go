package manifest_test

import (
	"testing"
	"time"

	"github.com/JailtonJunior94/ai-spec-harness/internal/fs"
	"github.com/JailtonJunior94/ai-spec-harness/internal/manifest"
	"github.com/JailtonJunior94/ai-spec-harness/internal/skills"
)

func TestExists_false(t *testing.T) {
	store := manifest.NewStore(fs.NewFakeFileSystem())
	if store.Exists("/project") {
		t.Error("Exists() should be false when manifest not present")
	}
}

func TestSaveAndLoad_roundtrip(t *testing.T) {
	fsys := fs.NewFakeFileSystem()
	store := manifest.NewStore(fsys)

	m := &manifest.Manifest{
		Version:   "1.0.0",
		CreatedAt: time.Now().UTC().Truncate(time.Second),
		UpdatedAt: time.Now().UTC().Truncate(time.Second),
		SourceDir: "/source",
		LinkMode:  skills.LinkCopy,
		Tools:     []skills.Tool{skills.ToolClaude},
		Langs:     []skills.Lang{skills.LangGo},
		Skills:    []string{"bugfix", "review"},
		Checksums: map[string]string{"file.md": "abc123"},
	}

	if err := store.Save("/project", m); err != nil {
		t.Fatalf("Save() error: %v", err)
	}

	if !store.Exists("/project") {
		t.Error("Exists() should be true after Save()")
	}

	got, err := store.Load("/project")
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	if got.Version != m.Version {
		t.Errorf("Version = %q, want %q", got.Version, m.Version)
	}
	if got.SourceDir != m.SourceDir {
		t.Errorf("SourceDir = %q, want %q", got.SourceDir, m.SourceDir)
	}
	if got.LinkMode != m.LinkMode {
		t.Errorf("LinkMode = %q, want %q", got.LinkMode, m.LinkMode)
	}
	if len(got.Skills) != len(m.Skills) {
		t.Errorf("Skills len = %d, want %d", len(got.Skills), len(m.Skills))
	}
	if got.Checksums["file.md"] != "abc123" {
		t.Errorf("Checksums[file.md] = %q, want 'abc123'", got.Checksums["file.md"])
	}
}

func TestLoad_missingFile(t *testing.T) {
	store := manifest.NewStore(fs.NewFakeFileSystem())
	_, err := store.Load("/no/such/project")
	if err == nil {
		t.Error("Load() should return error when manifest file does not exist")
	}
}

func TestLoad_invalidJSON(t *testing.T) {
	fsys := fs.NewFakeFileSystem()
	_ = fsys.WriteFile("/project/"+manifest.ManifestFile, []byte("not-json"))
	store := manifest.NewStore(fsys)
	_, err := store.Load("/project")
	if err == nil {
		t.Error("Load() should return error for invalid JSON")
	}
}

func TestManifestFile_constant(t *testing.T) {
	if manifest.ManifestFile == "" {
		t.Error("ManifestFile constant should not be empty")
	}
}
