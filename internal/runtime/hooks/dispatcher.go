// Package hooks implementa o dispatcher de hooks in-process para F3-Claude.
// Replica o padrão de compozy/internal/core/hooks para o harness.
//
// Pontos canônicos: runtime.pre_open, prompt.pre_build, prompt.post_build,
// tool_call.pre_dispatch, tool_call.post_complete, session.post_end.
// Ponto futuro: session.post_review (F5-Claude, task 8.0).
package hooks

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/JailtonJunior94/ai-spec-harness/internal/hookcontract"
	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/events"
	"github.com/JailtonJunior94/ai-spec-harness/internal/telemetry"
)

const (
	PointRuntimePreOpen       = "runtime.pre_open"
	PointPromptPreBuild       = "prompt.pre_build"
	PointPromptPostBuild      = "prompt.post_build"
	PointToolCallPreDispatch  = "tool_call.pre_dispatch"
	PointToolCallPostComplete = "tool_call.post_complete"
	PointSessionPostEnd       = "session.post_end"
	PointSessionPostReview    = "session.post_review" // F5-Claude (task 8.0)
)

// Event é a interface base de todos os eventos de hook.
// Cada ponto canônico recebe um tipo concreto específico.
type Event interface {
	Kind() string
}

// Hook é a interface que todo hook Go deve implementar.
type Hook interface {
	// Name retorna o identificador único do hook (para logs e rastreamento de erro).
	Name() string
	// Run executa o hook. Retornar erro interrompe o fan-out sequencial.
	Run(ctx context.Context, evt Event) error
}

// Dispatcher registra hooks por ponto e os executa em fan-out sequencial.
type Dispatcher interface {
	// Register associa um hook a um ponto canônico.
	// Thread-safe: pode ser chamado de múltiplas goroutines.
	Register(point string, hook Hook)
	// Dispatch executa todos os hooks registrados no ponto dado.
	// Fan-out sequencial, abort-on-first-error.
	// Ponto sem hooks registrados retorna nil sem panic.
	// Erros são envelopados: "hook <name> in <point>: <err>".
	Dispatch(ctx context.Context, point string, evt Event) error
	SetHookTimeout(hook string, timeout time.Duration) error
}

// RuntimePreOpenEvent é o evento emitido antes de abrir a sessão ACP.
// Hook governance valida AGENTS.md neste ponto.
// TasksDir é o caminho do diretório de tasks do PRD ativo (ex: ".specs/prd-foo").
// Quando vazio, SpecDriftHook opera em modo no-op (uso ad-hoc/F1 preservado).
type RuntimePreOpenEvent struct {
	WorkDir  string
	SpecID   string
	Launcher string
	// TasksDir é o diretório do PRD ativo; propagado de Job.TasksDir.
	// Vazio = sem guard de drift (comportamento F1 preservado).
	TasksDir string
}

func (e RuntimePreOpenEvent) Kind() string { return PointRuntimePreOpen }

type PromptBuildEvent struct {
	Prompt *string
	Spec   string
	Phase  string
}

func (e PromptBuildEvent) Kind() string {
	if e.Phase == "post_build" {
		return PointPromptPostBuild
	}
	return PointPromptPreBuild
}

type ToolCallEvent struct {
	Call  events.NormalizedToolCall
	Phase string
}

func (e ToolCallEvent) Kind() string {
	if e.Phase == "post_complete" {
		return PointToolCallPostComplete
	}
	return PointToolCallPreDispatch
}

// SessionPostEndEvent é o evento emitido ao final da sessão ACP.
// Hooks de memória persistem MEMORY.md neste ponto.
type SessionPostEndEvent struct {
	Summary any // *runtime.Summary (mutable; typed após task 6.0 wiring)
}

func (e SessionPostEndEvent) Kind() string { return PointSessionPostEnd }

// SessionPostReviewEvent é o evento emitido após o auto-review (F5-Claude).
// Disparado em session.post_review quando AutoReview=true e review completou.
type SessionPostReviewEvent struct {
	ReviewPath string
	Blocked    bool
}

func (e SessionPostReviewEvent) Kind() string { return PointSessionPostReview }

type invocationDepthKey struct{}

func depthFromContext(ctx context.Context) int {
	if v, ok := ctx.Value(invocationDepthKey{}).(int); ok {
		return v
	}
	return 0
}

func withInvocationDepth(ctx context.Context, depth int) context.Context {
	return context.WithValue(ctx, invocationDepthKey{}, depth)
}

// dispatcher é a implementação concreta de Dispatcher.
type dispatcher struct {
	mu        sync.RWMutex
	hooks     map[string][]Hook
	timeouts  *hookcontract.HookTimeoutRegistry
	telemetry telemetry.HookTelemetry
	rootDir   string
}

var _ Dispatcher = (*dispatcher)(nil)

// New retorna um Dispatcher vazio pronto para uso.
func New(rootDir ...string) Dispatcher {
	dir := "."
	if len(rootDir) > 0 && rootDir[0] != "" {
		dir = rootDir[0]
	}
	return &dispatcher{
		hooks:     make(map[string][]Hook),
		timeouts:  hookcontract.NewHookTimeoutRegistry(),
		telemetry: telemetry.NewHookTelemetry(),
		rootDir:   dir,
	}
}

func (d *dispatcher) SetHookTimeout(hook string, timeout time.Duration) error {
	return d.timeouts.Declare(hook, timeout)
}

// Register associa um hook a um ponto. Thread-safe.
func (d *dispatcher) Register(point string, hook Hook) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.hooks[point] = append(d.hooks[point], hook)
}

// Dispatch executa todos os hooks do ponto em ordem de registro.
// Abort-on-first-error: primeiro erro interrompe; hooks subsequentes não rodam.
// Ponto sem registro retorna nil sem panic.
func (d *dispatcher) Dispatch(ctx context.Context, point string, evt Event) error {
	depth := depthFromContext(ctx)
	guard, err := hookcontract.CheckRecursionGuard(depth)
	if err != nil {
		return fmt.Errorf("hook dispatch at %s: %w", point, err)
	}
	if guard.Decision() == hookcontract.DecisionBlock {
		return fmt.Errorf("hook dispatch at %s: %s", point, guard.Reason())
	}
	hookCtx := withInvocationDepth(ctx, depth+1)

	d.mu.RLock()
	hs := d.hooks[point]
	// Copiar slice sob read lock para liberar o lock antes de executar os hooks.
	hsCopy := make([]Hook, len(hs))
	copy(hsCopy, hs)
	d.mu.RUnlock()

	for _, h := range hsCopy {
		if err := d.runWithTimeout(hookCtx, point, h, evt); err != nil {
			return err
		}
	}
	return nil
}

func (d *dispatcher) runWithTimeout(ctx context.Context, point string, h Hook, evt Event) error {
	timeout := d.timeouts.TimeoutFor(h.Name())
	runCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	start := time.Now()
	runErr := h.Run(runCtx, evt)
	elapsed := time.Since(start)

	d.telemetry.RecordDuration(d.rootDir, h.Name(), point, "", elapsed.Milliseconds())

	if errors.Is(runCtx.Err(), context.DeadlineExceeded) {
		d.telemetry.RecordTimeout(d.rootDir, h.Name(), point, "")
		return fmt.Errorf("hook %s in %s after %s: %w", h.Name(), point, timeout, hookcontract.ErrHookTimedOut)
	}
	if runErr != nil {
		d.telemetry.RecordDecision(d.rootDir, h.Name(), point, "", hookcontract.DecisionError.String())
		return fmt.Errorf("hook %s in %s: %w", h.Name(), point, runErr)
	}
	d.telemetry.RecordDecision(d.rootDir, h.Name(), point, "", hookcontract.DecisionAllow.String())
	return nil
}
