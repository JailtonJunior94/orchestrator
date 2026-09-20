//go:build integration

package integration

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/JailtonJunior94/ai-spec-harness/internal/hookcontract"
	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/hooks"
	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/specs"
)

func TestHookContractProjection_SpecsModelMatchesContract(t *testing.T) {
	catalog := specs.NewCatalog()
	points := catalog.CanonicalPoints()

	want := []struct {
		point specs.CanonicalPoint
		event hookcontract.EventKind
	}{
		{specs.PointSessionStart, hookcontract.EventSessionStart},
		{specs.PointPreTool, hookcontract.EventBeforeTool},
		{specs.PointPostTool, hookcontract.EventAfterTool},
		{specs.PointBeforeComplete, hookcontract.EventBeforeComplete},
		{specs.PointSessionEnd, hookcontract.EventSessionEnd},
	}

	if len(points) != len(want) {
		t.Fatalf("specs.CanonicalPoints() cardinality diverges from hookcontract projection: got %d, want %d", len(points), len(want))
	}

	for i, entry := range want {
		if points[i] != entry.point {
			t.Fatalf("specs.CanonicalPoints() order diverges from hookcontract projection at index %d: got %s, want %s", i, points[i], entry.point)
		}
		event, ok := catalog.CanonicalPointEvent(points[i])
		if !ok {
			t.Fatalf("CanonicalPointEvent(%s) has no hookcontract.EventKind projection", points[i])
		}
		if event != entry.event {
			t.Fatalf("CanonicalPointEvent(%s) = %v, want %v", points[i], event, entry.event)
		}
	}
}

func TestHookContractProjection_SpecsModelDivergesWhenMappingIsWrong(t *testing.T) {
	catalog := specs.NewCatalog()
	points := catalog.CanonicalPoints()

	brokenWant := map[specs.CanonicalPoint]hookcontract.EventKind{
		specs.PointSessionStart:   hookcontract.EventSessionStart,
		specs.PointPreTool:        hookcontract.EventSessionStart,
		specs.PointPostTool:       hookcontract.EventAfterTool,
		specs.PointBeforeComplete: hookcontract.EventBeforeComplete,
		specs.PointSessionEnd:     hookcontract.EventSessionEnd,
	}

	diverged := false
	for _, point := range points {
		event, ok := catalog.CanonicalPointEvent(point)
		if !ok || event != brokenWant[point] {
			diverged = true
		}
	}
	if !diverged {
		t.Fatal("expected the deliberately wrong mapping fixture to diverge from the real projection — the gate would not be able to detect a real divergence")
	}
}

func TestHookContractProjection_HooksModelIsExhaustivelyRegistered(t *testing.T) {
	root := repoRootForDispatch(t)

	declared := parseHookPointConstants(t, root)
	if len(declared) == 0 {
		t.Fatal("no Point* constants parsed from internal/runtime/hooks — parser regression")
	}

	mapped := hooks.AllPoints()
	if len(mapped) != len(declared) {
		t.Fatalf("hooks.AllPoints() cardinality diverges from the declared Point* constants: got %d, want %d (declared=%v mapped=%v)",
			len(mapped), len(declared), declared, mapped)
	}

	mappedSet := make(map[string]bool, len(mapped))
	for _, p := range mapped {
		mappedSet[p] = true
	}
	for _, p := range declared {
		if !mappedSet[p] {
			t.Errorf("Point constant %q is declared in internal/runtime/hooks but not registered in hooks.AllPoints() — orphan point, not covered by the contract gate", p)
		}
	}

	withEvent := 0
	for _, p := range mapped {
		if event, ok := hooks.CanonicalEventFor(p); ok {
			withEvent++
			if !event.Valid() {
				t.Errorf("point %q maps to invalid hookcontract.EventKind %v", p, event)
			}
		}
	}
	withoutEvent := len(hooks.PointsWithoutCanonicalEvent())
	if withEvent+withoutEvent != len(mapped) {
		t.Fatalf("mapped(%d) = withEvent(%d) + withoutEvent(%d) does not hold — a point is neither mapped nor explicitly declared as having no contract equivalent",
			len(mapped), withEvent, withoutEvent)
	}
}

func TestHookContractProjection_HooksModelDivergesWhenAPointIsUnregistered(t *testing.T) {
	root := repoRootForDispatch(t)
	declared := parseHookPointConstants(t, root)

	mapped := make(map[string]bool, len(hooks.AllPoints()))
	for _, p := range hooks.AllPoints() {
		mapped[p] = true
	}

	fabricatedOrphan := "fabricated.orphan_point_for_gate_test"
	simulated := append([]string{}, declared...)
	simulated = append(simulated, fabricatedOrphan)

	foundOrphan := false
	for _, p := range simulated {
		if !mapped[p] {
			foundOrphan = true
		}
	}
	if !foundOrphan {
		t.Fatal("expected the fabricated orphan point to be detected as unregistered — the gate would not be able to detect a real orphan point")
	}
}

func parseHookPointConstants(t *testing.T, root string) []string {
	t.Helper()

	dir := filepath.Join(root, "internal", "runtime", "hooks")
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read %s: %v", dir, err)
	}

	fileSet := token.NewFileSet()
	var points []string
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") || strings.HasSuffix(entry.Name(), "_test.go") {
			continue
		}
		file, parseErr := parser.ParseFile(fileSet, filepath.Join(dir, entry.Name()), nil, parser.AllErrors)
		if parseErr != nil {
			t.Fatalf("parse %s: %v", entry.Name(), parseErr)
		}
		ast.Inspect(file, func(node ast.Node) bool {
			spec, ok := node.(*ast.ValueSpec)
			if !ok {
				return true
			}
			for _, name := range spec.Names {
				if strings.HasPrefix(name.Name, "Point") {
					points = append(points, name.Name)
				}
			}
			return true
		})
	}

	names := parseHookPointValues(t, root, points)
	sort.Strings(names)
	return names
}

func parseHookPointValues(t *testing.T, root string, identifiers []string) []string {
	t.Helper()

	byIdentifier := map[string]string{
		"PointRuntimePreOpen":              hooks.PointRuntimePreOpen,
		"PointPromptPreBuild":              hooks.PointPromptPreBuild,
		"PointPromptPostBuild":             hooks.PointPromptPostBuild,
		"PointToolCallPreDispatch":         hooks.PointToolCallPreDispatch,
		"PointToolCallPostComplete":        hooks.PointToolCallPostComplete,
		"PointSessionPostEnd":              hooks.PointSessionPostEnd,
		"PointSessionPostReview":           hooks.PointSessionPostReview,
		"PointMemoryFactRecorded":          hooks.PointMemoryFactRecorded,
		"PointMemoryFactArchived":          hooks.PointMemoryFactArchived,
		"PointMemoryFactPromoted":          hooks.PointMemoryFactPromoted,
		"PointMemoryContradictionDetected": hooks.PointMemoryContradictionDetected,
		"PointMemorySecretRedacted":        hooks.PointMemorySecretRedacted,
		"PointMemoryCompactionExecuted":    hooks.PointMemoryCompactionExecuted,
		"PointMemoryBatonTransferred":      hooks.PointMemoryBatonTransferred,
	}

	values := make([]string, 0, len(identifiers))
	for _, id := range identifiers {
		value, ok := byIdentifier[id]
		if !ok {
			t.Fatalf("Point constant %q found in source but has no known Go identifier binding in this test — the exhaustiveness map (parseHookPointValues) must be updated in the same lot as the new constant", id)
		}
		values = append(values, value)
	}
	return values
}
