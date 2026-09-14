package approval_test

import (
	"context"
	"slices"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"

	"github.com/JailtonJunior94/ai-spec-harness/internal/approval"
	"github.com/JailtonJunior94/ai-spec-harness/internal/approval/mocks"
)

type CycleMarkerFeedbackSuite struct {
	suite.Suite
	reviewer *mocks.Reviewer
	fixer    *mocks.Fixer
	repo     *mocks.Repository
}

func TestCycleMarkerFeedbackSuite(t *testing.T) {
	suite.Run(t, new(CycleMarkerFeedbackSuite))
}

func (s *CycleMarkerFeedbackSuite) SetupTest() {
	s.reviewer = mocks.NewReviewer(s.T())
	s.fixer = mocks.NewFixer(s.T())
	s.repo = mocks.NewRepository(s.T())
}

func (s *CycleMarkerFeedbackSuite) criterion() approval.AcceptanceCriterion {
	c, err := approval.NewAcceptanceCriterion("only criterion")
	s.Require().NoError(err)
	return c
}

func (s *CycleMarkerFeedbackSuite) completeMap() approval.CriteriaMap {
	c := s.criterion()
	evidence, err := approval.NewCommandEvidence("go test ./...", "ok")
	s.Require().NoError(err)
	criteriaMap, err := approval.NewCriteriaMap([]approval.AcceptanceCriterion{c})
	s.Require().NoError(err)
	criteriaMap, err = criteriaMap.WithEvidence(c, evidence)
	s.Require().NoError(err)
	return criteriaMap
}

func (s *CycleMarkerFeedbackSuite) newCycle() *approval.Cycle {
	task, err := approval.NewTaskIdentity("task-9.9")
	s.Require().NoError(err)
	agent, err := approval.NewAgentIdentity("claude")
	s.Require().NoError(err)
	policy, err := approval.NewApprovalPolicy(approval.WithMaxRounds(2))
	s.Require().NoError(err)
	cycle, err := approval.NewCycle(task, agent, policy,
		[]approval.AcceptanceCriterion{s.criterion()},
		s.reviewer, s.fixer, s.repo)
	s.Require().NoError(err)
	return cycle
}

func (s *CycleMarkerFeedbackSuite) expectRepository(delta approval.ReviewTarget) {
	checkpoint, err := approval.NewCheckpoint("sha-base")
	s.Require().NoError(err)
	s.repo.EXPECT().Checkpoint(mock.Anything).Return(checkpoint, nil).Maybe()
	s.repo.EXPECT().FullTarget(mock.Anything).Return(approval.NewReviewTarget("full diff"), nil).Maybe()
	s.repo.EXPECT().Delta(mock.Anything, mock.Anything).Return(delta, nil).Maybe()
}

func (s *CycleMarkerFeedbackSuite) TestCanonicalMarkersFeedTheFixerAndReachApproval() {
	s.expectRepository(approval.NewReviewTarget("changed"))

	rejected := "## Achados\n" +
		"[CRITICAL] internal/x/a.go:12 estado corrompido no handler\n" +
		"[HIGH] internal/x/b.go:7 validação de entrada ausente\n\n" +
		"verdict: REJECTED\n"
	approved := "## Achados\n\nSem achados.\n\nverdict: APPROVED\n"

	s.reviewer.EXPECT().Review(mock.Anything, mock.Anything).
		RunAndReturn(func(_ context.Context, request approval.ReviewRequest) (approval.ReviewerOutput, error) {
			if request.Round() == 1 {
				raw := rejected
				return approval.NewReviewerOutput(raw, approval.ParseReviewFindings(raw), s.completeMap()), nil
			}
			raw := approved
			return approval.NewReviewerOutput(raw, approval.ParseReviewFindings(raw), s.completeMap()), nil
		})
	s.fixer.EXPECT().Fix(mock.Anything, mock.MatchedBy(func(request approval.FixRequest) bool {
		return len(slices.Collect(request.Findings())) == 2
	})).Return(nil).Once()

	result, err := s.newCycle().Run(context.Background())

	s.Require().NoError(err)
	s.True(result.Approved())
	s.Equal(approval.ReasonApproved, result.Reason())
	s.Equal(2, result.RoundCount())
}

func (s *CycleMarkerFeedbackSuite) TestOutputWithoutMarkersStaysFailClosed() {
	s.expectRepository(approval.NewReviewTarget("changed"))

	raw := "## Achados\n\nO handler tem um problema serio, mas nenhuma severidade foi marcada.\n\nverdict: REJECTED\n"
	s.reviewer.EXPECT().Review(mock.Anything, mock.Anything).
		Return(approval.NewReviewerOutput(raw, approval.ParseReviewFindings(raw), s.completeMap()), nil).Once()

	cycle := s.newCycle()
	result, err := cycle.Run(context.Background())

	s.Require().NoError(err)
	s.False(result.Approved())
	s.Equal(approval.ReasonBlockedInput, result.Reason())
	s.Equal(approval.StateBlocked, cycle.State())
	s.fixer.AssertNotCalled(s.T(), "Fix", mock.Anything, mock.Anything)
}
