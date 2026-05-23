<!-- spec-hash-prd: __pending_sync__ -->
<!-- Para materializar o hash: rodar `ai-spec sync-spec-hash .specs/prd-claude-cli-acp-2026/tasks.md` quando tasks.md existir. -->

# Especificação Técnica — Claude-CLI 2026 (Paridade Compozy Acima da Camada ACP)

**PRD:** [`prd.md`](prd.md)
**Data:** 2026-05-21
**Status:** Pronta para implementação (após `--sync` do spec-hash)
**ADRs:**
- [ADR-014 — Claude CLI 2026 (esta entrega)](../adr/014-claude-cli-acp-native.md)
- [ADR-013 — Codex CLI ACP nativo (`BootstrapArgs` infra reusada)](../adr/013-codex-cli-acp-native.md)
- [ADR-012 — Copilot CLI ACP nativo (generalização runner)](../adr/012-copilot-cli-acp-native.md)
- [ADR-011 — Agent Registry declarativo (consumido por MCP `run_agent`)](../adr/011-agent-registry-declarativo.md)
- [ADR-009 — ACP protocol adoption (base de transporte)](../adr/009-acp-protocol-adoption.md)
- [ADR-008 — Paridade multi-tool (invariantes a estender)](../../docs/adr/008-parity-multi-tool-invariants.md)
- [ADR-006 — Telemetria opt-in (campos novos)](../../docs/adr/006-telemetria-feedback-cycle.md)

**Pesquisa:** [`docs/research/compozy-adaptation-claude-2026.md`](../../docs/research/compozy-adaptation-claude-2026.md)

---

## Resumo Executivo

Adicionar cinco capacidades acima da camada ACP em Claude, replicando o padrão de `compozy/compozy@7f38c44` mantendo os diferenciais do harness. Implementação decomposta em quatro waves (F2 → F5), cada uma com tasks independentes paralelizáveis dentro da wave via `execute-all-tasks`.

**Wave F2-Claude (~1 sprint):** Novo `internal/runtime/mcpserver/` expõe tool `run_agent` (paridade Compozy `internal/core/agents/mcpserver/server.go`); novo `internal/runtime/events/normalize.go` com tabelas em `.agents/normalization-rules.yaml` (paridade `internal/core/agent/tool_call_input.go::buildNormalizedToolUseBlock`).

**Wave F3-Claude (~1.5 sprints):** Novo `internal/runtime/memory/` espelhando Compozy 2-tier (limites 150/12KB workflow + 200/16KB task); novo `internal/runtime/hooks/dispatcher.go` com 6 pontos canônicos + migração inicial de 2 shell hooks para Go (governance, token_budget).

**Wave F4-Claude (~3 dias):** Estender `internal/runtime/events/convert.go` e `internal/evidence/evidence.go` com campos `cache_read_tokens`, `cache_creation_tokens`, `thinking_tokens`, `tool_calls_normalized_count`.

**Wave F5-Claude (~3 dias):** Flag `--auto-review` em `task_loop.go`; nova invocação de `ACPRunner` com skill `review` + diff; hook `session.post_review`.

Nenhuma alteração breaking. Defaults preservam comportamento atual de F1-Claude.

---

## Arquitetura do Sistema

### Diagrama de fluxo (sessão Claude com F2..F5 ativos)

```
ai-spec task-loop --tool claude --runtime acp --mcp-nested --auto-review .specs/prd-X
                                        │
                                        ▼
                          cmd/ai_spec_harness/task_loop.go
                                        │  (resolve specs.Claude(), flags)
                                        ▼
                          internal/runtime/runner.go::Run()
        ┌───────────────────────────────┼────────────────────────────────┐
        │  Fase 1: probe launcher (já existente)                          │
        │  Fase 2: criar persistence (já existente)                       │
        │  Fase 3: emitir runtime_init (já existente)                     │
        │                                                                  │
        │  ★ F3-Claude: hooks.Dispatch("runtime.pre_open", &job)          │
        │      └─ governance hook valida AGENTS.md → aborta se ausente   │
        │                                                                  │
        │  ★ F3-Claude: memory.Read(workflow+task) → prepend ao prompt   │
        │      └─ se NeedsCompaction: anexa diretiva textual              │
        │                                                                  │
        │  ★ F3-Claude: hooks.Dispatch("prompt.pre_build", &prompt)      │
        │  ★ F3-Claude: hooks.Dispatch("prompt.post_build", &prompt)     │
        │      └─ token_budget hook valida via internal/metrics          │
        │                                                                  │
        │  ★ F2-Claude: se --mcp-nested: spawnar mcpserver.Server        │
        │      └─ inject --mcp-server stdio://... no launcher            │
        │                                                                  │
        │  Fase 4: c.Open(ctx, effectiveLauncher, prompt) [existente]     │
        │  Fase 5: watchdog [existente]                                    │
        │                                                                  │
        │  Fase 6: loop de eventos                                         │
        │     for evt := range c.Updates():                                │
        │       ★ F2-Claude: evt.RawInput → normalize.BuildNormalized(..) │
        │       ★ F3-Claude: hooks.Dispatch("tool_call.pre_dispatch")    │
        │       persist.AppendEvent(evt) [existente; estendido com fields]│
        │       ★ F3-Claude: hooks.Dispatch("tool_call.post_complete")   │
        │       ★ F4-Claude: extrair cache_read/thinking → Summary       │
        │                                                                  │
        │  ★ F3-Claude: hooks.Dispatch("session.post_end", &summary)     │
        │      └─ memory.persist hook escreve MEMORY.md/<task>.md         │
        │                                                                  │
        │  ★ F5-Claude: se --auto-review: spawnar nova ACPRunner          │
        │      └─ prompt = skill review + git diff                        │
        │      └─ persist em evidence/<task>/review.md                    │
        │      └─ parse hard issues → Summary.ReviewStatus="blocked"      │
        │      └─ hooks.Dispatch("session.post_review", &review)         │
        │                                                                  │
        │  Fase 7: persist.EnrichReport(summary) [existente; campos F4]    │
        └─────────────────────────────────────────────────────────────────┘
```

Componentes (todos os caminhos relativos a `/Users/jailtonjunior/Git/orchestrator/`):

```
internal/runtime/
├── runner.go                          ★ modificado: dispatcher calls, memory read, auto-review spawn
├── mcpserver/                         ★ NOVO (F2-Claude)
│   ├── server.go                      ★ tool run_agent (stdio MCP)
│   ├── server_test.go                 ★ T-MCP-01..T-MCP-06
│   ├── engine.go                      ★ spawn child ACPRunner
│   └── engine_test.go
├── events/
│   ├── convert.go                     ★ modificado (F4-Claude): cache/thinking
│   └── normalize.go                   ★ NOVO (F2-Claude): BuildNormalizedToolCall
├── memory/                            ★ NOVO (F3-Claude)
│   ├── store.go                       ★ Read/Write/Limits 2-tier
│   └── store_test.go                  ★ T-MEM-01..T-MEM-05
└── hooks/                             ★ NOVO (F3-Claude)
    ├── dispatcher.go                  ★ Hook interface + Dispatch
    ├── governance.go                  ★ migra validate-governance.sh
    ├── token_budget.go                ★ migra validate-token-budget.sh
    ├── memory_persist.go              ★ escreve MEMORY.md em session.post_end
    └── dispatcher_test.go             ★ T-HOOK-01..T-HOOK-04

internal/evidence/
└── evidence.go                        ★ modificado (F4-Claude): seção Métricas Claude-2026

internal/parity/
└── invariants.go                      ★ modificado (F2/F3): +2 invariantes

cmd/ai_spec_harness/
└── task_loop.go                       ★ modificado: 7 flags novas

.agents/
└── normalization-rules.yaml           ★ NOVO (F2-Claude): tabela alias por driver

CLAUDE.md                              ★ modificado: §Runtime Capabilities (F2-Claude)
```

---

## Design de Implementação

### Interfaces Chave

#### F2-Claude

**`internal/runtime/mcpserver/server.go`**

```go
package mcpserver

const ReservedToolName = "run_agent"

type RunAgentInput struct {
    AgentName string `json:"agent_name"`
    Prompt    string `json:"prompt"`
    Model     string `json:"model,omitempty"`
    Timeout   int    `json:"timeout,omitempty"` // segundos; default 300; max 1800
}

type RunAgentOutput struct {
    Summary     string `json:"summary"`
    EvidenceDir string `json:"evidence_dir"`
}

type Server interface {
    // Serve roda o stdio MCP loop até o context cancelar.
    Serve(ctx context.Context, in io.Reader, out io.Writer) error
}

func New(registry agents.Registry, maxDepth int, persistFactory PersistenceFactory) Server
```

Profundidade máxima rastreada via env var `AISPEC_RUN_AGENT_CONTEXT` (JSON com `parent_session_id`, `depth`, `workspace_root`). Child runner herda contexto via env var ao spawnar.

**`internal/runtime/events/normalize.go`**

```go
package events

type NormalizedToolCall struct {
    RawName        string
    NormalizedName string
    Kind           acp.ToolKind
    RawInput       json.RawMessage
    NormalizedInput json.RawMessage
    Locations      []acp.ToolCallLocation
}

func BuildNormalizedToolCall(
    driverID string,
    kind acp.ToolKind,
    rawName string,
    rawInput json.RawMessage,
    locations []acp.ToolCallLocation,
) (NormalizedToolCall, error)
```

Tabelas via `go:embed`:

```go
//go:embed normalization-rules.yaml
var defaultRulesYAML []byte
```

Resolução: `.agents/normalization-rules.yaml` no workspace **vence** sobre o embeded default (override por projeto).

#### F3-Claude

**`internal/runtime/memory/store.go`**

```go
package memory

const (
    DirName                  = "memory"
    WorkflowFileName         = "MEMORY.md"
    DefaultWorkflowLineLimit = 150
    DefaultWorkflowByteLimit = 12 * 1024
    DefaultTaskLineLimit     = 200
    DefaultTaskByteLimit     = 16 * 1024
)

type Limits struct {
    WorkflowLines int
    WorkflowBytes int
    TaskLines     int
    TaskBytes     int
}

type FileState struct {
    Path            string
    LineCount       int
    ByteCount       int
    NeedsCompaction bool
}

type Document struct {
    FileState
    Content string
    Exists  bool
}

type WriteMode string
const (
    WriteModeReplace WriteMode = "replace"
    WriteModeAppend  WriteMode = "append"
)

type Store interface {
    ReadWorkflow(ctx context.Context) (Document, error)
    ReadTask(ctx context.Context, taskFileName string) (Document, error)
    WriteWorkflow(ctx context.Context, content string, mode WriteMode) error
    WriteTask(ctx context.Context, taskFileName, content string, mode WriteMode) error
}

func New(tasksDir string, limits Limits) Store
```

**`internal/runtime/hooks/dispatcher.go`**

```go
package hooks

type Event interface { Kind() string }

type Hook interface {
    Name() string
    Run(ctx context.Context, evt Event) error
}

type Dispatcher interface {
    Register(point string, hook Hook)
    Dispatch(ctx context.Context, point string, evt Event) error
}

func New() Dispatcher

// Pontos canônicos (constantes exportadas)
const (
    PointRuntimePreOpen        = "runtime.pre_open"
    PointPromptPreBuild        = "prompt.pre_build"
    PointPromptPostBuild       = "prompt.post_build"
    PointToolCallPreDispatch   = "tool_call.pre_dispatch"
    PointToolCallPostComplete  = "tool_call.post_complete"
    PointSessionPostEnd        = "session.post_end"
    PointSessionPostReview     = "session.post_review" // F5-Claude
)
```

**Tipos de eventos (cada ponto recebe tipo específico):**

```go
type RuntimePreOpenEvent struct {
    WorkDir   string
    SpecID    string
    Launcher  string
}

type PromptBuildEvent struct {
    Prompt *string // mutable; hooks podem alterar antes do build final
    Spec   string
}

type ToolCallEvent struct {
    Call  events.NormalizedToolCall
    Phase string // "pre_dispatch" | "post_complete"
}

type SessionPostEndEvent struct {
    Summary *runtime.Summary // mutable: hooks podem enriquecer
}

type SessionPostReviewEvent struct {
    Summary    *runtime.Summary
    ReviewPath string
    Blocked    bool
}
```

#### F4-Claude

Extensão de `internal/runtime/runner.go::Summary`:

```go
type Summary struct {
    // ... campos existentes ...

    // F4-Claude — Métricas Claude-2026 (opcionais; 0 quando ausentes no payload).
    CacheReadTokens         int
    CacheCreationTokens     int
    ThinkingTokens          int
    ToolCallsNormalizedCount int

    // F5-Claude
    ReviewStatus            string // "" | "ok" | "blocked"
    ReviewPath              string // caminho de evidence/<task>/review.md
}
```

#### F5-Claude — apenas integração; sem interface nova

A invocação de auto-review reusa `ACPRunner` existente com `Job` configurado:

```go
reviewJob := runner.Job{
    Prompt:   buildReviewPrompt(skillReviewBody, gitDiffOutput),
    WorkDir:  parentJob.WorkDir,
    EvidenceDir: filepath.Join(parentJob.EvidenceDir, "review"),
    ActivityTimeout: 5 * time.Minute,
    Quiet:    true,
}
```

### Modelos de Dados

**`.agents/normalization-rules.yaml`** (exemplo inicial, ~30 linhas):

```yaml
# Tabelas de normalização de tool-calls por driver ACP.
# Carregadas em runtime via go:embed; override por projeto em .agents/normalization-rules.yaml local.
# Referência: docs/research/compozy-adaptation-claude-2026.md §"Normalização de tool-calls"

version: 1

aliases:
  claude:
    # nome_raw : nome_normalizado
    bash:       bash
    read_file:  read
    write_file: write
    str_replace_editor: edit

  codex:
    shell:        bash
    search_query: web_search
    image_query:  image_search

  copilot:
    run:   bash
    fetch: web_search

  cursor:
    terminal:      bash
    grep_search:   grep
    file_search:   glob

input_mappings:
  # canonicalização de campos de input por driver
  claude:
    bash:
      command: command
  codex:
    shell:
      cmd:     command  # Codex usa cmd; canonizar para command
  cursor:
    read_file:
      startLineNumberBaseOne: start_line
      endLineNumberBaseOne:   end_line
```

### Assinaturas principais

```go
// internal/runtime/runner.go — modificação cirúrgica em Run()
//
// Após Fase 3 (runtime_init) e antes de Fase 4 (c.Open):

// ★ F3-Claude: hook runtime.pre_open
if err := r.hooks.Dispatch(ctx, hooks.PointRuntimePreOpen, &hooks.RuntimePreOpenEvent{
    WorkDir: j.WorkDir, SpecID: r.spec.ID, Launcher: launcherCmd,
}); err != nil {
    return Summary{}, fmt.Errorf("runner: hook runtime.pre_open: %w", err)
}

// ★ F3-Claude: memory read + injeção
if r.memory != nil {
    wf, _ := r.memory.ReadWorkflow(ctx)
    tk, _ := r.memory.ReadTask(ctx, j.TaskFileName)
    j.Prompt = injectMemoryContext(j.Prompt, wf, tk)
}

// ★ F3-Claude: hook prompt.pre_build
ev := &hooks.PromptBuildEvent{Prompt: &j.Prompt, Spec: r.spec.ID}
if err := r.hooks.Dispatch(ctx, hooks.PointPromptPreBuild, ev); err != nil {
    return Summary{}, fmt.Errorf("runner: hook prompt.pre_build: %w", err)
}
// ... (final prompt build) ...
if err := r.hooks.Dispatch(ctx, hooks.PointPromptPostBuild, ev); err != nil {
    return Summary{}, fmt.Errorf("runner: hook prompt.post_build: %w", err)
}

// ★ F2-Claude: spawn MCP server se habilitado
if j.MCPNested {
    mcpAddr, mcpStop, err := r.mcpServer.Serve(ctx)  // retorna stdio:// addr e stop func
    if err != nil { return Summary{}, err }
    defer mcpStop()
    // injeta no launcher
    effectiveLauncher = specs.NewBinaryLauncher(launcherCmd, append(argv, "--mcp-server", mcpAddr)...)
}
```

Loop de eventos modificado:

```go
for evt := range c.Updates() {
    wd.Touch()
    counters.Record(evt)

    // ★ F2-Claude: normalização antes da persistência
    if tc := evt.AsToolCall(); tc != nil {
        norm, err := events.BuildNormalizedToolCall(r.spec.ID, tc.Kind, tc.RawName, tc.RawInput, tc.Locations)
        if err == nil {
            tc.NormalizedName = norm.NormalizedName
            tc.NormalizedInput = norm.NormalizedInput
            summary.ToolCallsNormalizedCount++
        }
        _ = r.hooks.Dispatch(ctx, hooks.PointToolCallPreDispatch, &hooks.ToolCallEvent{Call: norm, Phase: "pre_dispatch"})
    }

    // ★ F4-Claude: extrair métricas Claude-2026
    extractClaudeMetrics(&summary, evt)

    // ... persist existente ...
    if persist != nil {
        _ = persist.AppendEvent(evt)
    }

    if tc := evt.AsToolCall(); tc != nil && tc.IsComplete() {
        _ = r.hooks.Dispatch(ctx, hooks.PointToolCallPostComplete, &hooks.ToolCallEvent{Call: tc.Normalized, Phase: "post_complete"})
    }
}

// ★ F3-Claude: session.post_end
postEnd := &hooks.SessionPostEndEvent{Summary: &summary}
_ = r.hooks.Dispatch(ctx, hooks.PointSessionPostEnd, postEnd)

// ★ F5-Claude: auto-review se habilitado
if j.AutoReview {
    reviewSummary, err := r.runAutoReview(ctx, j, summary)
    if err == nil {
        summary.ReviewStatus = reviewSummary.Status
        summary.ReviewPath = reviewSummary.Path
        _ = r.hooks.Dispatch(ctx, hooks.PointSessionPostReview, &hooks.SessionPostReviewEvent{
            Summary: &summary, ReviewPath: reviewSummary.Path, Blocked: summary.ReviewStatus == "blocked",
        })
    }
}
```

### Job struct estendido

```go
// internal/runtime/runner.go
type Job struct {
    // ... campos existentes ...

    TaskFileName    string // F3-Claude: necessário para resolver memory.ReadTask
    MCPNested       bool   // F2-Claude
    AutoReview      bool   // F5-Claude
    MemoryLimits    memory.Limits  // F3-Claude
    DisableHooks    bool   // F3-Claude (debug)
    NoNormalize     bool   // F2-Claude (debug)
}
```

### Lógica de normalização cross-tool

`normalize.go::BuildNormalizedToolCall`:
1. Lookup `aliases[driverID][rawName]` → se hit, `NormalizedName = result`. Se miss, `NormalizedName = rawName`.
2. Decodificar `rawInput` para `map[string]any`.
3. Aplicar `input_mappings[driverID][rawName]` renomeando chaves quando hit. Preservar valores.
4. Re-encodar como `NormalizedInput`.
5. Retornar struct com `RawName`/`NormalizedName`/`RawInput`/`NormalizedInput` separados.

**Garantia**: `RawInput` nunca é mutado — debug `--no-normalize` recupera comportamento idêntico ao pré-F2.

### Auto-review prompt

```go
func buildReviewPrompt(skillBody, gitDiff string) string {
    return fmt.Sprintf(
        "%s\n\n## Diff a Revisar\n\n```diff\n%s\n```\n\n## Instrução\n"+
        "Revise o diff acima conforme as regras da skill. Reporte issues por severidade.\n"+
        "Para issues `hard`/`CRÍTICO`/`BLOQUEADO`, prefixar a linha com [HARD].\n",
        skillBody, gitDiff,
    )
}

func parseReviewStatus(reviewOutput string) string {
    if strings.Contains(reviewOutput, "[HARD]") ||
       strings.Contains(reviewOutput, "BLOQUEADO") ||
       strings.Contains(reviewOutput, "CRÍTICO") {
        return "blocked"
    }
    return "ok"
}
```

---

## Pontos de Integração

| Subsistema | Integração | Direção |
|---|---|---|
| `internal/runtime/client/client.go` | Read-only — usado por mcpserver para spawnar child sessions | Leitor |
| `internal/agents/registry.go` (ADR-011) | Resolução de `agent_name` em `run_agent` tool | Consumidor |
| `internal/specdrift/` | Sem alteração; PRD/TechSpec consomem via cabeçalho `spec-hash-prd` | Externo |
| `internal/metrics/` | Hook `token_budget.go` chama `metrics.CheckBudget` | Consumidor |
| `internal/telemetry/` | Append entries `claude.cache_read=N`, `claude.thinking=N` quando opt-in | Produtor |
| `internal/parity/` | +2 invariantes (normalização cross-tool; MCP nested depth) | Produtor |
| `internal/evidence/evidence.go` | +seção "Métricas Claude-2026" (opcional) | Produtor |
| `.agents/skills/review/SKILL.md` | Carregada por F5 como prompt do auto-review | Leitor |
| `.agents/normalization-rules.yaml` | Lida via `go:embed` (default) + override de projeto | Leitor |
| `.claude/hooks/*.sh` | **Não modificados** — coexistem com hooks Go | — |
| `.claude/agents/*.md` | Resolvidos via registry quando MCP `run_agent` invocado | Leitor |

Nenhum endpoint HTTP. Nenhum serviço externo. Tudo local ao processo do harness.

---

## Abordagem de Testes

### Testes Unitários

#### F2-Claude — `internal/runtime/mcpserver/server_test.go`

| Caso | Setup | Expectativa |
|---|---|---|
| T-MCP-01 `TestServerExposesRunAgentOnly` | server.Serve com mock registry | tools list retorna apenas `run_agent` |
| T-MCP-02 `TestRunAgentResolvesViaRegistry` | mock registry com agent "reviewer" | invocação resolve spec; spawn iniciado |
| T-MCP-03 `TestRunAgentUnknownAgentReturnsError` | agent_name "ghost" não no registry | erro MCP tipado; sessão não spawna |
| T-MCP-04 `TestRunAgentDepthLimit` | ctx com depth=3 e maxDepth=3 | erro "depth limit reached" |
| T-MCP-05 `TestRunAgentTimeoutHonored` | timeout=1s; child trava por 5s | erro de timeout; child cancelado |
| T-MCP-06 `TestRunAgentEvidenceDirReturned` | spawn bem-sucedido | output contém `evidence_dir` válido |

#### F2-Claude — `internal/runtime/events/normalize_test.go`

| Caso | Setup | Expectativa |
|---|---|---|
| T-NORM-01 `TestNormalizeClaudeBash` | driverID="claude", rawName="bash" | NormalizedName="bash"; RawName="bash" |
| T-NORM-02 `TestNormalizeCodexShellToBash` | driverID="codex", rawName="shell" | NormalizedName="bash"; RawName="shell" |
| T-NORM-03 `TestNormalizeCodexSearchQueryToWebSearch` | driverID="codex", rawName="search_query" | NormalizedName="web_search" |
| T-NORM-04 `TestNormalizationPreservesRawInput` | qualquer | `RawInput == input antes` byte-identical |
| T-NORM-05 `TestNormalizationCanonicalizesFields` | driverID="cursor", rawInput com `startLineNumberBaseOne` | `NormalizedInput` tem `start_line` |
| T-NORM-06 `TestUnknownDriverFallsBack` | driverID="foo", rawName="bar" | NormalizedName="bar" (passthrough); sem erro |
| T-NORM-07 `TestProjectOverrideWinsOverEmbedded` | workspace com `.agents/normalization-rules.yaml` | regras locais sobrescrevem default |

#### F3-Claude — `internal/runtime/memory/store_test.go`

| Caso | Setup | Expectativa |
|---|---|---|
| T-MEM-01 `TestReadWriteWorkflowRoundTrip` | t.TempDir(); escrever 50 linhas | LineCount=50; NeedsCompaction=false |
| T-MEM-02 `TestNeedsCompactionLineLimit` | escrever 151 linhas | NeedsCompaction=true |
| T-MEM-03 `TestNeedsCompactionByteLimit` | escrever 50 linhas com 300 bytes cada (15000 > 12288) | NeedsCompaction=true |
| T-MEM-04 `TestReadAbsentReturnsExistsFalse` | dir sem MEMORY.md | Document{Exists:false}; err=nil |
| T-MEM-05 `TestWriteModeAppend` | escrever existente em append | conteúdo concatenado; sem trim |

#### F3-Claude — `internal/runtime/hooks/dispatcher_test.go`

| Caso | Setup | Expectativa |
|---|---|---|
| T-HOOK-01 `TestDispatcherRespectsRegistrationOrder` | registrar h1, h2, h3 em "prompt.pre_build" | Run() chamado na ordem h1→h2→h3 |
| T-HOOK-02 `TestDispatcherAbortsOnFirstError` | h2 retorna erro | h3 não chamado; Dispatch retorna erro |
| T-HOOK-03 `TestDispatcherUnknownPointNoop` | Dispatch em "nonexistent.point" | nil; sem panic |
| T-HOOK-04 `TestGovernanceHookBlocksMissingAGENTS` | t.TempDir() sem AGENTS.md | Run retorna erro; mensagem clara |

#### F4-Claude — `internal/runtime/events/convert_test.go` (estender existente)

| Caso | Setup | Expectativa |
|---|---|---|
| T-CONV-CR-01 `TestExtractCacheReadTokens` | mock acp.SessionUpdate com `usage.cache_read_input_tokens=500` | Summary.CacheReadTokens=500 |
| T-CONV-CR-02 `TestExtractThinkingTokens` | mock com reasoning content block 100 tokens | Summary.ThinkingTokens=100 |
| T-CONV-CR-03 `TestMissingFieldsRemainZero` | payload sem usage block | todos campos 0; sem erro |

#### F5-Claude — `internal/runtime/runner_test.go` (estender)

| Caso | Setup | Expectativa |
|---|---|---|
| T-REV-01 `TestAutoReviewSkipsWhenFlagFalse` | --auto-review=false | runAutoReview não chamado |
| T-REV-02 `TestAutoReviewBlocksOnHardIssue` | review output contém "[HARD] eval() detected" | Summary.ReviewStatus="blocked" |
| T-REV-03 `TestAutoReviewOkWhenNoHardMarkers` | review output sem marcadores hard | Summary.ReviewStatus="ok" |
| T-REV-04 `TestAutoReviewDoesntRecurse` | session de review tem --auto-review=true | runAutoReview pula em sessão de review |

### Testes de Integração

**`tests/integration/claude_2026_e2e_test.go`** (novo, ~300 LoC):

- T-INT-01 — `TestClaudeMCPNestedAgentE2E`: roda task simulada onde Claude (mock SDK) invoca `run_agent("reviewer", "<...>")`; verifica `events.jsonl` do parent tem evento `nested_agent` e `events.jsonl` do child existe em `evidence/<task>/nested/`.
- T-INT-02 — `TestClaudeNormalizationCrossToolE2E`: roda mesma operação shell em Claude e em Codex; verifica `events.jsonl` ambos têm `normalized_name="bash"` apesar de `raw_name` divergir.
- T-INT-03 — `TestClaudeMemoryCompactionDirectiveE2E`: pre-escreve `.specs/<prd>/memory/MEMORY.md` com 151 linhas; verifica prompt enviado a Claude contém "compact the flagged memory files".
- T-INT-04 — `TestClaudeGovernanceHookBlocksWithoutAGENTSE2E`: sem AGENTS.md em workdir; verifica sessão aborta antes de `c.Open` com mensagem clara.
- T-INT-05 — `TestClaudeAutoReviewBlocksOnHardIssueE2E`: --auto-review=true; mock skill review retorna "[HARD]"; verifica `Summary.ReviewStatus="blocked"` e `execution_report.md` tem seção "Review Block".

### Testes de Parity (ADR-008)

`internal/parity/invariants_test.go` ganha:

- **INV-30** (novo): `tool_calls_normalized_name_invariant` — para mesma operação semântica em Claude e Codex, `normalized_name` em `events.jsonl` deve ser idêntico.
- **INV-31** (novo): `mcp_nested_depth_never_exceeds_max` — em qualquer `events.jsonl` com kind `nested_agent`, `depth` ≤ `AISPEC_MAX_AGENT_DEPTH` (default 3).

Todos os 29 invariantes existentes devem continuar passando — regressão hard.

### Testes E2E manuais (smoke)

Documentar em `evidence/<task>/smoke.md` por wave:

```bash
# F2-Claude smoke
ai-spec task-loop --tool claude --runtime acp --mcp-nested .specs/prd-foo
# Verificar: events.jsonl tem normalized_name; tool_calls.md renderiza nome normalizado.

# F3-Claude smoke
mkdir -p .specs/prd-foo/memory
yes "linha repetida" | head -200 > .specs/prd-foo/memory/MEMORY.md
ai-spec task-loop --tool claude --runtime acp .specs/prd-foo
# Verificar: execution_report.md cita "Memory Compaction Requested: true".

# F4-Claude smoke
ai-spec task-loop --tool claude --runtime acp .specs/prd-foo
# Verificar: execution_report.md tem seção "Métricas Claude-2026" com cache_read > 0
# (assumindo prompt caching está ativo no claude-agent-acp).

# F5-Claude smoke
ai-spec task-loop --tool claude --runtime acp --auto-review .specs/prd-com-bug
# Verificar: evidence/<task>/review.md existe; Summary.ReviewStatus em log.
```

---

## Sequenciamento de Desenvolvimento

**Wave F2-Claude** (~1 sprint; 5 tasks paralelizáveis após task-1):

1. `task-1.0-internal-runtime-mcpserver-skeleton.md` — Server interface + RunAgentInput/Output structs; sem lógica de spawn ainda. (blocking)
2. `task-2.0-mcp-server-spawn-child-runner.md` — engine.go spawnando ACPRunner com contexto serializado.
3. `task-3.0-internal-runtime-events-normalize.md` — normalize.go + `.agents/normalization-rules.yaml` embedded.
4. `task-4.0-runner-integrate-mcp-and-normalize.md` — runner.go usa mcpserver e normalize quando flags ativas.
5. `task-5.0-task-loop-flags-mcp-nested-no-normalize.md` — `--mcp-nested`, `--no-normalize` em task_loop.go.
6. `task-6.0-parity-invariants-INV-30-INV-31.md` — adicionar invariantes em parity.
7. `task-7.0-claude-md-runtime-capabilities.md` — atualizar CLAUDE.md raiz com §"Runtime Capabilities".
8. `task-8.0-smoke-test-claude-acp-mcp-normalize.md` — smoke test E2E F2-Claude.

**Wave F3-Claude** (~1.5 sprints; 6 tasks):

9. `task-9.0-internal-runtime-memory-store.md` — memory/store.go + tipos.
10. `task-10.0-internal-runtime-hooks-dispatcher.md` — hooks/dispatcher.go + 6 pontos canônicos.
11. `task-11.0-hooks-governance-token-budget-migration.md` — hooks Go espelhando shell.
12. `task-12.0-runner-integrate-memory-and-hooks.md` — runner.go consome memory + hooks.
13. `task-13.0-task-loop-flags-memory-disable-hooks.md` — flags `--memory-*-limit-*`, `--disable-hooks`.
14. `task-14.0-smoke-test-claude-memory-hooks.md` — smoke test F3-Claude.

**Wave F4-Claude** (~3 dias; 3 tasks):

15. `task-15.0-events-convert-claude-2026-fields.md` — extrair cache/thinking.
16. `task-16.0-evidence-claude-2026-section.md` — estender evidence.go.
17. `task-17.0-telemetry-claude-cache-thinking.md` — append em `.agents/telemetry.log`.

**Wave F5-Claude** (~3 dias; 3 tasks):

18. `task-18.0-runner-auto-review-spawn.md` — runAutoReview em runner.go.
19. `task-19.0-task-loop-flag-auto-review.md` — flag em task_loop.go.
20. `task-20.0-smoke-test-claude-auto-review.md` — smoke test F5.

### Dependências Técnicas

- Biblioteca MCP-SDK Go (a confirmar maturidade em 2026-Q2). Fallback: implementar wire MCP em `internal/runtime/mcpserver/wire/` com schema oficial — ~150 LoC adicionais.
- Nenhuma outra nova dependência em `go.mod`.
- `coder/acp-go-sdk v0.6.3` permanece pinado (ADR-009).
- `claude-agent-acp@0.1.0` permanece pinado (ADR-009).

---

## Monitoramento e Observabilidade

- `events.jsonl` continua sendo a fonte canônica forense — ganha campo `normalized_name` ao lado de `raw_name` em F2.
- `tool_calls.md` ganha rendering com nome normalizado em F2.
- `execution_report.md` ganha seção "Métricas Claude-2026" opcional em F4 e seção "Review Block" condicional em F5.
- Telemetria (`internal/telemetry/`): novas entries `claude.cache_read=N`, `claude.cache_creation=N`, `claude.thinking=N`, `claude.normalized_tools=N` quando `GOVERNANCE_TELEMETRY=1`.
- Sem endpoints Prometheus / OTel para esta feature (ADR-006 §"Telemetria local opt-in").

---

## Considerações Técnicas

### Decisões Chave (inline; ver ADR-014 para racional completo)

| Decisão | Escolha | Justificativa |
|---|---|---|
| MCP wire | stdio | Paridade Compozy; menor overhead que SSE/WebSocket |
| MCP transport scope | server-only (harness não consome MCP externo) | Reduz surface; sem hospedar MCP de terceiros |
| Hooks dispatcher | sequencial, abort-on-first-error | Mais simples; alinhado com `kernel.Dispatch` de Compozy |
| Hooks transporte | in-process Go (não JSON-RPC) | Paridade Compozy; tipado; sem fork overhead |
| Shell hooks legacy | mantidos | Modo interativo Claude Code depende deles |
| Memória compactação | prompt-driven (não code) | Paridade Compozy; mais barato e auditável |
| Memória limites | configuráveis via flag | Defaults idênticos a Compozy; flexibilidade sem mudar código |
| Tool normalize | tabela YAML embedded + override de projeto | Auditável; sem rebuild para ajustar mapeamento por projeto |
| Auto-review provider | skill local (não GitHub/CodeRabbit) | R-GOV-001 §"Não introduzir dependência externa sem justificativa" |
| Auto-review recursão | bloqueada em sessão de review | Evita loop infinito |
| Claude em ValidTools | NÃO adicionar | Decisão ADR-014 §D-07; documentada em `wrapper_test.go:213-214` |

### Riscos Conhecidos

| Risco | Impacto | Mitigação |
|---|---|---|
| MCP-SDK Go interface muda em 2026-Q3 | Recompilação necessária | Interface interna isola dependência; vendor inicial se necessário |
| Shell hooks vs Go hooks confusão entre contribuidores | DX degradado | Documentar precedência em `CLAUDE.md` e em cada arquivo Go citando equivalente shell |
| Memória 2-tier duplica com auto-memory de Claude Code | Inconsistência | Documentar precedência: harness vence quando `.specs/<prd>/memory/` existe |
| MCP nested DoS por recursão | OOM/CPU | maxDepth=3 hard-coded; timeout obrigatório; trace `parent_session_id` |
| Auto-review dobra custo tokens | Custo operacional | Opt-in; documentar trade-off em `CLAUDE.md`; futuro cache por SHA diff |
| Normalização mascara bug upstream | Debug difícil | `raw_name` lado a lado; flag `--no-normalize` para diagnóstico |
| Compactação prompt-driven pode falhar silenciosamente | Memória cresce sem limite | Tarefa de validação: re-checar `NeedsCompaction` em `session.post_end`; se persistente, anexar warning ao `execution_report.md` |

### Conformidade com Padrões

- **R-GOV-001** — decisões documentadas neste spec, em ADR-014 e no PRD. Precedência respeitada (governança transversal > security > implementação).
- **R-DDD-001** — `Spec` continua construído via factory; `Server`, `Store`, `Dispatcher` expostos via interface; sem instanciação por literal externo.
- **ADR-008** (paridade) — +2 invariantes (INV-30, INV-31); todos os 29 existentes preservados.
- **ADR-006** (telemetria) — campos novos via append no log existente; opt-in via `GOVERNANCE_TELEMETRY=1` preservado.
- **ADR-009** (pinning ACP-SDK) — sem alteração em `coder/acp-go-sdk v0.6.3`.
- Convenções: erros em PT-BR; `fmt.Errorf("contexto: %w", err)`; `t.TempDir()` em testes; cobertura ≥ 70% global, ≥ 80% em subpacotes novos.

### Arquivos Relevantes e Dependentes

| Arquivo | Mudança | Wave |
|---|---|---|
| `internal/runtime/runner.go` | Integração com hooks/memory/mcp/auto-review; ~120 LoC de mudanças cirúrgicas | F2/F3/F5 |
| `internal/runtime/mcpserver/server.go` | **Novo** ~150 LoC | F2 |
| `internal/runtime/mcpserver/engine.go` | **Novo** ~80 LoC | F2 |
| `internal/runtime/mcpserver/*_test.go` | **Novo** ~250 LoC | F2 |
| `internal/runtime/events/normalize.go` | **Novo** ~120 LoC | F2 |
| `internal/runtime/events/normalize_test.go` | **Novo** ~180 LoC | F2 |
| `internal/runtime/events/convert.go` | Modificado: extrair cache/thinking; ~40 LoC | F4 |
| `internal/runtime/events/convert_test.go` | Modificado: 3 casos novos | F4 |
| `internal/runtime/memory/store.go` | **Novo** ~150 LoC | F3 |
| `internal/runtime/memory/store_test.go` | **Novo** ~200 LoC | F3 |
| `internal/runtime/hooks/dispatcher.go` | **Novo** ~100 LoC | F3 |
| `internal/runtime/hooks/governance.go` | **Novo** ~60 LoC | F3 |
| `internal/runtime/hooks/token_budget.go` | **Novo** ~60 LoC | F3 |
| `internal/runtime/hooks/memory_persist.go` | **Novo** ~50 LoC | F3 |
| `internal/runtime/hooks/*_test.go` | **Novo** ~250 LoC | F3 |
| `internal/evidence/evidence.go` | Modificado: seção opcional Métricas Claude-2026 | F4 |
| `internal/telemetry/telemetry.go` | Modificado: 4 campos novos no log | F4 |
| `internal/parity/invariants.go` | Modificado: +INV-30, +INV-31 | F2 |
| `cmd/ai_spec_harness/task_loop.go` | Modificado: 7 flags novas; propagar via Job | F2/F3/F5 |
| `cmd/ai_spec_harness/task_loop_test.go` | Modificado: 4 casos novos | F2/F3/F5 |
| `tests/integration/claude_2026_e2e_test.go` | **Novo** ~300 LoC | F2/F3/F4/F5 |
| `.agents/normalization-rules.yaml` | **Novo** ~30 linhas YAML | F2 |
| `CLAUDE.md` | Modificado: §"Runtime Capabilities" | F2 |
| `wrapper.go` / `wrapper_test.go` | **Não alterados** (Claude continua fora de ValidTools — decisão deliberada) | — |

---

## Rollout

**Feature flags por wave (default off em F2/F3; on em F4/F5 onde aplicável):**

- F2-Claude: `--mcp-nested` default `false`; `--no-normalize` default `false` (normalização sempre ativa).
- F3-Claude: limites `--memory-*` default = Compozy defaults (150/12KB workflow, 200/16KB task); hooks default ativos; `--disable-hooks` default `false`.
- F4-Claude: campos sempre extraídos quando presentes no payload; seção do `execution_report.md` é opcional (ausência não bloqueia).
- F5-Claude: `--auto-review` default `false`.

**Cronograma sugerido** (3 sprints total; assume velocidade do harness atual):

| Sprint | Wave | Tasks | Critério de saída |
|---|---|---|---|
| 1 | F2-Claude | T-1..T-8 | Smoke F2 verde; INV-30/INV-31 implementados |
| 2 | F3-Claude | T-9..T-14 | Smoke F3 verde; governance hook funcional |
| 3 (½) | F4-Claude | T-15..T-17 | Smoke F4 verde; telemetria mostra cache>0 |
| 3 (½) | F5-Claude | T-18..T-20 | Smoke F5 verde; review bloqueia em `eval()` |

**Procedimento de release:**

1. Mergear cada wave em branch separada (`feat/claude-2026-f2`, `feat/claude-2026-f3`, ...).
2. Após cada wave: rodar `make test && make integration && make parity` — todos verdes.
3. Atualizar CHANGELOG.md por wave (skill `semantic-commit` automatiza).
4. Documentação `CLAUDE.md` atualizada com cada wave entregue.
5. Release tag `vX.Y.Z` ao final de F5-Claude (skill `github-release-publication-flow`).
6. Rollback: cada wave é additive; flag `--disable-hooks`/`--no-normalize` recuperam comportamento pré-wave. Não há migração breaking de dados.

---

## Verificação Final (pré-merge de cada wave)

Comandos obrigatórios:

```bash
# Lint + format
make lint && make fmt-check

# Cobertura
make test  # cobertura global ≥ 70%
go test ./internal/runtime/mcpserver/...  -coverprofile=cov.out && go tool cover -func=cov.out | tail -1
# Esperado: ≥ 80% nos subpacotes novos

# Parity (ADR-008)
make parity  # 29 + 2 invariantes verdes

# Integration E2E
make integration  # T-INT-01..T-INT-05 verdes

# Spec-drift (governança PRD/TechSpec)
ai-spec check-spec-drift  # após edits, rodar --sync e re-commitar

# Smoke manual da wave (ver §"Testes E2E manuais")
```

Caso qualquer comando falhe, **não mergear**. Investigar root cause antes de criar PR.
