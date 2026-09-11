package events

import (
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestNormalizeOpenCodeInheritsCommonAliases(t *testing.T) {
	t.Parallel()

	input := json.RawMessage(`{"command":"echo hello"}`)
	result, err := NewCatalog().BuildNormalizedToolCall("opencode", "bash", input, "")
	if err != nil {
		t.Fatalf("BuildNormalizedToolCall unexpected error: %v", err)
	}
	if result.NormalizedName != "bash" {
		t.Errorf("NormalizedName = %q; want %q", result.NormalizedName, "bash")
	}

	result, err = NewCatalog().BuildNormalizedToolCall("opencode", "str_replace_editor", input, "")
	if err != nil {
		t.Fatalf("BuildNormalizedToolCall unexpected error: %v", err)
	}
	if result.NormalizedName != "edit" {
		t.Errorf("NormalizedName = %q; want %q", result.NormalizedName, "edit")
	}
}

func TestInheritCommonMirrorsBetweenSourceAndEmbedded(t *testing.T) {
	t.Parallel()

	sourcePath := filepath.Join("..", "..", "..", ".agents", "normalization-rules.yaml")
	sourceData, err := os.ReadFile(sourcePath)
	if err != nil {
		t.Fatalf("read source normalization-rules.yaml: %v", err)
	}

	var sourceRules normalizationRules
	if err := yaml.Unmarshal(sourceData, &sourceRules); err != nil {
		t.Fatalf("decode source normalization-rules.yaml: %v", err)
	}

	var embeddedRules normalizationRules
	if err := yaml.Unmarshal(_defaultRulesYAML, &embeddedRules); err != nil {
		t.Fatalf("decode embedded normalization-rules.yaml: %v", err)
	}

	sourceInherit := slices.Clone(sourceRules.InheritCommon)
	embeddedInherit := slices.Clone(embeddedRules.InheritCommon)
	slices.Sort(sourceInherit)
	slices.Sort(embeddedInherit)

	if !slices.Equal(sourceInherit, embeddedInherit) {
		t.Fatalf("inherit_common drift: source=%v embedded=%v", sourceInherit, embeddedInherit)
	}

	for _, driver := range []string{"opencode"} {
		if !slices.Contains(sourceInherit, driver) {
			t.Errorf("source inherit_common missing %q", driver)
		}
		if !slices.Contains(embeddedInherit, driver) {
			t.Errorf("embedded inherit_common missing %q", driver)
		}
	}
}
