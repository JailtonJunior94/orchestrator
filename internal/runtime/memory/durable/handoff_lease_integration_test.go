//go:build integration

package durable_test

import (
	"os"
	"os/exec"
	"testing"
	"time"

	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/memory/durable"
)

func TestOrphanLeaseDetectedByDeadOwner(t *testing.T) {
	cmd := exec.Command("true")
	if err := cmd.Run(); err != nil {
		t.Fatalf("spawn short-lived real process: %v", err)
	}
	deadPID := cmd.Process.Pid

	hostname, err := os.Hostname()
	if err != nil {
		t.Fatalf("resolve hostname: %v", err)
	}

	deadOwnerRef := durable.ProcessRef{PID: deadPID, Hostname: hostname, StartedAt: time.Now().Add(-time.Hour)}
	current := &durable.HandoffLease{
		Owner:      durable.LeaseOwner("pid:orphan"),
		Deadline:   time.Now().Add(20 * time.Minute),
		ProcessRef: deadOwnerRef,
	}

	claimant := durable.LeaseOwner("pid:successor")
	_, record, claimErr := durable.DefaultLeasePolicy.Claim(current, claimant, durable.CurrentProcessRef(), durable.DefaultLeaseTTL, time.Now())
	if claimErr != nil {
		t.Fatalf("orphan lease should be taken over without error, got: %v", claimErr)
	}
	if record.Outcome != durable.LeaseOutcomeTransferred {
		t.Fatalf("outcome = %v, want LeaseOutcomeTransferred (orphan takeover must be recorded, never silently overwritten)", record.Outcome)
	}
	if record.TransferReason != durable.TransferReasonOwnerDead {
		t.Fatalf("transfer reason = %v, want TransferReasonOwnerDead", record.TransferReason)
	}
	if record.PreviousOwner != current.Owner {
		t.Fatalf("previous owner = %q, want %q (takeover must record the displaced owner)", record.PreviousOwner, current.Owner)
	}
	if record.Claimant != claimant {
		t.Fatalf("claimant = %q, want %q", record.Claimant, claimant)
	}
}

func TestOrphanLeaseDetectedByExpiredDeadline(t *testing.T) {
	now := time.Now()
	current := &durable.HandoffLease{
		Owner:      durable.LeaseOwner("pid:expired"),
		Deadline:   now.Add(-time.Minute),
		ProcessRef: durable.CurrentProcessRef(),
	}

	claimant := durable.LeaseOwner("pid:successor")
	_, record, claimErr := durable.DefaultLeasePolicy.Claim(current, claimant, durable.CurrentProcessRef(), durable.DefaultLeaseTTL, now)
	if claimErr != nil {
		t.Fatalf("expired lease should be taken over without error, got: %v", claimErr)
	}
	if record.Outcome != durable.LeaseOutcomeTransferred {
		t.Fatalf("outcome = %v, want LeaseOutcomeTransferred", record.Outcome)
	}
	if record.TransferReason != durable.TransferReasonDeadlineExpired {
		t.Fatalf("transfer reason = %v, want TransferReasonDeadlineExpired", record.TransferReason)
	}
}

func TestLiveOwnerLeaseIsNeverTakenOver(t *testing.T) {
	current := &durable.HandoffLease{
		Owner:      durable.LeaseOwner("pid:alive"),
		Deadline:   time.Now().Add(20 * time.Minute),
		ProcessRef: durable.CurrentProcessRef(),
	}

	_, record, claimErr := durable.DefaultLeasePolicy.Claim(current, durable.LeaseOwner("pid:intruder"), durable.CurrentProcessRef(), durable.DefaultLeaseTTL, time.Now())
	if claimErr == nil {
		t.Fatal("expected refusal error for live owner within deadline")
	}
	if record.Outcome != durable.LeaseOutcomeRefused {
		t.Fatalf("outcome = %v, want LeaseOutcomeRefused", record.Outcome)
	}
}
