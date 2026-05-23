# Análise de Adaptação ao Padrão Compozy — Foco Gemini-CLI 2026

> **Status**: Pesquisa concluída — base para futuro PRD `prd-gemini-cli-acp-2026` (F0-Gemini como pré-requisito, depois F1..F5-Gemini)
> **Data**: 2026-05-22
> **Fonte primária (Compozy)**: leitura via `gh api` do repositório [`compozy/compozy`](https://github.com/compozy/compozy) — branch `main` SHA `7f38c445069bd83a8e96bcd925ee1f12fde74435`
> **Fonte primária (harness)**: árvore atual de `/Users/jailtonjunior/Git/orchestrator` na branch `feat/codex-acp-spec` (commit `822ae74` "feat(codex): implementar F1-Codex via ACP nativo (ADR-013)")
> **Pesquisas irmãs**: [`compozy-adaptation-claude-2026.md`](compozy-adaptation-claude-2026.md), [`compozy-adaptation-codex-2026.md`](compozy-adaptation-codex-2026.md), [`compozy-adaptation-copilot-2026.md`](compozy-adaptation-copilot-2026.md), [`compozy-adaptation-analysis.md`](compozy-adaptation-analysis.md)
> **Prompt de origem**: [`docs/prompts/compozy-adaptation-research-gemini.md`](../prompts/compozy-adaptation-research-gemini.md)

---

## Sumário Executivo

Ao contrário de Claude (ADR-009), Copilot (ADR-012) e Codex (ADR-013) — que possuem `Spec` em `internal/runtime/specs/{claude,codex,copilot}.go` e estão registrados em `cmd/ai_spec_harness/task_loop.go:27-31` (`runtimeACPCatalog`) — **Gemini-CLI permanece exclusivamente em modo legado wrapper** (`internal/wrapper/wrapper.go:91-95`: `gemini run --skill %s --project %s`). Não há `internal/runtime/specs/gemini.go`, não há entrada no `runtimeACPCatalog`, e portanto **toda a cascata F2-F5 entregue para Claude** (MCP nested-agent, normalização tool-calls, hooks dispatcher in-process Go, memory 2-tier, métricas Claude-2026, auto-review) **fica inacessível para Gemini**.

Investigação via `gh api repos/compozy/compozy/contents/internal/core/agent/registry_specs.go?ref=7f38c44` revelou que **Compozy já trata Gemini como cidadão ACP de primeira classe** (entrada `model.IDEGemini` em `registry_specs.go` lado a lado com Claude/Codex/Copilot/Cursor/Droid/OpenCode/Pi). A entrada Compozy:

```go
model.IDEGemini: {
    ID:             model.IDEGemini,
    DisplayName:    "Gemini",
    SetupAgentName: "gemini-cli",
    DefaultModel:   model.DefaultGeminiModel,  // "gemini-2.5-pro"
    Command:        "gemini",
    FixedArgs:      []string{"--acp"},
    ProbeArgs:      []string{"--acp", "--help"},
    Fallbacks: []Launcher{{
        Command:   "npx",
        FixedArgs: []string{"--yes", "@google/gemini-cli", "--acp"},
        ProbeArgs: []string{"--yes", "@google/gemini-cli", "--acp", "--help"},
    }},
    DocsURL:     "https://geminicli.com",
    InstallHint: "Install Gemini CLI with ACP support so `gemini --acp` succeeds.",
    BootstrapArgs: func(_ string, _ string, _ []string, _ string) []string { return nil },
}
```

A descoberta arquitetural central desta pesquisa: **o Google já enviou suporte ACP nativo via flag `--acp` no `@google/gemini-cli`** (anteriormente `--experimental-acp`, hoje estável conforme registro arquivado de integração ACP no Compozy). Isso elimina o maior risco que normalmente acompanharia um "F0-Bridge": **não é necessário construir adapter custom** — basta replicar a `Spec` Compozy no harness.

**Recomendação operacional**: tratar Gemini como continuação natural do trabalho ACP, com cinco fases:

- **F0-Gemini — ACP Spec Registration** (~50 LoC): novo `internal/runtime/specs/gemini.go` espelhando o template `claude.go` (sem `BootstrapArgs` dinâmico, sem `AccessModeFlag` dedicado). Registro em `runtimeACPCatalog`. Constantes `GeminiNpmPackage = "@google/gemini-cli"`, `GeminiNpmVersion` pinada via processo audit/. Testes T-13/T-14/T-15/T-16 estendidos.
- **F1-Gemini — Paridade ACP Mínima** (~100 LoC): habilitar `--runtime acp --tool gemini` end-to-end. Reuso de `internal/runtime/client/client.go`, `runner.go`, `events/`, `persistence/`. Compatibilidade table já contém modelos Gemini (`internal/taskloop/compatibility.go:34-43`); validar e estender CHANGELOG.
- **F2-Gemini — MCP nested-agent + tool-call normalization** (reuso F2-Claude): registro automático em `internal/runtime/mcpserver/` (tool-agnóstico); tabela de aliases em `.agents/normalization-rules.yaml` recebe seção Gemini (subset mínimo — Gemini usa nomes próximos a Claude, conforme `compozy/internal/core/agent/tool_call_name.go:84` lista Gemini ao lado de Claude/Cursor/Droid para a tabela `commonToolTitleAliases`).
- **F3-Gemini — Hooks + Memory 2-tier** (reuso F3-Claude): `internal/runtime/hooks/dispatcher.go` e `memory/store.go` são tool-agnósticos. Gemini ganha cascata automática quando F0+F1 entregues. Janela 1M+ tokens do Gemini permite limites de memória mais generosos como overlay opcional (`--memory-task-limit-lines=400`).
- **F4-Gemini — Métricas Gemini-2026** (~80 LoC): adaptar `internal/runtime/events/metrics.go` para campos específicos do Gemini — `cache_read_tokens` (Gemini context caching), `effective_context_tokens`, `prompt_tokens_billed`, `thoughts_tokens` (Gemini 2.5 reasoning). Não há `cache_creation_tokens` equivalente (modelo diferente do Claude).
- **F5-Gemini — Auto-review opt-in** (reuso F5-Claude): flag `--auto-review` já é tool-agnóstica quando F1-Gemini entregue.

**Esforço total**: ~3.5 sprints (F0+F1 são tijolos novos; F2-F5 são reuso da infra Claude com pequenos ajustes). Severidade hard apenas em F0/F1 (paridade arquitetural ACP); F2-F5 são guideline.

Correções factuais ao prompt original estão documentadas em §"Correções ao prompt original": (a) **Compozy não usa SDK Gemini direto** — fala via `gemini --acp` (a CLI faz a ponte); (b) **não há JSON-RPC custom** — envelope é o do `coder/acp-go-sdk v0.6.3` como nos outros runtimes; (c) MCP em Compozy é **apenas** para nested-agent, não para expor `spec-hash` ou GEMINI.md; (d) `GEMINI.md` é tratado como `CLAUDE.md` — documento estático lido pela CLI, não recurso dinâmico; (e) auto-review **não** é loop interno do agente — é modo separado.

---

## Correções ao prompt original

O prompt em `docs/prompts/compozy-adaptation-research-gemini.md` carrega quatro premissas que esta pesquisa precisou corrigir contra o código real do Compozy (validado via `gh api repos/compozy/compozy/contents/...`):

1. **"Exposição de Ferramentas via MCP"** + **"Compilação de Instruções Procedurais"** (§1 do prompt) — Compozy expõe **uma única tool MCP**: `run_agent` (`internal/core/agents/mcpserver/server.go::reservedToolName`). Essa tool serve para hand-off recursivo entre agentes, não para "expor skills ou recursos dinâmicos". Instruções procedurais (`GEMINI.md`, `CLAUDE.md`, equivalentes) **não são servidas via MCP** — são lidas pela própria CLI do agente (`gemini` lê `GEMINI.md` no boot, exatamente como `claude` lê `CLAUDE.md`). A Fase 2 do prompt ("transformar GEMINI.md em servidor de contexto dinâmico que o Gemini consome via MCP") precisa ser reformulada como **"servidor MCP interno expondo `run_agent` para nested-agent execution; GEMINI.md permanece estático, lido pela CLI no boot"**.

2. **"Padrão de Orquestração ACP"** + **"Streaming de eventos e chamadas de ferramentas granulares"** (§1 + Fase 1 do prompt) — Compozy não define wire format próprio. Toda comunicação ACP é o envelope nativo do `coder/acp-go-sdk v0.6.3`, idêntico ao consumido pelo harness em `internal/runtime/client/client.go`. **O bridge não precisa ser construído pelo harness** — Google já enviou ACP nativo via flag `--acp` no `@google/gemini-cli`. A Fase 1 do prompt ("implementação de adapter avançado em Go que transforme o gemini-cli em cliente ACP de primeira classe") precisa ser reformulada como **"registrar Spec Gemini em `internal/runtime/specs/gemini.go` apontando para o binário `gemini` com `FixedArgs=[--acp]`; reuso integral do runtime tool-agnóstico"** — bridge já existe upstream.

3. **"Loop de Auto-Validação Proativa"** (Fase 3 do prompt) — Compozy não invoca review como sub-loop dentro de execução normal. Review é um **modo de execução separado** (`ExecutionModePRReview` em `internal/core/run/executor/review_hooks.go`), disparado contra PR já aberto. A Fase 3 do prompt ("Gemini-CLI valida sua própria saída contra o execution_report.md antes de sinalizar conclusão") precisa ser reformulada como **"auto-review opt-in via flag `--auto-review` (paridade com F5-Claude), invocando skill `review` em sessão filha após session end — não loop interno do Gemini"**.

4. **"Gestão de Contexto de Longa Duração"** + **"manter o grafo de conhecimento do projeto (CODEX.md) sempre quente"** (Fase 4 do prompt) — Compozy implementa memória em `internal/core/memory/store.go` como markdown 2-tier (workflow 150 lin/12 KB + task 200 lin/16 KB), **sem hot-loading nem residência em RAM** entre invocações. Compactação é prompt-driven: quando limite é atingido, `NeedsCompaction=true` é sinalizado e o builder de prompt injeta diretiva textual ("compact the flagged memory files before proceeding") — o LLM faz o trabalho. Não há "grafo de conhecimento" estruturado. A Fase 4 do prompt precisa ser reformulada como **"adotar memória 2-tier markdown (paridade com F3-Claude) com limites configuráveis e compactação prompt-driven; aproveitar janela 1M+ do Gemini para defaults mais generosos (sugestão: 400 linhas task, 250 linhas workflow) sem residência em RAM"** — barata, determinística, auditável.

Essas correções não invalidam o roadmap proposto; alinham as recomendações ao código-fonte real e evitam que ADR-015 / PRD / TechSpec carreguem expectativas inexistentes.

---

## Mecânica Gemini-native em Compozy (achados por arquivo)

Investigação via `gh api repos/compozy/compozy/contents/<path>?ref=7f38c44` e `gh search code --repo compozy/compozy "<term>"`. Cada achado abaixo cita arquivo + propósito.

### 1. Transporte ACP unificado — Gemini trata-se como qualquer outro runtime (Compozy ✅ | harness 🔴)

**Compozy** — `internal/core/agent/registry_specs.go::model.IDEGemini`:

```go
model.IDEGemini: {
    ID:             model.IDEGemini,        // "gemini"
    DisplayName:    "Gemini",
    SetupAgentName: "gemini-cli",
    DefaultModel:   model.DefaultGeminiModel,  // "gemini-2.5-pro"
    Command:        "gemini",
    FixedArgs:      []string{"--acp"},
    ProbeArgs:      []string{"--acp", "--help"},
    Fallbacks: []Launcher{{
        Command:   "npx",
        FixedArgs: []string{"--yes", "@google/gemini-cli", "--acp"},
    }},
    BootstrapArgs: func(_, _ string, _ []string, _ string) []string { return nil },
}
```

Note três pontos arquiteturais relevantes:

- **`Command: "gemini"` + `FixedArgs: ["--acp"]`** — diferente de Claude (binário dedicado `claude-agent-acp`) e Codex (binário dedicado `codex-acp`); Gemini reutiliza o binário principal da CLI com flag de modo. Isso espelha o padrão Copilot (`copilot --acp` em `compozy/internal/core/agent/registry_specs.go::model.IDECopilot`).
- **`BootstrapArgs: () -> nil`** — sem argumentos dinâmicos. Modelo, reasoning e access mode não são propagados via flags `-c` (como Codex faz); são determinados pelo `gemini` upstream baseado em config própria.
- **`SupportsAddDirs` e `UsesBootstrapModel` não setados** (zero-value `false`) — Gemini não declara workspace extras nem participa do gating de `UsesBootstrapModel` que existe para Codex.

**ai-spec-harness**: **inexistência total**. `internal/runtime/specs/` contém `claude.go`, `codex.go`, `copilot.go`, mas não `gemini.go`. `cmd/ai_spec_harness/task_loop.go:27-31`:

```go
var runtimeACPCatalog = map[string]func() specs.Spec{
    "claude":  specs.Claude,
    "codex":   specs.Codex,
    "copilot": specs.Copilot,
}
```

Gemini é ausência declarada. Invocação atual via `internal/wrapper/wrapper.go:91-95`:

```go
case "gemini":
    return fmt.Sprintf(
        "Invoke Gemini with skill %q in project %s:\n  gemini run --skill %s --project %s%s",
        skill, projectDir, skill, projectDir, extraArgs,
    )
```

Esse comando `gemini run --skill ... --project ...` é o **modo legado wrapper**; não passa por ACP, não emite `events.jsonl` estruturado, não captura métricas, não despara hooks Go in-process. É equivalente ao "codex CLI legacy" antes do ADR-013 trazer `codex-acp`.

**Estado**: gap arquitetural fundamental — Gemini está atrasado uma geração relativa a Claude/Codex/Copilot dentro do harness.

**Gap técnico**: F0-Gemini (~50 LoC). Novo `internal/runtime/specs/gemini.go`:

```go
package specs

const (
    GeminiNpmPackage   = "@google/gemini-cli"
    GeminiNpmVersion   = "0.X.Y" // pinada via audit/
    GeminiSDKVersion   = "v0.13.0" // mesma de Claude/Codex/Copilot
    DefaultGeminiModel = "gemini-2.5-pro"
)

func Gemini() Spec {
    return newSpec(
        "gemini",
        "Gemini (ACP)",
        "gemini",
        []string{"--acp"},
        []FallbackLauncher{{
            Command:   "npx",
            FixedArgs: []string{"--yes", GeminiNpmPackage + "@" + GeminiNpmVersion, "--acp"},
        }},
        "", // AccessModeFlag vazio — Gemini não tem flag dedicada de sandbox
        GeminiSDKVersion, GeminiNpmVersion, GeminiNpmPackage,
    )
}
```

Registro adicional em `runtimeACPCatalog`. Por simetria com `claude.go:28-45`, sem `BootstrapArgs` dinâmico (Spec.BootstrapArgs() retorna nil — comportamento no-op default conforme `spec.go:59-64`).

### 2. Normalização de tool-calls driver-aware — Gemini usa tabela comum (Compozy ✅ | harness ❌)

**Compozy** — `internal/core/agent/tool_call_name.go:84`:

```go
func driverToolTitleAlias(driverID string, token string) (string, bool) {
    switch driverID {
    case model.IDEClaude, model.IDECursor, model.IDEDroid, model.IDEOpenCode, model.IDEPi, model.IDEGemini:
        // Estes drivers compartilham o conjunto canônico de aliases declarado
        // em commonToolTitleAliases (linhas 38-72): bash, click, edit_file,
        // grep, read_file, write_file, search_query, web_search, etc.
        // Não há override Gemini-específico nesta versão.
        return "", false
    case model.IDECodex:
        return codexToolTitleAlias(token)
    case model.IDECopilot:
        return copilotToolTitleAlias(token)
    }
    return "", false
}
```

Gemini está no grupo "drivers compartilhando aliases comuns" junto a Claude/Cursor/Droid/OpenCode/Pi. Isso significa: **Gemini não tem normalização específica em Compozy**. Os tool names que a CLI Gemini emite são suficientemente próximos do schema canônico (`Bash`, `Read`, `Edit`, `Grep`, `WebSearch`, etc.) para serem normalizados pela tabela `commonToolTitleAliases` (linhas 38-72 do mesmo arquivo).

**ai-spec-harness**: `internal/runtime/events/` propaga nomes/inputs como vêm do SDK, sem normalização driver-aware. Nem `normalize.go` (recém-adicionado conforme git status — untracked) nem qualquer registro Gemini.

**Impacto**: quando F0/F1-Gemini forem entregues, Gemini emitirá tool names crus no `events.jsonl` divergentes de Claude/Codex/Copilot — telemetria multi-tool fragmentada. Não há "ganho Gemini-específico" em ter normalização (Compozy comprova que tabela comum basta); o ganho é puramente sistêmico — mesmo schema canônico para todos os runtimes.

**Gap técnico**: F2-Gemini (~0 LoC novo + atualização YAML). O arquivo `.agents/normalization-rules.yaml` (já existente como untracked após F2-Claude) ganha entrada Gemini reutilizando a tabela "common":

```yaml
gemini:
  inherit: common
  overrides: {}  # Sem aliases Gemini-específicos — Compozy confirma
```

A implementação Go em `internal/runtime/events/normalize.go` resolve o `inherit: common` lendo a seção `common:` do mesmo YAML (já é como Compozy faz no array `commonToolTitleAliases`). Custo: tabela comum precisa estar declarada (provavelmente já está como subset da seção Claude).

### 3. MCP server reservado expondo `run_agent` — tool-agnóstico (Compozy ✅ | harness 🟡 F2-Claude WIP)

**Compozy** — `internal/core/agents/mcpserver/server.go::reservedToolName = "run_agent"` é **independente do driver**. O servidor MCP é spawnado pelo `Session` (em `internal/core/agent/client.go`) antes de qualquer tool específico ser conhecido; o agente (Claude, Gemini, Codex, qualquer) pode invocar `run_agent` se o seu modelo suportar tool use via ACP. Para o Gemini, `gemini --acp` expõe o canal de tool use ACP nativo, então `run_agent` funciona sem mudanças server-side.

**ai-spec-harness**: `internal/runtime/mcpserver/` existe (server.go, engine.go, wire.go — diretório untracked após F2-Claude). Implementação já é tool-agnóstica conforme padrão Compozy.

**Impacto**: assim que F0/F1-Gemini entregar `--runtime acp --tool gemini`, MCP nested-agent funciona automaticamente. Gemini pode invocar `run_agent("reviewer", "<prompt>")` e o registry (`internal/agents/registry.go`, ADR-011) resolve agente conforme padrão.

**Gap técnico**: F2-Gemini = **zero código novo**. Validação via teste de integração: `tests/integration/gemini_2026_e2e_test.go` (paralelo a `claude_2026_e2e_test.go` recente) com `--mcp-nested` ativo, verificando que `events.jsonl` registra `tool_call_kind="nested_agent"` quando Gemini invocar `run_agent` no diff.

### 4. Memória 2-tier markdown — tool-agnóstica + oportunidade Gemini-específica (Compozy ✅ | harness 🟡 sem cascata Gemini)

**Compozy** — `internal/core/memory/store.go` (limites `workflowLineLimit=150`, `workflowByteLimit=12*1024`, `taskLineLimit=200`, `taskByteLimit=16*1024`) é **tool-agnóstico**: o `Document` é lido antes do `c.Open` e injetado no prompt como bloco `## Memory Context`. Não há lógica Gemini-específica em `store.go` nem em `internal/core/prompt/common.go` (que monta o addendum).

**ai-spec-harness**: `internal/runtime/memory/` existe (store.go — diretório untracked após F3-Claude). Implementação tool-agnóstica.

**Impacto + oportunidade Gemini-específica**: limites Compozy (150/200 linhas) foram dimensionados para janelas Claude/Codex tradicionais (200K). **Gemini 2.5 Pro tem 1M+ tokens de contexto**; o overhead de injetar 400 linhas de memória workflow é ~0,1% da janela. Defaults conservadores deixam ROI na mesa.

**Gap técnico**: F3-Gemini (~10 LoC + flags). Estender `cmd/ai_spec_harness/task_loop.go` para aceitar **overrides Gemini-específicos**:

```go
// Quando --tool gemini, defaults de memory são mais generosos
case "gemini":
    if !memoryWorkflowLinesSet { memoryWorkflowLines = 250 }
    if !memoryTaskLinesSet     { memoryTaskLines = 400 }
    if !memoryWorkflowBytesSet { memoryWorkflowBytes = 20 * 1024 }
    if !memoryTaskBytesSet     { memoryTaskBytes = 32 * 1024 }
```

Idempotente: usuário pode override via `--memory-task-limit-lines`. Documentar trade-off em `GEMINI.md`: contexto longo é vantagem competitiva, mas custo de cache lookup e latência inicial sobem com prompt maior.

### 5. Pipeline de hooks in-process Go — tool-agnóstico (Compozy ✅ | harness 🟡 shell + F3-Claude WIP)

**Compozy** — `internal/core/kernel/` é **tool-agnóstico**. Hook points (`prompt.pre_build`, `prompt.post_build`, `tool_call.pre_dispatch`, `tool_call.post_complete`, `session.post_end`, etc.) são despachados pelo Orchestrator antes/depois da delegação ACP — não importa se o driver é Claude, Gemini, Codex.

**ai-spec-harness**: `internal/runtime/hooks/dispatcher.go` existe (diretório untracked após F3-Claude). Implementação tool-agnóstica.

**Hooks Shell em `.gemini/hooks/`** (paralelos a `.claude/hooks/`):

```
.gemini/hooks/
├── post-execute-task.sh
├── post-wave.sh
├── pre-execute-all-tasks.sh
├── subagent-stop-wrapper.sh
├── validate-governance.sh
└── validate-preload.sh
```

São acionados apenas no **modo interativo** do `gemini` (quando usuário invoca a CLI diretamente). No modo ACP orquestrado (após F0/F1-Gemini), os hooks shell **continuam coexistindo sem conflito** com hooks Go in-process — princípio idêntico ao documentado em `CLAUDE.md` raiz §"Hooks: Shell vs Go".

**Gap técnico**: F3-Gemini = **zero código novo no dispatcher**. Migração progressiva dos hooks Gemini-específicos (`.gemini/hooks/validate-governance.sh`) para `internal/runtime/hooks/governance.go` (compartilhado entre Claude e Gemini) — esta migração já foi feita no contexto Claude, basta verificar que `governance.go` não tem checks tool-específicos.

### 6. Auto-review como modo separado — tool-agnóstico (Compozy ✅ | harness 🟡 F5-Claude WIP)

**Compozy** — `internal/core/run/executor/review_hooks.go::afterTaskJobSuccess` chama `reviewProvider.ResolveIssues(...)` apenas quando `e.execMode == ExecutionModePRReview`. Provider é injetado (GitHub, CodeRabbit) e independe do driver. Gemini pode rodar em qualquer modo.

**ai-spec-harness**: `internal/runtime/runner_autoreview.go` existe (untracked após F5-Claude WIP). Implementação tool-agnóstica via flag `--auto-review`.

**Impacto + risco específico Gemini**: auto-review dobra custo de tokens. Para Claude/Codex (~200K janelas), isso é R$ X. Para Gemini-2.5-pro com 1M+ tokens potencialmente preenchidos, o custo pode subir desproporcionalmente — o diff anexado ao prompt de review pode ser parecido em tamanho, mas o **context caching** do Gemini é fundamentalmente diferente (TTL menor, granularidade diferente). Documentar trade-off explicitamente em `GEMINI.md` quando F5-Gemini for ativada.

**Gap técnico**: F5-Gemini = **zero código novo**. Reuso integral. Documentação ajustada.

### 7. Evidence enrichment Gemini-2026: cache, effective context, thoughts (Compozy 🟡 | harness ❌)

Compozy não extrai métricas Gemini-específicas do `acp.SessionUpdate` payload — registra eventos genericamente. Gemini 2.5 Pro emite no metadata payload três campos não cobertos pelo schema atual do harness:

- **`prompt_tokens_billed`** — tokens efetivamente cobrados (após cache hit, menor que `prompt_tokens`)
- **`cache_read_tokens`** — tokens lidos do context cache (Gemini context caching com TTL configurável; diferente do Claude prompt caching)
- **`effective_context_tokens`** — tamanho real do contexto carregado (pode ser <1M mesmo em sessões longas se houver pruning interno)
- **`thoughts_tokens`** — tokens consumidos por reasoning/thinking interno do Gemini 2.5 (similar a `thinking_tokens` do Claude mas com semântica distinta — Gemini não expõe os pensamentos por default)

**Diferenças vs Claude-2026**:

| Métrica | Claude-2026 | Gemini-2026 |
|---|---|---|
| Cache read | `cache_read_tokens` | `cache_read_tokens` (TTL menor) |
| Cache creation | `cache_creation_tokens` | — (modelo de cache implícito, sem cobrança de criação) |
| Reasoning tokens | `thinking_tokens` | `thoughts_tokens` |
| Billed tokens | `prompt_tokens` cru | `prompt_tokens_billed` (já descontado cache) |
| Effective context | n/a | `effective_context_tokens` |

**Gap técnico**: F4-Gemini (~80 LoC). Adaptar `internal/runtime/events/metrics.go::ExtractClaudeMetrics(raw)` para extrair também campos Gemini quando `driver_id == "gemini"`:

```go
type GeminiMetrics struct {
    CacheReadTokens         int
    EffectiveContextTokens  int
    PromptTokensBilled      int
    ThoughtsTokens          int
}

func ExtractGeminiMetrics(raw json.RawMessage) (GeminiMetrics, error) { /* ... */ }
```

`Summary` em `internal/runtime/runner.go` ganha campos paralelos. Validação em `internal/evidence/evidence.go` permanece opcional (ausência não bloqueia).

---

## Tabela de Gaps Consolidada (Gemini-CLI 2026)

Legenda: 🟢 implementado · 🟡 parcial · 🔴 ausente · ⭐ vantagem do harness a preservar · ⭐⭐ vantagem específica Gemini

| # | Feature | Status Orchestrator | Padrão Compozy | Gap Técnico | Fase | Severidade |
|---|---|---|---|---|---|---|
| 1 | Gemini via ACP nativo (`gemini --acp`) | 🔴 (wrapper-only) | 🟢 `registry_specs.go::IDEGemini` | Novo `specs/gemini.go` + registro `runtimeACPCatalog` (~50 LoC) | **F0-Gemini** | hard |
| 2 | Forense `events.jsonl`/`tool_calls.md`/`execution_report.md` | 🟡 (existe; Gemini não usa) | 🟡 (OTel/Grafana, sem markdown) | Cascata automática após F0/F1 | **F1-Gemini** | hard |
| 3 | `ActivityWatchdog` com `CancelCause` | 🟢 ⭐ | 🔴 | Cascata automática após F0/F1 | **F1-Gemini** | — |
| 4 | Servidor MCP reservado `run_agent` (nested-agent) | 🟡 (existe; Gemini não usa) | 🟢 `agents/mcpserver/server.go` | Cascata automática após F0/F1; F2 valida integração | **F2-Gemini** | guideline |
| 5 | Normalização de tool-calls driver-aware | 🟡 (existe; sem entry Gemini) | 🟢 tabela comum + driver overrides (Gemini herda comum) | Update em `.agents/normalization-rules.yaml`: `gemini: { inherit: common }` | **F2-Gemini** | guideline |
| 6 | Memória 2-tier markdown com limites byte/linha | 🟡 (existe; cascata automática) | 🟢 `memory/store.go` | Cascata automática + defaults Gemini-específicos (~10 LoC) | **F3-Gemini** | guideline |
| 7 | Pipeline de hooks in-process Go | 🟡 (existe; cascata automática) | 🟢 `kernel/` | Cascata automática | **F3-Gemini** | guideline |
| 8 | Evidence com campos Gemini-2026 (cache_read, thoughts, effective_context) | 🔴 | 🟡 (não extrai) | Estender `events/metrics.go` + `events/convert.go` + `evidence/evidence.go` (~80 LoC) | **F4-Gemini** | guideline |
| 9 | Auto-review opt-in via flag CLI | 🟡 (existe; cascata automática) | 🟢 (modo separado `ExecutionModePRReview`) | Cascata automática + docs trade-off de custo Gemini | **F5-Gemini** | guideline |
| 10 | Wrapper `ValidTools["gemini"]` | 🟢 (atual; modo legado) | n/a | **Manter durante transição** F0→F1; remover em release N+2 com aviso de deprecation | — | — |
| 11 | Registry de agentes declarativo (ADR-011) | 🟢 `internal/agents/registry.go` | 🟢 `.compozy/agents/<name>/manifest.yaml` | Resolução tool-agnóstica; sem mudança | F1-Gemini (cascata) | — |
| 12 | `spec-hash` validation | 🟢 ⭐ `internal/specdrift/` | 🔴 | Tool-agnóstico; sem mudança | — | — |
| 13 | Telemetria opt-in (`GOVERNANCE_TELEMETRY=1`) | 🟢 ⭐ ADR-006 | 🟡 (OTel sempre on) | Tool-agnóstico; sem mudança | — | — |
| 14 | `GEMINI.md` 2026 atualizado com Runtime Capabilities | 🟡 atual sem §Runtime Capabilities | 🟢 (`CLAUDE.md` em compozy raiz tem) | Adicionar §"Runtime Capabilities (F0-Gemini+)" listando ACP + cascata F2-F5 | **F0-Gemini** | guideline |
| 15 | Janela 1M+ tokens — defaults memory generosos | 🔴 (defaults Claude/Codex) | n/a (Compozy não diferencia) | ⭐⭐ Defaults Gemini-específicos via switch em task_loop.go | **F3-Gemini** | guideline |
| 16 | Context caching Gemini (TTL configurável) | 🔴 | n/a | ⭐⭐ Métrica `cache_read_tokens` (TTL menor que Claude) + flag `--gemini-cache-ttl` opcional | **F4-Gemini** | guideline |

**Critérios de esforço**: Baixo ≤ 1 sprint; Médio 1–2 sprints; Alto ≥ 2 sprints com pesquisa.

**Sumário**: F0+F1-Gemini são os tijolos novos (severidade hard). Uma vez entregues, F2-F5 cascateiam de forma quase gratuita reusando infra Claude já existente. Os únicos delta Gemini-específicos são: (a) entrada YAML de normalização (zero código); (b) defaults de memory generosos (~10 LoC); (c) métricas Gemini-2026 distintas (`thoughts_tokens`, `effective_context_tokens`, ~80 LoC); (d) opcional `--gemini-cache-ttl`. Severidade hard concentrada em F0/F1; F2-F5 são guideline porque a infraestrutura é compartilhada.

---

## Análise específica Gemini (distintiva)

Esta seção destaca o que torna Gemini-CLI 2026 diferente — não apenas como mais um runtime ACP, mas como classe própria dentro do portfólio do harness.

### 1. Vantagens competitivas

**Janela de contexto 1M+ tokens (Gemini 2.5 Pro)**. Concretamente:

- Suporta carregamento de monorepos médios (~500K LOC textuais) **em uma única chamada**, sem fragmentação por arquivo.
- Permite que `GEMINI.md` + `AGENTS.md` + toda `.agents/skills/*` + memória workflow + memória task + 50 arquivos relevantes do diff caibam no system prompt + prefix sem evicção.
- Habilita padrões de "whole-repo reasoning" infactíveis para Claude/Codex no estado atual.

**`GEMINI.md` como instrução procedural forte**. A CLI `gemini` lê `GEMINI.md` no boot e aplica suas diretivas com aderência alta (comportamento documentado por Google em geminicli.com). Trade-off vs Claude: menos liberdade criativa, mais determinismo procedural — o que casa muito bem com o modelo SDD (Software Development Design) `PRD-first` + `spec-hash` do harness.

**Context caching Gemini com TTL configurável**. Diferente do Claude (prompt caching estático, 5-min TTL implícito), Gemini permite anchorar uma porção do contexto via API `cachedContent` com TTL controlado pelo cliente. Para o harness, isso significa que `AGENTS.md` + skills procedurais podem ser cacheadas explicitamente por sessão (a CLI ainda não expõe essa flag diretamente, mas o roadmap upstream do Google indica suporte em 2026-Q3).

### 2. Limites do wrapper atual

O comando `gemini run --skill ... --project ...` (`internal/wrapper/wrapper.go:91-95`) tem três falhas estruturais para o caso de uso 2026:

1. **Sem streaming de eventos**: a saída do `gemini run` é capturada como bloco único pela skill `execute-task`. Não há `events.jsonl` granular, não há `tool_calls.md`, não há rastreamento de cada tool invocation. Forense post-mortem é fraca.
2. **Sem tool-call granular**: o Orchestrator não sabe quando o Gemini invocou `Bash` vs `Edit` vs `Read`. Não há `ActivityWatchdog` por tool, não há cancellation in-flight, não há normalização.
3. **Sem cascata para F2-F5**: porque `gemini run` não fala ACP, não há onde plugar MCP nested-agent, hooks dispatcher, memory store ou métricas. Toda a evolução arquitetural do harness contorna Gemini.

### 3. Opções para F0-Gemini

Esta pesquisa avaliou três caminhos:

**Opção A (Recomendada): adoção direta de `gemini --acp`**. Como Google já enviou ACP nativo no `@google/gemini-cli`, basta replicar o padrão `claude.go` em `gemini.go`. Esforço: ~50 LoC. Risco: depende da estabilidade da flag `--acp` upstream (estável conforme Compozy main; foi `--experimental-acp` em early-2026 segundo task arquivada de Compozy). Mitigação: pinning de `GeminiNpmVersion` via processo audit/, mesmo padrão de ADR-009 e ADR-013.

**Opção B (Rejeitada): bridge custom Go sobre Gemini SDK**. Construir adapter próprio em `internal/runtime/gemini-adapter/` que fale Gemini SDK e exponha ACP. Esforço: ≥2 sprints. Risco: alto — mantém wrapper em sync com mudanças do SDK Google, duplica trabalho que já existe em `@google/gemini-cli`. **Rejeitada porque opção A é equivalente em capacidade com 5% do esforço.**

**Opção C (Rejeitada): protocolo híbrido shim**. Manter wrapper `gemini run` + adicionar canal lateral via FIFO para streaming. Esforço: ~3 sprints. Risco: muito alto — mistura modos, expõe condições de corrida, não é o padrão Compozy. **Rejeitada por desvio do padrão de mercado.**

**Decisão**: Opção A. ADR-015 (proposto) documenta D-01..D-04 cobrindo: (D-01) escolha de `gemini --acp` como command canônico; (D-02) pinning npm via audit; (D-03) ausência de `BootstrapArgs` (no-op por design); (D-04) coexistência do wrapper legado com deprecation em N+2 releases.

---

## Roadmap de Adaptação (Gemini-específico)

### F0-Gemini — ACP Spec Registration (PRÉ-REQUISITO)

**Escopo**:

- Novo `internal/runtime/specs/gemini.go` (~50 LoC) seguindo template `claude.go:1-45`:
  - Constantes: `GeminiNpmPackage = "@google/gemini-cli"`, `GeminiNpmVersion` (pinada via audit/; verificar latest stable em 2026-05-22 via `npm view @google/gemini-cli versions --json`), `GeminiSDKVersion = "v0.13.0"` (mesma de Claude/Codex/Copilot), `DefaultGeminiModel = "gemini-2.5-pro"`.
  - `Gemini()` retorna `Spec` via `newSpec(...)` com `Command="gemini"`, `FixedArgs=["--acp"]`, `Fallbacks=[{npx, ["--yes", GeminiNpmPackage+"@"+GeminiNpmVersion, "--acp"]}]`, `AccessModeFlag=""`, sem `BootstrapArgs`.
- Registro em `cmd/ai_spec_harness/task_loop.go:27-31`:
  ```go
  var runtimeACPCatalog = map[string]func() specs.Spec{
      "claude":  specs.Claude,
      "codex":   specs.Codex,
      "copilot": specs.Copilot,
      "gemini":  specs.Gemini,   // F0-Gemini
  }
  ```
- Testes:
  - T-13 estendido: `TestRuntimeACPCatalogIncludesGemini`
  - T-14 estendido: `TestGeminiSpecHasCorrectCommandAndFlags`
  - T-15 estendido: `TestGeminiFallbackResolvesViaNpx`
  - T-16 estendido: `TestGeminiBootstrapArgsAlwaysNil`
- ADR-015 (`.specs/adr/015-gemini-cli-acp-native.md`) — status Proposta — documenta D-01..D-04.
- `GEMINI.md` raiz ganha §"Runtime Capabilities (F0-Gemini+)" (ver §"Exemplos de Configuração 2026").

**Esforço**: Baixo (~3 dias).  
**Risco**: Baixo (padrão `claude.go` é referência sólida; `gemini --acp` validado pelo Compozy main).  
**Dependências**: ADR-013 entregue (referência arquitetural); nenhuma outra.  
**Critério de aceitação**: `ai-spec task-loop --tool gemini --runtime acp --dry-run .specs/prd-X` retorna comando `gemini --acp` (ou `npx --yes @google/gemini-cli@X.Y.Z --acp` no fallback) sem erro.

### F1-Gemini — Paridade ACP Mínima

**Escopo**:

- Habilitar `--runtime acp --tool gemini` end-to-end:
  - `internal/runtime/runner.go::Run()` aceita Gemini via mapeamento já existente (após F0)
  - `internal/runtime/client/client.go` é tool-agnóstico (já consome `Spec`)
  - `internal/runtime/events/convert.go` é tool-agnóstico — mas validar que tipos ACP do Gemini não exigem casos especiais
  - `internal/runtime/persistence/session.go` é tool-agnóstico (persistência markdown)
- Verificar `internal/taskloop/compatibility.go:34-43` — tabela já contém modelos Gemini (`gemini-2.5-pro`, `flash`, `pro`, etc.); confirmar que `IsSupported("gemini", "gemini-2.5-pro")` retorna true. Sem mudanças esperadas.
- Atualizar CHANGELOG.md raiz com entrada `feat(gemini): F0+F1 ACP nativo via gemini --acp (ADR-015)`.
- Testes de integração:
  - `tests/integration/gemini_acp_smoke_test.go` (novo): invoca `gemini --acp` real (ou mock via fake ACP server) e verifica que `events.jsonl` é produzido.
  - `tests/integration/gemini_parity_test.go`: comparação side-by-side com Claude, mesma task simples, validando que `tool_calls.md` tem estrutura paralela.
- Atualizar `docs/cli-schema.json` para incluir `gemini` em enum de `--tool` quando `--runtime=acp`.
- Migração progressiva: `internal/wrapper/wrapper.go::ValidTools["gemini"] = true` **permanece**; warning informativo é emitido quando `--runtime` não é `acp` ("Gemini wrapper mode is deprecated since vN; use --runtime acp for full feature parity"). Remoção do wrapper Gemini fica para release N+2.

**Esforço**: Baixo–Médio (~5 dias).  
**Risco**: Médio (primeira sessão ACP Gemini pode revelar incompatibilidades sutis no event schema — ex: `acp.ToolKindThink` que Claude usa pode não ter equivalente no Gemini).  
**Dependências**: F0-Gemini entregue.  
**Critério de aceitação**: `ai-spec task-loop --tool gemini --runtime acp .specs/prd-X` completa uma task simples (edit de 1 arquivo) e produz `events.jsonl` + `tool_calls.md` + `execution_report.md` com seções obrigatórias.

### F2-Gemini — MCP nested-agent + tool-call normalization

**Escopo**:

- **MCP nested-agent**: zero código novo. Cascata automática após F1. Validar via teste:
  - T-17 `TestMCPNestedAgentSpawnsGeminiSession` — Gemini parent invoca `run_agent("reviewer", "...")` e child session é spawnada via mesmo `internal/runtime/mcpserver/engine.go`.
  - T-18 `TestMCPDepthLimitAppliesToGemini` — depth=4 retorna erro tipado.
- **Normalização**: atualizar `.agents/normalization-rules.yaml` (e arquivo embedded `internal/runtime/events/normalization-rules.yaml`):
  ```yaml
  gemini:
    inherit: common
    overrides: {}
  ```
- Pequeno ajuste em `internal/runtime/events/normalize.go` para resolver `inherit: common` (se ainda não suporta). Garantir que `BuildNormalizedToolCall("gemini", ...)` produz o mesmo schema que Claude para `bash`, `read`, `write`, `edit`, `grep`, `web_search`.
- Testes:
  - T-19 `TestNormalizeToolCallGemini` — Gemini emite `read_file` → normalizado `Read`
  - T-20 `TestNormalizeToolCallGeminiPreservesRaw` — `raw_name` preservado lado a lado.
- Atualizar `GEMINI.md` raiz: §"Runtime Capabilities (F2-Gemini+)" lista MCP nested-agent + normalização.

**Esforço**: Baixo (~3 dias).  
**Risco**: Baixo (infra existe; ajuste de tabela).  
**Dependências**: F1-Gemini entregue; F2-Claude entregue (provê infra MCP + normalize).  
**Critério de aceitação**: `ai-spec task-loop --tool gemini --runtime acp --mcp-nested .specs/prd-X` produz `events.jsonl` com `tool_call_kind="nested_agent"` quando Gemini invoca `run_agent`.

### F3-Gemini — Hooks + Memory 2-tier com defaults Gemini-generosos

**Escopo**:

- **Hooks dispatcher**: zero código novo. Cascata automática após F1. Validar via teste:
  - T-21 `TestHookDispatchOnGeminiSession` — hook `runtime.pre_open` é despachado antes de `c.Open` mesmo quando driver é Gemini.
- **Memory 2-tier**: zero código novo na store. Defaults Gemini-específicos via switch em `task_loop.go`:
  ```go
  // Apenas quando flags --memory-* não foram passadas explicitamente
  switch tool {
  case "gemini":
      if !cmd.Flags().Changed("memory-workflow-limit-lines") {
          memoryWorkflowLines = 250  // vs 150 default
      }
      if !cmd.Flags().Changed("memory-task-limit-lines") {
          memoryTaskLines = 400      // vs 200 default
      }
      if !cmd.Flags().Changed("memory-workflow-limit-bytes") {
          memoryWorkflowBytes = 20 * 1024  // vs 12*1024
      }
      if !cmd.Flags().Changed("memory-task-limit-bytes") {
          memoryTaskBytes = 32 * 1024      // vs 16*1024
      }
  }
  ```
- Testes:
  - T-22 `TestGeminiDefaultsMemoryLimitsAreGenerous`
  - T-23 `TestGeminiMemoryLimitOverrideByCliFlag`
- Documentar trade-off em `GEMINI.md`: contexto longo barateia compactação, mas aumenta latência do prompt-build e custo de cache lookup.

**Esforço**: Baixo (~2 dias).  
**Risco**: Baixo.  
**Dependências**: F1-Gemini entregue; F3-Claude entregue (provê infra hooks + memory).  
**Critério de aceitação**: sessão Gemini sem flag `--memory-task-limit-lines` aceita memory task de 350 linhas sem `NeedsCompaction=true`; mesma sessão Claude (defaults 200) sinaliza compaction.

### F4-Gemini — Evidence enrichment com métricas Gemini-2026

**Escopo**:

- Novo `internal/runtime/events/gemini_metrics.go` (~80 LoC):
  ```go
  type GeminiMetrics struct {
      CacheReadTokens        int
      EffectiveContextTokens int
      PromptTokensBilled     int
      ThoughtsTokens         int
  }
  
  func ExtractGeminiMetrics(raw json.RawMessage) (GeminiMetrics, error) { /* ... */ }
  func LogGeminiMetrics(ctx, m GeminiMetrics) { /* ... */ }
  ```
- Estender `Summary` em `internal/runtime/runner.go`:
  ```go
  GeminiCacheReadTokens        int
  GeminiEffectiveContextTokens int
  GeminiPromptTokensBilled     int
  GeminiThoughtsTokens         int
  ```
- Estender `internal/evidence/evidence.go` com seção "Métricas Gemini-2026" opcional:
  ```markdown
  ## Métricas Gemini-2026
  | Métrica                   | Valor |
  |---------------------------|-------|
  | cache_read_tokens         | N     |
  | effective_context_tokens  | N     |
  | prompt_tokens_billed      | N     |
  | thoughts_tokens           | N     |
  | tool_calls_normalized     | N     |
  ```
- Validador: presença é opcional; ausência não bloqueia evidence.
- Telemetria (`internal/telemetry/`): se `GOVERNANCE_TELEMETRY=1`, append entries `gemini.cache_read=N`, `gemini.thoughts=N`, `gemini.effective_context=N`.
- Testes:
  - T-24 `TestExtractGeminiMetricsFromACPPayload`
  - T-25 `TestEvidenceRendersGeminiMetricsSection`
  - T-26 `TestEvidenceMissingGeminiMetricsDoesNotBlock`

**Esforço**: Baixo (~3 dias).  
**Risco**: Baixo–Médio (depende do schema exato de payload Gemini no ACP; pode requerer ajuste após teste contra `gemini --acp` real).  
**Dependências**: F1-Gemini entregue.

### F5-Gemini — Auto-review opt-in

**Escopo**:

- Zero código novo na lógica de auto-review (cascata automática após F1).
- **Validação Gemini-específica**: testes que garantem que `--auto-review` funciona quando driver é Gemini.
- Documentar em `GEMINI.md` o **custo amplificado** quando combinado com janela 1M+: cada review pode consumir até 2x o orçamento da task original se o diff for grande e cache hit baixo.
- Sugestão: futuro `--gemini-auto-review-cache-ttl` para reuso de prompt cache entre review-run e task-run (especulação 2026-Q3; não escopo de F5-Gemini).
- Testes:
  - T-27 `TestAutoReviewWithGeminiDriver`
  - T-28 `TestAutoReviewBlocksOnHardIssuesWithGemini`

**Esforço**: Baixo (~1 dia, principalmente teste e doc).  
**Risco**: Baixo (custo de tokens é runtime concern, não correctness concern).  
**Dependências**: F1-Gemini entregue; F5-Claude entregue.

---

## Exemplos de Configuração 2026

### `GEMINI.md` raiz — adendos F0+/F2+/F3+

```markdown
# ai-spec-harness — Gemini CLI

Use `AGENTS.md` como fonte canonica das regras deste repositorio.

## Runtime Capabilities (F0-Gemini+)

Quando o Orchestrator invoca Gemini via ACP (`--runtime acp --tool gemini`), o runtime
expoe:

- **ACP nativo** via `gemini --acp`. Fallback: `npx --yes @google/gemini-cli@X.Y.Z --acp`.
  Modelo default: `gemini-2.5-pro`. Pinning em `internal/runtime/specs/gemini.go`.
- **Modo legado wrapper** (`gemini run --skill ...`) permanece disponivel via
  `internal/wrapper/` ate o release N+2; emite warning de deprecation.

## Runtime Capabilities (F2-Gemini+)

- **MCP nested-agent** (`--mcp-nested`): tool `run_agent(agent_name, prompt, model?, timeout?)`.
  Profundidade maxima: 3. Resolucao de `agent_name` via `internal/agents/registry.go`.
- **Tool-call normalization** (sempre ativa a partir de F2-Gemini): nomes/inputs sao
  canonicalizados via `.agents/normalization-rules.yaml` (Gemini herda tabela `common`).

## Runtime Capabilities (F3-Gemini+)

- **Hooks in-process Go**: pontos canonicos `runtime.pre_open`, `prompt.pre_build`,
  `prompt.post_build`, `tool_call.pre_dispatch`, `tool_call.post_complete`,
  `session.post_end`. Compartilhados com Claude. Para desabilitar: `--disable-hooks`.
- **Memoria 2-tier** com **defaults Gemini-generosos** (aproveitando janela 1M+):
  workflow 250 linhas / 20 KiB; task 400 linhas / 32 KiB. Override via
  `--memory-workflow-limit-lines`, `--memory-task-limit-lines`, etc.

## Runtime Capabilities (F4-Gemini+)

- **Evidence Gemini-2026**: `execution_report.md` ganha secao "Metricas Gemini-2026"
  com `cache_read_tokens`, `effective_context_tokens`, `prompt_tokens_billed`,
  `thoughts_tokens`. Captura via `internal/runtime/events/gemini_metrics.go`.

## Runtime Capabilities (F5-Gemini+)

- **Auto-review** (opt-in `--auto-review`): apos session end, spawna sessao extra com
  skill `review`. Custo amplificado em Gemini quando diff e grande — documentar
  trade-off explicitamente nas tasks.

## Instrucoes (mantidas)

1. Ler `AGENTS.md` no inicio da sessao.
2. `.agents/skills/` e a fonte de verdade dos fluxos procedurais.
3. `.gemini/commands/` sao adaptadores finos que apontam para a habilidade correta
   (apenas no modo wrapper legado; no modo ACP, skills sao invocadas pelo runner).
...
```

### `.gemini/agents/` evolução (alinhamento com `internal/agents/registry.go`)

Hoje `.gemini/agents/` contém apenas `task-executor.md` (mirror de `.claude/agents/`). A partir de F2-Gemini, esses arquivos passam a ser **resolvidos via `internal/agents/registry.go`** quando MCP `run_agent` for invocado pelo Gemini parent: `run_agent("reviewer", "<prompt>")` resolve para o spec de `reviewer` e spawna nova sessão ACP isolada (que pode ser Gemini ou Claude, conforme manifesto do agente).

Manifest opcional `.gemini/agents/<name>.yaml` (paralelo ao `.md`) declara model preferido (`gemini-2.5-pro` vs `gemini-2.5-flash` por agent), reasoning effort, timeout. Sem manifest, herda do parent.

### `.gemini/skills/` recebendo hooks `prompt.pre_build` (F3-Gemini+)

A partir de F3-Gemini, qualquer skill em `.gemini/commands/<name>.toml` (ou `.agents/skills/<name>/`) pode declarar em seu frontmatter um hook in-process:

```yaml
---
name: governance-preflight-gemini
hook: prompt.pre_build
implementation: internal/runtime/hooks/governance.go
priority: 100
applies_to: ["gemini"]  # opcional; default aplica a todos os drivers
---
```

Implementação Go obrigatória; YAML é só metadata para o registry. Hooks shell em `.gemini/hooks/*.sh` permanecem como camada de compatibilidade para modo interativo da CLI Gemini (sem migração forçada).

### Stub de `internal/runtime/specs/gemini.go` (F0-Gemini)

```go
// Package specs — runtime Gemini via ACP nativo @google/gemini-cli.
//
// Diferencas-chave vs Codex/Claude:
//   - Command "gemini" (binario padrao da CLI Google) — NAO um adapter dedicado.
//   - FixedArgs ["--acp"] (modo ACP nativo, estavel desde 2026-Q1).
//   - BootstrapArgs nil (modelo, reasoning, sandbox sao configurados pelo gemini
//     upstream via gemini config; nao propagamos via flags -c).
//   - AccessModeFlag vazio (Gemini nao tem flag dedicada de sandbox).
//
// ADR-015 D-01..D-04.
package specs

const (
    // GeminiNpmPackage e o nome do pacote npm canonico do Gemini CLI.
    // ADR-015 D-01.
    GeminiNpmPackage = "@google/gemini-cli"

    // GeminiNpmVersion e a versao npm pinada do @google/gemini-cli.
    // Pinada conforme ADR-015 D-02: constante Go atualizada somente via audit/.
    // Validada via `npm view @google/gemini-cli version dist-tags` em 2026-05-22:
    // dist-tag `latest = 0.43.0`. Branches preview/nightly ativas em 0.44.x mas
    // nao adotadas em produc'ao do harness.
    GeminiNpmVersion = "0.43.0"

    // GeminiSDKVersion e a versao do coder/acp-go-sdk sincronizada com go.mod.
    // Mesma de Claude/Codex/Copilot.
    GeminiSDKVersion = "v0.13.0"

    // DefaultGeminiModel espelha compozy/internal/core/model/constants.go.
    DefaultGeminiModel = "gemini-2.5-pro"
)

// Gemini retorna a Spec do runtime Gemini via @google/gemini-cli com flag --acp.
//
// Binario canonico: "gemini" (NAO "gemini-cli" — esse e o nome do pacote npm).
// Fallback: npx --yes @google/gemini-cli@<GeminiNpmVersion> --acp.
// FixedArgs ["--acp"] — modo ACP nativo.
// BootstrapArgs nil — Gemini nao usa -c overrides como Codex (ADR-015 D-03).
// AccessModeFlag vazio (ADR-015 D-04).
func Gemini() Spec {
    return newSpec(
        "gemini",
        "Gemini (ACP)",
        "gemini",
        []string{"--acp"},
        []FallbackLauncher{
            {
                Command:   "npx",
                FixedArgs: []string{"--yes", GeminiNpmPackage + "@" + GeminiNpmVersion, "--acp"},
            },
        },
        "", // AccessModeFlag vazio (ADR-015 D-04)
        GeminiSDKVersion, GeminiNpmVersion, GeminiNpmPackage,
    )
}
```

### Stub de `internal/runtime/events/gemini_metrics.go` (F4-Gemini)

```go
package events

import "encoding/json"

// GeminiMetrics captura metricas Gemini-2026 do payload acp.SessionUpdate.
// Campos sao opcionais — ausencia nao bloqueia evidence (ADR-015 D-04).
type GeminiMetrics struct {
    CacheReadTokens        int `json:"cache_read_tokens"`
    EffectiveContextTokens int `json:"effective_context_tokens"`
    PromptTokensBilled     int `json:"prompt_tokens_billed"`
    ThoughtsTokens         int `json:"thoughts_tokens"`
}

// ExtractGeminiMetrics le os campos Gemini-especificos do raw JSON do
// acp.SessionUpdate quando o driver_id e "gemini". Retorna GeminiMetrics{}
// zero-value quando o payload nao contem os campos (silencioso).
func ExtractGeminiMetrics(raw json.RawMessage) (GeminiMetrics, error) {
    var envelope struct {
        Usage struct {
            CacheReadTokens        int `json:"cache_read_tokens"`
            EffectiveContextTokens int `json:"effective_context_tokens"`
            PromptTokensBilled     int `json:"prompt_tokens_billed"`
            ThoughtsTokens         int `json:"thoughts_tokens"`
        } `json:"usage"`
    }
    if err := json.Unmarshal(raw, &envelope); err != nil {
        return GeminiMetrics{}, err
    }
    return GeminiMetrics{
        CacheReadTokens:        envelope.Usage.CacheReadTokens,
        EffectiveContextTokens: envelope.Usage.EffectiveContextTokens,
        PromptTokensBilled:     envelope.Usage.PromptTokensBilled,
        ThoughtsTokens:         envelope.Usage.ThoughtsTokens,
    }, nil
}
```

### Padrão de hooks metadata reutilizando Go in-process (F3-Gemini+)

`internal/runtime/hooks/governance.go` (já existente, criado em F3-Claude) é tool-agnóstico. Para confirmar aplicabilidade Gemini, basta verificar que sua implementação não tem switch por `driverID` que exclua "gemini". Caso tenha (improvável; F3-Claude foi projetada agnóstica), adicionar caso explícito Gemini.

### `.gemini/` mantendo paridade com `.claude/` + hot context

```
.gemini/
├── agents/
│   └── task-executor.md       (mirror auto-sincronizado)
├── commands/
│   ├── workspace.*.toml       (24 wrappers atuais — modo legado)
│   └── (F1+ os wrappers podem ser desativados quando --runtime acp for default)
├── docs/
│   └── workaround-preload.md
└── hooks/                     (shell, modo interativo)
    ├── post-execute-task.sh
    ├── post-wave.sh
    ├── pre-execute-all-tasks.sh
    ├── subagent-stop-wrapper.sh
    ├── validate-governance.sh
    └── validate-preload.sh
```

Estratégia "hot context" (residente em RAM entre invocações de subagent) **não é adotada** porque: (a) Compozy não faz; (b) violaria o princípio de auditabilidade (`audit/` append-only); (c) janela 1M+ do Gemini torna re-carga relativamente barata vs complexidade de IPC residente. Em vez disso, F4-Gemini introduz métrica `effective_context_tokens` que permite identificar quando a janela está sendo bem aproveitada — proxy observável vs otimização especulativa.

---

## Riscos e Premissas

| # | Risco/Premissa | Mitigação |
|---|---|---|
| 1 | Flag `--acp` em `@google/gemini-cli` pode ser renomeada ou ter breaking change em 2026-Q3 | Política de pinning ADR-015 D-02 igual a ADR-009: atualizar via `audit/`, não `@latest`; testes de probe validam que o comando responde antes de prosseguir |
| 2 | Schema ACP do Gemini pode ter campos divergentes de Claude (ex: `acp.ToolKindThink` ausente) | F1-Gemini tem teste de integração `gemini_acp_smoke_test.go`; descobertas alimentam ajustes em `convert.go` |
| 3 | Pacote `@google/gemini-cli` pode ser sucedido por novo pacote (ex: `@google/gemini-agent-cli`) | Constante `GeminiNpmPackage` é Go const; troca por audit/ + migration test |
| 4 | Janela 1M+ pode ter custos elevados de prompt-build quando memory carrega 400 linhas | Trade-off documentado em GEMINI.md; defaults reversíveis via flag; métrica `effective_context_tokens` monitora utilização real |
| 5 | Context caching do Gemini (TTL menor que Claude) pode degradar perf se task gerar muitas sub-tasks | Métrica `cache_read_tokens` em F4-Gemini permite detectar miss rate alto; futura flag `--gemini-cache-ttl` (não escopo F0..F5) |
| 6 | Modo wrapper legado (`gemini run --skill`) e modo ACP coexistindo podem confundir usuários | Mensagem clara em CLI quando wrapper detectado: "Gemini wrapper mode is deprecated since vN; use --runtime acp"; remoção planejada para release N+2 |
| 7 | Hook shell `.gemini/hooks/validate-governance.sh` pode divergir da implementação Go `internal/runtime/hooks/governance.go` | Lint script `scripts/check-hooks-sync` (já existe) cobre `.gemini/`; CI obrigatório |
| 8 | Métricas Gemini-2026 podem mudar de nomenclatura no payload ACP entre versões do gemini-cli | Extração defensiva (`ExtractGeminiMetrics` retorna zero-value silencioso em ausência de campos); pinning de versão limita drift |
| 9 | Auto-review em Gemini com diff grande pode ultrapassar quota de tokens da org | Documentar custo em GEMINI.md F5; default `--auto-review` permanece false; sugerir uso seletivo em tasks de alto risco |
| 10 | Reasoning (`thoughts_tokens`) em Gemini 2.5 não é exposto por default — pode causar campo sempre zero | F4-Gemini documenta como caveat conhecido; valor zero é semanticamente válido (não é erro) |

---

## Avaliação de Confiabilidade SDD

A integração proposta preserva integralmente as invariantes do harness:

1. **`spec-hash` + `PRD-first`**: nenhuma fase modifica `internal/specdrift/` nem o fluxo `create-prd → create-technical-specification → create-tasks → execute-task`. O Gemini se torna mais um driver no `runtimeACPCatalog`, sem privilégios especiais. Skill `execute-all-tasks` continua orquestrando — Gemini executa tasks individuais como Claude faria.

2. **Evidence append-only (ADR-006)**: F1+ produz `events.jsonl` append-only; F4 adiciona campos a `execution_report.md` sem modificar seções obrigatórias. Telemetria opt-in continua opt-in.

3. **Pinning de versões**: ADR-015 D-02 herda integralmente o padrão de ADR-009/ADR-013 — constantes Go com atualização via `audit/`, nunca `@latest`. Probe `internal/runtime/probe/` valida disponibilidade do binário antes de spawn.

4. **Governança transversal**: `R-GOV-001` (`.claude/rules/governance.md`) aplica-se a Gemini sem ajuste — precedência é tool-agnóstica.

5. **Coexistência sem regressão**: wrapper legado preservado durante transição (F1 não força remoção); shell hooks `.gemini/hooks/*.sh` mantidos para modo interativo da CLI Google. Modo orquestrado (ACP) e modo interativo (CLI direta) operam em planos distintos.

**Nota técnica**: a viabilidade desta integração é **alta**. O maior risco arquitetural — construir adapter ACP custom — foi eliminado pela descoberta de que `@google/gemini-cli` já implementa ACP nativo (`gemini --acp`). O esforço residual é, em essência, replicar o padrão `claude.go` para `gemini.go` (F0+F1) e adicionar pequenos ajustes Gemini-específicos em F3 (memory defaults) e F4 (métricas distintas). A integração mantém o rigor SDD sem trade-offs significativos.

---

## Decisão Recomendada

Abrir o pacote de governança completo conforme entregue por esta pesquisa:

- **ADR-015** (`.specs/adr/015-gemini-cli-acp-native.md`) — status Proposta — documenta D-01..D-04 cobrindo escolha de `gemini --acp` como command canônico, pinning npm via audit, BootstrapArgs no-op, AccessModeFlag vazio.
- **PRD** (`.specs/prd-gemini-cli-acp-2026/prd.md`) — RF-01..RF-06 + NFRs — consome ADR-015. RF-01 = F0+F1 (spec + paridade); RF-02 = F2 (MCP + normalization); RF-03 = F3 (hooks + memory + defaults Gemini); RF-04 = F4 (métricas Gemini-2026); RF-05 = F5 (auto-review); RF-06 = deprecation wrapper legado.
- **TechSpec** (`.specs/prd-gemini-cli-acp-2026/techspec.md`) — cabeçalho com `spec-hash-prd` placeholder — arquitetura, interfaces, riscos, estratégia de testes; pronto para `create-tasks` após `ai-spec sync-spec-hash` materializar o hash.

**Não escrever código nesta sessão.** Tarefas de implementação ficam para uma sessão posterior via skill `create-tasks` + `execute-task` (subdivididas por fase F0 → F5, com `execute-all-tasks` paralelizando dentro de cada fase quando o DAG permitir).

---

## Continuidade — Roadmap Pós-pesquisa

| PRD | Escopo | Esforço | Risco | Dependências |
|---|---|---|---|---|
| **prd-gemini-cli-acp-2026 (F0)** | Spec registration + ADR-015 | Baixo (3 dias) | Baixo | ADR-013 entregue |
| **prd-gemini-cli-acp-2026 (F1)** | Paridade ACP mínima E2E | Baixo–Médio (5 dias) | Médio (schema ACP) | F0 entregue |
| **prd-gemini-cli-acp-2026 (F2)** | MCP nested-agent + normalization | Baixo (3 dias) | Baixo | F1 + F2-Claude entregues |
| **prd-gemini-cli-acp-2026 (F3)** | Hooks + memory + Gemini defaults | Baixo (2 dias) | Baixo | F1 + F3-Claude entregues |
| **prd-gemini-cli-acp-2026 (F4)** | Métricas Gemini-2026 | Baixo (3 dias) | Baixo–Médio | F1 entregue |
| **prd-gemini-cli-acp-2026 (F5)** | Auto-review opt-in (cascata) | Baixo (1 dia) | Baixo | F1 + F5-Claude entregues |

Sugestão de decomposição: **um único PRD** com RF-N cobrindo F0..F5 e TechSpec subdividindo em waves. Alternativa: dois PRDs (F0+F1 isolados; F2..F5 como pacote de cascata). A decisão fica para o autor do PRD após aprovação deste pacote.

**Total estimado**: ~3.5 sprints (15 dias úteis, ~700 LoC novo + ~30 testes).

---

## Referências Cruzadas

**Compozy (leitura via `gh` — SHA `7f38c44506`)**:

- `go.mod` — pin `github.com/coder/acp-go-sdk v0.6.3` (mesmo SDK de Claude/Codex/Copilot)
- `internal/core/agent/registry_specs.go` — Spec Gemini (linhas 188-208 aprox., bloco `model.IDEGemini`)
- `internal/core/agent/registry_specs.go::supportedRegistryIDEOrder` — ordem `Claude, Codex, Droid, Cursor, OpenCode, Pi, Gemini, Copilot`
- `internal/core/agent/tool_call_name.go:84` — Gemini em grupo de drivers com aliases comuns (sem override)
- `internal/core/model/constants.go` — `IDEGemini = "gemini"`, `DefaultGeminiModel = "gemini-2.5-pro"`
- `internal/core/agents/mcpserver/server.go` — `reservedToolName = "run_agent"` (tool-agnóstico)
- `internal/core/agents/session_mcp.go` — `BuildSessionMCPServers`, `NestedExecutionContext` (tool-agnóstico)
- `internal/core/prompt/common.go` — `BuildSystemPromptAddendum`, hooks `prompt.pre_build`/`prompt.post_build`
- `internal/core/memory/store.go` — Memória 2-tier (limites 150/12KB e 200/16KB; tool-agnóstico)
- `internal/core/run/executor/review_hooks.go` — Auto-review como `ExecutionModePRReview`
- `internal/setup/runtime_agents.go` — `"gemini": "gemini-cli"` (mapping para setup)
- `internal/setup/agents.go` — referência a `.gemini/antigravity/skills` (setup compozy específico, não aplicável ao harness)
- Registro arquivado de integração ACP no Compozy — mencionava `--experimental-acp, --model <model>` (histórico; hoje é `--acp` estável)
- `README.md` (raiz Compozy) — "Multi-agent execution. Run tasks through ACP-capable runtimes like Claude Code, Codex, Cursor, Droid, OpenCode, Pi, or Gemini"

**ai-spec-harness (estado em `feat/codex-acp-spec`)**:

- `internal/runtime/specs/spec.go:29-101` — `Spec`, `AccessMode`, `BootstrapArgsFunc` (consumido pelo futuro `Gemini()`)
- `internal/runtime/specs/claude.go:28-45` — template canônico (sem `BootstrapArgs`)
- `internal/runtime/specs/codex.go:52-113` — referência de runtime com `BootstrapArgs` dinâmico (não aplicável a Gemini)
- `internal/runtime/specs/copilot.go` — referência mais próxima de Gemini (`copilot --acp` ↔ `gemini --acp`)
- `internal/runtime/specs/gemini.go` — **ausente; criar em F0-Gemini**
- `internal/runtime/runner.go:89-220` — `ACPRunner.Run()` (tool-agnóstico após F2-F5-Claude)
- `internal/runtime/client/client.go` — Cliente ACP (tool-agnóstico)
- `internal/runtime/events/convert.go` — alvo F4-Gemini (extração de métricas)
- `internal/runtime/events/metrics.go` — referência de `ExtractClaudeMetrics`; estender com `ExtractGeminiMetrics` em F4
- `internal/runtime/events/normalize.go` — alvo F2-Gemini (entrada YAML)
- `internal/runtime/mcpserver/` — tool-agnóstico após F2-Claude; Gemini cascata em F2-Gemini
- `internal/runtime/hooks/dispatcher.go` — tool-agnóstico após F3-Claude
- `internal/runtime/memory/store.go` — tool-agnóstico após F3-Claude; defaults Gemini via `task_loop.go`
- `internal/runtime/runner_autoreview.go` — tool-agnóstico após F5-Claude
- `internal/runtime/probe/probe.go` — Probe (estável)
- `internal/runtime/persistence/session.go` — Forense (estável)
- `internal/runtime/watchdog.go` — `ActivityWatchdog` (estável)
- `internal/agents/registry.go` — Registry F1 (ADR-011) consumido por MCP `run_agent`
- `internal/specdrift/specdrift.go` — `spec-hash` validation (preservar como concern de governança)
- `internal/evidence/evidence.go:29-49` — alvo F4-Gemini (campos Gemini-2026)
- `internal/telemetry/` — Telemetria opt-in ADR-006 (estender para Gemini métricas)
- `internal/wrapper/wrapper.go:14-18` — `ValidTools["gemini"] = true` (manter durante transição; remover em N+2)
- `internal/wrapper/wrapper.go:79-103` — `buildInstruction("gemini", ...)` (legado; manter durante transição)
- `internal/taskloop/compatibility.go:34-43` — Modelos Gemini já catalogados (validar em F1)
- `cmd/ai_spec_harness/task_loop.go:27-31` — `runtimeACPCatalog` (registrar Gemini em F0)
- `GEMINI.md` — atualizar com §"Runtime Capabilities (F0+/F2+/F3+/F4+/F5+)"
- `.gemini/hooks/` — Shell hooks atuais (coexistem com Go hooks após F3-Gemini)
- `.gemini/commands/` — Wrappers TOML legados (desativáveis após F1+)
- `.gemini/agents/` — Agentes declarativos (resolvidos via registry após F2-Gemini)
- ADR-006 (`docs/adr/006-telemetria-feedback-cycle.md`) — Telemetria opt-in (Gemini herda)
- ADR-009 (`.specs/adr/009-acp-protocol-adoption.md`) — Pinning SDK (precedente)
- ADR-011 (`.specs/adr/011-agent-registry-declarativo.md`) — Registry F1
- ADR-012 (`.specs/adr/012-copilot-cli-acp-native.md`) — Copilot ACP (modelo mais próximo de Gemini: CLI principal + flag `--acp`)
- ADR-013 (`.specs/adr/013-codex-cli-acp-native.md`) — Codex ACP (precedente de adapter dedicado)
- ADR-015 (`.specs/adr/015-gemini-cli-acp-native.md`) — **este pacote** (proposto)

**Pesquisa correlata**:

- [`compozy-adaptation-analysis.md`](compozy-adaptation-analysis.md) — análise genérica 10 dimensões
- [`compozy-adaptation-claude-2026.md`](compozy-adaptation-claude-2026.md) — F2-F5-Claude proposto (infra compartilhada por F2-F5-Gemini)
- [`compozy-adaptation-codex-2026.md`](compozy-adaptation-codex-2026.md) — F1-Codex entregue (precedente de runtime novo no harness)
- [`compozy-adaptation-copilot-2026.md`](compozy-adaptation-copilot-2026.md) — F1-Copilot entregue (precedente mais próximo: CLI principal + `--acp`)
- [`docs/prompts/compozy-adaptation-research-gemini.md`](../prompts/compozy-adaptation-research-gemini.md) — prompt enriquecido de origem

---

## Apêndice — Comparação Gemini vs Claude vs Codex vs Copilot (visão interna do compozy)

| Aspecto | Claude | Codex | Copilot | **Gemini** |
|---|---|---|---|---|
| **Command** | `claude-agent-acp` | `codex-acp` | `copilot` | **`gemini`** |
| **FixedArgs** | (vazio) | (vazio; usa BootstrapArgs) | `["--acp"]` | **`["--acp"]`** |
| **ProbeArgs** | (vazio) | (vazio) | `["--acp", "--help"]` | **`["--acp", "--help"]`** |
| **BootstrapArgs** | `nil` | `codexBootstrapArgs` (dinâmico) | `nil` | **`nil`** |
| **DefaultModel** | `opus` | `gpt-5.5` | `claude-sonnet-4.6` | **`gemini-2.5-pro`** |
| **Context Window** | ~200K | ~200K | ~200K | **1M+** ⭐⭐ |
| **Reasoning Effort** | thinking_tokens implícito | configurável via `-c` | n/a | **`thoughts_tokens` opt-in** |
| **Sandbox / Access mode** | `--bypass-permissions` | `approval_policy`, `sandbox_mode` | nenhum | **nenhum (n/a)** |
| **Cache** | prompt caching (5-min TTL implícito) | n/a | n/a | **context caching (TTL configurável)** |
| **Hooks ecosystem (harness)** | `.claude/hooks/*.sh` + Go in-process (F3) | nenhum específico | nenhum específico | **`.gemini/hooks/*.sh` + Go in-process (F3-Gemini cascata)** |
| **Fallback Package** | `@agentclientprotocol/claude-agent-acp` | `@zed-industries/codex-acp` | `@github/copilot` | **`@google/gemini-cli`** |
| **`wrapper.ValidTools`** | ❌ (usa hooks) | ✅ (legado, deprecated) | ✅ | **✅ (legado; deprecation planejada N+2)** |
| **Status harness 2026-05** | F1 entregue; F2-F5 propostos | F1 entregue | F1 entregue | **🔴 F0 ausente — gap raiz** |
| **Métricas distintivas (F4)** | `cache_read`, `cache_creation`, `thinking_tokens` | n/a | n/a | **`cache_read`, `effective_context`, `prompt_billed`, `thoughts`** |

**Conclusão do apêndice**: Gemini é o **único runtime ACP-capable atrasado** dentro do portfólio do harness — Claude, Codex e Copilot já têm Spec dedicada em `internal/runtime/specs/`, enquanto Gemini permanece em modo wrapper legado. A descoberta de que `@google/gemini-cli` já implementa ACP nativo via `gemini --acp` torna o gap **resolvível com baixo risco e baixo esforço** (~50 LoC para F0-Gemini, paridade com `claude.go`). Uma vez fechado o gap arquitetural, Gemini herda automaticamente toda a cascata F2-F5 da Claude wave (MCP, normalização, hooks, memory, métricas, auto-review) com apenas ajustes Gemini-específicos para aproveitar sua vantagem competitiva distintiva (janela 1M+, context caching configurável, modelo de cobrança via `prompt_tokens_billed`).

---

## Adendo — Validação do `@google/gemini-cli@0.43.0` (2026-05-22)

Após a primeira versão desta pesquisa, foi feito probe real do pacote npm `@google/gemini-cli@0.43.0` via `npx --yes ... --acp --help`. Achados que **refinam o roadmap proposto**:

### Flag `--acp` confirmada como estável

Saída de `gemini --help` (0.43.0) registra:

```
--acp                  Starts the agent in ACP mode  [boolean]
--experimental-acp     Starts the agent in ACP mode (deprecated, use --acp instead)  [boolean]
```

Confirma que a Spec proposta (`Command:"gemini"`, `FixedArgs:["--acp"]`) é exatamente o que o pacote estável aceita; `--experimental-acp` é alias deprecated mantido para compatibilidade. **Risco #1 da §"Riscos e Premissas" cai de Médio para Baixo**: a flag tem alias de compatibilidade explícito, então mesmo se houver rename futuro, haverá período de transição.

### Subcomando nativo `gemini hooks migrate`

Saída de `gemini hooks --help`:

```
gemini hooks <command>
Manage Gemini CLI hooks.
Commands:
  gemini hooks migrate    Migrate hooks from Claude Code to Gemini CLI
```

**Implicação para F3-Gemini**: Google explicitamente antecipou portabilidade de hooks Claude Code → Gemini. Isso significa que `.gemini/hooks/*.sh` no harness pode evoluir em duas direções complementares:

1. **Modo orquestrado (harness)**: hooks Go in-process via `internal/runtime/hooks/dispatcher.go` (cascata F3-Claude), conforme já proposto.
2. **Modo interativo (CLI direta)**: usuário pode rodar `gemini hooks migrate` uma vez para portar `.claude/hooks/*.sh` → formato esperado pela CLI Gemini, eliminando divergência entre `.claude/hooks/` e `.gemini/hooks/` (que hoje são mirrors mantidos manualmente via `scripts/sync-hooks`).

Sugestão para F3-Gemini: documentar `gemini hooks migrate` em `GEMINI.md` §"Hooks de Governanca" como **opção upstream** (não obrigatória; preserva o mirror manual como fallback determinístico).

### Subcomando nativo `gemini skills`

Saída de `gemini skills --help`:

```
gemini skills <command>
Manage agent skills.
Commands:
  list      Lists discovered agent skills.
  enable    Enables an agent skill.
  disable   Disables an agent skill.
  install   Installs an agent skill from a git repository URL or a local path.
  link      Links an agent skill from a local path.
  uninstall Uninstalls an agent skill by name.
```

**Implicação para F1-Gemini**: a abordagem atual do harness — wrappers TOML em `.gemini/commands/workspace.*.toml` apontando para `.agents/skills/<name>/` — pode coexistir com (ou ser substituída por) `gemini skills link <path>` que adiciona uma skill local ao discovery nativo da CLI Gemini.

Sugestão para F1-Gemini: validar via `gemini skills list` se as skills do harness (linkadas via wrapper TOML hoje) aparecem no discovery nativo. Se sim, a deprecation do wrapper Gemini (planejada para release N+2 conforme RF-06 do PRD) pode usar `gemini skills link` como caminho de migração — sem reescrever os adapters TOML manualmente.

### Flag `--approval-mode` revela paridade com Codex

Saída de `gemini --help` também lista:

```
--approval-mode    Set the approval mode: default (prompt for approval),
                   auto_edit (auto-approve edit tools),
                   yolo (auto-approve all tools),
                   plan (read-only mode)
```

**Implicação para F0-Gemini**: o `AccessModeFlag` na `Spec` permanece vazio (Gemini não tem flag estática como Claude `--bypass-permissions`), mas o mapeamento dinâmico de `AccessMode` → `--approval-mode` é introduzido via `BootstrapArgs` minimal. Os outros valores aceitos pela CLI (`auto_edit`, `plan`) são deliberadamente não usados — preservam simetria binária com Claude/Codex.

**Decisão tomada** (ADR-015 D-05, ver `.specs/adr/015-gemini-cli-acp-native.md`): mapeamento literal `AccessModeRestricted → "default"` e `AccessModeFull → "yolo"`. Testes T-29/T-30/T-31 (declarados na ADR) validam o mapeamento. Esta decisão **diverge intencionalmente do Compozy** (que mantém `BootstrapArgs: nil`) para aproveitar capability exposta pela CLI Gemini 0.43.0 que Compozy ainda não explora. Risco de drift documentado em ADR-015 §"Consequências/Negativas".

### Subcomando `gemini mcp`

Embora não explorado neste adendo, a CLI 0.43.0 também expõe `gemini mcp` (mencionado em `gemini --help`). Isso indica suporte nativo a clientes MCP — relevante para F2-Gemini, onde o servidor MCP do harness (`internal/runtime/mcpserver/`, F2-Claude) precisa ser **consumido** pelo Gemini. Compozy comprova que isso funciona via ACP (Gemini parent invoca `run_agent` sem configuração adicional), mas seria útil validar via `gemini mcp` quais MCPs estão configurados antes de habilitar `--mcp-nested` em produção.

Investigação detalhada do `gemini mcp` fica para F2-Gemini (não escopo desta pesquisa).
