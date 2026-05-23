# Tarefa 5.0: Hooks dispatcher + 6 pontos canônicos + migração governance/token_budget Go hooks

<critical>Ler prd.md e techspec.md desta pasta — sua tarefa será invalidada se você pular</critical>

## Visão Geral

Implementar dispatcher de hooks in-process Go em `internal/runtime/hooks/dispatcher.go` com 6 pontos canônicos. Migrar **dois** shell hooks (`validate-governance.sh`, `validate-token-budget.sh`) para Go in-process, mantendo os shell hooks coexistentes para modo interativo Claude Code. **Sem integração com `runner.go`** — task 6.0 faz wiring.

<requirements>
- Criar `internal/runtime/hooks/dispatcher.go` com interface `Hook`, `Dispatcher`, `Event`
- Constantes exportadas para 6 pontos canônicos: `PointRuntimePreOpen`, `PointPromptPreBuild`, `PointPromptPostBuild`, `PointToolCallPreDispatch`, `PointToolCallPostComplete`, `PointSessionPostEnd` (+ futuro `PointSessionPostReview` em task 8.0)
- Tipos tipados de evento por ponto: `RuntimePreOpenEvent`, `PromptBuildEvent`, `ToolCallEvent`, `SessionPostEndEvent`
- `Dispatcher.Register(point, hook)` registra; `Dispatcher.Dispatch(ctx, point, evt)` fan-out **sequencial, abort-on-first-error**
- Criar `internal/runtime/hooks/governance.go` replicando `validate-governance.sh`: hook em `PointRuntimePreOpen` valida `AGENTS.md` em `j.WorkDir`; falha aborta sessão
- Criar `internal/runtime/hooks/token_budget.go` replicando `validate-token-budget.sh`: hook em `PointPromptPostBuild` valida budget via `internal/metrics.CheckBudget`; falha aborta sessão
- Criar `internal/runtime/hooks/memory_persist.go` (stub): hook em `PointSessionPostEnd` que escreverá `MEMORY.md`/`<task>.md` — implementação completa fica para task 6.0 (depende da integração com memory store)
- Shell hooks `.claude/hooks/*.sh` **não modificados** — coexistem
- Esta task **não integra com `runner.go`** — apenas entrega o subpacote standalone testável
- Cobertura ≥ 80% no subpacote `hooks/`
</requirements>

## Subtarefas

- [ ] 5.1 Definir interface `Hook`, `Dispatcher`, `Event` em `dispatcher.go`
- [ ] 5.2 Implementar `Dispatcher` concreto com `map[string][]Hook` interno + lock
- [ ] 5.3 Definir 6 constantes de pontos canônicos + tipos tipados de evento
- [ ] 5.4 Implementar `Dispatch` com fan-out sequencial e abort-on-first-error
- [ ] 5.5 Criar `governance.go`: lê `AGENTS.md` em `evt.WorkDir`; falha com erro tipado se ausente
- [ ] 5.6 Criar `token_budget.go`: chama `metrics.CheckBudget(...)` em `evt.Prompt`; falha com erro se excedido
- [ ] 5.7 Criar `memory_persist.go` stub (assinatura + TODO para task 6.0)
- [ ] 5.8 Escrever T-HOOK-01..T-HOOK-04 (ver techspec §"Abordagem de Testes")

## Detalhes de Implementação

Ver `techspec.md` §"Design de Implementação" → "F3-Claude" → `internal/runtime/hooks/dispatcher.go`. Stub completo no documento. **Não duplicar aqui** — `execute-task` carrega techspec automaticamente.

Pontos críticos:
- **Ordem de execução**: `slice ordered` (não `map`) por ponto — registration order = execution order. Crítico para previsibilidade.
- **`abort-on-first-error`**: primeiro hook que retornar erro interrompe; hooks subsequentes não rodam. Erro envelopado com `fmt.Errorf("hook %s in %s: %w", hook.Name(), point, err)`.
- **Thread safety**: `Register` e `Dispatch` podem ser concorrentes em teoria — `sync.RWMutex` na slice de hooks por ponto.
- **`governance.go` deve replicar EXATAMENTE** a semântica de `validate-governance.sh`: comentário no arquivo Go cita o shell equivalente para que contribuidores entendam a precedência.
- **`token_budget.go` reusa `internal/metrics`** — não duplicar lógica de cálculo de tokens. Apenas adaptar input/output ao formato do hook.
- **`memory_persist.go` é stub** — assinatura `func (h *MemoryPersistHook) Run(ctx, evt SessionPostEndEvent) error` + `return nil`. Implementação completa em task 6.0 quando memory store estiver wired ao runner.
- **Dispatcher é injetável** — `runner.go` recebe via construtor (task 6.0). Esta task não toca runner.

## Critérios de Sucesso

- T-HOOK-01..T-HOOK-04 verdes
- `go test ./internal/runtime/hooks/... -coverprofile=cov.out` reporta ≥ 80% de cobertura
- T-HOOK-01: ordem de registro respeitada na execução
- T-HOOK-02: erro em h2 → h3 nunca chamado
- T-HOOK-03: `Dispatch("nonexistent")` retorna nil sem panic
- T-HOOK-04: hook governance retorna erro claro quando `AGENTS.md` ausente em `t.TempDir()`
- `validate-governance.sh` e `validate-token-budget.sh` **inalterados** (verificado por `git diff .claude/hooks/`)
- Sem dependências externas novas em `go.mod`

## Skills Necessárias

Nenhuma além das auto-carregadas (governance + linguagem).

## Testes da Tarefa

- [ ] Testes unitários: T-HOOK-01..T-HOOK-04 em `internal/runtime/hooks/dispatcher_test.go`
- [ ] Teste específico `governance_test.go`: `t.TempDir()` sem AGENTS.md → erro; com AGENTS.md → ok
- [ ] Teste específico `token_budget_test.go`: prompt acima do budget → erro; abaixo → ok
- [ ] Cobertura ≥ 80% no subpacote
- [ ] `memory_persist.go` tem teste smoke (chamar Run; retornar nil)

<critical>SEMPRE CRIAR E EXECUTAR TESTES DA TAREFA ANTES DE CONSIDERAR A TAREFA COMO `done`</critical>

## Arquivos Relevantes

- **Novo** `internal/runtime/hooks/dispatcher.go` (~100 LoC)
- **Novo** `internal/runtime/hooks/governance.go` (~60 LoC)
- **Novo** `internal/runtime/hooks/token_budget.go` (~60 LoC)
- **Novo** `internal/runtime/hooks/memory_persist.go` (~30 LoC stub)
- **Novo** `internal/runtime/hooks/dispatcher_test.go` (~150 LoC)
- **Novo** `internal/runtime/hooks/governance_test.go` (~50 LoC)
- **Novo** `internal/runtime/hooks/token_budget_test.go` (~50 LoC)
- **Leitor:** `internal/metrics/` (reusar `CheckBudget`)
- **Referência (não modificar):** `.claude/hooks/validate-governance.sh`, `.claude/hooks/validate-token-budget.sh`
