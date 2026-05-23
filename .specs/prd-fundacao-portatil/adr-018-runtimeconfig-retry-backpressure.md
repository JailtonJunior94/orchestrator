# Registro de Decisão Arquitetural (ADR)

## Metadados

- **Título:** RuntimeConfig unificado, retry com backoff e sessão ACP com backpressure observável
- **Data:** 2026-05-22
- **Status:** Proposta
- **Decisores:** Jailton Junior (owner), arquitetura ai-spec-harness
- **Relacionados:** [PRD](prd.md) RF-02/03/04/05; [techspec](techspec.md); `internal/runtime/types.go` (`Job`); `internal/taskloop/taskloop.go` (`Options`); `internal/runtime/client/client.go`; `internal/runtime/runner.go`; referência Compozy `internal/core/model/runtime_config.go` + `internal/core/agent/session.go`

## Contexto

Os parâmetros operacionais hoje estão **dispersos**:
- `runtime.Job` (per-invocação) tem `ActivityTimeout`, mas **não** tem `MaxRetries`,
  `RetryBackoffMultiplier`, `Concurrent`, `BatchSize`.
- `taskloop.Options` (orquestração) tem `MaxIterations`, `MaxBugfixIterations`, `Timeout`, mas a
  execução do loop é **estritamente sequencial** (`internal/taskloop/runloop.go`) — sem concorrência
  nem batch.
- **Não há retry genérico com backoff**; só `DefaultMaxBugfixIterations = 3` (específico de bugfix,
  sem backoff).
- A sessão ACP (`acpClient`) usa canal bufferizado **cap 64** e `trySend` **não-bloqueante** que
  **descarta eventos silenciosamente** quando cheio (`client.go:278-288`), **sem contadores** e
  **sem timeout de publicação**.

O Compozy unifica isso num `model.RuntimeConfig` (Timeout/MaxRetries/RetryBackoffMultiplier/
Concurrent/BatchSize + `ExplicitRuntimeFlags`) e numa `Session` com backpressure (canal cap 1024,
timeout de publish 5s, contadores `SlowPublishes`/`DroppedUpdates`). O PRD exige paridade dessas
propriedades mantendo zero regressão F1.

## Decisão

1. **RuntimeConfig unificado (encapsular, não substituir):** introduzir `runtime.RuntimeConfig`
   agrupando os parâmetros operacionais — `Timeout` (mapeia para `ActivityTimeout`), `MaxRetries`,
   `RetryBackoffMultiplier`, `Concurrent`, `BatchSize`. `Job` **embute** `RuntimeConfig` (composição)
   em vez de uma reescrita; os campos F2–F5 (`MCPNested`, `NoNormalize`, `MemoryLimits`,
   `DisableHooks`, `TasksDir`, `AutoReview`) permanecem em `Job`. `taskloop.Options` passa a
   preencher `RuntimeConfig` ao montar o `Job`. Decisão tomada para honrar "menor mudança segura"
   (AGENTS.md) e a questão aberta do PRD.
2. **Retry com backoff exponencial (RF-04):** envolver a execução ACP (no `taskloop`/invoker, não
   dentro de `ACPRunner.Run`) num loop `attempt < MaxRetries` que reexecuta em **falha transitória**
   (erro de launcher/IO/timeout de inatividade), com espera `base * RetryBackoffMultiplier^attempt`.
   Falhas **não-transitórias** (ex.: `ErrPermissionDenied`) **não** são reexecutadas.
   `MaxRetries=0` (default) ⇒ comportamento atual (uma tentativa) — RF-05.
3. **Concorrência/batch (RF-02):** parametrizar o `runloop` com um pool limitado por `Concurrent`
   (semáforo) e `BatchSize`, **preservando ordering/dependências** entre tasks. Default
   `Concurrent=1`, `BatchSize=1` ⇒ execução sequencial idêntica à atual — RF-05.
4. **Sessão com backpressure observável (RF-03):** no `acpClient`, substituir o `trySend`
   non-blocking puro por publicação com **timeout de backpressure** configurável (default preserva o
   comportamento atual de drop imediato quando `timeout=0`) e **contadores atômicos**
   `slowPublishes`/`droppedUpdates`, expostos via métodos no `Client` e propagados ao `Summary`. A
   capacidade do canal passa a ser configurável (default mantém 64 para não alterar timing atual).

## Alternativas Consideradas

- **Substituir `Job` por um `RuntimeConfig` único (paridade literal Compozy):** mais limpo a longo
  prazo, mas reescreve a fronteira `taskloop↔runtime` e arrisca regressão ampla. Rejeitada nesta
  fase (overengineering vs. PRD).
- **Retry dentro de `ACPRunner.Run`:** acoplaria política de orquestração ao serviço de sessão e
  duplicaria persistência/eventos por tentativa. Rejeitada — retry fica na camada de orquestração.
- **Backpressure por canal ilimitado:** elimina drop mas arrisca memória ilimitada sob agente
  verboso. Rejeitada — buffer limitado + timeout + contadores é o equilíbrio.

## Consequências

### Benefícios Esperados
- Parâmetros operacionais coesos e previsíveis nas 4 CLIs — RF-02.
- Resiliência a falhas transitórias — RF-04; vazão configurável — RF-02.
- Observabilidade de perda de eventos (slow/dropped) — RF-03.
- Defaults preservam F1 (zero regressão) — RF-05.

### Trade-offs e Custos
- Concorrência introduz risco de não-determinismo de ordering; mitigado por default sequencial e
  por respeitar dependências declaradas.
- Retry pode mascarar falhas reais; mitigado por limitar a erros transitórios e logar tentativas.

### Riscos e Mitigações
- **Risco:** retry sobre efeito colateral não-idempotente (ex.: commit) → **Mitigação:** retry só na
  fase de sessão ACP antes de efeitos persistentes irreversíveis; documentar contrato.
- **Risco:** mudança de timing do canal alterar testes de evento → **Mitigação:** defaults
  (cap 64, timeout 0) preservam o comportamento; cobrir com teste de regressão de contagem.

## Plano de Implementação
1. Definir `runtime.RuntimeConfig` + composição em `Job`; `ApplyDefaults()` para zero-values.
2. Adicionar contadores + timeout de publish no `acpClient`; expor no `Client` e no `Summary`.
3. Adicionar loop de retry/backoff no invoker do taskloop (classificação transitória vs. fatal).
4. Adicionar pool por `Concurrent`/`BatchSize` no `runloop` respeitando dependências.
5. Testes table-driven cobrindo defaults (F1), retry transitório, drop counters, concorrência.

## Monitoramento e Validação
- Métrica/Summary: `slow_publishes`, `dropped_updates`, `retry_attempts`.
- Teste: `MaxRetries=0` e `Concurrent=1` ⇒ saída byte-equivalente ao atual (RF-05).
- Teste: erro transitório injetado ⇒ reexecuta até `MaxRetries` com backoff; `ErrPermissionDenied`
  ⇒ não reexecuta.

## Impacto em Documentação e Operação
- Documentar novas chaves de config (timeout/max_retries/retry_backoff/concurrent/batch_size) na
  hierarquia da [ADR-016](adr-016-config-hierarquico-universal.md).
- Telemetria opt-in (ADR-006) ganha os novos contadores.

## Revisão Futura
- Reavaliar unificação total `Job→RuntimeConfig` quando entrar o modelo de execução assíncrona
  (PRD futuro de daemon file-first).
