package durable

import (
	"fmt"
	"time"
)

const DefaultLeaseTTL = 30 * time.Minute

type LeasePolicy struct {
	Probe LivenessProbe
}

var DefaultLeasePolicy = LeasePolicy{Probe: DefaultLivenessProbe}

func (p LeasePolicy) probe() LivenessProbe {
	if p.Probe == nil {
		return DefaultLivenessProbe
	}
	return p.Probe
}

func (p LeasePolicy) resolveTTL(ttl time.Duration) time.Duration {
	if ttl <= 0 {
		return DefaultLeaseTTL
	}
	return ttl
}

func (p LeasePolicy) Claim(current *HandoffLease, claimant LeaseOwner, ref ProcessRef, ttl time.Duration, now time.Time) (HandoffLease, LeaseRecord, error) {
	effectiveTTL := p.resolveTTL(ttl)

	if current == nil {
		lease := HandoffLease{Owner: claimant, Deadline: now.Add(effectiveTTL), ProcessRef: ref}
		record := LeaseRecord{Outcome: LeaseOutcomeGranted, Claimant: claimant, Deadline: lease.Deadline, At: now}
		return lease, record, nil
	}

	if !current.Deadline.After(now) {
		lease := HandoffLease{Owner: claimant, Deadline: now.Add(effectiveTTL), ProcessRef: ref}
		record := LeaseRecord{
			Outcome:        LeaseOutcomeTransferred,
			TransferReason: TransferReasonDeadlineExpired,
			PreviousOwner:  current.Owner,
			Claimant:       claimant,
			Deadline:       lease.Deadline,
			At:             now,
		}
		return lease, record, nil
	}

	liveness := p.probe().Probe(current.ProcessRef)
	if liveness.Reliable && !liveness.Alive {
		lease := HandoffLease{Owner: claimant, Deadline: now.Add(effectiveTTL), ProcessRef: ref}
		record := LeaseRecord{
			Outcome:        LeaseOutcomeTransferred,
			TransferReason: TransferReasonOwnerDead,
			PreviousOwner:  current.Owner,
			Claimant:       claimant,
			Deadline:       lease.Deadline,
			At:             now,
		}
		return lease, record, nil
	}

	record := LeaseRecord{
		Outcome:       LeaseOutcomeRefused,
		PreviousOwner: current.Owner,
		Claimant:      claimant,
		Deadline:      current.Deadline,
		At:            now,
	}
	err := fmt.Errorf("durable: baton held by %s until %s: %w", current.Owner, current.Deadline.Format(time.RFC3339), ErrBatonAlreadyClaimed)
	return *current, record, err
}

func (p LeasePolicy) Renew(current HandoffLease, owner LeaseOwner, ttl time.Duration, now time.Time) (HandoffLease, LeaseRecord, error) {
	if current.Owner != owner {
		record := LeaseRecord{
			Outcome:       LeaseOutcomeRefused,
			PreviousOwner: current.Owner,
			Claimant:      owner,
			Deadline:      current.Deadline,
			At:            now,
		}
		err := fmt.Errorf("durable: baton held by %s: %w", current.Owner, ErrBatonAlreadyClaimed)
		return current, record, err
	}

	effectiveTTL := p.resolveTTL(ttl)
	lease := HandoffLease{Owner: owner, Deadline: now.Add(effectiveTTL), ProcessRef: current.ProcessRef}
	record := LeaseRecord{
		Outcome:       LeaseOutcomeRenewed,
		PreviousOwner: current.Owner,
		Claimant:      owner,
		Deadline:      lease.Deadline,
		At:            now,
	}
	return lease, record, nil
}
