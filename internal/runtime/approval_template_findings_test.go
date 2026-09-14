package runtime

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func filledReviewTemplate(t *testing.T) string {
	t.Helper()
	raw, err := os.ReadFile(filepath.Clean("../../.agents/skills/review/assets/review-report-template.md"))
	require.NoError(t, err)
	filled := string(raw)
	replacements := [][2]string{
		{"- Severidade: critical | high | medium | low\n", "- Severidade: critical\n"},
		{"- Arquivo:\n", "- Arquivo: internal/runtime/approval_adapters.go\n"},
		{"- Linha:\n", "- Linha: 296\n"},
		{"- Impacto:\n", "- Impacto: revisor que segue a skill produzia zero findings\n"},
	}
	for _, pair := range replacements {
		require.Contains(t, filled, pair[0], "review report template no longer uses the expected field form: %q", pair[0])
		filled = strings.Replace(filled, pair[0], pair[1], 1)
	}
	return filled
}

func TestParseCycleFindingsReadsTheOfficialTemplateForm(t *testing.T) {
	findings := parseCycleFindings(filledReviewTemplate(t))
	require.NotEmpty(t, findings, "a review following .agents/skills/review must produce findings for the cycle to remediate")
	require.True(t, findings[0].Severity().Blocks())
	require.Equal(t, "internal/runtime/approval_adapters.go", findings[0].File())
	require.Equal(t, 296, findings[0].Line())
}

func TestParseCycleFindingsKeepsBracketForm(t *testing.T) {
	findings := parseCycleFindings("[HIGH] internal/x/fix.go:12 achado\n")
	require.Len(t, findings, 1)
	require.True(t, findings[0].Severity().Blocks())
}
