# Tarefa 3.0: RuntimeConfig unificado + sessão ACP observável (backpressure)

<critical>Ler prd.md e techspec.md desta pasta — sua tarefa será invalidada se você pular</critical>

## Visão Geral

Unificar os parâmetros operacionais num `runtime.RuntimeConfig` embutido em `Job` (composição, não
reescrita) e tornar a sessão ACP observável sob backpressure: timeout de publicação configurável e
contadores atômicos `slowPublishes`/`droppedUpdates` no `acpClient`, propagados ao `Summary`.
Conforme [ADR-018](adr-018-runtimeconfig-retry-backpressure.md). Concorrência/batch e retry ficam na
Tarefa 4.0.

<requirements>
- `runtime.RuntimeConfig` com `Timeout` (mapeia `ActivityTimeout`), `MaxRetries`,
  `RetryBackoffMultiplier`, `Concurrent`, `BatchSize` + `ApplyDefaults()` (zero-value => F1).
- `Job` embute `RuntimeConfig` (composição); campos F2–F5 preservados; `taskloop.Options` passa a preencher `RuntimeConfig`.
- `Client` ganha `SlowPublishes() uint64` e `DroppedUpdates() uint64`.
- `acpClient`: publicação com timeout de backpressure (default 0 = drop imediato atual) + contadores; capacidade de canal configurável (default 64).
- `Summary` carrega os contadores; telemetria opt-in (ADR-006) os inclui.
- Regressão zero (RF-05): defaults (cap 64, timeout 0, MaxRetries/Concurrent/BatchSize inertes) preservam timing e contagem de eventos atuais.
</requirements>

## Subtarefas

- [ ] 3.1 Definir `runtime.RuntimeConfig` + `ApplyDefaults()`; embutir em `Job` com mapeamento de `ActivityTimeout`.
- [ ] 3.2 Adicionar contadores atômicos + timeout de publish em `acpClient.trySend`/`Open`.
- [ ] 3.3 Expandir a interface `Client` com `SlowPublishes()`/`DroppedUpdates()` e implementar no `acpClient`.
- [ ] 3.4 Propagar contadores ao `Summary` (`runner.go` `buildSummary`) e à telemetria.
- [ ] 3.5 Ajustar `taskloop` para popular `RuntimeConfig` no `Job` (sem mudar comportamento default).
- [ ] 3.6 Testes: defaults inertes (contagem de eventos igual), drop com timeout=0, slow-publish com timeout>0.

## Detalhes de Implementação

Ver `techspec.md` §"Modelos de Dados" (`RuntimeConfig`, `Job` composto) e §"Interfaces Chave"
(`Client` estendido) + [ADR-018](adr-018-runtimeconfig-retry-backpressure.md). Tocar
`internal/runtime/types.go`, `internal/runtime/client/client.go`, `internal/runtime/runner.go`,
`internal/taskloop/taskloop.go`. Depende das chaves de config da Tarefa 1.0.

## Critérios de Sucesso

- Com defaults, contagem de eventos e timing byte-equivalentes ao atual (teste de regressão).
- Canal cheio: `timeout=0` → drop + `droppedUpdates++`; `timeout>0` → aguarda e `slowPublishes++`.
- `Summary` e telemetria expõem `slow_publishes`/`dropped_updates`.
- `make test`/`make lint` verdes; cobertura ≥ 75% nos pacotes alterados.

## Skills Necessárias

<!-- MANDATÓRIO: preenchido por `create-tasks` Etapa 4.1 via descoberta agnóstica em `.agents/skills/`.
     NÃO inclua aqui skills auto-carregadas em runtime: `agent-governance`, `execute-task`, `bugfix`,
     `review`, `refactor`, nem skills `*-implementation` (linguagem, inferida pelo diff).
     Use o conteúdo único `Nenhuma além das auto-carregadas (governance + linguagem).` se a tarefa
     não exigir skill processual extra. -->

Nenhuma além das auto-carregadas (governance + linguagem).

## Testes da Tarefa

- [ ] Testes unitários: `ApplyDefaults`, mapeamento `Timeout`↔`ActivityTimeout`, backpressure (drop/slow), regressão de contagem.
- [ ] Testes de integração: não obrigatórios (cliente testável via IOProvider/in-process).

<critical>SEMPRE CRIAR E EXECUTAR TESTES DA TAREFA ANTES DE CONSIDERAR A TAREFA COMO `done`</critical>

## Arquivos Relevantes
- `internal/runtime/types.go` (`RuntimeConfig`, `Job` composto)
- `internal/runtime/client/client.go` (contadores, timeout, interface `Client`)
- `internal/runtime/runner.go` (`buildSummary`, `Summary`)
- `internal/runtime/client/client_test.go`, `internal/runtime/runner_test.go`
- `internal/taskloop/taskloop.go` (popular `RuntimeConfig`)
