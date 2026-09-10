package approval

import (
	"fmt"
	"iter"
	"slices"
)

type CycleResult struct {
	reason   StopReason
	approved bool
	rounds   []Round
	proof    ApprovalProof
}

func NewApprovedResult(proof ApprovalProof, rounds []Round) (CycleResult, error) {
	if !proof.Valid() {
		return CycleResult{}, fmt.Errorf("%w: approved result requires a valid proof", ErrInsufficientProof)
	}
	return CycleResult{
		reason:   ReasonApproved,
		approved: true,
		rounds:   slices.Clone(rounds),
		proof:    proof,
	}, nil
}

func NewClosedResult(reason StopReason, rounds []Round) (CycleResult, error) {
	if _, err := NewStopReason(reason); err != nil {
		return CycleResult{}, err
	}
	if reason == ReasonApproved {
		return CycleResult{}, fmt.Errorf("%w: approved closure requires a proof", ErrInsufficientProof)
	}
	return CycleResult{
		reason:   reason,
		approved: false,
		rounds:   slices.Clone(rounds),
	}, nil
}

func (r CycleResult) Approved() bool {
	return r.approved
}

func (r CycleResult) Reason() StopReason {
	return r.reason
}

func (r CycleResult) Proof() ApprovalProof {
	return r.proof
}

func (r CycleResult) RoundCount() int {
	return len(r.rounds)
}

func (r CycleResult) Rounds() iter.Seq[Round] {
	return slices.Values(r.rounds)
}
