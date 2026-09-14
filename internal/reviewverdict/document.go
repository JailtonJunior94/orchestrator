package reviewverdict

import "strings"

type fence struct {
	marker byte
	length int
}

func (f *fence) consume(line string) bool {
	marker, length, closable := fenceDelimiter(line)
	if f.length == 0 {
		if length == 0 {
			return false
		}
		f.marker = marker
		f.length = length
		return true
	}
	if length > 0 && closable && marker == f.marker && length >= f.length {
		f.marker = 0
		f.length = 0
	}
	return true
}

func fenceDelimiter(line string) (byte, int, bool) {
	trimmed := strings.TrimLeft(strings.TrimRight(line, "\r\n"), " \t")
	if trimmed == "" {
		return 0, 0, false
	}
	marker := trimmed[0]
	if marker != '`' && marker != '~' {
		return 0, 0, false
	}
	length := 0
	for length < len(trimmed) && trimmed[length] == marker {
		length++
	}
	if length < 3 {
		return 0, 0, false
	}
	info := strings.TrimSpace(trimmed[length:])
	if marker == '`' && strings.Contains(info, "`") {
		return 0, 0, false
	}
	return marker, length, info == ""
}

func Declarations(text string) []string {
	var declared []string
	var block fence
	for _, line := range strings.Split(text, "\n") {
		if block.consume(line) {
			continue
		}
		if verdict, ok := Parse(line); ok {
			declared = append(declared, verdict)
		}
	}
	return declared
}

func ParseDocument(text string) (string, bool) {
	declared := Declarations(text)
	if len(declared) == 0 {
		return "", false
	}
	first := declared[0]
	for _, verdict := range declared[1:] {
		if verdict != first {
			return Blocked, true
		}
	}
	return first, true
}
