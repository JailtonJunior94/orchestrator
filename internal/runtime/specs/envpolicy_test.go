package specs_test

import (
	"os"
	"testing"

	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/specs"
)

func TestEnvPolicyZeroValueAppliesNil(t *testing.T) {
	t.Parallel()

	var pol specs.EnvPolicy
	if !pol.IsZero() {
		t.Fatal("zero-value EnvPolicy must report IsZero() == true")
	}
	if got := pol.Apply([]string{"FOO=bar"}); got != nil {
		t.Fatalf("zero-value EnvPolicy.Apply() = %v; want nil (inherit intact)", got)
	}
}

func TestEnvPolicyStripsKnownKillSwitches(t *testing.T) {
	t.Parallel()

	pol := specs.NewCatalog().NewEnvPolicy(specs.OpenCodeKillSwitchVars...)
	if pol.IsZero() {
		t.Fatal("non-empty EnvPolicy must not report IsZero()")
	}

	environ := []string{
		"PATH=/usr/bin",
		"OPENCODE_PURE=1",
		"OPENCODE_DISABLE_PROJECT_CONFIG=1",
		"OPENCODE_DISABLE_EXTERNAL_SKILLS=1",
		"HOME=/home/user",
	}
	out := pol.Apply(environ)
	for _, killSwitch := range specs.OpenCodeKillSwitchVars {
		for _, kv := range out {
			if len(kv) >= len(killSwitch) && kv[:len(killSwitch)] == killSwitch {
				t.Fatalf("Apply() did not strip %q: %v", killSwitch, out)
			}
		}
	}
	if len(out) != 2 {
		t.Fatalf("Apply() = %v; want only PATH and HOME to survive", out)
	}
}

func TestEnvPolicyApplyNeverMutatesInputSlice(t *testing.T) {
	t.Parallel()

	pol := specs.NewCatalog().NewEnvPolicy("OPENCODE_PURE")
	environ := []string{"OPENCODE_PURE=1", "PATH=/usr/bin"}
	snapshot := append([]string{}, environ...)

	_ = pol.Apply(environ)

	for i := range environ {
		if environ[i] != snapshot[i] {
			t.Fatalf("Apply() mutated the caller's environ slice: %v", environ)
		}
	}
}

func TestEnvPolicyStripVarsReturnsClone(t *testing.T) {
	t.Parallel()

	pol := specs.NewCatalog().NewEnvPolicy("A", "B")
	vars := pol.StripVars()
	vars[0] = "MUTATED"
	if pol.StripVars()[0] == "MUTATED" {
		t.Fatal("StripVars() must return a defensive clone")
	}
}

func TestOpenCodeAgentEnvPolicyStripsExactlyTheThreeKillSwitches(t *testing.T) {
	t.Parallel()

	agent, err := specs.NewCatalog().AgentByID("opencode")
	if err != nil {
		t.Fatalf("AgentByID(opencode): %v", err)
	}
	if agent.InheritsEnv() {
		t.Fatal("OpenCode agent must not report InheritsEnv() == true (RF-21)")
	}
	pol := agent.EnvPolicy()
	if pol.IsZero() {
		t.Fatal("OpenCode agent must have a non-zero EnvPolicy")
	}
	want := []string{"OPENCODE_PURE", "OPENCODE_DISABLE_PROJECT_CONFIG", "OPENCODE_DISABLE_EXTERNAL_SKILLS"}
	got := pol.StripVars()
	if len(got) != len(want) {
		t.Fatalf("StripVars() = %v; want %v", got, want)
	}
	for _, w := range want {
		found := false
		for _, g := range got {
			if g == w {
				found = true
			}
		}
		if !found {
			t.Errorf("StripVars() missing %q", w)
		}
	}
}

func TestOpenCodeAgentRequiresHandshake(t *testing.T) {
	t.Parallel()

	agent, err := specs.NewCatalog().AgentByID("opencode")
	if err != nil {
		t.Fatalf("AgentByID(opencode): %v", err)
	}
	if !agent.RequiresHandshake() {
		t.Fatal("OpenCode agent must require the governance plugin handshake (RF-21)")
	}
}

func TestOtherAgentsDoNotRequireHandshakeAndInheritEnvByDefault(t *testing.T) {
	t.Parallel()

	for _, id := range []string{"claude", "codex", "copilot"} {
		agent, err := specs.NewCatalog().AgentByID(id)
		if err != nil {
			t.Fatalf("AgentByID(%s): %v", id, err)
		}
		if agent.RequiresHandshake() {
			t.Errorf("agent %q must not require handshake (regression risk, O-06)", id)
		}
		if !agent.InheritsEnv() {
			t.Errorf("agent %q must inherit env intact by default (zero-value EnvPolicy, O-06)", id)
		}
		if !agent.EnvPolicy().IsZero() {
			t.Errorf("agent %q must have zero-value EnvPolicy (O-06)", id)
		}
	}
}

func TestEnvPolicyAppliedToRealEnvironProducesSameLengthMinusStripped(t *testing.T) {
	t.Parallel()

	pol := specs.NewCatalog().NewEnvPolicy(specs.OpenCodeKillSwitchVars...)
	before := os.Environ()
	out := pol.Apply(before)
	if len(out) > len(before) {
		t.Fatalf("Apply() grew the environment: before=%d after=%d", len(before), len(out))
	}
}
