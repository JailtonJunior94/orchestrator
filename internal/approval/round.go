package approval

import (
	"fmt"
	"iter"
	"slices"
)

type Round struct {
	number      int
	verdict     Verdict
	findings    []Finding
	fingerprint Fingerprint
	concluded   bool
}

func NewRound(number int) (Round, error) {
	if number < 1 {
		return Round{}, fmt.Errorf("%w: number %d", ErrInvalidRound, number)
	}
	return Round{number: number}, nil
}

func (r Round) Complete(verdict Verdict, findings []Finding, fingerprint Fingerprint) (Round, error) {
	if r.concluded {
		return Round{}, fmt.Errorf("%w: round %d", ErrRoundAlreadyCompleted, r.number)
	}
	if _, err := NewVerdict(verdict); err != nil {
		return Round{}, err
	}
	return Round{
		number:      r.number,
		verdict:     verdict,
		findings:    slices.Clone(findings),
		fingerprint: fingerprint,
		concluded:   true,
	}, nil
}

func (r Round) Number() int {
	return r.number
}

func (r Round) Verdict() Verdict {
	return r.verdict
}

func (r Round) Fingerprint() Fingerprint {
	return r.fingerprint
}

func (r Round) Concluded() bool {
	return r.concluded
}

func (r Round) Findings() iter.Seq[Finding] {
	return slices.Values(r.findings)
}

func (r Round) CountBySeverity() map[Severity]int {
	count := make(map[Severity]int, 4)
	for _, f := range r.findings {
		count[f.severity]++
	}
	return count
}
