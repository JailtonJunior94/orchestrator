package approval

import (
	"strings"

	"github.com/JailtonJunior94/ai-spec-harness/internal/reviewverdict"
)

type Translator struct{}

func NewTranslator() Translator {
	return Translator{}
}

func (t Translator) Translate(rawText string) Verdict {
	declarations := t.declarations(rawText)
	if len(declarations) == 0 {
		return VerdictBlocked
	}
	first := declarations[0]
	for _, declared := range declarations[1:] {
		if declared != first {
			return VerdictBlocked
		}
	}
	return first
}

func (t Translator) declarations(rawText string) []Verdict {
	var declared []Verdict
	var fence codeFence
	for line := range strings.Lines(rawText) {
		if fence.consume(line) {
			continue
		}
		if verdict, ok := t.fromLine(line); ok {
			declared = append(declared, verdict)
		}
	}
	return declared
}

type codeFence struct {
	marker byte
	length int
}

func (f *codeFence) consume(line string) bool {
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

func (t Translator) fromLine(line string) (Verdict, bool) {
	token, declared := reviewverdict.Parse(line)
	if !declared {
		return 0, false
	}
	return t.fromToken(token), true
}

func (t Translator) fromToken(token string) Verdict {
	switch token {
	case reviewverdict.ApprovedWithRemarks:
		return VerdictApprovedWithRemarks
	case reviewverdict.Approved:
		return VerdictApproved
	case reviewverdict.Rejected:
		return VerdictRejected
	default:
		return VerdictBlocked
	}
}
