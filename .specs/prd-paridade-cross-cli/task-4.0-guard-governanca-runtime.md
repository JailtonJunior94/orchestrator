# Tarefa 4.0: Guard de governança em runtime (spec-hash/drift + PRD-first)

<critical>Ler prd.md e techspec.md desta pasta — sua tarefa será invalidada se você pular</critical>

## Visão Geral

Adicionar fail-fast de governança no `runtime.pre_open`, reusando `internal/specdrift`: abortar a sessão ACP antes de `c.Open` quando o PRD/techspec divergir do hash em `tasks.md` (RG-01) ou quando não houver PRD rastreável (RG-02). Idêntico para as 4 CLIs.

<requirements>
- `SpecDriftHook` em `internal/runtime/hooks`, registrado em `PointRuntimePreOpen` (ao lado de `GovernanceHook`).
- Usa `specdrift.CheckDrift(tasksDir)`; `report.Pass==false` ⇒ `ErrSpecDrift` (aborta antes de `c.Open`).
- `NoHashFound` (sem hash em `tasks.md`) ⇒ `ErrPRDUntracked` (PRD-first).
- No-op quando `Job.TasksDir==""` (uso ad-hoc/F1 preservado).
- `Job.SkipDriftGuard bool` (default false) desabilita só este hook; `--disable-hooks` continua desabilitando tudo.
- Suíte de invariantes ADR-008 (`internal/parity`) promovida a gate de CI por CLI fica na Tarefa 8.0 (aqui só o hook).
</requirements>

## Subtarefas

- [ ] 4.1 `hooks/spec_drift.go`: `SpecDriftHook`, `ErrSpecDrift`, `ErrPRDUntracked`.
- [ ] 4.2 Estender `RuntimePreOpenEvent`/registro para receber `TasksDir`; `Job.SkipDriftGuard` (`types.go`).
- [ ] 4.3 Registrar no `prepareHooksDispatcher` (`runner.go`) respeitando `SkipDriftGuard`/`DisableHooks`.
- [ ] 4.4 Testes: drift, `NoHashFound`, `TasksDir==""` (no-op), `SkipDriftGuard` (bypass).

## Detalhes de Implementação

Ver techspec.md §"Fail Fast" e §"Design de Implementação" (`SpecDriftHook`). ADR: [022](adr-022-guard-governanca-runtime-spec-hash.md). Reusar `specdrift.CheckDrift`/`CheckHash` (já testados) — não reimplementar hash.

## Critérios de Sucesso

- Sessão com hash divergente aborta com `ErrSpecDrift` antes de `c.Open` (dispatcher abort-on-first-error).
- `tasks.md` sem hash de PRD ⇒ `ErrPRDUntracked`.
- `Job.TasksDir==""` ⇒ hook no-op (sem regressão F1/uso interativo).
- `make test` verde; erros tipados (`errors.Is`) com mensagem acionável (PT-BR).

## Skills Necessárias

<!-- MANDATÓRIO: preenchido por `create-tasks` Etapa 4.1 via descoberta agnóstica em `.agents/skills/`.
     NÃO inclua aqui skills auto-carregadas em runtime: `agent-governance`, `execute-task`, `bugfix`,
     `review`, `refactor`, nem skills `*-implementation` (linguagem, inferida pelo diff).
     Use o conteúdo único `Nenhuma além das auto-carregadas (governance + linguagem).` se a tarefa
     não exigir skill processual extra. -->

Nenhuma além das auto-carregadas (governance + linguagem).

## Testes da Tarefa

- [ ] Testes unitários
- [ ] Testes de integração

<critical>SEMPRE CRIAR E EXECUTAR TESTES DA TAREFA ANTES DE CONSIDERAR A TAREFA COMO `done`</critical>

## Arquivos Relevantes
- `internal/runtime/hooks/spec_drift.go` (novo) + `spec_drift_test.go`
- `internal/runtime/hooks/dispatcher.go` (`RuntimePreOpenEvent`)
- `internal/runtime/types.go` (`Job.SkipDriftGuard`)
- `internal/runtime/runner.go` (`prepareHooksDispatcher`)
- `internal/specdrift/specdrift.go` (reuso)
