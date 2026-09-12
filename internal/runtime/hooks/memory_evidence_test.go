package hooks_test

import (
	"context"
	"testing"

	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/hooks"
)

func TestMemoryEvidenceHook_Name(t *testing.T) {
	t.Parallel()
	h := hooks.NewMemoryEvidenceHook(hooks.NewMemoryEvidenceRecorder())
	if h.Name() != "memory_evidence" {
		t.Errorf("Name() = %q; want memory_evidence", h.Name())
	}
}

func TestMemoryEvidenceHook_RecordsKnownEvents(t *testing.T) {
	t.Parallel()
	recorder := hooks.NewMemoryEvidenceRecorder()
	h := hooks.NewMemoryEvidenceHook(recorder)

	err := h.Run(context.Background(), hooks.MemoryFactRecordedEvent{
		SessionID: "sess-1", CLI: "claude", TaskFileName: "task-8.0.md", Count: 3,
	})
	if err != nil {
		t.Fatalf("Run() retornou erro para evento conhecido: %v", err)
	}

	entries := recorder.Entries()
	if len(entries) != 1 {
		t.Fatalf("esperava 1 entrada registrada, obtido %d", len(entries))
	}
	if entries[0].Point != hooks.PointMemoryFactRecorded {
		t.Errorf("Point = %q; want %q", entries[0].Point, hooks.PointMemoryFactRecorded)
	}
}

func TestMemoryEvidenceHook_NeverPropagatesError(t *testing.T) {
	t.Parallel()
	recorder := hooks.NewMemoryEvidenceRecorder()
	h := hooks.NewMemoryEvidenceHook(recorder)

	err := h.Run(context.Background(), hooks.RuntimePreOpenEvent{})
	if err != nil {
		t.Fatalf("Run() propagou erro para evento inesperado; deveria compor na evidência: %v", err)
	}

	errs := recorder.Errors()
	if len(errs) != 1 {
		t.Fatalf("esperava 1 erro composto na evidência, obtido %d", len(errs))
	}
}

func TestMemoryEvidenceHook_DoesNotAbortFanOutForSubsequentHooks(t *testing.T) {
	t.Parallel()
	recorder := hooks.NewMemoryEvidenceRecorder()
	disp := hooks.New()
	disp.Register(hooks.PointMemoryFactRecorded, hooks.NewMemoryEvidenceHook(recorder))

	secondHookRan := false
	disp.Register(hooks.PointMemoryFactRecorded, hookFunc(func(context.Context, hooks.Event) error {
		secondHookRan = true
		return nil
	}))

	if err := disp.Dispatch(context.Background(), hooks.PointMemoryFactRecorded, hooks.RuntimePreOpenEvent{}); err != nil {
		t.Fatalf("Dispatch retornou erro inesperado: %v", err)
	}
	if !secondHookRan {
		t.Fatal("hook registrado após o hook de memória não executou (fan-out abortado indevidamente)")
	}
}

type hookFunc func(ctx context.Context, evt hooks.Event) error

func (f hookFunc) Name() string { return "test_hook" }

func (f hookFunc) Run(ctx context.Context, evt hooks.Event) error { return f(ctx, evt) }
