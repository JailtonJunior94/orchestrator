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
	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/hooks"
	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/memory"
	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/probe"
	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/render"
	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/specs"
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
	// mcpServer é o servidor MCP interno (F2-Claude).
	// nil = MCP desabilitado (default, preserva comportamento F1-Claude).
	mcpServer MCPServer
	// reviewOutputFn é injetável para testes unitários (F5-Claude).
	// nil = usar spawnReviewSession (produção).
	reviewOutputFn autoReviewOutputFn
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
// Orquestra: probe → memory read → hooks dispatch → open → fan-out → persistência → Summary.
// eventLoopResult agrega os contadores do loop de eventos.
type eventLoopResult struct {
	eventsCount  int
	unknownCount int
	unknownKinds []string

	// Contadores Claude-2026 (F4-Claude) — acumulados por extractClaudeMetricsFromEvent.
	// Default 0 quando o payload ACP não contém campo "usage" (opt-in, ADR-006).
	cacheReadTokens          int
	cacheCreationTokens      int
	thinkingTokens           int
	toolCallsNormalizedCount int

	// Contadores Gemini-2026 (F4-Gemini) — acumulados apenas quando driver == "gemini".
	// Default 0 para outros drivers (sem poluição em sessões Claude/Codex/Copilot).
	geminiCacheReadTokens        int
	geminiEffectiveContextTokens int
	geminiPromptTokensBilled     int
	geminiThoughtsTokens         int
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

	// ★ F3-Claude: instanciar memory store e injetar contexto no prompt.
	// j.TasksDir=="" → store nil → sem injeção (regressão F1/F2 preservada).
	memStore := prepareMemoryStore(j)
	j.Prompt = prepareMemoryContext(ctx, j, memStore)

	// ★ F3-Claude: instanciar hooks dispatcher e registrar hooks default.
	// j.DisableHooks=true → dispatcher vazio (debug; sem regressão F1/F2).
	disp := prepareHooksDispatcher(j, r.spec.ID, memStore)

	// Fase 3: emitir runtime_init e persistir.
	launcherCmd, launcherArgs := launcher.Command()
	argv := r.buildArgv(j, launcherArgs)

	effectiveLauncher, stopMCP := spawnMCPServer(ctx, r, j, launcherCmd, argv)
	r.emitRuntimeInit(ctx, launcher, launcherCmd, argv, persist)

	// ★ F3-Claude: hooks pre-open + prompt (governance + token_budget).
	if err := dispatchPreOpenHooks(ctx, disp, j, r.spec.ID, launcherCmd); err != nil {
		return Summary{}, err
	}

	// Fase 4: abrir cliente ACP.
	c := r.factory.New(j.WorkDir)
	defer func() { _ = c.Close() }()
	defer stopMCP()

	if err := c.Open(ctx, effectiveLauncher, j.Prompt); err != nil {
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
	cancelReason := mapCancelReason(cause, clientErr)

	// Fase 8: warning de unknowns (RF-05).
	emitUnknownWarnings(loopResult)
	if cancelReason == events.CancelReasonPermissionDenied {
		fmt.Fprintln(os.Stderr, "agent requested permission; configure accessMode=bypassPermissions no claude-agent-acp ou execute em ambiente que pré-aprove. Veja ADR-009")
	}

	// Fase 9: persistir tool_calls e enriquecer report.
	toolCallSummaries := counters.ToolCalls()
	summary := buildSummary(launcher.Kind(), loopResult, cancelReason, toolCallSummaries, c)
	persistSummary(persist, toolCallSummaries, summary)

	// ★ F3-Claude: hook session.post_end — memory_persist escreve MEMORY.md.
	dispatchSessionPostEnd(ctx, disp, j, loopResult, toolCallSummaries, cancelReason)

	// ★ F5-Claude: auto-review opt-in (default false; recursão hard-bloqueada no child Job).
	if j.AutoReview {
		reviewResult, reviewErr := r.runAutoReview(ctx, j)
		if reviewErr == nil {
			summary.ReviewStatus = reviewResult.Status
			summary.ReviewPath = reviewResult.Path
			_ = disp.Dispatch(ctx, hooks.PointSessionPostReview, hooks.SessionPostReviewEvent{
				ReviewPath: reviewResult.Path,
				Blocked:    reviewResult.Status == "blocked",
			})
		} else {
			fmt.Fprintf(os.Stderr, "runner: auto-review falhou (session continua): %v\n", reviewErr)
		}
	}

	return summary, mapRunError(cause, clientErr, c)
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
	bootstrap := r.spec.BootstrapArgs(j.Model, j.ReasoningEffort, j.AddDirs, j.AccessMode)
	argv := append([]string{}, launcherArgs...)
	return append(argv, bootstrap...)
}

// emitRuntimeInit constrói e persiste o evento runtime_init.
func (r *ACPRunner) emitRuntimeInit(_ context.Context, launcher specs.Launcher, launcherCmd string, argv []string, persist Persistence) {
	initRaw, initRawErr := buildRuntimeInitRaw(launcher.Kind(), launcherCmd, r.spec.ID, argv, r.spec.SDKVersion(), r.spec.NPMVersion())
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
func dispatchPreOpenHooks(ctx context.Context, disp hooks.Dispatcher, j Job, specID, launcherCmd string) error {
	if err := disp.Dispatch(ctx, hooks.PointRuntimePreOpen, hooks.RuntimePreOpenEvent{
		WorkDir:  j.WorkDir,
		SpecID:   specID,
		Launcher: launcherCmd,
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
// driverID é o ID da spec ativa ("claude", "gemini", "codex", "copilot") e é usado
// para extração condicional de métricas F4-Gemini (RF-20).
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
		eventsCount                  int
		unknownCount                 int
		unknownKinds                 []string
		unknownSet                   = make(map[string]struct{})
		cacheReadTokens              int
		cacheCreationTokens          int
		thinkingTokens               int
		toolCallsNormalizedCnt       int
		geminiCacheReadTokens        int
		geminiEffectiveContextTokens int
		geminiPromptTokensBilled     int
		geminiThoughtsTokens         int
	)

	for evt := range c.Updates() {
		wd.Touch()
		evt = normalizeEventInline(evt, r.spec.ID, j)

		if evt.Kind() == events.KindToolCallStart {
			_ = disp.Dispatch(ctx, hooks.PointToolCallPreDispatch, hooks.ToolCallEvent{Phase: "pre_dispatch"})
		}

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

		// ★ F4-Claude: extrair métricas Claude-2026 do payload bruto (opt-in, ADR-006).
		// Acumula cache_read, cache_creation e thinking tokens por update.
		m := events.ExtractClaudeMetrics(evt.Raw())
		cacheReadTokens += m.CacheReadTokens
		cacheCreationTokens += m.CacheCreationTokens
		thinkingTokens += m.ThinkingTokens

		// ★ F4-Gemini: extrair métricas Gemini-2026 do payload bruto (RF-20).
		// ExtractGeminiMetricsForDriver retorna zero-value para drivers não-Gemini (sem poluição).
		gm := events.ExtractGeminiMetricsForDriver(driverID, evt.Raw())
		geminiCacheReadTokens += gm.CacheReadTokens
		geminiEffectiveContextTokens += gm.EffectiveContextTokens
		geminiPromptTokensBilled += gm.PromptTokensBilled
		geminiThoughtsTokens += gm.ThoughtsTokens

		// Acumular tool-calls normalizadas (já incrementado em counters.Record via task 3.0;
		// aqui capturamos para persistência no report — sem dupla contagem de counters).
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
		eventsCount:                  eventsCount,
		unknownCount:                 unknownCount,
		unknownKinds:                 unknownKinds,
		cacheReadTokens:              cacheReadTokens,
		cacheCreationTokens:          cacheCreationTokens,
		thinkingTokens:               thinkingTokens,
		toolCallsNormalizedCount:     toolCallsNormalizedCnt,
		geminiCacheReadTokens:        geminiCacheReadTokens,
		geminiEffectiveContextTokens: geminiEffectiveContextTokens,
		geminiPromptTokensBilled:     geminiPromptTokensBilled,
		geminiThoughtsTokens:         geminiThoughtsTokens,
	}
}

// buildSummary constrói o Summary com os contadores e cancel reason.
// c é o client ACP; quando implementa as métricas de backpressure (SlowPublishes/DroppedUpdates),
// os valores são incorporados ao Summary (ADR-018, RF-03).
func buildSummary(launcher string, res eventLoopResult, cancelReason events.CancelReason, toolCalls []events.ToolCallSummary, c client.Client) Summary {
	s := Summary{
		Launcher:                     launcher,
		EventsCount:                  res.eventsCount,
		UnknownEventsCount:           res.unknownCount,
		CancelReason:                 cancelReason,
		ToolCalls:                    toolCalls,
		UnknownKinds:                 res.unknownKinds,
		CacheReadTokens:              res.cacheReadTokens,
		CacheCreationTokens:          res.cacheCreationTokens,
		ThinkingTokens:               res.thinkingTokens,
		ToolCallsNormalizedCount:     res.toolCallsNormalizedCount,
		GeminiCacheReadTokens:        res.geminiCacheReadTokens,
		GeminiEffectiveContextTokens: res.geminiEffectiveContextTokens,
		GeminiPromptTokensBilled:     res.geminiPromptTokensBilled,
		GeminiThoughtsTokens:         res.geminiThoughtsTokens,
	}
	// Propagar contadores de backpressure quando o client os expõe (ADR-018, RF-03).
	s.SlowPublishes = c.SlowPublishes()
	s.DroppedUpdates = c.DroppedUpdates()
	return s
}

// persistSummary persiste tool_calls e enriquece o report quando persist está disponível.
func persistSummary(persist Persistence, toolCalls []events.ToolCallSummary, summary Summary) {
	if persist == nil {
		return
	}
	_ = persist.WriteToolCalls(toolCalls)
	_ = persist.EnrichReport(summary)
}

// emitUnknownWarnings emite warning de unknowns no stderr.
func emitUnknownWarnings(res eventLoopResult) {
	if res.unknownCount > 0 {
		sort.Strings(res.unknownKinds)
		fmt.Fprintf(os.Stderr, "%d unknown ACP events skipped (kinds: %s)\n",
			res.unknownCount, strings.Join(res.unknownKinds, ", "))
	}
}

// dispatchSessionPostEnd despacha o hook session.post_end com o summary da sessão.
func dispatchSessionPostEnd(
	ctx context.Context,
	disp hooks.Dispatcher,
	j Job,
	res eventLoopResult,
	toolCalls []events.ToolCallSummary,
	cancelReason events.CancelReason,
) {
	_ = disp.Dispatch(ctx, hooks.PointSessionPostEnd, hooks.SessionPostEndEvent{
		Summary: hooks.SessionSummary{
			TaskFileName: j.TaskFileName,
			ExitStatus:   string(cancelReason),
			EventsCount:  res.eventsCount,
			ToolCalls:    len(toolCalls),
		},
	})
}

// mapRunError mapeia cause + clientErr para o erro de retorno de Run().
func mapRunError(cause, clientErr error, c client.Client) error {
	if cause != nil && !errors.Is(cause, context.Canceled) {
		return cause
	}
	if clientErr != nil {
		return clientErr
	}
	return c.Err()
}

// prepareMemoryStore instancia o memory.Store quando j.TasksDir != "".
// Aplica fallback de defaults quando os limites são zero-value (F3-Claude).
// Retorna nil quando TasksDir está vazio (regressão F1/F2 preservada).
func prepareMemoryStore(j Job) memory.Store {
	if j.TasksDir == "" {
		return nil
	}
	limits := j.MemoryLimits
	if limits.WorkflowLines == 0 {
		limits.WorkflowLines = memory.DefaultWorkflowLineLimit
	}
	if limits.WorkflowBytes == 0 {
		limits.WorkflowBytes = memory.DefaultWorkflowByteLimit
	}
	if limits.TaskLines == 0 {
		limits.TaskLines = memory.DefaultTaskLineLimit
	}
	if limits.TaskBytes == 0 {
		limits.TaskBytes = memory.DefaultTaskByteLimit
	}
	return memory.New(j.TasksDir, limits)
}

// prepareMemoryContext lê workflow + task do memory store e injeta ## Memory Context no prompt.
// Quando NeedsCompaction=true, anexa diretiva de compactação ao final do prompt.
// Retorna prompt original quando store é nil (regressão F1/F2).
func prepareMemoryContext(ctx context.Context, j Job, store memory.Store) string {
	if store == nil {
		return j.Prompt
	}

	wf, wfErr := store.ReadWorkflow(ctx)
	tk, tkErr := store.ReadTask(ctx, j.TaskFileName)

	return injectMemoryContext(j.Prompt, wf, tk, wfErr, tkErr)
}

// injectMemoryContext é a função pura que injeta ## Memory Context no prompt.
// Input: prompt base + documentos de workflow e task (podem ser zero-value).
// Testar isoladamente via T-MEM-INJECT-01.
func injectMemoryContext(prompt string, wf, tk memory.Document, wfErr, tkErr error) string {
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
func prepareHooksDispatcher(j Job, specID string, store memory.Store) hooks.Dispatcher {
	disp := hooks.New()

	if j.DisableHooks {
		return disp
	}

	// governance: valida AGENTS.md em runtime.pre_open.
	disp.Register(hooks.PointRuntimePreOpen, hooks.NewGovernanceHook())

	// token_budget: valida tamanho do prompt em prompt.post_build.
	disp.Register(hooks.PointPromptPostBuild, hooks.NewTokenBudgetHook(specID))

	// memory_persist: escreve MEMORY.md em session.post_end (apenas quando store disponível).
	if store != nil {
		disp.Register(hooks.PointSessionPostEnd, hooks.NewMemoryPersistHook(store))
	}

	return disp
}

// spawnMCPServer spawna o servidor MCP em goroutine quando j.MCPNested=true.
// Retorna o launcher efetivo (possivelmente com --mcp-server injetado) e uma função de parada.
// Quando MCPNested=false ou r.mcpServer==nil, retorna launcher original e func vazia.
// Extração de helper segue heurística OC (Run() cresceria >300 LoC sem esta separação).
func spawnMCPServer(ctx context.Context, r *ACPRunner, j Job, launcherCmd string, argv []string) (specs.Launcher, func()) {
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
// Extração de helper segue heurística OC (mantém Run() legível).
func normalizeEventInline(evt events.Event, specID string, j Job) events.Event {
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
	rawName := tc.Name()
	rawInput := json.RawMessage(tc.Input())

	norm, err := events.BuildNormalizedToolCall(specID, rawName, rawInput, j.WorkDir)
	if err != nil {
		// Erro de normalização: preservar evento original sem falhar a sessão (RF-02, graceful).
		return evt
	}

	return evt.WithNormalization(norm.RawName, norm.NormalizedName)
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

// InjectMemoryContextForTest expõe injectMemoryContext para testes (T-MEM-INJECT-01).
// Não usar em produção: helper puro sem efeitos colaterais.
func InjectMemoryContextForTest(prompt string, wf, tk memory.Document, wfErr, tkErr error) string {
	return injectMemoryContext(prompt, wf, tk, wfErr, tkErr)
}

func buildRuntimeInitRaw(launcher, command, toolID string, args []string, sdkVersion, npmVersion string) ([]byte, error) {
	return json.Marshal(map[string]any{
		"launcher":    launcher,
		"command":     command,
		"args":        args,
		"sdk_version": sdkVersion,
		"npm_version": npmVersion,
		"tool":        toolID,
	})
}
