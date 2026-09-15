package approval

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/suite"
)

type ApprovalPolicySuite struct {
	suite.Suite
}

func TestApprovalPolicySuite(t *testing.T) {
	suite.Run(t, new(ApprovalPolicySuite))
}

func (s *ApprovalPolicySuite) TestDefaultMaxRounds() {
	policy, err := NewApprovalPolicy()
	s.Require().NoError(err)
	s.Equal(5, policy.MaxRounds())
}

func (s *ApprovalPolicySuite) TestRejectsMaxRoundsBelowOne() {
	for _, ceiling := range []int{0, -1, -10} {
		_, err := NewApprovalPolicy(WithMaxRounds(ceiling))
		s.Require().Error(err)
		var typed InvalidMaxRoundsError
		s.Require().True(errors.As(err, &typed))
		s.Equal(ceiling, typed.Value)
	}
}

func (s *ApprovalPolicySuite) TestDecideChecksOrder() {
	calc := NewFingerprintCalculator()
	repeated := calc.Compute([]Finding{mustFinding(s.T(), SeverityHigh, "a.go", "R1")})
	distinct := calc.Compute([]Finding{mustFinding(s.T(), SeverityLow, "b.go", "R2")})

	policy, err := NewApprovalPolicy(WithMaxRounds(1))
	s.Require().NoError(err)

	s.Run("blocked_verdict_wins_first", func() {
		reason, stop := policy.Decide(StopDecision{
			Verdict:             VerdictBlocked,
			CurrentRound:        1,
			CurrentFingerprint:  repeated,
			PreviousFingerprint: repeated,
			Changed:             false,
		})
		s.True(stop)
		s.Equal(ReasonBlockedInput, reason)
	})

	s.Run("repeated_fingerprint_before_ceiling", func() {
		reason, stop := policy.Decide(StopDecision{
			Verdict:             VerdictRejected,
			CurrentRound:        1,
			CurrentFingerprint:  repeated,
			PreviousFingerprint: repeated,
			Changed:             false,
		})
		s.True(stop)
		s.Equal(ReasonNoConvergence, reason)
	})

	s.Run("empty_diff_before_ceiling", func() {
		reason, stop := policy.Decide(StopDecision{
			Verdict:             VerdictRejected,
			CurrentRound:        1,
			CurrentFingerprint:  distinct,
			PreviousFingerprint: repeated,
			Changed:             false,
		})
		s.True(stop)
		s.Equal(ReasonEmptyDiff, reason)
	})

	s.Run("ceiling_last", func() {
		reason, stop := policy.Decide(StopDecision{
			Verdict:             VerdictRejected,
			CurrentRound:        1,
			CurrentFingerprint:  distinct,
			PreviousFingerprint: repeated,
			Changed:             true,
		})
		s.True(stop)
		s.Equal(ReasonMaxRounds, reason)
	})

	s.Run("no_stop_when_room_remains", func() {
		roomy, err := NewApprovalPolicy(WithMaxRounds(5))
		s.Require().NoError(err)
		_, stop := roomy.Decide(StopDecision{
			Verdict:             VerdictRejected,
			CurrentRound:        1,
			CurrentFingerprint:  distinct,
			PreviousFingerprint: repeated,
			Changed:             true,
		})
		s.False(stop)
	})
}
