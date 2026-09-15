package approval

import "testing"

func TestParseReviewFindingsDetectsLegitimateFindingForms(t *testing.T) {
	cases := map[string]struct {
		line string
		want Severity
	}{
		"plain list":     {"- [HIGH] internal/a.go:1 corrida de dados", SeverityHigh},
		"bold list":      {"- **[HIGH]** internal/a.go:1 corrida de dados", SeverityHigh},
		"star list":      {"* [high] internal/a.go:1 corrida de dados", SeverityHigh},
		"quoted list":    {"> - [HIGH] internal/a.go:1 corrida de dados", SeverityHigh},
		"indented list":  {"    - [HIGH] internal/a.go:1 corrida de dados", SeverityHigh},
		"no space":       {"-[HIGH] internal/a.go:1 corrida de dados", SeverityHigh},
		"accented":       {"- [crítico] internal/a.go:1 vazamento", SeverityCritical},
		"underscored":    {"__[HIGH]__ internal/a.go:1 corrida de dados", SeverityHigh},
		"numbered list":  {"1. [HIGH] internal/a.go:1 corrida de dados", SeverityHigh},
		"numbered paren": {"1) [HIGH] internal/a.go:1 corrida de dados", SeverityHigh},
		"heading":        {"### [HIGH] internal/a.go:1 corrida de dados", SeverityHigh},
		"table cell":     {"| [HIGH] | internal/a.go:1 | corrida de dados |", SeverityHigh},
		"checkbox":       {"- [ ] [HIGH] internal/a.go:1 corrida de dados", SeverityHigh},
		"checked box":    {"- [x] [HIGH] internal/a.go:1 corrida de dados", SeverityHigh},
		"backtick tag":   {"- `[HIGH]` internal/a.go:1 corrida de dados", SeverityHigh},
	}

	for name, testCase := range cases {
		t.Run(name, func(t *testing.T) {
			findings := ParseReviewFindings(testCase.line + "\n")
			if len(findings) != 1 {
				t.Fatalf("esperado 1 achado para %q, obtido %d", testCase.line, len(findings))
			}
			if findings[0].Severity() != testCase.want {
				t.Fatalf("severidade %v, esperada %v para %q", findings[0].Severity(), testCase.want, testCase.line)
			}
		})
	}
}

func TestParseReviewFindingsDetectsSeverityFieldForms(t *testing.T) {
	cases := map[string]struct {
		block string
		want  Severity
	}{
		"plain field":    {"- Severidade: high\n- Arquivo: internal/a.go\n- Linha: 1\n- Impacto: corrida\n", SeverityHigh},
		"bold field":     {"- **Severidade**: high\n- Arquivo: internal/a.go\n- Linha: 1\n- Impacto: corrida\n", SeverityHigh},
		"plus field":     {"+ Severidade: high\n- Arquivo: internal/a.go\n- Linha: 1\n- Impacto: corrida\n", SeverityHigh},
		"numbered field": {"1. Severidade: high\n- Arquivo: internal/a.go\n- Linha: 1\n- Impacto: corrida\n", SeverityHigh},
		"backtick value": {"- Severidade: `high`\n- Arquivo: internal/a.go\n- Linha: 1\n- Impacto: corrida\n", SeverityHigh},
	}

	for name, testCase := range cases {
		t.Run(name, func(t *testing.T) {
			findings := ParseReviewFindings(testCase.block)
			if len(findings) != 1 {
				t.Fatalf("esperado 1 achado para %q, obtido %d", testCase.block, len(findings))
			}
			if findings[0].Severity() != testCase.want {
				t.Fatalf("severidade %v, esperada %v", findings[0].Severity(), testCase.want)
			}
		})
	}
}

func TestParseReviewFindingsIgnoresProseMention(t *testing.T) {
	cases := map[string]string{
		"inline mention":  "Veredito APPROVED_WITH_REMARKS, sem tag [CRITICAL]/[HIGH] bloqueante.",
		"inline backtick": "Veredito `APPROVED_WITH_REMARKS`, sem tag `[CRITICAL]`/`[HIGH]` bloqueante.",
		"bullet prose":    "- Veredito do Revisor: APPROVED_WITH_REMARKS (sem tag `[critical]`/`[blocker]`)",
		"tail mention":    "- o revisor nao usou a tag [HIGH] em nenhum achado",
	}

	for name, line := range cases {
		t.Run(name, func(t *testing.T) {
			if findings := ParseReviewFindings(line + "\n"); len(findings) != 0 {
				t.Fatalf("mencao em prosa virou achado: %q -> %v", line, findings[0].Severity())
			}
		})
	}
}
