package output

import (
	"regexp"
	"strings"
)

var (
	singleQuoteKeyRE   = regexp.MustCompile(`'([^']+)'\s*:`)
	singleQuoteValueRE = regexp.MustCompile(`:\s*'([^']*)'`)
	trailingCommaRE    = regexp.MustCompile(`,\s*([}\]])`)
)

// FixJSON applies conservative auto-fixes to malformed JSON.
func FixJSON(input string) string {
	fixed := strings.TrimSpace(input)
	fixed = strings.ReplaceAll(fixed, "\r\n", "\n")
	fixed = singleQuoteKeyRE.ReplaceAllString(fixed, `"$1":`)
	fixed = singleQuoteValueRE.ReplaceAllString(fixed, `: "$1"`)
	fixed = trailingCommaRE.ReplaceAllString(fixed, "$1")
	return fixed
}
