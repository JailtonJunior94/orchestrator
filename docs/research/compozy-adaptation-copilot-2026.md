# Análise de Adaptação ao Padrão Compozy — Foco Copilot CLI 2026

> **Status**: Pesquisa concluída — base para PRD `prd-copilot-acp-spec` (Fase 1) e PRDs futuros (F2–F4)
> **Data**: 2026-05-21
> **Fonte primária (Compozy)**: leitura via `gh` do repositório [`compozy/compozy`](https://github.com/compozy/compozy) — branch `main` SHA `7f38c445069bd83a8e96bcd925ee1f12fde74435`
> **Fonte primária (harness)**: árvore atual de `/Users/jailtonjunior/Git/orchestrator` na branch `feat/acp-runtime-claude`
> **Pesquisa correlata**: [`docs/research/compozy-adaptation-analysis.md`](compozy-adaptation-analysis.md) (análise genérica, 10 dimensões)
> **Prompt de origem**: [`docs/prompts/compozy-adaptation-research-copilot.md`](../prompts/compozy-adaptation-research-copilot.md)

---

## Sumário Executivo

O `compozy/compozy` resolveu em 2026 a limitação que motivou nossa [ADR-007](../adr/007-copilot-cli-stateless-workaround.md) ("Copilot CLI stateless workaround"): o `copilot` CLI passou a expor um servidor ACP nativo (`copilot --acp`) com fallback `npx --yes @github/copilot --acp`. No catálogo `internal/core/agent/registry_specs.go:222-242` do compozy, Copilot está registrado em pé de igualdade com Claude/Codex/Gemini/Cursor/Droid/OpenCode/Pi — mesma estrutura `Spec`, mesmo subprocess JSON-RPC, mesma observabilidade.

A consequência para o `ai-spec-harness` é direta: **a Fase 1 ("Copilot ACP Spec") é a alavanca de maior valor por menor esforço em 2026**. Reutilizando o stack já validado para Claude (`ACPRunner`, `acpClient`, persistência forense, `ActivityWatchdog`, telemetria opt-in), o Copilot deixa de ser caixa-preta (CLI legado via `copilotInvoker`) e ganha paridade observacional total. ADR-007 é formalmente substituída por [ADR-012](../../tasks/adr/012-copilot-cli-acp-native.md).

Esta pesquisa apresenta sete dimensões comparativas com `file:line` reais de ambos os repos, um roadmap de quatro fases (F1 Copilot ACP Spec, F2 Memória 2-níveis, F3 Hook System Go in-process, F4 TUI Bubble Tea + daemon) e exemplos de configuração 2026 para `COPILOT.md` e `.github/copilot-instructions.md`. Correções factuais à literatura interna são documentadas na §"Correções ao prompt original": o sistema de hooks do compozy **não usa JSON-RPC 2.0 wire protocol**, são 33 nomes canônicos (não 32) despachados in-process Go; a compactação de memória é **prompt-driven** (máquina mede, modelo decide), não código-driven; o MCP server reservado expõe **apenas** `run_agent`, não memory/registry tools.

A recomendação operacional é executar F1 nesta sessão (PRD/TechSpec/Tasks já em `tasks/prd-copilot-acp-spec/`) e tratar F2–F4 como PRDs futuros independentes com escopo bruto.

---

## Correções ao prompt original

O prompt em `docs/prompts/compozy-adaptation-research-copilot.md` carrega três premissas que esta pesquisa precisou corrigir contra o código real do compozy (validado via `gh api`):

1. **"Hooks baseados em subprocessos (JSON-RPC 2.0) cobrindo 32 pontos de ciclo de vida"** — Os hooks do compozy **são despachados in-process em Go** via `RuntimeManager.DispatchMutableHook(ctx, name, payload) (any, error)` e `DispatchObserverHook(...)` declarados em `internal/core/model/hooks.go`. São **33 nomes canônicos** (não 32) declarados em `internal/core/extension/manifest.go`. Extensões externas são subprocessos gerenciados pelo `extension.Manager`, mas o ponto de extensão lógico é a chamada Go — não há mensagem JSON-RPC 2.0 entre o motor e os hooks internos.

2. **"Memória de dois níveis com compactação automática"** — Os limites (workflow 150 linhas / 12 KiB; task 200 linhas / 16 KiB) existem e estão em `internal/core/memory/store.go`, mas o store **não compacta automaticamente**. Calcula `NeedsCompaction bool` e o builder de prompt em `internal/core/prompt/prd.go` injeta diretiva "compact the flagged memory files before proceeding" no system prompt. O LLM (via skill `cy-workflow-memory`) executa a compactação reescrevendo o arquivo. Não há arquivo `.archive`; é replace in-place.

3. **"Sistema de hooks com 32 pontos"** — Recontado a partir do manifesto canônico: 33 hooks distribuídos em sete famílias (plan 7, prompt 3, agent 5, job 3, run 4, review 9, artifact 2). Capability gating (`prompt.mutate`, `plan.mutate`, `memory.read`, etc.) controla quais payloads cada extensão pode tocar.

4. **"Servidor MCP reservado para orquestração"** — Confirmado em `internal/core/agents/mcpserver/server.go`, mas a tool exposta é **única**: `run_agent`, usada para hand-off recursivo agente-pai → agente-filho. Não há tools de memory/registry/skills nesse MCP.

5. **"Foco no copilot-cli em 2026"** — Confirmado plenamente. Copilot é runtime ACP nativo em `registry_specs.go:222-242` e está documentado em https://docs.github.com/en/copilot/reference/copilot-cli-reference/acp-server.

Essas correções não invalidam o roadmap proposto pelo prompt; apenas alinham as recomendações ao código-fonte real do compozy.

---

## Achados por Dimensão

### 1. Copilot CLI como runtime ACP nativo (Compozy ✅ | harness ❌)

**Compozy** — `internal/core/agent/registry_specs.go:222-242`:
```go
model.IDECopilot: {
    ID: model.IDECopilot, DisplayName: "Copilot CLI",
    SetupAgentName: "github-copilot",
    DefaultModel: model.DefaultCopilotModel,
    Command: "copilot",
    FixedArgs: []string{"--acp"},
    ProbeArgs: []string{"--acp", "--help"},
    Fallbacks: []Launcher{{Command: "npx",
        FixedArgs: []string{"--yes", "@github/copilot", "--acp"}, ...}},
    DocsURL: "https://docs.github.com/en/copilot/reference/copilot-cli-reference/acp-server",
    BootstrapArgs: func(_ string, _ string, _ []string, _ string) []string { return nil },
},
```

Spec idêntico em forma ao registro de Claude/Codex/Gemini/Droid no mesmo arquivo. `BootstrapArgs` retorna `nil` — Copilot não injeta flags por chamada (diferente de Codex/Droid que injetam `-c model=...` ou `--model`). `internal/setup/agents.go` espelha o diretório de skills do harness em `~/.copilot/skills`, indicando que o setup do Copilot esperando esse caminho é parte da convenção do compozy.

**ai-spec-harness** — `internal/taskloop/agent.go:381-388`:
```go
func (c *copilotInvoker) Invoke(ctx context.Context, prompt, workDir, model string) (string, string, int, error) {
    args := make([]string, 0, 6)
    if model != "" { args = append(args, "--model", model) }
    args = append(args, "--autopilot", "--yolo", "-p", prompt)
    return runCmd(ctx, workDir, c.liveOut, "copilot", args...)
}
```

Hoje o Copilot é invocado como CLI stateless via `--autopilot --yolo -p <prompt>`. Stdout é capturado bruto; não há `events.jsonl`, `tool_calls.md`, `execution_report.md`, watchdog ou telemetria. O bloqueio explícito em `cmd/ai_spec_harness/task_loop.go:77` confirma a assimetria:

```go
if runtime == "acp" {
    effectiveTool := tool
    if effectiveTool == "" { effectiveTool = execTool }
    if effectiveTool != "claude" {
        _, _ = fmt.Fprintf(os.Stderr, "runtime acp suporta apenas --tool claude nesta versão\n")
        return fmt.Errorf("exit2")
    }
}
```

**Impacto**: Copilot é caixa-preta. Telemetria opt-in (ADR-006), persistência forense (ADR-009/010) e paridade multi-tool (ADR-008) não cobrem Copilot.

**Gap técnico**: criar `internal/runtime/specs/copilot.go` espelhando `claude.go:24-42`, generalizar `internal/runtime/runner.go:113-120` e `internal/runtime/probe/probe.go:70-82` para deixar de hardcodar `specs.ClaudeNpmPackage`/`ClaudeNpmVersion`, e remover o gating em `task_loop.go:77` para aceitar Copilot.

---

### 2. MCP server reservado para hand-off (Compozy ✅ | harness ❌, baixa prioridade)

**Compozy** — `internal/core/agents/mcpserver/server.go`:
```go
const (
    reservedToolName        = "run_agent"
    reservedToolDescription = "Run a reusable Compozy agent by name and return its structured result."
)
func (s *Server) RunStdio(ctx context.Context, host HostContext) error {
    server := mcp.NewServer(s.impl(), nil)
    mcp.AddTool(server, &mcp.Tool{Name: reservedToolName, ...}, s.runAgentTool(host))
    return server.Run(ctx, &mcp.StdioTransport{})
}
```

`internal/core/agents/agents.go` declara `ReservedMCPServerName = "compozy"` — nenhum agente pode sobrescrever este server em seu próprio `mcp.json`. Host context é passado por env `COMPOZY_RUN_AGENT_CONTEXT` (JSON). Engine resolve agente-filho via `reusableagents.ResolveExecutionContext`, valida `Depth < MaxDepth` e `cycleDetected(name, AgentPath)`, então executa nova sessão ACP isolada (`execpkg.ExecutePreparedPrompt`).

**ai-spec-harness**: não expõe MCP. O harness é single-shot por design hoje; hand-off recursivo entre agentes não é cenário previsto.

**Impacto**: ausência impede que um agente sugira "rode `claude-revisor-rigoroso` em seguida" como ação executável dentro da mesma sessão. Para o uso atual do harness (loop sequencial de tasks com um agente por vez) isso é aceitável.

**Gap técnico**: criar `internal/mcp/server/` espelhando `mcpserver/server.go` é trabalho de F-futura (não F1–F4 deste roadmap). Depende de F1 (Agent Registry — já entregue em [`tasks/prd-agent-registry-declarativo/`](../../tasks/prd-agent-registry-declarativo/)) para que `run_agent` tenha algo para resolver.

---

### 3. TUI Bubble Tea + AttachRemote a daemon (Compozy ✅ | harness ❌)

**Compozy** — `go.mod` declara `charm.land/bubbletea/v2 v2.0.2`. Estrutura em `internal/core/run/ui/` (19 arquivos):

- `model.go`, `view.go`, `update.go` — MVU clássico do Bubble Tea
- `sidebar.go`, `timeline.go`, `layout.go`, `summary.go`, `validation_form.go` — componentes
- `remote.go` — anexação a sessão remota
- `styles.go`, `adapter_test.go`

`internal/core/run/ui/remote.go`:
```go
// AttachRemote boots the Bubble Tea cockpit from a daemon snapshot and then follows the daemon stream.
func AttachRemote(ctx context.Context, opts RemoteAttachOptions) (Session, error) {
    jobs, initialMsgs := remoteSnapshotBootstrap(opts.Snapshot)
    session := setupRemoteUISession(ctx, jobs, &localCfg, nil, true)
    ...
    if err := ensureInitialRemoteStream(ctx, opts, session); err != nil { ... }
}
```

`internal/cli/run_observe.go` orquestra `attachCLIRunUI`, `attachStartedCLIRunUI` e expõe `openCLIRemoteUISession = uipkg.AttachRemote`. Cursor de stream (`apicore.StreamCursor`) permite retomada após reconexão (`remoteReconnectDelay = 100ms`). Há jornal append-before-publish em `internal/core/run/journal` que garante que eventos não se perdem mesmo se o cliente UI desconectar e voltar.

**ai-spec-harness** — `internal/runtime/render/human.go`:
```go
type HumanRenderer struct { out io.Writer }
func (r *HumanRenderer) Render(evt events.Event) {
    switch evt.Kind() {
    case events.KindAgentMessage: fmt.Fprintf(r.out, "[agent] %s\n", p.Text())
    ...
}
```

Renderer texto plano para stdout. Suprimível por `--quiet`. Sem TUI, sem daemon, sem reconexão, sem cursor de stream.

**Impacto**: monitoramento em tempo real de sessões longas é fraco. Não dá para "anexar" a uma sessão Copilot em curso de outra janela.

**Gap técnico**: F4 do roadmap. Alto esforço — exige daemon HTTP, journaling antes de publish e UI client em Bubble Tea. Bom como visão de longo prazo, mas só faz sentido após F1 (Copilot via ACP) para que haja eventos Copilot a renderizar.

---

### 4. Sistema de hooks (Compozy ✅ Go in-process | harness ❌)

**Compozy** — `internal/core/extension/manifest.go` declara 33 nomes canônicos distribuídos em sete famílias:

| Família | Quantidade | Nomes |
|---|---|---|
| Plan | 7 | `plan.pre_discover`, `plan.post_discover`, `plan.pre_group`, `plan.post_group`, `plan.pre_prepare_jobs`, `plan.pre_resolve_task_runtime`, `plan.post_prepare_jobs` |
| Prompt | 3 | `prompt.pre_build`, `prompt.post_build`, `prompt.pre_system` |
| Agent | 5 | `agent.pre_session_create`, `agent.post_session_create`, `agent.pre_session_resume`, `agent.on_session_update`, `agent.post_session_end` |
| Job | 3 | `job.pre_execute`, `job.post_execute`, `job.pre_retry` |
| Run | 4 | `run.pre_start`, `run.post_start`, `run.pre_shutdown`, `run.post_shutdown` |
| Review | 9 | `review.pre_fetch`, `review.post_fetch`, `review.pre_batch`, `review.post_fix`, `review.pre_resolve`, `review.watch_pre_round`, `review.watch_post_round`, `review.watch_pre_push`, `review.watch_finished` |
| Artifact | 2 | `artifact.pre_write`, `artifact.post_write` |

`internal/core/model/hooks.go` expõe dois despachadores:
```go
func DispatchMutableHook[T any](ctx, manager, hook string, payload T) (T, error)   // pode alterar payload
func DispatchObserverHook(ctx, manager, hook string, payload any)                  // fire-and-forget
func WaitForObserverHooks(ctx, manager) error                                       // drain antes do shutdown
```

Capabilities (declaradas no manifesto TOML da extensão): `prompt.mutate`, `plan.mutate`, `agent.mutate`, `job.mutate`, `run.mutate`, `review.mutate`, `memory.read`, `memory.write`.

`internal/core/extension/manager.go` carrega manifestos TOML, faz spawn de subprocesso via `internal/core/subprocess`, mantém `dispatcher` com priority chain e fan-out, com timeouts (`defaultExtensionHookTimeout = 5s`, `extensionEventQueueCap = 256`).

**ai-spec-harness** — `.agents/hooks/` contém shell scripts ad-hoc (`pre-execute-all-tasks.sh`, `post-execute-task.sh`, `post-wave.sh`, `subagent-stop-wrapper.sh`). Não há sistema de hooks tipado em Go nem dispatcher com priority chain.

**Impacto**: customização exige fork de templates embed ou modificação direta dos shell scripts. Sem isolamento de payload mutável vs observador.

**Gap técnico**: F3 do roadmap. Esforço médio. Decisão de design importante: adotar dispatcher Go in-process com 33 hooks canônicos (não JSON-RPC). Capability gating via manifesto TOML é a interface estável para extensões externas (subprocess), mas o ponto de extensão lógico em código é Go.

---

### 5. Pipeline 6-stages — convenção via skills, não tipo (Compozy ⚖️ | harness ⚖️)

**Compozy** — não modela as fases (Idea → PRD → TechSpec → Tasks → Implementation → Validation) como tipo no código. Modela como **skills bundled**:

```
skills/
├── cy-create-prd
├── cy-create-techspec
├── cy-create-tasks
├── cy-execute-task
├── cy-fix-reviews
├── cy-review-round
├── cy-final-verify
├── cy-workflow-memory
└── compozy
```

O dispatcher de execução (`internal/core/run/executor/runner.go`) trata cada task como `model.IssueEntry` em modo `model.ExecutionModePRDTasks`. `internal/core/prompt/common.go` só conhece dois modos formais:

```go
if p.Mode == model.ExecutionModePRDTasks {
    rendered = buildPRDTasksPrompt(p)
} else {
    rendered = buildCodeReviewPrompt(p)
}
```

A pipeline de 6 fases é **convenção**, não tipo enumerado.

**ai-spec-harness** — `.agents/skills/` já cobre as mesmas fases via skills `create-prd`, `create-technical-specification`, `create-tasks`, `execute-task`, `review`, `bugfix`, `refactor`, `analyze-project`. A convenção está em pé de igualdade com compozy.

**Impacto**: paridade conceitual já existe. Nenhum gap a fechar nesta dimensão.

**Gap técnico**: nenhum. Marca de checkbox para o prompt original; concretizada via skills declaradas, não código.

---

### 6. Memória 2-níveis prompt-driven (Compozy ✅ | harness ❌)

**Compozy** — `internal/core/memory/store.go`:
```go
const (
    workflowLineLimit = 150
    workflowByteLimit = 12 * 1024
    taskLineLimit     = 200
    taskByteLimit     = 16 * 1024
)
type FileState struct {
    Path string; LineCount int; ByteCount int
    NeedsCompaction bool
}
```

Write é atômico via tmpfile + rename. Diretório padrão: `<tasksDir>/memory/`. Templates bootstrap:

- **Workflow** (`MEMORY.md`): `## Current State` / `## Shared Decisions` / `## Shared Learnings` / `## Open Risks` / `## Handoffs`
- **Task** (`task_<N>.md`): `## Objective Snapshot` / `## Important Decisions` / `## Learnings` / `## Files / Surfaces` / `## Errors / Corrections` / `## Ready for Next Run`

Compactação não é código-driven. `internal/core/prompt/prd.go` injeta diretiva no system prompt quando flags estão setadas:

```go
if memory.WorkflowNeedsCompaction || memory.TaskNeedsCompaction {
    sb.WriteString("- Compact the flagged memory files before proceeding with implementation.\n")
    if memory.WorkflowNeedsCompaction {
        fmt.Fprintf(&sb, "- Shared workflow memory is over its soft limit: `%s`\n", memory.WorkflowPath)
    }
    ...
}
```

O LLM lê essa diretiva (via skill `cy-workflow-memory`) e executa a compactação reescrevendo o arquivo. **Princípio**: máquina mede, modelo decide o que vale preservar.

**ai-spec-harness**: não há equivalente. Contexto entre runs depende de releitura de `prd.md`/`techspec.md`/`tasks.md` a cada invocação. Aprendizados acumulados não são persistidos.

**Impacto**: agente não acumula conhecimento entre tasks/runs. Cada invocação parte do zero exceto pelo que está no PRD.

**Gap técnico**: F2 do roadmap. Esforço baixo — ~250 LOC em `internal/memory/store.go` espelhando `store.go` do compozy + injeção da diretiva de compactação no prompt builder. Decisão de design importante: manter o princípio prompt-driven (não code-driven). Coexiste com persistência forense (não substitui `events.jsonl`).

---

### 7. `COPILOT.md` / `.github/copilot-instructions.md` (Compozy ❌ | harness ✅)

**Compozy** — `gh api repos/compozy/compozy/contents/.github` confirma: não há `copilot-instructions.md`. Raiz do repo tem `CLAUDE.md` (governança canônica) e `AGENTS.md`, mas **não** `COPILOT.md`. O suporte a Copilot é runtime-only (registro de Spec) — o compozy não dedica template de instruções a Copilot.

`internal/setup/agents.go` apenas espelha o diretório de skills em `~/.copilot/skills` para que o Copilot CLI consuma como bibliotec.

**ai-spec-harness** — possui `COPILOT.md` raiz documentando o workaround stateless de ADR-007:
```
### gh copilot CLI — Sem Suporte Nativo
O gh copilot CLI **não lê** .github/copilot-instructions.md nem qualquer arquivo de contexto automaticamente. Cada invocação é stateless.
Workaround: gh copilot suggest "$(cat .github/copilot-instructions.md)\n\nTarefa: ..."
```

**Impacto**: o `COPILOT.md` do harness vira obsoleto após F1. Documenta uma limitação que deixou de ser verdade em 2026.

**Gap técnico**: parte da F1 — `COPILOT.md` precisa ser reescrito para documentar o caminho ACP nativo. Ver §"Exemplos de Configuração 2026" abaixo.

---

## Gap Map Consolidado

Legenda: 🟢 implementado · 🟡 parcial · 🔴 ausente · ⭐ vantagem do harness a preservar

| # | Feature | Status Orchestrator | Padrão Compozy | Gap Técnico | Fase |
|---|---|---|---|---|---|
| 1 | Copilot via ACP nativo | 🔴 CLI stateless (ADR-007) | 🟢 `copilot --acp` + npx fallback (`registry_specs.go:222-242`) | Criar `specs/copilot.go`; generalizar `runner.go:113-120` e `probe/probe.go:70-82`; remover gating em `task_loop.go:77` | **F1** |
| 2 | MCP server reservado `run_agent` | 🔴 | 🟢 `mcpserver/server.go` com tool única `run_agent` | Pacote `internal/mcp/server/` (depende de Agent Registry já entregue) | F-futura |
| 3 | TUI Bubble Tea + AttachRemote | 🔴 `HumanRenderer` stdout | 🟢 cockpit + daemon + stream cursor (`run/ui/remote.go`) | Daemon HTTP + journal append-before-publish + UI client | F4 |
| 4 | Hooks 33-canon Go in-process | 🔴 shell scripts ad-hoc em `.agents/hooks/` | 🟢 `DispatchMutableHook`/`DispatchObserverHook` + capabilities + manifesto TOML | `internal/extension/{manager,dispatcher,manifest}` + 33 dispatch points | F3 |
| 5 | Pipeline 6-stages via skills | 🟢 skills em `.agents/skills/` | 🟢 skills em `skills/cy-*` | Paridade conceitual já existe | — |
| 6 | Memória 2-níveis prompt-driven | 🔴 | 🟢 `memory/store.go` + injeção no prompt (`prd.go`) | `internal/memory/store.go` ~250 LOC + hint no prompt builder | F2 |
| 7 | `COPILOT.md` / `.github/copilot-instructions.md` | 🟡 documenta workaround ADR-007 obsoleto | 🔴 não existe | Reescrever `COPILOT.md` (parte de F1) | **F1** |
| 8 | Persistência forense (`events.jsonl`, `tool_calls.md`, `execution_report.md`) | 🟢 ⭐ | 🔴 | Preservar | — |
| 9 | ActivityWatchdog com `CancelCause` | 🟢 ⭐ | 🟡 sem watchdog dedicado | Preservar | — |
| 10 | Telemetria opt-in (ADR-006) | 🟢 ⭐ | 🟡 OTel/Grafana mais sofisticado | Manter; reavaliar OTel em F4 | — |

**Critérios de esforço**: Baixo ≤ 1 sprint; Médio 1–2 sprints; Alto ≥ 2 sprints com pesquisa.

---

## Roadmap de Adaptação (4 fases)

### F1 — Copilot ACP Spec (esta entrega)

**Escopo**:
- Novo `internal/runtime/specs/copilot.go` espelhando `claude.go:24-42`:
  - `Command: "copilot"`, `FixedArgs: []string{"--acp"}`, `ProbeArgs: []string{"--acp", "--help"}`
  - Fallback único `npx --yes @github/copilot --acp` (versão npm pinada em constante Go, padrão ADR-009)
  - `AccessModeFlag` apropriado quando documentado (sem `--bypass-permissions` análogo no v0)
- Generalizar `internal/runtime/runner.go:113-120`: deixar de hardcodar `specs.ClaudeSDKVersion`/`ClaudeNpmVersion`. Versões passam a ser metadata do `Spec` (método `Spec.SDKVersion()` / `Spec.NPMVersion()`).
- Generalizar `internal/runtime/probe/probe.go:70-82`: template de erro consome metadata do `Spec` recebido.
- Remover gating em `cmd/ai_spec_harness/task_loop.go:77`: aceitar qualquer `Spec` registrado. Modelo: tabela `runtimeACPCatalog map[string]specs.Spec`.
- Atualizar `internal/runtime/acp_integration_test.go` com matriz Copilot.
- Manter `copilotInvoker` CLI legado por uma versão (rota de compatibilidade com aviso de depreciação).
- Reescrever `COPILOT.md` raiz removendo bloco de workaround stateless e adicionando seção "Copilot via ACP".
- ADR-012 ([`tasks/adr/012-copilot-cli-acp-native.md`](../../tasks/adr/012-copilot-cli-acp-native.md)) substitui formalmente ADR-007.

**Esforço**: Baixo. **Risco**: Baixo (additivo; flag `--runtime=acp` ainda é opt-in).

**Dependências**: nenhuma. ADR-007 supersedure é a única decisão prévia.

**Pré-requisito de viabilidade**: confirmar versão mínima do Copilot CLI que expõe `--acp`. Documentar na techspec.

### F2 — Memória 2-níveis prompt-driven

**Escopo**:
- `internal/memory/store.go` espelhando `internal/core/memory/store.go` do compozy:
  - Limites `workflowLineLimit = 150`, `workflowByteLimit = 12 * 1024`, `taskLineLimit = 200`, `taskByteLimit = 16 * 1024`
  - Write atômico (tmpfile + rename)
  - Bootstrap templates para workflow e task
  - `FileState{Path, LineCount, ByteCount, NeedsCompaction bool}`
- Injeção de hint no prompt builder (`internal/taskloop/agent.go` ou nova função adjacente) quando `NeedsCompaction == true`.
- Coexistência total com `events.jsonl`/`tool_calls.md`/`execution_report.md` — memory-first não substitui forense.
- Skill nova `harness-workflow-memory` (espelha `cy-workflow-memory`) que o LLM invoca para compactar.

**Esforço**: Baixo (~250 LOC + skill). **Risco**: Baixo (additivo, não toca runtime ACP).

**Dependências**: nenhuma. F1 facilita medição de impacto via telemetria, mas não bloqueia.

**Riscos**:
- Memory bloat se compactação for negligenciada pelo LLM — testes de carga obrigatórios
- Decisão de design: manter princípio prompt-driven (máquina mede, modelo decide). Não code-compaction.

### F3 — Hook System Go in-process (33 canônicos)

**Escopo**:
- `internal/extension/{manifest,manager,dispatcher}.go` espelhando `internal/core/extension/` do compozy
- `internal/core/model/hooks.go` equivalente: `DispatchMutableHook[T any]` + `DispatchObserverHook` + `WaitForObserverHooks`
- 33 dispatch points distribuídos nas sete famílias (plan/prompt/agent/job/run/review/artifact)
- Capability gating via manifesto TOML (`prompt.mutate`, `plan.mutate`, `memory.read`, etc.)
- Timeout default 5s por hook; queue cap 256 eventos
- Migrar `.agents/hooks/` (shell) para dispatcher novo: scripts shell viram extensões subprocess gerenciadas pelo manager

**Esforço**: Médio. **Risco**: Médio (surface area grande; capability gating crítico para segurança).

**Dependências**: F1 entregue (para que hooks `agent.*` tenham eventos Copilot a observar). F2 facilita hooks `memory.*` mas não bloqueia.

**Decisão de design**: dispatcher é Go in-process, não JSON-RPC. Extensões externas são subprocesses padrão (transporte estruturado via `internal/subprocess`, não JSON-RPC ACP).

### F4 — TUI Bubble Tea + daemon

**Escopo**:
- Daemon HTTP que mantém sessões ACP em background (`ai-spec-harness daemon start`)
- Journal append-before-publish (`internal/run/journal`) garantindo que eventos não se perdem em desconexão
- UI client em `internal/ui/` usando `charm.land/bubbletea/v2` espelhando `internal/core/run/ui/`:
  - `model.go`, `view.go`, `update.go`, `sidebar.go`, `timeline.go`, `layout.go`, `summary.go`
  - `remote.go` com `AttachRemote` + cursor de stream + reconnect delay
- Comando `ai-spec-harness attach <run-id>` (espelha `compozy run attach`)

**Esforço**: Alto. **Risco**: Médio-Alto (daemon adiciona estado persistente; protocolo HTTP de sessões precisa de versionamento).

**Dependências**: F1 (para haver eventos Copilot a renderizar). F2 facilita summary view. F3 facilita hooks `run.*` para integrar com daemon lifecycle.

**Pré-requisito**: decisão sobre dependência `charm.land/bubbletea/v2` (não é dependência atual; adicionar ao `go.mod`).

---

## Exemplos de Configuração 2026

### `COPILOT.md` raiz — reescrita F1

```markdown
# Copilot — ai-spec-harness

## Modo Recomendado (2026): Copilot via ACP

Em 2026 o GitHub Copilot CLI passou a expor servidor ACP nativo (`copilot --acp`).
O harness suporta esse modo via `--runtime=acp --tool=copilot`.

### Pré-requisitos
- `copilot` CLI versão >= X.Y.Z (verifique com `copilot --version`)
- `gh auth status` deve mostrar token Copilot válido
- Alternativa: `npx --yes @github/copilot@<pin> --acp` (fallback automático)

### Uso
\`\`\`bash
ai-spec-harness task-loop \
  --tool copilot \
  --runtime acp \
  tasks/prd-minha-feature
\`\`\`

A sessão produz os mesmos artefatos forenses do modo Claude:
- `events.jsonl` (linha-a-linha de eventos ACP)
- `tool_calls.md` (agregado de tool calls)
- `execution_report.md` (summary final)

Telemetria opt-in (`GOVERNANCE_TELEMETRY=1`) registra invocações com `tool=copilot`.

## Modo Legado (deprecado, será removido em vX): Copilot CLI stateless

\`\`\`bash
ai-spec-harness task-loop \
  --tool copilot \
  tasks/prd-minha-feature
\`\`\`

Este modo invoca `copilot --autopilot --yolo -p <prompt>` sem ACP.
Não produz `events.jsonl` nem `tool_calls.md`. Mantido por compatibilidade
até versão vX (ver ADR-012 §"Consequências").

## ADRs Relevantes

- [ADR-012](tasks/adr/012-copilot-cli-acp-native.md) — Copilot via ACP nativo (substitui ADR-007)
- [ADR-007](docs/adr/007-copilot-cli-stateless-workaround.md) — workaround histórico (substituído)
- [ADR-009](tasks/adr/009-acp-protocol-adoption.md) — pinning de SDK ACP
- [ADR-008](docs/adr/008-parity-multi-tool-invariants.md) — invariantes de paridade multi-tool
```

### `.github/copilot-instructions.md` — manter funcional

Compozy não usa este arquivo. No harness, mantém-se funcional para Copilot Chat no editor (cenário não-CLI). Conteúdo mínimo recomendado em 2026:

```markdown
# Copilot Instructions — ai-spec-harness

> Este arquivo é consumido por Copilot Chat (IDE). Para Copilot CLI em modo ACP,
> as regras vêm de PRD/TechSpec/Tasks e dos templates em `internal/taskloop/executor_template.tmpl`.

## Convenções

- Linguagem: Go (>= 1.22)
- Arquitetura: ver `AGENTS.md`
- Governança: ver `.claude/rules/governance.md`
- Skills procedurais: `.agents/skills/`
- ADRs: `docs/adr/` e `tasks/adr/`

## Pipeline

Use as skills declaradas em `.agents/skills/` para cada fase:
- create-prd → create-technical-specification → create-tasks → execute-task → review

## Validação

Antes de propor mudanças, rode validações proporcionais ao risco:
- `go build ./...`
- `go vet ./...`
- `go test ./internal/...`
```

### Exemplo de Spec Copilot esperado (`internal/runtime/specs/copilot.go` — F1)

```go
package specs

const (
    CopilotNpmPackage = "@github/copilot"
    CopilotNpmVersion = "0.1.0"          // pinada via processo audit/
    CopilotSDKVersion = "v0.13.0"         // sincronizada via go.mod (mesma do Claude)
    CopilotMinCLIVersion = "X.Y.Z"        // a confirmar na techspec
)

func Copilot() Spec {
    return newSpec(
        "copilot",
        "GitHub Copilot CLI (ACP)",
        "copilot",
        []string{"--acp"},
        []FallbackLauncher{
            {
                Command:   "npx",
                FixedArgs: []string{"--yes", CopilotNpmPackage + "@" + CopilotNpmVersion, "--acp"},
            },
        },
        "",  // sem flag análoga a --bypass-permissions no v0
    )
}
```

---

## Continuidade — PRDs Futuros

> **Estado em 2026-05-21**: F1 entregue como PRD/TechSpec/Tasks em [`tasks/prd-copilot-acp-spec/`](../../tasks/prd-copilot-acp-spec/) + ADR-012 em [`tasks/adr/012-copilot-cli-acp-native.md`](../../tasks/adr/012-copilot-cli-acp-native.md). F2–F4 abaixo são PRDs futuros independentes.

### F2 — Memória 2-níveis prompt-driven

**Escopo bruto**:
- `internal/memory/store.go` com limites e bootstrap templates
- Hint no prompt builder quando `NeedsCompaction == true`
- Skill `harness-workflow-memory` para o LLM compactar
- Coexistência com persistência forense

**Riscos**: memory bloat se LLM negligenciar compactação. Mitigação: testes de carga + telemetria de `LineCount`/`ByteCount` em `runtime_init`.

**Dependências**: nenhuma bloqueante. F1 facilita medição.

### F3 — Hook System Go in-process

**Escopo bruto**:
- `internal/extension/{manifest,manager,dispatcher}.go`
- 33 dispatch points nas sete famílias
- Capability gating via manifesto TOML
- Migração de `.agents/hooks/` shell → extensões subprocess

**Riscos**: surface area de extensão amplia vetor de ataque — capability gating é crítico. Mitigação: revisão R-SEC-001 obrigatória.

**Dependências**: F1 entregue. F2 facilita hooks `memory.*` mas não bloqueia.

### F4 — TUI Bubble Tea + daemon

**Escopo bruto**:
- Daemon HTTP com journal append-before-publish
- UI client `charm.land/bubbletea/v2` (`model/view/update`)
- `AttachRemote` com stream cursor e reconnect

**Riscos**: daemon introduz estado persistente; protocolo HTTP precisa versionamento. Mitigação: ADR dedicada antes de implementar.

**Dependências**: F1 obrigatória (precisa eventos Copilot/Claude a renderizar). F2 e F3 desejáveis para summary/lifecycle hooks.

---

## Referências Cruzadas

**Compozy (leitura via `gh` — SHA `7f38c44506`)**:
- `internal/core/agent/registry_specs.go:222-242` — Spec Copilot ACP canônica
- `internal/core/agents/mcpserver/server.go` — MCP reservado `compozy`/`run_agent`
- `internal/core/agents/agents.go` — `ReservedMCPServerName = "compozy"`
- `internal/core/memory/store.go` — limites 150/12KiB + 200/16KiB; atomic write
- `internal/core/prompt/prd.go` — injeção de hint de compactação
- `internal/core/prompt/common.go` — modos PRDTasks/CodeReview
- `internal/core/extension/manifest.go` — 33 hooks canônicos
- `internal/core/extension/manager.go` — `extension.Manager` com subprocess + dispatcher
- `internal/core/model/hooks.go` — `DispatchMutableHook[T any]`, `DispatchObserverHook`
- `internal/core/run/ui/remote.go` — `AttachRemote` Bubble Tea
- `internal/cli/run_observe.go` — orquestração `openCLIRemoteUISession`
- `internal/setup/agents.go` — mirror de skills em `~/.copilot/skills`
- `CLAUDE.md` raiz — governança canônica
- `skills/cy-*` — pipeline 6-fases via skills

**ai-spec-harness (estado em `feat/acp-runtime-claude`)**:
- `internal/runtime/specs/claude.go:24-42` — template para `Copilot()`
- `internal/runtime/specs/spec.go:10-37` — value object `Spec` + `newSpec`
- `internal/runtime/specs/launcher.go:13-47` — `Launcher` (binary/npx)
- `internal/runtime/runner.go:113-120` — `runtime_init` hardcoded em Claude (a generalizar)
- `internal/runtime/probe/probe.go:70-82` — erro template Claude-specific (a generalizar)
- `internal/runtime/client/client.go` — `acpClient` agnóstico de IDE (já reusável)
- `internal/runtime/persistence/session.go:13-63` — forense (preservar)
- `internal/runtime/watchdog.go:18-110` — `ActivityWatchdog` (preservar)
- `internal/runtime/events/convert.go:117-197` — conversão SDK→domínio (já reusável)
- `internal/runtime/events/event.go:11-150` — tagged union (ADR-010)
- `internal/runtime/render/human.go:12-30` — `HumanRenderer` stdout
- `internal/taskloop/agent.go:381-388` — `copilotInvoker` CLI legado
- `cmd/ai_spec_harness/task_loop.go:77` — gating `--runtime=acp → claude only`
- `COPILOT.md` — documenta workaround ADR-007 (a reescrever)
- `.agents/hooks/` — shell scripts (referência para migração F3)
- ADR-007 (`docs/adr/007-copilot-cli-stateless-workaround.md`) — substituída por ADR-012
- ADR-009 (`tasks/adr/009-acp-protocol-adoption.md`) — pinning SDK
- ADR-008 (`docs/adr/008-parity-multi-tool-invariants.md`) — paridade multi-tool
- ADR-010 (`tasks/prd-acp-runtime-claude/adr-010-event-tagged-union.md`) — tagged union de eventos
- ADR-011 (`tasks/adr/011-agent-registry-declarativo.md`) — Agent Registry F1 anterior
- [ADR-012](../../tasks/adr/012-copilot-cli-acp-native.md) — Copilot ACP nativo (substitui ADR-007)

**Pesquisa correlata**:
- [`docs/research/compozy-adaptation-analysis.md`](compozy-adaptation-analysis.md) — análise genérica original (10 dimensões, F1–F5 roadmap)
- [`docs/prompts/compozy-adaptation-research-copilot.md`](../prompts/compozy-adaptation-research-copilot.md) — prompt enriquecido de origem
- [`docs/prompts/compozy-adaptation-research-claude.md`](../prompts/compozy-adaptation-research-claude.md) — variante Claude (já consumida)
