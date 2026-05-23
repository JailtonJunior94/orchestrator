<!-- spec-version: 1 -->

# Documento de Requisitos do Produto (PRD)

## Claude-CLI 2026 — Paridade Compozy Acima da Camada ACP

**Slug:** `claude-cli-acp-2026`
**Data:** 2026-05-21
**Status:** Proposta
**ADR consumida:** [ADR-014 — Claude CLI 2026: paridade Compozy acima da camada ACP](../adr/014-claude-cli-acp-native.md)
**Pesquisa de base:** [`docs/research/compozy-adaptation-claude-2026.md`](../../docs/research/compozy-adaptation-claude-2026.md)

---

## Visão Geral

Claude já é runtime ACP nativo no `ai-spec-harness` desde ADR-009. O substrato ACP é tool-agnóstico (`internal/runtime/client/client.go`, `runner.go`, `events/`, `persistence/`) e foi validado pelas entregas de F1-Copilot (ADR-012) e F1-Codex (ADR-013). **Não há gap de transporte.**

O gap Claude-2026 está em **paridade funcional acima da camada ACP**: o `compozy/compozy` (`main` SHA `7f38c44506`), consumindo o mesmo `coder/acp-go-sdk v0.6.3`, construiu cinco capacidades ausentes no Orchestrator — MCP nested-agent, normalização de tool-calls driver-aware, memória 2-tier markdown, pipeline de hooks in-process Go e auto-review opt-in. Investigação detalhada em [`compozy-adaptation-claude-2026.md`](../research/compozy-adaptation-claude-2026.md); decisões em ADR-014.

Esta PRD escopa as quatro fases de adaptação (F2-Claude → F5-Claude) como pacote único, decomposto em RF-01..RF-06, preservando os diferenciais do harness (`events.jsonl`, `tool_calls.md`, `ActivityWatchdog`, telemetria opt-in, `spec-hash`) e a regra deliberada de manter Claude fora de `wrapper.ValidTools` (ver ADR-014 §D-07 e `internal/wrapper/wrapper_test.go:213-214`).

**Ator principal:** Mantenedor do `ai-spec-harness` que orquestra Claude via `ai-spec task-loop --tool claude --runtime acp`.
**Ator secundário:** Desenvolvedor que invoca skills do harness em sessões orquestradas Claude e depende de governance hooks executarem (hoje silenciosamente ignorados em modo ACP).

---

## Objetivos

- Fechar a paridade funcional Claude ↔ Compozy nas cinco capacidades documentadas em ADR-014, sem regressão dos diferenciais do harness.
- Garantir que governance hooks (`validate-governance`, `validate-token-budget`) executem em modo ACP — fim do gap entre modo interativo Claude Code e modo orquestrado.
- Tornar visíveis métricas Claude-2026 (`cache_read_tokens`, `cache_creation_tokens`, `thinking_tokens`) em `execution_report.md` e telemetria opt-in.
- Habilitar nested-agent execution via MCP `run_agent` para fluxos como "executor delega revisão a reviewer" — sem precisar do orquestrador humano sequenciar.
- Permitir compactação determinística e auditável de contexto via memória 2-tier (workflow + task) com limites byte/linha e diretiva prompt-driven.

**Critérios de sucesso mensuráveis:**

1. `ai-spec task-loop --tool claude --runtime acp --mcp-nested .specs/prd-X` registra eventos `tool_call_kind="nested_agent"` em `events.jsonl` quando o Claude invoca `run_agent` no diff (RF-01).
2. `events.jsonl` em sessão Claude e Codex executando a mesma operação shell registra o mesmo `normalized_name="bash"` (RF-02), com `raw_name` preservado lado a lado.
3. Sessão Claude com `.specs/<prd>/memory/MEMORY.md` > 150 linhas registra `NeedsCompaction=true` e o `execution_report.md` documenta "Memory Compaction Requested: true" (RF-03).
4. Hook `governance.go` (in-process Go) bloqueia sessão com `AGENTS.md` ausente, retornando erro antes de `c.Open` — paridade com `validate-governance.sh` em modo interativo (RF-04).
5. `execution_report.md` em sessão Claude com prompt caching ativo registra `cache_read_tokens > 0` em "Métricas Claude-2026" (RF-05).
6. `--auto-review` ativo em sessão que produz código com issue `hard` resulta em `Summary.ReviewStatus="blocked"` e seção "Review Block" no relatório (RF-06).
7. `make test` e `make integration` verdes; cobertura ≥ 70% (threshold atual enforced); 29 invariantes de ADR-008 + 2 novos invariantes (normalização e MCP nested) passam em parity check.
8. Zero regressão em sessões Claude existentes sem flags novas (defaults preservam comportamento atual).

---

## Histórias de Usuário

- Como **mantenedor do harness**, quero que governance hooks executem também em modo ACP (não só interativo) para que skills `execute-task` orquestradas não bypassem validações de `AGENTS.md` e budget de tokens.
- Como **mantenedor**, quero ver `cache_read_tokens` no `execution_report.md` para correlacionar uso de prompt caching com economia real de custo.
- Como **agente executor**, quero poder invocar `run_agent("reviewer", "<diff>")` em runtime sem depender do orquestrador humano sequenciar a sessão — fechando o ciclo dentro da própria task.
- Como **agente em refatoração longa**, quero que a memória 2-tier sinalize quando MEMORY.md ultrapassou 150 linhas para que eu compacte o conteúdo dentro do próprio turn (prompt-driven).
- Como **analista de telemetria multi-tool**, quero que `events.jsonl` de Claude e Codex registrem o mesmo `normalized_name` para tools com mesma semântica (`bash`/`shell`), permitindo agregação cross-runtime.
- Como **autor de PRD**, quero opt-in `--auto-review` para batches longos onde quero defesa em profundidade contra código quebrado sem custo adicional em tasks pequenas.

---

## Funcionalidades Core

### RF-01 — MCP nested-agent server reservado com tool `run_agent` (F2-Claude)

Implementar servidor MCP stdio expondo **uma única tool** `run_agent(agent_name, prompt, model?, timeout?)`. Resolução de `agent_name` via `internal/agents/registry.go` (ADR-011). Spawn de child session ACP no mesmo processo, com contexto serializado via env var `AISPEC_RUN_AGENT_CONTEXT` (JSON com workspace_root, parent_task, parent_session_id, depth). Profundidade máxima configurável (default 3).

**Por que:** sem nested-agent, sub-agents só existem na orquestração explícita do harness; agente não pode delegar a outro agente em runtime. Bloqueia fluxos como "executor delega revisão a reviewer" dentro da mesma task.

**Requisitos funcionais:**
- RF-01.1: Flag `--mcp-nested` em `cmd/ai_spec_harness/task_loop.go` (default `false`). Quando true, spawna o servidor MCP em goroutine e injeta `--mcp-server stdio://...` no launcher do Claude.
- RF-01.2: Tool `run_agent` aceita `{agent_name: string, prompt: string, model?: string, timeout?: int}`; retorna `{summary: string, evidence_dir: string}`.
- RF-01.3: `agent_name` desconhecido retorna erro tipado MCP (não exit do server).
- RF-01.4: Profundidade > `AISPEC_MAX_AGENT_DEPTH` (default 3) retorna erro tipado.
- RF-01.5: Cada child session ACP produz `events.jsonl` e `execution_report.md` em sub-diretório próprio dentro do `evidence_dir` do parent. Eventos do child são também espelhados no `events.jsonl` do parent com kind `nested_agent`.
- RF-01.6: Timeout default = 5 min; max permitido = 30 min. Override via parâmetro `timeout` da tool.

### RF-02 — Normalização de tool-calls driver-aware (F2-Claude, sempre ativa)

Implementar `BuildNormalizedToolCall(driverID, kind, rawInput) NormalizedToolCall` em `internal/runtime/events/normalize.go`, com tabelas de alias por driver carregadas de `.agents/normalization-rules.yaml` via `go:embed`.

**Por que:** sem normalização, `events.jsonl` registra `bash` em sessões Claude e `shell` em sessões Codex — telemetria multi-tool fragmentada e ADR-008 (paridade) precisa cobrir mapeamentos em test fixtures.

**Requisitos funcionais:**
- RF-02.1: `events.jsonl` ganha campo `normalized_name` ao lado de `raw_name`. Ambos preservados.
- RF-02.2: `tool_calls.md` renderiza nome normalizado.
- RF-02.3: Tabela inicial em `.agents/normalization-rules.yaml` cobre pelo menos: `bash`↔`shell`, `read_file`↔`read`, `write_file`↔`write`, `web_search`↔`search_query`, `image_search`↔`image_query`.
- RF-02.4: Flag `--no-normalize` em `task_loop.go` desliga normalização (debug). Default `false` (normalização ativa).
- RF-02.5: Normalização **não** pode mascarar inputs — `RawInput` em `events.jsonl` permanece preservado.

### RF-03 — Memória 2-tier markdown com limites configuráveis (F3-Claude)

Implementar `internal/runtime/memory/store.go` replicando `compozy/internal/core/memory/store.go`:

**Por que:** harness usa auto-memory de Claude Code sem split workflow/task e sem limites enforced; PRDs grandes acumulam contexto irrelevante; LLM não recebe sinal para compactar.

**Requisitos funcionais:**
- RF-03.1: Workflow memory em `.specs/<prd>/memory/MEMORY.md`; defaults 150 lin / 12 KB.
- RF-03.2: Task memory em `.specs/<prd>/memory/<taskFileName>`; defaults 200 lin / 16 KB.
- RF-03.3: `FileState.NeedsCompaction` setado quando limite ultrapassado em qualquer dos dois eixos (linhas OU bytes).
- RF-03.4: `runner.go` lê workflow + task memory antes de `c.Open` e prepend bloco "## Memory Context" ao prompt. Quando `NeedsCompaction=true` em qualquer scope, anexa diretiva textual "compact the flagged memory files before proceeding".
- RF-03.5: Flags `--memory-workflow-limit-lines` (default 150), `--memory-workflow-limit-bytes` (default 12288), `--memory-task-limit-lines` (default 200), `--memory-task-limit-bytes` (default 16384).
- RF-03.6: Quando `.specs/<prd>/memory/` **não** existe, harness não força criação — auto-memory de Claude Code permanece como fallback sem mudança.
- RF-03.7: `execution_report.md` documenta "Memory Compaction Requested: true|false" e "Memory Workflow Bytes / Task Bytes" como métricas.

### RF-04 — Pipeline de hooks in-process Go (F3-Claude)

Implementar `internal/runtime/hooks/dispatcher.go` com 6 pontos canônicos despachados em `runner.go`, e migração inicial de dois shell hooks (`validate-governance.sh`, `validate-token-budget.sh`) para Go.

**Por que:** shell hooks em `.claude/hooks/*.sh` são executados pelo próprio Claude Code em modo interativo, **não pelo Orchestrator** quando ele spawna `claude-agent-acp` via ACPRunner. Governance hooks são silenciosamente ignorados em sessões orquestradas.

**Requisitos funcionais:**
- RF-04.1: Interface `type Hook interface { Name() string; Run(ctx context.Context, evt Event) error }`.
- RF-04.2: `Dispatcher.Dispatch(ctx, name string, evt any) error` — fan-out sequencial, abort-on-first-error.
- RF-04.3: 6 pontos canônicos despachados em `runner.go`: `runtime.pre_open`, `prompt.pre_build`, `prompt.post_build`, `tool_call.pre_dispatch`, `tool_call.post_complete`, `session.post_end`.
- RF-04.4: Hook `governance.go` (in-process Go) replica `validate-governance.sh`: valida existência de `AGENTS.md` em `j.WorkDir`. Falha em `runtime.pre_open` aborta sessão antes de `c.Open`.
- RF-04.5: Hook `token_budget.go` replica `validate-token-budget.sh`: valida budget via `internal/metrics`. Falha em `prompt.post_build` aborta antes do prompt ser enviado.
- RF-04.6: Shell hooks em `.claude/hooks/*.sh` **coexistem** — não são removidos. Continuam servindo modo interativo Claude Code.
- RF-04.7: Flag `--disable-hooks` em `task_loop.go` desliga dispatcher (debug). Default `false`.
- RF-04.8: Skills `.claude/skills/<name>/` podem declarar metadata `hook:` no frontmatter (declaração apenas; implementação obrigatória em Go).

### RF-05 — Evidence enrichment Claude-2026 (F4-Claude)

Extrair campos opcionais do payload ACP e estender `execution_report.md`.

**Por que:** `cache_read_tokens`, `cache_creation_tokens`, `thinking_tokens` estão disponíveis no `acp.SessionUpdate` payload e medem economia real de custo (prompt caching ~80%) e qualidade (thinking ↔ output). Sem extração, telemetria fica cega.

**Requisitos funcionais:**
- RF-05.1: `internal/runtime/events/convert.go` extrai (quando presentes): `cache_read_tokens`, `cache_creation_tokens`, `thinking_tokens`, `tool_calls_normalized_count`.
- RF-05.2: `internal/runtime/runner.go::Summary` ganha esses campos (todos opcionais, default 0).
- RF-05.3: `internal/evidence/evidence.go` ganha seção "Métricas Claude-2026" no `execution_report.md` — opcional. Validador: ausência não bloqueia.
- RF-05.4: Telemetria (`internal/telemetry/`): quando `GOVERNANCE_TELEMETRY=1`, append entries `claude.cache_read=N`, `claude.cache_creation=N`, `claude.thinking=N` ao log telemetria.

### RF-06 — Auto-review opt-in (F5-Claude)

Implementar flag `--auto-review` que, após session end sem erro, spawna nova `ACPRunner` com prompt da skill `review` + diff acumulado.

**Por que:** skill `review` existe em `.agents/skills/review/SKILL.md` mas é invocada manualmente. Loops batch (`execute-all-tasks`) produzem código não-revisado; defesa em profundidade ausente.

**Requisitos funcionais:**
- RF-06.1: Flag `--auto-review` em `task_loop.go` (default `false`).
- RF-06.2: Quando true e session end sem erro, spawnar nova `ACPRunner` (mesmo tool, mesmo spec) com:
  - Prompt: skill `review` carregada de `.agents/skills/review/SKILL.md` + diff (`git diff --staged + git diff` em `workDir`)
  - WorkDir: mesmo do parent
  - EvidenceDir: `evidence/<task>/review/`
- RF-06.3: Persistir resultado em `evidence/<task>/review.md`.
- RF-06.4: Parser procura por marcadores `BLOQUEADO`, `CRÍTICO`, severidade `hard` no resultado. Quando presentes, setar `Summary.ReviewStatus = "blocked"`.
- RF-06.5: `execution_report.md` ganha seção "Review Block" com lista de issues hard quando review bloquear.
- RF-06.6: Hook canônico novo `session.post_review` despachado depois do review (entry-point para futuras integrações).
- RF-06.7: Não invocar `--auto-review` recursivamente — review session **não** dispara nova review session mesmo com flag setada.

---

## Experiência do Usuário

CLI puro — sem UI gráfica.

Fluxo principal F2..F5 ativos:

```bash
ai-spec task-loop \
  --tool claude --runtime acp \
  --mcp-nested \
  --auto-review \
  --memory-workflow-limit-lines 150 \
  .specs/prd-minha-feature
```

A sessão Claude:
1. Lê `.specs/prd-minha-feature/memory/MEMORY.md` e `<task>.md` (se existirem); injeta no prompt.
2. Despacha `runtime.pre_open` (validar AGENTS.md), `prompt.pre_build`, `prompt.post_build` (validar budget tokens).
3. Spawna `claude-agent-acp` com MCP server stdio reservado (`run_agent` disponível).
4. Loop de eventos com normalização de tool-calls; `events.jsonl` ganha `normalized_name` + `raw_name`.
5. Se memória sinalizou `NeedsCompaction=true`, prompt inclui diretiva.
6. Se Claude invoca `run_agent("reviewer", "<context>")`, spawna nested session em `evidence/<task>/nested/`; depth tracked.
7. Session end → `session.post_end` hook executa.
8. Se `--auto-review` ativa, nova `ACPRunner` invoca skill `review` com diff; resultado em `evidence/<task>/review.md`; `session.post_review` hook executa.
9. `execution_report.md` registra seções padrão + "Métricas Claude-2026" + "Review Block" (se bloqueou).

Defaults preservam comportamento atual:
- Sem flags → idêntico a F1-Claude (ADR-009)
- Apenas `--mcp-nested` → habilita F2-Claude
- Adicionar memória/hooks/auto-review é opt-in incremental

---

## Restrições Técnicas de Alto Nível

- Implementar em Go seguindo convenções do projeto (`internal/`, DI via construtor, `internal/output.Printer` quando aplicável).
- **Não fork ou estender o schema ACP wire-format** — usar `coder/acp-go-sdk v0.6.3` como vem (R-GOV-001 §"Precedência").
- **Não introduzir dependências externas além das já declaradas em `go.mod`**, exceto biblioteca MCP-SDK Go quando madura. Caso não esteja madura em 2026-Q2, implementar wire MCP em `internal/runtime/mcpserver/wire/` com schema oficial.
- **Não migrar 100% dos shell hooks para Go** — shell hooks coexistem como camada de modo interativo Claude Code.
- **Não adicionar Claude a `internal/wrapper/ValidTools`** — decisão deliberada documentada em ADR-014 §D-07 e `wrapper_test.go:213-214`.
- **Não introduzir vector store, embeddings ou ML** para memória — manter markdown 2-tier com limites byte/linha + compactação prompt-driven (ADR-014 §Não-objetivos).
- **Não copiar literalmente** o modelo Compozy de auto-review (provider GitHub/CodeRabbit) — para o harness, "provider" é a skill local (ADR-014 §D-06).
- Cobertura de testes ≥ 70% (threshold atual enforced no CI). Cobertura específica de `mcpserver/`, `memory/`, `hooks/`: ≥ 80%.
- Compatível com Go 1.26+.
- `spec-hash` continua em `internal/specdrift/` — qualquer edição em PRD/TechSpec exige `ai-spec check-spec-drift --sync` antes de tasks consumirem.
- Telemetria continua opt-in via `GOVERNANCE_TELEMETRY=1` (ADR-006).

---

## Fora de Escopo

- Vector store, embeddings, indexação semântica de memória.
- Fork ou extensão do schema ACP wire-format.
- Provider GitHub/CodeRabbit para review (auto-review usa skill local).
- Migração de 100% dos shell hooks para Go — shell hooks coexistem.
- Adicionar Claude a `wrapper.ValidTools` (decisão deliberada deliberada).
- Substituir telemetria existente (ADR-006 cobre opt-in append-only; RF-05 apenas estende campos).
- Substituir `spec-hash` validation (`internal/specdrift/` permanece).
- Suporte a múltiplos workspaces / monorepo agregados em uma única sessão MCP.
- Dashboard gráfico ou UI web para `events.jsonl` / `tool_calls.md` / `execution_report.md`.
- Replay determinístico de sessões (Compozy tem `runs reader.go`; fora de escopo para esta PRD).
- Migrar `compozy-adaptation-codex-2026.md` ou `copilot-2026.md` para incorporar F2..F5 — pesquisas correlatas mantêm-se em estado atual.

---

## Suposições e Questões em Aberto

| # | Suposição / Questão | Decisão |
|---|---|---|
| 1 | Biblioteca Go MCP-SDK madura em 2026-Q2? | Suposição: parcialmente. Decisão: isolar atrás de interface; vendor inicial se necessário. |
| 2 | `claude-agent-acp@0.1.0` permanece estável até F5-Claude entregar? | Suposição: sim (pinning ADR-009). Decisão: política de upgrade via `audit/`. |
| 3 | Defaults dos limites de memória (150/12KB workflow; 200/16KB task) são adequados ao harness? | Suposição: sim (cópia direta de Compozy). Decisão: flags configuráveis em F3-Claude permitem ajuste sem mudar código. |
| 4 | Auto-review deve falhar a task ou apenas marcar `blocked`? | Decisão: marcar `blocked` (RF-06.4); skill consumidora (`execute-all-tasks`) decide se interrompe wave. |
| 5 | F2..F5 como PRD único ou quatro PRDs sequenciais? | Decisão: **um PRD** (este) com RF-N cobrindo F2..F5; TechSpec subdivide em waves. Reavaliar se complexidade explodir durante implementação. |
| 6 | Profundidade máxima de nested-agent deve ser configurável globalmente? | Decisão: env var `AISPEC_MAX_AGENT_DEPTH` (default 3). |
| 7 | `--auto-review` deve cachear resultado por SHA do diff? | Decisão na v1: **não** (simplicidade). Tracking como follow-up se telemetria mostrar duplicação. |
| 8 | `.specs/<prd>/memory/` vs `~/.claude/projects/.../memory/` — quem vence? | Decisão: memory do harness vence quando `.specs/<prd>/memory/` existe; auto-memory de Claude Code é fallback. Documentar em `CLAUDE.md`. |
| 9 | Normalização cobre quantos tools na v1? | Decisão RF-02.3: lista mínima (`bash`, `read`, `write`, `web_search`, `image_search`). Expansão via PR sobre `.agents/normalization-rules.yaml`. |
| 10 | Como propagar `parent_session_id` para child via env var sem vazar PII? | Decisão: usar UUID v4 gerado por sessão (não path de evidence); registrar em `events.jsonl` do parent E do child. |

---

## Critérios de Validação (resumo executivo)

| # | Critério | Como Validar |
|---|---|---|
| C-01 | F1-Claude (ADR-009) zero regressão | `make integration` cobre cenários atuais sem flags novas |
| C-02 | RF-01 MCP nested funcional | Teste E2E: prompt instrui Claude a chamar `run_agent("reviewer", "<x>")`; verificar evento `nested_agent` em `events.jsonl` |
| C-03 | RF-02 normalização cross-tool | Teste de invariante: rodar mesma operação shell em Claude e Codex; comparar `normalized_name` |
| C-04 | RF-03 memória 2-tier | Teste: criar MEMORY.md com 151 linhas; rodar sessão; checar `NeedsCompaction=true` em log e seção do `execution_report.md` |
| C-05 | RF-04 hooks Go | Teste: rodar sem AGENTS.md → hook governance aborta antes de `c.Open`; mensagem clara |
| C-06 | RF-05 evidence Claude-2026 | Teste: rodar sessão com prompt caching ativo; verificar `cache_read_tokens > 0` em `execution_report.md` |
| C-07 | RF-06 auto-review | Teste: rodar com `--auto-review` em task que produz código com `eval()` (review marca hard); checar `ReviewStatus="blocked"` |
| C-08 | Telemetria estendida | Teste: `GOVERNANCE_TELEMETRY=1 ai-spec task-loop ...`; verificar entries `claude.cache_read=N` em `.agents/telemetry.log` |
| C-09 | Parity ADR-008 estendido | `internal/parity/` ganha 2 invariantes novos (normalização cross-tool; MCP nested-agent depth); todos passam |
| C-10 | Cobertura ≥ 70% | `make test` reporta cobertura agregada ≥ 70%; `mcpserver/`, `memory/`, `hooks/` ≥ 80% |
