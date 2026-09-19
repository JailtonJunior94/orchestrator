# Runtime ACP com Claude — capacidades F2 a F7

Conteudo movido de `CLAUDE.md` (setembro/2026) para manter o arquivo raiz curto. Descreve as
capacidades ativadas por `--runtime acp --tool claude`. Defaults preservam o comportamento
F1-Claude. Referencia geral do `task-loop`: [`task-loop-reference.md`](task-loop-reference.md).

> Os ADRs 016-019 (fundacao portatil) e PP-001/PP-002 citados abaixo foram removidos do repositorio
> no commit `91f8cb9`; o conteudo sobrevive apenas no historico git.

## Memoria Duravel de Agentes (opt-in, F7)

```bash
ai-spec task-loop --tool claude --runtime acp --durable-memory .specs/prd-X
```

- Fachada (`internal/runtime/memory/durable.Facade`) e o **unico ponto de contato** do runtime com o subsistema — a operacao humana (`ai-spec memory`, tarefa 9.0) acessa os colaboradores diretamente, sem passar pela fachada (MD-001).
- Porta estreita `MemoryPort` declarada no pacote consumidor (`internal/runtime/memory_port.go`), satisfeita por `durable.Facade` (`var _ MemoryPort = (*durable.Facade)(nil)`). Nenhum colaborador do subsistema (`Layer`, `Page`, politicas) e importado pelo pacote consumidor.
- Ativacao: flag `--durable-memory` em `task-loop` **e** chave `durable_memory_enabled` na cascata de configuracao (`flags > workspace > global > defaults`), propagada nos dois pontos: `internal/config/resolver.go` (`mergeInto`) e `internal/taskloop/runtimeconfig.go` (`optionsToConfigOverrides`). Zero-value (`false`) preserva o caminho legado (`memory.Store`) byte a byte — provado pelo golden de paridade (`internal/runtime/prompt_parity_golden_test.go`).
- Precedencia de invariantes aplicada em um unico lugar, dentro da fachada: segredo, depois nao-perda de Fato, depois dono unico de bastao, depois orcamento de contexto.
- `internal/runtime/runner.go`: `dispatchSessionPostEnd` executa **antes** de `persistSummary` (reordenado na tarefa 7.0, RF-30/RF-34).

ADRs: [MD-001](../.specs/prd-memoria-duravel-agentes/adr-001-fachada-porta-unica-memoria.md),
[MD-002](../.specs/prd-memoria-duravel-agentes/adr-002-fato-pagina-roundtrip-lossless.md),
[MD-003](../.specs/prd-memoria-duravel-agentes/adr-003-escrita-atomica-lock-camada-lease.md),
[MD-004](../.specs/prd-memoria-duravel-agentes/adr-004-optin-paridade-byte-a-byte.md),
[MD-005](../.specs/prd-memoria-duravel-agentes/adr-005-evidencia-metricas-memoria.md).

## Fallback Launchers e RuntimeConfig

Quando o binario ACP direto nao esta no PATH, o harness tenta os launchers alternativos da cadeia
(ex.: `npx @zed-industries/codex-acp`). O fallback e transparente — resultado identico ao binario
direto.

`RuntimeConfig` embute em `Job`: `Timeout`, `MaxRetries`, `RetryBackoffMultiplier`, `Concurrent`,
`BatchSize`. Zero-value de cada campo preserva comportamento F1 (sem regressao). Configuravel via
`config.yaml` ou flags CLI — ver [`config-hierarchy.md`](config-hierarchy.md).

## MCP Nested Agent (`--mcp-nested`, RF-01)

```bash
ai-spec task-loop --tool claude --runtime acp --mcp-nested .specs/prd-X
```

- Spawna `internal/runtime/mcpserver.Server` em goroutine antes de `c.Open`.
- Expoe tool `run_agent(agent_name, prompt, model?, timeout?)` via protocolo MCP stdio.
- Profundidade maxima: `AISPEC_MAX_AGENT_DEPTH` (default 3); exceder retorna erro MCP tipado.
- Child sessions produzem `events.jsonl` e `execution_report.md` em sub-dir proprio.
- Eventos do child espelhados no parent com kind `nested_agent`.
- **Nao modifica `.claude/hooks/*.sh`** — shell hooks coexistem para modo interativo.

## Normalizacao de Tool-Calls (`--no-normalize`, RF-02)

- Sempre ativa por default; `--no-normalize` desabilita (debug).
- `events.jsonl` ganha campos `normalized_name` e `raw_name` lado a lado.
- Tabela de alias em `.agents/normalization-rules.yaml` (embedded via `go:embed`).
  - Claude: `bash→bash`, `read_file→read`, `write_file→write`, `str_replace_editor→edit`
  - Codex: `shell→bash`, `search_query→web_search`, `image_query→image_search`
- `RawInput` nunca mutado — `--no-normalize` recupera comportamento pre-F2 byte-identical.
- `tool_calls.md` renderiza nome normalizado quando presente.

## Memory Store 2-tier + Hooks Dispatcher (F3-Claude, RF-03 + RF-04)

```bash
ai-spec task-loop --tool claude --runtime acp \
  --memory-workflow-limit-lines 100 .specs/prd-X
```

- `internal/runtime/memory/` implementa store 2-tier: workflow (150 linhas / 12 KB) + task (200 linhas / 16 KB).
- Memory injetada como `## Memory Context` no prompt antes de `c.Open`.
- `NeedsCompaction=true` anexa diretiva textual de compactacao ao prompt.
- `internal/runtime/hooks/` dispatcher registra 6 pontos canonicos:
  - `runtime.pre_open` — governance hook (valida existencia de `AGENTS.md` no WorkDir)
  - `prompt.pre_build` / `prompt.post_build` — token_budget hook
  - `tool_call.pre_dispatch` / `tool_call.post_complete` — extensivel
  - `session.post_end` — memory_persist hook escreve MEMORY.md
- `--disable-hooks` desabilita todos os hooks (debug; sem regressao F1/F2).

### Precedencia de memoria

Quando `.specs/<prd>/memory/` existe, memoria do harness vence sobre auto-memory de Claude Code.
Fallback: sem o diretorio, auto-memory de Claude Code permanece sem alteracao.

### Hooks: shell vs Go

Shell hooks em `.claude/hooks/*.sh` servem o **modo interativo** (uso direto de Claude Code CLI pelo
usuario) e sao registrados em `.claude/settings.local.json` por `ai-spec install .`
(`internal/install/install.go`, `writeClaudeSettings`). Go hooks em `internal/runtime/hooks/` servem
o **modo orquestrado** (ACPRunner via `--runtime acp`). Os dois conjuntos coexistem sem conflito;
`.claude/hooks/*.sh` nao sao modificados pelo harness.

## Metricas Claude-2026 (F4-Claude, RF-05)

- `internal/runtime/events/extractor.go` exporta `ExtractClaudeMetrics(raw)` e `LogClaudeMetrics`.
- Campos acumulados em `Summary`: `cache_read_tokens`, `cache_creation_tokens`, `thinking_tokens`.
- Telemetria opt-in via `GOVERNANCE_TELEMETRY=1` — campos aparecem no relatorio final.
- Nenhum dado enviado sem consentimento explicito ([ADR-006](adr/006-telemetria-feedback-cycle.md)).

## Auto-review Opt-in (F5-Claude, RF-06)

```bash
ai-spec task-loop --tool claude --runtime acp --auto-review .specs/prd-X
```

- Desabilitado por default (`--auto-review` necessario para ativar).
- Apos sessao principal: spawna nova `ACPRunner` com skill `review` + git diff como prompt.
- Resultado persistido em `evidence/<task>/review.md`.
- Issues com tag `[HARD]` → `Summary.ReviewStatus="blocked"`.
- Recursao hard-bloqueada: child Job tem `AutoReview=false` forcado.
- Hook `session.post_review` disparado apos review (extensivel).
