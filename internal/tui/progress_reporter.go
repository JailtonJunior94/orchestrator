package tui

import "github.com/jailtonjunior/orchestrator/internal/runtime"

// tuiProgressReporter converts ProgressReporter calls into channel sends so
// the engine goroutine can communicate with the Bubbletea event loop.
type tuiProgressReporter struct {
	ch chan<- progressEvent
}

// newProgressReporter returns a ProgressReporter that publishes events to ch.
func newProgressReporter(ch chan<- progressEvent) runtime.ProgressReporter {
	return &tuiProgressReporter{ch: ch}
}

// Reset replaces the destination channel while preserving the same reporter
// instance already referenced by the engine.
func (r *tuiProgressReporter) Reset(ch chan<- progressEvent) {
	r.ch = ch
}

// StepStarted emits a stepStartedMsg for the given step.
func (r *tuiProgressReporter) StepStarted(step runtime.ProgressStep) {
	r.trySend(stepStartedMsg{step: stepItem{
		name:     step.Name,
		provider: step.Provider,
		status:   "running",
	}})
}

// StepFinished emits a stepFinishedMsg for the given step.
func (r *tuiProgressReporter) StepFinished(step runtime.ProgressStep) {
	r.trySend(stepFinishedMsg{step: stepItem{
		name:     step.Name,
		provider: step.Provider,
		status:   step.Status,
		duration: step.Duration,
	}})
}

// OutputChunk emits an outputChunkMsg with the incremental provider output.
func (r *tuiProgressReporter) OutputChunk(stepName string, chunk []byte) {
	r.trySend(outputChunkMsg{stepName: stepName, chunk: chunk})
}

// WaitingApproval emits a waitApprovalMsg requesting HITL action.
func (r *tuiProgressReporter) WaitingApproval(stepName string, output string) {
	r.trySend(waitApprovalMsg{stepName: stepName, output: output})
}

// RunCompleted emits a runCompletedMsg to signal successful workflow end.
func (r *tuiProgressReporter) RunCompleted(_ string, _ string) {
	r.trySend(runCompletedMsg{})
}

// RunFailed emits a runFailedMsg to signal workflow failure.
func (r *tuiProgressReporter) RunFailed(_ string, _ error) {
	r.trySend(runFailedMsg{})
}

// trySend sends e to the channel without blocking. Events are dropped when the
// channel is full or already closed to ensure the engine goroutine never
// stalls and never panics on a closed channel (e.g. when the TUI exits first).
func (r *tuiProgressReporter) trySend(e progressEvent) {
	defer func() { recover() }() //nolint:errcheck // intentional panic guard for closed channel
	select {
	case r.ch <- e:
	default:
	}
}

// Compile-time interface conformance check.
var _ runtime.ProgressReporter = (*tuiProgressReporter)(nil)
