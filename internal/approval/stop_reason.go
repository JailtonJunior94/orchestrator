package approval

import "fmt"

type StopReason int

const (
	_ StopReason = iota
	ReasonApproved
	ReasonMaxRounds
	ReasonNoConvergence
	ReasonEmptyDiff
	ReasonBlockedInput
)

func NewStopReason(r StopReason) (StopReason, error) {
	if !r.valid() {
		return 0, fmt.Errorf("%w: value %d", ErrInvalidStopReason, int(r))
	}
	return r, nil
}

func (r StopReason) String() string {
	switch r {
	case ReasonApproved:
		return "approved"
	case ReasonMaxRounds:
		return "max_rounds"
	case ReasonNoConvergence:
		return "no_convergence"
	case ReasonEmptyDiff:
		return "empty_diff"
	case ReasonBlockedInput:
		return "blocked_input"
	default:
		return "invalid_reason"
	}
}

func (r StopReason) valid() bool {
	return r >= ReasonApproved && r <= ReasonBlockedInput
}
