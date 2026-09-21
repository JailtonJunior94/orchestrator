package conformance_test

import (
	"testing"

	"github.com/JailtonJunior94/ai-spec-harness/internal/conformance"
)

func TestRF61Scenarios_HasFourteenEntriesInPRDOrder(t *testing.T) {
	scenarios := conformance.RF61Scenarios()
	if len(scenarios) != 14 {
		t.Fatalf("RF-61 enumerates fourteen scenarios in prd.md:364-368; got %d", len(scenarios))
	}
	for i, s := range scenarios {
		wantIndex := i + 1
		if s.Index != wantIndex {
			t.Fatalf("scenario at position %d has Index=%d, want %d — order must match prd.md:364-368", i, s.Index, wantIndex)
		}
		if s.Name == "" {
			t.Fatalf("scenario #%d has an empty name", s.Index)
		}
	}
}

func TestRF61Scenarios_AreADistinctCatalogFromTheManifest(t *testing.T) {
	manifestNames := make(map[string]bool)
	for _, s := range conformance.Manifest() {
		manifestNames[s.Name] = true
	}
	for _, s := range conformance.RF61Scenarios() {
		if manifestNames[s.Name] {
			t.Fatalf(
				"RF-61 scenario %q collides with a Manifest() catalog name; the two enumerations serve different purposes (Manifest is the User Story session-level product catalog used only to classify determinism; RF61Scenarios is the command-level conformance suite enumeration) and must stay explicitly distinct rather than accidentally identical",
				s.Name,
			)
		}
	}
}
