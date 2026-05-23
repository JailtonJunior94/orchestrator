# Tarefa 8.0: Sub-suite Copilot em acp_integration_test.go reusando fake server

<critical>Ler prd.md e techspec.md desta pasta — sua tarefa será invalidada se você pular</critical>

## Visão Geral

Estender `internal/runtime/acp_integration_test.go` com sub-suite Copilot que reusa a fake ACP server existente em `internal/runtime/client/client_test.go`. Cobertura mínima: T-10 (open + prompt + completion), T-11 (≥ 2 tool calls + agregação em `tool_calls.md`), T-12 (cancel por `ActivityWatchdog`).

Esta é a tarefa de **gate de paridade observacional**: prova end-to-end (com fake server) que `events.jsonl`, `tool_calls.md` e `execution_report.md` saem com a mesma estrutura para Copilot e para Claude.

<requirements>
- Sub-suite Copilot instancia ACPRunner com specs.Copilot() e fake ACP server.
- Validar que runtime_init carrega tool=copilot quando aplicável e carrega versões Copilot.
- Validar paridade estrutural de events.jsonl, tool_calls.md, execution_report.md.
- T-10, T-11, T-12 verdes.
- Reuso do fake server existente — nenhuma duplicação.
</requirements>

## Subtarefas

- [ ] 8.1 Identificar a fake ACP server em `internal/runtime/client/client_test.go` e os helpers de fixture existentes para Claude.
- [ ] 8.2 Criar sub-suite Copilot em `internal/runtime/acp_integration_test.go` (ou função `TestACPIntegration_Copilot`) reusando os helpers via build de Spec com `specs.Copilot()`.
- [ ] 8.3 T-10: sessão completa com open → prompt → agent_message → completion. Verificar que `events.jsonl` contém os kinds esperados e que o `runtime_init` event carrega `sdk_version == CopilotSDKVersion`, `npm_version == CopilotNpmVersion`.
- [ ] 8.4 T-11: sessão com ≥ 2 tool calls (kinds distintos como `Read` e `Bash`, ou equivalente). Validar que `tool_calls.md` agrega corretamente (counts por tool name).
- [ ] 8.5 T-12: sessão com `ActivityWatchdog` configurado para timeout curto e server que deixa de emitir eventos. Validar `CancelReason == activity_timeout` no `execution_report.md`.
- [ ] 8.6 Garantir que `events.jsonl` produzido para Copilot é estruturalmente idêntico ao produzido para Claude (mesmos kinds disponíveis; cardinalidade `tool=copilot` no payload de `runtime_init`).

## Detalhes de Implementação

Ver `techspec.md` §"Abordagem de Testes" → "Testes de Integração". Decisão D-02 (reuso total do stack ACP) e tabela de cobertura RF-10/RF-21.

Anti-padrão: NÃO instanciar fake server novo só para Copilot. NÃO mockar `acpClient` — usar fake server real para preservar paridade semântica com Claude.

## Critérios de Sucesso

- T-10, T-11, T-12 verdes.
- Saída estrutural de Copilot é idêntica a Claude exceto por `tool=copilot` em `runtime_init`.
- Suíte integration de Copilot reusa fake server sem duplicação.

## Skills Necessárias

<!-- MANDATÓRIO: preenchido por `create-tasks` Etapa 4.1 via descoberta agnóstica em `.agents/skills/`.
     NÃO inclua aqui skills auto-carregadas em runtime: `agent-governance`, `execute-task`, `bugfix`,
     `review`, `refactor`, nem skills `*-implementation` (linguagem, inferida pelo diff).
     Use o conteúdo único `Nenhuma além das auto-carregadas (governance + linguagem).` se a tarefa
     não exigir skill processual extra. -->

Nenhuma além das auto-carregadas (governance + linguagem).

## Testes da Tarefa

- [ ] T-10: open + prompt + agent_message + completion + `runtime_init` event com versões Copilot.
- [ ] T-11: ≥ 2 tool calls com kinds distintos; `tool_calls.md` agrega counts corretos por tool name.
- [ ] T-12: server inativo → `ActivityWatchdog` cancela; `execution_report.md` registra `CancelReason: activity_timeout`.
- [ ] Verificação adicional: `events.jsonl` Copilot tem os mesmos kinds de evento que Claude para a mesma sequência de fake server.
- [ ] `go test ./internal/runtime/... -run TestACPIntegration` → 100% verde.

<critical>SEMPRE CRIAR E EXECUTAR TESTES DA TAREFA ANTES DE CONSIDERAR A TAREFA COMO `done`</critical>

## Definition of Done (DoD)

- [ ] Sub-suite Copilot em `acp_integration_test.go` (função ou subtest dedicado).
- [ ] Reusa fake server existente em `internal/runtime/client/client_test.go` sem duplicação.
- [ ] T-10/T-11/T-12 verdes.
- [ ] `events.jsonl` Copilot tem mesma estrutura que Claude; `tool=copilot` em `runtime_init` confirmado em pelo menos um caso.
- [ ] `tool_calls.md` agrega corretamente para Copilot.
- [ ] `execution_report.md` carrega `CancelReason` correto.
- [ ] `go test ./internal/runtime/...` → 100% verde.
- [ ] Diff zero em `internal/runtime/client/client.go` (fake server existente reusado sem mudança).

## Arquivos Relevantes

- `internal/runtime/acp_integration_test.go` (estender — sub-suite Copilot)
- `internal/runtime/client/client_test.go` (consultar — fake server reusado, não modificar)
- `internal/runtime/specs/copilot.go` (Tarefa 2.0)
- `internal/runtime/runner.go` (Tarefa 3.0 — runtime_init generalizado)
- `internal/runtime/probe/probe.go` (Tarefa 4.0 — probe generalizado)
- ADR-012 §"Decisão D-02" (reuso total do stack)
