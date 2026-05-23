# Documento de Requisitos do Produto (PRD) — Gemini CLI via ACP Nativo

<!-- spec-version: 1 -->

> **Insumo de pesquisa**: [docs/research/compozy-adaptation-gemini-2026.md](../../docs/research/compozy-adaptation-gemini-2026.md)
> **ADR material**: [.specs/adr/015-gemini-cli-acp-native.md](../adr/015-gemini-cli-acp-native.md)
> **Precedentes diretos**: F1-Codex ([.specs/prd-codex-acp-spec/](../prd-codex-acp-spec/)) — `BootstrapArgsFunc` reutilizável; F1-Copilot ([.specs/prd-copilot-acp-spec/](../prd-copilot-acp-spec/)) — CLI principal com flag `--acp`
> **Pesquisa irmã**: F2-F5-Claude ([.specs/prd-claude-cli-acp-2026/](../prd-claude-cli-acp-2026/)) — provê infra tool-agnostica que Gemini cascateia
> **Fase do roadmap**: F0..F5-Gemini em pacote unico (adaptacao Compozy 2026 — variante Gemini)
> **Data**: 2026-05-22

## Visão Geral

O `ai-spec-harness` integra Claude (ADR-009, F1-Claude), Copilot (ADR-012, F1-Copilot) e Codex (ADR-013, F1-Codex) via protocolo ACP nativo com persistência forense (`events.jsonl`, `tool_calls.md`, `execution_report.md`), watchdog de inatividade e telemetria opt-in (ADR-006). **Gemini permanece como único runtime ACP-capable em modo wrapper legado**: invocação atual em `internal/wrapper/wrapper.go:91-95` retorna comando opaco `gemini run --skill ... --project ...` consumido pela skill `execute-task` como bloco único — sem `events.jsonl` estruturado, sem `ActivityWatchdog`, sem telemetria, sem cascata para infra F2-F5 (MCP nested-agent, normalização de tool-calls, hooks Go in-process, memory 2-tier, métricas Gemini-2026, auto-review).

Em 2026 o Google enviou suporte ACP nativo no `@google/gemini-cli` via flag `--acp` (probe via `npx --yes @google/gemini-cli@0.43.0 --acp --help` em 2026-05-22 confirma a flag estável; `--experimental-acp` mantida como alias deprecated). O `compozy/compozy` (`main` SHA `7f38c445069bd83a8e96bcd925ee1f12fde74435`) registra Gemini em `internal/core/agent/registry_specs.go::model.IDEGemini` com `Command:"gemini"`, `FixedArgs:["--acp"]`, fallback `npx --yes @google/gemini-cli --acp`, `DefaultModel:"gemini-2.5-pro"` e `BootstrapArgs:nil`.

Esta funcionalidade introduz **Gemini CLI como runtime ACP nativo** no harness em pacote único cobrindo seis fases incrementais (F0-Gemini até F5-Gemini), atingindo paridade observacional total com Claude/Codex/Copilot e divergindo intencionalmente do Compozy em D-05 (ADR-015) ao traduzir `AccessMode` em flag `--approval-mode` exposta pela CLI Gemini 0.43.0. Beneficia operadores que usam Gemini e hoje perdem telemetria, persistência forense e watchdog. Também desbloqueia o aproveitamento da janela 1M+ tokens do `gemini-2.5-pro` dentro da governança `spec-hash`/`PRD-first` do harness (defaults Gemini-generosos para memory 2-tier em F3-Gemini) e a captura de métricas Gemini-2026 distintas (`cache_read_tokens`, `effective_context_tokens`, `prompt_tokens_billed`, `thoughts_tokens` em F4-Gemini).

Não introduz nova camada estrutural — reaproveita 100% do stack ACP construído para Claude (F1-Claude), generalizado por Copilot (F1-Copilot) e estendido por Codex (F1-Codex com `BootstrapArgsFunc`). A divergência F2-F5 em relação às fases Claude é mínima: tabela de aliases (F2), defaults de memory (F3), métricas (F4) — total ~110 LoC delta sobre infra compartilhada.

## Objetivos

- **OB-01**: Permitir executar tasks com Gemini via protocolo ACP nativo usando `--tool gemini --runtime acp`.
- **OB-02**: Atingir paridade observacional Gemini ↔ Claude ↔ Codex ↔ Copilot: `events.jsonl`, `tool_calls.md`, `execution_report.md` e telemetria opt-in (`GOVERNANCE_TELEMETRY=1`) cobrem invocações Gemini com mesmos campos e granularidade.
- **OB-03**: Traduzir `AccessMode` em flag `--approval-mode` exposta pela CLI Gemini (D-05 ADR-015): `Restricted → "default"`, `Full → "yolo"`. Validado por testes T-29/T-30/T-31.
- **OB-04**: Cascatar toda a infra F2-F5 tool-agnóstica (MCP nested-agent, hooks dispatcher, memory store, auto-review) automaticamente para Gemini após F1, com adaptações Gemini-específicas mínimas: entrada YAML de normalização, defaults de memory generosos, métricas Gemini-2026 distintas.
- **OB-05**: Capturar métricas Gemini-2026 distintivas em `execution_report.md`: `cache_read_tokens` (Gemini context caching), `effective_context_tokens` (uso real da janela 1M+), `prompt_tokens_billed` (cobrança pós-cache), `thoughts_tokens` (reasoning Gemini 2.5).
- **OB-06**: Preservar 100% retrocompatibilidade. Invocações sem `--runtime=acp` continuam usando wrapper legado em `internal/wrapper/wrapper.go:91-95`; aviso de depreciação informa migração. Remoção planejada para release N+2 (mesma política de F1-Copilot Q5 e F1-Codex).
- **OB-07**: Preservar invariantes forenses, watchdog (`ActivityWatchdog` com `CancelCause`), pinning de SDK (ADR-009), tagged union de eventos (ADR-010), interface `Spec` (ADR-013) sem regressão para Claude/Codex/Copilot.
- **OB-08**: Documentar mapeamento `AccessMode → --approval-mode` (D-05), divergência intencional do Compozy e aproveitamento da janela 1M+ em `GEMINI.md`.

**Métricas mensuráveis**:

- 100% dos testes de regressão Claude, Codex **e Copilot** permanecem verdes após F0-Gemini (novo entry em `runtimeACPCatalog`).
- Matriz de teste ACP (`internal/runtime/acp_integration_test.go`) cobre Gemini com ≥ 90% dos casos cobertos para Codex.
- Probe Gemini resolve em < 200ms p95 (binary path) e < 2s p95 (fallback npx) em ambiente padrão.
- Diff zero em `internal/runtime/persistence/`, `internal/runtime/watchdog.go`, `internal/runtime/client/client.go`, `internal/runtime/specs/spec.go` (não há nova extensão estrutural — apenas mais um spec consumindo infra existente).
- `geminiBootstrapArgs` produz output esperado em 100% dos casos `AccessModeRestricted`/`AccessModeFull`/zero-value (validado por T-29/T-30/T-31).
- `GeminiNpmVersion = "0.43.0"` confirmado disponível no npm registry e `--acp` validado em smoke test 2026-05-22.
- Tempo total de implementação: ≤ 3.5 sprints (15 dias úteis).
- LoC novo total: ≤ 700 (Spec ~50, Métricas Gemini ~80, defaults Gemini-generosos ~30, testes ~400, atualizações YAML/CHANGELOG/CLAUDE-GEMINI.md ~140).
- Após F1-Gemini: invocação `--tool gemini --runtime acp .specs/prd-X` em task simples (edit de 1 arquivo) completa e produz `events.jsonl` com ≥ 10 eventos estruturados.
- Após F3-Gemini: memory workflow Gemini suporta 250 linhas / 20 KiB sem `NeedsCompaction=true`; mesma sessão Claude (defaults 150) sinaliza compaction.

## Histórias de Usuário

- **HU-01**: Como **operador da CLI**, quero invocar `ai-spec-harness task-loop --tool gemini --runtime acp .specs/prd-minha-feature` e ver o harness conectar via ACP, produzir `events.jsonl` linha-a-linha e gerar `execution_report.md` no padrão Claude/Codex/Copilot.
- **HU-02**: Como **operador**, quero passar `--access-mode {restricted,full}` para escolher entre modo prompt-for-approval (`--approval-mode=default`) e modo yolo (`--approval-mode=yolo`). `full` aciona warning único em stderr antes de propagar para o Gemini.
- **HU-03**: Como **operador**, quero que quando o binário `gemini` não está no PATH, o harness tente `npx --yes @google/gemini-cli@0.43.0 --acp` como fallback automático sem mudança de comando.
- **HU-04**: Como **operador**, quero ver erro claro quando nem `gemini` nem `npx` estão disponíveis, com os três remédios (instalar via `npm install -g @google/gemini-cli@0.43.0`; instalar pacote npm direto; usar wrapper legado sem `--runtime=acp`) e referência a ADR-015.
- **HU-05**: Como **operador legado**, quero que `ai-spec-harness task-loop --tool gemini .specs/...` (sem `--runtime=acp`) continue funcionando via wrapper (`internal/wrapper/wrapper.go:91-95`), com aviso de depreciação único por execução do processo.
- **HU-06**: Como **mantenedor**, quero que telemetria opt-in (`GOVERNANCE_TELEMETRY=1`) registre invocações Gemini ACP com `tool=gemini`, `launcher=binary|npx`, `npm_version=0.43.0`, `sdk_version=v0.13.0` reais.
- **HU-07**: Como **mantenedor**, quero que `--reasoning-effort high --tool gemini --runtime acp` seja aceito mas **sem efeito** sobre o spawn de Gemini (que ignora `reasoning` em `geminiBootstrapArgs`) — validado por teste T-30. Mantém simetria com Claude (`BootstrapArgs` nil) e Codex (consome).
- **HU-08**: Como **mantenedor**, quero que o ADR-015 seja referenciado no índice em `AGENTS.md` e em `GEMINI.md`, e que a tabela `adrByID` em `internal/runtime/probe/probe.go` aponte `"gemini"` para ADR-015.
- **HU-09**: Como **operador**, quero que `ActivityWatchdog` cancele sessões Gemini inativas com o mesmo timeout configurável que cancela sessões Claude/Codex/Copilot.
- **HU-10**: Como **operador**, quero invocar Gemini com `--mcp-nested` (F2-Gemini) e ver Gemini parent invocar `run_agent("reviewer", "<prompt>")` resolvendo via `internal/agents/registry.go` para spawn de child session — sem código novo, cascata automática.
- **HU-11**: Como **operador**, quero que sessões Gemini aceitem memory workflow de até 250 linhas (vs 150 default Claude) e memory task de até 400 linhas (vs 200 default) sem `NeedsCompaction=true` — aproveitando a janela 1M+ do `gemini-2.5-pro`.
- **HU-12**: Como **mantenedor**, quero ver `execution_report.md` de sessões Gemini incluir seção "Métricas Gemini-2026" com `cache_read_tokens`, `effective_context_tokens`, `prompt_tokens_billed`, `thoughts_tokens` quando o payload ACP os contiver; ausência é silenciosa (não bloqueia evidence).
- **HU-13**: Como **operador**, quero passar `--auto-review --tool gemini --runtime acp` (F5-Gemini, opt-in) e ver session de review spawn após session principal, com aviso explícito sobre custo amplificado em janelas 1M+.

## Funcionalidades Core

### F-01 (F0-Gemini): Spec Gemini ACP em catálogo

Novo construtor `specs.Gemini()` em `internal/runtime/specs/gemini.go` retornando uma `Spec` com `ID="gemini"`, `DisplayName="Gemini (ACP)"`, `Command="gemini"`, `FixedArgs=["--acp"]`, `Fallbacks=[npx --yes @google/gemini-cli@<pin> --acp]`, `AccessModeFlag=""`, e `BootstrapArgs: geminiBootstrapArgs` (função local). Constantes `GeminiNpmPackage="@google/gemini-cli"`, `GeminiNpmVersion="0.43.0"`, `GeminiSDKVersion="v0.13.0"` (mesma de Claude/Codex/Copilot), `DefaultGeminiModel="gemini-2.5-pro"`.

### F-02 (F0-Gemini): Função `geminiBootstrapArgs`

Função local em `internal/runtime/specs/gemini.go` que mapeia `AccessMode` para flag `--approval-mode`. Ignora `model`, `reasoning`, `addDirs` (Gemini propaga modelo via `--model` separado; reasoning não é controlável; sem flag equivalente a `--add-dir`). Mapeamento literal D-05:

```go
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

### F-03 (F0-Gemini): Registro Gemini em `runtimeACPCatalog`

`cmd/ai_spec_harness/task_loop.go:27-31` ganha `"gemini": specs.Gemini` no catálogo. Gate em `task_loop.go` passa a aceitar `--tool gemini --runtime acp` sem mudança de código (mensagem de erro lista catálogo dinamicamente).

### F-04 (F0-Gemini): Tabela `adrByID` em probe

`internal/runtime/probe/probe.go` ganha `"gemini": ".specs/adr/015-gemini-cli-acp-native.md"` em `adrByID`. Mensagem de erro de probe Gemini referencia ADR-015 explicitamente.

### F-05 (F1-Gemini): Wiring de Gemini no taskloop service

`internal/taskloop/taskloop.go::Service.Execute` resolve Spec via `runtimeACPCatalog` quando `Runtime == "acp"` e `Tool == "gemini"`. Reusa `ACPRunner` existente. Propaga `Options.AccessMode` (e demais campos opcionais) para `Job`.

### F-06 (F1-Gemini): Matriz de teste ACP estendida

`internal/runtime/acp_integration_test.go` ganha sub-suite Gemini reusando o fake ACP server existente. Casos cobertos: open OK, prompt, ≥ 2 tipos de tool call, agent message, completion, cancel por `ActivityWatchdog`, erro de launcher unavailable, fallback npx, validação de que `geminiBootstrapArgs` produz `--approval-mode <value>` no spawn correto.

### F-07 (F1-Gemini): Aviso de depreciação no wrapper legado

`internal/wrapper/wrapper.go::buildInstruction("gemini", ...)` emite log WARNING uma única vez por execução do processo (via `sync.Once`): "Gemini wrapper legado (`gemini run --skill`) em uso. Migrar para --runtime=acp (binário `gemini` com `--acp`). Ver ADR-015." Mesma política de F1-Copilot Q5 e F1-Codex.

### F-08 (F2-Gemini): Entrada YAML de normalização

`.agents/normalization-rules.yaml` (e arquivo embedded `internal/runtime/events/normalization-rules.yaml`) ganha:

```yaml
gemini:
  inherit: common
  overrides: {}
```

`internal/runtime/events/normalize.go::BuildNormalizedToolCall("gemini", ...)` resolve `inherit: common` e aplica tabela comum. Compozy confirma suficiência em `tool_call_name.go:84` (Gemini herda `commonToolTitleAliases` sem override Gemini-específico).

### F-09 (F2-Gemini): Validação MCP nested-agent com Gemini

Zero código novo no `internal/runtime/mcpserver/`. Validação via teste de integração: Gemini parent invoca `run_agent("reviewer", "<prompt>")` e child session é spawnada via mesmo `engine.go`. Depth limit aplicado (≤ 3).

### F-10 (F3-Gemini): Defaults Gemini-generosos para memory 2-tier

`cmd/ai_spec_harness/task_loop.go` ganha switch tool-aware para defaults de memory quando `--tool gemini` e flag não foi setada explicitamente:

```go
case "gemini":
    if !cmd.Flags().Changed("memory-workflow-limit-lines") {
        memoryWorkflowLines = 250  // vs 150 default Claude/Codex/Copilot
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
```

`internal/runtime/memory/store.go` permanece tool-agnóstico — apenas defaults variam.

### F-11 (F3-Gemini): Validação hooks dispatcher com Gemini

Zero código novo no `internal/runtime/hooks/dispatcher.go`. Validação via teste: hook `runtime.pre_open` despachado antes de `c.Open` mesmo quando driver é Gemini.

### F-12 (F4-Gemini): Extração de métricas Gemini-2026

Novo `internal/runtime/events/gemini_metrics.go` (~80 LoC) com tipo `GeminiMetrics{CacheReadTokens, EffectiveContextTokens, PromptTokensBilled, ThoughtsTokens int}` e função `ExtractGeminiMetrics(raw json.RawMessage) (GeminiMetrics, error)`. Extração defensiva: ausência de campos no payload retorna zero-value silencioso, não bloqueia evidence.

### F-13 (F4-Gemini): Enriquecimento de Summary e execution_report.md

`internal/runtime/runner.go::Summary` ganha campos `GeminiCacheReadTokens`, `GeminiEffectiveContextTokens`, `GeminiPromptTokensBilled`, `GeminiThoughtsTokens` (todos opcionais). `internal/evidence/evidence.go` valida seção opcional "Métricas Gemini-2026" com tabela equivalente à Claude-2026. Telemetria opt-in registra entries `gemini.cache_read=N`, `gemini.thoughts=N`, `gemini.effective_context=N`.

### F-14 (F5-Gemini): Auto-review opt-in para Gemini

Zero código novo no `internal/runtime/runner_autoreview.go` (cascata F5-Claude tool-agnóstica). Validação via teste: `--auto-review --tool gemini --runtime acp` spawna sessão de review após session principal. `GEMINI.md` documenta custo amplificado em janelas 1M+.

### F-15: Reescrita de `GEMINI.md`

`GEMINI.md` raiz (esqueleto atual de 28 linhas) é reescrito (~80-100 linhas): seção "Runtime Capabilities (F0-Gemini+)" como primária; seção "Modo Legado Wrapper" deprecada; documentação do mapeamento D-05 `AccessMode → --approval-mode`; warning sobre `--access-mode=full`; seções "Runtime Capabilities (F2+/F3+/F4+/F5+)" listando MCP, normalização, hooks, memory defaults generosos, métricas Gemini-2026, auto-review. Mantém seção "Hooks de Governanca" atual.

### F-16: Telemetria enriquecida

Evento `runtime_init` no telemetria opt-in ganha tool real (`tool=gemini` quando aplicável), `npm_version=0.43.0`, `sdk_version=v0.13.0`. Sem novo kind de evento (preserva ADR-010).

## Requisitos Funcionais

- **RF-01**: Criar `specs.Gemini()` em `internal/runtime/specs/gemini.go` retornando `Spec` com `ID="gemini"`, `DisplayName="Gemini (ACP)"`, `Command="gemini"`, `FixedArgs=["--acp"]`, `AccessModeFlag=""`, `BootstrapArgs=geminiBootstrapArgs`.
- **RF-02**: Spec Gemini expõe pelo menos um `FallbackLauncher` com `Command="npx"`, `FixedArgs=["--yes", GeminiNpmPackage+"@"+GeminiNpmVersion, "--acp"]`.
- **RF-03**: Constantes `GeminiNpmPackage="@google/gemini-cli"`, `GeminiNpmVersion="0.43.0"`, `GeminiSDKVersion="v0.13.0"`, `DefaultGeminiModel="gemini-2.5-pro"` declaradas em `internal/runtime/specs/gemini.go` com política de atualização equivalente a Claude/Codex/Copilot (ADR-009/013).
- **RF-04**: Função `geminiBootstrapArgs(_, _ string, _ []string, mode AccessMode) []string` em `internal/runtime/specs/gemini.go` implementa mapeamento D-05: `AccessModeFull → ["--approval-mode", "yolo"]`; `AccessModeRestricted`, zero-value e default → `["--approval-mode", "default"]`. Não emite valores `auto_edit` nem `plan`.
- **RF-05**: `cmd/ai_spec_harness/task_loop.go:27-31` ganha entrada `"gemini": specs.Gemini` em `runtimeACPCatalog`.
- **RF-06**: `internal/runtime/probe/probe.go` ganha `"gemini": ".specs/adr/015-gemini-cli-acp-native.md"` em `adrByID`.
- **RF-07**: `internal/taskloop/taskloop.go` roteia Gemini via `ACPRunner` quando `Runtime == "acp"`. Propaga `Options.AccessMode` para `Job.AccessMode`.
- **RF-08**: `internal/wrapper/wrapper.go::buildInstruction("gemini", ...)` emite log WARNING único por execução do processo (via `sync.Once`) referenciando ADR-015 quando wrapper legado é invocado.
- **RF-09**: `internal/runtime/acp_integration_test.go` ganha sub-suite Gemini reusando fake ACP server. Cobertura mínima: open, prompt, ≥ 2 tipos de tool call, agent message, completion, cancel por `ActivityWatchdog`, validação de spawn args (`--approval-mode default` por default; `--approval-mode yolo` em `--access-mode=full`).
- **RF-10**: `internal/runtime/specs/gemini_test.go` cobre defaults da Spec, fallback npx, constantes pinadas, e função `geminiBootstrapArgs` com matriz: AccessMode `restricted`/`full`/zero-value; combinações com model/reasoning/addDirs setados para validar que são ignorados (T-30).
- **RF-11**: Testes T-13/T-14/T-15/T-16 (em `task_loop_test.go`) estendidos para incluir Gemini: `TestRuntimeACPCatalogIncludesGemini`, `TestGeminiSpecHasCorrectCommandAndFlags`, `TestGeminiFallbackResolvesViaNpx`, `TestGeminiBootstrapArgsForRestricted`. Novos T-29 (`TestGeminiBootstrapArgsForFull`), T-30 (`TestGeminiBootstrapArgsIgnoresModelAndReasoning`), T-31 (`TestGeminiBootstrapArgsDefaultsToRestricted`).
- **RF-12**: Smoke test em `tests/integration/gemini_acp_smoke_test.go` (skipable via `-short`) invoca `gemini --acp` real e verifica que `events.jsonl` é produzido para task simples.
- **RF-13**: `.agents/normalization-rules.yaml` e `internal/runtime/events/normalization-rules.yaml` (embedded) recebem entrada `gemini: { inherit: common, overrides: {} }`. `internal/runtime/events/normalize.go::BuildNormalizedToolCall("gemini", ...)` resolve `inherit: common`.
- **RF-14**: Teste T-32 novo: `TestNormalizeToolCallGeminiInheritsCommon` — Gemini emite `read_file` → normalizado `Read`; `raw_name` preservado.
- **RF-15**: Teste T-33 novo em `internal/runtime/mcpserver/`: `TestMCPNestedAgentSpawnsGeminiSession` — Gemini parent invoca `run_agent` e child session é spawnada. Depth limit aplicado.
- **RF-16**: `cmd/ai_spec_harness/task_loop.go` aplica defaults Gemini-generosos para memory quando `--tool gemini` e flags `--memory-*` não foram setadas explicitamente: workflow 250 linhas / 20 KiB; task 400 linhas / 32 KiB. Override via flag explícita preservado.
- **RF-17**: Testes T-34 (`TestGeminiDefaultsMemoryLimitsAreGenerous`) e T-35 (`TestGeminiMemoryLimitOverrideByCliFlag`) validam comportamento de defaults e override.
- **RF-18**: Novo `internal/runtime/events/gemini_metrics.go` exporta `type GeminiMetrics struct{CacheReadTokens, EffectiveContextTokens, PromptTokensBilled, ThoughtsTokens int}` e função `ExtractGeminiMetrics(raw json.RawMessage) (GeminiMetrics, error)`. Extração defensiva: ausência de campos retorna zero-value sem erro.
- **RF-19**: `internal/runtime/runner.go::Summary` ganha campos opcionais `GeminiCacheReadTokens`, `GeminiEffectiveContextTokens`, `GeminiPromptTokensBilled`, `GeminiThoughtsTokens int`. Defaults zero.
- **RF-20**: `internal/evidence/evidence.go` aceita seção opcional "Métricas Gemini-2026" no `execution_report.md` com tabela markdown contendo `cache_read_tokens`, `effective_context_tokens`, `prompt_tokens_billed`, `thoughts_tokens`. Ausência não bloqueia validação.
- **RF-21**: Testes T-36 (`TestExtractGeminiMetricsFromACPPayload`), T-37 (`TestEvidenceRendersGeminiMetricsSection`), T-38 (`TestEvidenceMissingGeminiMetricsDoesNotBlock`) validam extração e renderização.
- **RF-22**: Teste T-39 novo: `TestAutoReviewWithGeminiDriver` — `--auto-review --tool gemini --runtime acp` spawna sessão de review após session principal; resultado persistido em `evidence/<task>/review.md`.
- **RF-23**: Reescrever `GEMINI.md` raiz com seção "Runtime Capabilities (F0-Gemini+)" como primária; seção "Modo Legado Wrapper" deprecada; mapeamento D-05 documentado; warning sobre `--access-mode=full`; seções F2+/F3+/F4+/F5+ listando capabilities cascateadas.
- **RF-24**: Atualizar `AGENTS.md` adicionando linha na tabela de ADRs (ADR-015).
- **RF-25**: Atualizar `docs/cli-schema.json` adicionando `gemini` em enum de `--tool` quando `--runtime=acp` (atualmente lista apenas claude/codex/copilot).
- **RF-26**: Atualizar `CHANGELOG.md` raiz com entrada `feat(gemini): F0+F1 ACP nativo via gemini --acp (ADR-015)` e entradas adicionais para F2-F5 conforme as fases forem entregues.
- **RF-27**: Atualizar `docs/telemetry-feedback-cycle.md` documentando que invariantes Gemini ACP cobrem os mesmos kinds que Claude/Codex/Copilot, com aditivos Gemini-2026.
- **RF-28**: Tabela `internal/taskloop/compatibility.go::CompatibilityTable` já contém modelos Gemini (`gemini-2.5-pro`, `pro`, `flash`, `flash-lite`, `gemini-3-pro-preview`); validar em F1-Gemini que `IsSupported("gemini", "gemini-2.5-pro")` retorna `true`. Sem mudança de código esperada.
- **RF-29**: Cache de probe em `internal/runtime/probe/probe.go` continua keyed por `spec.ID` — múltiplas invocações Gemini em uma sessão CLI reutilizam o launcher resolvido sem re-probe.
- **RF-30**: Regressão obrigatória: 100% dos testes de `internal/runtime/specs/{claude,codex,copilot}_test.go` permanecem verdes após F0-Gemini (apenas nova entrada em catálogo; interface `Spec` inalterada).
- **RF-31**: Telemetria opt-in registra `runtime_init` com `tool=gemini` quando aplicável; campo `launcher` distingue `binary` de `npx`.
- **RF-32**: Persistência forense (`events.jsonl`, `tool_calls.md`, `execution_report.md`) e `ActivityWatchdog` permanecem inalterados em comportamento e código fonte. Diff zero em `internal/runtime/persistence/`, `internal/runtime/watchdog.go`, `internal/runtime/client/client.go`, `internal/runtime/specs/spec.go`.
- **RF-33**: Warning de `--access-mode=full` aciona aviso único em stderr (via `accessModeFullWarnOnce` já existente em `cmd/ai_spec_harness/task_loop.go:19-21`) antes de propagar para Gemini. Mensagem: `"WARNING: --access-mode=full ativa --approval-mode=yolo no gemini-cli. Pré-condição: consentimento operacional. Ver GEMINI.md."`

## Experiência do Usuário

UX é primariamente backend/CLI; materializa-se em cinco pontos:

1. **CLI — comando recomendado**:
   ```
   ai-spec-harness task-loop \
     --tool gemini \
     --runtime acp \
     --access-mode restricted \
     .specs/prd-minha-feature
   ```
   Comportamento idêntico a Claude/Codex/Copilot: stream humano em stdout (suprimível via `--quiet`), `events.jsonl`/`tool_calls.md`/`execution_report.md` em `audit/` ou `evidenceDir` configurado. Após F4-Gemini, `execution_report.md` ganha seção "Métricas Gemini-2026" quando o payload ACP contiver os campos.

2. **CLI — comando full-access (com warning)**:
   ```
   ai-spec-harness task-loop \
     --tool gemini --runtime acp \
     --access-mode full \
     .specs/prd-minha-feature
   ```
   `--access-mode=full` aciona warning único em stderr: `"WARNING: --access-mode=full ativa --approval-mode=yolo no gemini-cli. Pré-condição: consentimento operacional. Ver GEMINI.md."` antes de spawn. Propaga `--approval-mode yolo` via `geminiBootstrapArgs`.

3. **CLI — erro de launcher unavailable**:
   ```
   gemini não encontrado. Install via `npm install -g @google/gemini-cli@0.43.0`; OR instale o pacote npm direto; OR use wrapper legado sem --runtime=acp. Veja .specs/adr/015-gemini-cli-acp-native.md
   ```

4. **CLI — aviso de depreciação wrapper legado**:
   ```
   WARNING: Gemini wrapper legado (gemini run --skill) em uso. Migrar para --runtime=acp (binário gemini com --acp). Ver ADR-015.
   ```
   Emitido uma única vez por execução do processo (não por task).

5. **CLI — auto-review opt-in com Gemini**:
   ```
   ai-spec-harness task-loop \
     --tool gemini --runtime acp \
     --auto-review \
     .specs/prd-minha-feature
   ```
   Após session end, spawna sessão de review com skill `review` + diff acumulado. `evidence/<task>/review.md` persistido. Warning informativo: `"INFO: --auto-review com Gemini pode amplificar custo de tokens em janelas 1M+. Ver GEMINI.md §F5."`

6. **`GEMINI.md` raiz** reescrito conforme exemplo em [`docs/research/compozy-adaptation-gemini-2026.md`](../../docs/research/compozy-adaptation-gemini-2026.md) §"Exemplos de Configuração 2026".

## Restrições Técnicas de Alto Nível

- **Linguagem e protocolo**: Go, mantendo `coder/acp-go-sdk` como SDK ACP (ADR-009). Versão sincronizada com `go.mod` via `scripts/sync-acp-sdk-version.sh`.
- **Reuso obrigatório**: `ACPRunner`, `acpClient` (`internal/runtime/client/client.go`), `SessionPersistence`, `ActivityWatchdog`, `events` package, `internal/runtime/mcpserver/`, `internal/runtime/hooks/`, `internal/runtime/memory/`, `internal/runtime/runner_autoreview.go` são reusados sem modificação. Apenas extensões aditivas em `task_loop.go` (registro + defaults Gemini), em `evidence/evidence.go` (seção opcional Métricas Gemini-2026) e em `events/normalize.go` (resolução de `inherit:common`).
- **Pinning de SDK e pacote npm**: `GeminiNpmVersion = "0.43.0"` constante Go pinada (não `@latest`); atualização via processo `audit/` (espelha `ClaudeNpmVersion` em `claude.go` e `CodexNpmVersion` em `codex.go`). `GeminiSDKVersion = "v0.13.0"` mantida em sincronia com `go.mod` (mesma do Claude/Codex/Copilot).
- **Filesystem abstraction**: leitura de configs e write de artefatos forenses continuam sobre `fs.FileSystem` (ADR-002).
- **Invariantes preservadas**:
  - Persistência forense intacta (`internal/runtime/persistence/`).
  - `ActivityWatchdog` intacto (`internal/runtime/watchdog.go`).
  - `acpClient` intacto em comportamento de subprocess management e fan-out de eventos.
  - Interface `Spec` (`internal/runtime/specs/spec.go`) **inalterada** — ADR-013 já estendeu com `BootstrapArgsFunc`; Gemini consome a infra existente.
  - Tabela de compatibilidade tool↔model (`internal/taskloop/compatibility.go`) continua autoritativa (já contém Gemini).
  - ADR-009 (pinning SDK), ADR-010 (tagged union), ADR-011 (Agent Registry), ADR-012 (Copilot ACP), ADR-013 (Codex ACP), ADR-014 (Claude 2026) inalterados.
- **Compatibilidade**: caminho legado (wrapper em `internal/wrapper/wrapper.go:91-95`) é mantido por 2 versões minor com aviso de depreciação. Remoção é decisão de versão futura, não desta fase. Mesma política de F1-Copilot (Q5 de ADR-012) e F1-Codex.
- **Segurança (R-SEC-001)**: subprocess `gemini` segue mesmas regras de Claude/Codex/Copilot — sem shell, args via `exec.Command` com slice. **`AccessModeFull` (→ `--approval-mode=yolo`) é opt-in explícito**: warning único em stderr antes de propagar. Ausência de consentimento operacional não é detectável programaticamente — confiamos no consentimento via flag.
- **Limites operacionais**: probe Gemini deve resolver em < 200ms p95 (binary) e < 2s p95 (npx fallback) em ambiente padrão.
- **Telemetria**: aditiva. Campos `tool`, `launcher`, `npm_version`, `sdk_version` em `runtime_init` ganham cardinalidade `tool=gemini` mas sem novo kind de evento (ADR-010 invariante preservada). Métricas Gemini-2026 em F4 são aditivas em `execution_report.md` — não modificam tagged union de eventos.
- **Documentação versão mínima**: techspec e `GEMINI.md` documentam `GeminiNpmVersion=0.43.0` (pin atual, dist-tag `latest` em 2026-05-22) e alias `--experimental-acp` como deprecated upstream.
- **Mensagem de erro distingue `gemini` (binário CLI) vs `gemini-cli` (nome do pacote npm)**: probe error em `internal/runtime/probe/probe.go` deve referenciar **`gemini`** (não `gemini-cli`) para evitar confusão.
- **Divergência intencional do Compozy** (D-03/D-05 ADR-015): Compozy mantém `BootstrapArgs: nil`; harness emite `--approval-mode`. Documentado em comentário do `geminiBootstrapArgs` e em `GEMINI.md`. Revisão obrigatória quando `audit/` atualizar `GeminiNpmVersion`.
- **Cascata F2-F5 reusa infra Claude**: F2-Gemini, F3-Gemini, F5-Gemini não introduzem código novo na infra tool-agnóstica — apenas adicionam testes de regressão, entradas YAML (F2), defaults (F3), métricas (F4) e docs (F2/F3/F4/F5).

## Fora de Escopo

Os itens abaixo **não** fazem parte deste pacote F0..F5-Gemini. Estão documentados como follow-ups em [`docs/research/compozy-adaptation-gemini-2026.md`](../../docs/research/compozy-adaptation-gemini-2026.md) §"Adendo" ou em ADR-015 §"Consequências":

- **F0.1-Gemini — mapeamento `auto_edit` e `plan`** em `--approval-mode` (semântica de auto-aprovar apenas edits, ou modo read-only). Reservado para futura extensão se demanda concreta surgir; quebra simetria binária com Claude/Codex.
- **Flag `--gemini-cache-ttl`** para controle de TTL do context caching Gemini. Especulação 2026-Q3; depende de exposição na CLI Gemini.
- **Subcomandos nativos `gemini hooks migrate` e `gemini skills install/link`**. Capabilities upstream complementares; documentar em `GEMINI.md` como **opção do usuário** mas não usar programaticamente. Migração `.claude/hooks/` → formato Gemini fica a critério do operador.
- **`gemini mcp`** (cliente MCP nativo da CLI Gemini 0.43.0). Investigação fica para F2-Gemini fase 2 quando a infra MCP do harness estiver estabilizada.
- **Reasoning effort exposto via CLI Gemini**. Gemini 2.5 expõe `thoughts_tokens` na resposta mas sem controle programático; flag `--reasoning-effort` aceita mas ignorada (T-30 valida).
- **Mudança em modo avançado** (`--executor-tool=gemini --reviewer-tool=gemini`) — esta fase cobre apenas modo simples (`--tool gemini`); modo avançado é incremento aditivo posterior se demandado.
- **Validação runtime da versão de `@google/gemini-cli`** (ex: `gemini --version >= 0.43.0`) — probe não valida versão; assume disponibilidade quando `LookPath` resolve. Documentar limitação em `GEMINI.md`.
- **Remoção do wrapper legado** (`internal/wrapper/wrapper.go::ValidTools["gemini"]`) — decisão de versão futura (mesma política de F1-Copilot/F1-Codex). Esta fase mantém wrapper com aviso de depreciação.
- **`SupportsAddDirs` para Gemini**. Compozy não seta (default `false`) e CLI Gemini 0.43.0 não expõe flag equivalente a Claude `--add-dir`. Se mudar upstream, ajuste via audit/ ou nova ADR.
- **Mudanças em CI workflow** (além de validar matriz Gemini quando `gemini` disponível em runner) — fora de escopo desta fase. Smoke real é manual.
- **Migração de `.gemini/commands/workspace.*.toml` para `gemini skills link`** — modo wrapper continua usando TOML adapters; migração para discovery nativo é decisão futura quando wrapper for removido.

## Suposições e Questões em Aberto

**Suposições assumidas** (devem ser validadas no TechSpec):

- **A1**: `@google/gemini-cli@0.43.0` expõe ACP com semântica idêntica a Claude/Codex/Copilot — protocolo JSON-RPC sobre stdio, mesmos kinds de eventos. **Validação parcial em 2026-05-22**: `npx --yes @google/gemini-cli@0.43.0 --acp --help` confirma flag estável; smoke real (RF-12) consolida validação.
- **A2**: O pacote npm `@google/gemini-cli@0.43.0` permanece disponível no registry. **Validação concluída em 2026-05-22**: `npm view @google/gemini-cli version dist-tags` retorna `latest = 0.43.0`.
- **A3**: Auth do `gemini-cli` (token Google ou OAuth) é pré-condição operacional, não responsabilidade do harness. Ausência produz erro do subprocess que o harness reporta.
- **A4**: `ActivityWatchdog` com timeout default funciona para Gemini — eventos `agent.on_session_update` chegam com cadência compatível.
- **A5**: Mapeamento D-05 (`Restricted → "default"`, `Full → "yolo"`) é estável na CLI Gemini 0.43.0. Valores `auto_edit` e `plan` permanecem disponíveis mas não usados.
- **A6**: Wrapper legado (`gemini run --skill`) pode coexistir com modo ACP no mesmo binário sem conflito de flag (já coexistem hoje; apenas o roteamento muda).
- **A7**: Flag `--reasoning-effort` passada com `--tool gemini` é aceita mas sem efeito (`geminiBootstrapArgs` ignora — T-30 valida). Documentar em help text.
- **A8**: Campos do payload ACP Gemini para métricas (`cache_read_tokens`, `effective_context_tokens`, `prompt_tokens_billed`, `thoughts_tokens`) podem ter nomes diferentes na realidade; extração defensiva (RF-18) absorve ausência. F4-Gemini precisa probe real para confirmar nomes; TechSpec define alvos.
- **A9**: Defaults Gemini-generosos (250/400 linhas) não causam regressão observável em Claude/Codex/Copilot (switch é tool-aware em `task_loop.go`; defaults Claude/Codex/Copilot permanecem 150/200).
- **A10**: Cascata F2-F5 (MCP, hooks, normalize, memory, auto-review) é genuinamente tool-agnóstica — F2-F5-Claude entregues sem switches por driver. Validação via testes T-32/T-33/T-39 confirma.

**Questões em aberto** (não bloqueantes para PRD; serão resolvidas no TechSpec):

- **Q1**: Modelo default do Gemini quando `--model` não passado. Proposta: usar `DefaultGeminiModel="gemini-2.5-pro"` (mesmo default do Compozy) propagado via `--executor-model` quando não especificado.
- **Q2**: Validação de versão mínima do `@google/gemini-cli` (semver `>= 0.39.0` quando `--acp` saiu de `experimental`) — implementar em F0-Gemini ou adiar? Proposta: adiar (não bloqueante); F0-Gemini assume disponibilidade quando `LookPath` resolve.
- **Q3**: Tabela de compatibilidade `internal/taskloop/compatibility.go` já contém modelos Gemini — manter como está ou refinar entradas? Proposta: manter como está; ajustes via PR separado se necessário.
- **Q4**: Quanto tempo manter wrapper legado antes de remover? Proposta: 2 versões minor (alinhado a Q5 de ADR-012 e F1-Codex).
- **Q5**: Mensagem do warning de `--access-mode=full`. Proposta: explícita o suficiente para evitar uso acidental ("ATENÇÃO: `--approval-mode=yolo` no gemini-cli auto-aprova TODAS as ferramentas, incluindo destrutivas. Use somente em ambientes isolados.")
- **Q6**: `runtime_init` carrega `npm_version=GeminiNpmVersion` mesmo quando launcher é `binary` (sem npx)? Proposta: sim, consistente com Claude/Codex/Copilot.
- **Q7**: Decompor o PRD em wave-ordered tasks (F0 → F1 → F2 → F3 → F4 → F5) ou paralelizar F2/F3/F4/F5 após F1? Proposta: wave-ordered estrito até F1; F2/F3/F4/F5 podem rodar em paralelo (DAG independente). Decisão final em `create-tasks`.
- **Q8**: Campos exatos do payload ACP para métricas Gemini-2026 (A8). Proposta: TechSpec ou primeira execução de F4-Gemini consolida; extração defensiva (`omitempty` em JSON unmarshal) absorve incerteza temporária.
- **Q9**: Se F2-Claude ainda não estiver mergeada quando F2-Gemini for executada, F2-Gemini é bloqueado? Proposta: sim, F2-Gemini depende de F2-Claude entregar `internal/runtime/events/normalize.go` e `internal/runtime/mcpserver/`. Documentar em `tasks.md` como dependência inter-PRD.
- **Q10**: Mensagem de warning sobre custo amplificado em auto-review com janela 1M+. Proposta: emitir uma vez por session, não por task. Texto baseado em RF-22 a definir no TechSpec.
