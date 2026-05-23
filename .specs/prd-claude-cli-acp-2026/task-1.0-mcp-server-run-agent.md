# Tarefa 1.0: MCP server reservado com tool `run_agent` (skeleton + spawn engine)

<critical>Ler prd.md e techspec.md desta pasta — sua tarefa será invalidada se você pular</critical>

## Visão Geral

Implementar servidor MCP stdio interno expondo **uma única tool reservada** `run_agent(agent_name, prompt, model?, timeout?)`. Resolução de `agent_name` via `internal/agents/registry.go` (ADR-011). Spawn de child `ACPRunner` no mesmo processo com contexto serializado via env var `AISPEC_RUN_AGENT_CONTEXT`. Profundidade máxima 3 (paridade Compozy `internal/core/agents/mcpserver/server.go`).

<requirements>
- Criar `internal/runtime/mcpserver/server.go` com interface `Server`, `RunAgentInput`, `RunAgentOutput`, `New(registry, maxDepth, persistFactory)`
- Criar `internal/runtime/mcpserver/engine.go` com spawn do child `ACPRunner` reusando `internal/runtime/runner.go::ACPRunner`
- Decisão MCP-SDK Go: avaliar `github.com/modelcontextprotocol/go-sdk` (preferencial). Se interface instável em 2026-Q2, implementar wire minimal em `internal/runtime/mcpserver/wire/` com schema oficial (~150 LoC adicionais). Registrar a decisão no `execution_report.md`.
- Profundidade rastreada via env var `AISPEC_RUN_AGENT_CONTEXT` (JSON: `parent_session_id`, `depth`, `workspace_root`).
- Profundidade máxima default 3 (configurável via env `AISPEC_MAX_AGENT_DEPTH`).
- Timeout default 300s; max 1800s. Override via parâmetro `timeout` da tool.
- Erro tipado MCP quando: `agent_name` desconhecido OU profundidade ≥ maxDepth OU timeout excedido.
- Child session produz `events.jsonl` próprio em `<parent_evidence_dir>/nested/<child_session_id>/`. Eventos do child espelhados no `events.jsonl` do parent com `kind="nested_agent"`.
- Cobertura ≥ 80% no subpacote `mcpserver/`.
</requirements>

## Subtarefas

- [ ] 1.1 Definir interface `Server`, tipos `RunAgentInput`/`RunAgentOutput`, constante `ReservedToolName`
- [ ] 1.2 Implementar handshake MCP stdio (lib SDK ou wire minimal — decisão documentada)
- [ ] 1.3 Implementar handler `HandleRunAgent` que valida input, resolve agent via registry, checa depth
- [ ] 1.4 Implementar `engine.go::spawnNestedSession` reusando `ACPRunner`
- [ ] 1.5 Serializar `NestedExecutionContext` em env var ao spawnar child
- [ ] 1.6 Persistir eventos do child em `<evidence>/nested/<id>/events.jsonl` + espelhar no parent com kind `nested_agent`
- [ ] 1.7 Escrever testes T-MCP-01..T-MCP-06 (ver techspec §"Abordagem de Testes")

## Detalhes de Implementação

Ver `techspec.md` §"Design de Implementação" → "F2-Claude" → `internal/runtime/mcpserver/server.go`. Stubs prontos no documento. **Não duplicar conteúdo aqui** — `execute-task` carrega techspec automaticamente.

Pontos críticos:
- Schema da tool segue MCP spec oficial (não inventar campos).
- `agents.Registry` é interface já entregue por F1 (ADR-011) — apenas consumir.
- `ACPRunner` já é reusável (cliente factory injetado) — não duplicar lógica de spawn em `engine.go`; chamar `runtime.NewACPRunner(...).Run(ctx, childJob)`.
- Trace `parent_session_id` em **todos** os eventos do child (campo `parent_session_id` no Event) para auditoria forense.

## Critérios de Sucesso

- `go test ./internal/runtime/mcpserver/... -coverprofile=cov.out` reporta ≥ 80% de cobertura.
- T-MCP-01..T-MCP-06 verdes.
- `server_test.go` valida que **apenas** `run_agent` é exposta (lista de tools tem len=1).
- `engine_test.go` valida spawn de child com `agent_name="reviewer"` (mock registry).
- Erro tipado retornado em casos: agent desconhecido, depth=4 com maxDepth=3, timeout=1s + child trava 5s.
- `evidence_dir` retornado no output é válido (existe + contém `events.jsonl`).
- `execution_report.md` documenta decisão MCP-SDK vs wire minimal.

## Skills Necessárias

Nenhuma além das auto-carregadas (governance + linguagem).

## Testes da Tarefa

- [ ] Testes unitários: T-MCP-01..T-MCP-06 (`internal/runtime/mcpserver/server_test.go` + `engine_test.go`)
- [ ] Cobertura ≥ 80% no subpacote
- [ ] Sem testes de integração nesta task — task 3.0 cobre E2E com runner

<critical>SEMPRE CRIAR E EXECUTAR TESTES DA TAREFA ANTES DE CONSIDERAR A TAREFA COMO `done`</critical>

## Arquivos Relevantes

- **Novo** `internal/runtime/mcpserver/server.go` (~150 LoC)
- **Novo** `internal/runtime/mcpserver/engine.go` (~80 LoC)
- **Novo** `internal/runtime/mcpserver/server_test.go` (~180 LoC)
- **Novo** `internal/runtime/mcpserver/engine_test.go` (~70 LoC)
- **Leitor:** `internal/agents/registry.go` (ADR-011)
- **Leitor:** `internal/runtime/runner.go::ACPRunner`
- **Leitor:** `internal/runtime/specs/spec.go`
- **Leitor:** `internal/runtime/persistence/` (PersistenceFactory)
- **Possível wire fallback:** `internal/runtime/mcpserver/wire/` (decisão runtime)
