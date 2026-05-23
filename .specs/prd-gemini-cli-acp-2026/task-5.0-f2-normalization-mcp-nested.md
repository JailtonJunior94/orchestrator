# Tarefa 5.0: F2-Gemini — YAML `inherit: common` + MCP nested-agent integration test

<critical>Ler prd.md e techspec.md desta pasta — sua tarefa será invalidada se você pular</critical>

## Visão Geral

Adicionar entrada Gemini na tabela de normalização de tool-calls (`.agents/normalization-rules.yaml` e arquivo embedded `internal/runtime/events/normalization-rules.yaml`) usando `inherit: common` (Compozy confirma em `tool_call_name.go:84` que Gemini herda `commonToolTitleAliases` sem override). Se `internal/runtime/events/normalize.go::BuildNormalizedToolCall` ainda não suportar resolução de `inherit:`, adicionar ~10 LoC para isso. Criar integration test (`tests/integration/gemini_mcp_nested_test.go`) validando que MCP nested-agent (`internal/runtime/mcpserver/`, F2-Claude infra) funciona com Gemini parent invocando `run_agent("reviewer", ...)`.

**Dependência inter-PRD**: requer `.specs/prd-claude-cli-acp-2026/` F2-Claude entregue — provê `internal/runtime/events/normalize.go` e `internal/runtime/mcpserver/`. Se ausente, marcar como `blocked` em `tasks.md` até dependência resolver.

<requirements>
- Entrada YAML canônica: `gemini: { inherit: common, overrides: {} }` (em ambos os arquivos: canônico e embedded).
- Validação de simetria YAML: arquivos canônico e embedded **idênticos** (lint via diff direto ou script de sync).
- Se `resolveInherit` ausente em `normalize.go`, adicionar com testes dedicados.
- Integration test reusa fake ACP server + fake MCP responder; verifica que child session Gemini é spawnada com prompt isolado quando parent invoca `run_agent`.
- Depth limit (≤ 3) aplicado igual a Claude/Codex (Compozy convenção; já testado em F2-Claude).
- Diff zero em `internal/runtime/mcpserver/` (apenas teste novo).
</requirements>

## Subtarefas

- [ ] 5.1 Validar pré-requisito: `git log --oneline internal/runtime/events/normalize.go internal/runtime/mcpserver/` mostra entregas de F2-Claude. Se ausentes, marcar task como `blocked`.
- [ ] 5.2 Adicionar entrada `gemini: { inherit: common, overrides: {} }` em `.agents/normalization-rules.yaml`.
- [ ] 5.3 Espelhar entrada em `internal/runtime/events/normalization-rules.yaml` (embedded via `go:embed`).
- [ ] 5.4 Verificar/implementar `resolveInherit` em `internal/runtime/events/normalize.go` se ausente (F2-Claude deve ter entregue; validar via teste).
- [ ] 5.5 Adicionar teste T-32: `TestNormalizeToolCallGeminiInheritsCommon` — Gemini emite `read_file` → normalizado `Read`; `raw_name` preservado.
- [ ] 5.6 Criar `tests/integration/gemini_mcp_nested_test.go` (build tag `//go:build integration`) implementando T-33: `TestMCPNestedAgentSpawnsGeminiSession` (parent Gemini invoca `run_agent("reviewer", "<prompt>")`; child session é spawnada; depth limit aplicado em recursão depth=4).

## Detalhes de Implementação

Ver techspec.md:
- §"Arquitetura do Sistema / F2-Gemini" (linhas ~46-50) — escopo exato.
- §"Pontos de Integração / Compozy" — confirma que Gemini herda `commonToolTitleAliases`.
- §"Mapeamento RF → Componente → Teste" — RF-13, RF-14, RF-15.

Precedente direto: `.specs/prd-claude-cli-acp-2026/` Wave F2-Claude tasks (referenciar via `git log`).

## Critérios de Sucesso

- `.agents/normalization-rules.yaml` e `internal/runtime/events/normalization-rules.yaml` contêm entrada Gemini idêntica.
- `diff .agents/normalization-rules.yaml internal/runtime/events/normalization-rules.yaml` retorna 0 linhas divergentes (script de sync deve cobrir).
- `go test -run TestNormalizeToolCallGeminiInheritsCommon ./internal/runtime/events/...` retorna `PASS`.
- `go test -tags integration -run TestMCPNestedAgentSpawnsGeminiSession ./tests/integration/...` retorna `PASS` (skipable quando dependências externas indisponíveis).
- `git diff --stat internal/runtime/mcpserver/` retorna 0 arquivos modificados (apenas teste novo em `tests/integration/`).
- Suite normalization existente Claude/Codex/Copilot permanece 100% verde.

### Definition of Done

1. Entradas YAML canônico + embedded sincronizadas.
2. T-32 verde (normalização Gemini); T-33 verde (MCP nested).
3. Dependência inter-PRD F2-Claude confirmada via `git log` antes do início da task.
4. Diff zero em `internal/runtime/mcpserver/` (apenas testes novos).
5. Regressão de normalização Claude/Codex/Copilot verde.

## Skills Necessárias

<!-- MANDATÓRIO: preenchido por `create-tasks` Etapa 4.1 via descoberta agnóstica em `.agents/skills/`.
     NÃO inclua aqui skills auto-carregadas em runtime: `agent-governance`, `execute-task`, `bugfix`,
     `review`, `refactor`, nem skills `*-implementation` (linguagem, inferida pelo diff).
     Use o conteúdo único `Nenhuma além das auto-carregadas (governance + linguagem).` se a tarefa
     não exigir skill processual extra. -->

Nenhuma além das auto-carregadas (governance + linguagem).

## Testes da Tarefa

- [ ] T-32 `TestNormalizeToolCallGeminiInheritsCommon` (unit)
- [ ] T-32b `TestNormalizeGeminiPreservesRawName` (raw_name lado a lado com normalized_name)
- [ ] T-33 `TestMCPNestedAgentSpawnsGeminiSession` (integration)
- [ ] T-33b `TestMCPDepthLimitAppliesToGemini` (integration; depth=4 retorna erro tipado)
- [ ] Sync YAML: lint script `scripts/check-normalization-yaml-sync.sh` ou `diff` direto

<critical>SEMPRE CRIAR E EXECUTAR TESTES DA TAREFA ANTES DE CONSIDERAR A TAREFA COMO `done`</critical>

## Arquivos Relevantes

- **EDIÇÃO**: `.agents/normalization-rules.yaml` (entrada gemini)
- **EDIÇÃO**: `internal/runtime/events/normalization-rules.yaml` (espelho embedded)
- **EDIÇÃO (se necessário)**: `internal/runtime/events/normalize.go` (`resolveInherit` se ausente)
- **EDIÇÃO**: `internal/runtime/events/normalize_test.go` (T-32, T-32b)
- **NOVO**: `tests/integration/gemini_mcp_nested_test.go`
- **REFERÊNCIA (não modificar)**: `internal/runtime/mcpserver/server.go`, `engine.go`, `wire.go`
