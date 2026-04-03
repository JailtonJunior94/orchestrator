package catalog

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	install "github.com/jailtonjunior/orchestrator/internal/install/domain"
)

func TestFileCatalogDiscover(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	mustWriteFile(t, filepath.Join(root, ".claude", "commands", "summarize.md"), "# summarize")
	mustWriteFile(t, filepath.Join(root, ".claude", "skills", "reviewer", "SKILL.md"), "# reviewer")
	mustWriteFile(t, filepath.Join(root, ".claude", "skills", "reviewer", "notes.txt"), "ignored companion")
	mustWriteFile(t, filepath.Join(root, ".gemini", "commands", "draft.md"), "# draft")
	mustWriteFile(t, filepath.Join(root, ".gemini", "skills", "planner", "SKILL.md"), "# planner")
	mustWriteFile(t, filepath.Join(root, ".codex", "skills", "refactor", "SKILL.md"), "# refactor")
	mustWriteFile(t, filepath.Join(root, ".github", "copilot-instructions.md"), "# copilot")
	mustWriteFile(t, filepath.Join(root, "AGENTS.md"), "# agents")
	mustWriteFile(t, filepath.Join(root, ".codex", "skills", "broken", "README.md"), "missing skill entry")

	catalog := New()
	assets, err := catalog.Discover(context.Background(), root)
	if err != nil {
		t.Fatalf("Discover() error = %v", err)
	}

	if len(assets) != 7 {
		t.Fatalf("Discover() len = %d, want 7", len(assets))
	}

	assertAsset(t, assets, "agents", install.AssetKindInstruction, []install.Provider{install.ProviderCopilot})
	assertAsset(t, assets, "copilot-instructions", install.AssetKindInstruction, []install.Provider{install.ProviderCopilot})
	assertAsset(t, assets, "summarize", install.AssetKindCommand, []install.Provider{install.ProviderClaude, install.ProviderCopilot})
	assertAsset(t, assets, "draft", install.AssetKindCommand, []install.Provider{install.ProviderGemini})
	assertAsset(t, assets, "reviewer", install.AssetKindSkill, []install.Provider{install.ProviderClaude, install.ProviderCopilot})
	assertAsset(t, assets, "planner", install.AssetKindSkill, []install.Provider{install.ProviderGemini})
	assertAsset(t, assets, "refactor", install.AssetKindSkill, []install.Provider{install.ProviderCodex})
	assertAssetID(t, assets, "agents", "root:instruction:agents")
}

func TestFileCatalogDiscoverIgnoresMissingSources(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	catalog := New()

	assets, err := catalog.Discover(context.Background(), root)
	if err != nil {
		t.Fatalf("Discover() error = %v", err)
	}
	if len(assets) != 0 {
		t.Fatalf("Discover() len = %d, want 0", len(assets))
	}
}

func TestFileCatalogDiscoverRespectsContextCancellation(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	catalog := New()
	if _, err := catalog.Discover(ctx, root); err == nil {
		t.Fatal("Discover() expected cancellation error")
	}
}

func assertAsset(t *testing.T, assets []install.Asset, name string, kind install.AssetKind, wantProviders []install.Provider) {
	t.Helper()

	for _, asset := range assets {
		if asset.Name() != name || asset.Kind() != kind {
			continue
		}

		gotProviders := asset.Providers()
		if len(gotProviders) != len(wantProviders) {
			t.Fatalf("asset %q providers = %v, want %v", name, gotProviders, wantProviders)
		}
		for index := range wantProviders {
			if gotProviders[index] != wantProviders[index] {
				t.Fatalf("asset %q providers = %v, want %v", name, gotProviders, wantProviders)
			}
		}

		switch kind {
		case install.AssetKindSkill:
			if filepath.Base(asset.Metadata().EntryPath) != "SKILL.md" {
				t.Fatalf("skill asset %q entry path = %q", name, asset.Metadata().EntryPath)
			}
		case install.AssetKindInstruction, install.AssetKindCommand:
			if asset.SourcePath() != asset.Metadata().EntryPath {
				t.Fatalf("file asset %q source path = %q, entry path = %q", name, asset.SourcePath(), asset.Metadata().EntryPath)
			}
		}
		return
	}

	t.Fatalf("asset %q kind %q not found", name, kind)
}

func mustWriteFile(t *testing.T, path string, content string) {
	t.Helper()

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll(%q) error = %v", path, err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile(%q) error = %v", path, err)
	}
}

func assertAssetID(t *testing.T, assets []install.Asset, name string, want string) {
	t.Helper()

	for _, asset := range assets {
		if asset.Name() != name {
			continue
		}
		if asset.ID() != want {
			t.Fatalf("asset %q id = %q, want %q", name, asset.ID(), want)
		}
		return
	}

	t.Fatalf("asset %q not found for id assertion", name)
}
