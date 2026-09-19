package capability

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func capabilityMatrixJSONPath() string {
	return filepath.Join("..", "..", "testdata", "capability-matrix.json")
}

func capabilityMatrixMarkdownPath() string {
	return filepath.Join("..", "..", "docs", "capability-matrix.md")
}

func goldenMatches(path string, actual []byte) (bool, string, error) {
	expected, err := os.ReadFile(path)
	if err != nil {
		return false, "", err
	}
	if string(expected) == string(actual) {
		return true, "", nil
	}
	return false, goldenDiff(string(expected), string(actual)), nil
}

func assertMatchesGolden(t *testing.T, path string, actual []byte) {
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

func goldenDiff(expected, actual string) string {
	expLines := strings.Split(expected, "\n")
	actLines := strings.Split(actual, "\n")
	maxLen := len(expLines)
	if len(actLines) > maxLen {
		maxLen = len(actLines)
	}
	var b strings.Builder
	diffs := 0
	for i := 0; i < maxLen && diffs < 20; i++ {
		var exp, act string
		if i < len(expLines) {
			exp = expLines[i]
		}
		if i < len(actLines) {
			act = actLines[i]
		}
		if exp != act {
			fmt.Fprintf(&b, "line %d:\n  expected: %q\n  actual:   %q\n", i+1, exp, act)
			diffs++
		}
	}
	if diffs == 20 {
		b.WriteString("... (truncated after 20 differences)\n")
	}
	return b.String()
}

func TestCapabilityMatrix_GoldenJSONAndMarkdownFromSameGeneration(t *testing.T) {
	m, err := Generate()
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	jsonData, markdownData, err := RenderBoth(m)
	if err != nil {
		t.Fatalf("RenderBoth: %v", err)
	}
	t.Run("json", func(t *testing.T) {
		assertMatchesGolden(t, capabilityMatrixJSONPath(), jsonData)
	})
	t.Run("markdown", func(t *testing.T) {
		assertMatchesGolden(t, capabilityMatrixMarkdownPath(), markdownData)
	})
}

func TestCapabilityMatrix_DeterministicGeneration(t *testing.T) {
	m1, err := Generate()
	if err != nil {
		t.Fatalf("Generate (1): %v", err)
	}
	m2, err := Generate()
	if err != nil {
		t.Fatalf("Generate (2): %v", err)
	}

	json1, err := RenderJSON(m1)
	if err != nil {
		t.Fatalf("RenderJSON (1): %v", err)
	}
	json2, err := RenderJSON(m2)
	if err != nil {
		t.Fatalf("RenderJSON (2): %v", err)
	}
	if string(json1) != string(json2) {
		t.Errorf("Generate is not deterministic: two runs produced different JSON bytes")
	}

	md1 := RenderMarkdown(m1)
	md2 := RenderMarkdown(m2)
	if string(md1) != string(md2) {
		t.Errorf("Generate is not deterministic: two runs produced different Markdown bytes")
	}
}

func TestCapabilityMatrix_NoSupportedCellWithoutResolvedTest(t *testing.T) {
	m, err := Generate()
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	for _, c := range m.Cells {
		if c.State != StateSupported && c.State != StateProviderCapability {
			continue
		}
		if c.Test == "" {
			t.Errorf("cell provider=%s capability=%s is %s but has no resolved test", c.Provider, c.Capability, c.State)
		}
	}
}

func TestCapabilityMatrix_UnsupportedCellsCarryReason(t *testing.T) {
	m, err := Generate()
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	for _, c := range m.Cells {
		if c.State != StateUnsupported {
			continue
		}
		if c.Reason == "" {
			t.Errorf("cell provider=%s capability=%s is unsupported but carries no reason (RF-17: silent weakening must never happen)", c.Provider, c.Capability)
		}
		if c.Test != "" {
			t.Errorf("cell provider=%s capability=%s is unsupported but declares a test reference; only supported cells may reference evidence", c.Provider, c.Capability)
		}
	}
}

func TestCapabilityMatrix_UnknownCellsAreJustified(t *testing.T) {
	m, err := Generate()
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	found := false
	for _, c := range m.Cells {
		if c.State != StateUnknown {
			continue
		}
		found = true
		if c.Reason == "" {
			t.Errorf("cell provider=%s capability=%s is unknown but carries no reason", c.Provider, c.Capability)
		}
	}
	if !found {
		t.Fatal("expected at least one unknown cell (self-satisfied invariants, V-25) to exercise this gate meaningfully")
	}
}
