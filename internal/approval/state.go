package approval

import "slices"

type State int

const (
	_ State = iota
	StateInReview
	StateInFix
	StateApproved
	StateBlocked
)

var allowedTransitions = map[State][]State{
	StateInReview: {StateApproved, StateBlocked, StateInFix},
	StateInFix:    {StateInReview, StateBlocked},
	StateApproved: nil,
	StateBlocked:  nil,
}

type stateMachine struct {
	current State
}

func NewState(s State) (State, error) {
	if s < StateInReview || s > StateBlocked {
		return 0, ErrInvalidState
	}
	return s, nil
}

func (s State) String() string {
	switch s {
	case StateInReview:
		return "in_review"
	case StateInFix:
		return "in_fix"
	case StateApproved:
		return "approved"
	case StateBlocked:
		return "blocked"
	default:
		return "invalid_state"
	}
}

func (s State) Terminal() bool {
	targets, known := allowedTransitions[s]
	return known && len(targets) == 0
}

func (m *stateMachine) currentState() State {
	return m.current
}

func (m *stateMachine) transition(target State) error {
	if _, err := NewState(target); err != nil {
		return ForbiddenTransitionError{From: m.current, To: target}
	}
	if !slices.Contains(allowedTransitions[m.current], target) {
		return ForbiddenTransitionError{From: m.current, To: target}
	}
	m.current = target
	return nil
}
