package hooks

import (
	"context"
	"fmt"
	"sync"
)

type MemoryEvidenceEntry struct {
	Point  string
	Detail string
}

type MemoryEvidenceRecorder struct {
	mu      sync.Mutex
	entries []MemoryEvidenceEntry
	errors  []string
}

func NewMemoryEvidenceRecorder() *MemoryEvidenceRecorder {
	return &MemoryEvidenceRecorder{}
}

func (r *MemoryEvidenceRecorder) record(point, detail string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.entries = append(r.entries, MemoryEvidenceEntry{Point: point, Detail: detail})
}

func (r *MemoryEvidenceRecorder) recordError(point string, err error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.errors = append(r.errors, fmt.Sprintf("%s: %v", point, err))
}

func (r *MemoryEvidenceRecorder) Entries() []MemoryEvidenceEntry {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]MemoryEvidenceEntry, len(r.entries))
	copy(out, r.entries)
	return out
}

func (r *MemoryEvidenceRecorder) Errors() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]string, len(r.errors))
	copy(out, r.errors)
	return out
}

type MemoryEvidenceHook struct {
	recorder *MemoryEvidenceRecorder
}

var _ Hook = (*MemoryEvidenceHook)(nil)

func NewMemoryEvidenceHook(recorder *MemoryEvidenceRecorder) *MemoryEvidenceHook {
	return &MemoryEvidenceHook{recorder: recorder}
}

func (h *MemoryEvidenceHook) Name() string { return "memory_evidence" }

func (h *MemoryEvidenceHook) Run(_ context.Context, evt Event) error {
	switch e := evt.(type) {
	case MemoryFactRecordedEvent:
		h.recorder.record(e.Kind(), fmt.Sprintf("session=%s cli=%s task=%s count=%d", e.SessionID, e.CLI, e.TaskFileName, e.Count))
	case MemoryFactArchivedEvent:
		h.recorder.record(e.Kind(), fmt.Sprintf("session=%s cli=%s task=%s count=%d", e.SessionID, e.CLI, e.TaskFileName, e.Count))
	case MemoryFactPromotedEvent:
		h.recorder.record(e.Kind(), fmt.Sprintf("session=%s cli=%s task=%s count=%d", e.SessionID, e.CLI, e.TaskFileName, e.Count))
	case MemoryContradictionDetectedEvent:
		h.recorder.record(e.Kind(), fmt.Sprintf("session=%s cli=%s task=%s count=%d", e.SessionID, e.CLI, e.TaskFileName, e.Count))
	case MemorySecretRedactedEvent:
		h.recorder.record(e.Kind(), fmt.Sprintf("session=%s cli=%s task=%s count=%d", e.SessionID, e.CLI, e.TaskFileName, e.Count))
	case MemoryCompactionExecutedEvent:
		h.recorder.record(e.Kind(), fmt.Sprintf("session=%s cli=%s task=%s count=%d", e.SessionID, e.CLI, e.TaskFileName, e.Count))
	case MemoryBatonTransferredEvent:
		h.recorder.record(e.Kind(), fmt.Sprintf("session=%s cli=%s task=%s claimed=%v", e.SessionID, e.CLI, e.TaskFileName, e.Claimed))
	default:
		h.recorder.recordError("memory_evidence", fmt.Errorf("unexpected event type %T", evt))
	}
	return nil
}
