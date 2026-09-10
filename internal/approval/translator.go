package approval

import "strings"

var verdictPrefixes = []string{"verdict:", "veredicto:", "veredito:"}

type Translator struct{}

func NewTranslator() Translator {
	return Translator{}
}

func (t Translator) Translate(rawText string) Verdict {
	for line := range strings.Lines(rawText) {
		if verdict, declared := t.fromLine(line); declared {
			return verdict
		}
	}
	return VerdictBlocked
}

func (t Translator) fromLine(line string) (Verdict, bool) {
	normalized := strings.ToLower(strings.TrimSpace(line))
	normalized = strings.ReplaceAll(normalized, "*", "")
	normalized = strings.ReplaceAll(normalized, "#", "")
	normalized = strings.TrimSpace(normalized)
	for _, prefix := range verdictPrefixes {
		rest, found := strings.CutPrefix(normalized, prefix)
		if !found {
			continue
		}
		return t.fromToken(rest), true
	}
	return 0, false
}

func (t Translator) fromToken(raw string) Verdict {
	token := strings.Trim(raw, " \t`'\".,;:()[]")
	token = strings.Join(strings.Fields(token), "_")
	switch token {
	case "approved_with_remarks", "aprovado_com_ressalvas":
		return VerdictApprovedWithRemarks
	case "approved", "aprovado":
		return VerdictApproved
	case "rejected", "reprovado":
		return VerdictRejected
	case "blocked", "bloqueado":
		return VerdictBlocked
	default:
		return VerdictBlocked
	}
}
