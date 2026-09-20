package specs_test

import (
	"errors"
	"reflect"
	"slices"
	"testing"

	"github.com/JailtonJunior94/ai-spec-harness/internal/hookcontract"
	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/specs"
)

func TestAgentIsImmutableValueObject(t *testing.T) {
	t.Parallel()

	ptr := reflect.TypeOf(&specs.Agent{})
	for i := 0; i < ptr.NumMethod(); i++ {
		m := ptr.Method(i)
		if _, ok := reflect.TypeOf(specs.Agent{}).MethodByName(m.Name); !ok {
			t.Errorf("Agent exposes pointer-receiver method %q — value object must be immutable", m.Name)
		}
	}

	agent, err := specs.NewCatalog().AgentByID("copilot")
	if err != nil {
		t.Fatalf("AgentByID(copilot): %v", err)
	}
	if agent.ID() != "copilot" || agent.SpecID() != "copilot" {
		t.Fatalf("identity mismatch: %q / %q", agent.ID(), agent.SpecID())
	}
	if agent.Identity().DisplayName() == "" || agent.Identity().String() != "copilot" {
		t.Fatal("incomplete identity accessors")
	}
	if !agent.InheritsEnv() {
		t.Fatal("current agents inherit the environment (zero F1 regression)")
	}
	if agent.ADRPath() == "" {
		t.Fatal("agent has no ADR reference")
	}
	if agent.StandardBudget() != 2000 || agent.LargeBudget() != 0 {
		t.Fatalf("unexpected copilot budgets: %d / %d", agent.StandardBudget(), agent.LargeBudget())
	}
	if agent.Signals().Command() != "copilot" || len(agent.Signals().HomeDirs()) != 2 {
		t.Fatalf("unexpected detection signals: %+v", agent.Signals())
	}
}

func TestCanonicalOrderIsSingleSource(t *testing.T) {
	t.Parallel()

	got := specs.NewCatalog().CanonicalOrder()
	want := []string{"claude", "codex", "copilot", "opencode"}
	gotSet := make(map[string]bool, len(got))
	for _, id := range got {
		gotSet[id] = true
	}
	if len(got) != len(want) {
		t.Fatalf("CanonicalOrder() has %d entries; want %d entries (%v)", len(got), len(want), want)
	}
	for _, id := range want {
		if !gotSet[id] {
			t.Fatalf("CanonicalOrder() = %v; missing expected agent %q", got, id)
		}
	}

	catalog := specs.NewCatalog().ACPSpecCatalog()
	if len(catalog) != len(want) {
		t.Fatalf("ACPSpecCatalog has %d entries; registry has %d", len(catalog), len(want))
	}
	for _, id := range want {
		ctor, ok := catalog[id]
		if !ok {
			t.Errorf("ACPSpecCatalog missing %q", id)
			continue
		}
		if ctor().ID != id {
			t.Errorf("ACPSpecCatalog[%q]().ID = %q", id, ctor().ID)
		}
	}
}

func TestRegistryIsImmutable(t *testing.T) {
	t.Parallel()

	first := specs.NewCatalog().Registry()
	first[0] = specs.Agent{}
	second := specs.NewCatalog().Registry()
	if second[0].ID() != "claude" {
		t.Fatalf("mutation of returned slice leaked into registry: %q", second[0].ID())
	}

	a, err := specs.NewCatalog().AgentByID("codex")
	if err != nil {
		t.Fatalf("AgentByID(codex): %v", err)
	}
	b, _ := specs.NewCatalog().AgentByID("codex")
	if !a.Equal(b) {
		t.Fatal("two resolutions of the same agent are not value-equal")
	}
	c, _ := specs.NewCatalog().AgentByID("claude")
	if a.Equal(c) {
		t.Fatal("distinct agents compare equal")
	}
}

func TestAgentZeroValueIsInvalid(t *testing.T) {
	t.Parallel()

	var zero specs.Agent
	if zero.Valid() {
		t.Fatal("zero-value Agent reported valid")
	}
	var zeroID specs.AgentIdentity
	if zeroID.Valid() {
		t.Fatal("zero-value AgentIdentity reported valid")
	}
	if _, err := specs.NewCatalog().NewAgentIdentity("", "x"); err == nil {
		t.Fatal("NewAgentIdentity accepted empty id")
	}
	if _, err := specs.NewCatalog().NewAgentIdentity("x", ""); err == nil {
		t.Fatal("NewAgentIdentity accepted empty displayName")
	}
	if _, err := specs.NewCatalog().AgentByID("nope"); !errors.Is(err, specs.ErrUnknownAgent) {
		t.Fatalf("AgentByID(nope) err = %v; want ErrUnknownAgent", err)
	}
}

func TestMandatoryMatrixAgentsByCanonicalPoints(t *testing.T) {
	t.Parallel()

	points := specs.NewCatalog().CanonicalPoints()
	if len(points) != 5 {
		t.Fatalf("expected 5 canonical points; got %d", len(points))
	}

	scriptByPoint := make(map[specs.CanonicalPoint]string)
	for _, agent := range specs.NewCatalog().Registry() {
		enf := agent.Enforcement()
		if !enf.Valid() {
			t.Fatalf("agent %q: invalid enforcement", agent.ID())
		}
		if len(enf.Coverage()) != len(points) {
			t.Fatalf("agent %q: coverage smaller than matrix (%d/%d)", agent.ID(), len(enf.Coverage()), len(points))
		}
		for _, point := range points {
			cov, ok := enf.CoverageFor(point)
			if !ok {
				t.Fatalf("agent %q: point %s not covered", agent.ID(), point)
			}
			if cov.State() == hookcontract.SupportUnsupported {
				if cov.Reason() == "" {
					t.Errorf("agent %q point %s: unsupported without a declared reason", agent.ID(), point)
				}
				continue
			}
			if cov.NativeKey() == "" {
				t.Errorf("agent %q point %s: empty native key", agent.ID(), point)
			}
			if cov.State() == hookcontract.SupportAdapter && cov.Limitation() == "" {
				t.Errorf("agent %q point %s: adapter support without a declared limitation", agent.ID(), point)
			}
			if (cov.ScriptPath() == "") != (cov.ArtifactPath() == "") {
				t.Errorf("agent %q point %s: script and artifact must both be declared or both be empty", agent.ID(), point)
			}
			recognized, known := specs.RecognizedNativeKeys(agent.ID(), point)
			if !known {
				t.Errorf("agent %q point %s: no declared hook vocabulary", agent.ID(), point)
			} else if !slices.Contains(recognized, cov.NativeKey()) {
				t.Errorf("agent %q point %s: native key %q is not recognized by the CLI (%v)", agent.ID(), point, cov.NativeKey(), recognized)
			}
			if cov.ScriptPath() == "" {
				continue
			}
			if prev, seen := scriptByPoint[point]; seen {
				if prev != cov.ScriptPath() {
					t.Errorf("point %s: canonical script diverges between agents (%q vs %q)", point, prev, cov.ScriptPath())
				}
			} else {
				scriptByPoint[point] = cov.ScriptPath()
			}
		}
	}
}

func TestRegistryPreconditionsCarryRemedy(t *testing.T) {
	t.Parallel()

	for _, agent := range specs.NewCatalog().Registry() {
		for _, pre := range agent.Enforcement().Preconditions() {
			if !pre.Valid() {
				t.Errorf("agent %q: precondition not built via constructor", agent.ID())
			}
			if pre.Remedy() == "" {
				t.Errorf("agent %q: precondition %d has no actionable remedy", agent.ID(), pre.Kind())
			}
		}
	}
}

func TestTwentyProviderEventPairsHaveExplicitState(t *testing.T) {
	t.Parallel()

	want := map[string]map[specs.CanonicalPoint]hookcontract.SupportState{
		"claude": {
			specs.PointSessionStart:   hookcontract.SupportVerified,
			specs.PointPreTool:        hookcontract.SupportVerified,
			specs.PointPostTool:       hookcontract.SupportVerified,
			specs.PointBeforeComplete: hookcontract.SupportVerified,
			specs.PointSessionEnd:     hookcontract.SupportVerified,
		},
		"codex": {
			specs.PointSessionStart:   hookcontract.SupportVerified,
			specs.PointPreTool:        hookcontract.SupportVerified,
			specs.PointPostTool:       hookcontract.SupportVerified,
			specs.PointBeforeComplete: hookcontract.SupportVerified,
			specs.PointSessionEnd:     hookcontract.SupportVerified,
		},
		"copilot": {
			specs.PointSessionStart:   hookcontract.SupportVerified,
			specs.PointPreTool:        hookcontract.SupportVerified,
			specs.PointPostTool:       hookcontract.SupportVerified,
			specs.PointBeforeComplete: hookcontract.SupportVerified,
			specs.PointSessionEnd:     hookcontract.SupportVerified,
		},
		"opencode": {
			specs.PointSessionStart:   hookcontract.SupportAdapter,
			specs.PointPreTool:        hookcontract.SupportVerified,
			specs.PointPostTool:       hookcontract.SupportVerified,
			specs.PointBeforeComplete: hookcontract.SupportAdapter,
			specs.PointSessionEnd:     hookcontract.SupportUnsupported,
		},
	}

	points := specs.NewCatalog().CanonicalPoints()
	pairCount := 0
	for _, agent := range specs.NewCatalog().Registry() {
		byPoint, ok := want[agent.ID()]
		if !ok {
			t.Fatalf("agent %q has no expected state table entry — every provider must be declared", agent.ID())
		}
		for _, point := range points {
			pairCount++
			cov, ok := agent.Enforcement().CoverageFor(point)
			if !ok {
				t.Fatalf("agent %q point %s: not declared — absence of support is never omission (P07)", agent.ID(), point)
			}
			if cov.State() != byPoint[point] {
				t.Errorf("agent %q point %s: state = %s, want %s", agent.ID(), point, cov.State(), byPoint[point])
			}
			switch cov.State() {
			case hookcontract.SupportAdapter, hookcontract.SupportUnsupported:
				if cov.Reason() == "" {
					t.Errorf("agent %q point %s: state %s declared without a textual reason/limitation", agent.ID(), point, cov.State())
				}
			}
		}
	}
	if pairCount != 20 {
		t.Fatalf("expected exactly 20 (provider, event) pairs; got %d", pairCount)
	}
}

func TestACPSpecCatalogMatchesRegistry(t *testing.T) {
	t.Parallel()

	catalog := specs.NewCatalog().ACPSpecCatalog()
	ids := make([]string, 0, len(catalog))
	for id := range catalog {
		ids = append(ids, id)
	}
	slices.Sort(ids)

	if err := specs.NewCatalog().VerifyCatalogSync(ids); err != nil {
		t.Fatalf("ACPSpecCatalog out of sync with registry: %v", err)
	}
	for id, ctor := range catalog {
		if ctor().ID != id {
			t.Errorf("ACPSpecCatalog[%q]().ID = %q", id, ctor().ID)
		}
	}

	diverging := ids[:len(ids)-1]
	if err := specs.NewCatalog().VerifyCatalogSync(diverging); !errors.Is(err, specs.ErrCatalogOutOfSync) {
		t.Fatalf("gate did not catch artificial divergence: %v", err)
	}
	if err := specs.NewCatalog().VerifyCatalogSync(append(slices.Clone(ids), "gemini")); !errors.Is(err, specs.ErrCatalogOutOfSync) {
		t.Fatalf("gate did not catch a tool absent from the registry: %v", err)
	}
}
