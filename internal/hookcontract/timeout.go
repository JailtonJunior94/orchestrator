package hookcontract

import (
	"context"
	"errors"
	"fmt"
	"time"
)

const DefaultHookTimeout = 8000 * time.Millisecond

var (
	ErrInvalidHookTimeout = errors.New("invalid hook timeout declaration")
	ErrHookTimedOut       = errors.New("hook timed out — timeout is treated as denial, never approval")
)

type HookTimeoutRegistry struct {
	timeouts map[string]time.Duration
}

func NewHookTimeoutRegistry() *HookTimeoutRegistry {
	return &HookTimeoutRegistry{timeouts: make(map[string]time.Duration)}
}

func (r *HookTimeoutRegistry) Declare(hook string, timeout time.Duration) error {
	if hook == "" {
		return fmt.Errorf("%w: hook name required", ErrInvalidHookTimeout)
	}
	if timeout <= 0 {
		return fmt.Errorf("%w: timeout must be positive for hook %q", ErrInvalidHookTimeout, hook)
	}
	r.timeouts[hook] = timeout
	return nil
}

func (r *HookTimeoutRegistry) TimeoutFor(hook string) time.Duration {
	if d, ok := r.timeouts[hook]; ok {
		return d
	}
	return DefaultHookTimeout
}

func NewHookTimeoutResult(hook string, timeout time.Duration) Result {
	reason := fmt.Sprintf("hook %q timed out after %s — timeout is treated as denial, never approval", hook, timeout)
	result, err := NewResult(DecisionBlock, reason, "", "hook-timeout-guard")
	if err != nil {
		return NewNotApplicable(reason)
	}
	return result
}

func MeasureHook(ctx context.Context, hook string, timeout time.Duration, fn func(context.Context) Result) (result Result, elapsed time.Duration, timedOut bool) {
	childCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	start := time.Now()
	done := make(chan Result, 1)
	go func() {
		done <- fn(childCtx)
	}()
	select {
	case r := <-done:
		return r, time.Since(start), false
	case <-childCtx.Done():
		return NewHookTimeoutResult(hook, timeout), time.Since(start), true
	}
}
