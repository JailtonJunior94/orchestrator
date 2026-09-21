package hooks_test

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/JailtonJunior94/ai-spec-harness/internal/hookcontract"
	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/hooks"
	"github.com/JailtonJunior94/ai-spec-harness/internal/telemetry"
)

// --- helpers ---

type recordHook struct {
	name   string
	called *[]string
	err    error
}

func (h *recordHook) Name() string { return h.name }
func (h *recordHook) Run(_ context.Context, _ hooks.Event) error {
	*h.called = append(*h.called, h.name)
	return h.err
}

// T-HOOK-01: ordem de registro respeitada na execução.
func TestDispatcher_OrderPreserved(t *testing.T) {
	d := hooks.New()
	var called []string

	h1 := &recordHook{name: "h1", called: &called}
	h2 := &recordHook{name: "h2", called: &called}
	h3 := &recordHook{name: "h3", called: &called}

	d.Register(hooks.PointRuntimePreOpen, h1)
	d.Register(hooks.PointRuntimePreOpen, h2)
	d.Register(hooks.PointRuntimePreOpen, h3)

	evt := hooks.RuntimePreOpenEvent{WorkDir: "/tmp", SpecID: "test"}
	if err := d.Dispatch(context.Background(), hooks.PointRuntimePreOpen, evt); err != nil {
		t.Fatalf("Dispatch inesperado: %v", err)
	}

	want := []string{"h1", "h2", "h3"}
	if len(called) != len(want) {
		t.Fatalf("chamadas = %v; quero %v", called, want)
	}
	for i, name := range want {
		if called[i] != name {
			t.Errorf("chamada[%d] = %q; quero %q", i, called[i], name)
		}
	}
}

// T-HOOK-02: erro em h2 → h3 nunca chamado (abort-on-first-error).
func TestDispatcher_AbortOnFirstError(t *testing.T) {
	d := hooks.New()
	var called []string

	errH2 := errors.New("falha em h2")

	h1 := &recordHook{name: "h1", called: &called}
	h2 := &recordHook{name: "h2", called: &called, err: errH2}
	h3 := &recordHook{name: "h3", called: &called}

	d.Register(hooks.PointPromptPostBuild, h1)
	d.Register(hooks.PointPromptPostBuild, h2)
	d.Register(hooks.PointPromptPostBuild, h3)

	prompt := "prompt de teste"
	evt := hooks.PromptBuildEvent{Prompt: &prompt, Spec: "spec-1"}
	err := d.Dispatch(context.Background(), hooks.PointPromptPostBuild, evt)
	if err == nil {
		t.Fatal("esperava erro; Dispatch retornou nil")
	}
	if !errors.Is(err, errH2) {
		t.Errorf("erro = %v; deve envolver errH2", err)
	}

	if len(called) != 2 {
		t.Fatalf("chamadas = %v; quero apenas h1, h2 (h3 não deve executar)", called)
	}
	if called[0] != "h1" || called[1] != "h2" {
		t.Errorf("chamadas = %v; quero [h1 h2]", called)
	}
}

// T-HOOK-03: Dispatch em ponto sem registro retorna nil sem panic.
func TestDispatcher_UnknownPointReturnsNil(t *testing.T) {
	d := hooks.New()

	evt := hooks.RuntimePreOpenEvent{WorkDir: "/tmp"}
	err := d.Dispatch(context.Background(), "nonexistent.point", evt)
	if err != nil {
		t.Fatalf("ponto desconhecido deve retornar nil; got: %v", err)
	}
}

// Verifica que o erro é envelopado com o nome do hook e o ponto.
func TestDispatcher_ErrorWrapping(t *testing.T) {
	d := hooks.New()
	var called []string

	errInner := errors.New("erro interno")
	h := &recordHook{name: "meu-hook", called: &called, err: errInner}
	d.Register(hooks.PointSessionPostEnd, h)

	evt := hooks.SessionPostEndEvent{}
	err := d.Dispatch(context.Background(), hooks.PointSessionPostEnd, evt)
	if err == nil {
		t.Fatal("esperava erro")
	}

	want := fmt.Sprintf("hook %s in %s: %s", "meu-hook", hooks.PointSessionPostEnd, errInner)
	if err.Error() != want {
		t.Errorf("erro = %q; quero %q", err.Error(), want)
	}
	if !errors.Is(err, errInner) {
		t.Errorf("errors.Is deve localizar errInner")
	}
}

// Verifica que hooks em pontos distintos são independentes.
func TestDispatcher_IndependentPoints(t *testing.T) {
	d := hooks.New()
	var calledA, calledB []string

	hA := &recordHook{name: "hookA", called: &calledA}
	hB := &recordHook{name: "hookB", called: &calledB}

	d.Register(hooks.PointRuntimePreOpen, hA)
	d.Register(hooks.PointSessionPostEnd, hB)

	evtA := hooks.RuntimePreOpenEvent{WorkDir: "/tmp"}
	if err := d.Dispatch(context.Background(), hooks.PointRuntimePreOpen, evtA); err != nil {
		t.Fatalf("Dispatch A: %v", err)
	}
	evtB := hooks.SessionPostEndEvent{}
	if err := d.Dispatch(context.Background(), hooks.PointSessionPostEnd, evtB); err != nil {
		t.Fatalf("Dispatch B: %v", err)
	}

	if len(calledA) != 1 || calledA[0] != "hookA" {
		t.Errorf("calledA = %v; quero [hookA]", calledA)
	}
	if len(calledB) != 1 || calledB[0] != "hookB" {
		t.Errorf("calledB = %v; quero [hookB]", calledB)
	}
}

// Verifica thread-safety: Register e Dispatch concorrentes não causam race.
func TestDispatcher_ConcurrentSafe(t *testing.T) {
	d := hooks.New()
	var called []string
	h := &recordHook{name: "concurrent", called: &called}

	done := make(chan struct{})
	go func() {
		d.Register(hooks.PointToolCallPreDispatch, h)
		close(done)
	}()
	<-done

	// Dispatch após Register — sem race esperada.
	for i := 0; i < 10; i++ {
		evt := hooks.ToolCallEvent{Phase: "pre_dispatch"}
		if err := d.Dispatch(context.Background(), hooks.PointToolCallPreDispatch, evt); err != nil {
			t.Fatalf("Dispatch concorrente: %v", err)
		}
	}
}

type reentrantHook struct {
	name       string
	dispatcher hooks.Dispatcher
	point      string
	nestedErr  *error
}

func (h *reentrantHook) Name() string { return h.name }
func (h *reentrantHook) Run(ctx context.Context, evt hooks.Event) error {
	*h.nestedErr = h.dispatcher.Dispatch(ctx, h.point, evt)
	return nil
}

func TestDispatcher_RecursionGuardBlocksReentrantDispatch(t *testing.T) {
	d := hooks.New()
	var nestedErr error

	h := &reentrantHook{name: "self-reentrant", dispatcher: d, point: hooks.PointToolCallPreDispatch, nestedErr: &nestedErr}
	d.Register(hooks.PointToolCallPreDispatch, h)

	evt := hooks.ToolCallEvent{Phase: "pre_dispatch"}
	if err := d.Dispatch(context.Background(), hooks.PointToolCallPreDispatch, evt); err != nil {
		t.Fatalf("outer Dispatch unexpected error: %v", err)
	}
	if nestedErr == nil {
		t.Fatal("nested Dispatch call must be blocked by the recursion guard, got nil error")
	}
	wantSubstring := "invocation depth"
	if !strings.Contains(nestedErr.Error(), wantSubstring) {
		t.Fatalf("nested Dispatch error = %q, want it to mention %q (recursion guard denial)", nestedErr.Error(), wantSubstring)
	}
}

type slowHook struct {
	name  string
	delay time.Duration
}

func (h *slowHook) Name() string { return h.name }
func (h *slowHook) Run(ctx context.Context, _ hooks.Event) error {
	select {
	case <-time.After(h.delay):
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func TestDispatcher_TimeoutBlocksSlowHook(t *testing.T) {
	d := hooks.New()
	if err := d.SetHookTimeout("slow-validator", 20*time.Millisecond); err != nil {
		t.Fatalf("SetHookTimeout unexpected error: %v", err)
	}
	d.Register(hooks.PointSessionPostEnd, &slowHook{name: "slow-validator", delay: 200 * time.Millisecond})

	err := d.Dispatch(context.Background(), hooks.PointSessionPostEnd, hooks.SessionPostEndEvent{})
	if err == nil {
		t.Fatal("expected timeout error, Dispatch returned nil")
	}
	if !errors.Is(err, hookcontract.ErrHookTimedOut) {
		t.Fatalf("error = %v, want ErrHookTimedOut — timeout must be treated as denial", err)
	}
}

func TestDispatcher_FastHookWithinTimeoutSucceeds(t *testing.T) {
	d := hooks.New()
	if err := d.SetHookTimeout("fast-validator", 500*time.Millisecond); err != nil {
		t.Fatalf("SetHookTimeout unexpected error: %v", err)
	}
	d.Register(hooks.PointSessionPostEnd, &slowHook{name: "fast-validator", delay: time.Millisecond})

	if err := d.Dispatch(context.Background(), hooks.PointSessionPostEnd, hooks.SessionPostEndEvent{}); err != nil {
		t.Fatalf("unexpected error for a hook within its declared timeout: %v", err)
	}
}

type alwaysFailingTelemetryWriter struct {
	calls int
}

func (w *alwaysFailingTelemetryWriter) WriteLine(_, _ string) error {
	w.calls++
	return errors.New("simulated telemetry backend failure")
}

func TestDispatcher_TelemetryFailureDoesNotBlockDispatch(t *testing.T) {
	t.Setenv("GOVERNANCE_TELEMETRY", "1")

	writer := &alwaysFailingTelemetryWriter{}
	tel := telemetry.NewHookTelemetryWithWriter(writer)
	d := hooks.NewWithTelemetry(t.TempDir(), tel)

	var called []string
	h := &recordHook{name: "telemetry-dependent-hook", called: &called}
	d.Register(hooks.PointSessionPostEnd, h)

	evt := hooks.SessionPostEndEvent{}
	dispatchErr := d.Dispatch(context.Background(), hooks.PointSessionPostEnd, evt)

	retryAttempts := 0
	if dispatchErr != nil {
		retryAttempts++
	}

	if dispatchErr != nil {
		t.Fatalf("Dispatch must succeed even when the telemetry writer always fails (RF-48): %v", dispatchErr)
	}
	if len(called) != 1 || called[0] != "telemetry-dependent-hook" {
		t.Fatalf("hook must have run to completion despite telemetry failure; called = %v", called)
	}
	if writer.calls == 0 {
		t.Fatal("the failing telemetry writer must still have been invoked at least once")
	}
	if retryAttempts != 0 {
		t.Fatalf("retryAttempts = %d, want 0 — telemetry failure must never trigger a retry cascade (RF-48)", retryAttempts)
	}
}

func TestPromptBuildEvent_KindReflectsPhase(t *testing.T) {
	pre := hooks.PromptBuildEvent{Phase: "pre_build"}
	if got := pre.Kind(); got != hooks.PointPromptPreBuild {
		t.Errorf("Kind() with Phase=pre_build = %q, want %q", got, hooks.PointPromptPreBuild)
	}

	post := hooks.PromptBuildEvent{Phase: "post_build"}
	if got := post.Kind(); got != hooks.PointPromptPostBuild {
		t.Errorf("Kind() with Phase=post_build = %q, want %q", got, hooks.PointPromptPostBuild)
	}
}

func TestToolCallEvent_KindReflectsPhase(t *testing.T) {
	pre := hooks.ToolCallEvent{Phase: "pre_dispatch"}
	if got := pre.Kind(); got != hooks.PointToolCallPreDispatch {
		t.Errorf("Kind() with Phase=pre_dispatch = %q, want %q", got, hooks.PointToolCallPreDispatch)
	}

	post := hooks.ToolCallEvent{Phase: "post_complete"}
	if got := post.Kind(); got != hooks.PointToolCallPostComplete {
		t.Errorf("Kind() with Phase=post_complete = %q, want %q", got, hooks.PointToolCallPostComplete)
	}
}
