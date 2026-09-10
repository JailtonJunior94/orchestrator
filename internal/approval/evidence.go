package approval

import (
	"fmt"
	"iter"
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
}

type criterionBinding struct {
	criterion    AcceptanceCriterion
	evidence     EvidenceLine
	unverifiable bool
}

type diffReference struct {
	raw string
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
	cut := strings.LastIndex(d.raw, ":")
	if cut <= 0 || cut == len(d.raw)-1 {
		return false
	}
	file, line := d.raw[:cut], d.raw[cut+1:]
	if strings.TrimSpace(file) == "" {
		return false
	}
	for _, r := range line {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
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
	return CriteriaMap{bindings: next}, nil
}
