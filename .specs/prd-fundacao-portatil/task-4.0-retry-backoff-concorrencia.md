# Tarefa 4.0: Retry/backoff + concorrência/batch na orquestração

<critical>Ler prd.md e techspec.md desta pasta — sua tarefa será invalidada se você pular</critical>

## Visão Geral

Adicionar, na camada de orquestração (não dentro de `ACPRunner.Run`), retry com backoff exponencial
para falhas transitórias e execução concorrente limitada por `Concurrent`/`BatchSize`, consumindo o
`RuntimeConfig` da Tarefa 3.0. Conforme [ADR-018](adr-018-runtimeconfig-retry-backpressure.md).

<requirements>
- `RetryClassifier.IsTransient(err)`: erros de launcher/IO/inatividade são transitórios;
  `client.ErrPermissionDenied` (e equivalentes fatais) NÃO são reexecutados.
- Loop de retry no invoker (`internal/taskloop/acpinvoker.go`): `attempt < MaxRetries`, espera
  `base * RetryBackoffMultiplier^attempt`; `MaxRetries=0` (default) ⇒ uma tentativa (F1).
- Retry restrito à fase de sessão ACP, antes de efeitos persistentes irreversíveis (idempotência).
- Pool limitado por `Concurrent` (semáforo) e `BatchSize` no `runloop`, **respeitando dependências/ordering** de tasks; `Concurrent=1`,`BatchSize=1` (default) ⇒ sequencial idêntico ao atual.
- `Summary`/telemetria registram `retry_attempts`.
- Regressão zero (RF-05): defaults preservam execução sequencial e contagem atuais.
</requirements>

## Subtarefas

- [ ] 4.1 Criar `RetryClassifier` (transitório vs fatal) — `internal/runtime/retry.go` ou em `taskloop`.
- [ ] 4.2 Envolver a invocação ACP no `acpinvoker` com loop de retry/backoff respeitando `MaxRetries`/multiplier.
- [ ] 4.3 Adicionar pool por `Concurrent`/`BatchSize` no `runloop`, preservando dependências entre tasks.
- [ ] 4.4 Propagar `retry_attempts` ao `Summary`/telemetria.
- [ ] 4.5 Testes: erro transitório injetado reexecuta até `MaxRetries`; `ErrPermissionDenied` não reexecuta; concorrência respeita deps; defaults sequenciais.

## Detalhes de Implementação

Ver `techspec.md` §"Interfaces Chave" (`RetryClassifier`) e §"Sequenciamento" +
[ADR-018](adr-018-runtimeconfig-retry-backpressure.md). Tocar `internal/taskloop/acpinvoker.go`,
`internal/taskloop/runloop.go`, `internal/taskloop/taskloop.go` e (novo) `internal/runtime/retry.go`.
A execução atual é estritamente sequencial (`runloop.go`); o pool é aditivo e default-off.

## Critérios de Sucesso

- Erro transitório → reexecuta até `MaxRetries` com espera crescente; fatal → falha imediata.
- `Concurrent>1` executa tasks independentes em paralelo sem violar dependências declaradas.
- Defaults (`MaxRetries=0`, `Concurrent=1`, `BatchSize=1`) ⇒ comportamento byte-equivalente ao atual (RF-05).
- `make test`/`make lint` verdes (incl. `-race` para o pool); cobertura ≥ 75% nos pacotes alterados.

## Skills Necessárias

<!-- MANDATÓRIO: preenchido por `create-tasks` Etapa 4.1 via descoberta agnóstica em `.agents/skills/`.
     NÃO inclua aqui skills auto-carregadas em runtime: `agent-governance`, `execute-task`, `bugfix`,
     `review`, `refactor`, nem skills `*-implementation` (linguagem, inferida pelo diff).
     Use o conteúdo único `Nenhuma além das auto-carregadas (governance + linguagem).` se a tarefa
     não exigir skill processual extra. -->

Nenhuma além das auto-carregadas (governance + linguagem).

## Testes da Tarefa

- [ ] Testes unitários: classificação transitório/fatal, retry com backoff, concorrência respeitando deps, defaults sequenciais (`-race`).
- [ ] Testes de integração: não obrigatórios (invoker testável com runner fake/clock injetável).

<critical>SEMPRE CRIAR E EXECUTAR TESTES DA TAREFA ANTES DE CONSIDERAR A TAREFA COMO `done`</critical>

## Arquivos Relevantes
- `internal/runtime/retry.go` (novo) ou `internal/taskloop/retry.go`
- `internal/taskloop/acpinvoker.go`
- `internal/taskloop/runloop.go`
- `internal/taskloop/taskloop.go`
- testes correspondentes (`*_test.go`)
