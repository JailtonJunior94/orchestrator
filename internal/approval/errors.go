package approval

import (
	"errors"
	"fmt"
)

type InvalidMaxRoundsError struct {
	Value int
}

type ForbiddenTransitionError struct {
	From State
	To   State
}

var (
	ErrInvalidIdentity       = errors.New("approval: invalid identity")
	ErrInvalidVerdict        = errors.New("approval: verdict outside the closed set")
	ErrInvalidStopReason     = errors.New("approval: invalid stop reason")
	ErrInvalidSeverity       = errors.New("approval: invalid severity")
	ErrInvalidFinding        = errors.New("approval: invalid finding")
	ErrInvalidEvidence       = errors.New("approval: invalid evidence line")
	ErrInvalidCriterion      = errors.New("approval: invalid acceptance criterion")
	ErrInvalidCriteriaMap    = errors.New("approval: invalid criteria map")
	ErrInsufficientProof     = errors.New("approval: insufficient approval proof")
	ErrForbiddenTransition   = errors.New("approval: forbidden state transition")
	ErrRoundAlreadyCompleted = errors.New("approval: round already completed")
	ErrInvalidRound          = errors.New("approval: invalid round number")
	ErrInvalidState          = errors.New("approval: invalid state")
	ErrInvalidRequest        = errors.New("approval: invalid request")
	ErrInvalidCheckpoint     = errors.New("approval: invalid checkpoint")
	ErrCycleAlreadyStarted   = errors.New("approval: cycle already started")
	ErrCycleNotStarted       = errors.New("approval: cycle not started")
	ErrCycleClosed           = errors.New("approval: cycle already closed")
)

func (e InvalidMaxRoundsError) Error() string {
	return fmt.Sprintf("approval: max rounds %d invalid: minimum 1", e.Value)
}

func (e ForbiddenTransitionError) Error() string {
	return fmt.Sprintf("approval: forbidden transition %s -> %s", e.From, e.To)
}

func (e ForbiddenTransitionError) Is(target error) bool {
	return target == ErrForbiddenTransition
}
