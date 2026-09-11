package specs_test

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/specs"
)

func TestMergeOpenCodeConfigPreservesSchemaAndExistingFields(t *testing.T) {
	t.Parallel()

	existing := []byte(`{"$schema":"https://opencode.ai/config.json","theme":"dark"}`)
	out, err := specs.MergeOpenCodeConfig(existing, specs.DefaultOpenCodePermission(), "")
	if err != nil {
		t.Fatalf("MergeOpenCodeConfig: %v", err)
	}

	var doc map[string]any
	if err := json.Unmarshal(out, &doc); err != nil {
		t.Fatalf("decode result: %v", err)
	}

	if doc["$schema"] != "https://opencode.ai/config.json" {
		t.Errorf("$schema = %v; want preserved", doc["$schema"])
	}
	if doc["theme"] != "dark" {
		t.Errorf("theme = %v; want preserved", doc["theme"])
	}
	if _, ok := doc["permission"]; !ok {
		t.Error("permission block missing from merged config")
	}
}

func TestMergeOpenCodeConfigNeverWritesSkillsOrInstructions(t *testing.T) {
	t.Parallel()

	out, err := specs.MergeOpenCodeConfig(nil, specs.DefaultOpenCodePermission(), "")
	if err != nil {
		t.Fatalf("MergeOpenCodeConfig: %v", err)
	}

	var doc map[string]any
	if err := json.Unmarshal(out, &doc); err != nil {
		t.Fatalf("decode result: %v", err)
	}
	if _, ok := doc["skills"]; ok {
		t.Error("installer wrote skills key — must never write it (RF-13)")
	}
	if _, ok := doc["instructions"]; ok {
		t.Error("installer wrote instructions key — must never write it (RF-13)")
	}
}

func TestMergeOpenCodeConfigPreservesExistingSkillsAndInstructions(t *testing.T) {
	t.Parallel()

	existing := []byte(`{"skills":{"paths":["./x"]},"instructions":["AGENTS.md"]}`)
	out, err := specs.MergeOpenCodeConfig(existing, specs.DefaultOpenCodePermission(), "")
	if err != nil {
		t.Fatalf("MergeOpenCodeConfig: %v", err)
	}

	var doc map[string]any
	if err := json.Unmarshal(out, &doc); err != nil {
		t.Fatalf("decode result: %v", err)
	}
	if _, ok := doc["skills"]; !ok {
		t.Error("preexisting skills key was removed — installer must never strip user config")
	}
	if _, ok := doc["instructions"]; !ok {
		t.Error("preexisting instructions key was removed — installer must never strip user config")
	}
}

func TestMergeOpenCodeConfigIsIdempotent(t *testing.T) {
	t.Parallel()

	existing := []byte(`{"$schema":"https://opencode.ai/config.json"}`)
	first, err := specs.MergeOpenCodeConfig(existing, specs.DefaultOpenCodePermission(), "")
	if err != nil {
		t.Fatalf("MergeOpenCodeConfig first: %v", err)
	}
	second, err := specs.MergeOpenCodeConfig(first, specs.DefaultOpenCodePermission(), "")
	if err != nil {
		t.Fatalf("MergeOpenCodeConfig second: %v", err)
	}
	if !strings.EqualFold(string(first), string(second)) && string(first) != string(second) {
		t.Fatalf("merge is not idempotent:\nfirst=%s\nsecond=%s", first, second)
	}
}

func TestMergeOpenCodeConfigWritesModelKey(t *testing.T) {
	t.Parallel()

	out, err := specs.MergeOpenCodeConfig(nil, nil, "claude-opus-4")
	if err != nil {
		t.Fatalf("MergeOpenCodeConfig: %v", err)
	}
	var doc map[string]any
	if err := json.Unmarshal(out, &doc); err != nil {
		t.Fatalf("decode result: %v", err)
	}
	if doc["model"] != "claude-opus-4" {
		t.Errorf("model = %v; want %q", doc["model"], "claude-opus-4")
	}
}

func TestDefaultOpenCodePermissionForbidsWildcardDenyAndAsk(t *testing.T) {
	t.Parallel()

	var walk func(v any)
	walk = func(v any) {
		switch val := v.(type) {
		case string:
			if val == "ask" {
				t.Error("permission block must never use \"ask\" in orchestration (RF-20)")
			}
		case map[string]any:
			for _, nested := range val {
				walk(nested)
			}
		}
	}
	perm := specs.DefaultOpenCodePermission()
	for tool, rule := range perm {
		if rule == "deny" {
			t.Errorf("tool %q has a blanket deny — removes the tool from the model's set (RF-20)", tool)
		}
		walk(rule)
	}
}

func TestValidatePermissionBlockRejectsAsk(t *testing.T) {
	t.Parallel()

	err := specs.ValidatePermissionBlock(map[string]any{"bash": "ask"})
	if !errors.Is(err, specs.ErrOpenCodePermissionForbidsAsk) {
		t.Fatalf("ValidatePermissionBlock = %v; want ErrOpenCodePermissionForbidsAsk", err)
	}
}

func TestValidatePermissionBlockRejectsNestedAsk(t *testing.T) {
	t.Parallel()

	err := specs.ValidatePermissionBlock(map[string]any{
		"bash": map[string]any{"rm -rf*": "ask"},
	})
	if !errors.Is(err, specs.ErrOpenCodePermissionForbidsAsk) {
		t.Fatalf("ValidatePermissionBlock (nested) = %v; want ErrOpenCodePermissionForbidsAsk", err)
	}
}

func TestValidatePermissionBlockRejectsWildcardDenyOnRequiredTool(t *testing.T) {
	t.Parallel()

	err := specs.ValidatePermissionBlock(map[string]any{"edit": "deny"}, "edit")
	if !errors.Is(err, specs.ErrOpenCodePermissionWildcardDeny) {
		t.Fatalf("ValidatePermissionBlock = %v; want ErrOpenCodePermissionWildcardDeny", err)
	}
}

func TestValidatePermissionBlockAllowsWildcardDenyOnToolNotRequired(t *testing.T) {
	t.Parallel()

	err := specs.ValidatePermissionBlock(map[string]any{"someunneeded": "deny"}, "edit")
	if err != nil {
		t.Fatalf("ValidatePermissionBlock = %v; want nil for a tool the harness does not require", err)
	}
}

func TestValidatePermissionBlockAcceptsDefaultOpenCodePermission(t *testing.T) {
	t.Parallel()

	err := specs.ValidatePermissionBlock(specs.DefaultOpenCodePermission())
	if err != nil {
		t.Fatalf("ValidatePermissionBlock(DefaultOpenCodePermission()) = %v; want nil", err)
	}
}

func TestValidatePermissionBlockAcceptsDefaultOpenCodePermissionWithRequiredTools(t *testing.T) {
	t.Parallel()

	requiredTools := []string{"bash", "edit", "write", "multiedit", "patch"}
	err := specs.ValidatePermissionBlock(specs.DefaultOpenCodePermission(), requiredTools...)
	if err != nil {
		t.Fatalf("ValidatePermissionBlock(DefaultOpenCodePermission(), requiredTools...) = %v; want nil", err)
	}
}
