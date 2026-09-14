package approval

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func reviewRequestForTest(t *testing.T) ReviewRequest {
	t.Helper()
	task, err := NewTaskIdentity("task-2.0")
	require.NoError(t, err)
	agent, err := NewAgentIdentity("claude")
	require.NoError(t, err)
	request, err := NewReviewRequest(task, agent, 1, ReviewTarget{}, []AcceptanceCriterion{mustCriterion(t, "criterion one")})
	require.NoError(t, err)
	return request
}

func TestParseCriteriaMapReportsTheMissingMapSectionAsAFinding(t *testing.T) {
	criteriaMap, err := ParseCriteriaMap("Verdict: REJECTED\n\n## Achados\n\n- Severidade: high\n", reviewRequestForTest(t))
	require.NoError(t, err)

	findings := criteriaMap.FindingsList()
	require.Len(t, findings, 1, "a review without the criteria map section must give the cycle material to remediate")
	require.Equal(t, criteriaFindingRuleMissing, findings[0].Rule())
	require.Equal(t, SeverityHigh, findings[0].Severity())
	require.False(t, criteriaMap.Complete())
}

func TestParseCriteriaMapKeepsPerCriterionFindingsWhenTheSectionExists(t *testing.T) {
	raw := "## Mapa de Critérios de Aceite\n\n- [atendido] outro criterio -> go test ./... -> ok 0.4s\n"
	criteriaMap, err := ParseCriteriaMap(raw, reviewRequestForTest(t))
	require.NoError(t, err)

	for _, finding := range criteriaMap.FindingsList() {
		require.NotEqual(t, criteriaFindingRuleMissing, finding.Rule())
	}
}
