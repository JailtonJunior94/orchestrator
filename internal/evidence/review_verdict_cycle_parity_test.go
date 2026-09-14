package evidence_test

import (
	"testing"

	"github.com/JailtonJunior94/ai-spec-harness/internal/approval"
	"github.com/JailtonJunior94/ai-spec-harness/internal/reviewverdict"
)

func cycleVerdictName(v approval.Verdict) string {
	switch v {
	case approval.VerdictApproved:
		return reviewverdict.Approved
	case approval.VerdictApprovedWithRemarks:
		return reviewverdict.ApprovedWithRemarks
	case approval.VerdictRejected:
		return reviewverdict.Rejected
	default:
		return reviewverdict.Blocked
	}
}

func TestReviewVerdictExtractionMatchesApprovalCycle(t *testing.T) {
	cases := map[string]string{
		"verdict inside fenced block":     "Analise.\n\n```\nveredito: APPROVED\n```\n\nNada declarado fora da cerca.\n",
		"verdict inside tilde fence":      "Texto.\n\n~~~\nverdict: APPROVED\n~~~\n",
		"conflicting verdicts":            "verdict: APPROVED\nmais analise\nverdict: REJECTED\n",
		"repeated consistent verdicts":    "verdict: APPROVED\ntexto\nverdict: APPROVED\n",
		"single approved declaration":     "Revisao concluida.\nverdict: APPROVED\n",
		"single rejected declaration":     "verdict: REJECTED\n",
		"approved with remarks":           "veredito: APPROVED_WITH_REMARKS\n",
		"no declared verdict":             "Revisei o diff e nao encontrei nada relevante.\n",
		"unknown verdict marker":          "verdict: TALVEZ\n",
		"outside fence wins over inside":  "```\nverdict: APPROVED\n```\nverdict: REJECTED\n",
		"unterminated fence":              "```\nverdict: APPROVED\n",
		"accented portuguese declaration": "Veredito Final: Aprovado\n",
	}

	translator := approval.NewTranslator()
	for name, text := range cases {
		t.Run(name, func(t *testing.T) {
			want := cycleVerdictName(translator.Translate(text))
			got, _ := reviewverdict.ParseDocument(text)
			if got == "" {
				got = reviewverdict.Blocked
			}
			if got != want {
				t.Fatalf("validador e Ciclo divergem: validador=%q Ciclo=%q\ntexto:\n%s", got, want, text)
			}
		})
	}
}
