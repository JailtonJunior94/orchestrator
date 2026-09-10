package approval

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/suite"
)

type ForbiddenTransitionSuite struct {
	suite.Suite
}

func TestForbiddenTransitionSuite(t *testing.T) {
	suite.Run(t, new(ForbiddenTransitionSuite))
}

func (s *ForbiddenTransitionSuite) TestForbiddenTransitions() {
	cases := []struct {
		name string
		from State
		to   State
	}{
		{"fix_cannot_approve", StateInFix, StateApproved},
		{"approved_is_terminal_to_review", StateApproved, StateInReview},
		{"approved_is_terminal_to_fix", StateApproved, StateInFix},
		{"approved_is_terminal_to_blocked", StateApproved, StateBlocked},
		{"blocked_does_not_resume_review", StateBlocked, StateInReview},
		{"blocked_does_not_resume_fix", StateBlocked, StateInFix},
		{"no_self_transition_in_review", StateInReview, StateInReview},
		{"no_self_transition_in_fix", StateInFix, StateInFix},
	}

	for _, tc := range cases {
		s.Run(tc.name, func() {
			machine := stateMachine{current: tc.from}
			err := machine.transition(tc.to)

			s.Require().Error(err)
			s.Require().ErrorIs(err, ErrForbiddenTransition)
			var typed ForbiddenTransitionError
			s.Require().True(errors.As(err, &typed))
			s.Equal(tc.from, typed.From)
			s.Equal(tc.to, typed.To)
			s.Equal(tc.from, machine.currentState())
		})
	}
}

func (s *ForbiddenTransitionSuite) TestAllowedTransitions() {
	cases := []struct {
		name string
		from State
		to   State
	}{
		{"review_to_approved", StateInReview, StateApproved},
		{"review_to_blocked", StateInReview, StateBlocked},
		{"review_to_fix", StateInReview, StateInFix},
		{"fix_to_review", StateInFix, StateInReview},
		{"fix_to_blocked", StateInFix, StateBlocked},
	}

	for _, tc := range cases {
		s.Run(tc.name, func() {
			machine := stateMachine{current: tc.from}
			s.Require().NoError(machine.transition(tc.to))
			s.Equal(tc.to, machine.currentState())
		})
	}
}

func (s *ForbiddenTransitionSuite) TestTerminalStates() {
	s.True(StateApproved.Terminal())
	s.True(StateBlocked.Terminal())
	s.False(StateInReview.Terminal())
	s.False(StateInFix.Terminal())
}
