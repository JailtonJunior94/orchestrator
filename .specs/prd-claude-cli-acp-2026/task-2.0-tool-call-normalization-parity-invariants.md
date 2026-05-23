# Tarefa 2.0: Tool-call normalization driver-aware + invariantes INV-30/INV-31

<critical>Ler prd.md e techspec.md desta pasta — sua tarefa será invalidada se você pular</critical>

## Visão Geral

Implementar `BuildNormalizedToolCall(driverID, kind, rawInput, locations) NormalizedToolCall` em `internal/runtime/events/normalize.go`, replicando o padrão de `compozy/internal/core/agent/tool_call_input.go::buildNormalizedToolUseBlock`. Tabelas de alias em `.agents/normalization-rules.yaml` carregadas via `go:embed`. Adicionar invariantes INV-30 (normalização cross-tool) e INV-31 (MCP nested depth) ao `internal/parity/`.

<requirements>
- Criar `internal/runtime/events/normalize.go` com função `BuildNormalizedToolCall`
- Criar `.agents/normalization-rules.yaml` com tabela mínima cobrindo: `bash`↔`shell`, `read_file`↔`read`, `write_file`↔`write`, `web_search`↔`search_query`, `image_search`↔`image_query`
- `RawName` e `RawInput` **nunca** são mutados — preservados lado a lado com `NormalizedName`/`NormalizedInput`
- Override por projeto: workspace `.agents/normalization-rules.yaml` **vence** sobre embedded default
- Driver desconhecido OU rawName não-mapeado → passthrough (NormalizedName = rawName); sem erro
- Adicionar `INV-30 tool_calls_normalized_name_invariant` em `internal/parity/invariants.go`: mesma operação semântica em Claude e Codex → `normalized_name` idêntico em `events.jsonl`
- Adicionar `INV-31 mcp_nested_depth_never_exceeds_max` em `internal/parity/invariants.go`: eventos com kind `nested_agent` têm `depth ≤ AISPEC_MAX_AGENT_DEPTH`
- Os 29 invariantes existentes de ADR-008 continuam passando (regressão hard)
- Cobertura ≥ 80% no novo arquivo `normalize.go`
</requirements>

## Subtarefas

- [ ] 2.1 Criar `.agents/normalization-rules.yaml` com schema (`version: 1`, `aliases:`, `input_mappings:`)
- [ ] 2.2 Criar `internal/runtime/events/normalize.go` com tipos `NormalizedToolCall`, função `BuildNormalizedToolCall`, loader via `go:embed`
- [ ] 2.3 Implementar lookup de project override (`.agents/normalization-rules.yaml` no workspace) com fallback para embedded
- [ ] 2.4 Implementar normalização de input (renomear chaves conforme `input_mappings`); preservar valores
- [ ] 2.5 Criar `normalize_test.go` com T-NORM-01..T-NORM-07
- [ ] 2.6 Adicionar INV-30 em `internal/parity/invariants.go` (predicado + dados de validação)
- [ ] 2.7 Adicionar INV-31 em `internal/parity/invariants.go`
- [ ] 2.8 Atualizar `internal/parity/invariants_test.go` cobrindo INV-30 + INV-31

## Detalhes de Implementação

Ver `techspec.md` §"Design de Implementação" → "F2-Claude" → `internal/runtime/events/normalize.go`. Tabela YAML exemplo está no documento. **Não duplicar aqui** — `execute-task` carrega techspec automaticamente.

Pontos críticos:
- `RawInput` é `json.RawMessage` — não decodificar/recodificar quando não há mapping aplicável (preserva bytes idênticos).
- `NormalizedInput` é uma **cópia** de `RawInput` com chaves renomeadas — nunca mutar `RawInput`.
- Schema YAML é fechado (`version: 1`); rejeitar versions desconhecidas com erro claro de loader.
- INV-30 precisa de fixtures cross-tool — usar `tests/fixtures/parity/claude_bash.jsonl` + `tests/fixtures/parity/codex_shell.jsonl` (criar se não existirem).
- INV-31 é predicado sobre `events.jsonl` — varrer e checar `evt.Depth ≤ env("AISPEC_MAX_AGENT_DEPTH", 3)` quando `evt.Kind == "nested_agent"`.

## Critérios de Sucesso

- T-NORM-01..T-NORM-07 verdes (ver techspec §"Abordagem de Testes")
- `go test ./internal/runtime/events/... -coverprofile=cov.out` reporta ≥ 80% no arquivo `normalize.go`
- `make parity` reporta **31 invariantes** verdes (29 existentes + INV-30 + INV-31)
- Workspace override demonstrado por T-NORM-07: criar `.agents/normalization-rules.yaml` local com alias customizado e validar vitória sobre embedded
- `RawName`/`RawInput` byte-identical em todos os casos de teste (T-NORM-04)

## Skills Necessárias

Nenhuma além das auto-carregadas (governance + linguagem).

## Testes da Tarefa

- [ ] Testes unitários: T-NORM-01..T-NORM-07 em `internal/runtime/events/normalize_test.go`
- [ ] Testes de parity: INV-30 + INV-31 em `internal/parity/invariants_test.go`
- [ ] Cobertura ≥ 80% no subpacote
- [ ] Smoke manual: `make parity` reporta 31 invariantes verdes

<critical>SEMPRE CRIAR E EXECUTAR TESTES DA TAREFA ANTES DE CONSIDERAR A TAREFA COMO `done`</critical>

## Arquivos Relevantes

- **Novo** `internal/runtime/events/normalize.go` (~120 LoC)
- **Novo** `internal/runtime/events/normalize_test.go` (~180 LoC)
- **Novo** `.agents/normalization-rules.yaml` (~30 linhas)
- **Modificado** `internal/parity/invariants.go` (+INV-30, +INV-31)
- **Modificado** `internal/parity/invariants_test.go` (+casos)
- **Novo** `tests/fixtures/parity/claude_bash.jsonl` (fixture para INV-30)
- **Novo** `tests/fixtures/parity/codex_shell.jsonl` (fixture para INV-30)
- **Leitor:** `internal/runtime/events/event.go` (ADR-010 — tagged union)
