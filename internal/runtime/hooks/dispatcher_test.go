package hooks_test

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/hooks"
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
