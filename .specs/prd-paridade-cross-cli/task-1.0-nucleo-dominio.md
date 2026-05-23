# Tarefa 1.0: Núcleo de domínio (Value Objects sem IO)

<critical>Ler prd.md e techspec.md desta pasta — sua tarefa será invalidada se você pular</critical>

## Visão Geral

Criar o núcleo de domínio puro (sem IO) que habilita todas as demais tarefas: os Value Objects `DriverID`, `ContextWindow`/`WindowClass` e `MetricSet`. Esta tarefa não altera comportamento de runtime — só introduz os tipos e suas invariantes.

<requirements>
- `DriverID` VO em `internal/runtime/specs`: `ParseDriverID(string) (DriverID, error)` validando o conjunto canônico (`claude`, `codex`, `copilot`, `gemini`); `ErrUnknownDriver` para valor fora do conjunto; imutável; `String()`.
- `ContextWindow` VO + `WindowClass` (`WindowStandard` | `WindowLarge`) em `internal/runtime/specs`: `Class()` deriva da janela; zero-value (`MaxTokens==0`) ⇒ `WindowStandard` (F1).
- `MetricSet` VO em `internal/runtime/events`: contadores canônicos (`totalTokens`, `cacheReadTokens`, `thinkingTokens`) + `extra map[string]int`; `Merge(other) MetricSet`; `IsZero() bool`; `Fields()` retorna só campos não-zero, ordenados.
- Campos não exportados (R-DDD-001 §Entidades); VOs imutáveis e autovalidados.
</requirements>

## Subtarefas

- [ ] 1.1 `specs/driver.go`: `DriverID`, `ParseDriverID`, `ErrUnknownDriver`, `String()`.
- [ ] 1.2 `specs/window.go`: `ContextWindow`, `WindowClass` (enum tipado), `Class()`.
- [ ] 1.3 `events/metricset.go`: `MetricSet`, `Merge`, `IsZero`, `Fields` (+ tipo `MetricField`).
- [ ] 1.4 Testes table-driven para os 3 VOs (válido/inválido, zero-value/F1, soma associativa).

## Detalhes de Implementação

Ver techspec.md §"Modelagem de Domínio" (tabela de VOs + snippets de `DriverID` e `MetricSet`) e §"Object Calisthenics" (regras #3 e #4). ADRs: [020](adr-020-driverid-vo-normalizacao-paridade.md), [021](adr-021-metricset-vo-extractor-por-driver.md), [023](adr-023-window-policy-cli-aware.md).

## Critérios de Sucesso

- `ParseDriverID` rejeita driver desconhecido com `ErrUnknownDriver` (tipado, `errors.Is`).
- `MetricSet.IsZero()` true para zero-value; `Merge` soma campos canônicos e `extra`; `Fields()` omite zeros.
- `ContextWindow{}.Class() == WindowStandard` (preserva F1).
- `make test` e `make vet` verdes; sem dependências novas no `go.mod`.

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
- `internal/runtime/specs/driver.go` (novo) + `driver_test.go`
- `internal/runtime/specs/window.go` (novo) + `window_test.go`
- `internal/runtime/events/metricset.go` (novo) + `metricset_test.go`
