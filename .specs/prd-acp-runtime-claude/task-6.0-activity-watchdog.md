# Tarefa 6.0: Activity Watchdog + Sentinels

<critical>Ler prd.md e techspec.md desta pasta — sua tarefa será invalidada se você pular</critical>

## Visão Geral

Implementar o `ActivityWatchdog` em `internal/runtime/watchdog.go` e os sentinels de erro do pacote `runtime`. O watchdog monitora o intervalo entre eventos recebidos e cancela o contexto (via `context.CancelCauseFunc`) quando esse intervalo excede `ActivityTimeout`. Inclui `Clock` interface para determinismo de testes, sem `time.Sleep` em testes.

<requirements>
- `ActivityWatchdog` recebe `ActivityTimeout`, `CancelCauseFunc`, `Clock`.
- `Touch()` atualiza atomicamente o timestamp do último evento.
- `Start(ctx)` lança goroutine com `time.Ticker` em intervalo `min(timeout/2, 5s)`; nunca menor que 1ms.
- `Stop()` é idempotente; encerra a goroutine sem leak (validar com `go test -race`).
- Quando `timeout > 0` e `(now - lastSeen) > timeout`, chama `cancel(ErrActivityTimeout)`.
- Quando `timeout == 0` (RF-07: 0 desabilita), `Start` é no-op.
- Testes usam clock fake; nenhum `time.Sleep` real.
</requirements>

## Subtarefas

- [ ] 6.1 Criar `internal/runtime/clock.go` com interface `Clock { Now() time.Time }` e impl default `realClock`.
- [ ] 6.2 Criar `internal/runtime/errors.go` com sentinels: `ErrLauncherUnavailable`, `ErrActivityTimeout`, `ErrPermissionDenied`, `ErrSessionAborted`, `ErrUnsupportedTool`, `ErrInvalidEvent`. Cada um com mensagem em lowercase, estável.
- [ ] 6.3 Criar `internal/runtime/watchdog.go` com struct `ActivityWatchdog { timeout events.ActivityTimeout; cancel context.CancelCauseFunc; lastSeen atomic.Int64; clock Clock; stopCh chan struct{}; stopOnce sync.Once }`.
- [ ] 6.4 Implementar `NewActivityWatchdog(timeout events.ActivityTimeout, cancel context.CancelCauseFunc, clock Clock) *ActivityWatchdog`.
- [ ] 6.5 Implementar `(w *ActivityWatchdog) Touch()` usando `atomic.Int64.Store(clock.Now().UnixNano())`.
- [ ] 6.6 Implementar `(w *ActivityWatchdog) Start(ctx context.Context)`: se `timeout.Disabled()`, retorna imediatamente; senão lança goroutine com `time.Ticker` em `min(timeout/2, 5s)` que verifica se `clock.Now() - lastSeen > timeout` e chama `cancel(ErrActivityTimeout)`; encerra em `<-ctx.Done()` ou `<-w.stopCh`.
- [ ] 6.7 Implementar `(w *ActivityWatchdog) Stop()` que fecha `stopCh` via `sync.Once`.
- [ ] 6.8 Criar `watchdog_test.go` com clock fake (interface implementada em test); validar: (a) timeout=0 desabilita; (b) Touch periódico mantém alive; (c) sem Touch além do timeout dispara `cancel(ErrActivityTimeout)`; (d) Stop é idempotente; (e) `Run` com `-race` sem leak de goroutine.

## Detalhes de Implementação

Ver `techspec.md`:
- §"Design de Implementação" → "Watchdog (RF-06)"
- §"Estratégia de Erros" → "Sentinelas"
- PRD RF-06 (cancelamento via watchdog) e RF-07 (0 desabilita)

Padrão de extração de razão no caller (já documentado, implementado em 9.0):
```go
switch cause := context.Cause(ctx); {
case errors.Is(cause, ErrActivityTimeout): summary.CancelReason = events.CancelReasonActivityTimeout
...
}
```

## Critérios de Sucesso

- `go test ./internal/runtime/... -race` passa sem leak nem race condition.
- Cobertura do watchdog ≥ 90%.
- Testes determinísticos (zero `time.Sleep`).
- Sem dependência de `coder/acp-go-sdk` (apenas stdlib + `internal/runtime/events`).

## Skills Necessárias

<!-- MANDATÓRIO: preenchido por `create-tasks` Etapa 4.1 via descoberta agnóstica em `.agents/skills/`.
     NÃO inclua aqui skills auto-carregadas em runtime: `agent-governance`, `execute-task`, `bugfix`,
     `review`, `refactor`, nem skills `*-implementation` (linguagem, inferida pelo diff).
     Use o conteúdo único `Nenhuma além das auto-carregadas (governance + linguagem).` se a tarefa
     não exigir skill processual extra. -->

Nenhuma além das auto-carregadas (governance + linguagem).

## Testes da Tarefa

- [ ] `TestActivityWatchdog_Disabled`: timeout=0 → `Start` no-op, `cancel` nunca chamado
- [ ] `TestActivityWatchdog_KeptAlive`: Touch cada N < timeout, `cancel` nunca chamado
- [ ] `TestActivityWatchdog_Fires`: sem Touch além do timeout, `cancel(ErrActivityTimeout)` chamado uma vez
- [ ] `TestActivityWatchdog_StopIdempotent`: `Stop()` chamado N vezes sem panic
- [ ] `TestActivityWatchdog_NoGoroutineLeak`: usar `goleak` ou contagem manual antes/depois com `runtime.NumGoroutine`
- [ ] `TestSentinels`: cada `errors.Is(wrapped, sentinel)` retorna true

<critical>SEMPRE CRIAR E EXECUTAR TESTES DA TAREFA ANTES DE CONSIDERAR A TAREFA COMO `done`</critical>

## Arquivos Relevantes

- `internal/runtime/clock.go` (novo)
- `internal/runtime/errors.go` (novo)
- `internal/runtime/watchdog.go` + `watchdog_test.go` (novo)

## Validações (DoD)

- [ ] `make verify` executado com sucesso e capturado em `evidence/task-6.0/execution_report.md`
- [ ] `go test ./internal/runtime/... -count=1 -race -cover` ≥ 90% no watchdog
- [ ] `golangci-lint run ./internal/runtime/...` sem violações
- [ ] `grep -r "time.Sleep" internal/runtime/*_test.go` retorna vazio
- [ ] Commit semântico `feat(runtime): add activity watchdog and error sentinels`
