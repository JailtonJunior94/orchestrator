# Tarefa 6.0: Integrar memory + hooks no `runner.go`, flags F3 + smoke F3

<critical>Ler prd.md e techspec.md desta pasta — sua tarefa será invalidada se você pular</critical>

## Visão Geral

Wirear `internal/runtime/memory/Store` (task 4.0) e `internal/runtime/hooks/Dispatcher` (task 5.0) em `runner.go::Run()`. Implementar `memory_persist.go` completo (escreve MEMORY.md em `PointSessionPostEnd`). Adicionar flags F3 em `cmd/ai_spec_harness/task_loop.go`. Smoke E2E F3 cobrindo compactação prompt-driven e governance hook bloqueando.

<requirements>
- Estender `Job` com campos `MemoryLimits memory.Limits`, `DisableHooks bool` (`TaskFileName` já adicionado em task 3.0)
- Em `Run()`: instanciar `memory.New(j.TasksDir, j.MemoryLimits)` se `j.TasksDir != ""`; ler workflow + task; injetar `## Memory Context` no prompt; anexar diretiva "compact the flagged memory files before proceeding" quando `NeedsCompaction=true`
- Em `Run()`: instanciar `hooks.New()` com hooks default (`governance`, `token_budget`, `memory_persist`) registrados se `!j.DisableHooks`; despachar em todos os 6 pontos canônicos nas posições documentadas no techspec
- Implementar `hooks/memory_persist.go` completo: em `PointSessionPostEnd`, escrever `MEMORY.md` com `Summary` enriquecido (formato a definir; manter ≤150 linhas)
- 5 flags novas em `cmd/ai_spec_harness/task_loop.go`:
  - `--memory-workflow-limit-lines` (default 150)
  - `--memory-workflow-limit-bytes` (default 12288)
  - `--memory-task-limit-lines` (default 200)
  - `--memory-task-limit-bytes` (default 16384)
  - `--disable-hooks` (default `false`)
- T-16 em `task_loop_test.go` cobrindo flags F3
- T-INT-03 e T-INT-04 em `tests/integration/claude_2026_e2e_test.go`: compactação prompt-driven + governance hook bloqueando
- `execution_report.md` documenta "Memory Compaction Requested: true|false" + "Memory Workflow Bytes / Task Bytes" como métricas (via hook `memory_persist`)
- Defaults preservam comportamento atual: sessão sem `.specs/<prd>/memory/` e sem flags F3 roda idêntico a F2 (regressão hard)
- `runner.go` pode crescer >50 LoC — usar `object-calisthenics-go` para extração de helpers
</requirements>

## Subtarefas

- [ ] 6.1 Estender `Job` com 2 campos novos (`MemoryLimits`, `DisableHooks`)
- [ ] 6.2 Helper `injectMemoryContext(prompt, workflow, task) string` em `runner.go` (ou helper file)
- [ ] 6.3 Em `Run()`: instanciar memory store quando `j.TasksDir != ""`; ler workflow + task; injetar prompt + diretiva quando NeedsCompaction
- [ ] 6.4 Em `Run()`: instanciar hooks dispatcher; registrar hooks default se `!j.DisableHooks`
- [ ] 6.5 Em `Run()`: despachar `PointRuntimePreOpen` antes de `c.Open` (linha ~145)
- [ ] 6.6 Em `Run()`: despachar `PointPromptPreBuild` + `PointPromptPostBuild` ao redor da construção final do prompt
- [ ] 6.7 No loop de eventos: despachar `PointToolCallPreDispatch` antes de `persist.AppendEvent` (após normalize da task 3.0); `PointToolCallPostComplete` após persist quando evento é completion
- [ ] 6.8 Antes de `EnrichReport`: despachar `PointSessionPostEnd`
- [ ] 6.9 Implementar `hooks/memory_persist.go` completo: escrever `MEMORY.md` com Summary; emitir métricas via stdout/log para `execution_report.md`
- [ ] 6.10 Adicionar 5 flags em `task_loop.go` + propagação para `Job`
- [ ] 6.11 Adicionar T-16 em `task_loop_test.go`
- [ ] 6.12 Implementar T-INT-03 (compactação) + T-INT-04 (governance) em `tests/integration/claude_2026_e2e_test.go`
- [ ] 6.13 Refatorar `runner.go::Run()` se >400 LoC — extrair `prepareJob(j) *PreparedJob` ou similar (heurística OC)
- [ ] 6.14 Smoke manual: `mkdir -p .specs/<prd-teste>/memory && yes "linha" | head -200 > .specs/<prd-teste>/memory/MEMORY.md && ai-spec task-loop --tool claude --runtime acp .specs/<prd-teste>` — verificar `execution_report.md` cita "Memory Compaction Requested: true"

## Detalhes de Implementação

Ver `techspec.md` §"Design de Implementação" → "Assinaturas principais" para trechos cirúrgicos em `runner.go`. **Não duplicar aqui** — `execute-task` carrega techspec automaticamente.

Pontos críticos:
- **Conflito de merge com task 3.0**: ambas tocam `runner.go::Run()` em regiões similares. Esta task **depende** de 3.0 estar mergeada para evitar conflito. Coordenar com revisor.
- **`injectMemoryContext` é puro**: input prompt + workflow doc + task doc → string. Testar isoladamente (`runner_test.go` ganha T-MEM-INJECT-01).
- **Diretiva de compactação**: texto exato `"compact the flagged memory files before proceeding"` (paridade Compozy). Anexar como bloco final antes do prompt do usuário, não no meio.
- **`memory_persist.go` completo**: escrever `MEMORY.md` com formato:
  ```
  # Workflow Memory
  
  Keep only durable, cross-task context here. Do not duplicate facts that are obvious from the repository, PRD documents, or git history.
  
  ## Last Session Summary
  
  - Task: <task_file>
  - Exit Status: <status>
  - Events: <count>
  - Tool Calls: <count>
  ```
- **Hook order**: `governance` → `token_budget` em `PointRuntimePreOpen`/`PointPromptPostBuild`; `memory_persist` em `PointSessionPostEnd`. Registrar nessa ordem.
- **`--disable-hooks` desliga TUDO** — incluindo `memory_persist`. Documentar em help text.
- **Defaults zero-value para `MemoryLimits` significam usar defaults da constante** (não 0 efetivo) — implementar fallback `if limits.WorkflowLines == 0 { limits.WorkflowLines = memory.DefaultWorkflowLineLimit }` em `memory.New`.

## Critérios de Sucesso

- `make test` verde (cobertura global ≥ 70%)
- `make integration` verde (T-INT-03 e T-INT-04 inclusos)
- `make parity` reporta 31 invariantes verdes (sem regressão)
- T-INT-03: `MEMORY.md` com 151 linhas → `execution_report.md` cita "Memory Compaction Requested: true"
- T-INT-04: sem `AGENTS.md` em `j.WorkDir` → sessão aborta antes de `c.Open` com mensagem clara
- T-16 (novo caso `--disable-hooks --memory-workflow-limit-lines 100`) passa
- Smoke manual: `execution_report.md` ganhou seção com métricas de memória
- Regressão F2 (T-INT-01, T-INT-02) verde sem alteração
- `runner.go::Run()` continua legível (extrair helpers se necessário)

## Skills Necessárias

- `object-calisthenics-go` — `Run()` cresceu em 3.0 (~80 LoC) e ganha mais ~70 LoC nesta task. Tocando perto do limite de complexidade. Heurística OC orienta extração `prepareMemoryContext`, `dispatchPromptHooks`, `dispatchToolCallHooks` para manter função principal ≤100 LoC.

## Testes da Tarefa

- [ ] Testes unitários: T-MEM-INJECT-01 (helper puro) em `runner_test.go`; T-16 em `task_loop_test.go`
- [ ] Testes de integração: T-INT-03 (compactação E2E) + T-INT-04 (governance hook E2E) em `tests/integration/claude_2026_e2e_test.go`
- [ ] `memory_persist_test.go`: hook escreve `MEMORY.md` esperado
- [ ] Smoke manual documentado no `execution_report.md`
- [ ] Cobertura ≥ 70% global; ≥ 80% em `memory/` e `hooks/`

<critical>SEMPRE CRIAR E EXECUTAR TESTES DA TAREFA ANTES DE CONSIDERAR A TAREFA COMO `done`</critical>

## Arquivos Relevantes

- **Modificado** `internal/runtime/runner.go` (~80 LoC novas; refactor possível)
- **Modificado** `internal/runtime/hooks/memory_persist.go` (stub → completo)
- **Modificado** `cmd/ai_spec_harness/task_loop.go` (5 flags novas + propagação)
- **Modificado** `cmd/ai_spec_harness/task_loop_test.go` (+T-16)
- **Modificado** `internal/runtime/runner_test.go` (+T-MEM-INJECT-01)
- **Modificado** `tests/integration/claude_2026_e2e_test.go` (+T-INT-03, +T-INT-04)
- **Novo** `internal/runtime/hooks/memory_persist_test.go` (~70 LoC)
- **Leitor:** `internal/runtime/memory/store.go` (task 4.0)
- **Leitor:** `internal/runtime/hooks/dispatcher.go` (task 5.0)
- **Leitor:** `internal/runtime/hooks/governance.go`, `token_budget.go` (task 5.0)
- **Leitor:** `internal/metrics/` (reusar via `token_budget` hook)
