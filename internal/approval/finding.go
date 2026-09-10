package approval

import (
	"fmt"
	"strings"
)

type Severity int

type BugLevel int

const (
	_ Severity = iota
	SeverityLow
	SeverityMedium
	SeverityHigh
	SeverityCritical
)

const (
	_ BugLevel = iota
	BugLevelMinor
	BugLevelMajor
	BugLevelCritical
)

type Finding struct {
	severity    Severity
	file        string
	rule        string
	description string
}

func NewFinding(severity Severity, file, rule, description string) (Finding, error) {
	if !severity.valid() {
		return Finding{}, fmt.Errorf("%w: value %d", ErrInvalidSeverity, int(severity))
	}
	f := strings.TrimSpace(file)
	if f == "" {
		return Finding{}, fmt.Errorf("%w: finding without file", ErrInvalidFinding)
	}
	r := strings.TrimSpace(rule)
	if r == "" {
		return Finding{}, fmt.Errorf("%w: finding without rule identifier", ErrInvalidFinding)
	}
	return Finding{
		severity:    severity,
		file:        f,
		rule:        r,
		description: strings.TrimSpace(description),
	}, nil
}

func (s Severity) String() string {
	switch s {
	case SeverityLow:
		return "low"
	case SeverityMedium:
		return "medium"
	case SeverityHigh:
		return "high"
	case SeverityCritical:
		return "critical"
	default:
		return "invalid_severity"
	}
}

func (s Severity) BugLevel() (BugLevel, error) {
	switch s {
	case SeverityCritical:
		return BugLevelCritical, nil
	case SeverityHigh:
		return BugLevelMajor, nil
	case SeverityMedium, SeverityLow:
		return BugLevelMinor, nil
	default:
		return 0, fmt.Errorf("%w: value %d", ErrInvalidSeverity, int(s))
	}
}

func (s Severity) Blocks() bool {
	return s == SeverityHigh || s == SeverityCritical
}

func (s Severity) valid() bool {
	return s >= SeverityLow && s <= SeverityCritical
}

func (n BugLevel) String() string {
	switch n {
	case BugLevelMinor:
		return "minor"
	case BugLevelMajor:
		return "major"
	case BugLevelCritical:
		return "critical"
	default:
		return "invalid_level"
	}
}

func (f Finding) Severity() Severity {
	return f.severity
}

func (f Finding) File() string {
	return f.file
}

func (f Finding) Rule() string {
	return f.rule
}

func (f Finding) Description() string {
	return f.description
}

func (f Finding) identityKey() string {
	return fmt.Sprintf("%d\x1f%s\x1f%s", int(f.severity), f.file, f.rule)
}
