package evidence

import "testing"

func TestRemarksClosureFindingsAcceptsLegitimateFindingForms(t *testing.T) {
	cases := map[string]string{
		"plain list":     "- [HIGH] internal/a.go:1 corrida de dados",
		"bold list":      "- **[HIGH]** internal/a.go:1 corrida de dados",
		"star list":      "* [high] internal/a.go:1 corrida de dados",
		"quoted list":    "> - [HIGH] internal/a.go:1 corrida de dados",
		"indented list":  "    - [HIGH] internal/a.go:1 corrida de dados",
		"no space":       "-[HIGH] internal/a.go:1 corrida de dados",
		"accented":       "- [crítico] internal/a.go:1 vazamento",
		"underscored":    "__[HIGH]__ internal/a.go:1 corrida de dados",
		"severity field": "- Severidade: high",
		"bold field":     "- **Severidade**: high",
		"plus field":     "+ Severidade: high",
		"numbered list":  "1. [HIGH] internal/a.go:1 corrida de dados",
		"numbered paren": "1) [HIGH] internal/a.go:1 corrida de dados",
		"heading":        "### [HIGH] internal/a.go:1 corrida de dados",
		"table cell":     "| [HIGH] | internal/a.go:1 | corrida de dados |",
		"table column":   "| id | [HIGH] | internal/a.go:1 |",
		"checkbox":       "- [ ] [HIGH] internal/a.go:1 corrida de dados",
		"checked box":    "- [x] [HIGH] internal/a.go:1 corrida de dados",
		"backtick tag":   "- `[HIGH]` internal/a.go:1 corrida de dados",
		"backtick field": "- Severidade: `high`",
	}

	for name, line := range cases {
		t.Run(name, func(t *testing.T) {
			findings := remarksClosureFindings("# Achados\n" + line + "\n")
			if len(findings) == 0 {
				t.Fatalf("achado bloqueante nao detectado na forma %q", line)
			}
			if findings[0].Label != "veredito APPROVED_WITH_REMARKS nao encerra com achado high/critical declarado (RF-33)" {
				t.Fatalf("achado detectado pelo motivo errado: %q", findings[0].Label)
			}
		})
	}
}

func TestRemarksClosureFindingsIgnoresProseMention(t *testing.T) {
	cases := map[string]string{
		"inline mention":        "- Veredito do Revisor: APPROVED_WITH_REMARKS (sem tag [CRITICAL]/[HIGH])",
		"inline backtick":       "Veredito `APPROVED_WITH_REMARKS`, sem tag `[CRITICAL]`/`[HIGH]` bloqueante.",
		"mid sentence severity": "o texto cita severidade: high apenas como exemplo em prosa",
	}

	for name, line := range cases {
		t.Run(name, func(t *testing.T) {
			text := "# Achados\n" + line + "\n- [LOW] internal/a.go:3 renomear variavel\n"
			if findings := remarksClosureFindings(text); len(findings) != 0 {
				t.Fatalf("mencao em prosa tratada como achado bloqueante: %q -> %q", line, findings[0].Label)
			}
		})
	}
}

func TestRemarksClosureFindingsIgnoresFencedBlock(t *testing.T) {
	text := "# Achados\n" +
		"```text\n" +
		"- [CRITICAL] exemplo citado dentro de bloco de codigo\n" +
		"```\n" +
		"- [LOW] internal/a.go:3 renomear variavel\n"

	if findings := remarksClosureFindings(text); len(findings) != 0 {
		t.Fatalf("achado dentro de bloco cercado bloqueou o encerramento: %q", findings[0].Label)
	}
}

func TestRemarksClosureFindingsFailsClosedWithoutAnyDeclaredSeverity(t *testing.T) {
	findings := remarksClosureFindings("# Achados\nnenhuma severidade declarada aqui\n")
	if len(findings) != 1 {
		t.Fatalf("esperado exatamente 1 achado fail-closed, obtido %d", len(findings))
	}
	want := "veredito APPROVED_WITH_REMARKS sem achado declarado com severidade canonica: ausencia de high/critical nao verificavel (RF-33, fail-closed)"
	if findings[0].Label != want {
		t.Fatalf("label inesperado: %q", findings[0].Label)
	}
}
