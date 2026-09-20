package hookcontract

import (
	"errors"
	"fmt"
	"strings"
)

type Decision int

const (
	DecisionAllow Decision = iota + 1
	DecisionBlock
	DecisionWarn
	DecisionNotApplicable
	DecisionError
)

func (d Decision) Valid() bool {
	return d >= DecisionAllow && d <= DecisionError
}

func (d Decision) String() string {
	switch d {
	case DecisionAllow:
		return "ALLOW"
	case DecisionBlock:
		return "BLOCK"
	case DecisionWarn:
		return "WARN"
	case DecisionNotApplicable:
		return "NOT_APPLICABLE"
	case DecisionError:
		return "ERROR"
	default:
		return "UNKNOWN"
	}
}

type EvidenceRef string

var ErrInvalidResult = errors.New("invalid hook result")

type Result struct {
	decision Decision
	reason   string
	policyID string
	gateID   string
	evidence []EvidenceRef
	metadata map[string]string
	valid    bool
}

func NewResult(decision Decision, reason, policyID, gateID string) (Result, error) {
	if !decision.Valid() {
		return Result{}, fmt.Errorf("%w: invalid decision %d", ErrInvalidResult, int(decision))
	}
	if decision == DecisionBlock {
		if strings.TrimSpace(reason) == "" {
			return Result{}, fmt.Errorf("%w: block requires a reason", ErrInvalidResult)
		}
		if strings.TrimSpace(policyID) == "" && strings.TrimSpace(gateID) == "" {
			return Result{}, fmt.Errorf("%w: block requires policyID or gateID", ErrInvalidResult)
		}
	}
	return Result{
		decision: decision,
		reason:   reason,
		policyID: policyID,
		gateID:   gateID,
		valid:    true,
	}, nil
}

func NewAllow() Result {
	return Result{decision: DecisionAllow, valid: true}
}

func NewNotApplicable(reason string) Result {
	return Result{decision: DecisionNotApplicable, reason: reason, valid: true}
}

func (r Result) WithEvidence(refs ...EvidenceRef) Result {
	cloned := r
	cloned.evidence = append([]EvidenceRef(nil), refs...)
	return cloned
}

func (r Result) WithMetadata(metadata map[string]string) Result {
	cloned := r
	clonedMap := make(map[string]string, len(metadata))
	for key, value := range metadata {
		clonedMap[key] = value
	}
	cloned.metadata = clonedMap
	return cloned
}

func (r Result) Decision() Decision {
	return r.decision
}

func (r Result) Reason() string {
	return r.reason
}

func (r Result) PolicyID() string {
	return r.policyID
}

func (r Result) GateID() string {
	return r.gateID
}

func (r Result) Evidence() []EvidenceRef {
	return append([]EvidenceRef(nil), r.evidence...)
}

func (r Result) Metadata() map[string]string {
	cloned := make(map[string]string, len(r.metadata))
	for key, value := range r.metadata {
		cloned[key] = value
	}
	return cloned
}

func (r Result) Valid() bool {
	return r.valid
}
