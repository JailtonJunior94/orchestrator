package approval

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

const reviewTemplatePath = "../../.agents/skills/review/assets/review-report-template.md"

func reviewTemplate(t *testing.T) string {
	t.Helper()
	raw, err := os.ReadFile(filepath.Clean(reviewTemplatePath))
	require.NoError(t, err)
	return string(raw)
}

func filledTemplateReport(t *testing.T) string {
	t.Helper()
	raw := reviewTemplate(t)
	replacements := [][2]string{
		{"- Severidade: critical | high | medium | low\n", "- Severidade: high\n"},
		{"- Arquivo:\n", "- Arquivo: internal/approval/translator.go\n"},
		{"- Linha:\n", "- Linha: 42\n"},
		{"- Impacto:\n", "- Impacto: verdict declared inside a code fence was accepted\n"},
	}
	filled := raw
	for _, pair := range replacements {
		require.Contains(t, filled, pair[0], "review report template no longer uses the expected field form: %q", pair[0])
		filled = strings.Replace(filled, pair[0], pair[1], 1)
	}
	return filled
}

func TestParseReviewFindingsAcceptsTheOfficialTemplateForm(t *testing.T) {
	findings := ParseReviewFindings(filledTemplateReport(t))
	require.Len(t, findings, 1)
	require.Equal(t, SeverityHigh, findings[0].Severity())
	require.Equal(t, "internal/approval/translator.go", findings[0].File())
	require.Equal(t, 42, findings[0].Line())
	require.Equal(t, ReviewFindingRule, findings[0].Rule())
	require.Equal(t, "verdict declared inside a code fence was accepted", findings[0].Description())
}

func TestParseReviewFindingsIgnoresUnfilledTemplatePlaceholders(t *testing.T) {
	require.Empty(t, ParseReviewFindings(reviewTemplate(t)))
}

func TestParseReviewFindingsKeepsTheBracketForm(t *testing.T) {
	findings := ParseReviewFindings("- [critical] internal/x/fix.go:12 corrompe estado\n- [low] renomear variavel\n")
	require.Len(t, findings, 2)
	require.Equal(t, SeverityCritical, findings[0].Severity())
	require.Equal(t, "internal/x/fix.go", findings[0].File())
	require.Equal(t, 12, findings[0].Line())
	require.Equal(t, SeverityLow, findings[1].Severity())
	require.Equal(t, ReviewFindingFile, findings[1].File())
}

func TestParseReviewFindingsSeparatesConsecutiveBlocks(t *testing.T) {
	raw := strings.Join([]string{
		"## Achados",
		"- Severidade: critical",
		"- Arquivo: internal/a.go",
		"- Linha: 7",
		"- Impacto: primeiro",
		"- Dica de correção: x",
		"- Severidade: medium",
		"- Arquivo: internal/b.go",
		"- Impacto: segundo",
		"## Arquivos Revisados",
		"- Severidade: low",
	}, "\n")
	findings := ParseReviewFindings(raw)
	require.Len(t, findings, 3)
	require.Equal(t, SeverityCritical, findings[0].Severity())
	require.Equal(t, 7, findings[0].Line())
	require.Equal(t, SeverityMedium, findings[1].Severity())
	require.Equal(t, "internal/b.go", findings[1].File())
	require.Equal(t, 0, findings[1].Line())
	require.Equal(t, SeverityLow, findings[2].Severity())
}

func TestParseReviewFindingsAcceptsPortugueseSeverityTokens(t *testing.T) {
	findings := ParseReviewFindings("- Severidade: crítico\n- Arquivo: internal/a.go\n")
	require.Len(t, findings, 1)
	require.Equal(t, SeverityCritical, findings[0].Severity())
}
