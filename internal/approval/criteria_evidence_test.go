package approval

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func criteriaRequest(t *testing.T, target string, descriptions ...string) ReviewRequest {
	t.Helper()
	task, err := NewTaskIdentity("task-1.0")
	require.NoError(t, err)
	agent, err := NewAgentIdentity("claude")
	require.NoError(t, err)
	criteria := make([]AcceptanceCriterion, 0, len(descriptions))
	for _, description := range descriptions {
		criteria = append(criteria, mustCriterion(t, description))
	}
	request, err := NewReviewRequest(task, agent, 1, NewReviewTarget(target), criteria)
	require.NoError(t, err)
	return request
}

func TestParseCriteriaMapRejectsFabricatedEvidence(t *testing.T) {
	cases := []struct {
		name     string
		evidence string
	}{
		{"inspecao manual com seta dupla", "inspecao manual -> parece correto"},
		{"prosa livre", "porque confio no autor da mudanca"},
		{"apologia do autor", "o autor disse que esta ok -> tudo certo"},
		{"revisao visual", "revisao visual -> aprovado"},
		{"prefixo Test sem resultado canonico", "TestAlgo -> parece bom"},
		{"nome de teste com resultado em prosa", "TestAlgo -> o teste roda na minha maquina"},
		{"comando inexistente", "conferencia manual -> ok"},
		{"arquivo linha fora do diff", "internal/ausente.go:42"},
		{"comando plausivel com saida trivial", "make it work -> done"},
		{"git com saida trivial", "git log -> abc"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			raw := "## Mapa de Critérios de Aceite\n- [atendido] Criterio um -> " + tc.evidence + "\n"
			criteriaMap, err := ParseCriteriaMap(raw, criteriaRequest(t, "diff sem referencia", "Criterio um"))
			require.NoError(t, err)
			require.False(t, criteriaMap.Complete())

			findings := criteriaMap.FindingsList()
			require.Len(t, findings, 1)
			require.True(t, findings[0].Severity().Blocks())

			proof, proofErr := NewApprovalProof(VerdictApproved, criteriaMap, nil)
			require.ErrorIs(t, proofErr, ErrInsufficientProof)
			require.False(t, proof.Valid())
		})
	}
}

func TestParseCriteriaMapAcceptsCanonicalEvidenceForms(t *testing.T) {
	cases := []struct {
		name     string
		evidence string
		form     EvidenceForm
	}{
		{"comando com saida registrada", "go test ./internal/approval/... -> ok 0.2s", EvidenceFormCommandOutput},
		{"arquivo linha presente no diff", "internal/approval/evidence.go:42", EvidenceFormFileLineInDiff},
		{"teste com resultado canonico", "TestCriteria -> PASS", EvidenceFormTestResult},
		{"teste com resultado fail canonico", "TestCriteria -> fail", EvidenceFormTestResult},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			raw := "## Mapa de Critérios de Aceite\n- [atendido] Criterio um -> " + tc.evidence + "\n"
			target := "--- a/internal/approval/evidence.go\n" +
				"+++ b/internal/approval/evidence.go\n" +
				"@@ -40,3 +40,4 @@\n" +
				"+var added = 1\n"
			criteriaMap, err := ParseCriteriaMap(raw, criteriaRequest(t, target, "Criterio um"))
			require.NoError(t, err)
			require.True(t, criteriaMap.Complete())
			require.Empty(t, criteriaMap.FindingsList())

			proof, proofErr := NewApprovalProof(VerdictApproved, criteriaMap, nil)
			require.NoError(t, proofErr)
			require.True(t, proof.Valid())
		})
	}
}

func TestParseCriteriaMapUnmetCriterionIsNotAnInfrastructureError(t *testing.T) {
	raw := "## Mapa de Critérios de Aceite\n- [não atendido] Criterio um -> faltou cobrir o caso de erro\n"
	criteriaMap, err := ParseCriteriaMap(raw, criteriaRequest(t, "diff", "Criterio um"))
	require.NoError(t, err)
	require.False(t, criteriaMap.Complete())
	require.Equal(t, 0, criteriaMap.Bound())

	findings := criteriaMap.FindingsList()
	require.Len(t, findings, 1)
	require.Equal(t, SeverityHigh, findings[0].Severity())
	require.True(t, findings[0].Severity().Blocks())

	_, proofErr := NewApprovalProof(VerdictApproved, criteriaMap, nil)
	require.ErrorIs(t, proofErr, ErrInsufficientProof)
}

func TestParseCriteriaMapUnverifiableCriterionIsNotAnInfrastructureError(t *testing.T) {
	raw := "## Mapa de Critérios de Aceite\n- [não verificável] Criterio um -> sem acesso ao ambiente\n"
	criteriaMap, err := ParseCriteriaMap(raw, criteriaRequest(t, "diff", "Criterio um"))
	require.NoError(t, err)
	require.False(t, criteriaMap.Complete())

	unverifiable := 0
	for range criteriaMap.Unverifiable() {
		unverifiable++
	}
	require.Equal(t, 1, unverifiable)

	findings := criteriaMap.FindingsList()
	require.Len(t, findings, 1)
	require.Equal(t, SeverityHigh, findings[0].Severity())

	_, proofErr := NewApprovalProof(VerdictApproved, criteriaMap, nil)
	require.ErrorIs(t, proofErr, ErrInsufficientProof)
}

func TestParseCriteriaMapAcceptsAllThreeMarkersWithoutAccents(t *testing.T) {
	raw := "## Mapa de Critérios de Aceite\n" +
		"- [atendido] Criterio um -> go test ./... -> PASS\n" +
		"- [nao atendido] Criterio dois -> ainda falta\n" +
		"- [nao verificavel] Criterio tres -> sem ambiente\n"
	criteriaMap, err := ParseCriteriaMap(raw, criteriaRequest(t, "diff", "Criterio um", "Criterio dois", "Criterio tres"))
	require.NoError(t, err)
	require.Equal(t, 3, criteriaMap.Total())
	require.Equal(t, 1, criteriaMap.Bound())
	require.Len(t, criteriaMap.FindingsList(), 2)
}

func TestParseCriteriaMapReportsUnknownMarkerAsFinding(t *testing.T) {
	raw := "## Mapa de Critérios de Aceite\n- [talvez] Criterio um -> go test ./... -> PASS\n"
	criteriaMap, err := ParseCriteriaMap(raw, criteriaRequest(t, "diff", "Criterio um"))
	require.NoError(t, err)
	require.False(t, criteriaMap.Complete())

	findings := criteriaMap.FindingsList()
	require.Len(t, findings, 2)
	for _, finding := range findings {
		require.True(t, finding.Severity().Blocks())
	}

	_, proofErr := NewApprovalProof(VerdictApproved, criteriaMap, nil)
	require.ErrorIs(t, proofErr, ErrInsufficientProof)
}

func TestCycleFeedsUnmetCriteriaFindingsIntoTheRound(t *testing.T) {
	criterion := mustCriterion(t, "Criterio um")
	raw := "Verdict: REJECTED\n\n## Mapa de Critérios de Aceite\n- [não atendido] Criterio um -> faltou cobrir o caso de erro\n"

	request := criteriaRequest(t, "diff", "Criterio um")
	criteriaMap, err := ParseCriteriaMap(raw, request)
	require.NoError(t, err)

	task, err := NewTaskIdentity("task-1.0")
	require.NoError(t, err)
	agent, err := NewAgentIdentity("claude")
	require.NoError(t, err)
	policy, err := NewApprovalPolicy()
	require.NoError(t, err)

	cycle, err := NewCycle(
		task, agent, policy, []AcceptanceCriterion{criterion},
		stubReviewer{output: NewReviewerOutput(raw, nil, criteriaMap)},
		stubFixer{},
		stubRepository{full: NewReviewTarget("diff"), delta: NewReviewTarget("")},
	)
	require.NoError(t, err)

	round, mapped, err := cycle.reviewRound(t.Context(), 1, NewReviewTarget("diff"))
	require.NoError(t, err)
	require.False(t, mapped.Complete())

	count := 0
	for finding := range round.Findings() {
		require.Equal(t, SeverityHigh, finding.Severity())
		count++
	}
	require.Equal(t, 1, count)
}

func TestRoundVerdictIsDowngradedWhenTheRoundCarriesBlockingFindings(t *testing.T) {
	criterion := mustCriterion(t, "Criterio um")
	raw := "Veredito: APPROVED\n\n## Mapa de Critérios de Aceite\n- [nao atendido] Criterio um -> falta cobrir o erro\n"

	criteriaMap, err := ParseCriteriaMap(raw, criteriaRequest(t, "diff", "Criterio um"))
	require.NoError(t, err)
	require.Len(t, criteriaMap.FindingsList(), 1)

	task, err := NewTaskIdentity("task-1.0")
	require.NoError(t, err)
	agent, err := NewAgentIdentity("claude")
	require.NoError(t, err)
	policy, err := NewApprovalPolicy()
	require.NoError(t, err)

	cycle, err := NewCycle(
		task, agent, policy, []AcceptanceCriterion{criterion},
		stubReviewer{output: NewReviewerOutput(raw, nil, criteriaMap)},
		stubFixer{},
		stubRepository{full: NewReviewTarget("diff"), delta: NewReviewTarget("")},
	)
	require.NoError(t, err)

	round, _, err := cycle.reviewRound(t.Context(), 1, NewReviewTarget("diff"))
	require.NoError(t, err)
	require.Equal(t, VerdictRejected, round.Verdict())
	require.False(t, round.Verdict().Approves())
}

func TestReviewRoundKeepsApprovedWhenNoFindingBlocks(t *testing.T) {
	criterion := mustCriterion(t, "Criterio um")
	raw := "Veredito: APPROVED\n\n## Mapa de Critérios de Aceite\n- [atendido] Criterio um -> go test ./... -> ok 0.2s\n"

	criteriaMap, err := ParseCriteriaMap(raw, criteriaRequest(t, "diff", "Criterio um"))
	require.NoError(t, err)
	require.Empty(t, criteriaMap.FindingsList())

	task, err := NewTaskIdentity("task-1.0")
	require.NoError(t, err)
	agent, err := NewAgentIdentity("claude")
	require.NoError(t, err)
	policy, err := NewApprovalPolicy()
	require.NoError(t, err)

	cycle, err := NewCycle(
		task, agent, policy, []AcceptanceCriterion{criterion},
		stubReviewer{output: NewReviewerOutput(raw, nil, criteriaMap)},
		stubFixer{},
		stubRepository{full: NewReviewTarget("diff"), delta: NewReviewTarget("")},
	)
	require.NoError(t, err)

	round, _, err := cycle.reviewRound(t.Context(), 1, NewReviewTarget("diff"))
	require.NoError(t, err)
	require.Equal(t, VerdictApproved, round.Verdict())
}
