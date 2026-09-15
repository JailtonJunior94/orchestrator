package approval

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

const unifiedDiffFixture = "diff --git a/internal/a.go b/internal/a.go\n" +
	"index 1111111..2222222 100644\n" +
	"--- a/internal/a.go\n" +
	"+++ b/internal/a.go\n" +
	"@@ -1,5 +1,6 @@\n" +
	" package internal\n" +
	"-func Old() {}\n" +
	"+func New() {}\n" +
	"@@ -40,3 +41,4 @@\n" +
	"+var added = 1\n" +
	"diff --git a/internal/b.go b/internal/b.go\n" +
	"--- a/internal/b.go\n" +
	"+++ b/internal/b.go\n" +
	"@@ -10,2 +10,2 @@\n" +
	"-x\n" +
	"+y\n"

func parseWithDiff(t *testing.T, evidence string) CriteriaMap {
	t.Helper()
	raw := "## Mapa de Critérios de Aceite\n- [atendido] Criterio um -> " + evidence + "\n"
	criteriaMap, err := ParseCriteriaMap(raw, criteriaRequest(t, unifiedDiffFixture, "Criterio um"))
	require.NoError(t, err)
	return criteriaMap
}

func TestFileLineEvidenceIsResolvedAgainstARealUnifiedDiff(t *testing.T) {
	accepted := []string{
		"internal/a.go:4",
		"internal/a.go:1",
		"internal/a.go:42",
		"a.go:4",
		"internal/b.go:11",
	}

	for _, reference := range accepted {
		t.Run("aceita "+reference, func(t *testing.T) {
			criteriaMap := parseWithDiff(t, reference)
			require.True(t, criteriaMap.Complete())
			require.Empty(t, criteriaMap.FindingsList())
		})
	}

	rejected := []string{
		"internal/a.go:300",
		"internal/c.go:4",
		"internal/b.go:99",
	}

	for _, reference := range rejected {
		t.Run("rejeita "+reference, func(t *testing.T) {
			criteriaMap := parseWithDiff(t, reference)
			require.False(t, criteriaMap.Complete())
			require.Len(t, criteriaMap.FindingsList(), 1)
		})
	}
}

func TestCommandEvidenceRequiresArgumentsAndSubstantiveOutput(t *testing.T) {
	cases := []struct {
		name     string
		evidence string
		accepted bool
	}{
		{"prosa disfarcada de make", "make it work -> done", false},
		{"git com saida trivial", "git log -> abc", false},
		{"sh com saida trivial", "sh -> ok", false},
		{"cat com saida trivial", "cat x -> talvez", false},
		{"binario sem argumento", "make -> pass", false},
		{"comando real com saida", "go test ./internal/approval/... -> ok 0.4s", true},
		{"comando real com exit", "make test -> exit 0", true},
		{"comando real com resultado canonico", "go build ./... -> PASS", true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			criteriaMap := parseWithDiff(t, tc.evidence)
			require.Equal(t, tc.accepted, criteriaMap.Complete())
			if tc.accepted {
				require.Empty(t, criteriaMap.FindingsList())
				return
			}
			require.Len(t, criteriaMap.FindingsList(), 1)
			require.True(t, criteriaMap.FindingsList()[0].Severity().Blocks())
		})
	}
}

type replayReviewer struct {
	raw      string
	captured []ReviewRequest
}

func (r *replayReviewer) Review(_ context.Context, request ReviewRequest) (ReviewerOutput, error) {
	r.captured = append(r.captured, request)
	criteriaMap, err := ParseCriteriaMap(r.raw, request)
	if err != nil {
		return ReviewerOutput{}, err
	}
	return NewReviewerOutput(r.raw, nil, criteriaMap), nil
}

func TestCycleTurnsMalformedEvidenceIntoAnAuditableStopInsteadOfAnError(t *testing.T) {
	cases := []struct {
		name string
		raw  string
	}{
		{"mapa incompleto", "Veredito: APPROVED\n\n## Mapa de Critérios de Aceite\n- [atendido] Criterio um -> go test ./... -> ok 0.2s\n"},
		{"prosa no lugar da evidencia", "Veredito: APPROVED\n\n## Mapa de Critérios de Aceite\n- [atendido] Criterio um -> confio no autor\n- [atendido] Criterio dois -> go test ./... -> ok 0.2s\n"},
		{"referencia fora do diff", "Veredito: APPROVED\n\n## Mapa de Critérios de Aceite\n- [atendido] Criterio um -> internal/ausente.go:9\n- [atendido] Criterio dois -> go test ./... -> ok 0.2s\n"},
		{"teste sem resultado", "Veredito: APPROVED\n\n## Mapa de Critérios de Aceite\n- [atendido] Criterio um -> TestAlgo -> parece bom\n- [atendido] Criterio dois -> go test ./... -> ok 0.2s\n"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			task, err := NewTaskIdentity("task-1.0")
			require.NoError(t, err)
			agent, err := NewAgentIdentity("claude")
			require.NoError(t, err)
			policy, err := NewApprovalPolicy(WithMaxRounds(2))
			require.NoError(t, err)

			criteria := []AcceptanceCriterion{
				mustCriterion(t, "Criterio um"),
				mustCriterion(t, "Criterio dois"),
			}
			reviewer := &replayReviewer{raw: tc.raw}
			cycle, err := NewCycle(
				task, agent, policy, criteria, reviewer, stubFixer{},
				stubRepository{full: NewReviewTarget(unifiedDiffFixture), delta: NewReviewTarget("delta")},
			)
			require.NoError(t, err)

			result, err := cycle.Run(t.Context())
			require.NoError(t, err)
			require.False(t, result.Approved())
			require.NotEqual(t, StopReason(0), result.Reason())
			require.GreaterOrEqual(t, len(reviewer.captured), 2)
		})
	}
}
