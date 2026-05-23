# PRD — Paridade Absoluta Cross-CLI e Instalação Universal Transparente

> **Status:** Draft (capítulo inicial). Origem: auditoria mandatória `docs/prompts/compozy-acp-gap-analysis.md`.
> **Data:** 2026-05-22
> **Fonte de verdade externa:** `compozy/compozy` via GitHub CLI (`gh`).
> **Fonte de verdade interna:** `internal/runtime`, `internal/invocation`, `internal/install`, `internal/specdrift`, `.specs/*`.

## 0. Objetivo

Garantir **comportamento idêntico mandatório** das mesmas skills/comandos do `ai-spec-harness`
nas 4 CLIs alvo — `claude-code-cli`, `codex-cli`, `copilot-cli`, `gemini-cli` — e tornar a
instalação (`ai-spec install`) **transparente** em qualquer codebase (Go/Node/Python; escalas P/M/G),
sem o usuário precisar entender a CLI de IA subjacente.

Este capítulo elimina gaps comprovados por evidência. Não introduz comportamento especulativo.

## 1. Achado central da auditoria

Compozy e ai-spec-harness são **peers arquiteturais quase idênticos**: mesmo problema, mesmas
primitivas, mesma stack base. Ambos usam o **mesmo SDK ACP** (`github.com/coder/acp-go-sdk`),
spawnam cada CLI como subprocesso ACP via registry de specs, normalizam tool-calls por tabela de
alias preservando `RawInput`, e implementam memória 2-tier markdown com limites idênticos
(workflow 150 linhas/12 KB, task 200 linhas/16 KB).

| Evidência | Compozy (via `gh`) | ai-spec-harness (local) |
|---|---|---|
| SDK ACP | `coder/acp-go-sdk v0.6.3` (`go.mod`) | `coder/acp-go-sdk v0.13.0` (`go.mod`) |
| Registry de specs | `internal/core/agent/registry_specs.go` (8 IDEs) | `internal/runtime/specs/*.go` (4 CLIs) |
| Normalização tool-call | `internal/core/agent/tool_call_name.go` (`commonToolTitleAliases`) | `internal/runtime/events/normalize.go` + `.agents/normalization-rules.yaml` |
| Preserva raw | `ToolUseBlock{Input, RawInput}` (`content.go`) | `BuildNormalizedToolCall` mantém `raw_name`/`normalized_name` |
| Nested agent | MCP `run_agent` (`mcpserver/server.go`) | MCP `run_agent` (`internal/runtime/mcpserver/`) |
| Memória 2-tier | `memory/store.go` 150/12KB · 200/16KB | `internal/runtime/memory/store.go` mesmos limites |
| Config hierárquico | TOML, global→workspace→defaults (`config_merge.go`) | YAML, flags>workspace>global>defaults (ADR-016, `internal/config/resolver.go`) |
| Retry/backoff | `executor/runner.go` (multiplier 1.5) | `internal/runtime/retry.go` + RuntimeConfig (ADR-018) |
| Install auto-detect | `compozy setup` + `DetectInstalledAgents()` (40+ alvos) | `ai-spec install` + `internal/detect` (4 CLIs) |

**Conclusão estratégica:** não há lacuna fundacional de transporte. As lacunas reais para *paridade
mandatória* + *transparência total* são de camada de aplicação e governança, e em vários pontos
**estamos à frente do Compozy** (ver §6). Onde Compozy lidera, é em amplitude de instalação.

### Onde Compozy NÃO ajuda (NO EVIDENCE FOUND via `gh`)

- **Suíte de invariantes de paridade cross-runtime** — Compozy não tem. Nós temos ADR-008 (29 invariantes, 3 níveis). **Vantagem nossa.**
- **Gestão de contexto por token-budget** — Compozy compacta por linha/byte, igual a nós. Nenhum dos dois tem estratégia por janela de token. **Gap compartilhado → oportunidade escala G.**
- **Detecção de stack (Go/Node/Python) no instalador** — Compozy é agent-centric, não detecta linguagem. **Gap compartilhado → oportunidade de transparência.**
- **Workaround stateless Copilot** — Compozy depende de `copilot --acp` nativo, igual ao nosso pós ADR-012.

## 2. Matriz de Igualdade Cross-CLI

Estado atual por capability (✓ paridade · ⚠ assimétrico · ✗ ausente). F0/F1 estão completos nas 4 CLIs.

| Capability | Claude | Codex | Copilot | Gemini | Lacuna mandatória |
|---|---|---|---|---|---|
| F0 Spec/launcher + fallback npx | ✓ | ✓ | ✓ | ✓ | — |
| F1 ACP + events.jsonl + watchdog + retry | ✓ | ✓ | ✓ | ✓ | — |
| Normalização de **nome** de tool-call | ✓ | ✓ | ✓ | ⚠ (só `inherit_common`) | Gemini sem tabela própria |
| Normalização de **campo** de input (`input_mappings`) | ✓ | ✓ | ✗ | ✗ | **Copilot e Gemini sem `input_mappings`** |
| MCP nested-agent (`run_agent`) | ✓ | ✓ (cascata) | ✓ (cascata) | ✓ (cascata) | validar não-regressão |
| Memória 2-tier + compactação | ✓ | ⚠ (defaults F1) | ⚠ (defaults F1) | ⚠ (defaults F1) | sem override por janela de CLI |
| Hooks dispatcher (governance/token) | ✓ | ✓ | ✓ | ✓ | `token_budget` não é CLI-aware |
| Métricas F4 (tokens cache/thinking) | planejado | ✗ | ✗ | planejado | **sem conjunto mínimo unificado** |
| Auto-review opt-in | planejado | cascata | cascata | cascata | sem warning de custo (janela G) |
| Concurrent/BatchSize | ✓ (`runloop.go`) | ✓ | ✓ | ✓ | sem tuning por CLI |

**Requisito de paridade (RP):**

- **RP-01** — `input_mappings` definidos e testados para as 4 CLIs (ou documentados como no-op verificado). Hoje Copilot/Gemini estão indefinidos em `internal/runtime/events/normalization-rules.yaml`.
- **RP-02** — Conjunto **mínimo** de métricas normalizado em `execution_report.md`: `total_tokens`, `cache_read_tokens`, `thinking_tokens` (campos ausentes ⇒ omitidos, nunca divergentes). Hoje só Claude/Gemini planejam campos próprios.
- **RP-03** — Teste de invariância: a mesma task em cada CLI produz o mesmo conjunto de `normalized_name` e a mesma forma de evento, validado por suíte derivada de ADR-008.
- **RP-04** — Tabela de alias Gemini explícita (paridade com Claude/Codex/Copilot), não apenas herança de `common_aliases`.

## 3. Blueprint de Instalação Universal Transparente

Estado atual: `ai-spec install .` auto-detecta agentes (sem `--tools`), é idempotente, bootstrap < 30s
(RF-11, ADR-019). Lacunas para transparência total (P/M/G):

- **RI-01 — Detecção de stack ativa.** `internal/detect/detect.go` já lê `go.mod`/`package.json`/`pyproject.toml`, mas o install não usa o resultado para selecionar a skill de linguagem correta automaticamente. Tornar o scaffold stack-aware (Go→`go-implementation`, Node→`node-implementation`, Python→`python-implementation`) sem flag. *Diferencial vs Compozy, que é agent-centric.*
- **RI-02 — Probe de binário ACP no install.** Hoje `install` valida config dirs, mas `probe.EnsureAvailable` só roda em execução; usuário pode "instalar com sucesso" e falhar depois. Install deve reportar, por CLI detectada, se o binário/fallback npx está disponível (warning não-fatal), eliminando a falha tardia.
- **RI-03 — Geração de stubs de config por CLI.** `install` gera `.agents/config.yaml` unificado mas não os stubs por CLI. Gerar `.claude/`, `.codex/config.toml`, `.gemini/commands/*`, `.github/copilot-instructions.md` conforme as CLIs detectadas, para que o setup seja funcional sem ajuste manual.
- **RI-04 — Validação pós-install (`verify`) cobre as 4 CLIs igualmente.** Garantir que `ai-spec verify` reporte current/missing/drifted por skill **e** por binário ACP, fechando o loop de transparência.

**Critério P/M/G:** install validado em repo vazio (P), repo Go/Node/Python existente (M) e monorepo
com múltiplos agentes detectados (G), convergindo a 100% `current` na reexecução.

## 4. Requisitos de Infraestrutura (`internal/`)

- **RIN-01 — Wiring do config hierárquico no runtime.** `internal/config/resolver.go` (ADR-016) existe, mas o `ACPRunner` não consome `RuntimeConfig` mesclado (flags>workspace>global>defaults). Hoje os defaults vêm hardcoded em `specs/` + campos de `Job` propagados pelo `taskloop`. Mesclar a precedência antes de `runner.Run()`, idêntico para as 4 CLIs.
- **RIN-02 — Camada de input-normalization completa.** Estender `internal/runtime/events/normalize.go` + `normalization-rules.yaml` para cobrir `input_mappings` de Copilot/Gemini (RP-01), com teste por CLI garantindo campos canônicos idênticos.
- **RIN-03 — Harmonização de métricas.** `internal/runtime/events/metrics.go` deve expor extração por driver com fallback ao conjunto mínimo comum (RP-02), renderizado uniformemente em `persistence/report.go`.
- **RIN-04 — Estratégia de janela grande (escala G).** Nem Compozy nem nós temos gestão por token. Introduzir, no `memory/store.go` + hook `token_budget`, decisão de compactação sensível à janela da CLI (ex.: usar limite generoso quando a CLI sinaliza janela ≥ 1M), mantendo zero-value = comportamento F1. *Oportunidade de liderança.*
- **RIN-05 — Sunset do legacy mode.** `copilotInvoker`/`codexInvoker` (`internal/taskloop/agent.go`) e `internal/wrapper/wrapper.go` coexistem com avisos de deprecação. Definir release de remoção e guia de migração para reduzir superfície de divergência entre runtimes.

## 5. Estratégia de Governança Unificada

Invariantes que devem permanecer idênticos e transparentes em qualquer runtime:

- **RG-01 — spec-hash em runtime.** **Confirmado por grep:** não há referência a `specdrift`/`spec-hash` em `internal/runtime/`. O `ACPRunner` não valida hash do PRD/techspec antes de executar. Adicionar checagem (no hook `runtime.pre_open`, ao lado da validação de `AGENTS.md`) que falha-rápido em divergência, idêntica para as 4 CLIs. Reusar `internal/specdrift/specdrift.go` (`CheckHash`/`CheckDrift`).
- **RG-02 — PRD-first enforce.** A doutrina PRD-first é hoje manual (comentários `<!-- spec-hash-prd -->` em `tasks.md`). O hook de governança deve recusar execução de task sem PRD rastreável, uniformemente entre runtimes.
- **RG-03 — Invariantes ADR-008 em execução.** As 29 invariantes existem como framework de teste; promovê-las a verificação de runtime (ou CI obrigatório por CLI) torna a paridade *provável*, não apenas documentada. **Esta é nossa vantagem sobre o Compozy — capitalizar.**
- **RG-04 — Telemetria opt-in preservada.** Manter ADR-006 (`GOVERNANCE_TELEMETRY=1`, append-only, sem envio sem consentimento) invariante nas 4 CLIs.

## 6. Onde já lideramos (não regredir)

- Suíte de invariantes de paridade ADR-008 (29 invariantes / 3 níveis) — Compozy não tem.
- Telemetria opt-in append-only (ADR-006) — não observada no Compozy.
- `BootstrapArgs` por CLI expondo capacidades nativas (Codex `-c`, Gemini `--approval-mode` D-05/ADR-015) com zero-value preservando F1.

## 7. Critérios de Aceitação da Análise

- **Igualdade:** RP-01..RP-04 garantem que a mesma task gera eventos/normalização/métricas equivalentes nas 4 CLIs (provável por teste, não só documentado).
- **Transparência:** RI-01..RI-04 eliminam ajuste manual no install P/M/G.
- **Portabilidade:** install validado em Go/Node/Python e monorepo, idempotente.
- **Rastreabilidade:** cada gap acima referencia evidência `gh` (Compozy) ou caminho de arquivo local (§1, §2, §5).

## 8. Log de Evidências (resumo)

**Compozy (via `gh`):** `go.mod`→`coder/acp-go-sdk v0.6.3` + `modelcontextprotocol/go-sdk v1.5.0`;
`internal/core/agent/registry_specs.go`→specs claude/codex/copilot/gemini com `--acp` + fallback npx;
`internal/core/agent/tool_call_name.go`→`commonToolTitleAliases`; `internal/core/model/content.go`→
`ToolUseBlock{Input,RawInput}`; `internal/core/memory/store.go`→150/12KB·200/16KB; `internal/cli/setup.go`
+ `internal/setup/agents.go`→`DetectInstalledAgents()` 40+ alvos; `internal/core/run/executor/runner.go`→
retry backoff 1.5. NO EVIDENCE FOUND: suíte de invariantes cross-runtime, token-budget, detecção de stack, workaround Copilot.

**ai-spec-harness (local):** `go.mod`→`coder/acp-go-sdk v0.13.0`; Concurrent/BatchSize **implementados**
em `internal/taskloop/runloop.go` (com `TestRunLoop_Concurrent_Race`); spec-hash **ausente** em
`internal/runtime/` (grep vazio); `normalization-rules.yaml`→`copilot`/`gemini` sem `input_mappings`.
