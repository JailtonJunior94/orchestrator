<!-- spec-hash-prd: b5b59cad50a700d64f42aadb262086e4fee7011730a2164db9279afefa59e582 -->
<!-- MANDATÓRIO: preenchido por `create-technical-specification` Etapa 7.1 com sha256 do PRD consumido.
     Rastreabilidade: `create-tasks` e `execute-task` comparam este hash com o atual do prd.md
     para detectar drift entre techspec e PRD. NÃO remover este comentário ao editar a techspec. -->

# Especificação Técnica — ACP Runtime para Claude

## Resumo Executivo

Introduzir um novo módulo `internal/runtime/` que encapsula a comunicação com agentes de IA via **Agent Client Protocol (ACP)**, começando por **Claude**. O módulo expõe um *application service* (`ACPRunner`) consumido por um `acpInvoker` que implementa a interface `AgentInvoker` já existente em `internal/taskloop/agent.go`, preservando 100% do caminho legacy (default `--runtime=legacy`).

A comunicação com o subprocesso `claude-agent-acp` ocorre via `github.com/coder/acp-go-sdk` (stdio JSON-RPC). Os `acp.SessionUpdate` são traduzidos em um *value object* `runtime.Event` (tagged union) emitido por um canal único; consumidores (renderer humano, persistência JSONL, watchdog, contadores de telemetria) trabalham sobre o mesmo stream desacoplado por *fan-out*. Cancelamentos carregam a razão via `context.CancelCause` para que o `execution_report.md` possa registrar `cancel_reason` corretamente.

A arquitetura espelha o desenho do `compozy/compozy` (`internal/core/agent/acp_convert.go`, `internal/core/run/internal/acpshared/session_exec.go`, `registry_specs.go`), adaptada às convenções do `ai-spec-harness`: filesystem via `internal/fs.FileSystem`, evidência via `internal/evidence`, governança PRD-First com spec-hash.

## Arquitetura do Sistema

### Fronteiras (DDD)

```
┌─ Apresentação (CLI / Cobra) ────────────────────────────────────┐
│  cmd/ai_spec_harness/task_loop.go                               │
│   - parse de --runtime / --tool / --activity-timeout / --quiet  │
│   - validação RF-01, RF-02, RF-07                               │
└────────────────────────┬────────────────────────────────────────┘
                         │ chama
┌────────────────────────▼─────── Aplicação ──────────────────────┐
│  internal/taskloop/                                             │
│   - runloop.go (sequencial)                                     │
│   - agent.go: AgentInvoker (interface existente)                │
│   - acpinvoker.go (NOVO): adapta runtime.ACPRunner              │
│                                                                 │
│  internal/runtime/                                              │
│   - runner.go: application service ACPRunner.Run(Job)           │
│   - watchdog.go: ActivityWatchdog                               │
└────────────────────────┬────────────────────────────────────────┘
                         │ usa
┌────────────────────────▼─────── Domínio ────────────────────────┐
│  internal/runtime/events/                                       │
│   - Event (VO imutável)                                         │
│   - EventKind, ToolCallID, Launcher, CancelReason (VOs)         │
│   - SessionState (state pattern)                                │
│   - convert.go: acp.SessionUpdate → Event                       │
│                                                                 │
│  internal/runtime/specs/                                        │
│   - Spec, Launcher (catálogo)                                   │
│   - claude.go (única spec nesta fase)                           │
└────────────────────────┬────────────────────────────────────────┘
                         │ depende de
┌────────────────────────▼─────── Infraestrutura ─────────────────┐
│  internal/runtime/client/                                       │
│   - acpClient: wrapper sobre coder/acp-go-sdk                   │
│  internal/runtime/probe/                                        │
│   - probe.go: EnsureAvailable (PATH lookup, npx fallback)       │
│  internal/runtime/acpfake/                                      │
│   - server.go: servidor ACP fake (test-only, server real do SDK)│
│  internal/runtime/persistence/                                  │
│   - jsonl.go: writer append-only para events.jsonl              │
│   - report.go: enriquece execution_report.md                    │
│   - toolcalls.go: gera tool_calls.md                            │
└─────────────────────────────────────────────────────────────────┘
```

A regra de **fluxo de dependência** é uma só: setas apontam para baixo. Apresentação importa Aplicação; Aplicação importa Domínio; Infraestrutura implementa contratos do Domínio. **Domínio (`internal/runtime/events`, `internal/runtime/specs`) não importa o `coder/acp-go-sdk`** — a conversão ACP→Event vive em `events/convert.go` mas só toca o SDK como tipo de entrada, sem expô-lo na borda pública.

### Fluxo de Dados (uma execução de task no modo `--runtime=acp`)

```
task-loop --runtime=acp --tool=claude
       │
       ▼
acpInvoker.Invoke(ctx, prompt, workDir, model)
       │
       ▼
ACPRunner.Run(ctx, Job{Prompt, WorkDir, EvidenceDir})
       │
       ├──> probe.EnsureAvailable(spec)   ──── falha rápido (RF-03)
       │       resolve: binary | npx@VERSAO_PINADA
       │
       ├──> persistence.NewJSONLWriter(EvidenceDir/events.jsonl)
       │
       ├──> events.NewRuntimeInitEvent(launcher, cmd, args, sdkVer, npmVer)
       │       → emit() para o canal
       │
       ├──> client.Open(ctx, spec)         ──── spawn subprocesso ACP
       │
       ├──> watchdog := NewActivityWatchdog(120s, ctxCancelCause)
       │
       ├──> for update := range client.Updates():    [loop principal]
       │       evt := events.FromACPUpdate(update)
       │       watchdog.Touch()
       │       emit(evt) → fan-out:
       │          ├─ persistence.JSONLWriter.Append(evt)
       │          ├─ renderer.Render(evt) [stdout, se !--quiet]
       │          └─ counters.Inc(evt.Kind)
       │
       ├──> ao receber kind=session_end OU erro do canal:
       │       client.Close()
       │       watchdog.Stop()
       │       persistence.WriteToolCalls(evidenceDir, counters.ToolCalls())
       │       persistence.EnrichExecutionReport(reportPath, Summary{...})
       │
       └──> retorna (stdout consolidado, "", exitCode, error)
```

### Visão Geral dos Componentes

| Pacote | Responsabilidade | Camada |
|---|---|---|
| `internal/runtime` | `ACPRunner` (use case), `Job`, `Summary` | Aplicação |
| `internal/runtime/events` | `Event`, `EventKind`, VOs, `SessionState`, `convert.go` | Domínio |
| `internal/runtime/specs` | `Spec`, `Launcher`, `claude.go` | Domínio |
| `internal/runtime/client` | `acpClient` (wrapper coder/acp-go-sdk) | Infraestrutura |
| `internal/runtime/probe` | `EnsureAvailable` (PATH + npx) | Infraestrutura |
| `internal/runtime/persistence` | `JSONLWriter`, `ReportEnricher`, `ToolCallsRenderer` | Infraestrutura |
| `internal/runtime/render` | `HumanRenderer` (io.Writer injetado) | Infraestrutura |
| `internal/runtime/acpfake` | Servidor ACP fake para testes | Infraestrutura (test-only) |
| `internal/taskloop/acpinvoker.go` | Adapter `AgentInvoker` → `ACPRunner` | Aplicação |
| `cmd/ai_spec_harness/task_loop.go` | Flags novas, validação CLI | Apresentação |

## Modelagem de Domínio (DDD)

### Agregado: `Session` (raiz)

Existe **uma** sessão ACP por chamada `ACPRunner.Run`. A sessão é o *aggregate root* que protege as invariantes:

- Estado avança em transições válidas (state pattern).
- Tem exatamente um `Launcher` decidido em construção.
- Tem exatamente um `EvidenceDir` decidido em construção.
- Eventos são emitidos em ordem monotônica de timestamp.

A entidade `Session` não é persistida em banco — vive apenas em memória durante `Run`. Por isso, *aggregate root* aqui significa a fronteira de consistência transacional do uso, não persistência.

```go
// internal/runtime/session.go (Domínio)
type Session struct {
    id           SessionID        // VO: identificador opaco para correlação em logs
    launcher     Launcher          // VO
    state        SessionState      // state pattern
    eventsCount  EventCount        // collection-of-one type
    unknownCount EventCount
    counters     ToolCallCounters  // first-class collection
}

func NewSession(id SessionID, launcher Launcher) *Session { ... }
func (s *Session) AdvanceTo(next SessionState) error { ... }
func (s *Session) Record(evt events.Event) error { ... }
func (s *Session) Summary() Summary { ... }
```

### Entidades

- **`Session`** (acima): identidade por `SessionID`; estado mutável controlado.
- **Não há outras entidades de domínio.** `Event` é VO; `Spec` é VO de catálogo.

### Value Objects (todos imutáveis, autovalidados)

```go
// internal/runtime/events/kinds.go
type EventKind string  // enum fechado: "agent_message", "agent_thought",
                       // "tool_call_start", "tool_call_update", "session_end",
                       // "runtime_init", "unknown"

func ParseEventKind(s string) (EventKind, error) // valida domínio

// internal/runtime/events/event.go
type Event struct {
    timestamp   time.Time
    kind        EventKind
    toolCallID  ToolCallID  // VO; vazio quando não aplicável
    launcher    Launcher    // VO
    rawKind     string      // só preenchido quando kind == "unknown"
    raw         json.RawMessage
}

func NewEvent(...) (Event, error)  // valida invariantes
func (e Event) Kind() EventKind
func (e Event) ToolCallID() ToolCallID
func (e Event) IsTerminal() bool   // kind == session_end
func (e Event) MarshalJSON() ([]byte, error)  // envelope RF-08

// internal/runtime/events/toolcallid.go
type ToolCallID struct { value string }
func NewToolCallID(s string) ToolCallID
func (t ToolCallID) IsZero() bool
func (t ToolCallID) String() string

// internal/runtime/specs/launcher.go
type Launcher struct {
    kind  launcherKind  // private: binary | npx
    cmd   string
    args  []string
}
func (l Launcher) Kind() string  // "binary" | "npx" (RF-08, RF-10)
func (l Launcher) Command() (string, []string)

// internal/runtime/events/cancel_reason.go
type CancelReason string
const (
    CancelReasonNone              CancelReason = "none"
    CancelReasonActivityTimeout   CancelReason = "activity_timeout"
    CancelReasonContextCanceled   CancelReason = "context_canceled"
    CancelReasonToolError         CancelReason = "tool_error"
    CancelReasonPermissionDenied  CancelReason = "permission_denied"
)
func ParseCancelReason(s string) (CancelReason, error)

// internal/runtime/timeout.go
type ActivityTimeout time.Duration
func NewActivityTimeout(d time.Duration) (ActivityTimeout, error)  // d >= 0
func (t ActivityTimeout) Disabled() bool { return time.Duration(t) == 0 }
```

### Coleções de Primeira Classe (OC #4)

```go
// internal/runtime/events/counters.go
type ToolCallCounters struct {
    byID map[ToolCallID]toolCallRecord  // estado encapsulado, não exposto
}
func NewToolCallCounters() *ToolCallCounters
func (c *ToolCallCounters) Record(evt Event)
func (c *ToolCallCounters) ToolCalls() []ToolCallSummary  // snapshot ordenado
func (c *ToolCallCounters) MappedCount() int
func (c *ToolCallCounters) UnknownKinds() []string
```

### State Pattern

Estados explícitos da `Session` com transições permitidas:

```go
// internal/runtime/state.go
type SessionState string
const (
    StateInit       SessionState = "init"        // antes de runtime_init
    StateRunning    SessionState = "running"     // após primeiro update
    StateAwaiting   SessionState = "awaiting"    // requestPermission
    StateClosed     SessionState = "closed"      // session_end
    StateCanceled   SessionState = "canceled"    // cancelado externamente
)

var validTransitions = map[SessionState][]SessionState{
    StateInit:     {StateRunning, StateCanceled},
    StateRunning:  {StateAwaiting, StateClosed, StateCanceled},
    StateAwaiting: {StateCanceled},  // RF-16: requestPermission → cancel
    StateClosed:   {},
    StateCanceled: {},
}
```

### Application Service: `ACPRunner`

```go
// internal/runtime/runner.go
type ACPRunner struct {
    spec     specs.Spec
    probe    Prober                  // interface
    client   ClientFactory           // interface
    persist  Persistence             // interface
    renderer Renderer                // interface
    clock    Clock                   // interface (determinismo de testes)
}

type Job struct {
    Prompt         string
    WorkDir        string
    EvidenceDir    string
    ActivityTimeout ActivityTimeout
    Quiet          bool
}

type Summary struct {
    Launcher              string
    EventsCount           int
    UnknownEventsCount    int
    CancelReason          CancelReason
    ToolCalls             []ToolCallSummary
    UnknownKinds          []string
}

func NewACPRunner(opts ...Option) *ACPRunner
func (r *ACPRunner) Run(ctx context.Context, j Job) (Summary, error)
```

`ACPRunner.Run` orquestra probe → client → fan-out de eventos → persistência. **Não conhece nada de `acp.SessionUpdate`**: ele consome `events.Event`. A tradução acontece em `events.FromACPUpdate(...)` invocada pelo `client`.

### Fail Fast (R-DDD-001)

- `--runtime=acp --tool != claude` → erro na camada CLI (RF-02), exit 2.
- `--activity-timeout < 0` → erro na camada CLI, exit 2.
- `EnsureAvailable` resolve launcher antes de gerar prompt (RF-03); se falhar, abortar antes de qualquer escrita em `evidence/`.
- `NewToolCallID(s)`, `ParseEventKind(s)`, `NewActivityTimeout(d)` validam no construtor.

### Proibido (R-DDD-001)

- Importar `coder/acp-go-sdk` fora de `internal/runtime/events/convert.go` e `internal/runtime/client/`.
- Construir `Event{}` literal fora de `events_test.go` ou construtores `events.New*`.
- Comparar `EventKind` por string solta fora do pacote `events`.

## Design de Implementação

### Interfaces Chave

```go
// internal/runtime/runner.go — contratos para infra (testes injetam fakes)

type Prober interface {
    EnsureAvailable(ctx context.Context, spec specs.Spec) (specs.Launcher, error)
}

type Client interface {
    Open(ctx context.Context, launcher specs.Launcher, prompt string) error
    Updates() <-chan events.Event       // canal único, fechado em session_end/erro
    Err() error                          // erro acumulado pós-fechamento
    Close() error
}

type ClientFactory interface {
    New(workDir string) Client
}

type Persistence interface {
    AppendEvent(evt events.Event) error
    WriteToolCalls(summary []events.ToolCallSummary) error
    EnrichReport(summary Summary) error
}

type Renderer interface {
    Render(evt events.Event)  // não retorna erro; falha de escrita = log + segue
}

type Clock interface { Now() time.Time }
```

### Tipo `runtime.Event` (tagged union — ADR-010)

```go
// internal/runtime/events/event.go
type Event struct {
    // identidade do envelope
    ts          time.Time
    kind        EventKind
    toolCallID  ToolCallID
    launcher    specs.Launcher  // copiado em runtime_init; vazio nos demais

    // payloads tipados por kind (no máximo um não-nil)
    agentMessage   *AgentMessagePayload
    agentThought   *AgentThoughtPayload
    toolCallStart  *ToolCallStartPayload
    toolCallUpdate *ToolCallUpdatePayload
    sessionEnd     *SessionEndPayload
    runtimeInit    *RuntimeInitPayload
    unknown        *UnknownPayload   // contém RawKind + Raw

    // raw bruto do ACP (RF-08): "raw: <acp.SessionUpdate JSON inteiro>"
    raw json.RawMessage
}
```

**Justificativa:** struct com Kind + ponteiros opcionais (1) serializa em JSONL com `MarshalJSON` único; (2) permite `switch evt.Kind()` claro no consumidor; (3) evita o boilerplate de marshal/unmarshal de sealed interface. Ver ADR-010.

### Spec e Launcher (catálogo)

```go
// internal/runtime/specs/spec.go
type Spec struct {
    ID               string
    DisplayName      string
    Command          string
    FixedArgs        []string
    Fallbacks        []FallbackLauncher
    AccessModeFlag   string  // ex: "--bypass-permissions"
}

type FallbackLauncher struct {
    Command   string
    FixedArgs []string
}

// internal/runtime/specs/claude.go
const (
    ClaudeNpmPackage = "@agentclientprotocol/claude-agent-acp"
    ClaudeNpmVersion = "0.X.Y"   // pinada; atualização exige audit/
    ClaudeSDKVersion = "v0.X.Y"  // sincronizada com go.mod
)

func Claude() Spec {
    return Spec{
        ID:          "claude",
        DisplayName: "Claude (ACP)",
        Command:     "claude-agent-acp",
        Fallbacks: []FallbackLauncher{{
            Command:   "npx",
            FixedArgs: []string{"--yes", ClaudeNpmPackage + "@" + ClaudeNpmVersion},
        }},
        AccessModeFlag: "--bypass-permissions",
    }
}
```

### Watchdog (RF-06)

```go
// internal/runtime/watchdog.go
type ActivityWatchdog struct {
    timeout  ActivityTimeout
    cancel   context.CancelCauseFunc
    lastSeen atomic.Int64  // unix nano
    clock    Clock
    stopCh   chan struct{}
}

func NewActivityWatchdog(timeout ActivityTimeout, cancel context.CancelCauseFunc, clock Clock) *ActivityWatchdog
func (w *ActivityWatchdog) Touch()
func (w *ActivityWatchdog) Start(ctx context.Context)  // ticker a min(timeout/2, 5s)
func (w *ActivityWatchdog) Stop()

// erro sentinela carregado por context.CancelCause:
var ErrActivityTimeout = errors.New("activity timeout")
```

O caller extrai a razão:

```go
err := runner.Run(ctx, job)
if err != nil {
    switch cause := context.Cause(ctx); {
    case errors.Is(cause, ErrActivityTimeout):
        summary.CancelReason = CancelReasonActivityTimeout
    case errors.Is(cause, ErrPermissionDenied):
        summary.CancelReason = CancelReasonPermissionDenied
    case errors.Is(cause, context.Canceled):
        summary.CancelReason = CancelReasonContextCanceled
    default:
        summary.CancelReason = CancelReasonToolError
    }
}
```

### Renderer (RF-11, OC #9)

```go
// internal/runtime/render/human.go
type HumanRenderer struct { out io.Writer }

func NewHumanRenderer(out io.Writer) *HumanRenderer  // out=os.Stdout em prod; io.Discard com --quiet

func (r *HumanRenderer) Render(evt events.Event) {
    switch evt.Kind() {
    case events.KindAgentMessage:  fmt.Fprintf(r.out, "[agent] %s\n", evt.AgentMessage().Text())
    case events.KindAgentThought:  fmt.Fprintf(r.out, "[thought] %s\n", evt.AgentThought().Text())
    case events.KindToolCallStart: fmt.Fprintf(r.out, "[tool] %s %s\n", evt.ToolCallStart().Name(), evt.ToolCallID())
    case events.KindToolCallUpdate where evt.ToolCallUpdate().Final():
        fmt.Fprintf(r.out, "[tool:done] %s %s\n", evt.ToolCallID(), evt.ToolCallUpdate().Status())
    }
}
```

`--quiet` injeta `io.Discard` (RF-11 + decisão #17). Warning agregado de unknowns sai por `os.Stderr` no `ACPRunner.Run` final, **independente de `--quiet`** (decisão #17).

### Modelos de Dados — Envelope `events.jsonl` (RF-08)

```json
{
  "ts": "2026-05-20T14:32:18.123456789Z",
  "kind": "tool_call_start",
  "tool_call_id": "tc_abc123",
  "launcher": "binary",
  "raw": { "<acp.SessionUpdate JSON exato>": "..." }
}
```

Primeiro evento (`runtime_init`):
```json
{
  "ts": "...",
  "kind": "runtime_init",
  "tool_call_id": null,
  "launcher": "npx",
  "raw": {
    "launcher": "npx",
    "command": "npx",
    "args": ["--yes", "@agentclientprotocol/claude-agent-acp@0.X.Y"],
    "sdk_version": "v0.X.Y",
    "npm_version": "0.X.Y"
  }
}
```

### Enriquecimento do `execution_report.md` (RF-10)

Persistência adiciona ao final do arquivo (não substitui):

```markdown
## Runtime ACP

- runtime: acp
- launcher: binary
- events_count: 142
- unknown_events_count: 0
- cancel_reason: none
```

`internal/runtime/persistence/report.go` faz append idempotente: se a seção já existir, sobrescreve apenas ela.

### Endpoints de API

Não aplicável — CLI apenas. Adições nas flags:

| Flag | Tipo | Default | Validação |
|---|---|---|---|
| `--runtime` | `legacy\|acp` | `legacy` | RF-01 |
| `--activity-timeout` | `time.Duration` | `120s` | `>= 0` (RF-07) |
| `--quiet` | `bool` | `false` | — |

`--tool` permanece obrigatório com mesma semântica de hoje; com `--runtime=acp` exige `claude` exato (RF-02).

## Pontos de Integração

### `github.com/coder/acp-go-sdk`

- Pinado em `go.mod` na **última versão stable com tag semântica** (`require github.com/coder/acp-go-sdk vX.Y.Z`); sem pseudo-version; sem `replace`. Upgrades exigem decisão em `audit/`.
- Constante Go `ClaudeSDKVersion` em `internal/runtime/specs/claude.go` deve ser mantida em sincronia com `go.mod` por `go generate` (script `scripts/sync-acp-sdk-version.sh`).
- Importado **somente** por `internal/runtime/client/` e `internal/runtime/events/convert.go`.

### `claude-agent-acp` (subprocesso)

- Resolução por `internal/runtime/probe.EnsureAvailable` na ordem RF-03: PATH → `npx --yes ...@<VERSAO_PINADA>` → falha com 3 remédios.
- Cache em memória por processo (decisão #19): primeira resolução guardada em `sync.OnceValue`-style; sem persistência em disco.
- Stdio: o SDK gerencia pipes; o `client.acpClient` não escreve em `os.Stdin/Stdout` diretamente.

### Filesystem (`internal/fs.FileSystem`)

- Reutilizar a interface existente; `Persistence` recebe `fs.FileSystem` no construtor.
- Paths: `evidence.Path(task)` define o diretório base (decisão #20). Arquivos novos: `events.jsonl`, `tool_calls.md`. `execution_report.md` é enriquecido in-place.
- Segurança (R-SEC-001): paths normalizados via `filepath.Clean`; toda escrita usa `fs.WriteFile` (auditável); subprocesso construído com `exec.Command(name, args...)` — **nunca** shell.

### Telemetria (`internal/telemetry`)

- Adicionar campos opcionais ao evento existente do `task-loop`: `runtime`, `launcher`, `events_count`, `unknown_events_count`, `cancel_reason`.
- Opt-in mantém `GOVERNANCE_TELEMETRY=1`. Sem alteração no caminho legacy.

## Estratégia de Erros

Aderente a **R-ERR-001**.

### Sentinelas (tipos bem definidos, R-ERR-001 "Modelagem")

```go
// internal/runtime/errors.go
var (
    ErrLauncherUnavailable = errors.New("launcher unavailable")
    ErrActivityTimeout     = errors.New("activity timeout")
    ErrPermissionDenied    = errors.New("permission denied")
    ErrSessionAborted      = errors.New("session aborted")
    ErrUnsupportedTool     = errors.New("unsupported tool for runtime")
    ErrInvalidEvent        = errors.New("invalid event")
)
```

### Wrapping (R-ERR-001 "Wrapping")

- Toda fronteira de adapter adiciona contexto: `fmt.Errorf("probing %s: %w", spec.ID, err)`.
- `errors.Is` / `errors.As` no caller; **nunca** comparação por string.
- `context.CancelCause` carrega o sentinel exato; `context.Cause(ctx)` recupera no caller.

### Apresentação (R-ERR-001 "Apresentação")

| Sentinel | Exit code | Mensagem stderr |
|---|---|---|
| `ErrUnsupportedTool` | 2 | `runtime acp suporta apenas --tool claude nesta versão` |
| `ErrLauncherUnavailable` | 2 | `claude-agent-acp não encontrado.\n  Install claude-agent-acp; OR install @agentclientprotocol/claude-agent-acp@<VER> via npm; OR use --runtime=legacy.\n  Veja .specs/adr/009-acp-protocol-adoption.md` |
| `ErrActivityTimeout` | 1 | `agent inactive for >X; cancel_reason=activity_timeout` |
| `ErrPermissionDenied` | 3 | `agent requested permission; configure accessMode=bypassPermissions no claude-agent-acp ou execute em ambiente que pré-aprove. Veja ADR-009` |
| outros | 1 | `<wrapping chain>` |

### Captura e Propagação

- Captura na fronteira externa: `cmd/ai_spec_harness/task_loop.go` traduz erros em exit codes.
- `acpInvoker.Invoke` agrega stdout e retorna `(string, "", exitCode, error)` mantendo o contrato `AgentInvoker`.
- Erros do canal `Client.Updates()` propagam via `client.Err()` lido após fechamento do canal.

### Proibido (R-ERR-001)

- `panic` em qualquer fluxo recuperável (inclui drift do SDK — vira `Event{Kind: unknown}`).
- Silenciar erro de filesystem ao escrever `events.jsonl` (loga e propaga; permite à task continuar registrando no stderr, mas sinaliza `ErrSessionAborted` no fim).
- Comparar erro por string.
- Exibir stack trace bruto: `fmt.Errorf("%w", err)` preserva cadeia sem expor.

## Abordagem de Testes

Aderente a **R-TEST-001**.

### Testes Unitários

Camada de **domínio** (mais densa):

| Pacote | O que testa | Estratégia |
|---|---|---|
| `internal/runtime/events` | `FromACPUpdate` para cada kind ACP relevante + variantes unknown | Table-driven com fixtures `testdata/acp/*.json` (saídas reais capturadas do compozy ou do live test) |
| `internal/runtime/events` | `Event.MarshalJSON` envelope RF-08 | Golden file `testdata/envelopes/*.json` |
| `internal/runtime/events` | `EventKind.Parse`, `ToolCallID.New`, `CancelReason.Parse` | Table-driven |
| `internal/runtime/events` | `SessionState` transitions (state pattern) | Table-driven cobrindo cada par válido/inválido |
| `internal/runtime/events` | `ToolCallCounters` agregação (collection of primary class) | Sequência de eventos → snapshot esperado |
| `internal/runtime/specs` | `Claude()` retorna spec estável (snapshot test) | Golden |
| `internal/runtime/probe` | `EnsureAvailable`: cenários binary OK / só npx / nenhum | Mock `exec.LookPath` via interface injetada |
| `internal/runtime` | `ActivityWatchdog`: dispara após N touches espaçados; não dispara com Touch contínuo | Clock fake (`clockwork` ou interface própria); zero `time.Sleep` |
| `internal/runtime/render` | `HumanRenderer.Render` para cada kind | `bytes.Buffer` como writer |
| `internal/runtime/persistence` | `JSONLWriter` append-only; `EnrichReport` idempotente | `fs.FakeFileSystem` |
| `internal/runtime` | `ACPRunner.Run` orquestração com fakes de cada interface | Cenários: happy path, activity timeout, permission denied, unknown drift |
| `internal/taskloop` | `acpInvoker.Invoke` traduz Summary → contrato AgentInvoker | Fake `ACPRunner` |
| `cmd/ai_spec_harness` | flags: `--runtime=acp --tool=gemini` falha exit 2; `--activity-timeout=0` ok | Table-driven |

**Determinismo (R-TEST-001 "Determinismo"):** zero rede, zero `time.Sleep`, sem dependência de binário externo. Tempo via `Clock` injetado; subprocesso via interface `Prober`/`Client`. Estado compartilhado proibido.

### Testes de Integração

**Decisão:** sim, integration tests necessários.

Critérios atendidos do template:
- [x] Fronteira de IO crítica: o protocolo ACP é o ponto único de falha; mocks de interface não exercitam JSON-RPC sobre stdio.
- [x] Risco de divergência mocks vs realidade: convert.go pode passar em unit e falhar com payloads reais.
- [x] Custo proporcional: `acpfake` usa o próprio `coder/acp-go-sdk` no lado servidor — sem testcontainers, sem rede.

**Layout:**
- `internal/runtime/acpfake/server.go`: servidor ACP em processo usando o SDK; aceita um script `[]ScriptedUpdate` e os emite em ordem.
- `internal/runtime/acp_integration_test.go`: testes que sobem o fake via `os.Pipe()` e exercitam `acpClient + ACPRunner` end-to-end no processo Go. Build tag: nenhuma (rodam no `make test` padrão).

Cenários cobertos:
- Happy path: 3 mensagens + 2 tool calls + session_end → `events.jsonl` com 7 linhas, `tool_calls.md` válido, exit 0.
- Activity timeout: fake fica mudo 200ms; watchdog com timeout 50ms cancela; `cancel_reason=activity_timeout`.
- Unknown drift: fake emite update de tipo desconhecido; `events.jsonl` registra `kind=unknown`; warning agregado em stderr; exit 0.
- Permission requested: fake emite `requestPermission`; `cancel_reason=permission_denied`; exit 3.
- Launcher fallback: probe simula binary ausente; npx escolhido; `runtime_init.launcher=npx`.

### Testes Live

`tests/integration/acp_live/` protegido por build tag `//go:build acp_live` (RF-14).

```bash
go test -tags=acp_live ./tests/integration/acp_live
```

Dentro do teste, `t.Skip` se `claude-agent-acp` ausente do `PATH` **e** `npx` ausente. Sem `ANTHROPIC_API_KEY` válido, o teste roda só o handshake ACP até receber o primeiro `agent_message`/erro de auth e valida que o erro vira `ErrPermissionDenied` ou falha de inicialização limpa — não rodamos prompt real para evitar custo. Documentado em comentário do teste.

CI: job opcional nightly com a build tag; falha não bloqueia merge.

### Sem Rede (R-TEST-001 "Determinismo")

Nenhum teste fora de `acp_live` deve atingir rede. `acpfake` usa pipes em memória. `EnsureAvailable` em testes recebe `Prober` fake.

## Sequenciamento de Desenvolvimento

### Ordem de Build (DAG explícito; ordem topológica)

1. **`internal/runtime/events/`** (domínio puro, sem deps internas)
   - VOs: `EventKind`, `ToolCallID`, `CancelReason`, `Launcher` (em `specs`), `ActivityTimeout`
   - `Event` + payloads tipados
   - `MarshalJSON` envelope
   - `SessionState` + transições
   - `ToolCallCounters`
   - Testes unitários todos os VOs e collection
2. **`internal/runtime/specs/`**
   - `Spec`, `FallbackLauncher`
   - `Claude()` com constantes pinadas
   - Snapshot test
3. **`internal/runtime/events/convert.go`**
   - Adiciona dependência em `coder/acp-go-sdk` (primeiro ponto de entrada no go.mod)
   - `FromACPUpdate` table-driven com fixtures
4. **`internal/runtime/probe/`**
   - Interface `LookPather`; impl real + fake
   - `EnsureAvailable` com cache `sync.OnceValue`
5. **`internal/runtime/watchdog.go`**
   - `ActivityWatchdog` + `Clock` interface
   - Testes com clock fake
6. **`internal/runtime/persistence/`**
   - `JSONLWriter` (append idempotente)
   - `ReportEnricher` (seção `## Runtime ACP`)
   - `ToolCallsRenderer` (markdown)
7. **`internal/runtime/render/`**
   - `HumanRenderer` injetando `io.Writer`
8. **`internal/runtime/client/`**
   - `acpClient` sobre `coder/acp-go-sdk`
   - Adapta `acp.SessionUpdate` → canal `<-chan events.Event` usando `convert.FromACPUpdate`
9. **`internal/runtime/acpfake/`** (paralelo a #8)
   - Servidor ACP fake via SDK
   - `[]ScriptedUpdate` API
10. **`internal/runtime/runner.go`** — `ACPRunner` orquestra tudo
11. **Integration tests** `internal/runtime/acp_integration_test.go`
12. **`internal/taskloop/acpinvoker.go`** — adapter para `AgentInvoker`
13. **`cmd/ai_spec_harness/task_loop.go`** — flags novas + validação
14. **`tests/integration/acp_live/`** — build tag `acp_live`
15. **README + CHANGELOG + telemetria** — atualizações documentais

Cada item produz um commit/PR pequeno, testado isoladamente. `dependencies` no frontmatter das tasks geradas por `create-tasks` deve refletir essa ordem.

### Dependências Técnicas

- `github.com/coder/acp-go-sdk` adicionado no item 3 do DAG.
- Atualização do `Makefile` para suportar `make test-acp-live`.
- Script `scripts/sync-acp-sdk-version.sh` (item 2) para manter constante Go ↔ go.mod alinhadas.

## Monitoramento e Observabilidade

### Métricas (telemetria existente, opt-in)

Campos adicionados ao evento de telemetria do `task-loop`:

| Campo | Tipo | Cardinalidade |
|---|---|---|
| `runtime` | string | `legacy` \| `acp` (2 valores) |
| `launcher` | string | `binary` \| `npx` \| `none` (caminho legacy) |
| `events_count` | int | unbounded |
| `unknown_events_count` | int | unbounded |
| `cancel_reason` | string | enum CancelReason (5 valores) |

### Logs (R-OBS-001)

- Logger: continuar com o logger existente do harness.
- Eventos relevantes:
  - `info` em `probe.EnsureAvailable`: launcher escolhido.
  - `warn` em conversão `unknown`: kind + count agregado emitido **uma vez no fim**.
  - `error` em falhas de I/O do `events.jsonl`: registrado, não interrompe sessão.

### Sinais para revisar a decisão

- `unknown_events_count` > 5% das execuções → drift do SDK; abrir issue.
- `cancel_reason=activity_timeout` > 10% → timeout default mal calibrado.
- `cancel_reason=permission_denied` > 0 em CI → ambiente sem `bypassPermissions`.

## Considerações Técnicas

### Decisões Chave

Decisões materiais com ADR separada (criadas neste ciclo):

- **ADR-009 — Adoção do ACP via `coder/acp-go-sdk`** (`.specs/adr/009-acp-protocol-adoption.md`, status `Proposta`). Mantém-se. Migrar para `Aceita` no merge da implementação (RF-15).
- **ADR-010 — `runtime.Event` como tagged union** (`.specs/prd-acp-runtime-claude/adr-010-event-tagged-union.md`, status `Proposta`). Justifica struct com Kind + ponteiros opcionais vs sealed interface.

Decisões não materiais (registradas aqui, sem ADR — baixo impacto / facilmente reversíveis):
- Layout split de pacotes `runtime/{events,specs,client,probe,persistence,render,acpfake}` (decisão #13). Reversível para monolítico se prova de over-engineering aparecer.
- `acpfake` usa servidor real do `coder/acp-go-sdk` (decisão #21). Reversível: swap por mock se SDK fizer fake server lado-server inviável.
- Reaproveitar `executor_template.tmpl` sem mudanças (decisão #22).
- Constantes Go + probe leve para `sdk_version`/`npm_version` (decisão #23).
- `context.CancelCause` para propagação de razão (decisão #24).

### Conformidade com Padrões

- **R-DDD-001** ✓ — agregado `Session`, VOs imutáveis, fail fast, state pattern, sem struct anêmica.
- **R-ERR-001** ✓ — sentinelas tipadas, `%w` wrapping, exit codes documentados, sem `panic`, sem comparação por string.
- **R-SEC-001** ✓ — subprocesso sempre via `exec.Command(name, args...)`, sem shell; paths normalizados; segredos só via ENV (e somente lidos pelo `claude-agent-acp`, não pelo harness); npx pinado por versão.
- **R-TEST-001** ✓ — unit determinísticos sem rede, integration com fake real, live opt-in, table-driven, golden files.
- **R-OBS-001** — campos novos opt-in via telemetria existente.

### Object Calisthenics Aplicado (escopo: pacote `internal/runtime/`)

Aplicação **seletiva** das 9 regras (heurísticas, não dogma; `references/rules.md`).

| # | Regra | Aplicada em |
|---|---|---|
| 1 | Uma camada de indentação | `ACPRunner.Run`: extrair `runHappyPath`/`closeSession` para evitar mais que 2 níveis. `convert.FromACPUpdate`: usar early return por kind, sem switch aninhado. |
| 2 | Sem `else` | Early return em todos os construtores VO e em `EnsureAvailable`. |
| 3 | Encapsular primitivos | `ToolCallID`, `EventKind`, `CancelReason`, `Launcher.kind`, `ActivityTimeout` — todos VOs (cumprem invariante e/ou semântica de domínio). **Não** encapsular `string Prompt` em `Job` (sem invariante de domínio — primitivo cru basta). |
| 4 | Coleção de primeira classe | `ToolCallCounters` (encapsula `map[ToolCallID]record`); `[]ScriptedUpdate` no `acpfake` (com `Append`, `WithDelay`). |
| 5 | Um ponto por linha | `evt.AgentMessage().Text()` aceitável (1 hop); `summary.toolCalls[0].status` ⇒ trocar por método `summary.FirstToolCallStatus()` se aparecer em mais de um caller. |
| 6 | Sem abreviações opacas | `acpInvoker` (ACP é sigla oficial — aceita); `evt`, `ctx`, `cfg` idiomáticos Go — mantidos. `tmp`, `obj`, `svcX` proibidos. |
| 7 | Entidades pequenas | `ACPRunner` ≤ ~150 LOC; `Session` ≤ ~120 LOC; arquivos por responsabilidade (um arquivo = um conceito). |
| 8 | ≤ 2 variáveis de instância | **Não aplicada cegamente.** `ACPRunner` tem 6 colaboradores injetados — todos coesos (mesma responsabilidade do use case). `Spec` tem 5 campos (configuração — exceção explícita da regra). |
| 9 | Sem getters/setters mecânicos | Campos privados em todos os VOs; expostos via métodos com intenção: `Event.ToolCallID()`, `Session.AdvanceTo()`. Sem `SetXxx` mecânico. |

**Casos onde recusei OC** (registrar conforme `references/rules.md` "Quando interromper"):
- Não criar interface `Logger` por hora — log slog direto basta; sem consumidor real para abstrair.
- Não criar VO `Prompt` — string crua já carrega a semântica via tipo do parâmetro.
- Não dividir `ACPRunner` em mais classes — coesão alta; divisão aumentaria navegação sem reduzir acoplamento.

### Padrões de Projeto Aplicados

| Pattern | Onde | Justificativa |
|---|---|---|
| **Adapter** | `internal/taskloop/acpinvoker.go` adapta `ACPRunner` à `AgentInvoker` | Preservar API atual sem refatorar `taskloop`. |
| **Strategy** | `Prober`/`Client`/`Renderer` interfaces injetadas em `ACPRunner` | Permitir fakes em teste; um único algoritmo de orquestração com componentes variáveis. |
| **Template Method** | `ACPRunner.Run` define a sequência `probe → open → consume → close`; passos delegados | Sequência fixa, partes variáveis injetadas. |
| **State** | `SessionState` com `validTransitions` | Invariantes de transição explícitas (RF-16 exige cancelar em `awaiting`). |
| **Functional Options** | `NewACPRunner(opts ...Option)` aceita `WithClock`, `WithProber`, etc. | Construtor com muitos colaboradores opcionais por default. |
| **Decorator** | (futuro, não nesta fase) `loggingClient` wrapping `Client` | Hoje sem demanda; adicionar quando logs de IO virarem necessidade. |
| **Observer (canal)** | `Client.Updates() <-chan events.Event` consumido por fan-out manual | Idiomático Go; sem dispatcher genérico. |

Patterns **rejeitados** (`patterns-structural.md` "Sinais de uso indevido"):
- Singleton para `Registry` de specs — usar variável de pacote `specs.Claude()` resolve.
- Abstract Factory para `Client` — apenas um runtime nesta fase; uma factory function basta.

### Riscos Conhecidos

| Risco | Probabilidade | Impacto | Mitigação |
|---|---|---|---|
| `coder/acp-go-sdk` introduz breaking change entre tags | Alta (SDK < 1.0) | Build quebra | Pin estrito; camada `convert.go` isolada; teste de regressão por golden file |
| `claude-agent-acp` muda flags / fluxo de bypass | Média | Sessão não inicia | `Spec.AccessModeFlag` configurável sem mexer no client; manual de troubleshooting no README |
| Fake ACP server diverge do real | Média | Unit verdes, live falha | Live test opt-in em CI nightly; golden tests dos envelopes em ambos os caminhos |
| Watchdog falso-positivo em tarefas longas | Baixa | Cancel indevido | Default 120s (alinhado com compozy); flag `--activity-timeout` sobrescritível |
| `npx --yes` em ambiente air-gapped sem cache npm | Alta em CI offline | RF-03 falha | Mensagem clara com 3 remédios; ADR-009 documenta |
| Sobreposição com legacy aumenta dívida | Média (longo prazo) | Manutenção dual | Critério explícito de remoção em ADR futuro quando 4/4 tools tiverem cobertura ACP |
| Vazamento de pipe stdio do subprocesso | Baixa | Goroutine leak | `defer client.Close()` no `ACPRunner.Run`; `ctx` cancelado encerra reads do SDK |

### Áreas que Precisam de Pesquisa

- Comportamento exato do `coder/acp-go-sdk` quando o subprocesso morre abruptamente (SIGKILL): garante fechamento do canal? Validar em integration test usando `os.Process.Kill` no fake.
- Existe API estável no SDK para `requestPermission`? Confirmar antes de implementar RF-16; se não, detectar pelo kind do `SessionUpdate`.

### Plano de Rollout

1. **Iteração 1 (interno):** merge com `--runtime=acp` documentado como experimental; default `legacy`; CI roda unit + integration com fake (sem live).
2. **Iteração 2 (validação):** mantenedor executa `task-loop --runtime=acp` em 3 tasks reais por uma semana; coleta `cancel_reason` e `unknown_events_count`. Critério de promoção: 0 unknowns inesperados, watchdog não dispara falsamente.
3. **Iteração 3 (documentação):** README atualizado com seção "Runtime ACP (experimental)"; CHANGELOG entry; ADR-009 transita para `Aceita`.
4. **Iteração 4 (CI nightly):** habilitar job `acp_live` em nightly. Falha não bloqueia, mas alerta.
5. **Promoção:** apenas em **PRD futuro** após 30 dias de estabilidade. Não promover default nesta entrega.
6. **Rollback:** reverter por flag (`--runtime=legacy`) é suficiente. Para reverter código, basta remover `cmd/ai_spec_harness/task_loop.go` parsing de `--runtime=acp` (mantém pacote `runtime/` dormente). Reverter `go.mod` exige remover dependência do SDK.

### Arquivos Relevantes e Dependentes

**Novos (criar):**
- `internal/runtime/runner.go`, `runner_test.go`
- `internal/runtime/watchdog.go`, `watchdog_test.go`
- `internal/runtime/clock.go`
- `internal/runtime/errors.go`
- `internal/runtime/events/event.go`, `event_test.go`
- `internal/runtime/events/kinds.go`, `kinds_test.go`
- `internal/runtime/events/toolcallid.go`, `toolcallid_test.go`
- `internal/runtime/events/cancel_reason.go`, `cancel_reason_test.go`
- `internal/runtime/events/state.go`, `state_test.go`
- `internal/runtime/events/counters.go`, `counters_test.go`
- `internal/runtime/events/payloads.go`
- `internal/runtime/events/convert.go`, `convert_test.go`, `testdata/acp/*.json`, `testdata/envelopes/*.json`
- `internal/runtime/specs/spec.go`, `claude.go`, `claude_test.go`
- `internal/runtime/probe/probe.go`, `probe_test.go`
- `internal/runtime/client/client.go`, `client_test.go`
- `internal/runtime/persistence/jsonl.go`, `report.go`, `toolcalls.go` + tests
- `internal/runtime/render/human.go`, `human_test.go`
- `internal/runtime/acpfake/server.go`, `script.go`
- `internal/runtime/acp_integration_test.go`
- `internal/taskloop/acpinvoker.go`, `acpinvoker_test.go`
- `tests/integration/acp_live/live_test.go` (build tag `acp_live`)
- `scripts/sync-acp-sdk-version.sh`
- `.specs/prd-acp-runtime-claude/adr-010-event-tagged-union.md`

**Modificados:**
- `cmd/ai_spec_harness/task_loop.go` — flags `--runtime`, `--activity-timeout`, `--quiet`
- `cmd/ai_spec_harness/flags.go` — registro das flags
- `internal/taskloop/agent.go` — apenas se necessário expor hook (mantém interface igual)
- `internal/taskloop/runloop.go` — repassa `Job` para `acpInvoker` quando runtime=acp
- `internal/telemetry/*.go` — campos novos opt-in
- `go.mod`, `go.sum` — `coder/acp-go-sdk` pinado
- `Makefile` — target `test-acp-live`
- `README.md` — seção "Runtime ACP (experimental)"
- `CHANGELOG.md` — entry
- `AGENTS.md` — link para ADR-009 e ADR-010
- `.specs/adr/009-acp-protocol-adoption.md` — transição de status no merge

**Inalterados (referência apenas):**
- `internal/evidence/evidence.go` — reaproveitado via `Persistence`
- `internal/fs/fs.go` — reaproveitado
- `internal/taskloop/executor_template.tmpl` — reaproveitado sem alteração (decisão #22)

## Mapeamento Requisito → Decisão → Teste

| RF | Decisão de desenho | Teste que cobre |
|---|---|---|
| RF-01 | Flag `--runtime=legacy\|acp` em `cmd/ai_spec_harness/task_loop.go` | `cmd/ai_spec_harness/task_loop_test.go` table-driven |
| RF-02 | Validação CLI; sentinel `ErrUnsupportedTool` | `cmd/ai_spec_harness/task_loop_test.go` |
| RF-03 | `probe.EnsureAvailable` ordem PATH→npx@VERSAO_PINADA→fail | `internal/runtime/probe/probe_test.go` + integration "launcher fallback" |
| RF-04 | `client.acpClient.Open` + `Updates()` | `internal/runtime/acp_integration_test.go` "happy path" |
| RF-05 | `events.FromACPUpdate` + `Event{Kind: unknown}` + stderr aggregate | `convert_test.go` table-driven + integration "unknown drift" |
| RF-06 | `ActivityWatchdog` + `ErrActivityTimeout` via CancelCause | `watchdog_test.go` (clock fake) + integration "activity timeout" |
| RF-07 | `NewActivityTimeout(d)` valida `d >= 0`; `d == 0` desabilita | `watchdog_test.go` + CLI test |
| RF-08 | `Event.MarshalJSON` envelope; `runtime_init` primeiro evento | `event_test.go` golden + integration |
| RF-09 | `persistence.ToolCallsRenderer` | `persistence/toolcalls_test.go` + integration |
| RF-10 | `persistence.ReportEnricher` seção `## Runtime ACP` | `persistence/report_test.go` (idempotência) + integration |
| RF-11 | `HumanRenderer` com `io.Writer`; `--quiet` → `io.Discard` | `render/human_test.go` + CLI test |
| RF-12 | Caminho legacy intocado | suite existente continua passando (gate de merge) |
| RF-13 | `acpfake.Server` + integration tests | `internal/runtime/acp_integration_test.go` |
| RF-14 | `tests/integration/acp_live/` build tag `acp_live` | `make test-acp-live` |
| RF-15 | ADR-009 referenciado e migrado a `Aceita` no merge | gate manual |
| RF-16 | `SessionState.StateAwaiting` → cancel imediato; `ErrPermissionDenied` exit 3 | integration "permission requested" + CLI exit code test |

---

**Status final:** especificação técnica completa, zero questões em aberto, pronta para `create-tasks`.
