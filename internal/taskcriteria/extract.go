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

		if inSection && strings.HasPrefix(trimmed, "#") {
			inSection = false
			continue
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
	if !strings.HasPrefix(trimmed, "- ") {
		return "", false, false
	}

	item := strings.TrimSpace(trimmed[2:])
	if strings.HasPrefix(item, "[") {
		closing := strings.Index(item, "]")
		if closing < 0 {
			return "", false, false
		}
		marker := strings.TrimSpace(item[1:closing])
		checked = strings.EqualFold(marker, "x")
		item = strings.TrimSpace(item[closing+1:])
	}

	if item == "" {
		return "", false, false
	}

	return item, checked, true
}

func isAcceptanceSection(line string) bool {
	if !strings.HasPrefix(line, "#") {
		return false
	}
	heading := strings.ToLower(strings.TrimSpace(strings.TrimLeft(line, "#")))
	for _, name := range acceptanceHeadings {
		if strings.HasPrefix(heading, name) {
			return true
		}
	}
	return false
}

var acceptanceHeadings = []string{
	"definition of done",
	"acceptance criteria",
	"criterios de sucesso",
	"critérios de sucesso",
	"criterios de aceite",
	"critérios de aceite",
}
