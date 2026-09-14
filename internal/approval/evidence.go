package approval

import (
	"fmt"
	"iter"
	"regexp"
	"slices"
	"strconv"
	"strings"
)

type EvidenceForm int

const (
	_ EvidenceForm = iota
	EvidenceFormCommandOutput
	EvidenceFormFileLineInDiff
	EvidenceFormTestResult
)

type EvidenceLine struct {
	form      EvidenceForm
	reference string
	record    string
	valid     bool
}

type AcceptanceCriterion struct {
	description string
}

type CriteriaMap struct {
	bindings []criterionBinding
	findings []Finding
}

type criterionBinding struct {
	criterion    AcceptanceCriterion
	evidence     EvidenceLine
	unverifiable bool
}

type diffReference struct {
	raw string
}

type lineRange struct {
	start int
	end   int
}

func NewCommandEvidence(command, recordedOutput string) (EvidenceLine, error) {
	cmd := strings.TrimSpace(command)
	out := strings.TrimSpace(recordedOutput)
	if cmd == "" || out == "" {
		return EvidenceLine{}, fmt.Errorf("%w: command form needs a command and a recorded output", ErrInvalidEvidence)
	}
	return EvidenceLine{form: EvidenceFormCommandOutput, reference: cmd, record: out, valid: true}, nil
}

func NewFileLineEvidence(reference string, presentInDiff bool) (EvidenceLine, error) {
	ref := strings.TrimSpace(reference)
	if !(diffReference{raw: ref}).wellFormed() {
		return EvidenceLine{}, fmt.Errorf("%w: reference %q is not in file:line form", ErrInvalidEvidence, reference)
	}
	if !presentInDiff {
		return EvidenceLine{}, fmt.Errorf("%w: reference %q absent from the reviewed diff", ErrInvalidEvidence, ref)
	}
	return EvidenceLine{form: EvidenceFormFileLineInDiff, reference: ref, record: "present in diff", valid: true}, nil
}

func NewTestEvidence(testName, recordedResult string) (EvidenceLine, error) {
	name := strings.TrimSpace(testName)
	res := strings.TrimSpace(recordedResult)
	if name == "" || res == "" {
		return EvidenceLine{}, fmt.Errorf("%w: test form needs a name and a recorded result", ErrInvalidEvidence)
	}
	return EvidenceLine{form: EvidenceFormTestResult, reference: name, record: res, valid: true}, nil
}

func NewAcceptanceCriterion(description string) (AcceptanceCriterion, error) {
	d := strings.TrimSpace(description)
	if d == "" {
		return AcceptanceCriterion{}, fmt.Errorf("%w: empty description", ErrInvalidCriterion)
	}
	return AcceptanceCriterion{description: d}, nil
}

func NewCriteriaMap(criteria []AcceptanceCriterion) (CriteriaMap, error) {
	if len(criteria) == 0 {
		return CriteriaMap{}, fmt.Errorf("%w: no acceptance criteria", ErrInvalidCriteriaMap)
	}
	seen := make(map[string]struct{}, len(criteria))
	bindings := make([]criterionBinding, 0, len(criteria))
	for _, c := range criteria {
		if c.description == "" {
			return CriteriaMap{}, fmt.Errorf("%w: uninitialized criterion", ErrInvalidCriteriaMap)
		}
		if _, dup := seen[c.description]; dup {
			return CriteriaMap{}, fmt.Errorf("%w: duplicate criterion %q", ErrInvalidCriteriaMap, c.description)
		}
		seen[c.description] = struct{}{}
		bindings = append(bindings, criterionBinding{criterion: c})
	}
	return CriteriaMap{bindings: bindings}, nil
}

func (d diffReference) wellFormed() bool {
	_, _, ok := d.split()
	return ok
}

func (d diffReference) split() (string, int, bool) {
	cut := strings.LastIndex(d.raw, ":")
	if cut <= 0 || cut == len(d.raw)-1 {
		return "", 0, false
	}
	file, line := strings.TrimSpace(d.raw[:cut]), d.raw[cut+1:]
	if file == "" {
		return "", 0, false
	}
	number, err := strconv.Atoi(line)
	if err != nil || number < 1 {
		return "", 0, false
	}
	return file, number, true
}

func (l EvidenceLine) Form() EvidenceForm {
	return l.form
}

func (l EvidenceLine) Reference() string {
	return l.reference
}

func (l EvidenceLine) Record() string {
	return l.record
}

func (l EvidenceLine) Valid() bool {
	return l.valid
}

func (c AcceptanceCriterion) Description() string {
	return c.description
}

func (c AcceptanceCriterion) Zero() bool {
	return c.description == ""
}

func (m CriteriaMap) WithEvidence(criterion AcceptanceCriterion, evidence EvidenceLine) (CriteriaMap, error) {
	if !evidence.Valid() {
		return CriteriaMap{}, fmt.Errorf("%w: invalid evidence for %q", ErrInvalidEvidence, criterion.description)
	}
	return m.update(criterion, func(b *criterionBinding) {
		b.evidence = evidence
		b.unverifiable = false
	})
}

func (m CriteriaMap) AsUnverifiable(criterion AcceptanceCriterion) (CriteriaMap, error) {
	return m.update(criterion, func(b *criterionBinding) {
		b.evidence = EvidenceLine{}
		b.unverifiable = true
	})
}

func (m CriteriaMap) FindingsList() []Finding {
	return slices.Clone(m.findings)
}

func (m CriteriaMap) withFindings(findings []Finding) CriteriaMap {
	return CriteriaMap{bindings: m.bindings, findings: slices.Clone(findings)}
}

func (m CriteriaMap) Complete() bool {
	return m.Total() > 0 && m.Bound() == m.Total()
}

func (m CriteriaMap) Bound() int {
	n := 0
	for _, b := range m.bindings {
		if b.evidence.Valid() && !b.unverifiable {
			n++
		}
	}
	return n
}

func (m CriteriaMap) Total() int {
	return len(m.bindings)
}

func (m CriteriaMap) Criteria() iter.Seq[AcceptanceCriterion] {
	return func(yield func(AcceptanceCriterion) bool) {
		for _, b := range m.bindings {
			if !yield(b.criterion) {
				return
			}
		}
	}
}

func (m CriteriaMap) Unverifiable() iter.Seq[AcceptanceCriterion] {
	return func(yield func(AcceptanceCriterion) bool) {
		for _, b := range m.bindings {
			if b.unverifiable && !yield(b.criterion) {
				return
			}
		}
	}
}

func (m CriteriaMap) Pending() iter.Seq[AcceptanceCriterion] {
	return func(yield func(AcceptanceCriterion) bool) {
		for _, b := range m.bindings {
			if (b.unverifiable || !b.evidence.Valid()) && !yield(b.criterion) {
				return
			}
		}
	}
}

func (m CriteriaMap) update(criterion AcceptanceCriterion, mutate func(*criterionBinding)) (CriteriaMap, error) {
	if criterion.Zero() {
		return CriteriaMap{}, fmt.Errorf("%w: uninitialized criterion", ErrInvalidCriterion)
	}
	next := make([]criterionBinding, len(m.bindings))
	copy(next, m.bindings)
	found := false
	for i := range next {
		if next[i].criterion.description == criterion.description {
			mutate(&next[i])
			found = true
			break
		}
	}
	if !found {
		return CriteriaMap{}, fmt.Errorf("%w: criterion %q absent from the map", ErrInvalidCriteriaMap, criterion.description)
	}
	return CriteriaMap{bindings: next, findings: slices.Clone(m.findings)}, nil
}

const (
	criteriaFindingFile          = "acceptance-criteria"
	criteriaFindingRuleUnmet     = "RF-50/acceptance-criterion-unmet"
	criteriaFindingRuleUnverific = "RF-49/acceptance-criterion-unverifiable"
	criteriaFindingRuleForm      = "RF-48/evidence-form"
	criteriaFindingRuleMarker    = "RF-47/criterion-marker"
	criteriaFindingRuleOneToOne  = "RF-51/criteria-map-one-to-one"
	criteriaFindingRuleMissing   = "RF-47/criteria-map-missing"
)

type criterionStatus int

const (
	statusMet criterionStatus = iota + 1
	statusUnmet
	statusUnverifiable
)

type criterionEntry struct {
	status   criterionStatus
	evidence string
}

var (
	criteriaSectionRe = regexp.MustCompile(`(?i)^#+\s*mapa de crit(?:e|\x{00e9})rios de aceite`)
	testNameRe        = regexp.MustCompile(`^(?:Test|Benchmark|Example)[A-Za-z0-9_/]*$`)
	testResultRe      = regexp.MustCompile(`^(?i:pass|fail)$`)
	commandRe         = regexp.MustCompile(`^(?:go\s+(?:test|build|vet|run)\s+\S+|gotestsum\s+\S+|golangci-lint\s+run\b|gofmt\s+\S+|bash\s+\S+|sh\s+\S+|make\s+[A-Za-z0-9][A-Za-z0-9_.\-]*|grep\s+\S+|rg\s+\S+|python3?\s+\S+|pytest\s+\S+|npm\s+(?:run\s+)?\S+|pnpm\s+\S+|yarn\s+\S+|cargo\s+\S+|dotnet\s+\S+|git\s+(?:diff|log|show|status|rev-parse|grep|blame)\b|shasum\s+\S+|sha256sum\s+\S+|\./\S+)`)
	canonicalRecordRe = regexp.MustCompile(`^(?i:pass(?:ed)?|fail(?:ed)?|exit\s+\d+)$`)
	trivialRecordRe   = regexp.MustCompile(`^(?i:ok|okay|done|feito|pronto|sim|yes|no|nao|certo|tudo certo|tudo ok|aprovado|abc|talvez|maybe|n/?a|[-._]+)$`)
	recordSignalRe    = regexp.MustCompile(`(?:[0-9]|[A-Za-z0-9_-]+/[A-Za-z0-9_./-]+|[A-Za-z0-9_-]+\.(?:go|py|ts|tsx|js|jsx|cs|rs|java|rb|sh|sql|md|ya?ml|json|toml)\b|(?:Test|Benchmark|Example)[A-Za-z0-9_]+)`)
	diffFileHeaderRe  = regexp.MustCompile(`^(?:\+\+\+|---)\s+(?:[ab]/)?(\S+)`)
	diffHunkRe        = regexp.MustCompile(`^@@\s+-(\d+)(?:,(\d+))?\s+\+(\d+)(?:,(\d+))?\s+@@`)
	accentFolder      = strings.NewReplacer(
		"\u00e3", "a", "\u00e1", "a", "\u00e0", "a", "\u00e2", "a",
		"\u00e9", "e", "\u00ea", "e",
		"\u00ed", "i",
		"\u00f3", "o", "\u00f5", "o", "\u00f4", "o",
		"\u00fa", "u",
		"\u00e7", "c",
	)
)

func normalizeMarker(raw string) string {
	folded := accentFolder.Replace(strings.ToLower(strings.TrimSpace(raw)))
	return strings.Join(strings.Fields(folded), " ")
}

func statusFromMarker(marker string) (criterionStatus, bool) {
	switch normalizeMarker(marker) {
	case "atendido":
		return statusMet, true
	case "nao atendido":
		return statusUnmet, true
	case "nao verificavel":
		return statusUnverifiable, true
	default:
		return 0, false
	}
}

func ParseCriteriaMap(raw string, request ReviewRequest) (CriteriaMap, error) {
	criteria := make([]AcceptanceCriterion, 0)
	for criterion := range request.Criteria() {
		criteria = append(criteria, criterion)
	}
	criteriaMap, err := NewCriteriaMap(criteria)
	if err != nil {
		return CriteriaMap{}, err
	}
	entries, inSection, findings, err := parseCriteriaEntries(raw)
	if err != nil {
		return CriteriaMap{}, err
	}
	if !inSection {
		finding, findingErr := criterionFinding(criteriaFindingRuleMissing, "review report without the acceptance criteria map section")
		if findingErr != nil {
			return CriteriaMap{}, findingErr
		}
		return criteriaMap.withFindings(append(findings, finding)), nil
	}
	matched := 0
	for _, criterion := range criteria {
		entry, found := entries[criterion.Description()]
		if !found {
			finding, findingErr := criterionFinding(criteriaFindingRuleOneToOne, "missing evidence for acceptance criterion: "+criterion.Description())
			if findingErr != nil {
				return CriteriaMap{}, findingErr
			}
			findings = append(findings, finding)
			continue
		}
		matched++
		criteriaMap, findings, err = bindCriterion(criteriaMap, findings, criterion, entry, request.Target().String())
		if err != nil {
			return CriteriaMap{}, err
		}
	}
	if len(entries) != matched {
		finding, findingErr := criterionFinding(criteriaFindingRuleOneToOne, "criteria evidence map is not one-to-one")
		if findingErr != nil {
			return CriteriaMap{}, findingErr
		}
		findings = append(findings, finding)
	}
	return criteriaMap.withFindings(findings), nil
}

func parseCriteriaEntries(raw string) (map[string]criterionEntry, bool, []Finding, error) {
	entries := make(map[string]criterionEntry)
	var findings []Finding
	inSection := false
	for _, line := range strings.Split(raw, "\n") {
		trimmed := strings.TrimSpace(line)
		if criteriaSectionRe.MatchString(trimmed) {
			inSection = true
			continue
		}
		if inSection && strings.HasPrefix(trimmed, "## ") {
			break
		}
		if !inSection || !strings.HasPrefix(trimmed, "- [") {
			continue
		}
		closing := strings.Index(trimmed, "] ")
		if closing < 0 {
			continue
		}
		status, known := statusFromMarker(trimmed[3:closing])
		if !known {
			finding, findingErr := criterionFinding(criteriaFindingRuleMarker, "unknown criterion marker in "+trimmed)
			if findingErr != nil {
				return nil, false, nil, findingErr
			}
			findings = append(findings, finding)
			continue
		}
		criterionText, evidenceText := splitCriterionEntry(trimmed[closing+2:])
		if criterionText == "" {
			finding, findingErr := criterionFinding(criteriaFindingRuleForm, "criterion entry without description in "+trimmed)
			if findingErr != nil {
				return nil, false, nil, findingErr
			}
			findings = append(findings, finding)
			continue
		}
		if status == statusMet && evidenceText == "" {
			finding, findingErr := criterionFinding(criteriaFindingRuleForm, "criterion marked as met without an evidence line: "+criterionText)
			if findingErr != nil {
				return nil, false, nil, findingErr
			}
			findings = append(findings, finding)
			status = statusUnmet
		}
		if _, duplicate := entries[criterionText]; duplicate {
			finding, findingErr := criterionFinding(criteriaFindingRuleOneToOne, "duplicate criterion in the evidence map: "+criterionText)
			if findingErr != nil {
				return nil, false, nil, findingErr
			}
			findings = append(findings, finding)
			continue
		}
		entries[criterionText] = criterionEntry{status: status, evidence: evidenceText}
	}
	return entries, inSection, findings, nil
}

func splitCriterionEntry(rest string) (string, string) {
	criterionText, evidenceText, found := strings.Cut(rest, " -> ")
	if !found {
		return strings.TrimSpace(rest), ""
	}
	return strings.TrimSpace(criterionText), strings.TrimSpace(evidenceText)
}

func bindCriterion(criteriaMap CriteriaMap, findings []Finding, criterion AcceptanceCriterion, entry criterionEntry, target string) (CriteriaMap, []Finding, error) {
	switch entry.status {
	case statusMet:
		evidence, evidenceErr := evidenceFromText(entry.evidence, target)
		if evidenceErr != nil {
			finding, findingErr := criterionFinding(criteriaFindingRuleForm, evidenceErr.Error()+" (criterion: "+criterion.Description()+")")
			if findingErr != nil {
				return CriteriaMap{}, nil, findingErr
			}
			return criteriaMap, append(findings, finding), nil
		}
		bound, err := criteriaMap.WithEvidence(criterion, evidence)
		if err != nil {
			return CriteriaMap{}, nil, err
		}
		return bound, findings, nil
	case statusUnverifiable:
		bound, err := criteriaMap.AsUnverifiable(criterion)
		if err != nil {
			return CriteriaMap{}, nil, err
		}
		finding, findingErr := criterionFinding(criteriaFindingRuleUnverific, "unverifiable acceptance criterion: "+criterion.Description())
		if findingErr != nil {
			return CriteriaMap{}, nil, findingErr
		}
		return bound, append(findings, finding), nil
	default:
		finding, findingErr := criterionFinding(criteriaFindingRuleUnmet, "unmet acceptance criterion: "+criterion.Description())
		if findingErr != nil {
			return CriteriaMap{}, nil, findingErr
		}
		return criteriaMap, append(findings, finding), nil
	}
}

func criterionFinding(rule, description string) (Finding, error) {
	return NewFinding(SeverityHigh, criteriaFindingFile, rule, description)
}

func evidenceFromText(text, target string) (EvidenceLine, error) {
	value := strings.TrimSpace(text)
	if (diffReference{raw: value}).wellFormed() {
		return NewFileLineEvidence(value, referenceInDiff(target, value))
	}
	name, record, found := strings.Cut(value, " -> ")
	if !found {
		return EvidenceLine{}, fmt.Errorf("%w: evidence %q has no recorded result", ErrInvalidEvidence, value)
	}
	name = strings.TrimSpace(name)
	record = strings.TrimSpace(record)
	if testNameRe.MatchString(name) {
		if !testResultRe.MatchString(record) {
			return EvidenceLine{}, fmt.Errorf("%w: test evidence %q lacks a canonical pass/fail result", ErrInvalidEvidence, value)
		}
		return NewTestEvidence(name, record)
	}
	if !commandRe.MatchString(name) {
		return EvidenceLine{}, fmt.Errorf("%w: evidence %q is not a runnable command, a file:line reference, nor a test result", ErrInvalidEvidence, value)
	}
	if !substantiveRecord(record) {
		return EvidenceLine{}, fmt.Errorf("%w: evidence %q records no meaningful command output", ErrInvalidEvidence, value)
	}
	return NewCommandEvidence(name, record)
}

func substantiveRecord(record string) bool {
	trimmed := strings.TrimSpace(record)
	if trimmed == "" {
		return false
	}
	if canonicalRecordRe.MatchString(trimmed) {
		return true
	}
	if trivialRecordRe.MatchString(trimmed) {
		return false
	}
	if len(strings.Fields(trimmed)) < 2 {
		return false
	}
	return recordSignalRe.MatchString(trimmed)
}

func referenceInDiff(target, reference string) bool {
	file, line, ok := (diffReference{raw: reference}).split()
	if !ok {
		return false
	}
	touched, structured := indexDiffRanges(target)
	if !structured {
		return false
	}
	for name, ranges := range touched {
		if !sameDiffFile(name, file) {
			continue
		}
		for _, r := range ranges {
			if line >= r.start && line <= r.end {
				return true
			}
		}
	}
	return false
}

func indexDiffRanges(target string) (map[string][]lineRange, bool) {
	touched := make(map[string][]lineRange)
	current := ""
	structured := false
	for _, raw := range strings.Split(target, "\n") {
		line := strings.TrimRight(raw, "\r")
		if header := diffFileHeaderRe.FindStringSubmatch(line); header != nil {
			structured = true
			if header[1] == "/dev/null" {
				continue
			}
			current = header[1]
			if _, seen := touched[current]; !seen {
				touched[current] = nil
			}
			continue
		}
		hunk := diffHunkRe.FindStringSubmatch(line)
		if hunk == nil || current == "" {
			continue
		}
		structured = true
		touched[current] = append(touched[current], hunkRanges(hunk)...)
	}
	return touched, structured
}

func hunkRanges(hunk []string) []lineRange {
	var ranges []lineRange
	if r, ok := rangeFrom(hunk[1], hunk[2]); ok {
		ranges = append(ranges, r)
	}
	if r, ok := rangeFrom(hunk[3], hunk[4]); ok {
		ranges = append(ranges, r)
	}
	return ranges
}

func rangeFrom(startText, countText string) (lineRange, bool) {
	start, err := strconv.Atoi(startText)
	if err != nil || start < 1 {
		return lineRange{}, false
	}
	count := 1
	if countText != "" {
		parsed, parseErr := strconv.Atoi(countText)
		if parseErr != nil {
			return lineRange{}, false
		}
		count = parsed
	}
	if count < 1 {
		return lineRange{}, false
	}
	return lineRange{start: start, end: start + count - 1}, true
}

func sameDiffFile(diffPath, reference string) bool {
	if diffPath == reference {
		return true
	}
	return strings.HasSuffix(diffPath, "/"+reference) || strings.HasSuffix(reference, "/"+diffPath)
}
