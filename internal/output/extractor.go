package output

import (
	"errors"
	"regexp"
	"strings"
)

var (
	// ErrJSONNotFound indicates that no JSON payload could be extracted.
	ErrJSONNotFound = errors.New("json payload not found in provider output")
	codeBlockRE     = regexp.MustCompile("(?s)```json\\s*(.*?)\\s*```")
)

// ExtractJSON attempts to find a JSON payload inside the provider output.
func ExtractJSON(raw string) (string, error) {
	if matches := codeBlockRE.FindStringSubmatch(raw); len(matches) == 2 {
		return strings.TrimSpace(matches[1]), nil
	}

	if jsonText, ok := extractBalancedJSON(raw); ok {
		return jsonText, nil
	}

	trimmed := strings.TrimSpace(raw)
	if strings.HasPrefix(trimmed, "{") || strings.HasPrefix(trimmed, "[") {
		return trimmed, nil
	}

	return "", ErrJSONNotFound
}

func extractBalancedJSON(raw string) (string, bool) {
	start := -1
	var open rune
	for idx, r := range raw {
		if r == '{' || r == '[' {
			start = idx
			open = r
			break
		}
	}
	if start == -1 {
		return "", false
	}

	var close rune
	if open == '{' {
		close = '}'
	} else {
		close = ']'
	}

	depth := 0
	inString := false
	escape := false
	for idx, r := range raw[start:] {
		switch {
		case escape:
			escape = false
		case r == '\\' && inString:
			escape = true
		case r == '"':
			inString = !inString
		case !inString && r == open:
			depth++
		case !inString && r == close:
			depth--
			if depth == 0 {
				end := start + idx + 1
				return strings.TrimSpace(raw[start:end]), true
			}
		}
	}

	return "", false
}
