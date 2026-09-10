package approval

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

type TranslatorSuite struct {
	suite.Suite
	translator Translator
}

func TestTranslatorSuite(t *testing.T) {
	suite.Run(t, new(TranslatorSuite))
}

func (s *TranslatorSuite) SetupTest() {
	s.translator = NewTranslator()
}

func (s *TranslatorSuite) TestCanonicalDeclarations() {
	cases := []struct {
		name string
		text string
		want Verdict
	}{
		{"en_approved", "verdict: APPROVED", VerdictApproved},
		{"en_approved_with_remarks", "verdict: APPROVED_WITH_REMARKS", VerdictApprovedWithRemarks},
		{"en_rejected", "verdict: REJECTED", VerdictRejected},
		{"en_blocked", "verdict: BLOCKED", VerdictBlocked},
		{"pt_aprovado", "veredito: APROVADO", VerdictApproved},
		{"pt_aprovado_com_ressalvas", "veredito: aprovado com ressalvas", VerdictApprovedWithRemarks},
		{"pt_reprovado", "veredicto: REPROVADO", VerdictRejected},
		{"pt_bloqueado", "veredito: bloqueado", VerdictBlocked},
		{"markdown_bold", "**Verdict:** `approved`", VerdictApproved},
		{"embedded_line", "summary of review\nVerdict: rejected\nmore notes", VerdictRejected},
	}

	for _, tc := range cases {
		s.Run(tc.name, func() {
			s.Equal(tc.want, s.translator.Translate(tc.text))
		})
	}
}

func (s *TranslatorSuite) TestApprovedWithRemarksIsNotReadAsApproved() {
	s.Equal(VerdictApprovedWithRemarks, s.translator.Translate("verdict: APPROVED_WITH_REMARKS"))
	s.NotEqual(VerdictApproved, s.translator.Translate("verdict: APPROVED_WITH_REMARKS"))
}

func (s *TranslatorSuite) TestFailsClosedWithoutCanonicalDeclaration() {
	cases := []string{
		"",
		"the code looks fine to me, no blockers",
		"LGTM, ship it",
		"no critical findings were identified",
		"status: approved",
		"verdict: something-else",
	}
	for _, text := range cases {
		s.Run(text, func() {
			s.Equal(VerdictBlocked, s.translator.Translate(text))
		})
	}
}

func FuzzTranslator(f *testing.F) {
	f.Add("verdict: APPROVED")
	f.Add("verdict: APPROVED_WITH_REMARKS")
	f.Add("veredito: reprovado")
	f.Add("no negative markers here")
	f.Add("")
	f.Add("verdict:\n\n\tapproved   ")

	translator := NewTranslator()
	valid := map[Verdict]struct{}{
		VerdictApproved:            {},
		VerdictApprovedWithRemarks: {},
		VerdictRejected:            {},
		VerdictBlocked:             {},
	}

	f.Fuzz(func(t *testing.T, text string) {
		got := translator.Translate(text)
		if _, ok := valid[got]; !ok {
			t.Fatalf("translator produced an out-of-set verdict %d for %q", int(got), text)
		}
		if got == VerdictApproved && translator.Translate(text) != VerdictApproved {
			t.Fatal("translator is not deterministic")
		}
	})
}
