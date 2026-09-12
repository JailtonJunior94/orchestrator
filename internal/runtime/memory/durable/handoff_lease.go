package durable

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/JailtonJunior94/ai-spec-harness/internal/fs"
)

type LeaseOwner string

type HandoffLease struct {
	Owner      LeaseOwner
	Deadline   time.Time
	ProcessRef ProcessRef
}

type LeaseOutcome uint8

const (
	LeaseOutcomeUndefined LeaseOutcome = iota
	LeaseOutcomeGranted
	LeaseOutcomeRefused
	LeaseOutcomeTransferred
	LeaseOutcomeRenewed
)

func (o LeaseOutcome) String() string {
	switch o {
	case LeaseOutcomeGranted:
		return "granted"
	case LeaseOutcomeRefused:
		return "refused"
	case LeaseOutcomeTransferred:
		return "transferred"
	case LeaseOutcomeRenewed:
		return "renewed"
	default:
		return ""
	}
}

type TransferReason uint8

const (
	TransferReasonNone TransferReason = iota
	TransferReasonDeadlineExpired
	TransferReasonOwnerDead
)

func (r TransferReason) String() string {
	switch r {
	case TransferReasonDeadlineExpired:
		return "deadline_expired"
	case TransferReasonOwnerDead:
		return "owner_dead"
	default:
		return ""
	}
}

type LeaseRecord struct {
	Outcome        LeaseOutcome
	TransferReason TransferReason
	PreviousOwner  LeaseOwner
	Claimant       LeaseOwner
	Deadline       time.Time
	At             time.Time
}

type ReleaseOutcome uint8

const (
	ReleaseOutcomeNotFound ReleaseOutcome = iota
	ReleaseOutcomeReleased
	ReleaseOutcomeOwnerMismatch
)

var (
	ErrHandoffLeaseNotHeld        = errors.New("durable: handoff lease not held")
	ErrHandoffLeasePathUnresolved = errors.New("durable: handoff lease path not resolved")
)

const (
	HandoffLeaseSidecarSuffix = ".handoff.json"
	handoffLeaseLockSuffix    = ".handoff.lock"
	handoffLockRetryDeadline  = 5 * time.Second
	handoffLockRetryInterval  = 5 * time.Millisecond
)

func HandoffLeasePath(scope Scope) (string, error) {
	return scope.SidecarPath(HandoffLeaseSidecarSuffix)
}

func handoffLockPath(scope Scope) (string, error) {
	return scope.SidecarPath(handoffLeaseLockSuffix)
}

type handoffLeaseRecord struct {
	Owner     string    `json:"owner"`
	Deadline  time.Time `json:"deadline"`
	PID       int       `json:"pid"`
	Hostname  string    `json:"hostname"`
	StartedAt time.Time `json:"started_at"`
}

func LoadHandoffLeaseFile(filesystem fs.FileSystem, path string) (*HandoffLease, error) {
	if !filesystem.Exists(path) {
		return nil, nil
	}
	data, err := filesystem.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("durable: read handoff lease %s: %w", path, err)
	}
	var record handoffLeaseRecord
	if err := json.Unmarshal(data, &record); err != nil {
		return nil, fmt.Errorf("durable: parse handoff lease %s: %w", path, err)
	}
	return &HandoffLease{
		Owner:    LeaseOwner(record.Owner),
		Deadline: record.Deadline,
		ProcessRef: ProcessRef{
			PID:       record.PID,
			Hostname:  record.Hostname,
			StartedAt: record.StartedAt,
		},
	}, nil
}

func SaveHandoffLeaseFile(filesystem fs.FileSystem, projectDir, path string, lease HandoffLease) error {
	if err := fs.RefuseExternalSymlink(filesystem, projectDir, path, false); err != nil {
		return fmt.Errorf("durable: write handoff lease: %w", err)
	}
	record := handoffLeaseRecord{
		Owner:     string(lease.Owner),
		Deadline:  lease.Deadline,
		PID:       lease.ProcessRef.PID,
		Hostname:  lease.ProcessRef.Hostname,
		StartedAt: lease.ProcessRef.StartedAt,
	}
	data, err := json.MarshalIndent(record, "", "  ")
	if err != nil {
		return fmt.Errorf("durable: serialize handoff lease: %w", err)
	}
	if err := filesystem.WriteFileAtomic(path, data); err != nil {
		return fmt.Errorf("durable: write handoff lease %s: %w", path, err)
	}
	return nil
}

type HandoffLeaseStore struct {
	filesystem fs.FileSystem
	locker     LayerLocker
	projectDir string
}

func NewHandoffLeaseStore(filesystem fs.FileSystem, locker LayerLocker, projectDir string) *HandoffLeaseStore {
	return &HandoffLeaseStore{filesystem: filesystem, locker: locker, projectDir: projectDir}
}

func (s *HandoffLeaseStore) withLock(scope Scope, fn func(path string) (LeaseRecord, error)) (LeaseRecord, error) {
	path, err := HandoffLeasePath(scope)
	if err != nil {
		return LeaseRecord{}, fmt.Errorf("%w: %w", ErrHandoffLeasePathUnresolved, err)
	}
	lockPath, err := handoffLockPath(scope)
	if err != nil {
		return LeaseRecord{}, fmt.Errorf("%w: %w", ErrHandoffLeasePathUnresolved, err)
	}

	deadline := time.Now().Add(handoffLockRetryDeadline)
	var unlock func() error
	for {
		unlock, err = s.locker.Lock(lockPath)
		if err == nil {
			break
		}
		if !errors.Is(err, ErrLayerLocked) || time.Now().After(deadline) {
			return LeaseRecord{}, fmt.Errorf("durable: acquire handoff lock %s: %w", lockPath, err)
		}
		time.Sleep(handoffLockRetryInterval)
	}
	defer func() { _ = unlock() }()

	return fn(path)
}

func (s *HandoffLeaseStore) Load(scope Scope) (*HandoffLease, error) {
	path, err := HandoffLeasePath(scope)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrHandoffLeasePathUnresolved, err)
	}
	return LoadHandoffLeaseFile(s.filesystem, path)
}

func (s *HandoffLeaseStore) Claim(scope Scope, policy LeasePolicy, claimant LeaseOwner, ref ProcessRef, ttl time.Duration, now time.Time) (LeaseRecord, error) {
	return s.withLock(scope, func(path string) (LeaseRecord, error) {
		current, loadErr := LoadHandoffLeaseFile(s.filesystem, path)
		if loadErr != nil {
			return LeaseRecord{}, loadErr
		}
		lease, record, claimErr := policy.Claim(current, claimant, ref, ttl, now)
		if claimErr != nil {
			return record, claimErr
		}
		if saveErr := SaveHandoffLeaseFile(s.filesystem, s.projectDir, path, lease); saveErr != nil {
			return LeaseRecord{}, saveErr
		}
		return record, nil
	})
}

func (s *HandoffLeaseStore) Renew(scope Scope, policy LeasePolicy, owner LeaseOwner, ttl time.Duration, now time.Time) (LeaseRecord, error) {
	return s.withLock(scope, func(path string) (LeaseRecord, error) {
		current, loadErr := LoadHandoffLeaseFile(s.filesystem, path)
		if loadErr != nil {
			return LeaseRecord{}, loadErr
		}
		if current == nil {
			return LeaseRecord{}, ErrHandoffLeaseNotHeld
		}
		lease, record, renewErr := policy.Renew(*current, owner, ttl, now)
		if renewErr != nil {
			return record, renewErr
		}
		if saveErr := SaveHandoffLeaseFile(s.filesystem, s.projectDir, path, lease); saveErr != nil {
			return LeaseRecord{}, saveErr
		}
		return record, nil
	})
}

func (s *HandoffLeaseStore) Release(scope Scope, owner LeaseOwner) (ReleaseOutcome, LeaseOwner, error) {
	var outcome ReleaseOutcome
	var previous LeaseOwner
	_, err := s.withLock(scope, func(path string) (LeaseRecord, error) {
		current, loadErr := LoadHandoffLeaseFile(s.filesystem, path)
		if loadErr != nil {
			return LeaseRecord{}, loadErr
		}
		if current == nil {
			outcome = ReleaseOutcomeNotFound
			return LeaseRecord{}, nil
		}
		previous = current.Owner
		if current.Owner != owner {
			outcome = ReleaseOutcomeOwnerMismatch
			return LeaseRecord{}, nil
		}
		if removeErr := s.filesystem.Remove(path); removeErr != nil {
			return LeaseRecord{}, fmt.Errorf("durable: remove handoff lease %s: %w", path, removeErr)
		}
		outcome = ReleaseOutcomeReleased
		return LeaseRecord{}, nil
	})
	return outcome, previous, err
}

type ContinuityHandoff struct {
	store  *HandoffLeaseStore
	scope  Scope
	policy LeasePolicy
}

func NewContinuityHandoff(policy LeasePolicy, filesystem fs.FileSystem, locker LayerLocker, scope Scope, projectDir string) *ContinuityHandoff {
	return &ContinuityHandoff{
		store:  NewHandoffLeaseStore(filesystem, locker, projectDir),
		scope:  scope,
		policy: policy,
	}
}

func (h *ContinuityHandoff) Current() (HandoffLease, bool) {
	lease, err := h.store.Load(h.scope)
	if err != nil || lease == nil {
		return HandoffLease{}, false
	}
	return *lease, true
}

func (h *ContinuityHandoff) Claim(claimant LeaseOwner, ref ProcessRef, ttl time.Duration, now time.Time) (LeaseRecord, error) {
	return h.store.Claim(h.scope, h.policy, claimant, ref, ttl, now)
}

func (h *ContinuityHandoff) Renew(owner LeaseOwner, ttl time.Duration, now time.Time) (LeaseRecord, error) {
	return h.store.Renew(h.scope, h.policy, owner, ttl, now)
}
