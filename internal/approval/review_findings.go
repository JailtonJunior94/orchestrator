package approval

import (
	"regexp"
	"strconv"
	"strings"
)

const (
	ReviewFindingFile = "unspecified"
	ReviewFindingRule = "review-finding"
)

var (
	reviewFileLineRe     = regexp.MustCompile(`[\w./-]+\.[A-Za-z0-9]+:\d+`)
	findingPrefixRe      = regexp.MustCompile(`^[ \t]*(?:[-*+>][ \t]*)*(?:\*\*|__)?`)
	reviewHeadingRe      = regexp.MustCompile(`^#+\s`)
	severityFieldRe      = regexp.MustCompile(`(?i)^[-*>\s]*(?:severidade|severity)\s*:\s*(.*)$`)
	fileFieldRe          = regexp.MustCompile(`(?i)^[-*>\s]*(?:arquivo|file)\s*:\s*(.*)$`)
	lineFieldRe          = regexp.MustCompile(`(?i)^[-*>\s]*(?:linha|line)\s*:\s*(.*)$`)
	impactFieldRe        = regexp.MustCompile(`(?i)^[-*>\s]*(?:impacto|impact)\s*:\s*(.*)$`)
	bracketSeverityMarks = []struct {
		token    string
		severity Severity
	}{
		{"[critical]", SeverityCritical},
		{"[crítico]", SeverityCritical},
		{"[critico]", SeverityCritical},
		{"[hard]", SeverityHigh},
		{"[high]", SeverityHigh},
		{"[alta]", SeverityHigh},
		{"[alto]", SeverityHigh},
		{"[medium]", SeverityMedium},
		{"[important]", SeverityMedium},
		{"[importante]", SeverityMedium},
		{"[low]", SeverityLow},
		{"[suggestion]", SeverityLow},
		{"[sugestão]", SeverityLow},
		{"[sugestao]", SeverityLow},
	}
	fieldSeverityTokens = map[string]Severity{
		"critical":   SeverityCritical,
		"critico":    SeverityCritical,
		"crítico":    SeverityCritical,
		"hard":       SeverityHigh,
		"high":       SeverityHigh,
		"alta":       SeverityHigh,
		"alto":       SeverityHigh,
		"medium":     SeverityMedium,
		"media":      SeverityMedium,
		"média":      SeverityMedium,
		"important":  SeverityMedium,
		"importante": SeverityMedium,
		"low":        SeverityLow,
		"baixa":      SeverityLow,
		"suggestion": SeverityLow,
		"sugestao":   SeverityLow,
		"sugestão":   SeverityLow,
	}
)

type findingBlock struct {
	severity Severity
	file     string
	line     string
	impact   string
	open     bool
}

func (b findingBlock) finding() (Finding, bool) {
	if !b.open {
		return Finding{}, false
	}
	location := ReviewFindingFile
	if b.file != "" {
		location = b.file
		if number, err := strconv.Atoi(b.line); err == nil && number > 0 {
			location = b.file + ":" + b.line
		}
	}
	description := b.impact
	if description == "" {
		description = "review finding with severity " + b.severity.String()
	}
	finding, err := NewFinding(b.severity, location, ReviewFindingRule, description)
	if err != nil {
		return Finding{}, false
	}
	return finding, true
}

func ParseReviewFindings(rawText string) []Finding {
	var findings []Finding
	var block findingBlock

	flush := func() {
		if finding, ok := block.finding(); ok {
			findings = append(findings, finding)
		}
		block = findingBlock{}
	}

	for _, line := range strings.Split(rawText, "\n") {
		if reviewHeadingRe.MatchString(strings.TrimSpace(line)) {
			flush()
			continue
		}
		if severity, ok := bracketSeverity(line); ok {
			flush()
			if finding, ok := bracketFinding(severity, line); ok {
				findings = append(findings, finding)
			}
			continue
		}
		if match := severityFieldRe.FindStringSubmatch(line); match != nil {
			severity, ok := fieldSeverity(match[1])
			if !ok {
				continue
			}
			flush()
			block = findingBlock{severity: severity, open: true}
			continue
		}
		if !block.open {
			continue
		}
		if match := fileFieldRe.FindStringSubmatch(line); match != nil {
			block.file = strings.TrimSpace(match[1])
			continue
		}
		if match := lineFieldRe.FindStringSubmatch(line); match != nil {
			block.line = strings.TrimSpace(match[1])
			continue
		}
		if match := impactFieldRe.FindStringSubmatch(line); match != nil {
			block.impact = strings.TrimSpace(match[1])
		}
	}
	flush()
	return findings
}

func bracketSeverity(line string) (Severity, bool) {
	lower := strings.ToLower(line)
	lower = lower[len(findingPrefixRe.FindString(lower)):]
	for _, mark := range bracketSeverityMarks {
		if strings.HasPrefix(lower, mark.token) {
			return mark.severity, true
		}
	}
	return 0, false
}

func bracketFinding(severity Severity, line string) (Finding, bool) {
	location := ReviewFindingFile
	if reference := reviewFileLineRe.FindString(line); reference != "" {
		location = reference
	}
	finding, err := NewFinding(severity, location, ReviewFindingRule, strings.TrimSpace(line))
	if err != nil {
		return Finding{}, false
	}
	return finding, true
}

func fieldSeverity(raw string) (Severity, bool) {
	token := strings.ToLower(strings.TrimSpace(raw))
	token = strings.Trim(token, "`*_.,;:'\"")
	token = strings.TrimSpace(token)
	severity, ok := fieldSeverityTokens[token]
	return severity, ok
}
