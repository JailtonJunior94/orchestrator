package approval

import (
	"fmt"
	"slices"
)

type ApprovalProof struct {
	verdict     Verdict
	criteriaMap CriteriaMap
	remarks     []Finding
	valid       bool
}

func NewApprovalProof(verdict Verdict, criteriaMap CriteriaMap, findings []Finding) (ApprovalProof, error) {
	if !verdict.Closes(findings) {
		return ApprovalProof{}, fmt.Errorf("%w: verdict %q does not close the cycle", ErrInsufficientProof, verdict.String())
	}
	if !criteriaMap.Complete() {
		return ApprovalProof{}, fmt.Errorf("%w: 1:1 map incomplete (%d/%d criteria)", ErrInsufficientProof, criteriaMap.Bound(), criteriaMap.Total())
	}
	return ApprovalProof{verdict: verdict, criteriaMap: criteriaMap, remarks: slices.Clone(findings), valid: true}, nil
}

func (p ApprovalProof) Remarks() []Finding {
	return slices.Clone(p.remarks)
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
