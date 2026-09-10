package approval

import (
	"errors"
	"slices"
	"testing"

	"github.com/stretchr/testify/suite"
)

type ValueObjectsSuite struct {
	suite.Suite
}

func TestValueObjectsSuite(t *testing.T) {
	suite.Run(t, new(ValueObjectsSuite))
}

func (s *ValueObjectsSuite) TestIdentities() {
	_, err := NewTaskIdentity("  ")
	s.Require().ErrorIs(err, ErrInvalidIdentity)
	_, err = NewAgentIdentity("")
	s.Require().ErrorIs(err, ErrInvalidIdentity)

	task, err := NewTaskIdentity(" task-1 ")
	s.Require().NoError(err)
	s.Equal("task-1", task.String())
	s.False(task.Zero())
	s.True(TaskIdentity{}.Zero())
	s.True(AgentIdentity{}.Zero())
}

func (s *ValueObjectsSuite) TestVerdict() {
	s.True(VerdictApproved.Approves())
	s.False(VerdictApprovedWithRemarks.Approves())
	s.Equal("APPROVED_WITH_REMARKS", VerdictApprovedWithRemarks.String())
	s.Equal("REJECTED", VerdictRejected.String())
	s.Equal("BLOCKED", VerdictBlocked.String())
	s.Equal("INVALID_VERDICT", Verdict(0).String())

	_, err := NewVerdict(Verdict(99))
	s.Require().ErrorIs(err, ErrInvalidVerdict)
	v, err := NewVerdict(VerdictApproved)
	s.Require().NoError(err)
	s.Equal(VerdictApproved, v)
}

func (s *ValueObjectsSuite) TestStopReason() {
	s.Equal("approved", ReasonApproved.String())
	s.Equal("max_rounds", ReasonMaxRounds.String())
	s.Equal("no_convergence", ReasonNoConvergence.String())
	s.Equal("empty_diff", ReasonEmptyDiff.String())
	s.Equal("blocked_input", ReasonBlockedInput.String())
	s.Equal("invalid_reason", StopReason(0).String())

	_, err := NewStopReason(StopReason(42))
	s.Require().ErrorIs(err, ErrInvalidStopReason)
}

func (s *ValueObjectsSuite) TestState() {
	s.Equal("in_review", StateInReview.String())
	s.Equal("in_fix", StateInFix.String())
	s.Equal("invalid_state", State(0).String())
	_, err := NewState(State(9))
	s.Require().ErrorIs(err, ErrInvalidState)
}

func (s *ValueObjectsSuite) TestSeverityAndBugLevel() {
	s.Equal("low", SeverityLow.String())
	s.Equal("critical", SeverityCritical.String())
	s.Equal("invalid_severity", Severity(0).String())
	s.True(SeverityHigh.Blocks())
	s.True(SeverityCritical.Blocks())
	s.False(SeverityMedium.Blocks())

	for sev, want := range map[Severity]BugLevel{
		SeverityLow:      BugLevelMinor,
		SeverityMedium:   BugLevelMinor,
		SeverityHigh:     BugLevelMajor,
		SeverityCritical: BugLevelCritical,
	} {
		got, err := sev.BugLevel()
		s.Require().NoError(err)
		s.Equal(want, got)
	}
	_, err := Severity(0).BugLevel()
	s.Require().ErrorIs(err, ErrInvalidSeverity)
	s.Equal("major", BugLevelMajor.String())
	s.Equal("invalid_level", BugLevel(0).String())
}

func (s *ValueObjectsSuite) TestFindingValidation() {
	_, err := NewFinding(Severity(0), "a.go", "R1", "")
	s.Require().ErrorIs(err, ErrInvalidSeverity)
	_, err = NewFinding(SeverityLow, "  ", "R1", "")
	s.Require().ErrorIs(err, ErrInvalidFinding)
	_, err = NewFinding(SeverityLow, "a.go", " ", "")
	s.Require().ErrorIs(err, ErrInvalidFinding)

	f, err := NewFinding(SeverityHigh, " a.go ", " R1 ", " boom ")
	s.Require().NoError(err)
	s.Equal("a.go", f.File())
	s.Equal("R1", f.Rule())
	s.Equal("boom", f.Description())
	s.Equal(SeverityHigh, f.Severity())
}

func (s *ValueObjectsSuite) TestEvidenceForms() {
	_, err := NewCommandEvidence("cmd", "")
	s.Require().ErrorIs(err, ErrInvalidEvidence)
	_, err = NewTestEvidence("", "pass")
	s.Require().ErrorIs(err, ErrInvalidEvidence)
	_, err = NewFileLineEvidence("a.go", true)
	s.Require().ErrorIs(err, ErrInvalidEvidence)
	_, err = NewFileLineEvidence("a.go:12", false)
	s.Require().ErrorIs(err, ErrInvalidEvidence)
	_, err = NewFileLineEvidence("a.go:xx", true)
	s.Require().ErrorIs(err, ErrInvalidEvidence)

	line, err := NewFileLineEvidence("dir/a.go:12", true)
	s.Require().NoError(err)
	s.Equal(EvidenceFormFileLineInDiff, line.Form())
	s.Equal("dir/a.go:12", line.Reference())
	s.NotEmpty(line.Record())
	s.True(line.Valid())
	s.False(EvidenceLine{}.Valid())

	tst, err := NewTestEvidence("TestX", "pass")
	s.Require().NoError(err)
	s.Equal(EvidenceFormTestResult, tst.Form())
}

func (s *ValueObjectsSuite) TestCriteriaMap() {
	_, err := NewCriteriaMap(nil)
	s.Require().ErrorIs(err, ErrInvalidCriteriaMap)

	c1 := mustCriterion(s.T(), "c1")
	dup := mustCriterion(s.T(), "c1")
	_, err = NewCriteriaMap([]AcceptanceCriterion{c1, dup})
	s.Require().ErrorIs(err, ErrInvalidCriteriaMap)

	c2 := mustCriterion(s.T(), "c2")
	m, err := NewCriteriaMap([]AcceptanceCriterion{c1, c2})
	s.Require().NoError(err)
	s.Equal(2, m.Total())
	s.Equal(0, m.Bound())
	s.False(m.Complete())

	_, err = m.WithEvidence(c1, EvidenceLine{})
	s.Require().ErrorIs(err, ErrInvalidEvidence)
	_, err = m.WithEvidence(mustCriterion(s.T(), "absent"), mustCommandEvidence(s.T()))
	s.Require().ErrorIs(err, ErrInvalidCriteriaMap)
	_, err = m.WithEvidence(AcceptanceCriterion{}, mustCommandEvidence(s.T()))
	s.Require().ErrorIs(err, ErrInvalidCriterion)

	m, err = m.WithEvidence(c1, mustCommandEvidence(s.T()))
	s.Require().NoError(err)
	m, err = m.AsUnverifiable(c2)
	s.Require().NoError(err)
	s.Equal(1, len(slices.Collect(m.Unverifiable())))
	s.Equal(1, len(slices.Collect(m.Pending())))
	s.Equal(2, len(slices.Collect(m.Criteria())))
	s.False(m.Complete())

	m, err = m.WithEvidence(c2, mustCommandEvidence(s.T()))
	s.Require().NoError(err)
	s.True(m.Complete())
	s.Equal(0, len(slices.Collect(m.Pending())))
}

func (s *ValueObjectsSuite) TestRound() {
	_, err := NewRound(0)
	s.Require().ErrorIs(err, ErrInvalidRound)

	r, err := NewRound(1)
	s.Require().NoError(err)
	s.False(r.Concluded())

	_, err = r.Complete(Verdict(0), nil, Fingerprint{})
	s.Require().ErrorIs(err, ErrInvalidVerdict)

	findings := []Finding{
		mustFinding(s.T(), SeverityHigh, "a.go", "R1"),
		mustFinding(s.T(), SeverityHigh, "b.go", "R2"),
		mustFinding(s.T(), SeverityLow, "c.go", "R3"),
	}
	done, err := r.Complete(VerdictRejected, findings, NewFingerprintCalculator().Compute(findings))
	s.Require().NoError(err)
	s.True(done.Concluded())
	s.Equal(1, done.Number())
	s.Equal(VerdictRejected, done.Verdict())
	s.Equal(3, len(slices.Collect(done.Findings())))
	s.Equal(map[Severity]int{SeverityHigh: 2, SeverityLow: 1}, done.CountBySeverity())

	_, err = done.Complete(VerdictApproved, nil, Fingerprint{})
	s.Require().ErrorIs(err, ErrRoundAlreadyCompleted)
}

func (s *ValueObjectsSuite) TestResult() {
	_, err := NewClosedResult(ReasonApproved, nil)
	s.Require().ErrorIs(err, ErrInsufficientProof)
	_, err = NewClosedResult(StopReason(0), nil)
	s.Require().ErrorIs(err, ErrInvalidStopReason)

	closed, err := NewClosedResult(ReasonMaxRounds, []Round{})
	s.Require().NoError(err)
	s.False(closed.Approved())
	s.Equal(ReasonMaxRounds, closed.Reason())
	s.Equal(0, closed.RoundCount())

	proof, err := NewApprovalProof(VerdictApproved, completeCriteriaMap(s.T()))
	s.Require().NoError(err)
	approved, err := NewApprovedResult(proof, nil)
	s.Require().NoError(err)
	s.True(approved.Approved())
	s.Equal(proof.Verdict(), approved.Proof().Verdict())
	s.Equal(0, len(slices.Collect(approved.Rounds())))
}

func (s *ValueObjectsSuite) TestEvents() {
	events := []Event{
		RoundStarted{Number: 1},
		RoundCompleted{Number: 1, Verdict: VerdictRejected},
		ProofRejected{Number: 1, Reason: "x"},
		FixRequested{Number: 1},
		CycleClosed{Reason: ReasonMaxRounds},
	}
	names := make([]string, 0, len(events))
	for _, e := range events {
		names = append(names, e.EventName())
	}
	s.Equal([]string{"round_started", "round_completed", "proof_rejected", "fix_requested", "cycle_closed"}, names)
}

func (s *ValueObjectsSuite) TestPortsValidation() {
	task, _ := NewTaskIdentity("t")
	agent, _ := NewAgentIdentity("a")
	crit := []AcceptanceCriterion{mustCriterion(s.T(), "c")}

	_, err := NewReviewRequest(TaskIdentity{}, agent, 1, ReviewTarget{}, crit)
	s.Require().ErrorIs(err, ErrInvalidRequest)
	_, err = NewReviewRequest(task, agent, 0, ReviewTarget{}, crit)
	s.Require().ErrorIs(err, ErrInvalidRequest)
	_, err = NewReviewRequest(task, agent, 1, ReviewTarget{}, nil)
	s.Require().ErrorIs(err, ErrInvalidRequest)
	rr, err := NewReviewRequest(task, agent, 2, NewReviewTarget("d"), crit)
	s.Require().NoError(err)
	s.Equal(2, rr.Round())
	s.Equal("t", rr.Task().String())
	s.Equal("a", rr.Agent().String())
	s.Equal("d", rr.Target().String())
	s.Equal(1, len(slices.Collect(rr.Criteria())))

	_, err = NewFixRequest(task, agent, 1, nil, ReviewTarget{})
	s.Require().ErrorIs(err, ErrInvalidRequest)
	fr, err := NewFixRequest(task, agent, 1, []Finding{mustFinding(s.T(), SeverityHigh, "a.go", "R1")}, NewReviewTarget("d"))
	s.Require().NoError(err)
	s.Equal(1, len(slices.Collect(fr.Findings())))
	s.Equal("d", fr.Target().String())
	s.Equal(1, fr.Round())
	s.Equal("t", fr.Task().String())
	s.Equal("a", fr.Agent().String())

	_, err = NewCheckpoint("  ")
	s.Require().ErrorIs(err, ErrInvalidCheckpoint)
	cp, err := NewCheckpoint("sha")
	s.Require().NoError(err)
	s.Equal("sha", cp.String())
	s.False(cp.Zero())
	s.True(Checkpoint{}.Zero())

	out := NewReviewerOutput("raw", []Finding{mustFinding(s.T(), SeverityLow, "a.go", "R1")}, completeCriteriaMap(s.T()))
	s.Equal("raw", out.RawText())
	s.Equal(1, len(slices.Collect(out.Findings())))
	s.Equal(1, len(out.FindingsList()))
	s.True(out.CriteriaMap().Complete())
	s.True(NewReviewTarget(" ").Empty())
	s.False(NewReviewTarget("x").Empty())
}

func (s *ValueObjectsSuite) TestForbiddenTransitionErrorIs() {
	err := ForbiddenTransitionError{From: StateApproved, To: StateInReview}
	s.Require().True(errors.Is(err, ErrForbiddenTransition))
	s.Contains(err.Error(), "approved")
	s.Contains(InvalidMaxRoundsError{Value: -3}.Error(), "-3")
}
