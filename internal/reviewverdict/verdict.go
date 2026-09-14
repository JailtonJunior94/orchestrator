package reviewverdict

import (
	"regexp"
	"strings"
)

const (
	Approved            = "APPROVED"
	ApprovedWithRemarks = "APPROVED_WITH_REMARKS"
	Rejected            = "REJECTED"
	Blocked             = "BLOCKED"
)

var (
	declarationRe = regexp.MustCompile("^[\\s\\-*_>#`+]*(?:final\\s+)?(?:verdict|veredito|veredicto)(?:\\s+final)?[\\s*_`]*:\\s*(.*)$")
	accentFolder  = strings.NewReplacer(
		"ã", "a", "á", "a", "à", "a", "â", "a",
		"é", "e", "ê", "e",
		"í", "i",
		"ó", "o", "õ", "o", "ô", "o",
		"ú", "u",
		"ç", "c",
	)
	tokens = map[string]string{
		"approved":               Approved,
		"aprovado":               Approved,
		"approved_with_remarks":  ApprovedWithRemarks,
		"aprovado_com_ressalvas": ApprovedWithRemarks,
		"rejected":               Rejected,
		"reprovado":              Rejected,
		"blocked":                Blocked,
		"bloqueado":              Blocked,
	}
)

func Parse(line string) (string, bool) {
	normalized := accentFolder.Replace(strings.ToLower(strings.TrimSpace(line)))
	match := declarationRe.FindStringSubmatch(normalized)
	if match == nil {
		return "", false
	}
	return tokens[canonicalToken(match[1])], true
}

func ParseText(text string) (string, bool) {
	for _, line := range strings.Split(text, "\n") {
		if verdict, declared := Parse(line); declared {
			return verdict, true
		}
	}
	return "", false
}

func canonicalToken(raw string) string {
	token := strings.Trim(raw, " \t`'\"*_.,;:()[]")
	return strings.Join(strings.Fields(token), "_")
}
