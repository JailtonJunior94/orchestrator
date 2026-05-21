package runtime

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"

	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/client"
	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/events"
	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/probe"
	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/render"
	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/specs"
)

// Persistence define os contratos de persistência de evidências da sessão ACP.
// Implementado por internal/runtime/persistence; injetado no ACPRunner.
type Persistence interface {
	AppendEvent(evt events.Event) error
	WriteToolCalls(summary []events.ToolCallSummary) error
	EnrichReport(summary Summary) error
}

// PersistenceFactory cria uma instância de Persistence para um EvidenceDir.
type PersistenceFactory interface {
	New(evidenceDir string) (Persistence, error)
}

// Renderer define o contrato de renderização de eventos para o usuário.
type Renderer interface {
	Render(evt events.Event)
}

// Prober resolve o launcher para um Spec.
// Em produção, wraps probe.EnsureAvailable (com cache por spec ID).
// Em testes, pode ser uma implementação sem cache que retorna um Launcher fixo.
type Prober interface {
	EnsureAvailable(ctx context.Context, spec specs.Spec) (specs.Launcher, error)
}

// defaultProber é a implementação de produção de Prober: usa probe.EnsureAvailable com cache.
type defaultProber struct {
	look probe.LookPather
}

// NewDefaultProber cria um Prober de produção que delega para probe.EnsureAvailable.
func NewDefaultProber() Prober {
	return &defaultProber{look: probe.OsLookPather()}
}

func (p *defaultProber) EnsureAvailable(ctx context.Context, spec specs.Spec) (specs.Launcher, error) {
	return probe.EnsureAvailable(ctx, spec, p.look)
}

// ACPRunner é o application service que orquestra uma sessão ACP completa.
// Sequência: probe → client.Open → fan-out de eventos → persistência → Summary.
// Usa Functional Options para receber colaboradores opcionais; defaults de produção
// são injetados em NewACPRunner.
type ACPRunner struct {
	spec               specs.Spec
	prober             Prober
	factory            client.ClientFactory
	renderer           Renderer
	clock              Clock
	persistenceFactory PersistenceFactory
}

// NewACPRunner cria um ACPRunner com defaults de produção.
// Defaults: RealClock, DefaultProber, DefaultClientFactory, HumanRenderer(os.Stdout).
// A PersistenceFactory deve ser injetada via WithPersistenceFactory.
func NewACPRunner(spec specs.Spec, opts ...Option) *ACPRunner {
	r := &ACPRunner{
		spec:     spec,
		prober:   NewDefaultProber(),
		factory:  client.NewDefaultClientFactory(),
		renderer: render.NewHumanRenderer(os.Stdout),
		clock:    RealClock(),
	}
	for _, o := range opts {
		o(r)
	}
	return r
}

// Run executa uma sessão ACP completa para o job fornecido.
// Orquestra: probe → open → fan-out de eventos → persistência → Summary.
func (r *ACPRunner) Run(ctx context.Context, j Job) (Summary, error) {
	ctx, cancelCause := context.WithCancelCause(ctx)
	defer cancelCause(nil)

	// Fase 1: resolver launcher.
	launcher, err := r.prober.EnsureAvailable(ctx, r.spec)
	if err != nil {
		return Summary{}, fmt.Errorf("runner: resolver launcher: %w", err)
	}

	// Fase 2: criar persistence para o evidenceDir.
	var persist Persistence
	if r.persistenceFactory != nil {
		p, err := r.persistenceFactory.New(j.EvidenceDir)
		if err != nil {
			return Summary{}, fmt.Errorf("runner: criar persistence: %w", err)
		}
		persist = p
	}

	// Fase 3: emitir runtime_init e persistir.
	launcherCmd, launcherArgs := launcher.Command()
	initRaw, initRawErr := buildRuntimeInitRaw(launcher.Kind(), launcherCmd, launcherArgs, r.spec.SDKVersion(), r.spec.NPMVersion())
	initEvt, initErr := events.NewRuntimeInit(
		r.clock.Now(),
		launcher.Kind(),
		launcherCmd,
		launcherArgs,
		r.spec.SDKVersion(),
		r.spec.NPMVersion(),
		initRaw,
	)
	if initErr == nil && initRawErr == nil && persist != nil {
		_ = persist.AppendEvent(initEvt)
	}

	// Fase 4: abrir cliente ACP.
	c := r.factory.New(j.WorkDir)
	defer func() { _ = c.Close() }()

	if err := c.Open(ctx, launcher, j.Prompt); err != nil {
		return Summary{}, fmt.Errorf("runner: abrir sessão ACP: %w", err)
	}

	// Fase 5: watchdog de inatividade.
	wd := NewActivityWatchdog(j.ActivityTimeout, cancelCause, r.clock)
	wd.Start(ctx)
	defer wd.Stop()

	// Fase 6: loop de eventos com fan-out.
	counters := events.NewToolCallCounters()
	var (
		eventsCount  int
		unknownCount int
		unknownKinds []string
		unknownSet   = make(map[string]struct{})
	)

	for evt := range c.Updates() {
		wd.Touch()
		counters.Record(evt)

		if evt.Kind() == events.KindUnknown {
			unknownCount++
			if u := evt.Unknown(); u != nil {
				rk := u.RawKind()
				if _, seen := unknownSet[rk]; !seen {
					unknownSet[rk] = struct{}{}
					unknownKinds = append(unknownKinds, rk)
				}
			}
		} else {
			eventsCount++
		}

		if persist != nil {
			_ = persist.AppendEvent(evt)
		}

		if !j.Quiet {
			r.renderer.Render(evt)
		}
	}

	// Fase 7: determinar razão de cancelamento.
	cause := context.Cause(ctx)
	clientErr := c.Err()
	cancelReason := mapCancelReason(cause, clientErr)

	// Fase 8: warning de unknowns (RF-05).
	if unknownCount > 0 {
		sort.Strings(unknownKinds)
		fmt.Fprintf(os.Stderr, "%d unknown ACP events skipped (kinds: %s)\n",
			unknownCount, strings.Join(unknownKinds, ", "))
	}

	if cancelReason == events.CancelReasonPermissionDenied {
		fmt.Fprintln(os.Stderr, "agent requested permission; configure accessMode=bypassPermissions no claude-agent-acp ou execute em ambiente que pré-aprove. Veja ADR-009")
	}

	// Fase 9: persistir tool_calls e enriquecer report.
	toolCallSummaries := counters.ToolCalls()
	summary := Summary{
		Launcher:           launcher.Kind(),
		EventsCount:        eventsCount,
		UnknownEventsCount: unknownCount,
		CancelReason:       cancelReason,
		ToolCalls:          toolCallSummaries,
		UnknownKinds:       unknownKinds,
	}

	if persist != nil {
		_ = persist.WriteToolCalls(toolCallSummaries)
		_ = persist.EnrichReport(summary)
	}

	// Mapear erro de retorno.
	var runErr error
	if cause != nil && !errors.Is(cause, context.Canceled) {
		runErr = cause
	} else if clientErr != nil {
		runErr = clientErr
	} else if err := c.Err(); err != nil {
		runErr = err
	}

	return summary, runErr
}

// mapCancelReason mapeia o cause do contexto para um CancelReason.
func mapCancelReason(cause error, clientErr error) events.CancelReason {
	if cause == nil {
		if errors.Is(clientErr, client.ErrPermissionDenied) {
			return events.CancelReasonPermissionDenied
		}
		return events.CancelReasonNone
	}
	switch {
	case errors.Is(cause, ErrActivityTimeout):
		return events.CancelReasonActivityTimeout
	case errors.Is(cause, ErrPermissionDenied):
		return events.CancelReasonPermissionDenied
	case errors.Is(clientErr, client.ErrPermissionDenied):
		return events.CancelReasonPermissionDenied
	default:
		return events.CancelReasonContextCanceled
	}
}

// SetRenderer substitui o renderer. Usado em testes para capturar output.
func (r *ACPRunner) SetRenderer(w io.Writer) {
	r.renderer = render.NewHumanRenderer(w)
}

func buildRuntimeInitRaw(launcher, command string, args []string, sdkVersion, npmVersion string) ([]byte, error) {
	return json.Marshal(map[string]any{
		"launcher":    launcher,
		"command":     command,
		"args":        args,
		"sdk_version": sdkVersion,
		"npm_version": npmVersion,
	})
}
