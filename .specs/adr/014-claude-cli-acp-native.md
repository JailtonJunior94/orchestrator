# ADR-014: Claude CLI 2026 — paridade Compozy acima da camada ACP

**Status:** Proposta
**Data:** 2026-05-21
**Autores:** -

---

## Contexto

Claude já é runtime ACP de primeira classe no harness desde ADR-009. `internal/runtime/specs/claude.go:28-45` declara `Command: "claude-agent-acp"` com fallback `npx --yes @agentclientprotocol/claude-agent-acp@0.1.0`. O substrato ACP (`internal/runtime/client/client.go`, `runner.go`, `events/`, `persistence/`) é tool-agnóstico desde a generalização entregue por ADR-012 (Copilot) e ADR-013 (Codex). **Não há gap de transporte para Claude.**

Em 2026 o `compozy/compozy` (`main` SHA `7f38c445069bd83a8e96bcd925ee1f12fde74435`) construiu, acima da mesma camada ACP que o harness consome (`coder/acp-go-sdk v0.6.3`), cinco capacidades que ficam ausentes no Orchestrator quando Claude é invocado em modo `--runtime=acp`:

1. **Servidor MCP reservado** (`internal/core/agents/mcpserver/server.go`) com tool única `run_agent(agent_name, prompt)` permitindo hand-off recursivo entre agentes.
2. **Normalização de tool-calls driver-aware** (`internal/core/agent/tool_call_input.go::buildNormalizedToolUseBlock`) canonicalizando inputs divergentes (Claude `bash` vs Codex `shell`, `startLine` vs `startLineNumberBaseOne`, etc.).
3. **Memória 2-tier markdown** (`internal/core/memory/store.go`) — workflow (`MEMORY.md` 150 lin/12 KB) + task (`<task>.md` 200 lin/16 KB), com compactação **prompt-driven** quando o limite é atingido.
4. **Pipeline de hooks in-process Go** (`internal/core/kernel/` + `internal/core/prompt/common.go`) tipado, com pontos canônicos como `prompt.pre_build`, `prompt.post_build`, `review.pre_resolve`, `review.post_fix` (33 totais documentados).
5. **Modo de execução separado para review** (`internal/core/run/executor/review_hooks.go`) com provider (GitHub/CodeRabbit) — não é sub-loop dentro de execução normal.

Investigação detalhada das mecânicas Compozy está em [`docs/research/compozy-adaptation-claude-2026.md`](../../docs/research/compozy-adaptation-claude-2026.md). Resumo das implicações arquiteturais para o harness:

- **MCP nested-agent ausente**: sub-agents só existem na orquestração explícita de `execute-all-tasks`; agente não pode delegar a outro agente em runtime. Bloqueia fluxos como "executor delega revisão a reviewer".
- **Tool-call normalization ausente**: `events.jsonl` registra `bash` em sessões Claude e `shell` em sessões Codex — telemetria multi-tool fragmentada. ADR-008 (paridade) tem que cobrir mapeamentos em test fixtures, não em runtime.
- **Memória 2-tier ausente**: harness usa `~/.claude/projects/.../memory/MEMORY.md` (auto-memory do Claude Code, prompt-driven) sem split workflow/task e sem limites enforced. PRDs grandes acumulam contexto irrelevante.
- **Hooks em modo ACP ausentes**: shell hooks de `.claude/hooks/*.sh` (`validate-governance.sh`, `validate-token-budget.sh`, etc.) são lidos por Claude Code em modo interativo, **não pelo Orchestrator quando ele spawna `claude-agent-acp` via ACPRunner**. Governance hooks são silenciosamente ignorados em sessões orquestradas — gap crítico.
- **Evidence sem campos Claude-2026**: `cache_read_tokens`, `cache_creation_tokens`, `thinking_tokens` estão disponíveis no payload do ACP-SDK mas não são extraídos. Sem visibilidade de custo real ou correlação reasoning ↔ qualidade.
- **Auto-review ausente**: skill `review` existe em `.agents/skills/review/SKILL.md` mas é invocada manualmente. Loops batch produzem código não-revisado.

Adicionar paridade exige adições estruturais em quatro fases (F2–F5 do roadmap Claude), todas reusando o substrato ACP já validado. Esta ADR documenta as decisões. Implementação fica para PRD/TechSpec/Tasks subsequentes consumindo `create-tasks`.

## Alternativas Consideradas

| Alternativa | Vantagens | Desvantagens |
|-------------|-----------|--------------|
| Status quo (Claude apenas como F1-Claude / ADR-009) | Zero código novo; F2-Copilot/F2-Codex herdam automaticamente quando entregues | Hooks de governance silenciosamente ignorados em modo ACP; sem nested-agent; sem normalização cross-tool; evidência sem campos Claude-2026 |
| Implementar tudo de uma vez em F2-Claude monolítico | Entrega rápida do pacote completo | Risco alto (4 subsistemas novos juntos); difícil revisar; impossível paralelizar tasks; trade-offs ficam ocultos |
| Migrar 100% dos shell hooks `.claude/hooks/*.sh` para Go in-process eliminando os shell | Stack unificado | Quebra UX interativo de Claude Code (que lê os shell hooks via `.claude/settings.json`); shell hooks têm valor próprio fora de sessões ACP |
| Implementar MCP via servidor externo (não in-process) | Isolamento de processo | Overhead de IPC; complicação de lifecycle; perde o acesso direto ao registry/spec do parent runner |
| Reescrever auto-review como hook + provider espelhando Compozy literalmente (GitHub/CodeRabbit) | Paridade exata com Compozy | Adiciona dependências externas (gh provider, CodeRabbit API); fora do escopo de governança local do harness (R-GOV-001 §"Não introduzir dependência externa sem justificativa") |
| Trocar memória própria por integração direta com `~/.claude/projects/memory/` | Aproveita infra Claude Code | Sem split workflow/task; sem limites enforced; ausência de `NeedsCompaction` sinalizando ao LLM compactar |

## Decisão

Decidimos adicionar **paridade Compozy acima da camada ACP** em quatro fases incrementais (F2 → F5), preservando o substrato ACP entregue por ADR-009/012/013 e os diferenciais do harness (`events.jsonl`, `tool_calls.md`, `ActivityWatchdog`, telemetria opt-in, `spec-hash`). Decisões individuais:

### D-01 — MCP nested-agent server reservado com tool `run_agent`

Novo `internal/runtime/mcpserver/` com servidor stdio MCP expondo **uma única tool** `run_agent(agent_name, prompt, model?, timeout?)`. Resolução de `agent_name` via `internal/agents/registry.go` (ADR-011). Profundidade máxima 3 (alinhado ao Compozy). Contexto serializado para o child via env var `AISPEC_RUN_AGENT_CONTEXT` (JSON com workspace_root, parent_session_id, depth). Opt-in via flag `--mcp-nested` no `task_loop.go` (default off em F2-Claude; on em F4-Claude+ quando maduro).

### D-02 — Tool-call normalization driver-aware

Novo `internal/runtime/events/normalize.go` com `BuildNormalizedToolCall(driverID, kind, rawInput) NormalizedToolCall`. Tabelas de alias em `.agents/normalization-rules.yaml` carregado via `go:embed` em init. `events.jsonl` ganha campo `normalized_name` ao lado de `raw_name` (raw preservado para debug). `tool_calls.md` renderiza nome normalizado. Sempre ativo a partir de F2-Claude; flag `--no-normalize` desliga para diagnóstico.

### D-03 — Memória 2-tier markdown com compactação prompt-driven

Novo `internal/runtime/memory/store.go` replicando `compozy/internal/core/memory/store.go`:
- Workflow: `.specs/<prd>/memory/MEMORY.md` — defaults 150 lin / 12 KB
- Task: `.specs/<prd>/memory/<taskFileName>` — defaults 200 lin / 16 KB
- `FileState.NeedsCompaction` setado quando limite ultrapassado
- Integração em `runner.go`: antes de `c.Open`, prepend bloco "## Memory Context" + diretiva "compact the flagged memory files before proceeding" quando flag setada (idêntico Compozy, **prompt-driven, não code-driven**)
- Limites configuráveis via flags `--memory-workflow-limit-lines`, `--memory-workflow-limit-bytes`, `--memory-task-limit-lines`, `--memory-task-limit-bytes`

Coexiste com auto-memory de Claude Code (`~/.claude/projects/.../memory/MEMORY.md`): memory do harness **vence** quando `.specs/<prd>/memory/` existe; auto-memory é fallback.

### D-04 — Pipeline de hooks in-process Go

Novo `internal/runtime/hooks/dispatcher.go` com:
- `type Hook interface { Name() string; Run(ctx, evt) error }`
- `Dispatcher.Dispatch(ctx, name, evt)` — fan-out sequencial, abort-on-first-error
- 6 pontos canônicos iniciais despachados em `runner.go`:
  - `runtime.pre_open` — antes de `c.Open` (linha 145)
  - `prompt.pre_build` — antes de montar prompt final
  - `prompt.post_build` — depois de montar
  - `tool_call.pre_dispatch` — no loop de eventos, antes de persist
  - `tool_call.post_complete` — depois de tool result injetado
  - `session.post_end` — antes de `EnrichReport`

Migração progressiva (não substituição): hooks Go in-process **coexistem** com shell hooks em `.claude/hooks/*.sh`. Shell hooks continuam servindo modo interativo Claude Code; Go hooks servem modo orquestrado (ACPRunner). Hooks Go iniciais: `governance.go` (replica `validate-governance.sh`), `token_budget.go` (replica `validate-token-budget.sh`). Flag `--disable-hooks` desliga para debugging.

### D-05 — Evidence enrichment Claude-2026

Estender `internal/runtime/events/convert.go` para extrair, quando presentes no payload ACP:
- `cache_read_tokens`
- `cache_creation_tokens`
- `thinking_tokens`
- `tool_calls_normalized_count` (calculado em F2 via D-02)

Estender `internal/runtime/runner.go::Summary` com esses campos. Estender `internal/evidence/evidence.go` com seção opcional "Métricas Claude-2026" no `execution_report.md`. Validador: presença opcional; ausência não bloqueia. Telemetria (`internal/telemetry/`): se `GOVERNANCE_TELEMETRY=1`, append entries `claude.cache_read=N`, `claude.thinking=N`.

### D-06 — Auto-review opt-in

Flag `--auto-review` em `cmd/ai_spec_harness/task_loop.go`. Default `false`. Quando true e session end completou sem erro:
- Spawnar nova `ACPRunner` com prompt da skill `review` (`.agents/skills/review/SKILL.md`) + diff acumulado da task (via `git diff` no `workDir`)
- Persistir resultado em `evidence/<task>/review.md`
- Parsear resultado: presença de `BLOQUEADO`, `CRÍTICO`, severidade `hard` → `Summary.ReviewStatus = "blocked"` e `execution_report.md` ganha seção "Review Block"
- Hook canônico novo `session.post_review`

**Não copiar literalmente o Compozy** (provider GitHub/CodeRabbit). Para o harness, "provider" é a skill local; mais simples, alinhado com `R-GOV-001` (governança transversal sobre adição de complexidade).

### D-07 — `wrapper.ValidTools` permanece sem Claude (decisão deliberada)

Confirmação explícita: **Claude continua fora de `internal/wrapper/wrapper.go::ValidTools`** (validado por `internal/wrapper/wrapper_test.go:213-214` — comentário: "claude should not be in ValidTools (uses hooks)"). O wrapper serve tools que **não** têm sistema de hooks próprio (codex, gemini, copilot precisam da validação manual). Claude entra na via dos hooks (shell em modo interativo + Go in-process em modo ACP via D-04). Esta ADR **rejeita** explicitamente qualquer proposta de adicionar Claude ao wrapper.

### Não-objetivos

- Vector store, embeddings, indexação semântica de memória (R-GOV-001 §"Não introduzir dependência externa sem justificativa")
- Fork ou extensão do schema ACP wire-format (usar `coder/acp-go-sdk` como vem)
- Provider GitHub/CodeRabbit para review (ver D-06; review usa skill local)
- Substituir telemetria atual (ADR-006 cobre opt-in append-only; D-05 apenas estende campos)
- Substituir `spec-hash` validation (`internal/specdrift/` permanece como concern de governança, não de runtime)
- Migrar 100% dos shell hooks para Go (shell hooks têm valor próprio em modo interativo; ver D-04)

## Consequências

### Positivas

- Paridade funcional Claude ↔ Compozy acima da camada ACP: MCP nested, normalização cross-tool, memória 2-tier, hooks in-process, evidence Claude-2026, auto-review opt-in
- Governance hooks (`validate-governance`, `validate-token-budget`) passam a executar em modo ACP também — fecha gap de paridade entre modo interativo e orquestrado
- Telemetria ganha visibilidade de cache (potencial ~80% de redução de custo via prompt caching documentado pela Anthropic)
- Roadmap fica decomposto em fases incrementais (F2/F3/F4/F5) com riscos isolados — paralelização de tasks viável dentro de cada fase via `execute-all-tasks`
- Diferenciais do harness preservados: `events.jsonl`, `tool_calls.md`, `ActivityWatchdog`, telemetria opt-in, `spec-hash`
- Infra de `BootstrapArgs` entregue por ADR-013 já cobre Claude (no-op atual) — extensões D-04/D-05 reusam padrão

### Negativas / Riscos

- Biblioteca Go MCP-SDK ainda matura em 2026-Q2 — risco de interface mudar. Mitigação: isolar dependência atrás de interface interna; vendor inicial se necessário.
- Migração `.claude/hooks/*.sh` parcial para Go pode confundir contribuidores ("qual roda quando?"). Mitigação: documentar precedência em `CLAUDE.md` raiz; cada hook Go cita o shell equivalente em comentário.
- Memória 2-tier pode duplicar com auto-memory de Claude Code. Mitigação: documentar precedência (`.specs/<prd>/memory/` vence; auto-memory é fallback).
- MCP nested-agent expõe vetor de DoS (recursão infinita). Mitigação: profundidade máxima 3; timeout obrigatório; trace de `parent_session_id` em contexto serializado.
- Auto-review opt-in dobra custo de tokens por task. Mitigação: opt-in explícito; documentar trade-off; futuro cache de review por SHA do diff.
- Tool-call normalization pode mascarar bugs upstream. Mitigação: `raw_name` preservado lado a lado; flag `--no-normalize` para diagnóstico.

### Neutras / Observações

- F1-Claude (ADR-009) permanece operacional sem mudanças — nenhuma fase F2+ exige migração breaking de configurações existentes
- F2-Claude e F2-Copilot/F2-Codex podem ser entregues independentemente (MCP é per-runtime; shell hooks de Claude não dependem de paridade com Codex)
- Parity check (ADR-008) deve receber invariantes novos cobrindo normalização e MCP nested em F2; cobrir hooks em F3
- Skills `.claude/skills/` podem declarar hooks em frontmatter a partir de F3 (declaração apenas; implementação obrigatória em Go — ver D-04)
- `.claude/agents/` (bugfixer, reviewer, task-executor, etc.) passa a ser resolvido via `internal/agents/registry.go` quando MCP `run_agent` for invocado (F2-Claude)
- `CLAUDE.md` raiz precisa nova seção "Runtime Capabilities" listando MCP + hooks expostos (entregável da fase que primeiro materializar a capability; default em F2-Claude)

## Referencias

- ADR-009 — `.specs/adr/009-acp-protocol-adoption.md` (pinning SDK, ACP base)
- ADR-011 — `.specs/adr/011-agent-registry-declarativo.md` (registry consumido por MCP `run_agent`)
- ADR-012 — `.specs/adr/012-copilot-cli-acp-native.md` (Copilot ACP, generalização runner)
- ADR-013 — `.specs/adr/013-codex-cli-acp-native.md` (Codex ACP, `BootstrapArgs` infra)
- ADR-008 — `docs/adr/008-parity-multi-tool-invariants.md` (paridade — invariantes a estender em F2/F3)
- ADR-006 — `docs/adr/006-telemetria-feedback-cycle.md` (telemetria opt-in; campos novos em D-05)
- ADR-005 — `.specs/adr/005-skills-lock-sha256.md` (skills lock, contexto para `internal/specdrift/`)
- Pesquisa — [`docs/research/compozy-adaptation-claude-2026.md`](../../docs/research/compozy-adaptation-claude-2026.md) (gap map detalhado e roadmap por fase)
- PRD — [`.specs/prd-claude-cli-acp-2026/prd.md`](../prd-claude-cli-acp-2026/prd.md) (consome esta ADR; RF-01..RF-06)
- TechSpec — [`.specs/prd-claude-cli-acp-2026/techspec.md`](../prd-claude-cli-acp-2026/techspec.md)
- Compozy — `internal/core/agents/mcpserver/server.go` (`reservedToolName = "run_agent"`)
- Compozy — `internal/core/agent/tool_call_input.go::buildNormalizedToolUseBlock`
- Compozy — `internal/core/memory/store.go` (limites 150/12KB e 200/16KB)
- Compozy — `internal/core/kernel/` + `internal/core/prompt/common.go` (hook dispatcher)
- Compozy — `internal/core/run/executor/review_hooks.go` (review mode separado)
- Harness — `internal/runtime/specs/spec.go:5-101` (Spec estendida por ADR-013)
- Harness — `internal/runtime/runner.go:89-220` (ACPRunner.Run alvo F2-F5)
- Harness — `internal/wrapper/wrapper_test.go:213-214` (Claude fora de ValidTools por design)
- Harness — `.claude/hooks/` (shell hooks atuais, mantidos coexistindo com Go in-process)
