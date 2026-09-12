package durable

import (
	"fmt"
	"regexp"
	"strings"
)

type SanitizationPattern struct {
	Name    string
	Pattern *regexp.Regexp
}

func NewSanitizationPattern(name string, expr string) (SanitizationPattern, error) {
	compiled, err := regexp.Compile(expr)
	if err != nil {
		return SanitizationPattern{}, fmt.Errorf("durable: invalid sanitization pattern %q: %w", name, err)
	}
	return SanitizationPattern{Name: name, Pattern: compiled}, nil
}

type SanitizationConfig struct {
	ExtraPatterns []SanitizationPattern
}

type Redaction struct {
	Pattern string
}

type SanitizationResult struct {
	Content    string
	Redactions []Redaction
}

var minimalSanitizationCatalog = []SanitizationPattern{
	{
		Name:    "pem_private_key",
		Pattern: regexp.MustCompile(`(?s)-----BEGIN (?:RSA |EC |OPENSSH |DSA )?PRIVATE KEY-----.*?-----END (?:RSA |EC |OPENSSH |DSA )?PRIVATE KEY-----`),
	},
	{
		Name:    "provider_token",
		Pattern: regexp.MustCompile(`\b(?:ghp_|gho_|sk-|AKIA)[A-Za-z0-9_\-]{10,}\b`),
	},
	{
		Name:    "jwt",
		Pattern: regexp.MustCompile(`\beyJ[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+\b`),
	},
	{
		Name:    "authorization_header",
		Pattern: regexp.MustCompile(`(?i)Authorization:\s*(?:Bearer|Basic)\s+\S+`),
	},
	{
		Name:    "dotenv_secret",
		Pattern: regexp.MustCompile(`(?im)^[A-Z][A-Z0-9_]*(?:SECRET|TOKEN|KEY|PASSWORD|PASS|CREDENTIAL)[A-Z0-9_]*=.+$`),
	},
	{
		Name:    "connection_string_credential",
		Pattern: regexp.MustCompile(`\b[a-zA-Z][a-zA-Z0-9+.\-]*://[^\s:/@]+:[^\s:/@]+@[^\s]+`),
	},
}

var redactionMarkerPattern = regexp.MustCompile(`\[REDACTED:[^\]]+\]`)

type SanitizationPolicy struct{}

var DefaultSanitizationPolicy = SanitizationPolicy{}

func (p SanitizationPolicy) Sanitize(content string, cfg SanitizationConfig) (SanitizationResult, error) {
	sanitized := content
	redactions := make([]Redaction, 0)

	for _, pattern := range p.catalog(cfg) {
		sanitized, redactions = p.applyPattern(sanitized, pattern, redactions)
	}

	if len(redactions) == 0 {
		return SanitizationResult{Content: sanitized, Redactions: redactions}, nil
	}

	remainder := redactionMarkerPattern.ReplaceAllString(sanitized, "")
	if strings.TrimSpace(remainder) == "" {
		return SanitizationResult{}, ErrSecretNotRedactable
	}

	return SanitizationResult{Content: sanitized, Redactions: redactions}, nil
}

func (p SanitizationPolicy) catalog(cfg SanitizationConfig) []SanitizationPattern {
	patterns := make([]SanitizationPattern, 0, len(minimalSanitizationCatalog)+len(cfg.ExtraPatterns))
	patterns = append(patterns, minimalSanitizationCatalog...)
	patterns = append(patterns, cfg.ExtraPatterns...)
	return patterns
}

func (p SanitizationPolicy) applyPattern(content string, pattern SanitizationPattern, redactions []Redaction) (string, []Redaction) {
	matches := pattern.Pattern.FindAllStringIndex(content, -1)
	if len(matches) == 0 {
		return content, redactions
	}

	var b strings.Builder
	cursor := 0
	for _, m := range matches {
		b.WriteString(content[cursor:m[0]])
		b.WriteString("[REDACTED:")
		b.WriteString(pattern.Name)
		b.WriteString("]")
		redactions = append(redactions, Redaction{Pattern: pattern.Name})
		cursor = m[1]
	}
	b.WriteString(content[cursor:])

	return b.String(), redactions
}
