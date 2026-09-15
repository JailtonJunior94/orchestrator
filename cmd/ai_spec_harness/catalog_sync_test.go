package aispecharness

import (
	"errors"
	"sort"
	"testing"

	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/specs"
)

func acpCatalogKeys(m map[string]func() specs.Spec) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func TestRuntimeACPCatalogInSyncWithRegistry(t *testing.T) {
	t.Parallel()

	if err := specs.NewCatalog().VerifyCatalogSync(acpCatalogKeys(runtimeACPCatalog)); err != nil {
		t.Fatalf("CLI ACP catalog out of sync: %v", err)
	}

	diverging := make(map[string]func() specs.Spec, len(runtimeACPCatalog))
	for k, v := range runtimeACPCatalog {
		diverging[k] = v
	}
	delete(diverging, "opencode")
	if err := specs.NewCatalog().VerifyCatalogSync(acpCatalogKeys(diverging)); !errors.Is(err, specs.ErrCatalogOutOfSync) {
		t.Fatalf("sync gate did not catch artificial divergence: %v", err)
	}
}
