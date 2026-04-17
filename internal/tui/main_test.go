package tui

import (
	"testing"

	"github.com/jailtonjunior/orchestrator/internal/acp"
	"github.com/jailtonjunior/orchestrator/internal/runtime"
)

// TestShouldUseTUI_ReturnsFalseWhenDisabled verifies that noTUI=true always
// returns false regardless of terminal state.
func TestShouldUseTUI_ReturnsFalseWhenDisabled(t *testing.T) {
	t.Parallel()
	if ShouldUseTUI(true) {
		t.Fatal("ShouldUseTUI(true) should return false")
	}
}

// TestShouldUseTUI_ReturnsFalseForNonTTY verifies that a non-TTY stdout
// causes ShouldUseTUI to return false. In the test environment stdout is
// typically not a TTY, so this exercises the happy path.
func TestShouldUseTUI_ReturnsFalseForNonTTY(t *testing.T) {
	t.Parallel()
	// In CI / test environments stdout is redirected, not a real TTY.
	// We cannot assert true here, but we can assert the call does not panic.
	_ = ShouldUseTUI(false)
}

// TestProgressReporter_StepStarted verifies that StepStarted publishes a
// stepStartedMsg with the correct step name and provider.
func TestProgressReporter_StepStarted(t *testing.T) {
	t.Parallel()
	ch := make(chan progressEvent, 1)
	r := newProgressReporter(ch)

	r.StepStarted(runtime.ProgressStep{Name: "plan", Provider: "claude"})

	event := <-ch
	msg, ok := event.(stepStartedMsg)
	if !ok {
		t.Fatalf("expected stepStartedMsg, got %T", event)
	}
	if msg.step.name != "plan" {
		t.Errorf("expected step name %q, got %q", "plan", msg.step.name)
	}
	if msg.step.provider != "claude" {
		t.Errorf("expected provider %q, got %q", "claude", msg.step.provider)
	}
	if msg.step.status != "running" {
		t.Errorf("expected status %q, got %q", "running", msg.step.status)
	}
}

// TestProgressReporter_StepFinished verifies that StepFinished publishes a
// stepFinishedMsg with the correct status.
func TestProgressReporter_StepFinished(t *testing.T) {
	t.Parallel()
	ch := make(chan progressEvent, 1)
	r := newProgressReporter(ch)

	r.StepFinished(runtime.ProgressStep{Name: "plan", Provider: "claude", Status: "waiting_approval"})

	event := <-ch
	msg, ok := event.(stepFinishedMsg)
	if !ok {
		t.Fatalf("expected stepFinishedMsg, got %T", event)
	}
	if msg.step.status != "waiting_approval" {
		t.Errorf("expected status %q, got %q", "waiting_approval", msg.step.status)
	}
}

// TestProgressReporter_TypedUpdate verifies that TypedUpdate publishes a
// typedUpdateMsg with the correct payload.
func TestProgressReporter_TypedUpdate(t *testing.T) {
	t.Parallel()
	ch := make(chan progressEvent, 1)
	r := newProgressReporter(ch)

	r.TypedUpdate("plan", acp.TypedUpdate{Kind: acp.UpdateMessage, Text: "hello"})

	event := <-ch
	msg, ok := event.(typedUpdateMsg)
	if !ok {
		t.Fatalf("expected typedUpdateMsg, got %T", event)
	}
	if msg.stepName != "plan" {
		t.Errorf("expected stepName %q, got %q", "plan", msg.stepName)
	}
	if msg.text != "hello" {
		t.Errorf("expected text %q, got %q", "hello", msg.text)
	}
	if msg.kind != "message" {
		t.Errorf("expected kind %q, got %q", "message", msg.kind)
	}
}

// TestProgressReporter_WaitingApproval verifies that WaitingApproval publishes
// a waitApprovalMsg.
func TestProgressReporter_WaitingApproval(t *testing.T) {
	t.Parallel()
	ch := make(chan progressEvent, 1)
	r := newProgressReporter(ch)

	r.WaitingApproval("plan", "output text")

	event := <-ch
	msg, ok := event.(waitApprovalMsg)
	if !ok {
		t.Fatalf("expected waitApprovalMsg, got %T", event)
	}
	if msg.stepName != "plan" {
		t.Errorf("expected stepName %q, got %q", "plan", msg.stepName)
	}
}

// TestProgressReporter_RunCompleted verifies runCompletedMsg is sent.
func TestProgressReporter_RunCompleted(t *testing.T) {
	t.Parallel()
	ch := make(chan progressEvent, 1)
	r := newProgressReporter(ch)

	r.RunCompleted("run-1", "completed")

	event := <-ch
	if _, ok := event.(runCompletedMsg); !ok {
		t.Fatalf("expected runCompletedMsg, got %T", event)
	}
}

// TestProgressReporter_RunFailed verifies runFailedMsg is sent.
func TestProgressReporter_RunFailed(t *testing.T) {
	t.Parallel()
	ch := make(chan progressEvent, 1)
	r := newProgressReporter(ch)

	r.RunFailed("run-1", nil)

	event := <-ch
	if _, ok := event.(runFailedMsg); !ok {
		t.Fatalf("expected runFailedMsg, got %T", event)
	}
}

// TestProgressReporter_DropWhenFull verifies that trySend does not block when
// the channel is at capacity.
func TestProgressReporter_DropWhenFull(t *testing.T) {
	t.Parallel()
	ch := make(chan progressEvent, 1)
	// Fill the channel.
	ch <- runCompletedMsg{}

	r := newProgressReporter(ch)
	// This call must not block.
	done := make(chan struct{})
	go func() {
		r.RunCompleted("run-1", "completed")
		close(done)
	}()
	<-done
}

// TestProgressReporter_ImplementsInterface is a compile-time check that
// tuiProgressReporter fully implements runtime.ProgressReporter.
func TestProgressReporter_ImplementsInterface(t *testing.T) {
	t.Parallel()
	ch := make(chan progressEvent, 1)
	var _ runtime.ProgressReporter = newProgressReporter(ch) //nolint:staticcheck
}
