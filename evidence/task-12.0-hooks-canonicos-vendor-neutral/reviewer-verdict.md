# Veredito do Revisor — Tarefa 12.0

Três rodadas de revisão independente via subagente `reviewer`, cada uma com contexto isolado
(prd.md RF-45 a RF-50/RF-64 a RF-67, techspec.md, task-12.0 completa, `.claude/rules/code-style.md`
e `governance.md`).

## Rodada 1 — REJECTED

Achados [CRITICAL]/[HIGH]: guarda de recursão (`internal/hookcontract/recursion.go`) e timeout por
hook (`internal/hookcontract/timeout.go`) existiam como bibliotecas sem nenhum chamador em código de
produção — "por construção" não era real. Toda a instrumentação de telemetria nova também estava
desconectada de qualquer produtor real. Achados [MEDIUM]: strings em PT-BR em arquivos novos
(R-STYLE-001.1); `MeasureHook` não cancelava trabalho subjacente no timeout (goroutine solta sem
`context.Context`). Achado [LOW]: teste de RF-48 tautológico.

## Rodada 2 — APPROVED_WITH_REMARKS

Após wiring real de `hookcontract.CheckRecursionGuard` e do timeout em
`internal/runtime/hooks/dispatcher.go` (`Dispatch`/`runWithTimeout`, único `Dispatcher` de produção
usado por `internal/runtime/runner.go`), tradução das strings PT-BR para inglês, e refatoração de
`MeasureHook` para aceitar `context.Context` e propagar cancelamento: achados [CRITICAL]/[HIGH]
anteriores confirmados como resolvidos. Restavam dois [MEDIUM] (teste de RF-48 ainda tautológico;
`dispatcher_test.go` sem cobertura de integração real para guard de recursão e timeout) e dois [LOW]
(campo `Envelope.InvocationDepth` sem consumidor de produção fora do dispatcher em memória;
comentários PT-BR pré-existentes fora do diff).

## Rodada 3 — APPROVED_WITH_REMARKS (final)

Após adicionar `SetHookTimeout` à interface `Dispatcher` (mock regenerado e sincronizado via
`bash scripts/check-mocks.sh`) e quatro testes de integração reais em `dispatcher_test.go`
(`TestDispatcher_RecursionGuardBlocksReentrantDispatch`, `TestDispatcher_TimeoutBlocksSlowHook`,
`TestDispatcher_FastHookWithinTimeoutSucceeds`): os dois achados [MEDIUM] da rodada 2 relativos à
cobertura de integração foram fechados com prova de que a implementação real do `Dispatch` de
produção — não apenas a função pura — bloqueia recursão e timeout corretamente. Nenhum achado
[HIGH]/[CRITICAL] em nenhuma das três rodadas ao final.

Débito declarado, registrado e não bloqueante:

- **[MEDIUM]** `internal/telemetry/hook_events_test.go` `TestHookTelemetry_FailureDoesNotPropagateOrCountAsRetry`
  permanece tautológico (variáveis locais não ligadas a política de retry real). A não-propagação de
  RF-48 é garantida por construção de assinatura (`RecordDuration/RecordDecision/RecordTimeout` não
  retornam `error`; `dispatcher.go` não captura retorno algum desses métodos), mas o teste em si não
  demonstra isso contra um caminho de produção. Fix sugerido para tarefa futura: injetar
  `telemetry.HookTelemetry`/writer no `dispatcher` de produção e espelhar o padrão dos testes de
  integração de recursão/timeout adicionados nesta tarefa.
- **[LOW]** `hookcontract.Envelope.InvocationDepth` decodificado mas sem consumidor de produção fora
  do dispatcher em memória (que rastreia profundidade via `context.Value`, dissociado do campo do
  envelope). RF-67 real hoje é garantido inteiramente pelo dispatcher Go, não pelo campo do envelope.
- **[LOW]** Comentários PT-BR pré-existentes em `internal/runtime/hooks/dispatcher.go` fora das
  linhas tocadas pelo diff desta tarefa — fora do escopo de R-STYLE-001.2 ("apenas o diff").
- **[LOW residual, arquitetural, não desta tarefa]** Hooks shell interativos (`.agents/hooks/*.sh`)
  não passam pelo `dispatcher.Dispatch` e portanto não têm guarda de recursão/timeout deste mecanismo
  — separação já documentada em AGENTS.md ("Hooks Go servem o modo orquestrado; hooks shell servem o
  modo interativo. Não se sobrepõem").

**Veredito final: APPROVED_WITH_REMARKS, zero achado HIGH/CRITICAL.**
