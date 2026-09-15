package durable_test

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/memory/durable"
)

type HandoffLeaseSuite struct {
	suite.Suite
}

func TestHandoffLeaseSuite(t *testing.T) {
	suite.Run(t, new(HandoffLeaseSuite))
}

func (s *HandoffLeaseSuite) TestLeaseOutcomeString() {
	scenarios := []struct {
		name    string
		outcome durable.LeaseOutcome
		want    string
	}{
		{name: "should render granted", outcome: durable.LeaseOutcomeGranted, want: "granted"},
		{name: "should render refused", outcome: durable.LeaseOutcomeRefused, want: "refused"},
		{name: "should render transferred", outcome: durable.LeaseOutcomeTransferred, want: "transferred"},
		{name: "should render renewed", outcome: durable.LeaseOutcomeRenewed, want: "renewed"},
		{name: "should render empty for undefined", outcome: durable.LeaseOutcomeUndefined, want: ""},
	}

	for _, sc := range scenarios {
		s.Run(sc.name, func() {
			s.Equal(sc.want, sc.outcome.String())
		})
	}
}

func (s *HandoffLeaseSuite) TestTransferReasonString() {
	scenarios := []struct {
		name   string
		reason durable.TransferReason
		want   string
	}{
		{name: "should render deadline expired", reason: durable.TransferReasonDeadlineExpired, want: "deadline_expired"},
		{name: "should render owner dead", reason: durable.TransferReasonOwnerDead, want: "owner_dead"},
		{name: "should render empty for none", reason: durable.TransferReasonNone, want: ""},
	}

	for _, sc := range scenarios {
		s.Run(sc.name, func() {
			s.Equal(sc.want, sc.reason.String())
		})
	}
}
