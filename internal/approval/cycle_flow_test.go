package approval_test

import (
	"context"
	"strconv"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"

	"github.com/JailtonJunior94/ai-spec-harness/internal/approval"
	"github.com/JailtonJunior94/ai-spec-harness/internal/approval/mocks"
)

type CycleFlowSuite struct {
	suite.Suite
	reviewer *mocks.Reviewer
	fixer    *mocks.Fixer
	repo     *mocks.Repository
}

func TestCycleFlowSuite(t *testing.T) {
	suite.Run(t, new(CycleFlowSuite))
}

func (s *CycleFlowSuite) SetupTest() {
	s.reviewer = mocks.NewReviewer(s.T())
	s.fixer = mocks.NewFixer(s.T())
	s.repo = mocks.NewRepository(s.T())
}

func (s *CycleFlowSuite) criterion(desc string) approval.AcceptanceCriterion {
	c, err := approval.NewAcceptanceCriterion(desc)
	s.Require().NoError(err)
	return c
}

func (s *CycleFlowSuite) completeMap() approval.CriteriaMap {
	c := s.criterion("only criterion")
	ev, err := approval.NewCommandEvidence("go test ./...", "ok")
	s.Require().NoError(err)
	m, err := approval.NewCriteriaMap([]approval.AcceptanceCriterion{c})
	s.Require().NoError(err)
	m, err = m.WithEvidence(c, ev)
	s.Require().NoError(err)
	return m
}

func (s *CycleFlowSuite) finding(rule string) approval.Finding {
	f, err := approval.NewFinding(approval.SeverityHigh, "a.go", rule, "")
	s.Require().NoError(err)
	return f
}

func (s *CycleFlowSuite) newCycle() *approval.Cycle {
	task, err := approval.NewTaskIdentity("task-2.0")
	s.Require().NoError(err)
	agent, err := approval.NewAgentIdentity("claude")
	s.Require().NoError(err)
	policy, err := approval.NewApprovalPolicy(approval.WithMaxRounds(2))
	s.Require().NoError(err)
	cycle, err := approval.NewCycle(task, agent, policy,
		[]approval.AcceptanceCriterion{s.criterion("only criterion")},
		s.reviewer, s.fixer, s.repo)
	s.Require().NoError(err)
	return cycle
}

func (s *CycleFlowSuite) expectCheckpointAndTargets(delta approval.ReviewTarget) {
	cp, err := approval.NewCheckpoint("sha-base")
	s.Require().NoError(err)
	s.repo.EXPECT().Checkpoint(mock.Anything).Return(cp, nil).Maybe()
	s.repo.EXPECT().FullTarget(mock.Anything).Return(approval.NewReviewTarget("full diff"), nil).Maybe()
	s.repo.EXPECT().Delta(mock.Anything, mock.Anything).Return(delta, nil).Maybe()
}

func (s *CycleFlowSuite) TestApprovesOnFirstRound() {
	s.expectCheckpointAndTargets(approval.NewReviewTarget("later"))
	s.reviewer.EXPECT().Review(mock.Anything, mock.Anything).
		Return(approval.NewReviewerOutput("Verdict: APPROVED", nil, s.completeMap()), nil).Once()

	result, err := s.newCycle().Run(context.Background())

	s.Require().NoError(err)
	s.True(result.Approved())
	s.Equal(approval.ReasonApproved, result.Reason())
	s.Equal(1, result.RoundCount())
	s.fixer.AssertNotCalled(s.T(), "Fix", mock.Anything, mock.Anything)
}

func (s *CycleFlowSuite) TestBlockedInputStopsWithoutCallingFixer() {
	s.expectCheckpointAndTargets(approval.NewReviewTarget("later"))
	s.reviewer.EXPECT().Review(mock.Anything, mock.Anything).
		Return(approval.NewReviewerOutput("Verdict: BLOCKED", nil, s.completeMap()), nil).Once()

	cycle := s.newCycle()
	result, err := cycle.Run(context.Background())

	s.Require().NoError(err)
	s.False(result.Approved())
	s.Equal(approval.ReasonBlockedInput, result.Reason())
	s.Equal(approval.StateBlocked, cycle.State())
	s.fixer.AssertNotCalled(s.T(), "Fix", mock.Anything, mock.Anything)
}

func (s *CycleFlowSuite) TestExhaustsCeilingReturnsBlocked() {
	s.expectCheckpointAndTargets(approval.NewReviewTarget("changed"))
	s.fixer.EXPECT().Fix(mock.Anything, mock.Anything).Return(nil)
	s.reviewer.EXPECT().Review(mock.Anything, mock.Anything).
		RunAndReturn(func(_ context.Context, req approval.ReviewRequest) (approval.ReviewerOutput, error) {
			return approval.NewReviewerOutput("Verdict: REJECTED", []approval.Finding{s.finding("R" + strconv.Itoa(req.Round()))}, s.completeMap()), nil
		})

	cycle := s.newCycle()
	result, err := cycle.Run(context.Background())

	s.Require().NoError(err)
	s.False(result.Approved())
	s.Equal(approval.ReasonMaxRounds, result.Reason())
	s.Equal(2, result.RoundCount())
	s.Equal(approval.StateBlocked, cycle.State())
}

func (s *CycleFlowSuite) TestAbortsOnRepeatedFingerprint() {
	s.expectCheckpointAndTargets(approval.NewReviewTarget("changed"))
	s.fixer.EXPECT().Fix(mock.Anything, mock.Anything).Return(nil).Once()
	s.reviewer.EXPECT().Review(mock.Anything, mock.Anything).
		Return(approval.NewReviewerOutput("Verdict: REJECTED", []approval.Finding{s.finding("R1")}, s.completeMap()), nil)

	policy, err := approval.NewApprovalPolicy(approval.WithMaxRounds(5))
	s.Require().NoError(err)
	task, _ := approval.NewTaskIdentity("task-2.0")
	agent, _ := approval.NewAgentIdentity("claude")
	cycle, err := approval.NewCycle(task, agent, policy,
		[]approval.AcceptanceCriterion{s.criterion("only criterion")}, s.reviewer, s.fixer, s.repo)
	s.Require().NoError(err)

	result, err := cycle.Run(context.Background())

	s.Require().NoError(err)
	s.Equal(approval.ReasonNoConvergence, result.Reason())
	s.Equal(2, result.RoundCount())
}

func (s *CycleFlowSuite) TestAbortsOnEmptyDiff() {
	s.expectCheckpointAndTargets(approval.NewReviewTarget(""))
	s.fixer.EXPECT().Fix(mock.Anything, mock.Anything).Return(nil).Once()
	s.reviewer.EXPECT().Review(mock.Anything, mock.Anything).
		RunAndReturn(func(_ context.Context, req approval.ReviewRequest) (approval.ReviewerOutput, error) {
			return approval.NewReviewerOutput("Verdict: REJECTED", []approval.Finding{s.finding("R" + strconv.Itoa(req.Round()))}, s.completeMap()), nil
		})

	policy, err := approval.NewApprovalPolicy(approval.WithMaxRounds(5))
	s.Require().NoError(err)
	task, _ := approval.NewTaskIdentity("task-2.0")
	agent, _ := approval.NewAgentIdentity("claude")
	cycle, err := approval.NewCycle(task, agent, policy,
		[]approval.AcceptanceCriterion{s.criterion("only criterion")}, s.reviewer, s.fixer, s.repo)
	s.Require().NoError(err)

	result, err := cycle.Run(context.Background())

	s.Require().NoError(err)
	s.Equal(approval.ReasonEmptyDiff, result.Reason())
}

func (s *CycleFlowSuite) TestSecondRunIsRejected() {
	s.expectCheckpointAndTargets(approval.NewReviewTarget("later"))
	s.reviewer.EXPECT().Review(mock.Anything, mock.Anything).
		Return(approval.NewReviewerOutput("Verdict: APPROVED", nil, s.completeMap()), nil).Once()

	cycle := s.newCycle()
	_, err := cycle.Run(context.Background())
	s.Require().NoError(err)

	_, err = cycle.Run(context.Background())
	s.Require().ErrorIs(err, approval.ErrCycleAlreadyStarted)
}
