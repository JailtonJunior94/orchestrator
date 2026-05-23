# Análise de Adaptação ao Padrão Compozy — Foco Claude-CLI 2026

> **Status**: Pesquisa concluída — base para futuro PRD `prd-claude-cli-acp-2026` (F2-Claude até F5-Claude)
> **Data**: 2026-05-21
> **Fonte primária (Compozy)**: leitura via `gh` do repositório [`compozy/compozy`](https://github.com/compozy/compozy) — branch `main` SHA `7f38c445069bd83a8e96bcd925ee1f12fde74435`
> **Fonte primária (harness)**: árvore atual de `/Users/jailtonjunior/Git/orchestrator` na branch `feat/codex-acp-spec` (commit `822ae74` "feat(codex): implementar F1-Codex via ACP nativo (ADR-013)")
> **Pesquisas irmãs**: [`compozy-adaptation-codex-2026.md`](compozy-adaptation-codex-2026.md), [`compozy-adaptation-copilot-2026.md`](compozy-adaptation-copilot-2026.md), [`compozy-adaptation-analysis.md`](compozy-adaptation-analysis.md)
> **Prompt de origem**: [`docs/prompts/compozy-adaptation-research-claude.md`](../prompts/compozy-adaptation-research-claude.md)

---

## Sumário Executivo

Claude já é runtime ACP de primeira classe no harness desde ADR-009: `internal/runtime/specs/claude.go:28-45` declara `Command: "claude-agent-acp"` com fallback `npx --yes @agentclientprotocol/claude-agent-acp@0.1.0` e flag `--bypass-permissions`. O substrato ACP (`internal/runtime/client/client.go`, `runner.go`, `events/`, `persistence/`) é tool-agnóstico, validado pelas entregas de F1-Copilot (ADR-012) e F1-Codex (ADR-013). **Não há gap de transporte.**

O gap Claude-2026 está em outro lugar: **paridade funcional com o stack que o Compozy construiu acima da camada ACP**. Investigação via `gh api` no `compozy/compozy@7f38c44` revelou que, embora Compozy use o mesmo `coder/acp-go-sdk v0.6.3` que o harness, ele adiciona cinco capacidades não presentes no Orchestrator:

1. **Servidor MCP reservado** (`internal/core/agents/mcpserver/server.go`) expondo a tool `run_agent(agent_name, prompt)` para hand-off recursivo entre agentes (sub-agents nesting com profundidade ≤ 3).
2. **Normalização de tool-calls** (`internal/core/agent/tool_call_input.go::buildNormalizedToolUseBlock`) canonicalizando entradas de diferentes runtimes ACP (Claude vs Codex vs Cursor vs Droid) para um schema interno único — driver-aware.
3. **Memória markdown 2-tier com compactação prompt-driven** (`internal/core/memory/store.go`) — `WorkflowMemory` 150 linhas/12 KiB + `TaskMemory` 200 linhas/16 KiB. Compactação **não** é code-driven: quando o limite é atingido, `NeedsCompaction=true` é sinalizado e o builder de prompt injeta diretiva no system prompt para que o LLM compacte.
4. **Pipeline de hooks in-process Go** (`internal/core/kernel/` + `internal/core/prompt/common.go`) com pontos de despacho `prompt.pre_build`/`prompt.post_build`/`review.pre_resolve`/`review.post_fix`. **Não é JSON-RPC**: hooks são funções Go registradas em compile-time ou via extension manifest.
5. **Auto-review como modo de execução separado** (`internal/core/run/executor/review_hooks.go`) — `ExecutionModePRReview` invoca `reviewProvider.ResolveIssues(ctx, pr, issues)` contra GitHub/CodeRabbit após fixes; não é um sub-loop dentro de execução normal.

Recomendação operacional: tratar a trilha Claude como continuação natural de F1 (já entregue) com quatro fases novas — **F2-Claude (MCP nested + tool-call normalization)**, **F3-Claude (prompt hooks + 2-tier memory)**, **F4-Claude (evidence enrichment Claude-2026: cache_read_tokens, cache_creation_tokens, thinking_tokens)** e **F5-Claude (auto-review opt-in)**. Esforço total estimado: ~3 sprints. **Esta pesquisa não executa nenhuma fase**; entrega apenas ADR-014, PRD e TechSpec como pacote de governança que alimenta `create-tasks`.

Correções factuais ao prompt original estão documentadas em §"Correções ao prompt original": (a) Compozy **não** usa SDK Anthropic — fala via `claude-agent-acp` como qualquer outro runtime; (b) **não há JSON-RPC custom** — envelope é o do `coder/acp-go-sdk`; (c) MCP em Compozy é **apenas** para nested-agent, não para expor `spec-hash` ou skills; (d) memória **não** é evolutiva/vector-indexed — é markdown com limites byte/linha; (e) auto-review **não** é loop interno do agente — é modo separado com provider.

---

## Correções ao prompt original

O prompt em `docs/prompts/compozy-adaptation-research-claude.md` carrega quatro premissas que esta pesquisa precisou corrigir contra o código real do compozy (validado via `gh api repos/compozy/compozy/contents/...`):

1. **"Orquestração de Eventos ACP" + "Protocolo de Mensagens JSON-RPC" + "como o contexto é podado ou compactado para manter a eficiência em sessões longas"** (§1 do prompt) — Compozy **não define wire format próprio**. Toda comunicação ACP é o envelope nativo do `coder/acp-go-sdk v0.6.3` (`go.mod` linha equivalente), idêntico ao consumido pelo harness em `internal/runtime/client/client.go`. **Não há JSON-RPC custom** nem extensão do schema ACP. Compactação de contexto **não é code-driven**: o builder de prompt (`internal/core/prompt/common.go`) injeta diretiva textual ("compact the flagged memory files before proceeding") no system prompt quando `NeedsCompaction=true`, e o LLM executa a compactação. **Princípio prompt-driven, não algoritmo de janela deslizante.** A pesquisa correlata para Copilot já documentou isso (`compozy-adaptation-copilot-2026.md` §"Correções"); reaplicado aqui sem modificação.

2. **"Servidor MCP interno que exponha o `ai-spec-harness` como um recurso, garantindo que o Claude valide o `spec-hash` e o `drift` em tempo real antes de cada edição"** (Fase 2 do prompt) — Compozy expõe **uma única tool** via MCP em `internal/core/agents/mcpserver/server.go` (constante `reservedToolName = "run_agent"`). Essa tool serve para **hand-off recursivo entre agentes** (um agente parent invocar um child agente com prompt isolado), não para validação de hashes nem para expor skills do harness. `spec-hash` é conceito do harness (`internal/specdrift/`) e **não tem contraparte em Compozy** — Compozy não conhece "spec-hash". A Fase 2 do prompt precisa ser reformulada como **"servidor MCP interno expondo `run_agent` para nested-agent execution (paridade Compozy), mantendo `spec-hash` no Orchestrator como concern de governança, não de runtime"**.

3. **"Memória de Projeto Evolutiva: migrar de um `MEMORY.md` estático para um sistema de indexação de contexto inspirado no Compozy, otimizando o consumo de tokens para grandes refatorações"** (Fase 4 do prompt) — Compozy implementa memória em `internal/core/memory/store.go` com duas camadas:
   - **Workflow Memory**: `<tasksDir>/memory/MEMORY.md` — limite hard `workflowLineLimit = 150` linhas, `workflowByteLimit = 12 * 1024` bytes
   - **Task Memory**: `<tasksDir>/memory/<task-file>.md` — limite hard `taskLineLimit = 200` linhas, `taskByteLimit = 16 * 1024` bytes
   
   **Não há vector store, embeddings, indexação semântica nem ML**. A compactação é o LLM lendo a diretiva injetada no system prompt e reescrevendo o markdown. A Fase 4 do prompt precisa ser reformulada como **"adotar memória 2-tier markdown com limites de byte/linha e compactação prompt-driven"** — barata, determinística, auditável.

4. **"Loop de Refinamento e Revisão: adotar o padrão de Auto-Review do Compozy, onde o Claude Code invoca a skill `review` internamente antes de sinalizar o término de uma task"** (Fase 3 do prompt) — Compozy **não** invoca review como sub-loop dentro de execução normal. Review é um **modo de execução separado** (`ExecutionModePRReview` em `internal/core/run/executor/review_hooks.go`), disparado contra um PR já aberto, que carrega issues de provider (GitHub/CodeRabbit), executa fixes, e chama `reviewProvider.ResolveIssues(ctx, pr, issues)` via hooks `review.pre_resolve` / `review.post_fix`. Em execução normal (`ExecutionModePRDTasks`), o hook `afterTaskJobSuccess` apenas emite `TaskFileUpdated` — sem chamada a review skill. A Fase 3 do prompt precisa ser reformulada como **"auto-review opt-in via flag `--auto-review` no `runner.go`, invocando skill `review` (já existente em `.agents/skills/review/`) antes de `EnrichReport` quando a flag for passada"** — não é o padrão Compozy literalmente, mas é a interpretação mais defensável para o caso de uso do Orchestrator.

Essas correções não invalidam o roadmap proposto; alinham as recomendações ao código-fonte real e evitam que ADR-014 / PRD / TechSpec carreguem expectativas inexistentes.

---

## Mecânica Claude-native em Compozy (achados por arquivo)

Investigação via `gh api repos/compozy/compozy/contents/<path>?ref=7f38c44` e `gh search code --repo compozy/compozy "<term>"`. Cada achado abaixo cita arquivo + propósito.

### 1. Transporte ACP unificado — Claude trata-se como qualquer outro runtime (Compozy ✅ | harness 🟢)

**Compozy** — `internal/core/agent/client.go`:

```go
// ClientConfig.CreateSession(ctx, SessionRequest) -> spawn subprocess + bidirectional stdio ACP
// SessionRequest.Prompt: []byte (plaintext markdown)
// Driver-aware via registry_specs.go
```

Registry de runtimes em `internal/core/agent/registry_specs.go` lista Claude lado a lado com Codex/Copilot/Cursor/Droid/Gemini/OpenCode/Pi:

```go
model.IDEClaude: {
    Command: "claude-agent-acp",
    Fallbacks: []Launcher{{Command: "npx", FixedArgs: []string{"--yes", "@agentclientprotocol/claude-agent-acp"}}},
    BootstrapArgs: nil,  // no-op
}
```

**ai-spec-harness** — `internal/runtime/specs/claude.go:28-45` declara o equivalente:

```go
return newSpec(
    "claude", "Claude (ACP)", "claude-agent-acp",
    nil,
    []FallbackLauncher{{Command: "npx", FixedArgs: []string{"--yes", ClaudeNpmPackage + "@" + ClaudeNpmVersion}}},
    "--bypass-permissions",
    ClaudeSDKVersion, ClaudeNpmVersion, ClaudeNpmPackage,
)
```

**Estado**: paridade total. Sem gap.

### 2. Normalização de tool-calls driver-aware (Compozy ✅ | harness ❌)

**Compozy** — `internal/core/agent/tool_call_input.go::buildNormalizedToolUseBlock`:

```go
func buildNormalizedToolUseBlock(
    driverID string, toolCallID string, title string,
    kind acp.ToolKind, rawInput any, locations []acp.ToolCallLocation, meta any,
) (model.ContentBlock, error) {
    normalizedInput := normalizeACPToolInput(driverID, nameHint, kind, rawInput, locations)
    name := normalizeACPToolName(driverID, nameHint, kind, normalizedInput)
    // ...
    return model.NewContentBlock(model.ToolUseBlock{
        ID: toolCallID, Name: name, Title: displayTitle,
        ToolName: metaToolName, Input: inputPayload, RawInput: rawInputPayload,
    })
}
```

Função-irmã `normalizeACPToolInput(driverID, ...)` canonicaliza campos divergentes entre runtimes. Exemplos documentados: `startLine` (Claude) vs `startLineNumberBaseOne` (Cursor); `command` (Claude `bash`) vs `cmd` (Codex `shell`); etc. Saída é um `ToolUseBlock` com payload uniforme — dashboards, telemetria e replay tools consomem o mesmo schema independente do runtime de origem.

**ai-spec-harness** — `internal/runtime/events/` propaga nomes/inputs como vêm do SDK, sem normalização driver-aware. Em sessões cross-runtime (replay de Claude side-by-side com Codex), tools com mesma semântica aparecem com nomes/campos divergentes.

**Impacto**: telemetria multi-tool fica fragmentada; ADR-008 (paridade) tem que cobrir mapeamentos por driver no nível de teste, não no nível de runtime. Para Claude isolado, o impacto é cosmético; em frota multi-tool é estrutural.

**Gap técnico**: F2-Claude (~150 LoC + tabelas de alias). Novo `internal/runtime/events/normalize.go` com `BuildNormalizedToolCall(driverID, kind, rawInput) NormalizedToolCall`. Tabelas de alias em const block ou YAML (`.agents/normalization-rules.yaml`).

### 3. MCP server reservado expondo `run_agent` (Compozy ✅ | harness ❌)

**Compozy** — `internal/core/agents/mcpserver/server.go`:

```go
const reservedToolName = "run_agent"

// Server implementa um stdio MCP server com uma única tool: run_agent.
// Permite que o agente parent invoque um child agent com prompt isolado.
// Profundidade máxima: 3 (validado em internal/core/agents/session_mcp.go).
```

E `internal/core/agents/mcpserver/engine.go` executa o nested agent dentro do mesmo processo Compozy: serializa contexto via env var `COMPOZY_RUN_AGENT_CONTEXT` contendo `NestedExecutionContext` (workspace root, model, IDE, reasoning effort, timeout, nesting depth) e spawna outra `Session` ACP.

Agentes podem **também** declarar MCP servers próprios em `.compozy/agents/<name>/manifest.yaml` sob `resources.mcp`; `BuildSessionMCPServers()` em `internal/core/agents/session_mcp.go` merge esses com o reservado.

**ai-spec-harness**: **nenhum servidor MCP** em `internal/`. `.claude/` contém configuração MCP consumida pelo Claude Code em runtime mas o Orchestrator não hospeda nem expõe nada via MCP. Não há mecanismo de hand-off entre agentes do harness — sub-agents só existem dentro do escopo da skill `execute-all-tasks` (orchestrator decide a ordem; cada task é um spawn novo do Claude Code, sem MCP).

**Impacto**: sem `run_agent`, fluxos como "executor delega review a um sub-agent reviewer" exigem fora-de-banda (skill orchestrator decide; agente não pode pedir). Compozy resolve isso elegantemente; harness depende do orchestrator humano (ou skill) para sequenciar.

**Gap técnico**: F2-Claude. Novo `internal/runtime/mcpserver/server.go` (~200 LoC) com:
- Tool schema `run_agent` (input: `{agent_name: string, prompt: string, model?: string, timeout?: int}`; output: `{summary: string, evidence_dir: string}`)
- Wire: stdio MCP via `github.com/modelcontextprotocol/go-sdk` (ou implementação minimal própria)
- Profundidade máxima 3 (alinhada ao Compozy)
- Contexto serializado via env var `AISPEC_RUN_AGENT_CONTEXT`
- Integração com `internal/agents/registry.go` (ADR-011) para resolver `agent_name` → spec

### 4. Memória 2-tier markdown com limites (Compozy ✅ | harness 🟡 sem tier-2)

**Compozy** — `internal/core/memory/store.go`:

```go
const (
    DirName            = "memory"
    WorkflowFileName   = "MEMORY.md"
    workflowLineLimit  = 150
    workflowByteLimit  = 12 * 1024
    taskLineLimit      = 200
    taskByteLimit      = 16 * 1024
)

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
```

Função `ReadDocument(tasksDir, taskFileName)` resolve workflow vs task automaticamente: `taskFileName == ""` → MEMORY.md global; `taskFileName == "task-3.0-foo.md"` → memory/task-3.0-foo.md. `WriteDocument(...)` aplica os limites e seta `NeedsCompaction=true` quando ultrapassado. **Não há compactação automática**: o builder de prompt em `internal/core/prompt/prd.go` lê `NeedsCompaction` e injeta a diretiva "compact the flagged memory files before proceeding" no system prompt — o LLM faz o trabalho.

**ai-spec-harness**: usa `MEMORY.md` (auto memory de Claude Code, especificado em `CLAUDE.md` raiz) **sem tier de task-level** e **sem limites enforced**. O sistema de memory de Claude Code é prompt-driven mas sem o split workflow/task — todo conteúdo vai para `~/.claude/projects/.../memory/MEMORY.md` indexado por slug.

**Impacto**: em PRDs grandes (>10 tasks), MEMORY.md cresce além do útil. Não há sinalização para o LLM compactar; o agente humano (skill `execute-all-tasks`) faz manualmente entre waves.

**Gap técnico**: F3-Claude (~150 LoC). Novo `internal/runtime/memory/store.go` espelhando Compozy:
- `WorkflowPath(tasksDir) string` → `<tasksDir>/memory/MEMORY.md`
- `TaskPath(tasksDir, taskFileName) string` → `<tasksDir>/memory/<taskFileName>`
- `ReadDocument(tasksDir, taskFileName) (Document, error)`
- `WriteDocument(tasksDir, taskFileName, content, mode WriteMode) error`
- `NeedsCompaction(scope)` retornado em `FileState`
- Limites configuráveis via flags `--memory-workflow-limit`, `--memory-task-limit` (defaults idênticos a Compozy)
- Integração em `runner.go::Run()`: antes de `c.Open`, ler workflow + task memory e injetar no prompt; depois do session end, oferecer ao agente um stub para escrita (via hook `memory.post_build`).

### 5. Pipeline de hooks in-process Go (Compozy ✅ | harness 🟡 shell scripts)

**Compozy** — `internal/core/kernel/` + `internal/core/prompt/common.go`:

```go
// internal/core/prompt/common.go
func BuildSystemPromptAddendum(ctx context.Context, req BuildRequest) (string, error) {
    // ...
    if err := kernel.Dispatch(ctx, "prompt.pre_build", &req); err != nil { return "", err }
    addendum := assembleAddendum(req)
    if err := kernel.Dispatch(ctx, "prompt.post_build", &PostBuildEvent{Addendum: &addendum}); err != nil { return "", err }
    return addendum, nil
}
```

Lista de hook points documentados em `compozy-adaptation-copilot-2026.md` §"Hook System" — 33 canônicos no total, in-process Go (não shell, não JSON-RPC). Vantagens: tipados, testáveis, sem fork/exec overhead, podem mutar a request em-place.

**ai-spec-harness** — hooks atuais em `.claude/hooks/`:
```
post-execute-task.sh
post-wave.sh
pre-execute-all-tasks.sh
subagent-stop-wrapper.sh
validate-governance.sh
validate-preload.sh
validate-token-budget.sh
```

Todos shell scripts. Executados pelo próprio Claude Code (não pelo Orchestrator) via configuração em `.claude/settings.json`. **O Orchestrator não dispara nenhum hook** — quando o ACP runner spawna `claude-agent-acp`, hooks do `.claude/` são lidos pelo Claude Code, não pelo runner.

**Impacto**: hooks atuais só funcionam quando o usuário roda Claude Code interativamente, não quando o Orchestrator invoca via ACP. Em sessões ACP, governance hooks (`validate-governance.sh`, `validate-token-budget.sh`) são **silenciosamente ignorados** — gap crítico de paridade entre modo interativo e modo orquestrado.

**Gap técnico**: F3-Claude (~250 LoC). Novo `internal/runtime/hooks/` com:
- `type Hook interface { Name() string; Run(ctx context.Context, evt Event) error }`
- Dispatcher `kernel.Dispatch(ctx, name, evt)` com fan-out tipado
- Pontos canônicos iniciais (subset, não 33 de uma vez):
  - `runtime.pre_open` (antes de `c.Open` em `runner.go:145`)
  - `prompt.pre_build` (antes do prompt ser passado ao Claude)
  - `prompt.post_build` (depois do prompt finalizado)
  - `tool_call.pre_dispatch` (antes do tool call ser enviado de volta ao Claude)
  - `tool_call.post_complete` (depois do tool result ser injetado)
  - `session.post_end` (antes de `EnrichReport`)
- Migração progressiva de `.claude/hooks/*.sh` para hooks Go in-process; manter shell hooks como compatibilidade para modo interativo Claude Code.

### 6. Auto-review como modo separado, não sub-loop (Compozy ✅ | harness 🟡 manual)

**Compozy** — `internal/core/run/executor/review_hooks.go`:

```go
// Dois modos de execução, mutuamente exclusivos:
//   - ExecutionModePRDTasks: rodar tasks de PRD; review NÃO é chamado
//   - ExecutionModePRReview: rodar review contra PR já aberto

func (e *Executor) afterTaskJobSuccess(ctx context.Context, job Job) error {
    if e.execMode != ExecutionModePRReview {
        return e.emitTaskFileUpdated(job)
    }
    // Modo review: chamar provider
    if err := kernel.Dispatch(ctx, "review.pre_resolve", &job); err != nil { return err }
    if err := e.reviewProvider.ResolveIssues(ctx, job.PR, job.Issues); err != nil { return err }
    return kernel.Dispatch(ctx, "review.post_fix", &job)
}
```

`reviewProvider` é uma abstração em `internal/core/provider/provider.go` com implementações para GitHub e CodeRabbit.

**ai-spec-harness**: review é skill (`.agents/skills/review/SKILL.md`) invocada **manualmente** por humano ou por outro agente. O Orchestrator nunca chama review automaticamente. `internal/evidence/evidence.go` valida que o `execution_report.md` contém seções obrigatórias, mas não dispara revisão.

**Impacto**: skills `execute-task` e `execute-all-tasks` podem produzir código quebrado/inseguro e o ciclo só fecha quando humano roda `/review` ou `ai-spec review`. Em produção isso é aceitável; em loops longos (sessões batch) é fonte de regressão.

**Gap técnico**: F5-Claude (~100 LoC). Flag opt-in `--auto-review` no `cmd/ai_spec_harness/task_loop.go` que, quando true:
- Após session end e antes de `EnrichReport`, spawna nova sessão ACP com prompt da skill `review` e o diff acumulado da task
- Persiste o resultado em `evidence/<task>/review.md`
- Se review identificar issues `hard`, atualiza `Summary.ReviewStatus = "blocked"` e bloqueia transição da task para `done`

Não copiar literalmente o Compozy (provider GitHub/CodeRabbit) — para o harness, "provider" é a skill local; mais simples e alinhado com `R-GOV-001` (governança transversal sobre adição de complexidade).

### 7. Evidence enrichment Claude-2026: cache + thinking + tool_calls_normalized (Compozy 🟡 | harness ❌)

Compozy registra eventos em `pkg/compozy/events/events.go` com tipos `EventKindTaskFileUpdated`, `EventKindToolCallStarted`, etc. **Não há campos Claude-2026 específicos** (cache_read_tokens, cache_creation_tokens, thinking_tokens) — Compozy não os extrai do session metadata do ACP.

**ai-spec-harness** — `internal/evidence/evidence.go:29-49` valida secções obrigatórias do `execution_report.md`:
```
Contexto Carregado, Comandos Executados, Arquivos Alterados,
Resultados de Validacao, Suposicoes, Riscos Residuais
```

Sem campos numéricos Claude-2026. ADR-006 (telemetria) registra opt-in em `audit/`, mas não captura métricas de cache nem reasoning tokens.

**Impacto**: dois efeitos. (a) Não dá para analisar custo real de sessões Claude (cache_read pode reduzir 80% do custo); (b) Não dá para correlacionar `thinking_tokens` com qualidade do output em refatorações complexas. Ambas métricas estão disponíveis no `acp.SessionUpdate` payload — só falta extrair e persistir.

**Gap técnico**: F4-Claude (~80 LoC). Estender `internal/runtime/events/convert.go` para extrair campos do payload do ACP-SDK quando presentes; estender `internal/evidence/evidence.go` com novos campos opcionais:
```
| Métrica                    | Valor |
|----------------------------|-------|
| cache_read_tokens          | N     |
| cache_creation_tokens      | N     |
| thinking_tokens            | N     |
| tool_calls_normalized      | N     |
```

Validar via regex opcional (presença não obrigatória; ausência não bloqueia).

---

## Tabela de Gaps Consolidada (Claude-CLI 2026)

Legenda: 🟢 implementado · 🟡 parcial · 🔴 ausente · ⭐ vantagem do harness a preservar

| # | Feature | Status Orchestrator | Padrão Compozy | Gap Técnico | Fase | Severidade |
|---|---|---|---|---|---|---|
| 1 | Claude via ACP nativo (`claude-agent-acp`) | 🟢 ADR-009 + `specs/claude.go` | 🟢 `registry_specs.go::IDEClaude` | — | F1-Claude (entregue) | — |
| 2 | Forense `events.jsonl`/`tool_calls.md`/`execution_report.md` | 🟢 ⭐ | 🟡 (OTel/Grafana, sem markdown) | — (harness diferencial) | F1-Claude (entregue) | — |
| 3 | `ActivityWatchdog` com `CancelCause` | 🟢 ⭐ | 🔴 | — (harness diferencial) | F1-Claude (entregue) | — |
| 4 | Servidor MCP reservado `run_agent` (nested-agent) | 🔴 | 🟢 `agents/mcpserver/server.go` | Novo `internal/runtime/mcpserver/` (~200 LoC) | **F2-Claude** | hard |
| 5 | Normalização de tool-calls driver-aware | 🔴 | 🟢 `tool_call_input.go::buildNormalizedToolUseBlock` | Novo `events/normalize.go` (~150 LoC) + tabelas | **F2-Claude** | guideline |
| 6 | Memória 2-tier markdown com limites byte/linha | 🟡 (só workflow via Claude Code memory) | 🟢 `memory/store.go` (workflow 150/12KB; task 200/16KB) | Novo `internal/runtime/memory/` (~150 LoC) + integração runner | **F3-Claude** | hard |
| 7 | Pipeline de hooks in-process Go | 🟡 shell scripts em `.claude/hooks/` (não disparam em modo ACP) | 🟢 `kernel/` com 33 hook points | Novo `internal/runtime/hooks/` (~250 LoC) + 6 pontos canônicos iniciais | **F3-Claude** | hard |
| 8 | Evidence com campos Claude-2026 (cache_read, thinking) | 🔴 | 🟡 (não extrai) | Estender `events/convert.go` + `evidence/evidence.go` (~80 LoC) | **F4-Claude** | guideline |
| 9 | Auto-review opt-in via flag CLI | 🔴 | 🟢 (modo separado `ExecutionModePRReview`) | Flag `--auto-review` em `task_loop.go` + integração runner (~100 LoC) | **F5-Claude** | guideline |
| 10 | Wrapper `ValidTools` para Claude | 🔴 **intencional** (`wrapper_test.go:213` — "claude should not be in ValidTools (uses hooks)") | n/a | **Não alterar** — gap deliberado | — | — |
| 11 | Registry de agentes declarativo (ADR-011) | 🟢 `internal/agents/registry.go` | 🟢 `.compozy/agents/<name>/manifest.yaml` | — | F1-Claude (entregue) | — |
| 12 | `spec-hash` validation | 🟢 ⭐ `internal/specdrift/` | 🔴 | — (harness diferencial) | F1-Claude (entregue) | — |
| 13 | Telemetria opt-in (`GOVERNANCE_TELEMETRY=1`) | 🟢 ⭐ ADR-006 | 🟡 (OTel sempre on) | — (harness diferencial) | F1-Claude (entregue) | — |
| 14 | `CLAUDE.md` 2026 atualizado com Runtime Capabilities | 🟡 atual cita CLAUDE.md base | 🟢 (`CLAUDE.md` em compozy raiz) | Adicionar §"Runtime Capabilities" listando MCP + hooks expostos | **F2-Claude** | guideline |

**Critérios de esforço**: Baixo ≤ 1 sprint; Médio 1–2 sprints; Alto ≥ 2 sprints com pesquisa.

**Sumário**: F1-Claude entregue. Quatro fases novas (F2..F5) totalizando ~3 sprints. Severidade hard concentrada em F2 (MCP nested) e F3 (hooks + memory). F4 e F5 são guideline — entregam valor mas não bloqueiam paridade arquitetural.

---

## Roadmap de Adaptação (Claude-específico)

### F2-Claude — MCP nested-agent + tool-call normalization (próximo PRD candidato)

**Escopo**:

- Novo `internal/runtime/mcpserver/server.go` (~150 LoC) implementando stdio MCP server com tool única `run_agent(agent_name, prompt, model?, timeout?)`. Resolução de `agent_name` via `internal/agents/registry.go` (ADR-011). Profundidade máxima configurável (`AISPEC_MAX_AGENT_DEPTH`, default 3).
- Novo `internal/runtime/mcpserver/engine.go` (~50 LoC) spawnando novo `ACPRunner` dentro do mesmo processo, com prompt isolado, e serializando contexto via env var `AISPEC_RUN_AGENT_CONTEXT` (JSON com: workspace_root, parent_task, parent_session_id, depth).
- Integração em `internal/runtime/runner.go`: se flag `--mcp-nested` ativa (default off em F2), spawna o MCP server em goroutine e injeta `--mcp-server stdio://...` no launcher do Claude.
- Novo `internal/runtime/events/normalize.go` (~150 LoC) com `BuildNormalizedToolCall(driverID, kind, rawInput) NormalizedToolCall`. Tabelas de alias em arquivo `.agents/normalization-rules.yaml` (lido em init via `go:embed`).
- Persistência: `events.jsonl` ganha campo `normalized_name` ao lado de `raw_name`; `tool_calls.md` renderiza nome normalizado.
- Testes:
  - T-01 `TestMCPServerExposesRunAgentOnly` — única tool exposta
  - T-02 `TestRunAgentResolvesViaRegistry` — `agent_name` desconhecido retorna erro
  - T-03 `TestRunAgentDepthLimit` — depth=4 retorna erro
  - T-04 `TestNormalizeToolCallClaude` — Claude `bash` mantido como `bash`
  - T-05 `TestNormalizeToolCallCodex` — Codex `shell` normalizado para `bash`
  - T-06 `TestNormalizationDoesntLoseRawPayload` — `raw_name` preservado
- Atualização de `CLAUDE.md` raiz com nova §"Runtime Capabilities":
  ```markdown
  ## Runtime Capabilities (F2-Claude+)
  - MCP nested-agent: tool `run_agent(agent_name, prompt)` disponível quando
    `--mcp-nested` for passado a `ai-spec task-loop`.
  - Tool calls são normalizados; o `tool_calls.md` registra o nome canônico.
  ```
- ADR-014 (este pacote) documenta as decisões.

**Esforço**: Médio (~1 sprint).  
**Risco**: Médio (MCP é fronteira nova; biblioteca SDK Go ainda matura em 2026).  
**Dependências**: F1-Codex (ADR-013) entregue; nenhuma outra.  
**Critério de aceitação**: `ai-spec task-loop --tool claude --runtime acp --mcp-nested tasks/prd-X` produz `events.jsonl` com `tool_call_kind="nested_agent"` quando o Claude invocar `run_agent` no diff.

### F3-Claude — Prompt hooks + 2-tier memory store

**Escopo**:

- Novo `internal/runtime/hooks/dispatcher.go` (~100 LoC) com:
  - `type Hook interface { Name() string; Run(ctx, evt) error }`
  - `type Dispatcher struct { hooks map[string][]Hook }`
  - `Dispatcher.Dispatch(ctx, name string, evt any) error` (fan-out sequencial, abort-on-error)
  - Registro em init via `hooks.Register("prompt.pre_build", myHook)`
- 6 pontos canônicos despachados em `runner.go`:
  - `runtime.pre_open` (linha 145 antes de `c.Open`)
  - `prompt.pre_build` (antes de montar prompt final)
  - `prompt.post_build` (depois de montar)
  - `tool_call.pre_dispatch` (no loop de eventos, antes de persist)
  - `tool_call.post_complete` (depois)
  - `session.post_end` (antes de `EnrichReport`)
- Novo `internal/runtime/memory/store.go` (~150 LoC) replicando Compozy:
  - `Directory(tasksDir)` / `WorkflowPath(tasksDir)` / `TaskPath(tasksDir, taskFileName)`
  - `ReadDocument(tasksDir, taskFileName) (Document, error)`
  - `WriteDocument(tasksDir, taskFileName, content, mode WriteMode) error`
  - `FileState.NeedsCompaction` setado por limites configuráveis (defaults idênticos a Compozy: 150 lin/12 KB workflow, 200 lin/16 KB task)
- Integração runner: antes de `c.Open`, ler workflow + task memory, montar bloco "## Memory Context" e prepend ao prompt. Quando `NeedsCompaction=true`, anexar diretiva textual "compact the flagged memory files before proceeding" (idêntico ao Compozy, prompt-driven).
- Hook default `memory.persist` registrado em `session.post_end` que escreve `MEMORY.md`/`<task>.md` com `Summary` enriquecido.
- Migração progressiva dos shell hooks em `.claude/hooks/`:
  - `validate-governance.sh` → `internal/runtime/hooks/governance.go` (in-process)
  - `validate-token-budget.sh` → `internal/runtime/hooks/token_budget.go`
  - Demais permanecem como shell (modo interativo Claude Code)
- Testes:
  - T-07 `TestDispatcherRespectsRegistrationOrder`
  - T-08 `TestDispatcherAbortsOnFirstError`
  - T-09 `TestMemoryReadWriteRoundTrip`
  - T-10 `TestMemoryNeedsCompactionFlag` — escrever 151 linhas e checar flag
  - T-11 `TestRunnerInjectsMemoryContextWhenAvailable`
  - T-12 `TestGovernanceHookBlocksMissingAGENTS_md`
- Flags novas em `cmd/ai_spec_harness/task_loop.go`:
  - `--memory-workflow-limit-lines` (default 150)
  - `--memory-workflow-limit-bytes` (default 12288)
  - `--memory-task-limit-lines` (default 200)
  - `--memory-task-limit-bytes` (default 16384)
  - `--disable-hooks` (default false; útil para debugging)

**Esforço**: Médio (~1.5 sprints).  
**Risco**: Médio (migração de hooks shell pode quebrar UX interativo se não for cuidadosa — mitigar mantendo shell como fallback).  
**Dependências**: F2-Claude entregue (para hook `tool_call.pre_dispatch` consumir nome normalizado).  
**Critério de aceitação**: sessão Claude com memory workflow > 150 linhas emite no `execution_report.md` linha `Memory Compaction Requested: true` e o LLM faz a compactação dentro do turn.

### F4-Claude — Evidence enrichment com campos Claude-2026

**Escopo**:

- Estender `internal/runtime/events/convert.go` para extrair campos opcionais do payload ACP quando presentes:
  - `cache_read_tokens` (do session metadata)
  - `cache_creation_tokens` (idem)
  - `thinking_tokens` (do reasoning content block, se existir)
  - `tool_calls_normalized_count` (calculado em F2)
- Estender `Summary` em `internal/runtime/runner.go` com esses campos.
- Estender `internal/evidence/evidence.go` com seção "Métricas Claude-2026" opcional:
  ```markdown
  ## Métricas Claude-2026
  | Métrica | Valor |
  |---|---|
  | cache_read_tokens | N |
  | cache_creation_tokens | N |
  | thinking_tokens | N |
  | tool_calls_normalized | N |
  ```
- Validador: presença é opcional; ausência não bloqueia evidence.
- Telemetria (`internal/telemetry/`): se `GOVERNANCE_TELEMETRY=1`, append entries `claude.cache_read=N`, `claude.thinking=N`.

**Esforço**: Baixo (~3 dias).  
**Risco**: Baixo.  
**Dependências**: F3-Claude entregue (hook `session.post_end` é onde Summary é enriquecido).

### F5-Claude — Auto-review opt-in

**Escopo**:

- Flag `--auto-review` em `cmd/ai_spec_harness/task_loop.go`. Default `false`.
- Quando true e session end completou sem erro:
  - Spawnar nova `ACPRunner` com prompt da skill `review` (carregar de `.agents/skills/review/SKILL.md`) + diff acumulado da task (via `git diff` no `workDir`)
  - Persistir resultado em `evidence/<task>/review.md`
  - Parsear resultado procurando palavras-chave `BLOQUEADO`, `CRÍTICO`, severidade `hard`
  - Se hard issue presente, setar `Summary.ReviewStatus = "blocked"` e `execution_report.md` recebe seção "Review Block" com lista de issues
- Hook `session.post_review` (novo ponto canônico) despachado depois da revisão.
- Não copiar literalmente o modelo Compozy (provider GitHub/CodeRabbit) — para o harness, "provider" é a skill local; mais simples e alinhado com `R-GOV-001` (governança transversal sobre adição de complexidade).

**Esforço**: Baixo (~3 dias).  
**Risco**: Baixo–Médio (auto-review pode dobrar custo de tokens; documentar trade-off).  
**Dependências**: F3-Claude entregue.

---

## Exemplos de Configuração 2026

### `CLAUDE.md` raiz — adendos F2/F3-Claude

```markdown
# ai-spec-harness — Claude Code

> Use `AGENTS.md` como fonte canonica das regras deste repositorio.

## Runtime Capabilities (F2-Claude+)

Quando o Orchestrator invoca Claude via ACP, o runtime expõe:

- **MCP nested-agent** (`--mcp-nested`): tool `run_agent(agent_name, prompt, model?, timeout?)`.
  Profundidade máxima: 3. Resolução de `agent_name` via `internal/agents/registry.go`.
- **Tool-call normalization** (sempre ativo a partir de F2): nomes/inputs de tools
  são canonicalizados; `tool_calls.md` registra nome normalizado. Tabela em
  `.agents/normalization-rules.yaml`.
- **Hooks in-process Go** (F3-Claude+): pontos canônicos `runtime.pre_open`,
  `prompt.pre_build`, `prompt.post_build`, `tool_call.pre_dispatch`,
  `tool_call.post_complete`, `session.post_end`. Para desabilitar: `--disable-hooks`.
- **Memória 2-tier** (F3-Claude+): workflow `tasks/<prd>/memory/MEMORY.md` +
  task `tasks/<prd>/memory/<task>.md`. Limites configuráveis; compactação
  prompt-driven quando `NeedsCompaction=true`.
- **Evidence Claude-2026** (F4-Claude+): `execution_report.md` ganha seção
  "Métricas Claude-2026" com `cache_read_tokens`, `cache_creation_tokens`,
  `thinking_tokens`, `tool_calls_normalized`.
- **Auto-review** (F5-Claude+, opt-in `--auto-review`): após session end,
  spawna sessão extra com skill `review`. Hard issues bloqueiam transição
  da task para `done`.

## Instrucoes (mantidas)
1. Ler `AGENTS.md` no inicio da sessao.
...
```

### `.claude/agents/` evolução (alinhamento com `internal/agents/registry.go`)

Hoje `.claude/agents/` lista: `bugfixer.md`, `prd-writer.md`, `project-analyzer.md`, `refactorer.md`, `reviewer.md`, `task-executor.md`, `task-planner.md`, `technical-specification-writer.md`. A partir de F2-Claude, esses arquivos passam a ser **resolvidos via `internal/agents/registry.go` (ADR-011)** quando MCP `run_agent` for invocado: `run_agent("reviewer", "<prompt>")` resolve para o spec de `reviewer` e spawna nova sessão ACP isolada. Manifest opcional `.claude/agents/<name>.yaml` (paralelo ao `.md`) declara model preferido, reasoning effort, timeout. Sem manifest, herda do parent.

### `.claude/skills/` recebendo hooks `prompt.pre_build`

A partir de F3-Claude, qualquer skill em `.claude/skills/<name>/` pode declarar em seu frontmatter um hook in-process:

```yaml
---
name: governance-preflight
hook: prompt.pre_build
implementation: internal/runtime/hooks/governance.go
priority: 100
---
```

Implementação Go obrigatória; YAML é só metadata para o registry. Hooks shell em `.claude/hooks/*.sh` permanecem como camada de compatibilidade para modo interativo Claude Code (não migra para Go).

### Stub de `internal/runtime/mcpserver/server.go` (F2-Claude)

```go
package mcpserver

import (
    "context"
    "encoding/json"
    "fmt"
    "github.com/JailtonJunior94/ai-spec-harness/internal/agents"
)

const ReservedToolName = "run_agent"

type RunAgentInput struct {
    AgentName string `json:"agent_name"`
    Prompt    string `json:"prompt"`
    Model     string `json:"model,omitempty"`
    Timeout   int    `json:"timeout,omitempty"`
}

type RunAgentOutput struct {
    Summary     string `json:"summary"`
    EvidenceDir string `json:"evidence_dir"`
}

type Server struct {
    registry agents.Registry
    maxDepth int
}

func New(registry agents.Registry, maxDepth int) *Server {
    if maxDepth <= 0 { maxDepth = 3 }
    return &Server{registry: registry, maxDepth: maxDepth}
}

func (s *Server) HandleRunAgent(ctx context.Context, raw json.RawMessage) (json.RawMessage, error) {
    var in RunAgentInput
    if err := json.Unmarshal(raw, &in); err != nil {
        return nil, fmt.Errorf("invalid run_agent input: %w", err)
    }
    if depth := currentDepth(ctx); depth >= s.maxDepth {
        return nil, fmt.Errorf("nested agent depth limit reached (%d)", s.maxDepth)
    }
    spec, err := s.registry.Resolve(in.AgentName)
    if err != nil { return nil, fmt.Errorf("unknown agent %q: %w", in.AgentName, err) }
    out, err := s.spawnNestedSession(ctx, spec, in)
    if err != nil { return nil, err }
    return json.Marshal(out)
}
```

### Stub de `internal/runtime/memory/store.go` (F3-Claude)

```go
package memory

const (
    DirName           = "memory"
    WorkflowFileName  = "MEMORY.md"
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

type Store struct {
    tasksDir string
    limits   Limits
}

func New(tasksDir string, limits Limits) *Store { /* ... */ }
func (s *Store) ReadWorkflow() (Document, error) { /* ... */ }
func (s *Store) ReadTask(taskFileName string) (Document, error) { /* ... */ }
func (s *Store) WriteWorkflow(content string, mode WriteMode) error { /* ... */ }
func (s *Store) WriteTask(taskFileName, content string, mode WriteMode) error { /* ... */ }
```

---

## Riscos e Premissas

| # | Risco/Premissa | Mitigação |
|---|---|---|
| 1 | Biblioteca Go MCP-SDK ainda imatura em 2026-05 — pode quebrar interfaces | Vendor inicial em `internal/runtime/mcpserver/wire/` se necessário; isolar dependência atrás de interface |
| 2 | `claude-agent-acp@0.1.0` pode ser sucedido por novo pacote em 2026-Q3 | Política de pinning ADR-009 já cobre: atualizar via `audit/`, não `@latest` |
| 3 | Migração `.claude/hooks/*.sh` → Go pode quebrar UX interativo de Claude Code | Manter shell hooks como camada coexistente; hooks Go só rodam quando runner ACP é o orchestrator |
| 4 | Memória 2-tier pode duplicar com `~/.claude/projects/.../memory/MEMORY.md` (Claude Code auto-memory) | Documentar precedência: memory do harness vence quando `tasks/<prd>/memory/` existe; auto-memory de Claude Code é fallback |
| 5 | MCP nested-agent expõe vetor de DoS (recursão infinita) | Profundidade máxima 3 (igual Compozy); timeout obrigatório; trace de `parent_session_id` no contexto |
| 6 | Auto-review dobra custo de tokens por task | Opt-in `--auto-review`; documentar trade-off em `CLAUDE.md`; futuro: cache de review por SHA do diff |
| 7 | Tool-call normalization pode mascarar bugs reais do runtime upstream | `raw_name` preservado lado a lado com `normalized_name`; debug via `--no-normalize` |
| 8 | `spec-hash` invalida após qualquer edit em PRD/TechSpec — necessidade de `--sync` frequente | Skill `execute-all-tasks` já invoca `ai-spec check-spec-drift --sync` antes da primeira task; manter |

---

## Decisão Recomendada

Abrir o pacote de governança completo conforme entregue por esta pesquisa:

- **ADR-014** (`tasks/adr/014-claude-cli-acp-native.md`) — status Proposta — documenta D-01..D-07 cobrindo MCP nested-agent, hooks in-process, memória 2-tier, evidence enrichment, auto-review opt-in.
- **PRD** (`tasks/prd-claude-cli-acp-2026/prd.md`) — RF-01..RF-06 + NFRs — consome ADR-014.
- **TechSpec** (`tasks/prd-claude-cli-acp-2026/techspec.md`) — cabeçalho com `spec-hash-prd` placeholder — arquitetura, interfaces, riscos, estratégia de testes; pronto para `create-tasks` após `ai-spec sync-spec-hash` materializar o hash.

**Não escrever código nesta sessão.** Tarefas de implementação ficam para uma sessão posterior via skill `create-tasks` + `execute-task` (subdivididas por fase F2 → F5, com `execute-all-tasks` paralelizando dentro de cada fase quando o DAG permitir).

---

## Continuidade — Roadmap Pós-pesquisa

| PRD | Escopo | Esforço | Risco | Dependências |
|---|---|---|---|---|
| **prd-claude-cli-acp-2026 (F2)** | MCP `run_agent` + tool-call normalization | Médio (1 sprint) | Médio (MCP SDK) | ADR-013 entregue |
| **prd-claude-cli-acp-2026 (F3)** | Hooks in-process + 2-tier memory | Médio (1.5 sprints) | Médio (migração shell→Go) | F2 entregue |
| **prd-claude-cli-acp-2026 (F4)** | Evidence enrichment Claude-2026 | Baixo (3 dias) | Baixo | F3 entregue |
| **prd-claude-cli-acp-2026 (F5)** | Auto-review opt-in | Baixo (3 dias) | Baixo–Médio (custo tokens) | F3 entregue |

Sugestão de decomposição: **um único PRD** com RF-N cobrindo F2..F5 e TechSpec subdividindo em waves de tasks. Alternativa: quatro PRDs sequenciais. A decisão fica para o autor do PRD após aprovação deste pacote.

---

## Referências Cruzadas

**Compozy (leitura via `gh` — SHA `7f38c44506`)**:

- `go.mod` — pin `github.com/coder/acp-go-sdk v0.6.3`
- `internal/core/agent/client.go` — `Client.CreateSession`, `SessionRequest`
- `internal/core/agent/acp_convert.go` — `convertACPToolCallStart`, `convertACPToolCallUpdateHeader`
- `internal/core/agent/tool_call_input.go` — `buildNormalizedToolUseBlock`, `normalizeACPToolInput`
- `internal/core/agent/session.go` — `Session` interface com backpressure
- `internal/core/agent/registry_specs.go` — Spec Claude (template), Codex, Copilot
- `internal/core/agents/agents.go` — Agent discovery e validation
- `internal/core/agents/session_mcp.go` — `BuildSessionMCPServers`, `NestedExecutionContext`
- `internal/core/agents/mcpserver/server.go` — `reservedToolName = "run_agent"`
- `internal/core/agents/mcpserver/engine.go` — Nested session spawn
- `internal/core/prompt/common.go` — `BuildSystemPromptAddendum`, hooks `prompt.pre_build`/`prompt.post_build`
- `internal/core/prompt/templates.go` — Skill renderer
- `internal/core/memory/store.go` — Memória 2-tier (limites 150/12KB e 200/16KB)
- `internal/core/run/executor/review_hooks.go` — `afterTaskJobSuccess`, hooks review
- `internal/core/run/journal/` — Event journal append-only
- `internal/core/kernel/` — Hook dispatcher tipado
- `pkg/compozy/events/events.go` — Eventos públicos tipados
- `pkg/compozy/runs/reader.go` — Run reader (attach/watch/replay)
- `skills/embed.go` — Skills embedded via `go:embed`
- `CLAUDE.md` (raiz do Compozy) — Exemplo de governance file

**ai-spec-harness (estado em `feat/codex-acp-spec`)**:

- `internal/runtime/specs/spec.go:5-101` — `Spec`, `AccessMode`, `BootstrapArgsFunc` (já estendidos por ADR-013)
- `internal/runtime/specs/claude.go:28-45` — `Claude()` (estável, sem mudanças F2+)
- `internal/runtime/specs/codex.go` — referência de runtime com `BootstrapArgs` dinâmico
- `internal/runtime/runner.go:89-220` — `ACPRunner.Run()` (alvo F2-F5)
- `internal/runtime/client/client.go` — Cliente ACP (estável)
- `internal/runtime/events/convert.go` — alvo F2 (normalização) e F4 (cache/thinking)
- `internal/runtime/probe/probe.go` — Probe (estável)
- `internal/runtime/persistence/session.go` — Forense (preservar)
- `internal/runtime/watchdog.go` — `ActivityWatchdog` (preservar)
- `internal/agents/registry.go` — Registry F1 (ADR-011) consumido por MCP `run_agent`
- `internal/specdrift/specdrift.go` — `spec-hash` validation (preservar como concern de governança)
- `internal/evidence/evidence.go:29-49` — alvo F4 (campos Claude-2026)
- `internal/telemetry/` — Telemetria opt-in ADR-006 (estender para cache/thinking)
- `internal/wrapper/wrapper.go:14-18` — `ValidTools` (**não alterar**: `wrapper_test.go:213` documenta que Claude usa hooks, não wrapper)
- `cmd/ai_spec_harness/task_loop.go` — Flags `--mcp-nested`, `--auto-review`, `--memory-*` (F2/F3/F5)
- `.claude/hooks/` — Shell hooks atuais (coexistem com Go hooks após F3)
- `.claude/agents/` — Agentes declarativos (resolvidos via registry após F2)
- ADR-009 (`tasks/adr/009-acp-protocol-adoption.md`) — Pinning SDK (precedente)
- ADR-011 (`tasks/adr/011-agent-registry-declarativo.md`) — Registry F1
- ADR-012 (`tasks/adr/012-copilot-cli-acp-native.md`) — Copilot ACP (paridade)
- ADR-013 (`tasks/adr/013-codex-cli-acp-native.md`) — Codex ACP (paridade; `BootstrapArgs` infra reusada)
- ADR-014 (`tasks/adr/014-claude-cli-acp-native.md`) — **este pacote**

**Pesquisa correlata**:

- [`compozy-adaptation-analysis.md`](compozy-adaptation-analysis.md) — análise genérica 10 dimensões
- [`compozy-adaptation-codex-2026.md`](compozy-adaptation-codex-2026.md) — F1-Codex entregue
- [`compozy-adaptation-copilot-2026.md`](compozy-adaptation-copilot-2026.md) — F1-Copilot entregue
- [`docs/prompts/compozy-adaptation-research-claude.md`](../prompts/compozy-adaptation-research-claude.md) — prompt enriquecido de origem

---

## Apêndice — Comparação Claude vs Codex vs Copilot (visão interna do compozy)

| Aspecto | Claude | Codex | Copilot |
|---|---|---|---|
| **Command** | `claude-agent-acp` | `codex-acp` | `copilot` |
| **FixedArgs** | (vazio) | (vazio; usa BootstrapArgs) | `["--acp"]` |
| **BootstrapArgs** | `nil` | `codexBootstrapArgs` | `nil` |
| **DefaultModel** | `opus` (heredado) | `gpt-5.5` | (runtime-determined) |
| **Reasoning Effort** | n/a (Claude usa thinking_tokens) | configurável via `-c` | n/a |
| **Sandbox** | `--bypass-permissions` | `approval_policy`, `sandbox_mode` (em full) | nenhum |
| **Thinking/Reasoning tokens** | ✅ via `thinking_tokens` (F4-Claude) | n/a | n/a |
| **Cache (prompt caching)** | ✅ `cache_read_tokens`, `cache_creation_tokens` (F4-Claude) | n/a | n/a |
| **Hooks ecosystem** | `.claude/hooks/*.sh` (modo interativo) + Go in-process (F3-Claude) | nenhum específico | nenhum específico |
| **Fallback Package** | `@agentclientprotocol/claude-agent-acp` | `@zed-industries/codex-acp` | `@github/copilot` |
| **No `wrapper.ValidTools`** | ❌ deliberado (usa hooks) | ✅ | ✅ |

**Conclusão do apêndice**: Claude é o runtime mais maduro no harness (entregue desde ADR-009, com hooks shell já em produção). O gap 2026 não é de runtime — é de paridade funcional com o stack que Compozy construiu acima da camada ACP. F2 → F5 fecham esse gap incrementalmente, preservando os diferenciais do harness (`spec-hash`, `events.jsonl`, `tool_calls.md`, `ActivityWatchdog`, telemetria opt-in).
