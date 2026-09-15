package specs

import "strings"

var shellInterpreters = map[string]bool{
	"bash":   true,
	"sh":     true,
	"zsh":    true,
	"source": true,
	".":      true,
}

var shellCommandPrefixes = map[string]bool{
	"exec":    true,
	"command": true,
	"nohup":   true,
	"env":     true,
	"if":      true,
	"then":    true,
	"else":    true,
	"elif":    true,
	"do":      true,
	"while":   true,
	"until":   true,
	"!":       true,
}

func scriptExecutesTarget(body []byte, target string) bool {
	text := string(body)
	holders := variablesHoldingPath(text, target)
	for _, segment := range commandSegments(text) {
		if segmentExecutesTarget(segment, target, holders) {
			return true
		}
	}
	return false
}

func variablesHoldingPath(text, target string) map[string]bool {
	holders := make(map[string]bool)
	for _, segment := range commandSegments(text) {
		name, value, ok := assignmentIn(segment)
		if !ok {
			continue
		}
		if strings.Contains(value, target) {
			holders[name] = true
		}
	}
	return holders
}

func assignmentIn(segment string) (string, string, bool) {
	trimmed := strings.TrimSpace(segment)
	for _, keyword := range []string{"readonly ", "export ", "local ", "declare "} {
		trimmed = strings.TrimPrefix(trimmed, keyword)
		trimmed = strings.TrimSpace(trimmed)
	}
	name, value, ok := splitAssignment(trimmed)
	if !ok {
		return "", "", false
	}
	return name, value, true
}

func splitAssignment(token string) (string, string, bool) {
	idx := strings.Index(token, "=")
	if idx <= 0 {
		return "", "", false
	}
	name := token[:idx]
	for i, r := range name {
		isLetter := (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || r == '_'
		isDigit := r >= '0' && r <= '9'
		if !isLetter && (!isDigit || i == 0) {
			return "", "", false
		}
	}
	return name, token[idx+1:], true
}

func commandSegments(text string) []string {
	fields := strings.FieldsFunc(text, func(r rune) bool {
		return r == '\n' || r == ';' || r == '|' || r == '&'
	})
	out := make([]string, 0, len(fields))
	for _, f := range fields {
		if strings.TrimSpace(f) != "" {
			out = append(out, f)
		}
	}
	return out
}

func segmentExecutesTarget(segment, target string, holders map[string]bool) bool {
	tokens := strings.Fields(segment)
	for len(tokens) > 0 {
		head := unquote(tokens[0])
		if shellCommandPrefixes[head] {
			tokens = tokens[1:]
			continue
		}
		if _, _, isAssign := splitAssignment(tokens[0]); isAssign {
			tokens = tokens[1:]
			continue
		}
		break
	}
	if len(tokens) == 0 {
		return false
	}
	if tokenPointsToTarget(tokens[0], target, holders) {
		return true
	}
	if !shellInterpreters[unquote(tokens[0])] {
		return false
	}
	for _, arg := range tokens[1:] {
		if tokenPointsToTarget(arg, target, holders) {
			return true
		}
	}
	return false
}

func tokenPointsToTarget(token, target string, holders map[string]bool) bool {
	value := unquote(token)
	if strings.Contains(value, target) {
		return true
	}
	for name := range holders {
		if expandsVariable(value, name) {
			return true
		}
	}
	return false
}

func expandsVariable(value, name string) bool {
	if strings.Contains(value, "${"+name+"}") || strings.Contains(value, "${"+name+":") {
		return true
	}
	plain := "$" + name
	idx := strings.Index(value, plain)
	if idx < 0 {
		return false
	}
	after := idx + len(plain)
	if after >= len(value) {
		return true
	}
	r := rune(value[after])
	isLetter := (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || r == '_'
	isDigit := r >= '0' && r <= '9'
	return !isLetter && !isDigit
}

func unquote(token string) string {
	return strings.Trim(token, "\"'")
}
