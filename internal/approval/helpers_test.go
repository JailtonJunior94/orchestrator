package approval

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

type stubReviewer struct {
	output ReviewerOutput
	err    error
}

func (s stubReviewer) Review(context.Context, ReviewRequest) (ReviewerOutput, error) {
	return s.output, s.err
}

type stubFixer struct {
	err error
}

func (s stubFixer) Fix(context.Context, FixRequest) error {
	return s.err
}

type stubRepository struct {
	full  ReviewTarget
	delta ReviewTarget
}

func (s stubRepository) Checkpoint(context.Context) (Checkpoint, error) {
	return NewCheckpoint("sha-0")
}

func (s stubRepository) FullTarget(context.Context) (ReviewTarget, error) {
	return s.full, nil
}

func (s stubRepository) Delta(context.Context, Checkpoint) (ReviewTarget, error) {
	return s.delta, nil
}

func mustCriterion(t *testing.T, description string) AcceptanceCriterion {
	t.Helper()
	c, err := NewAcceptanceCriterion(description)
	require.NoError(t, err)
	return c
}

func mustCommandEvidence(t *testing.T) EvidenceLine {
	t.Helper()
	e, err := NewCommandEvidence("go test ./internal/approval/...", "ok  internal/approval  0.2s")
	require.NoError(t, err)
	return e
}

func completeCriteriaMap(t *testing.T) CriteriaMap {
	t.Helper()
	c1 := mustCriterion(t, "criterion one")
	c2 := mustCriterion(t, "criterion two")
	m, err := NewCriteriaMap([]AcceptanceCriterion{c1, c2})
	require.NoError(t, err)
	m, err = m.WithEvidence(c1, mustCommandEvidence(t))
	require.NoError(t, err)
	m, err = m.WithEvidence(c2, mustCommandEvidence(t))
	require.NoError(t, err)
	require.True(t, m.Complete())
	return m
}

func incompleteCriteriaMap(t *testing.T) CriteriaMap {
	t.Helper()
	c1 := mustCriterion(t, "criterion one")
	c2 := mustCriterion(t, "criterion two")
	m, err := NewCriteriaMap([]AcceptanceCriterion{c1, c2})
	require.NoError(t, err)
	m, err = m.WithEvidence(c1, mustCommandEvidence(t))
	require.NoError(t, err)
	require.False(t, m.Complete())
	return m
}

func unverifiableCriteriaMap(t *testing.T) CriteriaMap {
	t.Helper()
	c1 := mustCriterion(t, "criterion one")
	c2 := mustCriterion(t, "criterion two")
	m, err := NewCriteriaMap([]AcceptanceCriterion{c1, c2})
	require.NoError(t, err)
	m, err = m.WithEvidence(c1, mustCommandEvidence(t))
	require.NoError(t, err)
	m, err = m.AsUnverifiable(c2)
	require.NoError(t, err)
	require.False(t, m.Complete())
	return m
}

func mustFinding(t *testing.T, sev Severity, file, rule string) Finding {
	t.Helper()
	f, err := NewFinding(sev, file, rule, "")
	require.NoError(t, err)
	return f
}

func newStubCycle(t *testing.T) *Cycle {
	t.Helper()
	task, err := NewTaskIdentity("task-2.0")
	require.NoError(t, err)
	agent, err := NewAgentIdentity("claude")
	require.NoError(t, err)
	policy, err := NewApprovalPolicy()
	require.NoError(t, err)
	cycle, err := NewCycle(
		task, agent, policy,
		[]AcceptanceCriterion{mustCriterion(t, "criterion one")},
		stubReviewer{}, stubFixer{}, stubRepository{},
	)
	require.NoError(t, err)
	return cycle
}
