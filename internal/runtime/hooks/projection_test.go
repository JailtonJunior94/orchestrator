package hooks_test

import (
	"testing"

	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/hooks"
)

func TestCanonicalEventFor_MappedPointsReturnValidEvents(t *testing.T) {
	mapped := []string{
		hooks.PointRuntimePreOpen,
		hooks.PointToolCallPreDispatch,
		hooks.PointToolCallPostComplete,
		hooks.PointSessionPostEnd,
	}
	for _, point := range mapped {
		event, ok := hooks.CanonicalEventFor(point)
		if !ok {
			t.Errorf("CanonicalEventFor(%q) ok = false, want true", point)
		}
		if !event.Valid() {
			t.Errorf("CanonicalEventFor(%q) = %v, want a valid hookcontract.EventKind", point, event)
		}
	}
}

func TestCanonicalEventFor_UnmappedPointReturnsFalse(t *testing.T) {
	if _, ok := hooks.CanonicalEventFor(hooks.PointPromptPreBuild); ok {
		t.Fatal("CanonicalEventFor(PointPromptPreBuild) ok = true, want false")
	}
	if _, ok := hooks.CanonicalEventFor("unknown.point"); ok {
		t.Fatal("CanonicalEventFor(unknown.point) ok = true, want false")
	}
}

func TestPointsWithoutCanonicalEvent_ContainsHarnessOnlyPoints(t *testing.T) {
	without := hooks.PointsWithoutCanonicalEvent()
	want := map[string]bool{
		hooks.PointPromptPreBuild:              true,
		hooks.PointPromptPostBuild:             true,
		hooks.PointSessionPostReview:           true,
		hooks.PointMemoryFactRecorded:          true,
		hooks.PointMemoryFactArchived:          true,
		hooks.PointMemoryFactPromoted:          true,
		hooks.PointMemoryContradictionDetected: true,
		hooks.PointMemorySecretRedacted:        true,
		hooks.PointMemoryCompactionExecuted:    true,
		hooks.PointMemoryBatonTransferred:      true,
	}
	if len(without) != len(want) {
		t.Fatalf("PointsWithoutCanonicalEvent() len = %d, want %d", len(without), len(want))
	}
	for _, point := range without {
		if !want[point] {
			t.Errorf("PointsWithoutCanonicalEvent() has unexpected point %q", point)
		}
	}
}

func TestAllPoints_UnionOfMappedAndUnmapped(t *testing.T) {
	all := hooks.AllPoints()
	if len(all) != 14 {
		t.Fatalf("AllPoints() len = %d, want 14 (the real space of model B — 7 dispatcher + 7 memory points)", len(all))
	}

	seen := make(map[string]bool, len(all))
	for _, point := range all {
		if seen[point] {
			t.Errorf("AllPoints() has duplicate point %q", point)
		}
		seen[point] = true
		_, mapped := hooks.CanonicalEventFor(point)
		unmapped := false
		for _, p := range hooks.PointsWithoutCanonicalEvent() {
			if p == point {
				unmapped = true
				break
			}
		}
		if mapped == unmapped {
			t.Errorf("point %q must be exactly one of mapped or explicitly unmapped, got mapped=%v unmapped=%v", point, mapped, unmapped)
		}
	}
}
