package events

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"slices"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func openCodeMutatingTools(t *testing.T) []string {
	t.Helper()

	pluginPath := filepath.Join("..", "..", "embedded", "assets", ".opencode", "plugin", "governance.js")
	data, err := os.ReadFile(pluginPath)
	if err != nil {
		t.Fatalf("read opencode governance plugin: %v", err)
	}

	declaration := regexp.MustCompile(`MUTATING_TOOLS\s*=\s*new Set\(\[([^\]]*)\]\)`)
	match := declaration.FindStringSubmatch(string(data))
	if match == nil {
		t.Fatalf("MUTATING_TOOLS declaration not found in %s", pluginPath)
	}

	tools := make([]string, 0, 6)
	for _, raw := range strings.Split(match[1], ",") {
		name := strings.Trim(strings.TrimSpace(raw), `"'`)
		if name != "" {
			tools = append(tools, name)
		}
	}
	if len(tools) == 0 {
		t.Fatalf("MUTATING_TOOLS parsed empty from %s", pluginPath)
	}
	return tools
}

func TestNormalizeOpenCodeCoversEveryMutatingToolOfThePlugin(t *testing.T) {
	t.Parallel()

	input := json.RawMessage(`{"command":"echo hello"}`)
	want := map[string]string{
		"bash":        "bash",
		"write":       "write",
		"edit":        "edit",
		"multiedit":   "edit",
		"patch":       "edit",
		"apply_patch": "edit",
	}

	for _, tool := range openCodeMutatingTools(t) {
		expected, declared := want[tool]
		if !declared {
			t.Fatalf("plugin declares mutating tool %q with no expected normalized_name; "+
				"extend aliases.opencode in normalization-rules.yaml and this table", tool)
		}

		result, err := NewCatalog().BuildNormalizedToolCall("opencode", tool, input, "")
		if err != nil {
			t.Fatalf("BuildNormalizedToolCall(opencode, %s): %v", tool, err)
		}
		if result.RawName != tool {
			t.Errorf("RawName = %q; want %q (raw name must never be mutated)", result.RawName, tool)
		}
		if result.NormalizedName != expected {
			t.Errorf("opencode %q: NormalizedName = %q; want %q", tool, result.NormalizedName, expected)
		}
	}
}

func TestNormalizeOpenCodeExplicitTableBeatsInheritCommon(t *testing.T) {
	t.Parallel()

	input := json.RawMessage(`{}`)
	for _, inherited := range []string{"read_file", "write_file", "str_replace_editor"} {
		result, err := NewCatalog().BuildNormalizedToolCall("opencode", inherited, input, "")
		if err != nil {
			t.Fatalf("BuildNormalizedToolCall(opencode, %s): %v", inherited, err)
		}
		if result.NormalizedName != inherited {
			t.Errorf("opencode %q: NormalizedName = %q; common_aliases must not leak into opencode, "+
				"the OpenCode CLI never emits this name", inherited, result.NormalizedName)
		}
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

	if len(sourceInherit) == 0 {
		t.Fatal("inherit_common is empty; RF-06 requires the shared-alias structure to stay declared")
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

func TestNormalizationRulesMirrorIsByteIdentical(t *testing.T) {
	t.Parallel()

	sourcePath := filepath.Join("..", "..", "..", ".agents", "normalization-rules.yaml")
	sourceData, err := os.ReadFile(sourcePath)
	if err != nil {
		t.Fatalf("read source normalization-rules.yaml: %v", err)
	}

	if string(sourceData) != string(_defaultRulesYAML) {
		t.Error(".agents/normalization-rules.yaml diverges from the embedded copy; " +
			"the workspace file wins at load time, so drift changes normalization silently")
	}
}

func TestAliasesDeclareExplicitTablePerRegisteredDriver(t *testing.T) {
	t.Parallel()

	var rules normalizationRules
	if err := yaml.Unmarshal(_defaultRulesYAML, &rules); err != nil {
		t.Fatalf("decode embedded normalization-rules.yaml: %v", err)
	}

	for _, driver := range []string{"claude", "codex", "copilot", "opencode"} {
		table, ok := rules.Aliases[driver]
		if !ok || len(table) == 0 {
			t.Errorf("aliases.%s missing or empty; RF-18 requires an own alias table per CLI", driver)
		}
	}

	if _, ok := rules.InputMappings["opencode"]; !ok {
		t.Error("input_mappings.opencode missing; RF-18 requires the bash command key to be declared")
	}

	openCode := rules.Aliases["opencode"]
	inherited := map[string]string{"read_file": "read", "write_file": "write", "str_replace_editor": "edit"}
	if reflect.DeepEqual(openCode, inherited) {
		t.Error("aliases.opencode equals common_aliases; the OpenCode CLI emits none of those names")
	}
}
