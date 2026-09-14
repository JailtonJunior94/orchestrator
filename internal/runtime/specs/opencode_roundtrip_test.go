package specs_test

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/specs"
)

const authoredOpenCodeConfig = "{\n" +
	"    \"$schema\": \"https://opencode.ai/config.json\",\n" +
	"\t\"model\": \"anthropic/claude-sonnet-4\",\n" +
	"    \"theme\":     \"opencode\",\n" +
	"    \"share\": \"disabled\"\n" +
	"}\n"

func TestMergeOpenCodeConfigPreservesAuthoredFormatting(t *testing.T) {
	t.Parallel()

	merged, err := specs.MergeOpenCodeConfig([]byte(authoredOpenCodeConfig), specs.DefaultOpenCodePermission(), "")
	if err != nil {
		t.Fatalf("MergeOpenCodeConfig: %v", err)
	}

	for _, authored := range []string{
		"    \"$schema\": \"https://opencode.ai/config.json\",\n",
		"\t\"model\": \"anthropic/claude-sonnet-4\",\n",
		"    \"theme\":     \"opencode\",\n",
		"    \"share\": \"disabled\"",
	} {
		if !strings.Contains(string(merged), authored) {
			t.Errorf("authored formatting rewritten; missing byte-identical line %q in:\n%s", authored, merged)
		}
	}
}

func TestMergeOpenCodeConfigIsAByteIdenticalNoOpWhenAlreadyApplied(t *testing.T) {
	t.Parallel()

	first, err := specs.MergeOpenCodeConfig([]byte(authoredOpenCodeConfig), specs.DefaultOpenCodePermission(), "")
	if err != nil {
		t.Fatalf("MergeOpenCodeConfig: %v", err)
	}
	second, err := specs.MergeOpenCodeConfig(first, specs.DefaultOpenCodePermission(), "")
	if err != nil {
		t.Fatalf("MergeOpenCodeConfig (second run): %v", err)
	}

	if string(first) != string(second) {
		t.Errorf("install is not idempotent at byte level\nfirst:\n%s\nsecond:\n%s", first, second)
	}
}

func TestJSONTopLevelKeyRoundTripIsByteIdentical(t *testing.T) {
	t.Parallel()

	scenarios := []struct {
		name string
		raw  string
	}{
		{name: "authored non canonical layout", raw: authoredOpenCodeConfig},
		{name: "single member", raw: "{\n  \"model\": \"x\"\n}\n"},
		{name: "compact one line", raw: `{"a":1,"b":[2,3],"c":{"d":"e"}}`},
		{name: "tab indented", raw: "{\n\t\"a\": 1,\n\t\"b\": 2\n}\n"},
		{name: "trailing newlines", raw: "{\n  \"a\": \"}\"\n}\n\n"},
	}

	catalog := specs.NewCatalog()
	for _, scenario := range scenarios {
		t.Run(scenario.name, func(t *testing.T) {
			t.Parallel()

			inserted, err := catalog.SetJSONTopLevelKey([]byte(scenario.raw), "permission", specs.DefaultOpenCodePermission())
			if err != nil {
				t.Fatalf("SetJSONTopLevelKey: %v", err)
			}
			assertDecodes(t, inserted)

			restored, err := catalog.DeleteJSONTopLevelKey(inserted, "permission")
			if err != nil {
				t.Fatalf("DeleteJSONTopLevelKey: %v", err)
			}
			if string(restored) != scenario.raw {
				t.Errorf("round trip is not byte identical\nwant:\n%q\ngot:\n%q", scenario.raw, string(restored))
			}
		})
	}
}

func TestSetJSONTopLevelKeyReplacesExistingValueInPlace(t *testing.T) {
	t.Parallel()

	raw := "{\n    \"model\": \"old\",\n    \"theme\":     \"opencode\"\n}\n"
	updated, err := specs.NewCatalog().SetJSONTopLevelKey([]byte(raw), "model", "new")
	if err != nil {
		t.Fatalf("SetJSONTopLevelKey: %v", err)
	}
	want := "{\n    \"model\": \"new\",\n    \"theme\":     \"opencode\"\n}\n"
	if string(updated) != want {
		t.Errorf("in place replacement rewrote surrounding layout\nwant:\n%q\ngot:\n%q", want, string(updated))
	}
}

func TestSetJSONTopLevelKeyRejectsNonObject(t *testing.T) {
	t.Parallel()

	if _, err := specs.NewCatalog().SetJSONTopLevelKey([]byte(`[1,2]`), "permission", map[string]any{}); err == nil {
		t.Error("expected an error for a non-object document")
	}
}

func assertDecodes(t *testing.T, data []byte) {
	t.Helper()
	doc := map[string]any{}
	if err := json.Unmarshal(data, &doc); err != nil {
		t.Fatalf("surgical edit produced invalid JSON: %v\n%s", err, data)
	}
	permission, ok := doc["permission"].(map[string]any)
	if !ok {
		t.Fatalf("permission block missing after insertion:\n%s", data)
	}
	if !reflect.DeepEqual(permission, toAnyMap(specs.DefaultOpenCodePermission())) {
		t.Errorf("permission block content changed: %#v", permission)
	}
}

func toAnyMap(in map[string]any) map[string]any {
	encoded, err := json.Marshal(in)
	if err != nil {
		return nil
	}
	out := map[string]any{}
	if err := json.Unmarshal(encoded, &out); err != nil {
		return nil
	}
	return out
}
