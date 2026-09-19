package parity

import (
	"testing"

	"github.com/JailtonJunior94/ai-spec-harness/internal/skills"
)

func TestSelfSatisfiedInvariantsMatchInjectedStubs(t *testing.T) {
	checker := NewChecker()
	toolSet := map[skills.Tool]bool{
		skills.ToolClaude:   true,
		skills.ToolCodex:    true,
		skills.ToolCopilot:  true,
		skills.ToolOpenCode: true,
	}

	injected := checker.selfSatisfiedInvariantStubPaths(toolSet)
	injectedSet := make(map[string]bool, len(injected))
	for _, p := range injected {
		injectedSet[p] = true
	}

	marked := make(map[string]bool, len(injected))
	for _, inv := range checker.Invariants() {
		for _, p := range inv.SelfSatisfiedStubPaths {
			if !injectedSet[p] {
				t.Fatalf("invariant %s marks path %q as self-satisfied but Generate does not inject it", inv.ID, p)
			}
			marked[p] = true
		}
	}

	for _, p := range injected {
		if !marked[p] {
			t.Fatalf("stub path %q injected by Generate has no invariant marked as self-satisfied evidence (V-25 sanitization gap)", p)
		}
	}
}

func TestSelfSatisfiedInvariantIDs(t *testing.T) {
	checker := NewChecker()
	want := map[string]bool{
		"CL03": true,
		"CL04": true,
		"CL05": true,
		"CL06": true,
		"CL07": true,
		"CL08": true,
		"X03":  true,
	}

	got := make(map[string]bool)
	for _, inv := range checker.Invariants() {
		if inv.EvidenceInvalid() {
			got[inv.ID] = true
		}
	}

	for id := range want {
		if !got[id] {
			t.Errorf("invariant %s expected to be marked evidence-invalid (V-25) but is not", id)
		}
	}
	for id := range got {
		if !want[id] {
			t.Errorf("invariant %s unexpectedly marked evidence-invalid; update the expected set if this is deliberate", id)
		}
	}
}

func TestScopeDerivesFromAppliesToNotLevel(t *testing.T) {
	checker := NewChecker()
	providerSpecificDespiteCommonLevel := map[string]bool{
		"CL01": true,
		"CL02": true,
		"CP01": true,
		"CD01": true,
		"CD02": true,
	}

	for _, inv := range checker.Invariants() {
		if !providerSpecificDespiteCommonLevel[inv.ID] {
			continue
		}
		if inv.Level != Common {
			t.Fatalf("invariant %s expected Level=Common for this regression to be meaningful, got %s", inv.ID, inv.Level)
		}
		if inv.Scope() != ScopeProviderSpecific {
			t.Errorf("invariant %s has Level=Common but single-tool AppliesTo; Scope() must be ScopeProviderSpecific (V-26), got %s", inv.ID, inv.Scope())
		}
	}
}

func TestScopeUniversalWhenAppliesToNilOrCoversAllProviders(t *testing.T) {
	inv := &Invariant{ID: "TEST-NIL", AppliesTo: nil}
	if inv.Scope() != ScopeUniversal {
		t.Errorf("nil AppliesTo must be ScopeUniversal, got %s", inv.Scope())
	}

	invAll := &Invariant{ID: "TEST-ALL", AppliesTo: CanonicalProviders()}
	if invAll.Scope() != ScopeUniversal {
		t.Errorf("AppliesTo covering all canonical providers must be ScopeUniversal, got %s", invAll.Scope())
	}

	invPartial := &Invariant{ID: "TEST-PARTIAL", AppliesTo: []skills.Tool{skills.ToolClaude, skills.ToolCodex}}
	if invPartial.Scope() != ScopeProviderSpecific {
		t.Errorf("AppliesTo covering a strict subset must be ScopeProviderSpecific, got %s", invPartial.Scope())
	}
}
