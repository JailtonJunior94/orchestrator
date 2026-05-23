# ai-spec-harness — Claude Code

> Use `AGENTS.md` como fonte canonica das regras deste repositorio. Stack, comandos, convencoes, estrutura, CI e padroes estao documentados em `AGENTS.md` — nao duplicados aqui.

## Instrucoes

1. Ler `AGENTS.md` no inicio da sessao.
2. `.agents/skills/` e a fonte de verdade dos fluxos procedurais.
3. Em tarefas de execucao: carregar `AGENTS.md` + `agent-governance` + skill da linguagem afetada.
4. Skills de planejamento entram apenas quando a tarefa pedir explicitamente.
5. Referencias adicionais apenas quando a tarefa exigir.
6. Preservar estilo, arquitetura e fronteiras existentes.
7. Validar mudancas com comandos proporcionais ao risco.

## Telemetria e Auditoria

- Telemetria: `GOVERNANCE_TELEMETRY=1`; ver [`docs/telemetry-feedback-cycle.md`](docs/telemetry-feedback-cycle.md); relatorio: `ai-spec-harness telemetry report`
- Auditorias salvas em `audit/`, indexadas em [`audit/README.md`](audit/README.md)

## ADRs

Consultar antes de mudancas estruturais. Template: [`.specs/adr/000-template.md`](.specs/adr/000-template.md)

- [001](.specs/adr/001-go-embed-baseline.md) — assets via `go:embed`
- [002](.specs/adr/002-fake-filesystem-testes.md) — FakeFileSystem vs afero
- [003](.specs/adr/003-paridade-semantica.md) — invariantes semanticas vs diff textual
- [004](.specs/adr/004-lazy-loading-referencias.md) — references sob demanda
- [005](.specs/adr/005-skills-lock-sha256.md) — lock file SHA-256
- [006](docs/adr/006-telemetria-feedback-cycle.md) — telemetria opt-in append-only
- [007](docs/adr/007-copilot-cli-stateless-workaround.md) — Copilot injecao manual
- [008](docs/adr/008-parity-multi-tool-invariants.md) — 29 invariantes 3 niveis
- [016](.specs/prd-fundacao-portatil/adr-016-config-hierarquico-universal.md) — config hierarquico universal (global+projeto, upward-walk, precedencia)
- [017](.specs/prd-fundacao-portatil/adr-017-fallback-launcher-chain.md) — fallback launchers genericos ordenados
- [018](.specs/prd-fundacao-portatil/adr-018-runtimeconfig-retry-backpressure.md) — RuntimeConfig + retry/backoff + backpressure observavel
- [019](.specs/prd-fundacao-portatil/adr-019-instalador-portatil-detect-verify.md) — instalador portatil: auto-deteccao, escopo global, verify

## Fundacao Portatil (Fases 1–3)

Comportamento implementado (ver [`docs/guia-instalacao-universal.md`](docs/guia-instalacao-universal.md) e [`docs/config-hierarchy.md`](docs/config-hierarchy.md)):

### Instalacao e Verificacao

```bash
ai-spec-harness install .              # auto-detecta agentes, instala assets
ai-spec-harness install . --global     # escopo global em ~/.aispec
ai-spec-harness verify .               # reporta current/missing/drifted por skill/agente
ai-spec-harness verify --global        # verificacao global
```

- `--tools` e opcional: sem a flag, detecta automaticamente via binario no PATH + dirs de config.
- Idempotente: reexecutar converge para o mesmo estado (100% `current`).
- Bootstrap em repo vazio < 30s (RF-11).

### Hierarquia de Config

Precedencia deterministica (ADR-016):

```
flags CLI  >  workspace (.claude/config.yaml)  >  global (~/.aispec/config.yaml)  >  defaults built-in
```

- Config global: `~/.aispec/config.yaml` (opt-in; ausencia nao-fatal).
- Config de projeto: upward-walk a partir do CWD (marcadores: `.git/`, `.aispec/`, `.claude/`, `.agents/`).
- Merge campo a campo: cada camada so sobrescreve campos nao-zero.

### Fallback Launchers (ADR-017)

Quando o binario ACP direto nao esta no PATH, o harness tenta os launchers alternativos da cadeia
(ex.: `npx @zed-industries/codex-acp`). O fallback e transparente — resultado identico ao binario
direto.

### RuntimeConfig Unificado (ADR-018)

`RuntimeConfig` embute em `Job`: `Timeout`, `MaxRetries`, `RetryBackoffMultiplier`, `Concurrent`,
`BatchSize`. Zero-value de cada campo preserva comportamento F1 (sem regressao). Configuravel via
`config.yaml` ou flags CLI.

## Runtime Capabilities (F2-Claude+)

Wave F2-Claude ativa quando `--runtime acp` e flags especificas. Defaults preservam comportamento F1-Claude.

### MCP Nested Agent (`--mcp-nested`, RF-01)

```bash
ai-spec task-loop --tool claude --runtime acp --mcp-nested .specs/prd-X
```

- Spawna `internal/runtime/mcpserver.Server` em goroutine antes de `c.Open`.
- Expoe tool `run_agent(agent_name, prompt, model?, timeout?)` via protocolo MCP stdio.
- Profundidade maxima: `AISPEC_MAX_AGENT_DEPTH` (default 3); exceder retorna erro MCP tipado.
- Child sessions produzem `events.jsonl` e `execution_report.md` em sub-dir proprio.
- Eventos do child espelhados no parent com kind `nested_agent`.
- **Nao modifica `.claude/hooks/*.sh`** — shell hooks coexistem para modo interativo.

### Normalizacao de Tool-Calls (`--no-normalize`, RF-02)

- Sempre ativa por default; `--no-normalize` desabilita (debug).
- `events.jsonl` ganha campos `normalized_name` e `raw_name` lado a lado.
- Tabela de alias em `.agents/normalization-rules.yaml` (embedded via `go:embed`).
  - Claude: `bash→bash`, `read_file→read`, `write_file→write`, `str_replace_editor→edit`
  - Codex: `shell→bash`, `search_query→web_search`, `image_query→image_search`
- `RawInput` nunca mutado — `--no-normalize` recupera comportamento pre-F2 byte-identical.
- `tool_calls.md` renderiza nome normalizado quando presente.

### Memory Store 2-tier + Hooks Dispatcher (F3-Claude, RF-03 + RF-04)

```bash
ai-spec task-loop --tool claude --runtime acp \
  --memory-workflow-limit-lines 100 .specs/prd-X
```

- `internal/runtime/memory/` implementa store 2-tier: workflow (150 linhas / 12 KB) + task (200 linhas / 16 KB).
- Memory injetada como `## Memory Context` no prompt antes de `c.Open`.
- `NeedsCompaction=true` anexa diretiva textual de compactacao ao prompt.
- `internal/runtime/hooks/` dispatcher registra 6 pontos canonicos:
  - `runtime.pre_open` — governance hook (valida AGENTS.md)
  - `prompt.pre_build` / `prompt.post_build` — token_budget hook
  - `tool_call.pre_dispatch` / `tool_call.post_complete` — extensivel
  - `session.post_end` — memory_persist hook escreve MEMORY.md
- `--disable-hooks` desabilita todos os hooks (debug; sem regressao F1/F2).

### Precedencia Memoria (F3-Claude)

Quando `.specs/<prd>/memory/` existe, memoria do harness vence sobre auto-memory de Claude Code.
Fallback: sem o diretorio, auto-memory de Claude Code permanece sem alteracao.

### Hooks: Shell vs Go

Shell hooks em `.claude/hooks/*.sh` continuam servindo o **modo interativo** (uso direto de Claude Code CLI pelo usuario).
Go hooks em `internal/runtime/hooks/` servem o **modo orquestrado** (ACPRunner via `--runtime acp`).
Os dois conjuntos coexistem sem conflito; `.claude/hooks/*.sh` nao sao modificados pelo harness.

### Metricas Claude-2026 (F4-Claude, RF-05)

- `internal/runtime/events/metrics.go` exporta `ExtractClaudeMetrics(raw)` e `LogClaudeMetrics`.
- Campos acumulados em `Summary`: `cache_read_tokens`, `cache_creation_tokens`, `thinking_tokens`.
- Telemetria opt-in via `GOVERNANCE_TELEMETRY=1` — campos aparecem no relatorio final.
- Nenhum dado enviado sem consentimento explcito (ADR-006).

### Auto-review Opt-in (F5-Claude, RF-06)

```bash
ai-spec task-loop --tool claude --runtime acp --auto-review .specs/prd-X
```

- Desabilitado por default (`--auto-review` necessario para ativar).
- Apos sessao principal: spawna nova `ACPRunner` com skill `review` + git diff como prompt.
- Resultado persistido em `evidence/<task>/review.md`.
- Issues com tag `[HARD]` → `Summary.ReviewStatus="blocked"`.
- Recursao hard-bloqueada: child Job tem `AutoReview=false` forcado.
- Hook `session.post_review` disparado apos review (extensivel).
