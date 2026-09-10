package specs_test

import (
	"errors"
	"reflect"
	"slices"
	"testing"

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
	want := []string{"claude", "gemini", "codex", "copilot"}
	if !slices.Equal(got, want) {
		t.Fatalf("CanonicalOrder() = %v; want %v", got, want)
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
	if len(points) != 3 {
		t.Fatalf("expected 3 canonical points; got %d", len(points))
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
			if cov.NativeKey() == "" {
				t.Errorf("agent %q point %s: empty native key", agent.ID(), point)
			}
			if cov.ScriptPath() == "" {
				t.Errorf("agent %q point %s: empty canonical script", agent.ID(), point)
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
