package taskloop

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseFindingsReadsTheOfficialTemplateForm(t *testing.T) {
	raw, err := os.ReadFile(filepath.Clean("../../.agents/skills/review/assets/review-report-template.md"))
	if err != nil {
		t.Fatal(err)
	}
	filled := string(raw)
	replacements := [][2]string{
		{"- Severidade: critical | high | medium | low\n", "- Severidade: critical\n"},
		{"- Arquivo:\n", "- Arquivo: internal/taskloop/reviewer.go\n"},
		{"- Linha:\n", "- Linha: 231\n"},
		{"- Impacto:\n", "- Impacto: revisor que segue a skill produzia zero findings\n"},
	}
	for _, pair := range replacements {
		if !strings.Contains(filled, pair[0]) {
			t.Fatalf("review report template no longer uses the expected field form: %q", pair[0])
		}
		filled = strings.Replace(filled, pair[0], pair[1], 1)
	}

	findings := NewCatalog().parseFindings(filled)
	if len(findings) != 1 {
		t.Fatalf("parseFindings() len = %d, want 1", len(findings))
	}
	if findings[0].Severity != SeverityCritical {
		t.Errorf("severity = %q, want %q", findings[0].Severity, SeverityCritical)
	}
	if findings[0].File != "internal/taskloop/reviewer.go" || findings[0].Line != 231 {
		t.Errorf("file/line = %q:%d, want internal/taskloop/reviewer.go:231", findings[0].File, findings[0].Line)
	}
}
