package skills_test

import (
	"slices"
	"testing"

	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/specs"
	"github.com/JailtonJunior94/ai-spec-harness/internal/skills"
)

func TestAllToolsDerivesFromRegistry(t *testing.T) {
	t.Parallel()

	order := specs.NewCatalog().CanonicalOrder()
	got := make([]string, 0, len(skills.AllTools))
	for _, tool := range skills.AllTools {
		got = append(got, string(tool))
	}
	if !slices.Equal(got, order) {
		t.Fatalf("skills.AllTools = %v; registry canonical order = %v", got, order)
	}

	for _, id := range order {
		if _, ok := skills.NewCatalog().ParseTool(id); !ok {
			t.Errorf("ParseTool(%q) rejected a registry id", id)
		}
	}
	if _, ok := skills.NewCatalog().ParseTool("nonexistent"); ok {
		t.Error("ParseTool accepted a tool outside the registry")
	}
}
