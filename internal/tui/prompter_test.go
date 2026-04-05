package tui

import (
	"context"
	"testing"
	"time"

	"github.com/jailtonjunior/orchestrator/internal/hitl"
)

func TestPrompt_SendsRequestAndReceivesResponse(t *testing.T) {
	p := NewTUIPrompter()
	ctx := context.Background()

	go func() {
		// Simulate TUI: wait for request, then respond.
		cmd := p.WaitForRequest()
		msg := cmd()
		resp, ok := msg.(promptResponseMsg)
		if !ok {
			t.Errorf("expected promptResponseMsg, got %T", msg)
			return
		}
		if resp.req.output != "hello" {
			t.Errorf("expected output %q, got %q", "hello", resp.req.output)
		}
		p.Respond(hitl.PromptResult{Action: hitl.ActionApprove})
	}()

	result, err := p.Prompt(ctx, "hello")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Action != hitl.ActionApprove {
		t.Errorf("expected ActionApprove, got %v", result.Action)
	}
}

func TestPrompt_ReturnsErrorOnContextCancellation_BeforeSend(t *testing.T) {
	p := NewTUIPrompter()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := p.Prompt(ctx, "irrelevant")
	if err == nil {
		t.Fatal("expected context cancellation error")
	}
}

func TestPrompt_ReturnsErrorOnContextCancellation_BeforeReceive(t *testing.T) {
	p := NewTUIPrompter()
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	// Consume the request but never respond.
	go func() {
		<-p.requestCh
	}()

	_, err := p.Prompt(ctx, "waiting")
	if err == nil {
		t.Fatal("expected context deadline exceeded error")
	}
}

func TestWaitForRequest_UnblocksOnClose(t *testing.T) {
	p := NewTUIPrompter()
	done := make(chan struct{})

	go func() {
		defer close(done)
		cmd := p.WaitForRequest()
		msg := cmd()
		if msg != nil {
			t.Errorf("expected nil msg after Close, got %T", msg)
		}
	}()

	p.Close()

	select {
	case <-done:
	case <-time.After(500 * time.Millisecond):
		t.Fatal("WaitForRequest goroutine did not unblock after Close")
	}
}

// TestIntegration_EngineAndHITLCycle simulates a full engine ↔ TUI HITL cycle
// using a fake engine goroutine, TUIPrompter, and HITLBar together.
func TestIntegration_EngineAndHITLCycle(t *testing.T) {
	p := NewTUIPrompter()
	defer p.Close()
	ctx := context.Background()

	// Fake engine: sends a prompt request and expects ActionRedo back.
	engineDone := make(chan hitl.PromptResult, 1)
	go func() {
		result, err := p.Prompt(ctx, "engine output")
		if err != nil {
			t.Errorf("engine Prompt error: %v", err)
		}
		engineDone <- result
	}()

	// Fake TUI: WaitForRequest returns the pending request, then responds.
	cmd := p.WaitForRequest()
	msg := cmd()
	resp, ok := msg.(promptResponseMsg)
	if !ok {
		t.Fatalf("expected promptResponseMsg, got %T", msg)
	}
	if resp.req.output != "engine output" {
		t.Errorf("unexpected output: %q", resp.req.output)
	}

	p.Respond(hitl.PromptResult{Action: hitl.ActionRedo})

	select {
	case result := <-engineDone:
		if result.Action != hitl.ActionRedo {
			t.Errorf("expected ActionRedo, got %v", result.Action)
		}
	case <-time.After(500 * time.Millisecond):
		t.Fatal("engine did not receive response in time")
	}
}
