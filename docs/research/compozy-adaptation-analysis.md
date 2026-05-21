# Análise de Adaptação ao Padrão Compozy

> **Status**: Pesquisa concluída — base para PRD `agent-registry-declarativo`
> **Data**: 2026-05-21
> **Fonte primária (Compozy)**: leitura via `gh` do repositório [`compozy/compozy`](https://github.com/compozy/compozy) (branch `main` em 2026-05-21)
> **Fonte primária (harness)**: árvore atual de `/Users/jailtonjunior/Git/orchestrator` na branch `feat/acp-runtime-claude`

---

## Sumário Executivo

O `compozy/compozy` e o `ai-spec-harness` compartilham o **mesmo stack-base**: Go + ACP via `coder/acp-go-sdk`, com subprocess local para cada provedor LLM. A escolha estratégica é, portanto, **arquitetural**, não tecnológica: ambos projetos resolvem o mesmo problema (orquestrar agentes de IA via ACP) com filosofias distintas. Compozy modela tudo como **artefato declarativo** (`AGENT.md`, `mcp.json`, `MEMORY.md`) descoberto em runtime; o harness modela como **código compilado** (`specs.Claude()` hardcoded, profiles via flags CLI).

Comparando dimensão a dimensão, o Compozy é mais maduro em **7 frentes** (agent registry declarativo, composição dinâmica de system prompt, MCP integration, memory layer hierárquico, normalização de tool input, hook system, runtime config com retries) e o harness é mais maduro em **2 frentes** (persistência forense via `events.jsonl` + `tool_calls.md` + `execution_report.md`, e watchdog de inatividade com `CancelCause`). A divergência mais determinante é a primeira: sem **Agent Registry declarativo** (AGENT.md), todas as outras camadas (MCP, memory, hooks) não têm onde se acoplar.

Este relatório recomenda **iniciar a adaptação pelo Agent Registry** (Fase 1 do roadmap), preservando os pontos fortes do harness (persistência forense + watchdog) e migrando o conceito de `Spec` hardcoded para `ResolvedAgent` descoberto de `~/.ai-harness/agents/` e `.ai-harness/agents/`. Fases subsequentes (MCP, Memory, Hooks, Multi-IDE via ACP) serão PRDs independentes que dependem do registry como ponto de acoplamento.

---

## Achados por Dimensão

### 1. Agent Registry Declarativo (Compozy ✅ | harness ❌)

**Compozy** — `internal/core/agents/agents.go`:
```go
const (
    agentDirName   = "agents"
    agentFileName  = "AGENT.md"
    agentMCPConfig = "mcp.json"
)

type ResolvedAgent struct {
    Name     string
    Metadata Metadata
    Runtime  RuntimeDefaults
    Prompt   string
    MCP      *MCPConfig
}

type RuntimeDefaults struct {
    IDE             string
    Model           string
    ReasoningEffort string
    AccessMode      string
}
```

Agentes são **descobertos recursivamente** em `~/.compozy/agents/<name>/AGENT.md` (global) e `.compozy/agents/<name>/AGENT.md` (workspace). Workspace prevalece sobre global em caso de colisão.

**ai-spec-harness** — `internal/runtime/specs/claude.go:24-42`:
```go
func Claude() Spec {
    return newSpec(
        "claude",
        "Claude (ACP)",
        "claude-agent-acp",
        nil,
        []FallbackLauncher{{
            Command:   "npx",
            FixedArgs: []string{"--yes", "@agentclientprotocol/claude-agent-acp@0.1.0"},
        }},
        "--bypass-permissions",
    )
}
```

Não há descoberta dinâmica nem entidade `Agent` separada de `Spec`. Todo agente novo exige código Go.

**Impacto**: bloqueia ergonomia, multi-agente, e qualquer composição declarativa downstream.

---

### 2. Composição Dinâmica de System Prompt (Compozy ✅ | harness ⚠️ parcial)

**Compozy** — `internal/core/agents/execution.go`:
```go
func (c *ExecutionContext) SystemPrompt(baseSystemPrompt string) string {
    sections := []string{
        baseSystemPrompt,
        buildAgentMetadataBlock(c.Agent),
        buildAvailableAgentsBlock(c.Agent.Name, c.Catalog.Agents),
        c.Agent.Prompt,
    }
    return strings.Join(sections, "\n\n")
}
```

System prompt é montado em runtime injetando: (a) prompt base, (b) metadata do agente selecionado, (c) catálogo dos agentes disponíveis (permite hand-off), (d) corpo do `AGENT.md`.

**ai-spec-harness** — `internal/taskloop/agent.go:62-96`:
```go
//go:embed executor_template.tmpl
var defaultExecutorTemplate string

func BuildPromptContext(prdFolder, workDir string, fsys fs.FileSystem) PromptContext {
    // Extrai seções "Arquitetura", detecta linguagens, frameworks
    // Retorna Architecture e References
}
```

Template é embed estático. Há extração de contexto (`PromptContext`), mas sem composição dinâmica de metadata de agente nem catálogo.

**Impacto**: harness não permite que um agente sugira hand-off para outro nem expõe metadata de capacidades.

---

### 3. MCP Integration (Compozy ✅ | harness ❌)

**Compozy** — `internal/core/model/mcp.go`:
```go
type MCPServer struct { Stdio *MCPServerStdio }

type MCPServerStdio struct {
    Name    string
    Command string
    Args    []string
    Env     map[string]string
}
```

Cada `AGENT.md` pode declarar `mcp.json` adjacente. MCP servers são iniciados como subprocessos stdio quando o agente é resolvido. Isso amplia as tools disponíveis ao modelo sem código novo.

**ai-spec-harness**: ausente. Tools disponíveis ao modelo são fixadas pelo CLI subjacente (Claude Code, Codex CLI, etc.).

**Impacto**: harness não tem extensibilidade de tools por configuração.

---

### 4. Memory Layer Hierárquico (Compozy ✅ memory-first | harness ✅ forensic-first)

**Compozy** — `internal/core/memory/store.go`:
```go
const (
    WorkflowFileName   = "MEMORY.md"
    workflowLineLimit  = 150
    workflowByteLimit  = 12 * 1024
    taskLineLimit      = 200
    taskByteLimit      = 16 * 1024
)

func WriteDocument(tasksDir, taskFileName, content string, mode WriteMode) (Document, int, error) {
    // compactação atômica se exceder limites
}
```

Memória workflow + task-local em Markdown com **compactação automática** quando excede limites. Persistência granular intencional para o modelo reler.

**ai-spec-harness** — `internal/runtime/persistence/session.go:13-63`:
```go
type SessionPersistence struct {
    jsonl       *JSONLWriter
    evidenceDir string
    fsys        fs.FileSystem
}

// AppendEvent() → events.jsonl
// WriteToolCalls() → tool_calls.md
// EnrichReport() → execution_report.md
```

Foco em **forense pós-mortem**: tudo é registrado em `events.jsonl` linha-a-linha + Markdown resumido. Sem compactação, sem hierarquia workflow/task, sem intenção de releitura pelo modelo.

**Comparação**: filosofias complementares — não conflitantes. Harness pode adicionar memory-layer mantendo persistência forense.

**Impacto**: harness não tem mecanismo para o agente acumular conhecimento entre runs.

---

### 5. Tool Input Normalization (Compozy ✅ | harness ⚠️ raw-only)

**Compozy** — `internal/core/agent/tool_call_input.go`:
```go
func normalizeToolInputByName(name string, title string, rawInput any, raw map[string]any, locations []acp.ToolCallLocation) map[string]any {
    switch name {
    case toolNameBash:
        normalizeBashToolInput(normalized, rawInput, raw)
    case toolNameGrep:
        normalizeGrepToolInput(normalized, rawInput, raw)
    case toolNameRead, toolNameWrite, toolNameEdit, toolNameDelete:
        normalizeFileToolInput(normalized, raw, locations)
    case toolNameWebFetch, toolNameOpenURL:
        normalizeOpenURLInput(normalized, raw)
    case toolNameWebSearch:
        mergeWebSearchInput(normalized, raw, title)
    }
    return normalized
}
```

Inputs heterogêneos de tools de diferentes CLIs são reduzidos a um formato canônico — facilita logs, comparações cross-CLI, persistência.

**ai-spec-harness** — `internal/runtime/events/convert.go:147-183`:
```go
update.ToolCall != nil → NewToolCallStart(now, id, name, inputStr, raw)
// Apenas serializa RawInput sem normalizar
```

Preserva fidelidade absoluta (bom para forense) mas dificulta análise cross-CLI.

**Impacto**: divergência sutil entre payloads de Claude vs. Codex para mesma operação (ex: `Read`) é opaca ao consumidor.

---

### 6. Hook System (Compozy ✅ | harness ❌)

**Compozy** — `internal/core/prompt/common.go`:
```go
func Build(p BatchParams) (string, error) {
    p, err := dispatchPromptPreBuild(p)
    if err != nil { return "", err }
    rendered := buildCodeReviewPrompt(p)
    return dispatchPromptPostBuild(p, rendered)
}
```

Hooks `prompt.pre_build`, `prompt.post_build`, `prompt.pre_system` são pontos de extensão para plugins modificarem prompt ou estado.

**ai-spec-harness**: ausente. Toda customização de prompt exige fork ou modificação direta dos templates embed.

**Impacto**: harness é menos extensível por terceiros.

---

### 7. Runtime Config com Retries/Backoff (Compozy ✅ | harness ⚠️ watchdog-only)

**Compozy** — `internal/core/model/runtime_config.go`:
```go
type RuntimeConfig struct {
    Timeout                time.Duration
    MaxRetries             int
    RetryBackoffMultiplier float64
}

func (cfg *RuntimeConfig) ApplyDefaults() {
    if cfg.RetryBackoffMultiplier <= 0 {
        cfg.RetryBackoffMultiplier = 1.5
    }
}
```

Declarativo no `AGENT.md`. Falhas transitórias têm retry/backoff configurável.

**ai-spec-harness** — `internal/runtime/watchdog.go:18-110`:
```go
type ActivityWatchdog struct {
    timeout  events.ActivityTimeout
    cancel   context.CancelCauseFunc
    lastSeen atomic.Int64
}
```

Watchdog detecta inatividade e cancela com `CancelCause`. **Não há retry** — uma falha mata a sessão.

**Impacto**: harness é menos resiliente a falhas transitórias de provider (rate limit, 5xx).

---

### 8. Multi-IDE Real via ACP (Compozy ✅ | harness ⚠️ Claude-only)

**Compozy** suporta **Codex, Claude, Droid, Cursor, OpenCode, Pi, Gemini, Copilot** todos via ACP (cada subprocess implementa o protocolo). Path: `internal/core/agent/client.go` com `ClientConfig.IDE` controlando qual subprocess subir.

**ai-spec-harness**:
- **Claude**: ACP nativo via `claude-agent-acp`
- **Codex/Gemini/Copilot**: CLI invokers (não-ACP) em `internal/taskloop/agent.go:200+`

```go
type AgentInvoker interface {
    Invoke(ctx context.Context, prompt, workDir, model string) (stdout, stderr string, exitCode int, err error)
    BinaryName() string
}
```

Bons abstratamente, mas perdem streaming e tool call observability dos demais providers.

**Impacto**: paridade observacional é hoje exclusiva de Claude. Outros providers são caixa-preta.

---

### 9. Streaming/Backpressure (Compozy ✅ tracked | harness ⚠️ untracked)

**Compozy** — `internal/core/agent/session.go`:
```go
type sessionImpl struct {
    updates        chan model.SessionUpdate
    slowPublishes  atomic.Uint64
    droppedUpdates atomic.Uint64
}

type Session interface {
    Updates() <-chan model.SessionUpdate
    SlowPublishes() uint64
    DroppedUpdates() uint64
}
```

Quando o consumidor é lento, drops/slow publishes são contabilizados explicitamente (5s timeout).

**ai-spec-harness** — `internal/runtime/client/client.go:269-271`:
```go
func (c *acpClient) Updates() <-chan events.Event {
    return c.eventCh  // canal size 64
}
```

Sem métricas de drop/slow. Bloqueio do consumidor faria o producer travar (caso patológico não observado em produção).

**Impacto**: telemetria de saúde do stream é fraca.

---

### 10. Persistência Forense (Compozy ❌ | harness ✅) — **vantagem do harness**

`internal/runtime/persistence/` produz três artefatos por sessão:
- `events.jsonl` — append-only com cada evento ACP
- `tool_calls.md` — tabela Markdown agregada por tool
- `execution_report.md` — Summary com metadados

Compozy não tem equivalente — opera memory-first com Markdown vivo.

**Posição**: **preservar como diferencial**. Adoção de Agent Registry não deve sacrificar esta camada.

---

## Gap Map Expandido

Legenda: 🟢 implementado · 🟡 parcial · 🔴 ausente · ⭐ vantagem a preservar

| # | Dimensão | Compozy | ai-spec-harness | Esforço | Risco | Fase |
|---|---|---|---|---|---|---|
| 1 | Agent Registry declarativo | 🟢 | 🔴 | Médio | Baixo | **F1** |
| 2 | System Prompt Composition dinâmica | 🟢 | 🟡 | Baixo | Baixo | **F1** |
| 3 | MCP Integration por agente | 🟢 | 🔴 | Médio | Médio | F2 |
| 4 | Memory Layer hierárquico (workflow/task) | 🟢 | 🟡 (forense) | Médio | Baixo | F3 |
| 5 | Tool Input Normalization | 🟢 | 🟡 (raw only) | Médio | Médio | F3 |
| 6 | Hook System (pre/post build) | 🟢 | 🔴 | Médio | Médio | F4 |
| 7 | Runtime Retry/Backoff | 🟢 | 🟡 (watchdog) | Baixo | Baixo | F4 |
| 8 | Streaming backpressure metrics | 🟢 | 🟡 (untracked) | Baixo | Baixo | F4 |
| 9 | Multi-IDE real via ACP | 🟢 | 🟡 (Claude apenas) | **Alto** | **Alto** | F5 |
| 10 | Persistência Forense (jsonl + .md) | 🔴 | 🟢 ⭐ | — | — | manter |
| 11 | Watchdog de Inatividade (CancelCause) | 🟡 | 🟢 ⭐ | — | — | manter |

**Critério de esforço**: Baixo = ≤1 sprint, Médio = 1-2 sprints, Alto = ≥2 sprints com SDK research.
**Critério de risco**: probabilidade de quebrar invariantes ACP/governança existentes.

---

## Conclusão e Justificativa Fase 1

A escolha de **Agent Registry declarativo** como primeiro incremento é estratégica por três razões:

1. **Destrava todas as fases subsequentes**. MCP (F2), Memory hierárquico (F3), Hook System (F4) e Multi-IDE via ACP (F5) precisam de uma entidade `Agent` declarativa onde se acoplar. Sem ela, cada uma dessas frentes inventaria seu próprio mecanismo de configuração.

2. **Risco baixo e reversível**. A entidade `Agent` é additiva — `specs.Claude()` continua existindo e a flag `--tool claude` continua funcionando. A nova flag `--agent <name>` é opt-in. Migração de invariantes ACP/governança não é exigida.

3. **Reuso máximo de código existente**. `internal/skills/frontmatter.go` (parser YAML) e `internal/skills/schema.go` (JSON Schema validation) são reaproveitáveis com pequena generalização. `specs.Launcher` é mantido — `Agent` resolvido produz um `Spec`, não substitui.

A Fase 1 produz um PRD, um TechSpec e um conjunto de tasks executáveis no padrão já estabelecido pelo `tasks/prd-acp-runtime-claude/`. As Fases 2–5 são tratadas como PRDs futuros independentes, conforme seção de Continuidade abaixo.

---

## Próximos PRDs (Continuidade)

> **Estado em 2026-05-21**: Fase 1 entregue como PRD/TechSpec/Tasks em [`tasks/prd-agent-registry-declarativo/`](../../tasks/prd-agent-registry-declarativo/) (ADR-011 em [`tasks/adr/011-agent-registry-declarativo.md`](../../tasks/adr/011-agent-registry-declarativo.md)). As Fases 2–5 abaixo são PRDs **futuros independentes** com escopo bruto e dependências mapeadas. Cada uma é candidata a virar um par PRD+TechSpec próprio quando priorizada.

### F2 — MCP Integration por Agente

**Escopo bruto**
- `mcp.json` adjacente ao `AGENT.md` (`<scope>/agents/<name>/mcp.json`).
- Schema MCP stdio espelhando `internal/core/model/mcp.go` do Compozy: `{name, command, args, env}`.
- Subprocess stdio gerenciado pelo registry quando agente é resolvido.
- Lifecycle: start no `Resolve` (ou no `Run`); kill com grace period idêntico ao do `acpClient`.

**Riscos**
- Coordenação de múltiplos subprocessos stdio em paralelo com o subprocess ACP principal.
- Segurança: comandos externos exigem revisão de `R-SEC-001` (sem shell, args quoted).
- Watchdog precisa contemplar inatividade do MCP subprocess sem afetar `ActivityWatchdog` do ACP principal.

**Dependências**: F1 entregue (Agent Registry).

**Pré-requisito de viabilidade**: estudo de como `coder/acp-go-sdk` interage com MCP servers expostos pelo agente subjacente — Claude já consome MCP nativamente.

### F3 — Memory Layer Hierárquico + Tool Input Normalization

**Escopo bruto — Memory**
- `MEMORY.md` workflow-level (limite 150 linhas / 12KB) e task-level (200 linhas / 16KB).
- Compactação atômica espelhando `internal/core/memory/store.go` do Compozy.
- API `memory.Read/Write` com modos `append|overwrite|patch`.
- Coexistência com persistência forense (`events.jsonl`) — memory-first **não** substitui forense.

**Escopo bruto — Normalization**
- `internal/runtime/events/normalize.go` normalizando tool inputs por nome (`bash`, `grep`, `read`, `write`, `edit`, `web_*`) para formato canônico.
- Mantém `RawInput`/`RawOutput` originais (RF de não regressão de fidelidade forense).
- Habilita comparações cross-CLI quando F5 entregar Codex/Gemini/Copilot em ACP.

**Riscos**
- Memory bloat se compactação for mal calibrada — testes de carga obrigatórios.
- Normalização lossy: cuidado para não esconder divergências entre CLIs (manter raw + normalized separados).

**Dependências**: F1 entregue. Normalization independe de F2; Memory depende parcialmente de F2 se memory passar a alimentar prompts injetados por hooks.

### F4 — Hook System + Runtime Resilience

**Escopo bruto — Hooks**
- Hooks `prompt.pre_build`, `prompt.post_build`, `prompt.pre_system` espelhando `internal/core/prompt/common.go` do Compozy.
- Mecanismo de registro (plugin Go ou config YAML) — decisão de design adiada ao PRD próprio.
- Composição com builders existentes (`BuildPromptContext`, `BuildAgentBlocks`).

**Escopo bruto — Resilience**
- `RuntimeConfig{Timeout, MaxRetries, RetryBackoffMultiplier}` declarável no `AGENT.md` (espelha `internal/core/model/runtime_config.go` do Compozy).
- Retry com backoff exponencial em falhas transitórias de subprocess ACP.
- Métricas de backpressure no canal `acpClient.Updates()`: contadores `slowPublishes` e `droppedUpdates` espelhando `internal/core/agent/session.go` do Compozy.

**Riscos**
- Retry interage com `ActivityWatchdog` — risco de double cancellation; precisa de design coordenado.
- Hooks aumentam surface area; abuso pode quebrar invariantes ACP.

**Dependências**: F1 entregue. Hooks ganham mais valor após F2 (MCP) e F3 (Memory) introduzirem mais pontos onde injetar contexto.

### F5 — Multi-IDE Real via ACP

**Escopo bruto**
- Substituir `codexInvoker`, `geminiInvoker`, `copilotInvoker` (em `internal/taskloop/agent.go`) por SDKs ACP nativos quando disponíveis upstream.
- Generalizar `internal/runtime/specs/` para catalogar Codex/Gemini/Copilot com Launcher e tabela de probe.
- Atualizar mapping `runtime.ide → specs.<Tool>()` para que `--agent` ganhe paridade observacional cross-CLI.

**Riscos**
- **Alto** — depende inteiramente da maturidade dos SDKs ACP por provider. Em 2026-05-21, apenas Claude tem SDK estável (`@agentclientprotocol/claude-agent-acp`).
- Migração quebraria invokers existentes; deprecação faseada por provider exigida.
- Telemetria, persistência forense e watchdog precisam validar paridade semântica (ADR-008 / ADR-010).

**Dependências**: F1 entregue + **estudo de viabilidade por provider** (não bloqueada por F2/F3/F4).

**Pré-requisito**: monitorar releases de SDKs ACP de Codex/Gemini/Copilot; reabrir esta fase quando ≥1 SDK adicional estabilizar.

---

## Apêndice: Tabela de Mapeamento de Fases

| Fase | Status | PRD | Dependências | Risco | Esforço |
|---|---|---|---|---|---|
| F1 — Agent Registry | **PRD pronto** (TechSpec + 10 tasks) | `tasks/prd-agent-registry-declarativo/` | — | Baixo | Médio |
| F2 — MCP Integration | Não iniciado | — | F1 | Médio | Médio |
| F3 — Memory + Tool Normalization | Não iniciado | — | F1 (Memory: + F2) | Baixo | Médio |
| F4 — Hooks + Resilience | Não iniciado | — | F1 (idealmente após F2 e F3) | Médio | Médio |
| F5 — Multi-IDE via ACP | Não iniciado | — | F1 + estudo de viabilidade por provider | **Alto** | **Alto** |

---

## Referências Cruzadas

**Compozy (leitura via gh — branch `main`)**
- `internal/core/agents/agents.go` — descoberta de AGENT.md + mcp.json
- `internal/core/agents/execution.go` — composição de system prompt e precedência
- `internal/core/agent/client.go` — `ClientConfig{IDE, Model, ReasoningEffort, AccessMode}`
- `internal/core/agent/session.go` — `Session` com streaming e backpressure
- `internal/core/agent/tool_call_input.go` — normalização de tool input
- `internal/core/memory/store.go` — workflow + task memory com compactação
- `internal/core/model/runtime_config.go` — retry/backoff config
- `internal/core/model/mcp.go` — MCP server stdio
- `internal/core/prompt/common.go` — hook system de prompt

**ai-spec-harness (estado atual em `feat/acp-runtime-claude`)**
- `internal/runtime/specs/claude.go:24-42` — Spec hardcoded
- `internal/runtime/specs/spec.go:10-37` — value object Spec
- `internal/runtime/specs/launcher.go:13-47` — Launcher binary/npx
- `internal/runtime/probe/probe.go:39-82` — EnsureAvailable + cache
- `internal/runtime/runner.go:59-217` — ACPRunner orquestrador
- `internal/runtime/client/client.go:27-374` — Client ACP + subprocess
- `internal/runtime/events/convert.go:117-197` — conversão SDK → domínio
- `internal/runtime/events/event.go:11-150` — Event tagged union (ADR-010)
- `internal/runtime/persistence/session.go:13-63` — SessionPersistence
- `internal/runtime/watchdog.go:18-110` — ActivityWatchdog
- `internal/skills/frontmatter.go:23-67` — parser YAML frontmatter (reaproveitável)
- `internal/skills/schema.go:47-70` — JSON Schema validation (reaproveitável)
- `internal/taskloop/taskloop.go:20-150` — Service orquestrador
- `internal/taskloop/agent.go:18-250` — AgentInvoker + BuildPromptContext
- `internal/taskloop/profile.go:34-135` — ExecutionProfile + ProfileConfig
- `internal/taskloop/acpinvoker.go:21-100` — adapter ACP → AgentInvoker
- `internal/invocation/invocation.go:9-31` — recursion depth limit
