package specs_test

import (
	"strings"
	"testing"

	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/specs"
)

func TestNoLauncherUsesLatestVersion(t *testing.T) {
	t.Parallel()

	for _, agent := range specs.NewCatalog().Registry() {
		spec, err := specs.NewCatalog().SpecOf(agent)
		if err != nil {
			t.Fatalf("SpecOf(%q): %v", agent.ID(), err)
		}
		if strings.Contains(spec.NPMVersion(), "@latest") || spec.NPMVersion() == "latest" {
			t.Errorf("agent %q: NPMVersion() = %q uses @latest", agent.ID(), spec.NPMVersion())
		}
		for _, arg := range spec.FixedArgs {
			if strings.Contains(arg, "@latest") {
				t.Errorf("agent %q: FixedArgs contains @latest: %v", agent.ID(), spec.FixedArgs)
			}
		}
		for _, fb := range spec.Fallbacks {
			for _, arg := range fb.FixedArgs {
				if strings.Contains(arg, "@latest") {
					t.Errorf("agent %q: fallback FixedArgs contains @latest: %v", agent.ID(), fb.FixedArgs)
				}
			}
		}
	}
}
