package approval

import "fmt"

type ApprovalProof struct {
	verdict     Verdict
	criteriaMap CriteriaMap
	valid       bool
}

func NewApprovalProof(verdict Verdict, criteriaMap CriteriaMap) (ApprovalProof, error) {
	if !verdict.Approves() {
		return ApprovalProof{}, fmt.Errorf("%w: verdict %q does not approve", ErrInsufficientProof, verdict.String())
	}
	if !criteriaMap.Complete() {
		return ApprovalProof{}, fmt.Errorf("%w: 1:1 map incomplete (%d/%d criteria)", ErrInsufficientProof, criteriaMap.Bound(), criteriaMap.Total())
	}
	return ApprovalProof{verdict: verdict, criteriaMap: criteriaMap, valid: true}, nil
}

func (p ApprovalProof) Verdict() Verdict {
	return p.verdict
}

func (p ApprovalProof) CriteriaMap() CriteriaMap {
	return p.criteriaMap
}

func (p ApprovalProof) Valid() bool {
	return p.valid
}
