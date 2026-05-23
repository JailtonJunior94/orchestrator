# Tarefa 9.0: Sub-suite Codex em acp_integration_test.go reusando fake server

<critical>Ler prd.md e techspec.md desta pasta — sua tarefa será invalidada se você pular</critical>

## Visão Geral

Adicionar sub-suite de testes de integração para Codex em `internal/runtime/acp_integration_test.go`, reusando o fake ACP server existente em `internal/runtime/client/client_test.go`. Esta é a **validação de paridade observacional** entre Codex, Claude e Copilot — sem ela, F1-Codex não tem evidência de que os artefatos forenses (`events.jsonl`, `tool_calls.md`, `execution_report.md`) saem com a mesma estrutura.

Casos cobertos:
- T-17: Spec Codex + fake server, `AccessModeRestricted` → spawn args contêm `-c model="...", -c model_reasoning_effort="...", -c features.code_mode=false, -c features.code_mode_only=false`; **NÃO** contêm sandbox flags.
- T-18: Spec Codex + fake server, `AccessModeFull` → spawn args contêm todos os de T-17 + `-c approval_policy="never", -c sandbox_mode="danger-full-access", -c web_search="live"`.
- T-19: Regressão Claude — spawn args **NÃO** contêm nenhum `-c` flag (BootstrapArgs no-op preservado).
- T-20: Spec Codex + fake server emite ≥ 2 tool calls → `tool_calls.md` agregado corretamente; `execution_report.md` com counts certos.
- T-21: Spec Codex + fake server inativo → `ActivityWatchdog` cancela via `CancelCause(ErrActivityTimeout)`; `CancelReason == "activity_timeout"`.

Paralelizável com tarefa 10.0 (documentação) — arquivos disjuntos.

<requirements>
- Reusar fake ACP server existente em internal/runtime/client/client_test.go (não duplicar).
- Cada caso valida spawn args correctness (capture via exec.Command mock ou intercepção do launcher).
- events.jsonl produzido durante o teste tem mesma estrutura que Claude/Copilot (kinds, order, fields).
- tool_calls.md e execution_report.md gerados com mesmos campos que Claude/Copilot.
- ActivityWatchdog cancela Codex stuck session da mesma forma que Claude/Copilot.
- T-31 regressão suíte completa de internal/runtime/ 100% verde após adições.
- Sem novo kind de evento ADR-010 (tagged union preservada).
- Sub-suite organizada como sub-test Go (t.Run) sob TestACPIntegrationCodex ou similar.
</requirements>

## Subtarefas

- [ ] 9.1 Identificar estrutura de subtests Codex existente em `acp_integration_test.go` (Claude/Copilot têm padrão similar).
- [ ] 9.2 Criar função/grupo `TestACPIntegrationCodex` reusando fixture do fake ACP server.
- [ ] 9.3 Implementar T-17 (Codex restricted): construir Job com AccessModeRestricted, capturar argv passado ao launcher, assertar `-c` flags esperadas; assertar **ausência** de sandbox flags.
- [ ] 9.4 Implementar T-18 (Codex full): construir Job com AccessModeFull, assertar argv contém triplet sandbox/approval/web_search.
- [ ] 9.5 Implementar T-19 (Claude regressão): construir Job para Claude com flags Codex aleatórias, assertar argv **NÃO** contém `-c` flags.
- [ ] 9.6 Implementar T-20 (eventos): fake server emite agent_message + 2 tool_call_start + 2 tool_call_end + completion; assertar `events.jsonl` linha-a-linha; assertar `tool_calls.md` agregado; assertar `execution_report.md` counts.
- [ ] 9.7 Implementar T-21 (watchdog): configurar fake server inativo; rodar Job com activity-timeout curto; assertar erro `context.Canceled` com `CancelCause == ErrActivityTimeout`; assertar `Summary.CancelReason == "activity_timeout"`.
- [ ] 9.8 Rodar `go test ./internal/runtime/... -run TestACPIntegrationCodex` 100% verde.
- [ ] 9.9 Rodar suíte completa `go test ./internal/runtime/...` → 100% verde (regressão Claude/Copilot intactos).

## Detalhes de Implementação

Ver `techspec.md` §"Abordagem de Testes" → "Testes de Integração" e §"Sequenciamento de Desenvolvimento" → item 12. Decisão registrada em ADR-013 §"Decisão" D-09 (tool name aliasing adiado para F2-Codex; testes nesta fase usam nomes nativos Codex).

Anti-padrão: NÃO criar novo fake ACP server (reusar `internal/runtime/client/client_test.go`); NÃO mockar `exec.Command` (capturar argv via injeção `LookPather` ou `Launcher` já é o padrão dos testes Claude/Copilot).

## Critérios de Sucesso

- 5 subtests novos (T-17..T-21) implementados sob a estrutura existente.
- Fake ACP server reusado (não há duplicação).
- Spawn args para Codex contêm `-c` flags corretos por AccessMode (restricted vs full).
- Spawn args para Claude **não** ganham `-c` flags (regressão T-19).
- `events.jsonl`, `tool_calls.md`, `execution_report.md` produzidos com estrutura paritária a Claude/Copilot.
- `ActivityWatchdog` cancela Codex session como Claude/Copilot.
- Suíte completa de `internal/runtime/...` 100% verde.

## Skills Necessárias

<!-- MANDATÓRIO: preenchido por `create-tasks` Etapa 4.1 via descoberta agnóstica em `.agents/skills/`.
     NÃO inclua aqui skills auto-carregadas em runtime: `agent-governance`, `execute-task`, `bugfix`,
     `review`, `refactor`, nem skills `*-implementation` (linguagem, inferida pelo diff).
     Use o conteúdo único `Nenhuma além das auto-carregadas (governance + linguagem).` se a tarefa
     não exigir skill processual extra. -->

Nenhuma além das auto-carregadas (governance + linguagem).

## Testes da Tarefa

- [ ] T-17: Codex restricted → argv contém model, reasoning, features.code_mode=false; NÃO contém sandbox.
- [ ] T-18: Codex full → argv contém todos de T-17 + sandbox/approval/web_search.
- [ ] T-19: Claude com flags Codex aleatórias → argv **sem** `-c` flags.
- [ ] T-20: Codex + ≥ 2 tool calls → `tool_calls.md` agregado + counts em `execution_report.md`.
- [ ] T-21: Codex stuck → `Summary.CancelReason == "activity_timeout"`.
- [ ] T-31 regressão: `go test ./internal/runtime/...` 100% verde.
- [ ] `go vet ./internal/runtime/...` sem warnings.

<critical>SEMPRE CRIAR E EXECUTAR TESTES DA TAREFA ANTES DE CONSIDERAR A TAREFA COMO `done`</critical>

## Definition of Done (DoD)

- [ ] 5 subtests novos (T-17..T-21) adicionados sob `TestACPIntegrationCodex` ou estrutura equivalente em `acp_integration_test.go`.
- [ ] Fake ACP server reusado (sem duplicação de fixture).
- [ ] Cada subtest valida: argv esperado, events.jsonl estrutura, tool_calls.md/execution_report.md gerados.
- [ ] T-19 (regressão Claude) confirma BootstrapArgs no-op preservado.
- [ ] T-31 suíte completa de `internal/runtime/...` 100% verde (sem regressão Claude/Copilot).
- [ ] `go test -race ./internal/runtime/...` 100% verde (sem race conditions).
- [ ] `go vet ./...` → sem warnings.

## Arquivos Relevantes

- `internal/runtime/acp_integration_test.go` (estender: adicionar subtests Codex)
- `internal/runtime/client/client_test.go` (reusar: fake ACP server)
- `internal/runtime/specs/codex.go` (consumir: tarefa 2.0)
- `internal/runtime/runner.go` (consumir: tarefa 4.0 — BootstrapArgs no spawn)
- ADR-013 §"Decisão" → D-09 (aliasing adiado)
- techspec.md §"Abordagem de Testes" → testes de integração
