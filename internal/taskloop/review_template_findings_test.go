package taskloop

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func filledReviewTemplateForTaskLoop(t *testing.T) string {
	t.Helper()
	raw, err := os.ReadFile(filepath.Clean("../../.agents/skills/review/assets/review-report-template.md"))
	require.NoError(t, err)

	filled := string(raw)
	replacements := [][2]string{
		{"- Severidade: critical | high | medium | low\n", "- Severidade: critical\n"},
		{"- Arquivo:\n", "- Arquivo: internal/taskloop/reviewer.go\n"},
		{"- Linha:\n", "- Linha: 228\n"},
		{"- Impacto:\n", "- Impacto: revisor que segue a skill produzia zero findings\n"},
	}
	for _, pair := range replacements {
		require.Contains(t, filled, pair[0], "review report template no longer uses the expected field form: %q", pair[0])
		filled = strings.Replace(filled, pair[0], pair[1], 1)
	}
	return filled
}

func TestCatalogParseFindingsReadsTheOfficialTemplateForm(t *testing.T) {
	findings := NewCatalog().parseFindings(filledReviewTemplateForTaskLoop(t))
	require.NotEmpty(t, findings, "a review following .agents/skills/review must feed the task loop with findings")
	require.Equal(t, SeverityCritical, findings[0].Severity)
	require.Equal(t, "internal/taskloop/reviewer.go", findings[0].File)
	require.Equal(t, 228, findings[0].Line)
}

func TestCatalogParseFindingsKeepsBracketForm(t *testing.T) {
	findings := NewCatalog().parseFindings("[HIGH] internal/x/fix.go:12 achado\n")
	require.Len(t, findings, 1)
}
