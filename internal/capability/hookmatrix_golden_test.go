package capability

import (
	"os"
	"path/filepath"
	"testing"
)

func hookCapabilityMatrixJSONPath() string {
	return filepath.Join("..", "..", "testdata", "hook-capability-matrix.json")
}

func hookCapabilityMatrixMarkdownPath() string {
	return filepath.Join("..", "..", "docs", "hook-capability-matrix.md")
}

func assertMatchesHookGolden(t *testing.T, path string, actual []byte) {
	t.Helper()

	if os.Getenv("UPDATE_SNAPSHOTS") == "1" {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("create golden dir: %v", err)
		}
		if err := os.WriteFile(path, actual, 0o644); err != nil {
			t.Fatalf("write golden file: %v", err)
		}
		return
	}

	matches, diff, err := goldenMatches(path, actual)
	if err != nil {
		t.Fatalf("golden file not found: %s (run with UPDATE_SNAPSHOTS=1 go test ./internal/capability/... to create/regenerate)", path)
	}
	if !matches {
		t.Errorf("golden file diverges: %s\n\nRegenerate with: UPDATE_SNAPSHOTS=1 go test ./internal/capability/...\n\n%s", path, diff)
	}
}

func TestHookCapabilityMatrix_GoldenJSONAndMarkdownFromSameGeneration(t *testing.T) {
	m, err := GenerateHooks()
	if err != nil {
		t.Fatalf("GenerateHooks: %v", err)
	}
	jsonData, markdownData, err := RenderHooksBoth(m)
	if err != nil {
		t.Fatalf("RenderHooksBoth: %v", err)
	}
	t.Run("json", func(t *testing.T) {
		assertMatchesHookGolden(t, hookCapabilityMatrixJSONPath(), jsonData)
	})
	t.Run("markdown", func(t *testing.T) {
		assertMatchesHookGolden(t, hookCapabilityMatrixMarkdownPath(), markdownData)
	})
}

func TestHookCapabilityMatrix_DeterministicGeneration(t *testing.T) {
	m1, err := GenerateHooks()
	if err != nil {
		t.Fatalf("GenerateHooks (1): %v", err)
	}
	m2, err := GenerateHooks()
	if err != nil {
		t.Fatalf("GenerateHooks (2): %v", err)
	}

	json1, err := RenderHooksJSON(m1)
	if err != nil {
		t.Fatalf("RenderHooksJSON (1): %v", err)
	}
	json2, err := RenderHooksJSON(m2)
	if err != nil {
		t.Fatalf("RenderHooksJSON (2): %v", err)
	}
	if string(json1) != string(json2) {
		t.Errorf("GenerateHooks is not deterministic: two runs produced different JSON bytes")
	}

	md1 := RenderHooksMarkdown(m1)
	md2 := RenderHooksMarkdown(m2)
	if string(md1) != string(md2) {
		t.Errorf("GenerateHooks is not deterministic: two runs produced different Markdown bytes")
	}
}

func TestHookCapabilityMatrix_HasExactlyOneHundredCells(t *testing.T) {
	m, err := GenerateHooks()
	if err != nil {
		t.Fatalf("GenerateHooks: %v", err)
	}
	const want = 5 * 5 * 4
	if len(m.Cells) != want {
		t.Fatalf("expected %d cells (5 events x 5 families x 4 providers), got %d", want, len(m.Cells))
	}
}

func TestHookCapabilityMatrix_UnsupportedCellsNeverCarryTest(t *testing.T) {
	m, err := GenerateHooks()
	if err != nil {
		t.Fatalf("GenerateHooks: %v", err)
	}
	for _, c := range m.Cells {
		if c.State != "unsupported" {
			continue
		}
		if c.Test != "" {
			t.Errorf("cell provider=%s event=%s family=%s is unsupported but declares a test reference; only affirmative cells may reference dispatch evidence", c.Provider, c.Event, c.Family)
		}
		if c.Reason == "" {
			t.Errorf("cell provider=%s event=%s family=%s is unsupported but carries no reason (RF-17: silent weakening must never happen)", c.Provider, c.Event, c.Family)
		}
	}
}

func TestHookCapabilityMatrix_AffirmativeCellsCarryTestAndAdaptersCarryLimitation(t *testing.T) {
	m, err := GenerateHooks()
	if err != nil {
		t.Fatalf("GenerateHooks: %v", err)
	}
	found := 0
	for _, c := range m.Cells {
		if c.State != "verified" && c.State != "adapter" {
			continue
		}
		found++
		if c.Test == "" {
			t.Errorf("cell provider=%s event=%s family=%s is %s but has no resolved test", c.Provider, c.Event, c.Family, c.State)
		}
		if c.State == "adapter" && c.Limitation == "" {
			t.Errorf("cell provider=%s event=%s family=%s is adapter but carries no limitation (RF-59)", c.Provider, c.Event, c.Family)
		}
	}
	if found == 0 {
		t.Fatal("expected at least one affirmative cell to exercise this gate meaningfully")
	}
}
