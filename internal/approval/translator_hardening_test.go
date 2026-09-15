package approval

func (s *TranslatorSuite) TestVerdictInsideCodeFenceIsNotADeclaration() {
	cases := []struct {
		name string
		text string
	}{
		{"backtick_fence", "Analise do formato esperado:\n```\nVerdict: APPROVED\n```\nNa verdade o codigo tem problemas.\n"},
		{"backtick_fence_with_language", "exemplo:\n```markdown\nVerdict: APPROVED\n```\nfim\n"},
		{"long_backtick_fence", "exemplo:\n````\nVerdict: APPROVED\n````\nfim\n"},
		{"tilde_fence", "exemplo:\n~~~\nVerdict: APPROVED\n~~~\nfim\n"},
		{"unclosed_fence_swallows_the_rest", "exemplo:\n```\nVerdict: APPROVED\n"},
		{"inner_fence_does_not_close_outer", "```\n~~~\nVerdict: APPROVED\n~~~\n"},
	}
	for _, tc := range cases {
		s.Run(tc.name, func() {
			s.Equal(VerdictBlocked, s.translator.Translate(tc.text))
		})
	}
}

func (s *TranslatorSuite) TestDeclarationAfterAClosedFenceStillCounts() {
	s.Equal(VerdictApproved, s.translator.Translate("```\nexemplo sem veredito\n```\nVerdict: APPROVED\n"))
}

func (s *TranslatorSuite) TestConflictingDeclarationsFailClosed() {
	cases := []struct {
		name string
		text string
	}{
		{"approved_then_rejected", "Verdict: APPROVED\nmais analise\nVerdict: REJECTED\n"},
		{"rejected_then_approved", "Verdict: REJECTED\nmais analise\nVerdict: APPROVED\n"},
		{"cross_language_conflict", "Verdict: APPROVED\nVeredito: reprovado\n"},
		{"approved_then_approved_with_remarks", "Verdict: APPROVED\nVerdict: APPROVED_WITH_REMARKS\n"},
	}
	for _, tc := range cases {
		s.Run(tc.name, func() {
			s.Equal(VerdictBlocked, s.translator.Translate(tc.text))
		})
	}
}

func (s *TranslatorSuite) TestRedundantIdenticalDeclarationsAreAccepted() {
	s.Equal(VerdictApproved, s.translator.Translate("Verdict: APPROVED\nresumo\nVerdict: approved\n"))
	s.Equal(VerdictRejected, s.translator.Translate("Verdict: REJECTED\nVerdict: REJECTED\n"))
}

func (s *TranslatorSuite) TestHonestDeclarationsRemainAccepted() {
	cases := []struct {
		name string
		text string
		want Verdict
	}{
		{"plain_english", "Verdict: APPROVED", VerdictApproved},
		{"plain_portuguese", "Veredito: aprovado", VerdictApproved},
		{"markdown_decorated", "- **Veredito:** APPROVED", VerdictApproved},
		{"inline_code_token", "Veredito: `aprovado`", VerdictApproved},
		{"full_report", "# Relatorio\n\n- Veredito: APPROVED\n\n## Achados\n\nSem achados.\n", VerdictApproved},
	}
	for _, tc := range cases {
		s.Run(tc.name, func() {
			s.Equal(tc.want, s.translator.Translate(tc.text))
		})
	}
}
