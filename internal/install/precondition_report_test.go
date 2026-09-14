package install

import (
	"testing"

	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/precondition"
	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/specs"
	"github.com/JailtonJunior94/ai-spec-harness/internal/skills"
)

func codexTrustedHashPrecondition(t *testing.T) specs.EnforcementPrecondition {
	t.Helper()
	pre, err := specs.NewCatalog().NewEnforcementPrecondition(specs.PreconditionTrustedHash, "grant via TUI", true)
	if err != nil {
		t.Fatalf("NewEnforcementPrecondition: %v", err)
	}
	return pre
}

func TestPreconditionItemsReportsEachCanonicalPointSeparately(t *testing.T) {
	t.Parallel()

	report := precondition.Report{
		Kind:   specs.PreconditionTrustedHash,
		State:  specs.PreconditionInert,
		Remedy: "grant via TUI",
		Points: []precondition.CodexTrustPoint{
			{EventName: "PreToolUse", State: specs.PreconditionCurrent},
			{EventName: "PostToolUse", State: specs.PreconditionInert},
			{EventName: "Stop", State: specs.PreconditionInert},
		},
	}

	items := preconditionItems(skills.Tool("codex"), codexTrustedHashPrecondition(t), report)
	if len(items) != 3 {
		t.Fatalf("items = %d; a precondition covering three canonical points must be reported point by point", len(items))
	}

	want := map[string]VerifyState{
		"precondition(trusted-hash:PreToolUse)":  VerifyStateCurrent,
		"precondition(trusted-hash:PostToolUse)": VerifyStateInert,
		"precondition(trusted-hash:Stop)":        VerifyStateInert,
	}
	for _, item := range items {
		state, ok := want[item.Skill]
		if !ok {
			t.Fatalf("unexpected item %q", item.Skill)
		}
		if item.State != state {
			t.Fatalf("item %q state = %v; want %v", item.Skill, item.State, state)
		}
		if item.Kind != VerifyKindPrecondition {
			t.Fatalf("item %q kind = %v; want precondition", item.Skill, item.Kind)
		}
	}
}

func TestEnforcementNativeKeysCoversEveryCodexCanonicalPoint(t *testing.T) {
	t.Parallel()

	agent, err := specs.NewCatalog().AgentByID("codex")
	if err != nil {
		t.Fatalf("AgentByID: %v", err)
	}
	keys := enforcementNativeKeys(agent.Enforcement())
	want := []string{"PreToolUse", "PostToolUse", "Stop"}
	if len(keys) != len(want) {
		t.Fatalf("keys = %v; want %v", keys, want)
	}
	for i, key := range want {
		if keys[i] != key {
			t.Fatalf("keys[%d] = %q; want %q", i, keys[i], key)
		}
	}
}
