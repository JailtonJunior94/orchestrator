package taskloop

import (
	"errors"
	"sort"
	"testing"

	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/specs"
)

func TestACPSpecCatalogInSyncWithRegistry(t *testing.T) {
	t.Parallel()

	ids := make([]string, 0, len(acpSpecCatalog))
	for k := range acpSpecCatalog {
		ids = append(ids, k)
	}
	sort.Strings(ids)

	if err := specs.NewCatalog().VerifyCatalogSync(ids); err != nil {
		t.Fatalf("acpSpecCatalog out of sync with registry: %v", err)
	}

	diverging := append([]string{}, ids...)
	diverging = diverging[:len(diverging)-1]
	if err := specs.NewCatalog().VerifyCatalogSync(diverging); !errors.Is(err, specs.ErrCatalogOutOfSync) {
		t.Fatalf("gate did not catch artificial divergence: %v", err)
	}

	for id, ctor := range acpSpecCatalog {
		if ctor().ID != id {
			t.Errorf("acpSpecCatalog[%q]().ID = %q", id, ctor().ID)
		}
	}
}
