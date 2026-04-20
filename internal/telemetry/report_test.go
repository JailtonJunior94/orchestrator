package telemetry

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func writeLogLines(t *testing.T, dir string, lines []string) {
	t.Helper()
	logPath := filepath.Join(dir, ".agents", "telemetry.log")
	if err := os.MkdirAll(filepath.Dir(logPath), 0755); err != nil {
		t.Fatalf("criar diretório de log: %v", err)
	}
	if err := os.WriteFile(logPath, []byte(strings.Join(lines, "\n")+"\n"), 0644); err != nil {
		t.Fatalf("escrever log: %v", err)
	}
}

func nowLine(skill, ref string) string {
	ts := time.Now().UTC().Format(time.RFC3339)
	if ref == "" {
		return fmt.Sprintf("%s skill=%s", ts, skill)
	}
	return fmt.Sprintf("%s skill=%s ref=%s", ts, skill, ref)
}

func oldLine(skill, ref string) string {
	ts := time.Now().UTC().Add(-48 * time.Hour).Format(time.RFC3339)
	if ref == "" {
		return fmt.Sprintf("%s skill=%s", ts, skill)
	}
	return fmt.Sprintf("%s skill=%s ref=%s", ts, skill, ref)
}

func TestReport(t *testing.T) {
	tests := []struct {
		name      string
		lines     []string
		since     time.Duration
		setup     func(dir string) // setup alternativo ao writeLogLines
		wantZero  bool
		wantTotal int
		check     func(t *testing.T, data ReportData)
	}{
		{
			name:     "log_ausente",
			lines:    nil,
			wantZero: true,
		},
		{
			name:  "sem_dados_no_periodo",
			lines: []string{oldLine("bugfix", "bug.md"), oldLine("review", "schema.json")},
			since: 1 * time.Hour,
			check: func(t *testing.T, data ReportData) {
				if data.TotalInvocations != 0 {
					t.Errorf("esperava TotalInvocations=0, obteve %d", data.TotalInvocations)
				}
			},
		},
		{
			name: "agregacao_correta",
			lines: []string{
				nowLine("bugfix", "bug.md"),
				nowLine("bugfix", "bug.md"),
				nowLine("bugfix", "bug.md"),
				nowLine("review", "schema.json"),
			},
			wantTotal: 4,
			check: func(t *testing.T, data ReportData) {
				if len(data.Skills) == 0 {
					t.Fatal("esperava ao menos uma skill")
				}
				if data.Skills[0].Name != "bugfix" {
					t.Errorf("esperava top skill=bugfix, obteve %s", data.Skills[0].Name)
				}
				if data.Skills[0].Count != 3 {
					t.Errorf("esperava bugfix count=3, obteve %d", data.Skills[0].Count)
				}
				if fmt.Sprintf("%.1f", data.Skills[0].Percentage) != "75.0" {
					t.Errorf("esperava bugfix pct=75.0, obteve %.1f", data.Skills[0].Percentage)
				}
				// refs: bug.md=2 (contadas por load, não por invocação)
				if len(data.Refs) == 0 || data.Refs[0].Name != "bug.md" {
					t.Errorf("esperava top ref=bug.md")
				}
				if data.Refs[0].Count != 3 {
					t.Errorf("esperava bug.md count=3, obteve %d", data.Refs[0].Count)
				}
			},
		},
		{
			name: "alerta_skill_sem_ref",
			lines: []string{
				nowLine("foo", ""),
				nowLine("foo", ""),
				nowLine("bar", "bar.md"),
			},
			check: func(t *testing.T, data ReportData) {
				if len(data.Alerts) == 0 {
					t.Fatal("esperava alerta para skill sem ref")
				}
				found := false
				for _, a := range data.Alerts {
					if strings.Contains(a, "foo") && strings.Contains(a, "bypass") {
						found = true
					}
				}
				if !found {
					t.Errorf("alerta esperado não encontrado, got: %v", data.Alerts)
				}
			},
		},
		{
			name: "top5_trunca_lista",
			lines: func() []string {
				skills := []string{"a", "b", "c", "d", "e", "f", "g"}
				var lines []string
				for _, s := range skills {
					lines = append(lines, nowLine(s, s+".md"))
				}
				return lines
			}(),
			check: func(t *testing.T, data ReportData) {
				if len(data.Skills) != 5 {
					t.Errorf("esperava Skills com 5 entradas, obteve %d", len(data.Skills))
				}
				if len(data.Refs) != 5 {
					t.Errorf("esperava Refs com 5 entradas, obteve %d", len(data.Refs))
				}
			},
		},
		{
			name: "json_valido",
			lines: []string{
				nowLine("bugfix", "bug.md"),
				nowLine("review", "schema.json"),
			},
			check: func(t *testing.T, data ReportData) {
				b, err := FormatJSON(data)
				if err != nil {
					t.Fatalf("FormatJSON: %v", err)
				}
				var round ReportData
				if err := json.Unmarshal(b, &round); err != nil {
					t.Fatalf("Unmarshal do JSON gerado: %v", err)
				}
				if round.TotalInvocations != data.TotalInvocations {
					t.Errorf("round-trip divergiu: %d != %d", round.TotalInvocations, data.TotalInvocations)
				}
			},
		},
		{
			name: "linha_malformada_ignorada",
			lines: []string{
				"linha inválida sem timestamp",
				nowLine("bugfix", "bug.md"),
				"outra linha ruim",
				nowLine("review", ""),
			},
			wantTotal: 2,
			check: func(t *testing.T, data ReportData) {
				if data.TotalInvocations != 2 {
					t.Errorf("esperava TotalInvocations=2, obteve %d", data.TotalInvocations)
				}
			},
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()

			if tc.lines != nil {
				writeLogLines(t, dir, tc.lines)
			}

			data, err := Report(dir, tc.since)
			if err != nil {
				t.Fatalf("Report retornou erro inesperado: %v", err)
			}

			if tc.wantZero {
				if data.TotalInvocations != 0 || len(data.Skills) != 0 {
					t.Errorf("esperava ReportData zero-value, obteve: %+v", data)
				}
				return
			}

			if tc.wantTotal > 0 && data.TotalInvocations != tc.wantTotal {
				t.Errorf("TotalInvocations: queria %d, obteve %d", tc.wantTotal, data.TotalInvocations)
			}

			if tc.check != nil {
				tc.check(t, data)
			}
		})
	}
}

func TestFormatText_SemDados(t *testing.T) {
	out := FormatText(ReportData{})
	if !strings.Contains(out, "Sem dados") {
		t.Errorf("esperava mensagem de sem dados, obteve: %s", out)
	}
}

func TestFormatText_ComDados(t *testing.T) {
	data := ReportData{
		Period:            "all",
		TotalInvocations:  2,
		Skills:            []SkillMetric{{Name: "bugfix", Count: 2, Percentage: 100}},
		Refs:              []RefMetric{{Name: "bug.md", Count: 2}},
		EstimatedTokens:   1140,
		RefsPerInvocation: 1.0,
		Alerts:            []string{"skill 'x' invocada sem ref"},
	}
	out := FormatText(data)
	for _, want := range []string{"bugfix", "bug.md", "Alertas", "sem ref"} {
		if !strings.Contains(out, want) {
			t.Errorf("esperava %q na saída texto, obteve:\n%s", want, out)
		}
	}
}
