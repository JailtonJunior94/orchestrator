package approval

import "fmt"

type Verdict int

const (
	_ Verdict = iota
	VerdictApproved
	VerdictApprovedWithRemarks
	VerdictRejected
	VerdictBlocked
)

func NewVerdict(v Verdict) (Verdict, error) {
	if !v.valid() {
		return 0, fmt.Errorf("%w: value %d", ErrInvalidVerdict, int(v))
	}
	return v, nil
}

func (v Verdict) Approves() bool {
	return v == VerdictApproved
}

func (v Verdict) String() string {
	switch v {
	case VerdictApproved:
		return "APPROVED"
	case VerdictApprovedWithRemarks:
		return "APPROVED_WITH_REMARKS"
	case VerdictRejected:
		return "REJECTED"
	case VerdictBlocked:
		return "BLOCKED"
	default:
		return "INVALID_VERDICT"
	}
}

func (v Verdict) valid() bool {
	return v >= VerdictApproved && v <= VerdictBlocked
}
