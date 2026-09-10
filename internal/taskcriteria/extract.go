package taskcriteria

import "strings"

func Extract(content []byte) []string {
	return collect(content, false)
}

func Pending(content []byte) []string {
	return collect(content, true)
}

func collect(content []byte, onlyUnchecked bool) []string {
	var items []string
	inSection := false
	for _, line := range strings.Split(string(content), "\n") {
		trimmed := strings.TrimSpace(line)

		if isAcceptanceSection(trimmed) {
			inSection = true
			continue
		}

		if inSection && strings.HasPrefix(trimmed, "## ") && !isAcceptanceSection(trimmed) {
			break
		}

		if !inSection {
			continue
		}

		text, checked, ok := parseChecklistItem(trimmed)
		if !ok {
			continue
		}
		if onlyUnchecked && checked {
			continue
		}
		items = append(items, text)
	}

	return items
}

func parseChecklistItem(trimmed string) (text string, checked bool, ok bool) {
	if !strings.HasPrefix(trimmed, "- [") {
		return "", false, false
	}

	item := trimmed[3:]
	if len(item) < 2 {
		return "", false, false
	}

	checked = item[0] == 'x' || item[0] == 'X'
	rest := strings.TrimPrefix(item[1:], "]")
	text = strings.TrimSpace(rest)
	if text == "" {
		return "", false, false
	}

	return text, checked, true
}

func isAcceptanceSection(line string) bool {
	lower := strings.ToLower(line)
	return strings.HasPrefix(lower, "## definition of done") ||
		strings.HasPrefix(lower, "## criterios de sucesso") ||
		strings.HasPrefix(lower, "## critérios de sucesso") ||
		strings.HasPrefix(lower, "## acceptance criteria")
}
