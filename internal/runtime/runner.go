package runtime

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/JailtonJunior94/ai-spec-harness/internal/approval"
	"github.com/JailtonJunior94/ai-spec-harness/internal/fs"
	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/client"
	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/events"
	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/handshake"
	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/hooks"
	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/memory"
	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/memory/durable"
	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/probe"
	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/render"
	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/specs"
	"github.com/JailtonJunior94/ai-spec-harness/internal/taskcriteria"
	"github.com/JailtonJunior94/ai-spec-harness/internal/telemetry"
)

// MCPServer define a interface do servidor MCP interno (F2-Claude).
// Implementado por internal/runtime/mcpserver.Server.
// Injetado via WithMCPServer; nil = MCP desabilitado (default, preserva F1-Claude).
type MCPServer interface {
	// Serve executa o loop stdio MCP até o context cancelar ou EOF no reader.
	Serve(ctx context.Context, in io.Reader, out io.Writer) error
}

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

var _ Renderer = (*render.HumanRenderer)(nil)

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

var _ Prober = (*defaultProber)(nil)

// NewDefaultProber cria um Prober de produção que delega para probe.EnsureAvailable.
func NewDefaultProber() Prober {
	return &defaultProber{look: probe.NewCatalog().OsLookPather()}
}

func (p *defaultProber) EnsureAvailable(ctx context.Context, spec specs.Spec) (specs.Launcher, error) {
	return probe.NewCatalog().EnsureAvailable(ctx, spec, p.look)
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
	// mcpServer é o servidor MCP interno (F2-Claude).
	// nil = MCP desabilitado (default, preserva comportamento F1-Claude).
	mcpServer MCPServer
	// reviewOutputFn é injetável para testes unitários (F5-Claude).
	// nil = usar spawnReviewSession (produção).
	reviewOutputFn autoReviewOutputFn
	// handshakeWaiterFactory constrói o waiter do handshake de governança do OpenCode.
	// nil = usar defaultHandshakeWaiterFactory (produção).
	handshakeWaiterFactory  HandshakeWaiterFactory
	promptPostBuildTestHook hooks.Hook
}

// NewACPRunner cria um ACPRunner com defaults de produção.
// Defaults: RealClock, DefaultProber, DefaultClientFactory, HumanRenderer(os.Stdout).
// A PersistenceFactory deve ser injetada via WithPersistenceFactory.
func NewACPRunner(spec specs.Spec, opts ...Option) *ACPRunner {
	r := &ACPRunner{
		spec:                   spec,
		prober:                 NewDefaultProber(),
		factory:                client.NewDefaultClientFactory(),
		renderer:               render.NewHumanRenderer(os.Stdout),
		clock:                  NewCatalog().RealClock(),
		handshakeWaiterFactory: NewDefaultHandshakeWaiterFactory(),
	}
	for _, o := range opts {
		o(r)
	}
	return r
}

// Run executa uma sessão ACP completa para o job fornecido.
// Orquestra: probe → memory read → hooks dispatch → open → fan-out → persistência → Summary.
// eventLoopResult agrega os contadores do loop de eventos.
const durableMemoryDeclaredSectionHeading = "## Memory Declared"

type eventLoopResult struct {
	eventsCount     int
	unknownCount    int
	unknownKinds    []string
	declaredSection string

	// Métricas unificadas por driver (ADR-021): substitui os 8 acumuladores paralelos.
	// MetricSet zero-value preserva comportamento F1 (nenhum campo emitido).
	metrics events.MetricSet

	// toolCallsNormalizedCount acumula tool-calls normalizadas para persistência no report.
	toolCallsNormalizedCount int
}

func (r *ACPRunner) Run(ctx context.Context, j Job) (Summary, error) {
	ctx, cancelCause := context.WithCancelCause(ctx)
	defer cancelCause(nil)

	// Fase 1: resolver launcher.
	launcher, err := r.prober.EnsureAvailable(ctx, r.spec)
	if err != nil {
		return Summary{}, fmt.Errorf("runner: resolver launcher: %w", err)
	}

	// Fase 2: criar persistence para o evidenceDir.
	persist, err := r.createPersistence(j.EvidenceDir)
	if err != nil {
		return Summary{}, err
	}

	// ★ ADR-023: propagar WindowClass da Spec para o Job (sem leitura de runtime/handshake).
	// Zero-value (WindowStandard) preserva comportamento F1.
	j.WindowClass = r.spec.ResolveWindow(j.Model).Class()

	sessionID := NewCatalog().newSessionID(r.clock)

	var memPort MemoryPort
	var memStore memory.Store
	var memRecorder *hooks.MemoryEvidenceRecorder
	var memReadContext durable.MemoryContext
	if j.DurableMemoryEnabled {
		log.Printf("runner: durable memory enabled (RF-28)")
		memPort = NewCatalog().prepareDurableMemoryFacade(j)
		j.Prompt, memReadContext = NewCatalog().prepareDurableMemoryPromptContext(ctx, j, memPort)
		memRecorder = hooks.NewMemoryEvidenceRecorder()
	} else {
		log.Printf("runner: durable memory disabled (RF-28); legacy path preserved")
		memStore = NewCatalog().prepareMemoryStore(j)
		j.Prompt = NewCatalog().prepareMemoryContext(ctx, j, memStore)
	}

	// ★ F3-Claude: instanciar hooks dispatcher e registrar hooks default.
	// j.DisableHooks=true → dispatcher vazio (debug; sem regressão F1/F2).
	// ★ ADR-023: WindowClass propagada da Spec para sensibilizar token_budget.
	disp := NewCatalog().prepareHooksDispatcher(j, r.spec.ID, memStore, r.spec.ResolveWindow(j.Model).Class(), r.promptPostBuildTestHook, memRecorder)

	// Fase 3: emitir runtime_init e persistir.
	launcherCmd, launcherArgs := launcher.Command()
	argv := r.buildArgv(j, launcherArgs)

	effectiveLauncher, stopMCP := NewCatalog().spawnMCPServer(ctx, r, j, launcherCmd, argv)
	r.emitRuntimeInit(ctx, launcher, launcherCmd, argv, persist)

	// ★ F3-Claude: hooks pre-open + prompt (governance + token_budget).
	if err := NewCatalog().dispatchPreOpenHooks(ctx, disp, j, r.spec.ID, launcherCmd); err != nil {
		return Summary{}, err
	}

	// Fase 4: abrir cliente ACP.
	c := r.factory.New(j.WorkDir)
	// AccessMode==full ⇒ auto-aprovar RequestPermission via ACP. Necessário para CLIs cujo
	// bypass não é negociado por flag de CLI (ex.: Copilot — ADR D-07). Codex/Claude já
	// recebem o bypass via BootstrapArgs (sandbox/--bypass-permissions); este wiring cobre a lacuna.
	if j.AccessMode == specs.AccessModeFull {
		if bp, ok := c.(interface{ SetBypassPermissions(bool) }); ok {
			bp.SetBypassPermissions(true)
		}
	}
	defer func() { _ = c.Close() }()
	defer stopMCP()

	releaseEnforcement, err := r.applyEnforcement(c, j)
	if err != nil {
		return Summary{}, err
	}
	defer releaseEnforcement()

	if err := c.Open(ctx, effectiveLauncher, j.Prompt); err != nil {
		if errors.Is(err, handshake.ErrSignalNotReceived) {
			_ = telemetry.NewCatalog().LogPreconditionRejection(j.WorkDir, r.spec.ID, "handshake_signal_not_received")
		}
		return Summary{}, fmt.Errorf("runner: abrir sessão ACP: %w", err)
	}

	// Fase 5: watchdog de inatividade.
	wd := NewActivityWatchdog(j.Timeout, cancelCause, r.clock)
	wd.Start(ctx)
	defer wd.Stop()

	// Fase 6: loop de eventos com fan-out.
	counters := events.NewToolCallCounters()
	loopResult := r.runEventLoop(ctx, c, wd, j, disp, counters, persist, r.spec.ID)

	// Fase 7: determinar razão de cancelamento.
	cause := context.Cause(ctx)
	clientErr := c.Err()
	cancelReason := NewCatalog().mapCancelReason(cause, clientErr)
	NewCatalog().emitUnknownWarnings(loopResult)
	if cancelReason == events.CancelReasonPermissionDenied {
		fmt.Fprintln(os.Stderr, "agente solicitou permissão e foi negado: reexecute com --access-mode full para auto-aprovar tool calls via ACP (ou rode em ambiente que pré-aprove). Veja ADR-009/ADR-012")
	}

	toolCallSummaries := counters.ToolCalls()
	summary := NewCatalog().buildSummary(launcher.Kind(), loopResult, cancelReason, toolCallSummaries, c)

	postEndErr := NewCatalog().dispatchSessionPostEnd(ctx, disp, j, loopResult, toolCallSummaries, cancelReason)
	if postEndErr != nil {
		log.Printf("runner: session.post_end hook dispatch failed (session continues): %v", postEndErr)
		summary.HookDispatchErrors = append(summary.HookDispatchErrors, postEndErr.Error())
	}

	if memPort != nil {
		NewCatalog().recordDurableMemorySession(ctx, memPort, j, loopResult, toolCallSummaries, cancelReason, &summary, sessionID, r.spec.ID, disp, memReadContext)
	}

	NewCatalog().persistSummary(persist, toolCallSummaries, summary)

	if j.AutoReview {
		reviewOutcome, reviewErr := r.performAutoReview(ctx, j)
		if reviewErr == nil {
			summary.ReviewStatus = reviewOutcome.status
			summary.ReviewPath = reviewOutcome.path
			summary.CycleRounds = reviewOutcome.cycleRounds
			summary.CycleStopReason = reviewOutcome.cycleStopReason
			if postReviewErr := disp.Dispatch(ctx, hooks.PointSessionPostReview, hooks.SessionPostReviewEvent{
				ReviewPath: reviewOutcome.path,
				Blocked:    reviewOutcome.status == "blocked",
			}); postReviewErr != nil {
				log.Printf("runner: session.post_review hook dispatch failed (session continues): %v", postReviewErr)
				summary.HookDispatchErrors = append(summary.HookDispatchErrors, postReviewErr.Error())
			}
		} else {
			fmt.Fprintf(os.Stderr, "runner: auto-review falhou (session continua): %v\n", reviewErr)
		}
	}

	return summary, NewCatalog().mapRunError(cause, clientErr, c)
}

// createPersistence cria uma instância de Persistence para o evidenceDir.
// Retorna nil, nil quando PersistenceFactory não está configurada.
func (r *ACPRunner) createPersistence(evidenceDir string) (Persistence, error) {
	if r.persistenceFactory == nil {
		return nil, nil
	}
	p, err := r.persistenceFactory.New(evidenceDir)
	if err != nil {
		return nil, fmt.Errorf("runner: criar persistence: %w", err)
	}
	return p, nil
}

// buildArgv compõe o argv final a partir dos launcherArgs e bootstrap args do Spec.
// Anti-padrão: nunca mutar launcherArgs diretamente — criar slice novo (ADR-013 D-02).
func (r *ACPRunner) buildArgv(j Job, launcherArgs []string) []string {
	bootstrap := r.spec.BootstrapArgs(j.Model, j.ReasoningEffort, j.AddDirs, j.AccessMode, j.WorkDir)
	argv := append([]string{}, launcherArgs...)
	return append(argv, bootstrap...)
}

// emitRuntimeInit constrói e persiste o evento runtime_init.
func (r *ACPRunner) emitRuntimeInit(_ context.Context, launcher specs.Launcher, launcherCmd string, argv []string, persist Persistence) {
	initRaw, initRawErr := NewCatalog().buildRuntimeInitRaw(launcher.Kind(), launcherCmd, r.spec.ID, argv, r.spec.SDKVersion(), r.spec.NPMVersion())
	initEvt, initErr := events.NewRuntimeInit(
		r.clock.Now(),
		launcher.Kind(),
		launcherCmd,
		argv,
		r.spec.SDKVersion(),
		r.spec.NPMVersion(),
		initRaw,
	)
	if initErr == nil && initRawErr == nil && persist != nil {
		_ = persist.AppendEvent(initEvt)
	}
}

// dispatchPreOpenHooks despacha os hooks antes de abrir a sessão ACP.
// Sequência: runtime.pre_open → prompt.pre_build → prompt.post_build.
// Retorna erro se qualquer hook falhar (abort-on-first-error).
func (c *Catalog) dispatchPreOpenHooks(ctx context.Context, disp hooks.Dispatcher, j Job, specID, launcherCmd string) error {
	if err := disp.Dispatch(ctx, hooks.PointRuntimePreOpen, hooks.RuntimePreOpenEvent{
		WorkDir:  j.WorkDir,
		SpecID:   specID,
		Launcher: launcherCmd,
		TasksDir: j.TasksDir,
	}); err != nil {
		return fmt.Errorf("runner: hook runtime.pre_open: %w", err)
	}
	promptRef := &j.Prompt
	if err := disp.Dispatch(ctx, hooks.PointPromptPreBuild, hooks.PromptBuildEvent{
		Prompt: promptRef,
		Spec:   specID,
	}); err != nil {
		return fmt.Errorf("runner: hook prompt.pre_build: %w", err)
	}
	if err := disp.Dispatch(ctx, hooks.PointPromptPostBuild, hooks.PromptBuildEvent{
		Prompt: promptRef,
		Spec:   specID,
	}); err != nil {
		return fmt.Errorf("runner: hook prompt.post_build: %w", err)
	}
	return nil
}

// runEventLoop executa o loop de fan-out de eventos até o canal c.Updates() fechar.
// driverID é o ID da spec ativa ("claude", "codex", "copilot", "opencode"); usado para
// selecionar o MetricsExtractor adequado (ADR-021).
// Retorna os contadores agregados do loop.
func (r *ACPRunner) runEventLoop(
	ctx context.Context,
	c client.Client,
	wd *ActivityWatchdog,
	j Job,
	disp hooks.Dispatcher,
	counters *events.ToolCallCounters,
	persist Persistence,
	driverID string,
) eventLoopResult {
	var (
		eventsCount            int
		unknownCount           int
		unknownKinds           []string
		unknownSet             = make(map[string]struct{})
		metrics                events.MetricSet
		toolCallsNormalizedCnt int
		declaredSection        string
	)

	// Selecionar extractor de métricas por driver (ADR-021, Strategy).
	// ParseDriverID: zero-value (driver vazio) → nullExtractor via ExtractorFor.
	drvID, _ := specs.NewCatalog().ParseDriverID(driverID)
	extractor := events.NewCatalog().ExtractorFor(drvID)

	for evt := range c.Updates() {
		evt = NewCatalog().normalizeEventInline(evt, r.spec.ID, j)

		// Resetar o watchdog apenas em progresso RECONHECIDO. Eventos keep-alive/desconhecidos
		// (usage_update, available_commands_update, chunks vazios, etc.) NÃO contam como atividade:
		// CLIs como o codex-acp emitem keep-alives sem encerrar o turn (sem end_turn) e, ao resetar
		// o watchdog a cada keep-alive, a sessão ficava viva para sempre (hang indefinido observado).
		// Com isto, o watchdog dispara após inatividade real e o teardown por ctx mata o subprocesso.
		if evt.Kind() != events.KindUnknown {
			wd.Touch()
		}

		if evt.Kind() == events.KindToolCallStart {
			_ = disp.Dispatch(ctx, hooks.PointToolCallPreDispatch, hooks.ToolCallEvent{Phase: "pre_dispatch"})
		}

		counters.Record(evt)

		if evt.Kind() == events.KindAgentMessage {
			if msg := evt.AgentMessage(); msg != nil {
				declaredSection = NewCatalog().extractDeclaredSection(declaredSection, msg.Text())
			}
		}

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

		// ★ ADR-021: acumular métricas via MetricSet único por driver (substitui os 8 acumuladores).
		// extractor.Extract retorna MetricSet{} para drivers sem métricas (codex, copilot).
		metrics = metrics.Merge(extractor.Extract(evt.Raw()))

		// Acumular tool-calls normalizadas para persistência no report.
		if evt.Kind() == events.KindToolCallStart && evt.NormalizedName() != "" {
			toolCallsNormalizedCnt++
		}

		if persist != nil {
			_ = persist.AppendEvent(evt)
		}

		if evt.Kind() == events.KindToolCallUpdate {
			_ = disp.Dispatch(ctx, hooks.PointToolCallPostComplete, hooks.ToolCallEvent{Phase: "post_complete"})
		}

		if !j.Quiet {
			r.renderer.Render(evt)
		}
	}

	return eventLoopResult{
		eventsCount:              eventsCount,
		unknownCount:             unknownCount,
		unknownKinds:             unknownKinds,
		metrics:                  metrics,
		toolCallsNormalizedCount: toolCallsNormalizedCnt,
		declaredSection:          declaredSection,
	}
}

func (c *Catalog) extractDeclaredSection(existing, agentMessageText string) string {
	if existing != "" {
		return existing
	}
	idx := strings.Index(agentMessageText, durableMemoryDeclaredSectionHeading)
	if idx < 0 {
		return ""
	}
	return strings.TrimSpace(agentMessageText[idx+len(durableMemoryDeclaredSectionHeading):])
}

// buildSummary constrói o Summary com os contadores e cancel reason.
// c é o client ACP; quando implementa as métricas de backpressure (SlowPublishes/DroppedUpdates),
// os valores são incorporados ao Summary (ADR-018, RF-03).
func (c *Catalog) buildSummary(launcher string, res eventLoopResult, cancelReason events.CancelReason, toolCalls []events.ToolCallSummary, cl client.Client) Summary {
	s := Summary{
		Launcher:                 launcher,
		EventsCount:              res.eventsCount,
		UnknownEventsCount:       res.unknownCount,
		CancelReason:             cancelReason,
		ToolCalls:                toolCalls,
		UnknownKinds:             res.unknownKinds,
		Metrics:                  res.metrics,
		ToolCallsNormalizedCount: res.toolCallsNormalizedCount,
	}
	// Propagar contadores de backpressure quando o client os expõe (ADR-018, RF-03).
	s.SlowPublishes = cl.SlowPublishes()
	s.DroppedUpdates = cl.DroppedUpdates()
	return s
}

// persistSummary persiste tool_calls e enriquece o report quando persist está disponível.
func (c *Catalog) persistSummary(persist Persistence, toolCalls []events.ToolCallSummary, summary Summary) {
	if persist == nil {
		return
	}
	_ = persist.WriteToolCalls(toolCalls)
	_ = persist.EnrichReport(summary)
}

// emitUnknownWarnings emite warning de unknowns no stderr.
func (c *Catalog) emitUnknownWarnings(res eventLoopResult) {
	if res.unknownCount > 0 {
		sort.Strings(res.unknownKinds)
		fmt.Fprintf(os.Stderr, "%d unknown ACP events skipped (kinds: %s)\n",
			res.unknownCount, strings.Join(res.unknownKinds, ", "))
	}
}

func (c *Catalog) dispatchSessionPostEnd(
	ctx context.Context,
	disp hooks.Dispatcher,
	j Job,
	res eventLoopResult,
	toolCalls []events.ToolCallSummary,
	cancelReason events.CancelReason,
) error {
	return disp.Dispatch(ctx, hooks.PointSessionPostEnd, hooks.SessionPostEndEvent{
		Summary: hooks.SessionSummary{
			TaskFileName: j.TaskFileName,
			ExitStatus:   string(cancelReason),
			EventsCount:  res.eventsCount,
			ToolCalls:    len(toolCalls),
		},
	})
}

// mapRunError mapeia cause + clientErr para o erro de retorno de Run().
func (c *Catalog) mapRunError(cause, clientErr error, cl client.Client) error {
	if cause != nil && !errors.Is(cause, context.Canceled) {
		return cause
	}
	if clientErr != nil {
		return clientErr
	}
	return cl.Err()
}

// prepareMemoryStore instancia o memory.Store quando j.TasksDir != "".
// Aplica WindowPolicy para ajustar limites por WindowClass (ADR-023):
//   - WindowStandard (zero-value) ⇒ defaults F1 (150/12KB · 200/16KB) — sem regressão.
//   - WindowLarge ⇒ limites ampliados para CLIs com janela ≥1M (ex: OpenCode com modelo Gemini).
//
// Retorna nil quando TasksDir está vazio (regressão F1/F2 preservada).
func (c *Catalog) prepareMemoryStore(j Job) memory.Store {
	if j.TasksDir == "" {
		return nil
	}
	// WindowPolicy resolve limites considerando WindowClass + base fornecido pelo Job.
	// Overrides explicitos de memoria prevalecem sobre defaults WindowLarge.
	limits := memory.DefaultWindowPolicy.LimitsForWithOverride(j.WindowClass, j.MemoryLimits, j.MemoryLimitsExplicit)
	return memory.New(j.TasksDir, limits)
}

// prepareMemoryContext lê workflow + task do memory store e injeta ## Memory Context no prompt.
// Quando NeedsCompaction=true, anexa diretiva de compactação ao final do prompt.
// Retorna prompt original quando store é nil (regressão F1/F2).
func (c *Catalog) prepareMemoryContext(ctx context.Context, j Job, store memory.Store) string {
	if store == nil {
		return j.Prompt
	}

	wf, wfErr := store.ReadWorkflow(ctx)
	tk, tkErr := store.ReadTask(ctx, j.TaskFileName)

	return NewCatalog().injectMemoryContext(j.Prompt, wf, tk, wfErr, tkErr)
}

// injectMemoryContext é a função pura que injeta ## Memory Context no prompt.
// Input: prompt base + documentos de workflow e task (podem ser zero-value).
// Testar isoladamente via T-MEM-INJECT-01.
func (c *Catalog) injectMemoryContext(prompt string, wf, tk memory.Document, wfErr, tkErr error) string {
	var sb strings.Builder

	hasWorkflow := wfErr == nil && wf.Exists && wf.Content != ""
	hasTask := tkErr == nil && tk.Exists && tk.Content != ""

	if !hasWorkflow && !hasTask {
		return prompt
	}

	sb.WriteString(prompt)
	sb.WriteString("\n\n## Memory Context\n")

	if hasWorkflow {
		sb.WriteString("\n### Workflow Memory\n\n")
		sb.WriteString(wf.Content)
	}

	if hasTask {
		sb.WriteString("\n### Task Memory\n\n")
		sb.WriteString(tk.Content)
	}

	// Diretiva de compactação: texto exato (paridade Compozy).
	// Anexar como bloco final quando qualquer tier necessita compactação.
	if (wfErr == nil && wf.NeedsCompaction) || (tkErr == nil && tk.NeedsCompaction) {
		sb.WriteString("\ncompact the flagged memory files before proceeding\n")
	}

	return sb.String()
}

// prepareHooksDispatcher instancia o Dispatcher e registra hooks default.
// Quando j.DisableHooks=true, retorna dispatcher vazio (debug; sem hooks ativos).
// Hook order: governance → token_budget em PointRuntimePreOpen/PointPromptPostBuild;
// memory_persist em PointSessionPostEnd (conforme task spec).
// windowClass é propagado da Spec para o TokenBudgetHook (ADR-023).
// Zero-value (WindowStandard) preserva comportamento F1.
func (c *Catalog) prepareHooksDispatcher(
	j Job,
	specID string,
	store memory.Store,
	windowClass specs.WindowClass,
	promptPostBuildTestHook hooks.Hook,
	memRecorder *hooks.MemoryEvidenceRecorder,
) hooks.Dispatcher {
	disp := hooks.New()

	if j.DisableHooks {
		return disp
	}

	// governance: valida AGENTS.md em runtime.pre_open.
	disp.Register(hooks.PointRuntimePreOpen, hooks.NewGovernanceHook())

	// spec_drift: valida spec-hash/PRD-first em runtime.pre_open (ADR-022, RG-01/RG-02).
	// No-op quando TasksDir=="" (propagado no evento); SkipDriftGuard desabilita só este hook.
	disp.Register(hooks.PointRuntimePreOpen, hooks.NewSpecDriftHook(j.SkipDriftGuard))

	// token_budget: valida tamanho do prompt em prompt.post_build, sensível à WindowClass (ADR-023).
	// WindowStandard ⇒ teto F1; WindowLarge ⇒ teto generoso para CLIs com janela ≥1M.
	disp.Register(hooks.PointPromptPostBuild, hooks.NewTokenBudgetHookWithClass(specID, windowClass))

	if promptPostBuildTestHook != nil {
		disp.Register(hooks.PointPromptPostBuild, promptPostBuildTestHook)
	}

	// memory_persist: escreve MEMORY.md em session.post_end (apenas quando store disponível).
	if store != nil {
		disp.Register(hooks.PointSessionPostEnd, hooks.NewMemoryPersistHook(store))
	}

	if memRecorder != nil {
		memoryEvidenceHook := hooks.NewMemoryEvidenceHook(memRecorder)
		disp.Register(hooks.PointMemoryFactRecorded, memoryEvidenceHook)
		disp.Register(hooks.PointMemoryFactArchived, memoryEvidenceHook)
		disp.Register(hooks.PointMemoryFactPromoted, memoryEvidenceHook)
		disp.Register(hooks.PointMemoryContradictionDetected, memoryEvidenceHook)
		disp.Register(hooks.PointMemorySecretRedacted, memoryEvidenceHook)
		disp.Register(hooks.PointMemoryCompactionExecuted, memoryEvidenceHook)
		disp.Register(hooks.PointMemoryBatonTransferred, memoryEvidenceHook)
	}

	return disp
}

// spawnMCPServer spawna o servidor MCP em goroutine quando j.MCPNested=true.
// Retorna o launcher efetivo (possivelmente com --mcp-server injetado) e uma função de parada.
// Quando MCPNested=false ou r.mcpServer==nil, retorna launcher original e func vazia.
// Extração de helper segue heurística OC (Run() cresceria >300 LoC sem esta separação).
func (c *Catalog) spawnMCPServer(ctx context.Context, r *ACPRunner, j Job, launcherCmd string, argv []string) (specs.Launcher, func()) {
	noop := func() {}

	if !j.MCPNested || r.mcpServer == nil {
		return specs.NewBinaryLauncher(launcherCmd, argv...), noop
	}

	// Criar par de pipes para comunicação stdio com o servidor MCP.
	// serverIn: leitura pelo servidor; serverOut: escrita pelo servidor.
	serverIn, clientOut, err := os.Pipe()
	if err != nil {
		// Falha de pipe: desabilitar MCP graciosamente (não abortar sessão).
		fmt.Fprintf(os.Stderr, "runner: criar pipe MCP server (in): %v — MCP desabilitado\n", err)
		return specs.NewBinaryLauncher(launcherCmd, argv...), noop
	}
	clientIn, serverOut, err := os.Pipe()
	if err != nil {
		_ = serverIn.Close()
		_ = clientOut.Close()
		fmt.Fprintf(os.Stderr, "runner: criar pipe MCP server (out): %v — MCP desabilitado\n", err)
		return specs.NewBinaryLauncher(launcherCmd, argv...), noop
	}

	mcpCtx, mcpCancel := context.WithCancel(ctx)

	// Spawnar servidor MCP em goroutine — lifecycle vinculado ao ctx da sessão.
	go func() {
		defer func() {
			_ = serverIn.Close()
			_ = serverOut.Close()
		}()
		if serveErr := r.mcpServer.Serve(mcpCtx, serverIn, serverOut); serveErr != nil && mcpCtx.Err() == nil {
			fmt.Fprintf(os.Stderr, "runner: mcp server encerrou com erro: %v\n", serveErr)
		}
	}()

	stopFn := func() {
		mcpCancel()
		_ = clientOut.Close()
		_ = clientIn.Close()
	}

	// Injetar --mcp-server no launcher como sinal para o Claude usar o servidor MCP.
	// Endereço stdio://mcp-server-ready sinaliza que pipes estão disponíveis.
	// Nota: integração real com claude-agent-acp requer suporte a --mcp-server no binário.
	// Para esta task (F2-Claude), o spawn + flag injection está implementado conforme spec;
	// integração wire end-to-end é validada nos smoke tests (subtarefa 3.9).
	mcpArgv := append([]string{}, argv...)
	mcpArgv = append(mcpArgv, "--mcp-server", "stdio://mcp-server-ready")
	effectiveLauncher := specs.NewBinaryLauncher(launcherCmd, mcpArgv...)

	return effectiveLauncher, stopFn
}

// normalizeEventInline aplica normalização de tool-calls ao evento quando aplicável.
// Retorna o evento enriquecido com WithNormalization quando:
//   - evt.Kind() == KindToolCallStart
//   - !j.NoNormalize
//
// Em qualquer outro caso ou erro de normalização, retorna o evento original sem modificação.
// Resolve DriverID na fronteira (ADR-020, Tarefa 3.0): specID inválido falha cedo e preserva
// o evento original (comportamento graceful — sem abortar a sessão, RF-02).
// Extração de helper segue heurística OC (mantém Run() legível).
func (c *Catalog) normalizeEventInline(evt events.Event, specID string, j Job) events.Event {
	if j.NoNormalize {
		return evt
	}
	if evt.Kind() != events.KindToolCallStart {
		return evt
	}
	tc := evt.ToolCallStart()
	if tc == nil {
		return evt
	}

	// Resolver DriverID na fronteira (fail-fast ADR-020): driver inválido → passthrough graceful.
	drvID, err := specs.NewCatalog().ParseDriverID(specID)
	if err != nil {
		// DriverID inválido (ErrUnknownDriver): preservar evento original sem abortar sessão.
		return evt
	}

	rawName := tc.Name()
	rawInput := json.RawMessage(tc.Input())

	norm, err := events.NewCatalog().BuildNormalizedToolCallByDriver(drvID, rawName, rawInput, j.WorkDir)
	if err != nil {
		// Erro de normalização: preservar evento original sem falhar a sessão (RF-02, graceful).
		return evt
	}

	return evt.WithNormalization(norm.RawName, norm.NormalizedName)
}

// mapCancelReason mapeia o cause do contexto para um CancelReason.
func (c *Catalog) mapCancelReason(cause error, clientErr error) events.CancelReason {
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

// InjectMemoryContextForTest expõe injectMemoryContext para testes (T-MEM-INJECT-01).
// Não usar em produção: helper puro sem efeitos colaterais.
func (c *Catalog) InjectMemoryContextForTest(prompt string, wf, tk memory.Document, wfErr, tkErr error) string {
	return NewCatalog().injectMemoryContext(prompt, wf, tk, wfErr, tkErr)
}

func (c *Catalog) prepareDurableMemoryFacade(j Job) MemoryPort {
	return durable.NewFacade(fs.NewOSFileSystem(), durable.FacadeConfig{
		ProjectDir: j.WorkDir,
		TasksDir:   j.TasksDir,
		LeaseTTL:   j.HandoffLeaseTTL,
	})
}

func (c *Catalog) newSessionID(clock Clock) string {
	return fmt.Sprintf("%s-%d", clock.Now().UTC().Format("20060102T150405.000000000"), os.Getpid())
}

func (c *Catalog) prepareDurableMemoryPromptContext(ctx context.Context, j Job, port MemoryPort) (string, durable.MemoryContext) {
	memCtx, err := port.BuildContext(ctx, durable.MemoryScope{
		TaskFileName: j.TaskFileName,
		WindowClass:  j.WindowClass,
	})
	if err != nil {
		log.Printf("runner: durable memory build context failed (session continues): %v", err)
		return j.Prompt, durable.MemoryContext{}
	}
	if memCtx.Block == "" {
		return j.Prompt, memCtx
	}
	return j.Prompt + "\n\n" + memCtx.Block, memCtx
}

func (c *Catalog) recordDurableMemorySession(
	ctx context.Context,
	port MemoryPort,
	j Job,
	res eventLoopResult,
	toolCalls []events.ToolCallSummary,
	cancelReason events.CancelReason,
	summary *Summary,
	sessionID string,
	cli string,
	disp hooks.Dispatcher,
	readContext durable.MemoryContext,
) {
	report, err := port.RecordSession(ctx, durable.SessionFacts{
		TaskFileName:    j.TaskFileName,
		ExitStatus:      string(cancelReason),
		EventsCount:     res.eventsCount,
		ToolCalls:       len(toolCalls),
		DeclaredSection: res.declaredSection,
		SessionID:       sessionID,
		CLI:             cli,
	})
	if err != nil {
		log.Printf("runner: durable memory record session failed (session continues): %v", err)
		summary.HookDispatchErrors = append(summary.HookDispatchErrors, fmt.Sprintf("durable memory: %v", err))
		return
	}
	log.Printf("runner: durable memory session recorded: writes=%d redactions=%d archived=%d baton_claimed=%v",
		report.Writes, report.Redactions, report.Archived, report.BatonClaimed)

	c.dispatchDurableMemoryEvents(ctx, disp, sessionID, cli, j.TaskFileName, report, summary)

	evidence := MemoryEvidence{
		SessionID:           sessionID,
		CLI:                 cli,
		TaskFileName:        j.TaskFileName,
		FactsByLayer:        readContext.FactsByLayer,
		FactsOmitted:        readContext.Omitted,
		FactsContradicted:   readContext.Contradicted,
		PagesUnreadable:     readContext.Unreadable,
		BudgetByLayer:       readContext.BudgetByLayer,
		WritesByLayer:       report.WritesByLayer,
		ArchivedByLayer:     report.ArchivedByLayer,
		Redactions:          report.Redactions,
		Compactions:         report.Compactions,
		Contradictions:      report.Contradictions,
		BatonClaimed:        report.BatonClaimed,
		ContextBuildLatency: readContext.BuildLatencyMs,
		RecordLatency:       report.RecordLatencyMs,
	}
	summary.MemoryEvidence = &evidence
	summary.Metrics = summary.Metrics.Merge(events.NewMetricSet(0, 0, 0, c.buildDurableMemoryMetrics(evidence)))

	if telemetryErr := telemetry.NewCatalog().LogDurableMemorySession(j.WorkDir, telemetry.DurableMemorySessionEvent{
		SessionID:      sessionID,
		CLI:            cli,
		FactsWritten:   report.Writes,
		FactsArchived:  report.Archived,
		Redactions:     report.Redactions,
		Compactions:    report.Compactions,
		Contradictions: report.Contradictions,
		BatonClaimed:   report.BatonClaimed,
	}); telemetryErr != nil {
		log.Printf("runner: durable memory telemetry logging failed (session continues): %v", telemetryErr)
	}
}

func (c *Catalog) dispatchDurableMemoryEvents(
	ctx context.Context,
	disp hooks.Dispatcher,
	sessionID, cli, taskFileName string,
	report durable.MemoryReport,
	summary *Summary,
) {
	if disp == nil {
		return
	}

	dispatchOne := func(point string, evt hooks.Event) {
		if err := disp.Dispatch(ctx, point, evt); err != nil {
			log.Printf("runner: %s hook dispatch failed (session continues): %v", point, err)
			summary.HookDispatchErrors = append(summary.HookDispatchErrors, fmt.Sprintf("%s: %v", point, err))
		}
	}

	if report.Writes > 0 {
		dispatchOne(hooks.PointMemoryFactRecorded, hooks.MemoryFactRecordedEvent{
			SessionID: sessionID, CLI: cli, TaskFileName: taskFileName, Count: report.Writes,
		})
	}
	if report.Archived > 0 {
		dispatchOne(hooks.PointMemoryFactArchived, hooks.MemoryFactArchivedEvent{
			SessionID: sessionID, CLI: cli, TaskFileName: taskFileName, Count: report.Archived,
		})
	}
	if report.Promoted > 0 {
		dispatchOne(hooks.PointMemoryFactPromoted, hooks.MemoryFactPromotedEvent{
			SessionID: sessionID, CLI: cli, TaskFileName: taskFileName, Count: report.Promoted,
		})
	}
	if report.Contradictions > 0 {
		dispatchOne(hooks.PointMemoryContradictionDetected, hooks.MemoryContradictionDetectedEvent{
			SessionID: sessionID, CLI: cli, TaskFileName: taskFileName, Count: report.Contradictions,
		})
	}
	if report.Redactions > 0 {
		dispatchOne(hooks.PointMemorySecretRedacted, hooks.MemorySecretRedactedEvent{
			SessionID: sessionID, CLI: cli, TaskFileName: taskFileName, Count: report.Redactions,
		})
	}
	if report.Compactions > 0 {
		dispatchOne(hooks.PointMemoryCompactionExecuted, hooks.MemoryCompactionExecutedEvent{
			SessionID: sessionID, CLI: cli, TaskFileName: taskFileName, Count: report.Compactions,
		})
	}
	if report.BatonClaimed {
		dispatchOne(hooks.PointMemoryBatonTransferred, hooks.MemoryBatonTransferredEvent{
			SessionID: sessionID, CLI: cli, TaskFileName: taskFileName, Claimed: report.BatonClaimed,
		})
	}
}

func (c *Catalog) buildRuntimeInitRaw(launcher, command, toolID string, args []string, sdkVersion, npmVersion string) ([]byte, error) {
	return json.Marshal(map[string]any{
		"launcher":    launcher,
		"command":     command,
		"args":        args,
		"sdk_version": sdkVersion,
		"npm_version": npmVersion,
		"tool":        toolID,
	})
}

type CycleRoundSummary struct {
	Number             int            `json:"number"`
	Verdict            string         `json:"verdict"`
	FindingsBySeverity map[string]int `json:"findings_by_severity,omitempty"`
}

type autoReviewOutcome struct {
	status          string
	path            string
	cycleRounds     []CycleRoundSummary
	cycleStopReason string
}

func (r *ACPRunner) performAutoReview(ctx context.Context, j Job) (autoReviewOutcome, error) {
	if strings.TrimSpace(j.TaskFileName) == "" {
		result, err := r.runAutoReview(ctx, j)
		if err != nil {
			return autoReviewOutcome{}, err
		}
		return autoReviewOutcome{status: result.Status, path: result.Path}, nil
	}
	return r.runApprovalCycle(ctx, j)
}

func (r *ACPRunner) runApprovalCycle(ctx context.Context, j Job) (autoReviewOutcome, error) {
	criteria, err := criteriaFromTaskFile(j.TasksDir, j.TaskFileName)
	if err != nil {
		return autoReviewOutcome{}, fmt.Errorf("runApprovalCycle: %w", err)
	}

	taskIdentity, err := approval.NewTaskIdentity(j.TaskFileName)
	if err != nil {
		return autoReviewOutcome{}, fmt.Errorf("runApprovalCycle: %w", err)
	}
	agentIdentity, err := approval.NewAgentIdentity(r.spec.ID)
	if err != nil {
		return autoReviewOutcome{}, fmt.Errorf("runApprovalCycle: %w", err)
	}
	var policyOpts []approval.PolicyOption
	if j.MaxBugfixIterations > 0 {
		policyOpts = append(policyOpts, approval.WithMaxRounds(j.MaxBugfixIterations))
	}
	policy, err := approval.NewApprovalPolicy(policyOpts...)
	if err != nil {
		return autoReviewOutcome{}, fmt.Errorf("runApprovalCycle: %w", err)
	}

	baseJob := j
	baseJob.AutoReview = false

	reviewer := NewReviewerAdapter(r, baseJob)
	fixer := NewFixerAdapter(r, baseJob)
	repository := NewRepositoryAdapter(j.WorkDir)

	cycle, err := approval.NewCycle(taskIdentity, agentIdentity, policy, criteria, reviewer, fixer, repository)
	if err != nil {
		return autoReviewOutcome{}, fmt.Errorf("runApprovalCycle: %w", err)
	}

	result, runErr := cycle.Run(ctx)
	if runErr != nil {
		return autoReviewOutcome{}, fmt.Errorf("runApprovalCycle: %w", runErr)
	}

	return buildCycleOutcome(j, result), nil
}

func buildCycleOutcome(j Job, result approval.CycleResult) autoReviewOutcome {
	rounds := make([]CycleRoundSummary, 0, result.RoundCount())
	for round := range result.Rounds() {
		rounds = append(rounds, CycleRoundSummary{
			Number:             round.Number(),
			Verdict:            round.Verdict().String(),
			FindingsBySeverity: severityCounts(round.CountBySeverity()),
		})
	}

	status := "blocked"
	if result.Approved() {
		status = "ok"
	}

	return autoReviewOutcome{
		status:          status,
		path:            filepath.Join(j.EvidenceDir, "review"),
		cycleRounds:     rounds,
		cycleStopReason: result.Reason().String(),
	}
}

func severityCounts(counts map[approval.Severity]int) map[string]int {
	if len(counts) == 0 {
		return nil
	}
	result := make(map[string]int, len(counts))
	for severity, n := range counts {
		result[severity.String()] = n
	}
	return result
}

func criteriaFromTaskFile(tasksDir, taskFileName string) ([]approval.AcceptanceCriterion, error) {
	content, err := os.ReadFile(filepath.Join(tasksDir, taskFileName))
	if err != nil {
		return nil, fmt.Errorf("read task file for cycle criteria: %w", err)
	}

	seen := make(map[string]bool)
	var criteria []approval.AcceptanceCriterion
	for _, description := range taskcriteria.Extract(content) {
		key := strings.TrimSpace(description)
		if key == "" || seen[key] {
			continue
		}
		seen[key] = true
		criterion, err := approval.NewAcceptanceCriterion(description)
		if err != nil {
			return nil, err
		}
		criteria = append(criteria, criterion)
	}
	return criteria, nil
}
