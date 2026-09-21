package hookcontract

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestHookTimeoutRegistry_DefaultMatchesOpenCodeBaseline(t *testing.T) {
	registry := NewHookTimeoutRegistry()
	if got := registry.TimeoutFor("unknown-hook"); got != DefaultHookTimeout {
		t.Fatalf("TimeoutFor(unknown) = %s, want %s", got, DefaultHookTimeout)
	}
	if DefaultHookTimeout.Milliseconds() != 8000 {
		t.Fatalf("DefaultHookTimeout = %dms, want 8000ms to preserve AISPEC_OPENCODE_VALIDATOR_TIMEOUT_MS default", DefaultHookTimeout.Milliseconds())
	}
}

func TestHookTimeoutRegistry_DeclareAndOverride(t *testing.T) {
	registry := NewHookTimeoutRegistry()
	if err := registry.Declare("validate-preload", 3*time.Second); err != nil {
		t.Fatalf("Declare unexpected error: %v", err)
	}
	if got := registry.TimeoutFor("validate-preload"); got != 3*time.Second {
		t.Fatalf("TimeoutFor(validate-preload) = %s, want 3s", got)
	}
}

func TestHookTimeoutRegistry_RejectsInvalidDeclaration(t *testing.T) {
	registry := NewHookTimeoutRegistry()
	if err := registry.Declare("", time.Second); !errors.Is(err, ErrInvalidHookTimeout) {
		t.Fatalf("Declare(empty hook) error = %v, want ErrInvalidHookTimeout", err)
	}
	if err := registry.Declare("hook", 0); !errors.Is(err, ErrInvalidHookTimeout) {
		t.Fatalf("Declare(zero timeout) error = %v, want ErrInvalidHookTimeout", err)
	}
	if err := registry.Declare("hook", -time.Second); !errors.Is(err, ErrInvalidHookTimeout) {
		t.Fatalf("Declare(negative timeout) error = %v, want ErrInvalidHookTimeout", err)
	}
}

func TestMeasureHook_ExceedingTimeoutIsDenial(t *testing.T) {
	result, elapsed, timedOut := MeasureHook(context.Background(), "slow-hook", 20*time.Millisecond, func(ctx context.Context) Result {
		select {
		case <-time.After(200 * time.Millisecond):
			return NewAllow()
		case <-ctx.Done():
			return NewAllow()
		}
	})
	if !timedOut {
		t.Fatal("timedOut = false, want true when the hook exceeds its declared timeout")
	}
	if result.Decision() != DecisionBlock {
		t.Fatalf("Decision() = %v, want DecisionBlock — timeout must be treated as denial, never approval", result.Decision())
	}
	if elapsed <= 0 {
		t.Fatal("elapsed must be measurable and greater than zero")
	}
}

func TestMeasureHook_WithinTimeoutPassesThrough(t *testing.T) {
	result, elapsed, timedOut := MeasureHook(context.Background(), "fast-hook", 200*time.Millisecond, func(ctx context.Context) Result {
		return NewAllow()
	})
	if timedOut {
		t.Fatal("timedOut = true, want false when the hook finishes within its declared timeout")
	}
	if result.Decision() != DecisionAllow {
		t.Fatalf("Decision() = %v, want DecisionAllow", result.Decision())
	}
	if elapsed < 0 {
		t.Fatal("elapsed must be measurable")
	}
}

func TestMeasureHook_PropagatesCancellationToFn(t *testing.T) {
	fnSawCancellation := make(chan bool, 1)
	MeasureHook(context.Background(), "cancel-aware-hook", 20*time.Millisecond, func(ctx context.Context) Result {
		<-ctx.Done()
		fnSawCancellation <- errors.Is(ctx.Err(), context.DeadlineExceeded)
		return NewAllow()
	})

	select {
	case sawDeadline := <-fnSawCancellation:
		if !sawDeadline {
			t.Fatal("fn context must report DeadlineExceeded once the declared timeout elapses")
		}
	case <-time.After(time.Second):
		t.Fatal("fn never observed context cancellation")
	}
}
