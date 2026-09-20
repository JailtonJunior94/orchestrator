package runtime

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/client"
	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/events"
	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/hooks"
	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/specs"
)

type eventLoopFakeClient struct {
	updates chan events.Event
}

func (f *eventLoopFakeClient) Open(context.Context, specs.Launcher, string) error { return nil }

func (f *eventLoopFakeClient) Updates() <-chan events.Event { return f.updates }

func (f *eventLoopFakeClient) Err() error { return nil }

func (f *eventLoopFakeClient) Close() error { return nil }

func (f *eventLoopFakeClient) SlowPublishes() uint64 { return 0 }

func (f *eventLoopFakeClient) DroppedUpdates() uint64 { return 0 }

func (f *eventLoopFakeClient) SetChildEnv([]string) {}

func (f *eventLoopFakeClient) SetHandshakeWaiter(client.HandshakeWaiter) {}

var _ client.Client = (*eventLoopFakeClient)(nil)

type failingHook struct {
	err error
}

func (h *failingHook) Name() string { return "failing-hook" }

func (h *failingHook) Run(context.Context, hooks.Event) error { return h.err }

func TestRunEventLoop_ToolCallPreDispatchErrorIsPropagated(t *testing.T) {
	t.Parallel()

	hookErr := errors.New("boom pre_dispatch")
	disp := hooks.New()
	disp.Register(hooks.PointToolCallPreDispatch, &failingHook{err: hookErr})

	toolCallStart, err := events.NewToolCallStart(time.Now(), events.NewToolCallID("tc1"), "Bash", "{}", nil)
	if err != nil {
		t.Fatalf("NewToolCallStart: %v", err)
	}

	fc := &eventLoopFakeClient{updates: make(chan events.Event, 1)}
	fc.updates <- toolCallStart
	close(fc.updates)

	r := &ACPRunner{spec: specs.NewCatalog().Claude(), clock: NewCatalog().RealClock()}
	wd := NewActivityWatchdog(0, func(error) {}, r.clock)

	result := r.runEventLoop(context.Background(), fc, wd, Job{Quiet: true}, disp, events.NewToolCallCounters(), nil, "claude")
	if result.err == nil {
		t.Fatal("runEventLoop did not propagate the tool_call.pre_dispatch hook error")
	}
	if !errors.Is(result.err, hookErr) {
		t.Errorf("result.err = %v; want wrapping %v", result.err, hookErr)
	}
}

func TestRunEventLoop_ToolCallPostCompleteErrorIsPropagated(t *testing.T) {
	t.Parallel()

	hookErr := errors.New("boom post_complete")
	disp := hooks.New()
	disp.Register(hooks.PointToolCallPostComplete, &failingHook{err: hookErr})

	toolCallUpdate, err := events.NewToolCallUpdate(time.Now(), events.NewToolCallID("tc1"), "output", "completed", true, nil)
	if err != nil {
		t.Fatalf("NewToolCallUpdate: %v", err)
	}

	fc := &eventLoopFakeClient{updates: make(chan events.Event, 1)}
	fc.updates <- toolCallUpdate
	close(fc.updates)

	r := &ACPRunner{spec: specs.NewCatalog().Claude(), clock: NewCatalog().RealClock()}
	wd := NewActivityWatchdog(0, func(error) {}, r.clock)

	result := r.runEventLoop(context.Background(), fc, wd, Job{Quiet: true}, disp, events.NewToolCallCounters(), nil, "claude")
	if result.err == nil {
		t.Fatal("runEventLoop did not propagate the tool_call.post_complete hook error")
	}
	if !errors.Is(result.err, hookErr) {
		t.Errorf("result.err = %v; want wrapping %v", result.err, hookErr)
	}
}
