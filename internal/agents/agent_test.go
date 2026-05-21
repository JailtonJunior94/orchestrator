package agents_test

import (
	"testing"

	"github.com/JailtonJunior94/ai-spec-harness/internal/agents"
)

func TestScopeString(t *testing.T) {
	t.Parallel()

	tests := []struct {
		scope agents.Scope
		want  string
	}{
		{agents.ScopeGlobal, "global"},
		{agents.ScopeWorkspace, "workspace"},
	}

	for _, tc := range tests {
		t.Run(tc.want, func(t *testing.T) {
			t.Parallel()
			got := tc.scope.String()
			if got != tc.want {
				t.Errorf("Scope.String() = %q, quer %q", got, tc.want)
			}
		})
	}
}

func TestResolvedAgentZeroValue(t *testing.T) {
	t.Parallel()
	var a agents.ResolvedAgent
	if a.Name != "" {
		t.Error("zero value de ResolvedAgent.Name deve ser vazio")
	}
	if a.Scope != agents.ScopeGlobal {
		t.Error("zero value de ResolvedAgent.Scope deve ser ScopeGlobal")
	}
}
