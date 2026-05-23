# Tarefa 3.0: Integrar MCP + normalize no `runner.go`, flags CLI F2 + smoke E2E + `CLAUDE.md`

<critical>Ler prd.md e techspec.md desta pasta — sua tarefa será invalidada se você pular</critical>

## Visão Geral

Integrar artefatos de F2-Claude (MCP server da task 1.0 + normalize da task 2.0) em `internal/runtime/runner.go::Run()`. Adicionar flags CLI `--mcp-nested` e `--no-normalize` em `cmd/ai_spec_harness/task_loop.go`. Estender `Job` com campos `MCPNested`/`NoNormalize`. Adicionar smoke test E2E F2 em `tests/integration/`. Atualizar `CLAUDE.md` raiz com §"Runtime Capabilities".

<requirements>
- Estender `runner.go::Job` com campos `MCPNested bool`, `NoNormalize bool`, `TaskFileName string`
- Em `runner.go::Run()`, quando `j.MCPNested=true`: spawnar `mcpserver.Server` em goroutine antes de `c.Open`; injetar `--mcp-server stdio://...` no launcher
- No loop de eventos, normalização aplicada quando `!j.NoNormalize`. Persistência ganha campos `normalized_name` ao lado de `raw_name`
- `tool_calls.md` renderiza nome normalizado (já agnóstico após F1-Copilot — confirmar)
- Flags em `cmd/ai_spec_harness/task_loop.go`: `--mcp-nested` (default `false`), `--no-normalize` (default `false`). Propagar via `taskloop.Options` até `Job`
- T-15 (`task_loop_test.go`): caso novo cobrindo `--mcp-nested --no-normalize`
- T-INT-01 e T-INT-02 (smoke E2E) em `tests/integration/claude_2026_e2e_test.go`
- `CLAUDE.md` raiz: nova §"Runtime Capabilities (F2-Claude+)" listando MCP + normalize (esboço em `techspec.md` §"Exemplos de Configuração 2026")
- Defaults preservam comportamento atual: sessão sem flag novas roda idêntico a F1-Claude (regressão hard)
- Possível refatoração de `runner.go` se complexidade ciclomática crescer — usar `object-calisthenics-go` como heurística
</requirements>

## Subtarefas

- [ ] 3.1 Estender `Job` struct em `runner.go` com 3 campos novos (defaults zero-value)
- [ ] 3.2 Implementar spawn condicional do `mcpserver.Server` em `Run()` antes de `c.Open` (linha ~145)
- [ ] 3.3 Aplicar `events.BuildNormalizedToolCall` no loop de eventos (linhas ~163-185) quando `!j.NoNormalize`
- [ ] 3.4 Estender `persistence` para escrever `normalized_name`/`raw_name` em `events.jsonl`
- [ ] 3.5 Adicionar 2 flags em `cmd/ai_spec_harness/task_loop.go` + propagação
- [ ] 3.6 Adicionar T-15 em `task_loop_test.go`
- [ ] 3.7 Implementar T-INT-01 e T-INT-02 em `tests/integration/claude_2026_e2e_test.go` (mock ACP client + prompt forçando `run_agent`)
- [ ] 3.8 Atualizar `CLAUDE.md` com §"Runtime Capabilities (F2-Claude+)"
- [ ] 3.9 Rodar smoke manual: `ai-spec task-loop --tool claude --runtime acp --mcp-nested .specs/<prd-de-teste>` e verificar `events.jsonl` com kind `nested_agent`

## Detalhes de Implementação

Ver `techspec.md` §"Design de Implementação" → "Assinaturas principais" para os trechos cirúrgicos a aplicar em `runner.go`. **Não duplicar aqui** — `execute-task` carrega techspec automaticamente.

Pontos críticos:
- **Não introduzir conflito de merge com task 6.0**: ambas tocam `runner.go::Run()`. Esta task ocupa a região "antes de `c.Open`" (MCP spawn) e o "loop de eventos" (normalize). Task 6.0 ocupa "antes de `c.Open`" também (memory read + hook dispatch) — coordenar via dependência sequencial declarada (6.0 depende de 3.0).
- **Defaults preservam comportamento**: T-19 (`runner_test.go` existente) tem que continuar passando sem alteração.
- **MCP server lifecycle**: `mcpserver.Server` tem `Serve(ctx, in, out)` retornando quando `ctx` cancela. Spawn em goroutine; `defer` para garantir cleanup quando `Run()` retornar.
- **Endereço stdio**: passar via param `--mcp-server stdio://fd-3` (FD herdado) ou via socket Unix temporário em `os.TempDir()`. Decisão documentada no `execution_report.md`.
- **`tool_calls.md` renderer** já vive em `internal/runtime/persistence/` — confirmar que consume `normalized_name` quando presente; caso contrário, adicionar (mínimo ~10 LoC).
- **`CLAUDE.md` raiz** está em modo de governança transversal — manter estilo conciso (≤30 linhas para a §"Runtime Capabilities").

## Critérios de Sucesso

- `make test` verde (cobertura global ≥ 70%)
- `make integration` verde (T-INT-01 e T-INT-02 inclusos)
- `make parity` reporta 31 invariantes verdes (do INV-30 ativado por esta task)
- T-19 (regressão F1-Claude) passa sem alteração
- T-15 (novo caso `--mcp-nested --no-normalize`) passa
- Smoke manual: `events.jsonl` contém eventos com `kind="nested_agent"` quando prompt instrui `run_agent`
- `events.jsonl` contém `normalized_name="bash"` + `raw_name="bash"` em sessão Claude com `bash` tool call
- `tool_calls.md` renderiza nome normalizado (confirmar visualmente)
- `CLAUDE.md` ganhou §"Runtime Capabilities (F2-Claude+)" com listagem de capabilities
- Sem regressão em sessões existentes sem flags novas (idêntico a F1-Claude)

## Skills Necessárias

- `object-calisthenics-go` — `runner.go` cresceu para >300 LoC após F1-Codex; integração F2 adiciona ~50 LoC tocando função `Run()` que já está próxima do limite. Heurística OC orienta extração de helpers (`spawnMCPServer`, `normalizeEventInline`) para manter clareza sem quebrar contrato público.

## Testes da Tarefa

- [ ] Testes unitários: T-15 em `cmd/ai_spec_harness/task_loop_test.go`; regressão T-19 mantida
- [ ] Testes de integração: T-INT-01 (MCP nested E2E) e T-INT-02 (normalização cross-tool E2E) em `tests/integration/claude_2026_e2e_test.go`
- [ ] Smoke manual documentado no `execution_report.md` com paths reais de `events.jsonl` produzido
- [ ] Cobertura ≥ 70% global mantida

<critical>SEMPRE CRIAR E EXECUTAR TESTES DA TAREFA ANTES DE CONSIDERAR A TAREFA COMO `done`</critical>

## Arquivos Relevantes

- **Modificado** `internal/runtime/runner.go` (~80 LoC novas; região `Run()`)
- **Modificado** `internal/runtime/persistence/session.go` (campos `normalized_name`/`raw_name` em `AppendEvent`)
- **Modificado** `cmd/ai_spec_harness/task_loop.go` (2 flags novas + propagação)
- **Modificado** `cmd/ai_spec_harness/task_loop_test.go` (+T-15)
- **Modificado** `internal/runtime/runner_test.go` (validar regressão T-19 e novos casos)
- **Modificado** `CLAUDE.md` (raiz) — nova §"Runtime Capabilities"
- **Novo** `tests/integration/claude_2026_e2e_test.go` (esqueleto + T-INT-01 + T-INT-02 — testes posteriores adicionam casos)
- **Leitor:** `internal/runtime/mcpserver/` (task 1.0)
- **Leitor:** `internal/runtime/events/normalize.go` (task 2.0)
