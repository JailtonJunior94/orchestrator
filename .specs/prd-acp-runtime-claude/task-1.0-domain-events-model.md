# Tarefa 1.0: Domain Events Model

<critical>Ler prd.md e techspec.md desta pasta — sua tarefa será invalidada se você pular</critical>

## Visão Geral

Implementar o núcleo de domínio do novo pacote `internal/runtime/events/`: tipo `Event` como tagged union (ADR-010), value objects imutáveis (`EventKind`, `ToolCallID`, `CancelReason`, `ActivityTimeout`), payloads tipados por kind, state pattern em `SessionState` com transições explícitas e coleção de primeira classe `ToolCallCounters`. Nesta task **não há dependência do `coder/acp-go-sdk`** — apenas tipos puros + `encoding/json` da stdlib.

<requirements>
- Domínio sem importação de `coder/acp-go-sdk` (proibido nesta task).
- `Event` exportado via construtores validadores (`events.NewAgentMessage(...)` etc.) — proibido struct literal fora de testes (R-DDD-001).
- `MarshalJSON` produz o envelope canônico do RF-08: `{ts, kind, tool_call_id, launcher, raw}`.
- VOs autovalidados em construtor (`NewToolCallID`, `ParseEventKind`, `NewActivityTimeout`, `ParseCancelReason`).
- `SessionState` com mapa `validTransitions` e método `Transition(next)` que retorna erro tipado em transições inválidas.
- `ToolCallCounters` encapsula `map[ToolCallID]record`; expõe `Record(Event)`, `ToolCalls() []ToolCallSummary` (ordenado), `MappedCount() int`, `UnknownKinds() []string`.
- 100% dos exports cobertos por testes unitários table-driven; golden files para `MarshalJSON`.
</requirements>

## Subtarefas

- [ ] 1.1 Criar `internal/runtime/events/kinds.go`: type `EventKind string`; constantes `KindAgentMessage`, `KindAgentThought`, `KindToolCallStart`, `KindToolCallUpdate`, `KindSessionEnd`, `KindRuntimeInit`, `KindUnknown`; função `ParseEventKind(s string) (EventKind, error)` validando enum fechado.
- [ ] 1.2 Criar `internal/runtime/events/toolcallid.go`: VO `ToolCallID{ value string }`; `NewToolCallID(s string) ToolCallID`; `(t ToolCallID) IsZero() bool`; `(t ToolCallID) String() string`; `MarshalJSON` que produz `null` quando zero.
- [ ] 1.3 Criar `internal/runtime/events/cancel_reason.go`: type `CancelReason string`; constantes `CancelReasonNone`, `CancelReasonActivityTimeout`, `CancelReasonContextCanceled`, `CancelReasonToolError`, `CancelReasonPermissionDenied`; `ParseCancelReason(s string) (CancelReason, error)`.
- [ ] 1.4 Criar `internal/runtime/events/timeout.go`: VO `ActivityTimeout time.Duration`; `NewActivityTimeout(d time.Duration) (ActivityTimeout, error)` exigindo `d >= 0`; `(t ActivityTimeout) Disabled() bool`.
- [ ] 1.5 Criar `internal/runtime/events/payloads.go`: structs privados `agentMessagePayload`, `agentThoughtPayload`, `toolCallStartPayload`, `toolCallUpdatePayload`, `sessionEndPayload`, `runtimeInitPayload`, `unknownPayload`; getters de intenção (não-mecânicos) — ex.: `(p *toolCallStartPayload) Name() string`, `(p *toolCallUpdatePayload) Final() bool`.
- [ ] 1.6 Criar `internal/runtime/events/event.go`: struct `Event` com campos privados (ts, kind, toolCallID, launcher, payloads opcionais, raw); construtores `NewAgentMessage`, `NewAgentThought`, `NewToolCallStart`, `NewToolCallUpdate`, `NewSessionEnd`, `NewRuntimeInit`, `NewUnknown` validando invariantes; métodos de acesso `Kind()`, `Timestamp()`, `ToolCallID()`, `Launcher()`, `AgentMessage()`, etc. (retornam ponteiro ou nil sem panic); `IsTerminal()` true quando `kind == KindSessionEnd`.
- [ ] 1.7 Implementar `(e Event) MarshalJSON() ([]byte, error)` produzindo o envelope RF-08 com `ts` em RFC3339Nano, `tool_call_id` como string ou `null`, `launcher` como `"binary"` / `"npx"`, `raw` como `json.RawMessage` cru.
- [ ] 1.8 Criar `internal/runtime/events/state.go`: `SessionState string`; constantes `StateInit`, `StateRunning`, `StateAwaiting`, `StateClosed`, `StateCanceled`; mapa `validTransitions`; método `(s SessionState) Transition(next SessionState) (SessionState, error)` retornando `ErrInvalidTransition` quando proibido.
- [ ] 1.9 Criar `internal/runtime/events/counters.go`: type `ToolCallCounters`; `NewToolCallCounters() *ToolCallCounters`; `Record(Event)` indexa por `ToolCallID`; `ToolCalls() []ToolCallSummary` ordenado por timestamp de início; `MappedCount() int`; `UnknownKinds() []string` deduplica e ordena.
- [ ] 1.10 Adicionar erro sentinela `ErrInvalidTransition`, `ErrInvalidEvent` em `internal/runtime/events/errors.go`.
- [ ] 1.11 Criar testes table-driven para cada VO + construtor + transição + counter; golden files em `internal/runtime/events/testdata/envelopes/*.json` para validar `MarshalJSON`.

## Detalhes de Implementação

Ver `techspec.md`:
- §"Modelagem de Domínio (DDD)" → "Value Objects", "State Pattern", "Coleções de Primeira Classe"
- §"Design de Implementação" → "Tipo runtime.Event (tagged union — ADR-010)", "Modelos de Dados — Envelope events.jsonl"
- `adr-010-event-tagged-union.md` para a decisão de desenho

## Critérios de Sucesso

- `go test ./internal/runtime/events/...` passa com cobertura ≥ 90% no pacote.
- `go vet ./internal/runtime/events/...` sem warnings.
- `golangci-lint run ./internal/runtime/events/...` limpo.
- Nenhum import de `github.com/coder/acp-go-sdk` no pacote (verificável por `grep -r "coder/acp-go-sdk" internal/runtime/events/` retornar vazio).
- Construtores rejeitam input inválido com erro tipado (não panic).
- Golden test do envelope `Event.MarshalJSON` bate byte-a-byte com fixture esperada.

## Skills Necessárias

<!-- MANDATÓRIO: preenchido por `create-tasks` Etapa 4.1 via descoberta agnóstica em `.agents/skills/`.
     NÃO inclua aqui skills auto-carregadas em runtime: `agent-governance`, `execute-task`, `bugfix`,
     `review`, `refactor`, nem skills `*-implementation` (linguagem, inferida pelo diff).
     Use o conteúdo único `Nenhuma além das auto-carregadas (governance + linguagem).` se a tarefa
     não exigir skill processual extra. -->

Nenhuma além das auto-carregadas (governance + linguagem).

## Testes da Tarefa

- [ ] Testes unitários table-driven para cada VO (`kinds_test.go`, `toolcallid_test.go`, `cancel_reason_test.go`, `timeout_test.go`)
- [ ] Testes table-driven para `SessionState.Transition` cobrindo cada par (válido + inválido)
- [ ] Testes para `ToolCallCounters` cobrindo: zero events, vários tool calls, evento unknown
- [ ] Golden test para `Event.MarshalJSON` por kind (8 fixtures)
- [ ] Testes de cada construtor `New*` cobrindo input válido e inválido

<critical>SEMPRE CRIAR E EXECUTAR TESTES DA TAREFA ANTES DE CONSIDERAR A TAREFA COMO `done`</critical>

## Arquivos Relevantes

- `internal/runtime/events/kinds.go` + `kinds_test.go` (novo)
- `internal/runtime/events/toolcallid.go` + `toolcallid_test.go` (novo)
- `internal/runtime/events/cancel_reason.go` + `cancel_reason_test.go` (novo)
- `internal/runtime/events/timeout.go` + `timeout_test.go` (novo)
- `internal/runtime/events/payloads.go` (novo)
- `internal/runtime/events/event.go` + `event_test.go` (novo)
- `internal/runtime/events/state.go` + `state_test.go` (novo)
- `internal/runtime/events/counters.go` + `counters_test.go` (novo)
- `internal/runtime/events/errors.go` (novo)
- `internal/runtime/events/testdata/envelopes/*.json` (novo, golden fixtures)

## Validações (DoD)

- [ ] `make verify` executado com sucesso e capturado em `evidence/task-1.0/execution_report.md`
- [ ] `go test ./internal/runtime/events/... -count=1 -race -cover` ≥ 90% cobertura
- [ ] `golangci-lint run ./internal/runtime/events/...` sem violações
- [ ] `grep -r "coder/acp-go-sdk" internal/runtime/events/` retorna vazio
- [ ] Commit semântico `feat(runtime/events): add domain events model with VOs and state pattern`
- [ ] Branch local `feat/acp-runtime-claude` com este commit empurrado para origin
