package telemetry

import "strings"

var sensitiveFieldKeywords = []string{
	"token",
	"secret",
	"password",
	"passwd",
	"credential",
	"api_key",
	"apikey",
	"authorization",
	"cookie",
	"session_id",
	"private_key",
	"access_key",
}

var rf44MetricLogKeyAllowlist = map[string]bool{
	"tokens_in":  true,
	"tokens_out": true,
}

func FindSensitiveField(line string) (string, bool) {
	for _, part := range strings.Fields(line) {
		key, _, hasEq := strings.Cut(part, "=")
		if !hasEq {
			continue
		}
		lower := strings.ToLower(key)
		if rf44MetricLogKeyAllowlist[lower] {
			continue
		}
		for _, kw := range sensitiveFieldKeywords {
			if strings.Contains(lower, kw) {
				return key, true
			}
		}
	}
	return "", false
}
