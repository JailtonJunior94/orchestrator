# Tarefa 4.0: ACP→Event Conversion

<critical>Ler prd.md e techspec.md desta pasta — sua tarefa será invalidada se você pular</critical>

## Visão Geral

Implementar a tradução de `acp.SessionUpdate` (tipo externo do `coder/acp-go-sdk`) para o tipo de domínio `events.Event` (entregue na task 1.0). Esta é a **única** camada autorizada a tocar tanto o SDK quanto o domínio (R-DDD-001: domínio não importa infraestrutura, mas a fronteira de tradução pode). Cobertura por testes table-driven com fixtures reais capturadas como JSON.

<requirements>
- `FromACPUpdate(driverID string, update acp.SessionUpdate) (events.Event, error)` exportado.
- Cobertura de todos os kinds documentados no PRD (RF-05): `agent_message`, `agent_thought`, `tool_call_start`, `tool_call_update`, `session_end`.
- Tipos não mapeados produzem `events.NewUnknown(rawKind, rawJSON)` sem erro — comportamento RF-05.
- Sem panic em qualquer input válido do SDK; payloads nil tratados defensivamente.
- Fixtures JSON em `internal/runtime/events/testdata/acp/` documentando o shape esperado por kind.
- Convert preserva o `raw` (JSON original) no `Event` para o envelope RF-08.
</requirements>

## Subtarefas

- [ ] 4.1 Criar `internal/runtime/events/convert.go` com `FromACPUpdate(driverID string, update acp.SessionUpdate) (events.Event, error)`.
- [ ] 4.2 Implementar despacho por tipo de update usando os campos do SDK (espelhar `internal/core/agent/acp_convert.go` do compozy/compozy): `UserMessageChunk`, `AgentMessageChunk`, `AgentThoughtChunk`, `ToolCall`, `ToolCallUpdate`, `SessionEnd`/`Status` final.
- [ ] 4.3 Para `ToolCall`/`ToolCallUpdate`: extrair `ToolCallId` para o VO `events.ToolCallID`; mapear `acp.ToolCallStatus` para o estado interno.
- [ ] 4.4 Implementar função auxiliar `convertContentBlocks(blocks []acp.ContentBlock) ([]events.ContentBlock, error)` se necessário; preferir manter blocks como `json.RawMessage` no payload para evitar acoplamento.
- [ ] 4.5 Capturar `update` como `json.RawMessage` (marshal do próprio update) para preencher `Event.raw` (alimenta RF-08).
- [ ] 4.6 Para kinds não reconhecidos: retornar `events.NewUnknown(rawKind, raw)` onde `rawKind` é descoberto via reflection mínima (`fmt.Sprintf("%T", update)`) ou campo discriminador do SDK; nunca retornar erro.
- [ ] 4.7 Criar fixtures JSON em `internal/runtime/events/testdata/acp/*.json` para cada kind suportado (8+ fixtures incluindo edge cases: tool_call sem content, agent_thought vazio, session_end com erro).
- [ ] 4.8 Criar `convert_test.go` com tabela parameterizada que: lê fixture, faz unmarshal para `acp.SessionUpdate`, chama `FromACPUpdate`, valida `Kind()`, `ToolCallID()`, e o envelope `MarshalJSON` contra golden file em `testdata/envelopes/`.

## Detalhes de Implementação

Ver `techspec.md`:
- §"Modelagem de Domínio" → "Application Service: ACPRunner" (regra: domínio não importa SDK; convert é a exceção autorizada)
- §"Design de Implementação" → "Tipo runtime.Event (tagged union)"
- §"Pontos de Integração" → "github.com/coder/acp-go-sdk"
- Compozy reference: `internal/core/agent/acp_convert.go` (padrão a espelhar; **não copiar literalmente**)

## Critérios de Sucesso

- `go test ./internal/runtime/events/... -run TestFromACP` ≥ 90% cobertura nas funções de convert.
- Todas as 8+ fixtures convertem sem erro.
- Fixture de kind desconhecido produz `Event{Kind: KindUnknown}` sem erro.
- Envelope `MarshalJSON` do evento convertido bate byte-a-byte com golden.
- Sem panic em fuzz curto (`go test -fuzz=FuzzFromACPUpdate -fuzztime=10s`).

## Skills Necessárias

<!-- MANDATÓRIO: preenchido por `create-tasks` Etapa 4.1 via descoberta agnóstica em `.agents/skills/`.
     NÃO inclua aqui skills auto-carregadas em runtime: `agent-governance`, `execute-task`, `bugfix`,
     `review`, `refactor`, nem skills `*-implementation` (linguagem, inferida pelo diff).
     Use o conteúdo único `Nenhuma além das auto-carregadas (governance + linguagem).` se a tarefa
     não exigir skill processual extra. -->

Nenhuma além das auto-carregadas (governance + linguagem).

## Testes da Tarefa

- [ ] Tabela `TestFromACPUpdate` cobrindo cada kind documentado no PRD + ao menos um kind desconhecido
- [ ] Fuzz target `FuzzFromACPUpdate(f *testing.F)` que injeta bytes arbitrários como `acp.SessionUpdate` e valida ausência de panic
- [ ] Golden test do envelope final por kind (consome `testdata/envelopes/` da task 1.0)
- [ ] Test cobrindo edge cases: `tool_call_id` vazio, payloads nil, status final indefinido

<critical>SEMPRE CRIAR E EXECUTAR TESTES DA TAREFA ANTES DE CONSIDERAR A TAREFA COMO `done`</critical>

## Arquivos Relevantes

- `internal/runtime/events/convert.go` + `convert_test.go` (novo)
- `internal/runtime/events/testdata/acp/agent_message.json` (novo)
- `internal/runtime/events/testdata/acp/agent_thought.json` (novo)
- `internal/runtime/events/testdata/acp/tool_call_start.json` (novo)
- `internal/runtime/events/testdata/acp/tool_call_update_in_progress.json` (novo)
- `internal/runtime/events/testdata/acp/tool_call_update_final.json` (novo)
- `internal/runtime/events/testdata/acp/session_end_success.json` (novo)
- `internal/runtime/events/testdata/acp/session_end_error.json` (novo)
- `internal/runtime/events/testdata/acp/unknown_drift.json` (novo)
- `go.mod` (sem alteração; SDK já adicionado em 3.0)

## Validações (DoD)

- [ ] `make verify` executado com sucesso e capturado em `evidence/task-4.0/execution_report.md`
- [ ] `go test ./internal/runtime/events/... -count=1 -race -cover` ≥ 90% no arquivo de convert
- [ ] `go test -fuzz=FuzzFromACPUpdate -fuzztime=10s ./internal/runtime/events/` sem panic e sem failing corpus persistido
- [ ] `golangci-lint run ./internal/runtime/events/...` sem violações
- [ ] Commit semântico `feat(runtime/events): translate acp.SessionUpdate to domain Event`
