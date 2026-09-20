package hookaudit

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

var ErrInvalidEntry = errors.New("invalid hook audit entry")

type Entry struct {
	Timestamp  time.Time `json:"ts"`
	Event      string    `json:"event"`
	Provider   string    `json:"provider"`
	Hook       string    `json:"hook"`
	Decision   string    `json:"decision"`
	Reason     string    `json:"reason,omitempty"`
	PolicyID   string    `json:"policy_id,omitempty"`
	GateID     string    `json:"gate_id,omitempty"`
	DurationMS int64     `json:"duration_ms"`
	Evidence   []string  `json:"evidence,omitempty"`
}

func NewEntry(timestamp time.Time, event, provider, hook, decision string, durationMS int64) (Entry, error) {
	if strings.TrimSpace(event) == "" {
		return Entry{}, fmt.Errorf("%w: empty event", ErrInvalidEntry)
	}
	if strings.TrimSpace(hook) == "" {
		return Entry{}, fmt.Errorf("%w: empty hook", ErrInvalidEntry)
	}
	if strings.TrimSpace(decision) == "" {
		return Entry{}, fmt.Errorf("%w: empty decision", ErrInvalidEntry)
	}
	if durationMS < 0 {
		return Entry{}, fmt.Errorf("%w: negative duration", ErrInvalidEntry)
	}
	return Entry{
		Timestamp:  timestamp,
		Event:      event,
		Provider:   provider,
		Hook:       hook,
		Decision:   decision,
		DurationMS: durationMS,
	}, nil
}

func (e Entry) WithReason(reason, policyID, gateID string) Entry {
	cloned := e
	cloned.Reason = reason
	cloned.PolicyID = policyID
	cloned.GateID = gateID
	return cloned
}

func (e Entry) WithEvidence(evidence ...string) Entry {
	cloned := e
	cloned.Evidence = append([]string(nil), evidence...)
	return cloned
}
