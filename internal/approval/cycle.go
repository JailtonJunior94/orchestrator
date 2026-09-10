package approval

import (
	"context"
	"fmt"
	"iter"
	"slices"
)

type Cycle struct {
	task     TaskIdentity
	agent    AgentIdentity
	policy   ApprovalPolicy
	criteria []AcceptanceCriterion

	reviewer   Reviewer
	fixer      Fixer
	repository Repository
	translator Translator
	calculator FingerprintCalculator

	machine stateMachine
	rounds  []Round
	events  []Event
	started bool
	closed  bool
}

func NewCycle(
	task TaskIdentity,
	agent AgentIdentity,
	policy ApprovalPolicy,
	criteria []AcceptanceCriterion,
	reviewer Reviewer,
	fixer Fixer,
	repository Repository,
) (*Cycle, error) {
	if task.Zero() || agent.Zero() {
		return nil, fmt.Errorf("%w: missing identity in cycle", ErrInvalidIdentity)
	}
	if policy.MaxRounds() < 1 {
		return nil, InvalidMaxRoundsError{Value: policy.MaxRounds()}
	}
	if len(criteria) == 0 {
		return nil, fmt.Errorf("%w: cycle without acceptance criteria", ErrInvalidCriteriaMap)
	}
	for _, c := range criteria {
		if c.Zero() {
			return nil, fmt.Errorf("%w: uninitialized criterion in cycle", ErrInvalidCriterion)
		}
	}
	if reviewer == nil || fixer == nil || repository == nil {
		return nil, fmt.Errorf("%w: nil port in cycle", ErrInvalidRequest)
	}
	return &Cycle{
		task:       task,
		agent:      agent,
		policy:     policy,
		criteria:   slices.Clone(criteria),
		reviewer:   reviewer,
		fixer:      fixer,
		repository: repository,
		translator: NewTranslator(),
		calculator: NewFingerprintCalculator(),
		machine:    stateMachine{current: StateInReview},
	}, nil
}

func (c *Cycle) Run(ctx context.Context) (CycleResult, error) {
	if c.started {
		return CycleResult{}, ErrCycleAlreadyStarted
	}
	c.started = true

	var (
		previousCheckpoint  Checkpoint
		previousFingerprint Fingerprint
		changed             = true
	)
	guard := c.policy.MaxRounds() + 1

	for number := 1; number <= guard; number++ {
		c.emit(RoundStarted{Number: number})

		checkpoint, err := c.repository.Checkpoint(ctx)
		if err != nil {
			return CycleResult{}, fmt.Errorf("checkpoint round %d: %w", number, err)
		}

		target, err := c.roundTarget(ctx, number, previousCheckpoint)
		if err != nil {
			return CycleResult{}, err
		}

		round, criteriaMap, err := c.reviewRound(ctx, number, target)
		if err != nil {
			return CycleResult{}, err
		}
		c.rounds = append(c.rounds, round)
		c.emit(RoundCompleted{Number: number, Verdict: round.Verdict(), Fingerprint: round.Fingerprint()})

		proof, proofErr := NewApprovalProof(round.Verdict(), criteriaMap)
		if proofErr == nil {
			return c.approve(proof)
		}
		c.emit(ProofRejected{Number: number, Reason: proofErr.Error()})

		reason, stop := c.policy.Decide(StopDecision{
			Verdict:             round.Verdict(),
			CurrentRound:        number,
			CurrentFingerprint:  round.Fingerprint(),
			PreviousFingerprint: previousFingerprint,
			Changed:             changed,
		})
		if stop {
			return c.closeBlocked(reason)
		}

		findings := slices.Collect(round.Findings())
		if len(findings) == 0 {
			return c.closeBlocked(ReasonBlockedInput)
		}

		didChange, err := c.fixRound(ctx, number, findings, target, checkpoint)
		if err != nil {
			return CycleResult{}, err
		}
		changed = didChange
		previousFingerprint = round.Fingerprint()
		previousCheckpoint = checkpoint
	}

	return CycleResult{}, fmt.Errorf("%w: cycle exceeded the defensive guard of %d rounds", ErrCycleClosed, guard)
}

func (c *Cycle) State() State {
	return c.machine.currentState()
}

func (c *Cycle) RoundCount() int {
	return len(c.rounds)
}

func (c *Cycle) CurrentVerdict() Verdict {
	if len(c.rounds) == 0 {
		return 0
	}
	return c.rounds[len(c.rounds)-1].Verdict()
}

func (c *Cycle) Closed() bool {
	return c.closed
}

func (c *Cycle) Rounds() iter.Seq[Round] {
	return slices.Values(c.rounds)
}

func (c *Cycle) Events() iter.Seq[Event] {
	return slices.Values(c.events)
}

func (c *Cycle) roundTarget(ctx context.Context, number int, previousCheckpoint Checkpoint) (ReviewTarget, error) {
	if number == 1 {
		target, err := c.repository.FullTarget(ctx)
		if err != nil {
			return ReviewTarget{}, fmt.Errorf("full target round %d: %w", number, err)
		}
		return target, nil
	}
	target, err := c.repository.Delta(ctx, previousCheckpoint)
	if err != nil {
		return ReviewTarget{}, fmt.Errorf("delta round %d: %w", number, err)
	}
	return target, nil
}

func (c *Cycle) reviewRound(ctx context.Context, number int, target ReviewTarget) (Round, CriteriaMap, error) {
	request, err := NewReviewRequest(c.task, c.agent, number, target, c.criteria)
	if err != nil {
		return Round{}, CriteriaMap{}, err
	}
	output, err := c.reviewer.Review(ctx, request)
	if err != nil {
		return Round{}, CriteriaMap{}, fmt.Errorf("review round %d: %w", number, err)
	}

	verdict := c.translator.Translate(output.RawText())
	findings := output.FindingsList()
	fingerprint := c.calculator.Compute(findings)

	round, err := NewRound(number)
	if err != nil {
		return Round{}, CriteriaMap{}, err
	}
	completed, err := round.Complete(verdict, findings, fingerprint)
	if err != nil {
		return Round{}, CriteriaMap{}, err
	}
	return completed, output.CriteriaMap(), nil
}

func (c *Cycle) fixRound(ctx context.Context, number int, findings []Finding, target ReviewTarget, checkpoint Checkpoint) (bool, error) {
	if err := c.machine.transition(StateInFix); err != nil {
		return false, err
	}
	c.emit(FixRequested{Number: number})

	request, err := NewFixRequest(c.task, c.agent, number, findings, target)
	if err != nil {
		return false, err
	}
	if err := c.fixer.Fix(ctx, request); err != nil {
		return false, fmt.Errorf("fix round %d: %w", number, err)
	}

	delta, err := c.repository.Delta(ctx, checkpoint)
	if err != nil {
		return false, fmt.Errorf("post-fix delta round %d: %w", number, err)
	}
	if err := c.machine.transition(StateInReview); err != nil {
		return false, err
	}
	return !delta.Empty(), nil
}

func (c *Cycle) approve(proof ApprovalProof) (CycleResult, error) {
	if !proof.Valid() {
		return CycleResult{}, fmt.Errorf("%w: approve rejects a zero-value proof", ErrInsufficientProof)
	}
	if err := c.machine.transition(StateApproved); err != nil {
		return CycleResult{}, err
	}
	c.closed = true
	c.emit(CycleClosed{Reason: ReasonApproved, Approved: true, Rounds: len(c.rounds)})
	return NewApprovedResult(proof, c.rounds)
}

func (c *Cycle) closeBlocked(reason StopReason) (CycleResult, error) {
	if err := c.machine.transition(StateBlocked); err != nil {
		return CycleResult{}, err
	}
	c.closed = true
	c.emit(CycleClosed{Reason: reason, Approved: false, Rounds: len(c.rounds)})
	return NewClosedResult(reason, c.rounds)
}

func (c *Cycle) emit(event Event) {
	c.events = append(c.events, event)
}
