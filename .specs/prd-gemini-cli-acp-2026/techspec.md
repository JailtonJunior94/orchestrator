<!-- spec-hash-prd: f23aac99a055ce0b697646439bd80fd37b79239852c22e4c90a7cddade4dcbff -->
<!-- MANDATÓRIO: preenchido por `create-technical-specification` Etapa 7.1 com sha256 do PRD consumido.
     Rastreabilidade: `create-tasks` e `execute-task` comparam este hash com o atual do prd.md
     para detectar drift entre techspec e PRD. NÃO remover este comentário ao editar a techspec. -->

# Especificação Técnica — Gemini CLI via ACP Nativo (F0..F5-Gemini)

> **PRD consumido**: [prd.md](./prd.md) (spec-version 1, 33 RFs)
> **ADR material**: [015-gemini-cli-acp-native](../adr/015-gemini-cli-acp-native.md)
> **Precedentes diretos**:
>   - [F1-Codex techspec](../prd-codex-acp-spec/techspec.md) — `BootstrapArgsFunc` + `AccessMode` (reusados sem extensão)
>   - [F1-Copilot techspec](../prd-copilot-acp-spec/techspec.md) — CLI principal com flag `--acp`
>   - [F2-F5-Claude techspec](../prd-claude-cli-acp-2026/techspec.md) — infra tool-agnóstica (mcpserver, hooks, memory, autoreview)
> **Insumo de pesquisa**: [docs/research/compozy-adaptation-gemini-2026.md](../../docs/research/compozy-adaptation-gemini-2026.md) (968 linhas)
> **Fase**: F0..F5-Gemini (pacote único — adaptação Compozy 2026 variante Gemini)

## Resumo Executivo

Introduz suporte ao Gemini CLI como runtime ACP nativo via novo construtor `specs.Gemini()` em `internal/runtime/specs/gemini.go`, invocando o binário `gemini` (pacote npm `@google/gemini-cli@0.43.0`) com flag `--acp`. Probe via `npx --yes @google/gemini-cli@0.43.0 --acp --help` em 2026-05-22 confirma que a flag `--acp` é estável; `--experimental-acp` permanece como alias deprecated upstream. Espelha estruturalmente o padrão Copilot (`copilot --acp`) e diverge intencionalmente de Compozy (`BootstrapArgs: nil`) ao introduzir `geminiBootstrapArgs` que mapeia `AccessMode` em flag `--approval-mode` (D-05 de ADR-015).

A integração reaproveita 100% da infraestrutura ACP **e** da cascata F2-F5 já entregue para Claude:

1. **F0/F1-Gemini** registra Gemini no `runtimeACPCatalog` em `cmd/ai_spec_harness/task_loop.go:27-31` e adiciona Spec ao catálogo `internal/runtime/specs/`. Nenhuma extensão estrutural — `BootstrapArgsFunc` e `AccessMode` foram introduzidos por ADR-013/F1-Codex e estão reusados como-são.
2. **F2-Gemini** acresce entrada `gemini: { inherit: common, overrides: {} }` em `.agents/normalization-rules.yaml` (e arquivo embedded). Valida cascata do `internal/runtime/mcpserver/` (F2-Claude) via teste de integração — zero código novo.
3. **F3-Gemini** adiciona switch tool-aware em `task_loop.go` para defaults de memory generosos (250/400 linhas, aproveitando janela 1M+ tokens do `gemini-2.5-pro`). Valida cascata do `internal/runtime/hooks/dispatcher.go` (F3-Claude) via teste.
4. **F4-Gemini** introduz `internal/runtime/events/gemini_metrics.go` com extração defensiva (`omitempty`) de quatro campos distintivos: `cache_read_tokens`, `effective_context_tokens`, `prompt_tokens_billed`, `thoughts_tokens`. Schema é resiliente a renomeações upstream — ausência retorna zero-value silencioso.
5. **F5-Gemini** valida cascata do `internal/runtime/runner_autoreview.go` (F5-Claude) com Gemini driver — zero código novo, apenas teste e documentação de trade-off de custo em janelas 1M+.

Total: ~700 LoC novo (Spec ~50, BootstrapArgs ~20, GeminiMetrics ~80, defaults gen ~30, testes ~400, docs/CLI-schema/CHANGELOG/GEMINI.md ~120) em **3.5 sprints**. Diff zero em `internal/runtime/specs/spec.go`, `internal/runtime/client/client.go`, `internal/runtime/runner.go` (apenas para extração de métricas Gemini), `internal/runtime/persistence/`, `internal/runtime/watchdog.go`, `internal/runtime/mcpserver/`, `internal/runtime/hooks/`, `internal/runtime/memory/store.go`, `internal/runtime/runner_autoreview.go`. Wrapper legado em `internal/wrapper/wrapper.go:91-95` é mantido com aviso de depreciação (remoção em release N+2).

## Arquitetura do Sistema

### Visão Geral dos Componentes

Camadas afetadas, agrupadas por fase:

**F0-Gemini — Spec registration (novo + edição)**
- **NOVO**: `internal/runtime/specs/gemini.go` — construtor `Gemini()` retornando `Spec` + função `geminiBootstrapArgs` + constantes pinadas.
- **EDIÇÃO**: `cmd/ai_spec_harness/task_loop.go:27-31` — entrada `"gemini": specs.Gemini` no `runtimeACPCatalog`.
- **EDIÇÃO**: `internal/runtime/probe/probe.go` — entrada `"gemini": ".specs/adr/015-gemini-cli-acp-native.md"` em `adrByID`.

**F1-Gemini — Paridade ACP E2E (edição + testes)**
- **EDIÇÃO**: `internal/taskloop/taskloop.go::Service.Execute` — roteamento Gemini via `ACPRunner` quando `Runtime == "acp"`. Mudança aditiva: switch case atual estendido com `case "gemini":` no mesmo padrão de claude/codex/copilot. Propaga `Options.AccessMode` para `Job.AccessMode`.
- **EDIÇÃO**: `internal/wrapper/wrapper.go::buildInstruction("gemini", ...)` — emite log WARNING único via `sync.Once`.
- **NOVOS**: `internal/runtime/specs/gemini_test.go`, `tests/integration/gemini_acp_smoke_test.go`, sub-suite Gemini em `internal/runtime/acp_integration_test.go`.

**F2-Gemini — MCP nested-agent + tool-call normalization (edição YAML + testes)**
- **EDIÇÃO**: `.agents/normalization-rules.yaml` — entrada `gemini: { inherit: common, overrides: {} }`.
- **EDIÇÃO**: `internal/runtime/events/normalization-rules.yaml` — mesma entrada (mirror embedded via `go:embed`).
- **EDIÇÃO MENOR**: `internal/runtime/events/normalize.go` — função `resolveInherit` consome `inherit: common` se ainda não suporta (provavelmente já implementado em F2-Claude; validar). Sem se ainda não, ~10 LoC.
- **NOVO**: `tests/integration/gemini_mcp_nested_test.go` — Gemini parent invoca `run_agent("reviewer", ...)`.

**F3-Gemini — Hooks + Memory 2-tier com defaults Gemini-generosos (edição CLI + testes)**
- **EDIÇÃO**: `cmd/ai_spec_harness/task_loop.go` — switch tool-aware aplicando defaults Gemini quando `--tool gemini` e flags `--memory-*` não foram setadas explicitamente.
- **NOVO**: `tests/integration/gemini_hooks_test.go` — valida hook `runtime.pre_open` despachado.
- **NOVO**: `internal/runtime/memory/store_gemini_defaults_test.go` (ou bloco em `task_loop_test.go`) — T-34/T-35 validando defaults e override.

**F4-Gemini — Métricas Gemini-2026 (novo + edição)**
- **NOVO**: `internal/runtime/events/gemini_metrics.go` (~80 LoC) — tipo `GeminiMetrics` + `ExtractGeminiMetrics(raw)`.
- **EDIÇÃO**: `internal/runtime/runner.go::Summary` — campos opcionais `GeminiCacheReadTokens`, `GeminiEffectiveContextTokens`, `GeminiPromptTokensBilled`, `GeminiThoughtsTokens`.
- **EDIÇÃO**: `internal/runtime/events/convert.go` — chamada a `ExtractGeminiMetrics` quando `driver_id == "gemini"`. Schema é defensivo: ausência absorvida.
- **EDIÇÃO**: `internal/evidence/evidence.go` — aceita seção opcional "Métricas Gemini-2026" em `execution_report.md`.
- **EDIÇÃO**: `internal/telemetry/telemetry.go` — registra entries `gemini.cache_read`, `gemini.thoughts`, `gemini.effective_context` quando `GOVERNANCE_TELEMETRY=1`.

**F5-Gemini — Auto-review opt-in (testes + docs)**
- Zero código novo no `internal/runtime/runner_autoreview.go` (cascata F5-Claude tool-agnóstica).
- **NOVO**: `internal/runtime/runner_autoreview_gemini_test.go` (T-39).
- **EDIÇÃO**: `GEMINI.md` — documenta custo amplificado em janelas 1M+ (seção F5).

**Transversal (toda a entrega)**
- **EDIÇÃO**: `GEMINI.md` — reescrita com seções "Runtime Capabilities (F0+/F2+/F3+/F4+/F5+)".
- **EDIÇÃO**: `AGENTS.md` — linha adicional na tabela de ADRs (ADR-015).
- **EDIÇÃO**: `docs/cli-schema.json` — `gemini` em enum de `--tool` quando `--runtime=acp`.
- **EDIÇÃO**: `CHANGELOG.md` — entradas F0..F5-Gemini.
- **EDIÇÃO**: `docs/telemetry-feedback-cycle.md` — invariantes Gemini ACP cobrem mesmos kinds que Claude/Codex/Copilot, com aditivos Gemini-2026.

### Relacionamentos entre Componentes

```
[task-loop CLI] --runtime acp --tool gemini --access-mode {restricted,full}
       │
       ▼
[runtimeACPCatalog]  ──►  specs.Gemini()  ──►  Spec{Command:"gemini", FixedArgs:["--acp"], BootstrapArgs:geminiBootstrapArgs}
       │
       ▼
[ACPRunner.Run(ctx, Job)]
       │   ┌── Job{Model, AccessMode, ReasoningEffort(ignored), AddDirs(ignored)}
       │
       ├── spec.BootstrapArgs(Job.Model, Job.ReasoningEffort, Job.AddDirs, Job.AccessMode)
       │       ──►  []string{"--approval-mode", "default"|"yolo"}   (D-05)
       │
       ├── prepend BootstrapArgs to argv  ──►  spec.FixedArgs (["--acp"])  ──►  exec.Command
       │
       ├── HOOK dispatch (F3-Claude infra) — runtime.pre_open, prompt.pre_build, ...
       │
       ├── MEMORY load (F3-Claude infra) — defaults Gemini-generosos via task_loop switch
       │
       ├── ACP loop (client.go, events package — sem mudança)
       │       ├── on session_update with usage.{cache_read_tokens, ...}
       │       └── ExtractGeminiMetrics(raw)  ──►  Summary.Gemini*  (F4)
       │
       ├── MCP nested-agent (F2-Claude infra) — run_agent(...)
       │
       └── auto-review (F5-Claude infra) — spawn child ACP session
                │
                ▼
       [SessionPersistence] — events.jsonl, tool_calls.md, execution_report.md
                │
                ▼
       [evidence.Validate] — agora aceita seção opcional "Métricas Gemini-2026"
```

Fluxo idêntico a Claude/Codex/Copilot exceto pelos pontos marcados como Gemini-específicos (`geminiBootstrapArgs`, defaults Gemini-generosos em `task_loop.go`, `ExtractGeminiMetrics`).

## Design de Implementação

### Interface Pública — `specs.Gemini()`

```go
// internal/runtime/specs/gemini.go

const (
    GeminiNpmPackage   = "@google/gemini-cli"
    GeminiNpmVersion   = "0.43.0"  // ADR-015 D-02; audit/ atualiza
    GeminiSDKVersion   = "v0.13.0"  // mesma de Claude/Codex/Copilot
    DefaultGeminiModel = "gemini-2.5-pro"
)

func Gemini() Spec {
    return newSpecWithBootstrap(
        "gemini",
        "Gemini (ACP)",
        "gemini",
        []string{"--acp"},
        []FallbackLauncher{{
            Command:   "npx",
            FixedArgs: []string{"--yes", GeminiNpmPackage + "@" + GeminiNpmVersion, "--acp"},
        }},
        "", // AccessModeFlag vazio — ADR-015 D-04
        GeminiSDKVersion, GeminiNpmVersion, GeminiNpmPackage,
        geminiBootstrapArgs,
    )
}

// geminiBootstrapArgs implementa o mapeamento literal D-05 de ADR-015.
// Model, reasoning e addDirs são ignorados intencionalmente (sublinhados):
//   - model: propagado via --model separado pelo ACPRunner
//   - reasoning: Gemini 2.5 não expõe controle programático
//   - addDirs: CLI Gemini 0.43.0 não expõe flag equivalente a Claude --add-dir
func geminiBootstrapArgs(_, _ string, _ []string, mode AccessMode) []string {
    switch mode {
    case AccessModeFull:
        return []string{"--approval-mode", "yolo"}
    case AccessModeRestricted, "":
        return []string{"--approval-mode", "default"}
    default:
        return []string{"--approval-mode", "default"}
    }
}
```

Construtor `newSpecWithBootstrap` já existe em `internal/runtime/specs/spec.go:90-101` (ADR-013). **Nenhuma extensão da interface `Spec` necessária.**

### Interface Pública — Extração de Métricas Gemini-2026

```go
// internal/runtime/events/gemini_metrics.go

// GeminiMetrics captura métricas Gemini-2026 do payload acp.SessionUpdate.
// Todos os campos são opcionais — ausência retorna zero-value silencioso (RF-18).
//
// Diferenças vs Claude-2026 (compozy-adaptation-gemini-2026.md §"Mecânica"):
//   - cache_creation_tokens não existe em Gemini (modelo de cache implícito)
//   - thoughts_tokens (Gemini) ↔ thinking_tokens (Claude) — semântica distinta
//   - prompt_tokens_billed e effective_context_tokens são exclusivos de Gemini
type GeminiMetrics struct {
    CacheReadTokens        int `json:"cache_read_tokens,omitempty"`
    EffectiveContextTokens int `json:"effective_context_tokens,omitempty"`
    PromptTokensBilled     int `json:"prompt_tokens_billed,omitempty"`
    ThoughtsTokens         int `json:"thoughts_tokens,omitempty"`
}

// ExtractGeminiMetrics lê os campos Gemini-específicos do raw payload do
// acp.SessionUpdate. Retorna GeminiMetrics{} zero-value quando o payload não
// contém os campos (silencioso, alinhado com TD-02 — schema defensivo).
//
// Chamada por internal/runtime/events/convert.go quando driver_id == "gemini".
// Nunca retorna erro para campos ausentes; só falha em JSON syntactically inválido.
func ExtractGeminiMetrics(raw json.RawMessage) (GeminiMetrics, error) {
    var envelope struct {
        Usage GeminiMetrics `json:"usage"`
    }
    if len(raw) == 0 {
        return GeminiMetrics{}, nil
    }
    if err := json.Unmarshal(raw, &envelope); err != nil {
        return GeminiMetrics{}, err
    }
    return envelope.Usage, nil
}

// LogGeminiMetrics escreve no telemetria opt-in (`GOVERNANCE_TELEMETRY=1`)
// entries com prefixo "gemini." quando os campos são não-zero. Espelha
// LogClaudeMetrics em events/metrics.go.
func LogGeminiMetrics(ctx context.Context, m GeminiMetrics) {
    // implementação delega para internal/telemetry — sem mudança aqui
}
```

### Modelos de Dados — Extensão de `Summary`

```go
// internal/runtime/runner.go (apenas o trecho extendido)

type Summary struct {
    // ... campos existentes preservados ...

    // Métricas Claude-2026 (F4-Claude, já existentes)
    CacheReadTokens     int `json:"cache_read_tokens,omitempty"`
    CacheCreationTokens int `json:"cache_creation_tokens,omitempty"`
    ThinkingTokens      int `json:"thinking_tokens,omitempty"`

    // Métricas Gemini-2026 (F4-Gemini, novo) — RF-19
    GeminiCacheReadTokens        int `json:"gemini_cache_read_tokens,omitempty"`
    GeminiEffectiveContextTokens int `json:"gemini_effective_context_tokens,omitempty"`
    GeminiPromptTokensBilled     int `json:"gemini_prompt_tokens_billed,omitempty"`
    GeminiThoughtsTokens         int `json:"gemini_thoughts_tokens,omitempty"`
}
```

Campos `omitempty` garantem que sessões Claude/Codex/Copilot não emitam ruído no JSON. Persistência forense (`internal/runtime/persistence/`) continua agnóstica — apenas serializa o `Summary` recebido.

### Switch tool-aware para defaults de memory (F3-Gemini)

```go
// cmd/ai_spec_harness/task_loop.go — trecho aditivo após parsing de flags

memoryWorkflowLines := defaultMemoryWorkflowLines  // 150 (constante F3-Claude)
memoryTaskLines := defaultMemoryTaskLines          // 200
memoryWorkflowBytes := defaultMemoryWorkflowBytes  // 12 * 1024
memoryTaskBytes := defaultMemoryTaskBytes          // 16 * 1024

if v, _ := cmd.Flags().GetInt("memory-workflow-limit-lines"); v != 0 {
    memoryWorkflowLines = v
}
// ... idem para os outros três ...

// RF-16: defaults Gemini-generosos quando --tool gemini e flags não setadas
if tool == "gemini" {
    if !cmd.Flags().Changed("memory-workflow-limit-lines") {
        memoryWorkflowLines = 250
    }
    if !cmd.Flags().Changed("memory-task-limit-lines") {
        memoryTaskLines = 400
    }
    if !cmd.Flags().Changed("memory-workflow-limit-bytes") {
        memoryWorkflowBytes = 20 * 1024
    }
    if !cmd.Flags().Changed("memory-task-limit-bytes") {
        memoryTaskBytes = 32 * 1024
    }
}
```

Override explícito via flag preservado. `internal/runtime/memory/store.go` continua agnóstico — recebe limites como parâmetros, não conhece o driver.

### Surface CLI Atualizada

```bash
# Comando recomendado (F0+F1-Gemini)
ai-spec-harness task-loop \
    --tool gemini \
    --runtime acp \
    --access-mode restricted \
    .specs/prd-minha-feature

# Modo full-access com warning (F0+F1)
ai-spec-harness task-loop \
    --tool gemini --runtime acp \
    --access-mode full \
    .specs/prd-minha-feature

# Modo avançado com MCP nested-agent (F2-Gemini cascata)
ai-spec-harness task-loop \
    --tool gemini --runtime acp \
    --mcp-nested \
    .specs/prd-minha-feature

# Memory generoso explícito (override default F3-Gemini)
ai-spec-harness task-loop \
    --tool gemini --runtime acp \
    --memory-task-limit-lines 600 \
    .specs/prd-minha-feature

# Auto-review opt-in (F5-Gemini cascata)
ai-spec-harness task-loop \
    --tool gemini --runtime acp \
    --auto-review \
    .specs/prd-minha-feature

# Modo wrapper legado (preservado durante transição, emite warning)
ai-spec-harness task-loop \
    --tool gemini \
    .specs/prd-minha-feature
```

Flags `--mcp-nested`, `--auto-review`, `--memory-*-limit-*` foram introduzidas por F2/F3/F5-Claude e são tool-agnósticas — Gemini cascata sem novas flags.

### Mensagens de Erro e Warning Literais (RF-08, RF-33)

```
WARNING (RF-08, único por execução via sync.Once):
  "Gemini wrapper legado (`gemini run --skill`) em uso. Migrar para --runtime=acp (binário `gemini` com `--acp`). Ver ADR-015."

WARNING (RF-33, único por execução via accessModeFullWarnOnce já existente):
  "WARNING: --access-mode=full ativa --approval-mode=yolo no gemini-cli.
   Pré-condição: consentimento operacional. Ver GEMINI.md."

INFO (F5-Gemini, uma vez por session quando --auto-review com --tool gemini):
  "INFO: --auto-review com Gemini pode amplificar custo de tokens em janelas 1M+. Ver GEMINI.md §F5."

ERRO de probe (RF-06, mensagem distingue binary vs npm package):
  "gemini não encontrado. Install via `npm install -g @google/gemini-cli@0.43.0`;
   OR instale o pacote npm direto; OR use wrapper legado sem --runtime=acp.
   Veja .specs/adr/015-gemini-cli-acp-native.md"
```

## Pontos de Integração

### `@google/gemini-cli` (dependência externa)

- **Pacote npm**: `@google/gemini-cli@0.43.0` (dist-tag `latest` em 2026-05-22).
- **Probe upstream**: `npx --yes @google/gemini-cli@0.43.0 --acp --help` confirma flag `--acp` estável; `--experimental-acp` é alias deprecated.
- **Protocolo**: ACP via stdio (`coder/acp-go-sdk v0.6.3`, mesmo SDK Go pinado no harness e Compozy — referenciado em ADR-009).
- **Modelo default**: `gemini-2.5-pro` (`DefaultGeminiModel`). CLI Gemini resolve modelo via `gemini config` ou flag `--model`; harness propaga via `--model` quando `Job.Model != ""`.
- **Subcomandos opcionais** (não usados programaticamente pelo harness, documentados em `GEMINI.md`):
  - `gemini hooks migrate` — migrador upstream Claude Code → Gemini, complementar a `.gemini/hooks/` mirror.
  - `gemini skills install/link` — discovery nativo de skills, alternativa ao `.gemini/commands/workspace.*.toml` em modo legado.
  - `gemini mcp` — cliente MCP nativo. Habilita consumo do `internal/runtime/mcpserver/` exposto pelo harness (F2-Gemini valida).
- **Auth**: pré-condição operacional do usuário (token Google ou OAuth via `gemini auth`). Harness não gerencia.

### Compozy (referência de design)

`compozy/compozy@7f38c445...` é a referência arquitetural. Esta techspec **diverge intencionalmente** em D-03/D-05 de ADR-015 (Compozy mantém `BootstrapArgs: nil`; harness emite `--approval-mode`). Convergência preservada em:

- `Command:"gemini"` + `FixedArgs:["--acp"]` (idêntico)
- Fallback `npx --yes @google/gemini-cli --acp` (idêntico, sem versão pinada upstream — harness pinneia para 0.43.0)
- `DefaultModel:"gemini-2.5-pro"` (idêntico, espelha `compozy/internal/core/model/constants.go::DefaultGeminiModel`)
- Tabela `commonToolTitleAliases` herdada (`tool_call_name.go:84` em Compozy confirma Gemini no grupo "drivers compartilhando aliases comuns")

### Tratamento de Erros

| Cenário | Comportamento |
|---|---|
| Binário `gemini` ausente do PATH | Fallback npx automático (`npx --yes @google/gemini-cli@0.43.0 --acp`). Mensagem registrada em telemetria com `launcher=npx`. |
| Binário `gemini` e `npx` ambos ausentes | Erro `exit2` com 3 remédios (RF-06). Referência a ADR-015. |
| `gemini --acp` retorna não-zero (auth missing, modelo inválido) | Erro propagado pelo `acpClient` — mesmo path de Claude/Codex/Copilot. |
| Payload ACP Gemini sem campos `cache_read_tokens` etc. | `ExtractGeminiMetrics` retorna zero-value silenciosamente (RF-18); `Summary.Gemini*` permanece zero; seção em `execution_report.md` é omitida. |
| Wrapper legado invocado | Funciona; emite warning único (RF-08); funcionalidade preservada para release N+2. |
| `--access-mode=full` passado | Warning único (RF-33) antes de spawn; propaga `--approval-mode=yolo` via `geminiBootstrapArgs`. |
| `--reasoning-effort high --tool gemini` | Aceito; ignorado por `geminiBootstrapArgs` (T-30 valida). Documentado em `--help`. |
| `--add-dir /tmp --tool gemini` (se flag exposta) | Aceito; ignorado por `geminiBootstrapArgs`. Documentado. |

## Abordagem de Testes

### Mapeamento RF → Componente → Teste

| RF | Componente | Teste(s) | Tipo |
|---|---|---|---|
| RF-01 | `specs/gemini.go::Gemini()` | T-14: `TestGeminiSpecHasCorrectCommandAndFlags` | unit |
| RF-02 | `specs/gemini.go::Gemini().Fallbacks` | T-15: `TestGeminiFallbackResolvesViaNpx` | unit |
| RF-03 | constantes em `specs/gemini.go` | T-14 + lint pinning | unit + static |
| RF-04 | `specs/gemini.go::geminiBootstrapArgs` | T-16, T-29, T-30, T-31 | unit |
| RF-05 | `task_loop.go::runtimeACPCatalog` | T-13: `TestRuntimeACPCatalogIncludesGemini` | unit |
| RF-06 | `probe/probe.go::adrByID` | extensão de teste existente `TestProbeReferencesADR` | unit |
| RF-07 | `taskloop/taskloop.go::Service.Execute` | extensão de `TestServiceRoutesToACPRunner` | unit |
| RF-08 | `wrapper/wrapper.go::buildInstruction` | `TestWrapperEmitsDeprecationWarningOnce` | unit |
| RF-09 | `runtime/acp_integration_test.go` Gemini sub-suite | sub-suite com fake ACP server | integration |
| RF-10 | `specs/gemini_test.go` | matriz completa (AccessMode × campos ignorados) | unit |
| RF-11 | `task_loop_test.go` | T-13/T-14/T-15/T-16 estendidos + T-29/T-30/T-31 | unit |
| RF-12 | `tests/integration/gemini_acp_smoke_test.go` | smoke real (skip via `-short`) | integration |
| RF-13 | `events/normalize.go` + YAML | T-32: `TestNormalizeToolCallGeminiInheritsCommon` | unit |
| RF-14 | `events/normalize_test.go` | T-32 + preservação de `raw_name` | unit |
| RF-15 | `mcpserver/` + Gemini | T-33: `TestMCPNestedAgentSpawnsGeminiSession` | integration |
| RF-16 | `task_loop.go` defaults Gemini | T-34, T-35 | unit |
| RF-17 | `task_loop_test.go` | T-34/T-35 | unit |
| RF-18 | `events/gemini_metrics.go` | T-36: `TestExtractGeminiMetricsFromACPPayload` (incl. payload vazio, payload parcial, payload completo) | unit |
| RF-19 | `runner.go::Summary` | extensão de `TestSummarySerialization` | unit |
| RF-20 | `evidence/evidence.go` | T-37: `TestEvidenceRendersGeminiMetricsSection`; T-38: `TestEvidenceMissingGeminiMetricsDoesNotBlock` | unit |
| RF-21 | acima | T-36/T-37/T-38 | unit |
| RF-22 | `runner_autoreview.go` + Gemini | T-39: `TestAutoReviewWithGeminiDriver` | integration |
| RF-23 | `GEMINI.md` | lint markdown + check manual | docs |
| RF-24 | `AGENTS.md` | lint markdown + check manual | docs |
| RF-25 | `docs/cli-schema.json` | extensão de `TestCLISchemaContainsAllTools` | unit |
| RF-26 | `CHANGELOG.md` | manual + lint conventional commits | docs |
| RF-27 | `docs/telemetry-feedback-cycle.md` | check manual | docs |
| RF-28 | `taskloop/compatibility.go` | validação manual via `TestCompatibilityTableContainsGemini` (já existe) | unit |
| RF-29 | `probe/probe.go` cache | extensão de `TestProbeCacheKey` | unit |
| RF-30 | regressão Claude/Codex/Copilot | suíte existente roda inalterada | regression |
| RF-31 | `telemetry/telemetry.go` | extensão de `TestTelemetryRecordsRuntimeInit` | unit |
| RF-32 | diff zero em N módulos | revisão de PR + `git diff --stat` | static |
| RF-33 | `task_loop.go::accessModeFullWarnOnce` | extensão de `TestAccessModeFullEmitsWarning` (já existe para Codex) | unit |

### Testes Unitários

**T-13 estendido — `TestRuntimeACPCatalogIncludesGemini`**
```go
func TestRuntimeACPCatalogIncludesGemini(t *testing.T) {
    spec, ok := runtimeACPCatalog["gemini"]
    if !ok { t.Fatal("gemini ausente do runtimeACPCatalog") }
    got := spec()
    if got.Command != "gemini" {
        t.Errorf("Command: got %q, want %q", got.Command, "gemini")
    }
    if len(got.FixedArgs) != 1 || got.FixedArgs[0] != "--acp" {
        t.Errorf("FixedArgs: got %v, want [--acp]", got.FixedArgs)
    }
}
```

**T-14 estendido — `TestGeminiSpecHasCorrectCommandAndFlags`**
Valida `ID`, `DisplayName`, `Command`, `FixedArgs`, `AccessModeFlag`, `Fallbacks`, e versão das constantes.

**T-15 estendido — `TestGeminiFallbackResolvesViaNpx`**
```go
func TestGeminiFallbackResolvesViaNpx(t *testing.T) {
    spec := specs.Gemini()
    require.Len(t, spec.Fallbacks, 1)
    fb := spec.Fallbacks[0]
    assert.Equal(t, "npx", fb.Command)
    assert.Equal(t, []string{"--yes", "@google/gemini-cli@0.43.0", "--acp"}, fb.FixedArgs)
}
```

**T-16 estendido — `TestGeminiBootstrapArgsForRestricted`**
```go
got := geminiBootstrapArgs("", "", nil, AccessModeRestricted)
require.Equal(t, []string{"--approval-mode", "default"}, got)
```

**T-29 novo — `TestGeminiBootstrapArgsForFull`**
```go
got := geminiBootstrapArgs("", "", nil, AccessModeFull)
require.Equal(t, []string{"--approval-mode", "yolo"}, got)
```

**T-30 novo — `TestGeminiBootstrapArgsIgnoresModelAndReasoning`**
```go
got := geminiBootstrapArgs("gemini-2.5-pro", "high", []string{"/tmp"}, AccessModeRestricted)
require.Equal(t, []string{"--approval-mode", "default"}, got)
// Crítico: model/reasoning/addDirs NÃO aparecem no output.
```

**T-31 novo — `TestGeminiBootstrapArgsDefaultsToRestricted`**
```go
got := geminiBootstrapArgs("", "", nil, "")  // zero-value AccessMode
require.Equal(t, []string{"--approval-mode", "default"}, got)
```

**T-32 novo — `TestNormalizeToolCallGeminiInheritsCommon`**
```go
out, err := BuildNormalizedToolCall("gemini", acp.ToolKindRead, "read_file", map[string]any{"path": "/x"}, nil)
require.NoError(t, err)
assert.Equal(t, "Read", out.NormalizedName)
assert.Equal(t, "read_file", out.RawName)
```

**T-34 novo — `TestGeminiDefaultsMemoryLimitsAreGenerous`**
```go
// CLI sem flag explícita; tool=gemini
cmd := newRootCmd()
cmd.SetArgs([]string{"task-loop", "--tool", "gemini", "--runtime", "acp", "--dry-run", ".specs/prd-fake"})
err := cmd.Execute()
require.NoError(t, err)
// Inspecionar limites resolvidos (via injeção de testing hook ou inspeção do plan)
assert.Equal(t, 250, memoryWorkflowLines)
assert.Equal(t, 400, memoryTaskLines)
```

**T-35 novo — `TestGeminiMemoryLimitOverrideByCliFlag`**
```go
cmd.SetArgs([]string{"task-loop", "--tool", "gemini", "--runtime", "acp",
    "--memory-task-limit-lines", "600", "--dry-run", ".specs/prd-fake"})
require.NoError(t, cmd.Execute())
assert.Equal(t, 600, memoryTaskLines)  // override prevalece
assert.Equal(t, 250, memoryWorkflowLines)  // workflow ainda default Gemini
```

**T-36 novo — `TestExtractGeminiMetricsFromACPPayload`**
Cobre três cenários:
1. Payload vazio (`[]`) → `GeminiMetrics{}` zero-value, sem erro.
2. Payload parcial (apenas `cache_read_tokens`) → outros campos zero, sem erro.
3. Payload completo (todos os 4 campos) → valores populados.
4. Payload com chaves inesperadas → ignoradas silenciosamente (`json.Unmarshal` permissivo).
5. Payload JSON inválido → erro propagado.

**T-37 novo — `TestEvidenceRendersGeminiMetricsSection`**
Quando `Summary.Gemini*` são todos não-zero, `evidence.Validate` aceita seção "Métricas Gemini-2026" com tabela markdown válida.

**T-38 novo — `TestEvidenceMissingGeminiMetricsDoesNotBlock`**
Quando `Summary.Gemini*` são todos zero, `evidence.Validate` permite ausência da seção sem retornar erro (RF-20).

### Testes de Integração

> **Decisão**: integration tests SÃO necessários para F1, F2, F3, F5 (cascata real envolve fronteiras de subprocess + protocolo ACP). Build tag `//go:build integration`.

- **`gemini_acp_smoke_test.go`** (F1, RF-12) — invoca `gemini --acp` real ou via fake server. Verifica produção de `events.jsonl` para task simples.
- **`gemini_mcp_nested_test.go`** (F2, RF-15) — Gemini parent invoca `run_agent("reviewer", ...)`. Valida cascata MCP.
- **`gemini_hooks_test.go`** (F3) — hook `runtime.pre_open` despachado quando driver é Gemini.
- **`gemini_autoreview_test.go`** (F5, RF-22) — `--auto-review --tool gemini --runtime acp` spawna sessão filha.

Suíte completa rodável via `go test -tags integration ./tests/integration/...`. Skip silencioso quando `gemini` indisponível no PATH.

### Testes E2E

Smoke test único em `tests/integration/gemini_acp_smoke_test.go` cobre fluxo end-to-end com task fixture (`.specs/fixtures/`). Não há novos testes E2E além disso — a cascata F2-F5 reusa fixtures E2E do Claude.

### Regressão Obrigatória

- 100% dos testes em `internal/runtime/specs/claude_test.go`, `codex_test.go`, `copilot_test.go` permanecem verdes (RF-30).
- `internal/runtime/specs/spec_test.go` sem mudança (interface `Spec` inalterada).
- `internal/runtime/runner_test.go` testes existentes verdes; apenas extensão em `TestSummarySerialization` para incluir campos Gemini (zeros não-emitidos por `omitempty`).
- `internal/runtime/persistence/`, `internal/runtime/watchdog.go`, `internal/runtime/client/client.go`, `internal/runtime/mcpserver/`, `internal/runtime/hooks/`, `internal/runtime/memory/store.go`, `internal/runtime/runner_autoreview.go` — diff zero (RF-32).

## Sequenciamento de Desenvolvimento

### Ordem de Build (wave-ordered até F1; F2..F5 paralelos)

```
F0 ──► F1 ──► (F2 || F3 || F4 || F5)  ──► docs/CHANGELOG/release
```

1. **F0-Gemini — Spec registration** (wave 0, ~3 dias):
   - `specs/gemini.go` (Spec + BootstrapArgs + constantes)
   - `task_loop.go::runtimeACPCatalog` (entrada gemini)
   - `probe/probe.go::adrByID` (entrada gemini)
   - Testes unit: T-13/T-14/T-15/T-16/T-29/T-30/T-31
   - **Saída**: `--tool gemini --runtime acp --dry-run` retorna comando esperado sem erro.

2. **F1-Gemini — Paridade ACP E2E** (wave 1, ~5 dias):
   - `taskloop.go::Service.Execute` (roteamento)
   - `wrapper.go::buildInstruction` (warning sync.Once)
   - `gemini_test.go` (matriz completa)
   - Sub-suite Gemini em `acp_integration_test.go`
   - Smoke test `tests/integration/gemini_acp_smoke_test.go`
   - **Saída**: `--tool gemini --runtime acp .specs/prd-X` completa task simples e produz `events.jsonl` + `tool_calls.md` + `execution_report.md`.

3. **F2-Gemini — MCP + normalization** (wave 2, paralelizável, ~3 dias):
   - YAML normalization entries (canônico + embedded)
   - `events/normalize.go::resolveInherit` (se necessário)
   - Teste T-32
   - Integration test `gemini_mcp_nested_test.go` (T-33)
   - **Dependência inter-PRD (Q9 do PRD)**: F2-Claude deve estar mergeada (provê `internal/runtime/events/normalize.go` e `internal/runtime/mcpserver/`).

4. **F3-Gemini — Hooks + Memory defaults** (wave 2, paralelizável, ~2 dias):
   - Switch tool-aware em `task_loop.go`
   - Testes T-34/T-35
   - Integration test `gemini_hooks_test.go`
   - **Dependência inter-PRD**: F3-Claude deve estar mergeada (provê `internal/runtime/hooks/dispatcher.go` e `internal/runtime/memory/store.go`).

5. **F4-Gemini — Métricas Gemini-2026** (wave 2, paralelizável, ~3 dias):
   - `events/gemini_metrics.go`
   - Extensão `Summary` em `runner.go`
   - Edição `events/convert.go` (chamada a `ExtractGeminiMetrics`)
   - Edição `evidence/evidence.go` (seção opcional)
   - Edição `telemetry/telemetry.go` (entries gemini.*)
   - Testes T-36/T-37/T-38
   - **Sem dependência inter-PRD**: independente das outras fases (mas precisa de F1 para ter sessão Gemini real).

6. **F5-Gemini — Auto-review** (wave 2, paralelizável, ~1 dia):
   - Integration test `gemini_autoreview_test.go` (T-39)
   - Documentação de trade-off em `GEMINI.md`
   - **Dependência inter-PRD**: F5-Claude deve estar mergeada (provê `internal/runtime/runner_autoreview.go`).

7. **Wave 3 — Documentação consolidada** (~1 dia):
   - Reescrita de `GEMINI.md` completa
   - Atualização de `AGENTS.md`, `docs/cli-schema.json`, `docs/telemetry-feedback-cycle.md`
   - `CHANGELOG.md` com entradas F0..F5

**Total**: ~3.5 sprints (F0+F1 = 8 dias críticos; F2/F3/F4/F5 em paralelo = ~3 dias após F1; wave 3 = 1 dia; buffer ~3 dias).

### Dependências Técnicas

**Bloqueantes (devem existir antes de F2/F3/F5-Gemini)**:
- `internal/runtime/events/normalize.go` (de F2-Claude) — bloqueia F2-Gemini
- `internal/runtime/mcpserver/` (de F2-Claude) — bloqueia F2-Gemini (RF-15)
- `internal/runtime/hooks/dispatcher.go` (de F3-Claude) — bloqueia F3-Gemini
- `internal/runtime/memory/store.go` (de F3-Claude) — bloqueia F3-Gemini (RF-16)
- `internal/runtime/runner_autoreview.go` (de F5-Claude) — bloqueia F5-Gemini (RF-22)

**Não-bloqueantes**:
- F4-Gemini (métricas Gemini-2026) é independente da cascata Claude — pode ser entregue logo após F1.

**Externas**:
- `@google/gemini-cli@0.43.0` disponível no npm registry (validado em 2026-05-22).
- `npx` disponível no PATH (assumido — mesma premissa de Claude/Codex/Copilot).
- `coder/acp-go-sdk v0.6.3` em `go.mod` (já presente).

## Monitoramento e Observabilidade

### Telemetria Opt-in (`GOVERNANCE_TELEMETRY=1`)

Estende `internal/telemetry/telemetry.go` com:

- **`runtime_init`** — ganha cardinalidade `tool=gemini`, `launcher=binary|npx`, `npm_version=0.43.0`, `sdk_version=v0.13.0` (RF-31). Sem novo kind de evento (ADR-010 preservado).
- **`gemini.cache_read=N`** — quando `GeminiMetrics.CacheReadTokens > 0`.
- **`gemini.effective_context=N`** — quando `GeminiMetrics.EffectiveContextTokens > 0`.
- **`gemini.prompt_billed=N`** — quando `GeminiMetrics.PromptTokensBilled > 0`.
- **`gemini.thoughts=N`** — quando `GeminiMetrics.ThoughtsTokens > 0`.

### Persistência Forense (sem mudança estrutural)

`internal/runtime/persistence/` continua escrevendo:
- `events.jsonl` — eventos ACP brutos (linha-a-linha; sem mudança de schema).
- `tool_calls.md` — agora renderiza com `normalized_name` (F2-Gemini) ao lado de `raw_name`.
- `execution_report.md` — ganha seção opcional "Métricas Gemini-2026" (F4-Gemini, RF-20).

### Métricas Operacionais

| Métrica | Onde | Quando |
|---|---|---|
| Probe latency p95 (gemini binary) | `probe_init` event | sempre (ADR-006) |
| Probe latency p95 (npx fallback) | `probe_init` event | quando launcher=npx |
| Session duration | `session_end` event | sempre |
| Token usage (Gemini-2026) | `runtime_init` + `gemini.*` entries | sempre que `Summary.Gemini*` não-zero |
| Wrapper legacy invocations | `runtime_init` com `runtime=wrapper` | sempre que wrapper invocado |

### Grafana (aditivo, opcional)

Dashboard existente "Multi-tool ACP Runtimes" ganha painéis Gemini:
- Cache hit rate Gemini (`gemini.cache_read / (gemini.cache_read + gemini.prompt_billed)`)
- Effective context utilization (`gemini.effective_context / 1_000_000`)
- Thoughts tokens distribuição

Não é parte do PRD/TechSpec — referência para PR opcional pós-F4-Gemini.

## Considerações Técnicas

### Decisões Chave

**TD-01 — Reuso da `BootstrapArgsFunc` introduzida em ADR-013**

- **Escolha**: usar `BootstrapArgsFunc` existente em vez de criar nova abstração.
- **Justificativa**: F1-Codex introduziu o ponto de extensão exato para casos como Gemini (config dinâmico). Reusar elimina duplicação e mantém simetria estrutural.
- **Alternativa rejeitada**: criar `GeminiSpec` separado com campo dedicado. Desnecessariamente fragmentaria o catálogo.

**TD-02 — Schema defensivo em `ExtractGeminiMetrics`**

- **Escolha**: extração silenciosa com `omitempty` + zero-value em ausência.
- **Justificativa**: payload ACP Gemini é validado upstream mas nomes exatos dos campos não são contratuais (A8 do PRD). Schema defensivo absorve renomeações futuras sem quebrar evidence.
- **Trade-off**: perda de visibilidade quando upstream renomeia (telemetria reportará zero). Mitigação: monitorar `gemini.cache_read=0` em todas as sessões — sinal claro de drift.
- **Alternativa rejeitada**: schema estrito com erro em campo desconhecido. Quebraria sessões Gemini em qualquer mudança upstream.

**TD-03 — Divergência intencional do Compozy em `BootstrapArgs`**

- **Escolha**: `geminiBootstrapArgs` emite `--approval-mode` mapeando `AccessMode`; Compozy mantém `BootstrapArgs: nil` para Gemini.
- **Justificativa**: capability `--approval-mode` existe na CLI Gemini 0.43.0 (probe validado); Compozy ainda não explora. Harness ganha controle granular alinhado com `AccessMode` cross-runtime.
- **Trade-off**: drift potencial se Compozy futuramente popular `--approval-mode` com semântica diferente (ex: usar `auto_edit` para algum modo).
- **Mitigação**: documentado em `geminiBootstrapArgs` comentário + `GEMINI.md`; revisão obrigatória quando `audit/` atualizar `GeminiNpmVersion`; testes T-29..T-31 fixam mapeamento explicitamente.
- **Alternativa rejeitada**: paridade exata com Compozy (sem `--approval-mode`). Descartaria capability útil já exposta upstream.

**TD-04 — Defaults Gemini-generosos para memory (250/400 linhas)**

- **Escolha**: switch tool-aware em `task_loop.go` aplica defaults Gemini quando `--tool gemini` e flags `--memory-*` não setadas.
- **Justificativa**: janela 1M+ tokens do `gemini-2.5-pro` torna defaults conservadores (150/200) sub-otimizados — overhead de 400 linhas é ~0,1% da janela.
- **Trade-off**: pode amplificar latência de prompt-build em workflows com memory cheia; mitigado pela métrica `effective_context_tokens` (F4) que permite observar utilização real.
- **Alternativa rejeitada**: defaults uniformes para todos os drivers. Desperdiçaria capability Gemini-específica.
- **Alternativa rejeitada**: mudar defaults globais para 250/400. Quebraria Claude/Codex/Copilot em workflows ajustados aos defaults conservadores atuais.

**TD-05 — Manter wrapper legado (`gemini run --skill`) durante transição**

- **Escolha**: `internal/wrapper/wrapper.go::ValidTools["gemini"] = true` preservado; warning único via `sync.Once` quando invocado sem `--runtime=acp`.
- **Justificativa**: alinha com política F1-Copilot Q5 e F1-Codex. Remoção destrutiva quebraria invocações em scripts/CI existentes.
- **Trade-off**: dois caminhos coexistem por 2 versões minor.
- **Alternativa rejeitada**: remoção imediata. Quebra retrocompatibilidade.

**TD-06 — Modo wave-ordered estrito até F1; paralelo após**

- **Escolha**: F0 → F1 sequencial; F2/F3/F4/F5 paralelos após F1.
- **Justificativa**: F0/F1 entregam infra mínima testável (Spec + roteamento); F2/F3/F5 são cascatas com dependência inter-PRD (Claude). F4 é independente.
- **Trade-off**: orquestração via `execute-all-tasks` precisa respeitar DAG; tarefas com dependência inter-PRD podem ficar `pending` se F2/F3/F5-Claude não mergeadas.
- **Alternativa rejeitada**: F0..F5 estritamente sequencial. Atrasa entrega em ~5 dias.

**TD-07 — Nenhuma ADR local em `.specs/prd-gemini-cli-acp-2026/`**

- **Escolha**: D-01..D-05 já documentados em ADR-015 (`.specs/adr/015-gemini-cli-acp-native.md`); TD-01..TD-07 desta techspec são decisões de implementação, não material.
- **Justificativa**: convenção observada em `prd-codex-acp-spec/` e `prd-claude-cli-acp-2026/` (ADR-013/014 globais, sem ADR locais).
- **Alternativa rejeitada**: criar `adr-001-defaults-memory-generosos.md` local. Verbosidade desnecessária; decisão é refletida em TD-04 + RF-16 + teste T-34.

### Riscos Conhecidos

| # | Risco | Mitigação |
|---|---|---|
| 1 | Flag `--acp` da CLI Gemini renomeada/removida em 0.44.x ou 1.0.0 | `--experimental-acp` mantida como alias deprecated indica ciclo de compat upstream; pinning via D-02 limita exposição; testes detectam regressão. |
| 2 | Schema do payload ACP Gemini diverge entre versões (campos métrica renomeados) | TD-02 schema defensivo; sinal de drift: `gemini.cache_read=0` em sessões longas. Audit/ revisa quando atualizar versão. |
| 3 | Subprocess `gemini` requer auth interativa (token Google) | Documentado em `GEMINI.md` como pré-condição operacional. Probe pode falhar silenciosamente se auth missing — propagar exit code para o usuário com mensagem clara. |
| 4 | Cascata F2-F5 com Claude quebra se F2/F3/F5-Claude regredirem | Dependência inter-PRD documentada em `tasks.md` Q9; testes T-32/T-33/T-39 detectam regressão. |
| 5 | Defaults Gemini-generosos amplificam custo de cache miss em prompts grandes | F4-Gemini expõe `cache_read_tokens` — visível para o operador. Override via flag `--memory-task-limit-lines` preservado. |
| 6 | Divergência Compozy em D-03/D-05 mascara bug upstream | Mapeamento literal D-05 (T-29/T-30/T-31) é fixed-table; mudança requer nova ADR. Comentário em `geminiBootstrapArgs` referencia ADR-015. |
| 7 | Auto-review em sessão Gemini com diff grande dobra cobrança | F5-Gemini emite INFO antes do spawn; documentado em `GEMINI.md` §F5. Flag opt-in (default off). |
| 8 | `thoughts_tokens` Gemini pode ser sempre zero (não exposto por default em Gemini 2.5) | Documentado como caveat conhecido (A8/Q8 do PRD); valor zero é semanticamente válido (não é erro). |
| 9 | Migração de hooks shell `.gemini/hooks/` via `gemini hooks migrate` upstream pode divergir do mirror manual | Mantém mirror manual via `scripts/sync-hooks` como fonte de verdade; `gemini hooks migrate` é opcional para modo interativo. |
| 10 | Modo wrapper legado e modo ACP coexistindo geram confusão em logs | Warning único (RF-08) referencia ADR-015; mensagem explícita sobre migração. |

### Conformidade com Padrões

| Regra | Local | Aderência |
|---|---|---|
| **R-GOV-001** (governança transversal) | `.claude/rules/governance.md` | ✅ Aderente — esta techspec não substitui agent-governance; reusa skill de Go via tasks. |
| **R-DDD-001** (value object imutável; construtor canônico) | governance ref ddd.md | ✅ `Spec` continua construído via `newSpec`/`newSpecWithBootstrap`; nenhum literal externo. |
| **R-SEC-001** (sem shell, args via `exec.Command` slice) | governance ref security.md | ✅ `BootstrapArgsFunc` retorna `[]string`; sem interpolação shell. Mensagem warn em `AccessModeFull` antes de propagar. |
| **R-DOC-001** (decisões materiais em ADR) | governance ref | ✅ D-01..D-05 em ADR-015 (global); TD-01..TD-07 desta techspec são impl notes, não decisões materiais. |
| **R-TST-001** (tabela RF → teste) | governance ref testing.md | ✅ Tabela explícita em §"Mapeamento RF → Componente → Teste". |
| **ADR-009** (pinning SDK) | `.specs/adr/009-...` | ✅ `GeminiNpmVersion="0.43.0"`, `GeminiSDKVersion="v0.13.0"`; atualização via audit. |
| **ADR-010** (tagged union de eventos) | `.specs/prd-acp-runtime-claude/adr-010-...` | ✅ Sem novo kind de evento; métricas Gemini-2026 são aditivas em `Summary`, não em event payload. |
| **ADR-013** (interface Spec com BootstrapArgs) | `.specs/adr/013-...` | ✅ Reuso direto; nenhuma extensão. |
| **ADR-006** (telemetria opt-in append-only) | `docs/adr/006-...` | ✅ Entries `gemini.*` só quando `GOVERNANCE_TELEMETRY=1`. |
| **ADR-002** (FakeFileSystem em testes) | `.specs/adr/002-...` | ✅ Testes usam `fs.FakeFileSystem` quando aplicável. |

### Arquivos Relevantes e Dependentes

**Novos (criados nesta entrega)**:
- `internal/runtime/specs/gemini.go`
- `internal/runtime/specs/gemini_test.go`
- `internal/runtime/events/gemini_metrics.go`
- `internal/runtime/events/gemini_metrics_test.go`
- `tests/integration/gemini_acp_smoke_test.go`
- `tests/integration/gemini_mcp_nested_test.go`
- `tests/integration/gemini_hooks_test.go`
- `tests/integration/gemini_autoreview_test.go`
- (opcional) `internal/runtime/memory/store_gemini_defaults_test.go`

**Modificados (edição cirúrgica)**:
- `cmd/ai_spec_harness/task_loop.go` — entry no catalog + switch defaults Gemini + telemetria
- `cmd/ai_spec_harness/task_loop_test.go` — T-13/T-14/T-15/T-16 estendidos + T-29..T-35
- `internal/taskloop/taskloop.go` — case "gemini" em roteamento ACP
- `internal/runtime/probe/probe.go` — adrByID entry gemini
- `internal/runtime/runner.go` — campos `Summary.Gemini*` + extração via convert.go
- `internal/runtime/events/convert.go` — chamada a `ExtractGeminiMetrics` quando driver=gemini
- `internal/runtime/events/normalize.go` — resolveInherit (se necessário; provável já implementado em F2-Claude)
- `internal/runtime/events/normalization-rules.yaml` — entrada `gemini: { inherit: common }`
- `.agents/normalization-rules.yaml` — mesma entrada (canônico)
- `internal/runtime/acp_integration_test.go` — sub-suite Gemini
- `internal/wrapper/wrapper.go` — warning sync.Once em buildInstruction("gemini", ...)
- `internal/evidence/evidence.go` — aceita seção opcional "Métricas Gemini-2026"
- `internal/telemetry/telemetry.go` — entries `gemini.*`
- `GEMINI.md` — reescrita completa com Runtime Capabilities F0+/F2+/F3+/F4+/F5+
- `AGENTS.md` — linha tabela de ADRs
- `docs/cli-schema.json` — `gemini` em enum de `--tool` quando `--runtime=acp`
- `CHANGELOG.md` — entradas F0..F5-Gemini
- `docs/telemetry-feedback-cycle.md` — invariantes Gemini ACP

**Inalterados (diff zero — RF-32)**:
- `internal/runtime/specs/spec.go`
- `internal/runtime/specs/claude.go`, `codex.go`, `copilot.go`
- `internal/runtime/client/client.go`
- `internal/runtime/persistence/` (toda a árvore)
- `internal/runtime/watchdog.go`
- `internal/runtime/mcpserver/` (toda a árvore — apenas testes Gemini novos)
- `internal/runtime/hooks/dispatcher.go`, `internal/runtime/hooks/governance.go`, demais hooks
- `internal/runtime/memory/store.go`
- `internal/runtime/runner_autoreview.go`
- `internal/specdrift/specdrift.go`
- `internal/agents/registry.go`

## Resolução de Suposições e Questões em Aberto do PRD

| # | Item PRD | Resolução nesta TechSpec |
|---|---|---|
| **A1** | `@google/gemini-cli@0.43.0` expõe ACP com semântica idêntica a Claude/Codex/Copilot | Confirmado parcialmente (probe `--acp --help` em 2026-05-22). Smoke real (RF-12) consolida. Risco baixo: SDK é compartilhado (`coder/acp-go-sdk v0.6.3`). |
| **A2** | Pacote `@google/gemini-cli@0.43.0` disponível no registry | ✅ Confirmado via `npm view`. |
| **A3** | Auth do `gemini-cli` é pré-condição operacional | Documentado em `GEMINI.md`. Probe não valida auth. |
| **A4** | `ActivityWatchdog` funciona para Gemini com timeout default | Validação em RF-09 (sub-suite Gemini em `acp_integration_test.go`). |
| **A5** | Mapeamento D-05 é estável na CLI 0.43.0 | Probe confirmou. Testes T-29/T-30/T-31 fixam mapeamento. |
| **A6** | Wrapper legado e ACP coexistem sem conflito | ✅ Coexistem hoje (apenas roteamento muda). |
| **A7** | `--reasoning-effort` ignorada por Gemini | ✅ T-30 valida. Documentado em help text. |
| **A8** | Nomes dos campos métrica Gemini-2026 são placeholders | **Resolvido em TD-02**: schema defensivo com `omitempty` absorve incerteza. Smoke real (RF-12) revela nomes exatos; ajuste vai via PR de hot-fix ou audit. |
| **A9** | Defaults Gemini-generosos não causam regressão Claude/Codex/Copilot | ✅ Switch tool-aware preserva defaults atuais quando `tool != "gemini"`. T-34 valida ambos os caminhos. |
| **A10** | Cascata F2-F5 é genuinamente tool-agnóstica | Testes T-32/T-33/T-39 validam. Dependência inter-PRD declarada em TD-06. |
| **Q1** | Modelo default Gemini | ✅ `DefaultGeminiModel = "gemini-2.5-pro"`. Propagação via `--executor-model` quando não especificado (lógica existente). |
| **Q2** | Validação semver da CLI Gemini | **Adiado** (não bloqueante). F0-Gemini assume disponibilidade quando `LookPath` resolve. Documentar em `GEMINI.md`. |
| **Q3** | Tabela de compatibilidade Gemini | ✅ Já contém modelos Gemini (`compatibility.go:34-43`); manter como está. Validação em RF-28. |
| **Q4** | Tempo de manutenção do wrapper legado | ✅ 2 versões minor (alinhado a F1-Copilot/F1-Codex). |
| **Q5** | Mensagem de warning `--access-mode=full` | ✅ Texto definido em §"Mensagens de Erro e Warning Literais": `"WARNING: --access-mode=full ativa --approval-mode=yolo no gemini-cli. Pré-condição: consentimento operacional. Ver GEMINI.md."` |
| **Q6** | `npm_version` em `runtime_init` mesmo quando launcher=binary | ✅ Sim, consistente com Claude/Codex/Copilot (sempre carrega `GeminiNpmVersion` para rastreabilidade). |
| **Q7** | Decomposição em waves | ✅ TD-06: wave-ordered estrito até F1; F2/F3/F4/F5 paralelos após F1. Documentado em §"Ordem de Build". |
| **Q8** | Campos exatos do payload ACP métricas Gemini-2026 | ✅ TD-02 schema defensivo absorve incerteza. Nomes assumidos: `cache_read_tokens`, `effective_context_tokens`, `prompt_tokens_billed`, `thoughts_tokens` — confirmar via smoke. |
| **Q9** | Dependência inter-PRD (F2-Gemini bloqueado se F2-Claude não mergeado) | ✅ TD-06 + §"Dependências Técnicas" declaram dependências explícitas. `tasks.md` (próximo artefato via `create-tasks`) deve registrar como blockers. |
| **Q10** | Mensagem custo amplificado auto-review janela 1M+ | ✅ Definida em §"Mensagens de Erro e Warning Literais": `"INFO: --auto-review com Gemini pode amplificar custo de tokens em janelas 1M+. Ver GEMINI.md §F5."` Emitida uma vez por session via flag interno. |

## Apêndice — Validação do Spec-Hash

Hash do PRD consumido nesta techspec: `f23aac99a055ce0b697646439bd80fd37b79239852c22e4c90a7cddade4dcbff`.

Validação:
```bash
ai-spec hash .specs/prd-gemini-cli-acp-2026/prd.md
# Output esperado: f23aac99a055ce0b697646439bd80fd37b79239852c22e4c90a7cddade4dcbff
```

Quando o PRD for editado (incremento de `spec-version`), `ai-spec hash` retornará valor diferente. Esta techspec ficará em drift até que:
1. `ai-spec sync-spec-hash .specs/prd-gemini-cli-acp-2026/techspec.md` atualize o cabeçalho, OU
2. A techspec seja regenerada via `create-technical-specification` (procedimento canônico).

`create-tasks` e `execute-task` validam o `spec-hash-prd` em Stage 1 e bloqueiam com `blocked` se houver drift detectado.
