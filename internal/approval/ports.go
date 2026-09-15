package approval

import (
	"context"
	"fmt"
	"iter"
	"slices"
	"strings"
)

type Reviewer interface {
	Review(ctx context.Context, request ReviewRequest) (ReviewerOutput, error)
}

type Fixer interface {
	Fix(ctx context.Context, request FixRequest) error
}

type Repository interface {
	Checkpoint(ctx context.Context) (Checkpoint, error)
	FullTarget(ctx context.Context) (ReviewTarget, error)
	Delta(ctx context.Context, since Checkpoint) (ReviewTarget, error)
}

type ReviewRequest struct {
	task     TaskIdentity
	agent    AgentIdentity
	round    int
	target   ReviewTarget
	criteria []AcceptanceCriterion
}

type FixRequest struct {
	task     TaskIdentity
	agent    AgentIdentity
	round    int
	findings []Finding
	target   ReviewTarget
}

type ReviewerOutput struct {
	rawText     string
	findings    []Finding
	criteriaMap CriteriaMap
}

type Checkpoint struct {
	value string
}

type ReviewTarget struct {
	value string
}

func NewReviewRequest(task TaskIdentity, agent AgentIdentity, round int, target ReviewTarget, criteria []AcceptanceCriterion) (ReviewRequest, error) {
	if task.Zero() || agent.Zero() {
		return ReviewRequest{}, fmt.Errorf("%w: missing identity in review request", ErrInvalidRequest)
	}
	if round < 1 {
		return ReviewRequest{}, fmt.Errorf("%w: round %d", ErrInvalidRequest, round)
	}
	if len(criteria) == 0 {
		return ReviewRequest{}, fmt.Errorf("%w: review request without acceptance criteria", ErrInvalidRequest)
	}
	return ReviewRequest{
		task:     task,
		agent:    agent,
		round:    round,
		target:   target,
		criteria: slices.Clone(criteria),
	}, nil
}

func NewFixRequest(task TaskIdentity, agent AgentIdentity, round int, findings []Finding, target ReviewTarget) (FixRequest, error) {
	if task.Zero() || agent.Zero() {
		return FixRequest{}, fmt.Errorf("%w: missing identity in fix request", ErrInvalidRequest)
	}
	if round < 1 {
		return FixRequest{}, fmt.Errorf("%w: round %d", ErrInvalidRequest, round)
	}
	if len(findings) == 0 {
		return FixRequest{}, fmt.Errorf("%w: fix request without findings", ErrInvalidRequest)
	}
	return FixRequest{
		task:     task,
		agent:    agent,
		round:    round,
		findings: slices.Clone(findings),
		target:   target,
	}, nil
}

func NewReviewerOutput(rawText string, findings []Finding, criteriaMap CriteriaMap) ReviewerOutput {
	return ReviewerOutput{
		rawText:     rawText,
		findings:    slices.Clone(findings),
		criteriaMap: criteriaMap,
	}
}

func NewCheckpoint(value string) (Checkpoint, error) {
	v := strings.TrimSpace(value)
	if v == "" {
		return Checkpoint{}, fmt.Errorf("%w: empty checkpoint", ErrInvalidCheckpoint)
	}
	return Checkpoint{value: v}, nil
}

func NewReviewTarget(value string) ReviewTarget {
	return ReviewTarget{value: value}
}

func (r ReviewRequest) Task() TaskIdentity {
	return r.task
}

func (r ReviewRequest) Agent() AgentIdentity {
	return r.agent
}

func (r ReviewRequest) Round() int {
	return r.round
}

func (r ReviewRequest) Target() ReviewTarget {
	return r.target
}

func (r ReviewRequest) Criteria() iter.Seq[AcceptanceCriterion] {
	return slices.Values(r.criteria)
}

func (r FixRequest) Task() TaskIdentity {
	return r.task
}

func (r FixRequest) Agent() AgentIdentity {
	return r.agent
}

func (r FixRequest) Round() int {
	return r.round
}

func (r FixRequest) Target() ReviewTarget {
	return r.target
}

func (r FixRequest) Findings() iter.Seq[Finding] {
	return slices.Values(r.findings)
}

func (o ReviewerOutput) RawText() string {
	return o.rawText
}

func (o ReviewerOutput) Findings() iter.Seq[Finding] {
	return slices.Values(o.findings)
}

func (o ReviewerOutput) FindingsList() []Finding {
	return slices.Clone(o.findings)
}

func (o ReviewerOutput) CriteriaMap() CriteriaMap {
	return o.criteriaMap
}

func (c Checkpoint) String() string {
	return c.value
}

func (c Checkpoint) Zero() bool {
	return c.value == ""
}

func (t ReviewTarget) String() string {
	return t.value
}

func (t ReviewTarget) Empty() bool {
	return strings.TrimSpace(t.value) == ""
}
