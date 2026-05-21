<!-- spec-hash-prd: TBD-via-make-sync-spec-hash -->
<!-- spec-hash-techspec: TBD-via-make-sync-spec-hash -->

# Resumo das Tarefas de Implementação para Codex CLI via ACP Nativo

## Metadados
- **PRD:** `tasks/prd-codex-acp-spec/prd.md`
- **Especificação Técnica:** `tasks/prd-codex-acp-spec/techspec.md`
- **ADR material:** `tasks/adr/013-codex-cli-acp-native.md`
- **Insumo de pesquisa:** `docs/research/compozy-adaptation-codex-2026.md`
- **Precedente direto:** `tasks/prd-copilot-acp-spec/` (F1-Copilot — generalização runtime base)
- **Total de tarefas:** 12
- **Tarefas paralelizáveis:** 2.0 ↔ 3.0 (após 1.0); 8.0 ↔ 9.0 (após 6.0); 10.0 ↔ 11.0 (após 7.0)

## Tarefas

<!-- Colunas e formato canônico (MANDATÓRIO):
     - `#`: id decimal `X.Y` (sempre X.0 para tarefas de topo).
     - `Status`: ^(pending|in_progress|needs_input|blocked|failed|done)$
     - `Dependências`: ^(—|\d+\.\d+(,\s*\d+\.\d+)*)$  (em-dash unicode quando vazio)
     - `Paralelizável`: ^(—|Não|Com\s+\d+\.\d+(,\s*\d+\.\d+)*)$
     - `Skills`: skills processuais extras (descoberta agnóstica em `.agents/skills/`). Use `—` quando
       não houver. Nunca listar skills auto-carregadas (governance/linguagem) nem `*-implementation`.
     - `Fase` (OPCIONAL): inteiro positivo para agrupamento visual de fases de entrega. Pode ser
       omitida em PRDs pequenos; `execute-all-tasks` não consome esta coluna. Se incluída, mantenha
       em todas as linhas para não quebrar o parser de tabela markdown. -->

| # | Título | Status | Dependências | Paralelizável | Skills |
|---|--------|--------|--------------|---------------|--------|
| 1.0 | Estender Spec com AccessMode + BootstrapArgs (default no-op) | pending | — | — | — |
| 2.0 | Criar specs/codex.go com construtor Codex() e codexBootstrapArgs | pending | 1.0 | Com 3.0 | — |
| 3.0 | Estender Job com ReasoningEffort/AccessMode/AddDirs | pending | 1.0 | Com 2.0 | — |
| 4.0 | runner.go consome spec.BootstrapArgs e prepend ao argv | pending | 1.0, 3.0 | Não | — |
| 5.0 | Registrar codex em adrByID e runtimeACPCatalog | pending | 2.0 | Não | — |
| 6.0 | Flags --reasoning-effort + --access-mode + validação enum + warning full | pending | 5.0 | Não | — |
| 7.0 | Wiring Service.Execute propagando ReasoningEffort/AccessMode/AddDirs | pending | 4.0, 6.0 | Não | — |
| 8.0 | Aviso de depreciação em codexInvoker (sync.Once) | pending | 6.0 | Com 9.0 | — |
| 9.0 | Entrada codex → [gpt-5.5] em CompatibilityTable | pending | 6.0 | Com 8.0 | — |
| 10.0 | Sub-suite Codex em acp_integration_test.go reusando fake server | pending | 7.0 | Com 11.0 | — |
| 11.0 | Documentação F1-Codex cross-cutting (CODEX.md, AGENTS.md, cli-schema, telemetria) | pending | 7.0 | Com 10.0 | — |
| 12.0 | Smoke test real codex-acp + captura audit/ | pending | 10.0, 11.0 | — | — |

## Dependências Críticas

- **1.0 é gate de regressão Claude/Copilot**: estender interface `Spec` com `AccessMode` + `BootstrapArgs` no-op default. `claude_test.go`, `copilot_test.go`, `spec_test.go` devem permanecer 100% verdes antes de avançar para 2.0/3.0. Risco R-01 (crítico do techspec) exige esta ordem. T-05, T-10, T-11 garantem.
- **4.0 carrega invariante crítica**: `runner.go::Run` muda assinatura do argv (prepend de `BootstrapArgs`). **Diff zero** obrigatório em `internal/runtime/persistence/*`, `internal/runtime/watchdog.go`, `internal/runtime/client/*`. Risco R-08 (crítico do techspec) exige revisão obrigatória do diff antes de avançar para 5.0.
- **5.0 é precondição cirúrgica do wire-up**: registrar `"codex"` em `adrByID` e `runtimeACPCatalog` é o ponto que destrava 6.0/7.0/8.0/9.0. Sem ela, gate em `task_loop.go:82-97` continua rejeitando Codex.
- **6.0 carrega validação enum**: `--reasoning-effort {low,medium,high}` e `--access-mode {restricted,full}` validados antes de propagar. T-24, T-25 cobrem casos inválidos. Warning único para `--access-mode=full` (T-30) é obrigatório.
- **7.0 é gate de regressão CLI**: T-14 invertido (Codex aceito) + T-15 novo (combinação completa) + T-26 (Claude regressão com flags Codex ignoradas). Risco R-08 também aplica aqui.
- **10.0 é gate de paridade observacional**: reusa o fake ACP server existente (`internal/runtime/client/client_test.go`) para validar que `events.jsonl`, `tool_calls.md`, `execution_report.md` saem com a mesma estrutura que Claude/Copilot, e que spawn args carregam os `-c` overrides esperados (T-17, T-18, T-19).
- **12.0 é gate final E2E**: smoke real com `codex-acp` produz evidência forense em `audit/`. Sem isso, paridade observacional não está validada fora do fake server.

## Riscos de Integração

- **R-01 (crítico)**: regressão Claude/Copilot ao estender interface `Spec`. Tarefa 1.0 carrega gate explícito: `claude_test.go` + `copilot_test.go` + `spec_test.go` 100% verdes antes de avançar para 2.0/3.0. `BootstrapArgs(...)` no-op (campo `bootstrapArgs == nil`) retorna `nil` — testes T-10/T-11 validam.
- **R-08 (crítico)**: quebra silenciosa de invariantes forenses ao mudar `runner.go::Run`. Tarefas 4.0 e 7.0 carregam revisão obrigatória do diff em `internal/runtime/persistence/*`, `internal/runtime/watchdog.go`, `internal/runtime/client/*` (deve ser **zero linhas**). Falha desse critério bloqueia merge.
- **R-03 (alto)**: `--access-mode=full` usado acidentalmente em ambiente compartilhado. Tarefa 6.0 implementa warning único em stderr via `sync.Once`. Default permanece `restricted`. Documentação em `CODEX.md` (tarefa 11.0) explícita sobre risco.
- **Paralelismo 2.0 ↔ 3.0**: arquivos disjuntos (`specs/codex.go` vs `runtime/runner.go::Job`); ambas dependem apenas de 1.0. Seguro paralelizar.
- **Paralelismo 8.0 ↔ 9.0**: arquivos disjuntos (`taskloop/agent.go` vs `taskloop/compatibility.go`); ambas dependem de 6.0. Seguro paralelizar.
- **Paralelismo 10.0 ↔ 11.0**: arquivos disjuntos (testes vs docs); ambas dependem de 7.0. Seguro paralelizar.
- **Constantes pendentes (Q1/Q4 do PRD/research)**: `CodexNpmVersion = "0.14.0"` **confirmada em 2026-05-21** via `npm view @zed-industries/codex-acp versions`. `CodexMinNpmVersion = "0.12.0"` informacional (mínimo do compozy para `gpt-5.5`). Sem `needs_input` esperado para 2.0.
- **Caminho legado preservado por 2 versões minor**: `codexInvoker` em `internal/taskloop/agent.go:335-351` permanece operacional via decisão D-05. Tarefa 8.0 garante warning único; remoção é decisão de versão futura, fora deste PRD.
- **Smoke test (12.0) requer `codex-acp` instalado**: ambiente local em 2026-05-21 não tem o binário (`codex-acp not found`); só o CLI legado `codex` v0.132.0 está presente. Operador deve `npm install -g @zed-industries/codex-acp@0.14.0` antes do smoke OU usar fallback npx automático. CI cobre apenas matriz com fake server (10.0).
- **Tool name aliasing adiado para F2-Codex**: telemetria Codex usará nomes nativos (`search_query`, `image_query`) — dashboards multi-tool que filtram por `web_search` canônico verão divergência. Decisão D-09; risco R-06 aceito.

## Cobertura de Requisitos

| Tarefa | Requisitos cobertos | Casos de teste (techspec) |
|--------|---------------------|---------------------------|
| 1.0 | RF-04, RF-05; preserva A5 | T-05, T-10, T-11, T-19 |
| 2.0 | RF-01, RF-02, RF-03, RF-06, RF-18 | T-01, T-02, T-03, T-04, T-06, T-07, T-08, T-09, T-12 |
| 3.0 | RF-07 | T-19 |
| 4.0 | RF-08, RF-19 | T-17, T-18, T-19, T-31 |
| 5.0 | RF-12, RF-14 | T-13, T-14, T-15, T-16, T-22 |
| 6.0 | RF-09, RF-10, RF-11, RF-13, RF-22 | T-22, T-23, T-24, T-25, T-30 |
| 7.0 | RF-15, RF-26 | T-26, T-27 |
| 8.0 | RF-16 | T-28, T-29 |
| 9.0 | RF-24 | T-34, T-35 |
| 10.0 | RF-17, RF-27 | T-17, T-18, T-19, T-20, T-21 |
| 11.0 | RF-20, RF-21, RF-22, RF-23 | — (validação documental) |
| 12.0 | RF-19 (gate final), OB-02, OB-06 | smoke manual + suíte completa (T-31, T-32, T-33) |

## Grafo de Dependencias

```mermaid
graph TD
    T1["1.0 — Estender Spec com AccessMode + BootstrapArgs no-op"]
    T2["2.0 — Criar specs/codex.go + codexBootstrapArgs"] --> T1
    T3["3.0 — Estender Job com ReasoningEffort/AccessMode/AddDirs"] --> T1
    T4["4.0 — runner.go consome BootstrapArgs + prepend argv"] --> T1
    T4 --> T3
    T5["5.0 — Registrar codex em adrByID + runtimeACPCatalog"] --> T2
    T6["6.0 — Flags --reasoning-effort + --access-mode + warning full"] --> T5
    T7["7.0 — Wiring Service.Execute propagando flags"] --> T4
    T7 --> T6
    T8["8.0 — Aviso depreciação em codexInvoker (sync.Once)"] --> T6
    T9["9.0 — Entrada codex em CompatibilityTable"] --> T6
    T10["10.0 — Sub-suite Codex em acp_integration_test.go"] --> T7
    T11["11.0 — Documentação F1-Codex cross-cutting"] --> T7
    T12["12.0 — Smoke test real codex-acp + audit/"] --> T10
    T12 --> T11
```

## Legenda de Status
- `pending`: aguardando execução
- `in_progress`: em execução
- `needs_input`: aguardando informação do usuário
- `blocked`: bloqueado por dependência ou falha externa
- `failed`: falhou após limite de remediação
- `done`: completado e aprovado
