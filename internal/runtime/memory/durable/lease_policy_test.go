package durable_test

import (
	"errors"
	"testing"
	"time"

	"github.com/JailtonJunior94/ai-spec-harness/internal/fs"
	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/memory/durable"
)

type fakeLivenessProbe struct {
	result durable.LivenessResult
}

func (f fakeLivenessProbe) Probe(durable.ProcessRef) durable.LivenessResult {
	return f.result
}

func TestLeasePolicy_Claim_FreeLeaseGrants(t *testing.T) {
	t.Parallel()

	policy := durable.LeasePolicy{Probe: fakeLivenessProbe{}}
	now := time.Date(2026, 9, 11, 10, 0, 0, 0, time.UTC)

	lease, record, err := policy.Claim(nil, "session-a", durable.ProcessRef{PID: 100}, 0, now)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if lease.Owner != "session-a" {
		t.Fatalf("lease owner = %q, want session-a", lease.Owner)
	}
	if !lease.Deadline.Equal(now.Add(durable.DefaultLeaseTTL)) {
		t.Fatalf("deadline = %v, want %v", lease.Deadline, now.Add(durable.DefaultLeaseTTL))
	}
	if record.Outcome != durable.LeaseOutcomeGranted {
		t.Fatalf("outcome = %v, want granted", record.Outcome)
	}
}

func TestLeasePolicy_Claim_LiveOwnerWithinDeadlineRefuses(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 9, 11, 10, 0, 0, 0, time.UTC)
	current := &durable.HandoffLease{
		Owner:      "session-a",
		Deadline:   now.Add(20 * time.Minute),
		ProcessRef: durable.ProcessRef{PID: 100},
	}
	policy := durable.LeasePolicy{Probe: fakeLivenessProbe{result: durable.LivenessResult{Reliable: true, Alive: true}}}

	lease, record, err := policy.Claim(current, "session-b", durable.ProcessRef{PID: 200}, 0, now)
	if !errors.Is(err, durable.ErrBatonAlreadyClaimed) {
		t.Fatalf("err = %v, want ErrBatonAlreadyClaimed", err)
	}
	if lease.Owner != "session-a" {
		t.Fatalf("lease owner after refuse = %q, want session-a unchanged", lease.Owner)
	}
	if record.Outcome != durable.LeaseOutcomeRefused {
		t.Fatalf("outcome = %v, want refused", record.Outcome)
	}
	if record.PreviousOwner != "session-a" {
		t.Fatalf("record.PreviousOwner = %q, want session-a", record.PreviousOwner)
	}
	if !record.Deadline.Equal(current.Deadline) {
		t.Fatalf("record.Deadline = %v, want %v (dono e prazo restante informados)", record.Deadline, current.Deadline)
	}
}

func TestLeasePolicy_Claim_ExpiredDeadlineAllowsTakeover(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 9, 11, 10, 0, 0, 0, time.UTC)
	current := &durable.HandoffLease{
		Owner:      "session-a",
		Deadline:   now.Add(-1 * time.Minute),
		ProcessRef: durable.ProcessRef{PID: 100},
	}
	policy := durable.LeasePolicy{Probe: fakeLivenessProbe{result: durable.LivenessResult{Reliable: true, Alive: true}}}

	lease, record, err := policy.Claim(current, "session-b", durable.ProcessRef{PID: 200}, 0, now)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if lease.Owner != "session-b" {
		t.Fatalf("lease owner = %q, want session-b", lease.Owner)
	}
	if record.Outcome != durable.LeaseOutcomeTransferred {
		t.Fatalf("outcome = %v, want transferred", record.Outcome)
	}
	if record.TransferReason != durable.TransferReasonDeadlineExpired {
		t.Fatalf("transfer reason = %v, want deadline_expired", record.TransferReason)
	}
	if record.PreviousOwner != "session-a" {
		t.Fatalf("record.PreviousOwner = %q, want session-a (tomada registrada)", record.PreviousOwner)
	}
}

func TestLeasePolicy_Claim_DeadOwnerAllowsTakeover(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 9, 11, 10, 0, 0, 0, time.UTC)
	current := &durable.HandoffLease{
		Owner:      "session-a",
		Deadline:   now.Add(20 * time.Minute),
		ProcessRef: durable.ProcessRef{PID: 100},
	}
	policy := durable.LeasePolicy{Probe: fakeLivenessProbe{result: durable.LivenessResult{Reliable: true, Alive: false}}}

	lease, record, err := policy.Claim(current, "session-b", durable.ProcessRef{PID: 200}, 0, now)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if lease.Owner != "session-b" {
		t.Fatalf("lease owner = %q, want session-b", lease.Owner)
	}
	if record.Outcome != durable.LeaseOutcomeTransferred {
		t.Fatalf("outcome = %v, want transferred", record.Outcome)
	}
	if record.TransferReason != durable.TransferReasonOwnerDead {
		t.Fatalf("transfer reason = %v, want owner_dead", record.TransferReason)
	}
	if record.PreviousOwner != "session-a" {
		t.Fatalf("record.PreviousOwner = %q, want session-a (tomada registrada)", record.PreviousOwner)
	}
}

func TestLeasePolicy_Claim_UnreliableProbeFallsBackToDeadline(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 9, 11, 10, 0, 0, 0, time.UTC)
	current := &durable.HandoffLease{
		Owner:      "session-a",
		Deadline:   now.Add(20 * time.Minute),
		ProcessRef: durable.ProcessRef{PID: 100},
	}
	policy := durable.LeasePolicy{Probe: fakeLivenessProbe{result: durable.LivenessResult{Reliable: false, Alive: false}}}

	_, record, err := policy.Claim(current, "session-b", durable.ProcessRef{PID: 200}, 0, now)
	if !errors.Is(err, durable.ErrBatonAlreadyClaimed) {
		t.Fatalf("err = %v, want ErrBatonAlreadyClaimed (fallback por prazo: sonda nao confiavel nao libera antes do prazo)", err)
	}
	if record.Outcome != durable.LeaseOutcomeRefused {
		t.Fatalf("outcome = %v, want refused (assimetria assumida do fallback por prazo)", record.Outcome)
	}

	expired := &durable.HandoffLease{
		Owner:      "session-a",
		Deadline:   now.Add(-1 * time.Second),
		ProcessRef: durable.ProcessRef{PID: 100},
	}
	lease, expiredRecord, err := policy.Claim(expired, "session-b", durable.ProcessRef{PID: 200}, 0, now)
	if err != nil {
		t.Fatalf("unexpected error after deadline expired with unreliable probe: %v", err)
	}
	if lease.Owner != "session-b" {
		t.Fatalf("lease owner = %q, want session-b once deadline expires even with unreliable probe", lease.Owner)
	}
	if expiredRecord.TransferReason != durable.TransferReasonDeadlineExpired {
		t.Fatalf("transfer reason = %v, want deadline_expired", expiredRecord.TransferReason)
	}
}

func TestLeasePolicy_Renew_SameOwnerExtendsDeadline(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 9, 11, 10, 0, 0, 0, time.UTC)
	current := durable.HandoffLease{
		Owner:      "session-a",
		Deadline:   now.Add(5 * time.Minute),
		ProcessRef: durable.ProcessRef{PID: 100},
	}
	policy := durable.LeasePolicy{Probe: fakeLivenessProbe{}}

	lease, record, err := policy.Renew(current, "session-a", 0, now)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !lease.Deadline.Equal(now.Add(durable.DefaultLeaseTTL)) {
		t.Fatalf("deadline after renew = %v, want %v", lease.Deadline, now.Add(durable.DefaultLeaseTTL))
	}
	if record.Outcome != durable.LeaseOutcomeRenewed {
		t.Fatalf("outcome = %v, want renewed", record.Outcome)
	}
}

func TestLeasePolicy_Renew_DifferentOwnerRefuses(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 9, 11, 10, 0, 0, 0, time.UTC)
	current := durable.HandoffLease{
		Owner:      "session-a",
		Deadline:   now.Add(5 * time.Minute),
		ProcessRef: durable.ProcessRef{PID: 100},
	}
	policy := durable.LeasePolicy{Probe: fakeLivenessProbe{}}

	_, record, err := policy.Renew(current, "session-b", 0, now)
	if !errors.Is(err, durable.ErrBatonAlreadyClaimed) {
		t.Fatalf("err = %v, want ErrBatonAlreadyClaimed", err)
	}
	if record.Outcome != durable.LeaseOutcomeRefused {
		t.Fatalf("outcome = %v, want refused", record.Outcome)
	}
}

func TestContinuityHandoff_ClaimThenRenewThenTakeover(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 9, 11, 10, 0, 0, 0, time.UTC)
	fsys := fs.NewFakeFileSystem()
	scope := durable.Scope{Layer: durable.TargetLayerPRD, TasksDir: "/tasks"}
	handoff := durable.NewContinuityHandoff(durable.LeasePolicy{
		Probe: fakeLivenessProbe{result: durable.LivenessResult{Reliable: true, Alive: false}},
	}, fsys, newInMemoryLayerLocker(), scope, "/tasks")

	if _, ok := handoff.Current(); ok {
		t.Fatal("expected no current lease before first claim")
	}

	record, err := handoff.Claim("session-a", durable.ProcessRef{PID: 100}, 0, now)
	if err != nil {
		t.Fatalf("unexpected error on first claim: %v", err)
	}
	if record.Outcome != durable.LeaseOutcomeGranted {
		t.Fatalf("outcome = %v, want granted", record.Outcome)
	}

	renewRecord, err := handoff.Renew("session-a", 0, now.Add(time.Minute))
	if err != nil {
		t.Fatalf("unexpected error on renew: %v", err)
	}
	if renewRecord.Outcome != durable.LeaseOutcomeRenewed {
		t.Fatalf("outcome = %v, want renewed", renewRecord.Outcome)
	}

	takeoverRecord, err := handoff.Claim("session-b", durable.ProcessRef{PID: 200}, 0, now.Add(2*time.Minute))
	if err != nil {
		t.Fatalf("unexpected error on takeover of dead owner: %v", err)
	}
	if takeoverRecord.Outcome != durable.LeaseOutcomeTransferred {
		t.Fatalf("outcome = %v, want transferred", takeoverRecord.Outcome)
	}
	if takeoverRecord.TransferReason != durable.TransferReasonOwnerDead {
		t.Fatalf("transfer reason = %v, want owner_dead", takeoverRecord.TransferReason)
	}

	current, ok := handoff.Current()
	if !ok {
		t.Fatal("expected current lease after takeover")
	}
	if current.Owner != "session-b" {
		t.Fatalf("current owner = %q, want session-b", current.Owner)
	}
}

func TestContinuityHandoff_RenewWithoutLeaseFails(t *testing.T) {
	t.Parallel()

	fsys := fs.NewFakeFileSystem()
	scope := durable.Scope{Layer: durable.TargetLayerPRD, TasksDir: "/tasks"}
	handoff := durable.NewContinuityHandoff(durable.DefaultLeasePolicy, fsys, newInMemoryLayerLocker(), scope, "/tasks")
	_, err := handoff.Renew("session-a", 0, time.Now())
	if !errors.Is(err, durable.ErrHandoffLeaseNotHeld) {
		t.Fatalf("err = %v, want ErrHandoffLeaseNotHeld", err)
	}
}
