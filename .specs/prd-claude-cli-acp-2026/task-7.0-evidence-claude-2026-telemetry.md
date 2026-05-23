# Tarefa 7.0: Evidence Claude-2026 (cache/thinking) + telemetria opt-in estendida

<critical>Ler prd.md e techspec.md desta pasta — sua tarefa será invalidada se você pular</critical>

## Visão Geral

Extrair campos opcionais do payload ACP (`cache_read_tokens`, `cache_creation_tokens`, `thinking_tokens`, `tool_calls_normalized_count`) em `internal/runtime/events/convert.go`. Estender `Summary` em `runner.go`. Adicionar seção "Métricas Claude-2026" em `execution_report.md` (opcional, ausência não bloqueia). Estender telemetria opt-in (`internal/telemetry/`) com 4 entries novas.

<requirements>
- Estender `internal/runtime/events/convert.go` para extrair (quando presentes no `acp.SessionUpdate`):
  - `cache_read_tokens` (de `usage.cache_read_input_tokens`)
  - `cache_creation_tokens` (de `usage.cache_creation_input_tokens`)
  - `thinking_tokens` (de reasoning content block, se existir)
  - `tool_calls_normalized_count` (incrementado pela task 3.0 — já no Summary)
- Estender `internal/runtime/runner.go::Summary` com esses 4 campos (todos opcionais, default 0; já parcial em task 3.0 para `ToolCallsNormalizedCount`)
- Estender `internal/evidence/evidence.go` com seção opcional "Métricas Claude-2026" no `execution_report.md`:
  ```markdown
  ## Métricas Claude-2026
  | Métrica | Valor |
  |---|---|
  | cache_read_tokens | N |
  | cache_creation_tokens | N |
  | thinking_tokens | N |
  | tool_calls_normalized | N |
  ```
- Validador de evidence: presença opcional; ausência não bloqueia
- Estender `internal/telemetry/telemetry.go`: quando `GOVERNANCE_TELEMETRY=1`, append 4 entries novas no log:
  - `claude.cache_read=N`
  - `claude.cache_creation=N`
  - `claude.thinking=N`
  - `claude.normalized_tools=N`
- Sem campos novos quando `GOVERNANCE_TELEMETRY` não setado (opt-in preservado, ADR-006)
- 3 casos novos em `convert_test.go` (T-CONV-CR-01..T-CONV-CR-03)
- Sem regressão em invariantes (29 + INV-30 + INV-31 = 31 verdes)
- Cobertura ≥ 70% global mantida
</requirements>

## Subtarefas

- [ ] 7.1 Adicionar 4 campos a `Summary` em `runner.go` (com tag `json:"..."` para serialização opcional)
- [ ] 7.2 Implementar `extractClaudeMetrics(summary *Summary, evt Event)` em `events/convert.go` — extração tipo-safe do payload ACP
- [ ] 7.3 Chamar `extractClaudeMetrics` no loop de eventos do `runner.go` (linha ~166-185, antes de `persist.AppendEvent`)
- [ ] 7.4 Estender `evidence.go` com função `renderClaudeMetricsSection(summary) string` (retorna "" quando todos 0)
- [ ] 7.5 Integrar `renderClaudeMetricsSection` em `EnrichReport` (apêndice ao `execution_report.md`)
- [ ] 7.6 Validar: regex em `evidence.go` que aceita seção ausente (não bloqueia validação)
- [ ] 7.7 Estender `telemetry.go` com 4 novos entries no append quando `GOVERNANCE_TELEMETRY=1`
- [ ] 7.8 Adicionar T-CONV-CR-01..T-CONV-CR-03 em `events/convert_test.go`
- [ ] 7.9 Smoke manual: rodar sessão Claude com prompt caching ativo; verificar `execution_report.md` cita `cache_read_tokens > 0`

## Detalhes de Implementação

Ver `techspec.md` §"Design de Implementação" → "F4-Claude" → "Summary" e "extractClaudeMetrics". **Não duplicar aqui** — `execute-task` carrega techspec automaticamente.

Pontos críticos:
- **Payload ACP é dinâmico**: `usage` map pode não estar presente em todos os updates. Use `_, ok := payload["usage"]` defensivo; campos ausentes mantêm 0 (sem erro).
- **`thinking_tokens` é particular**: aparece em ACP via content block com `type=thinking` ou `type=reasoning` (a verificar contra `coder/acp-go-sdk v0.6.3` em `go.mod`). Documentar a fonte no comentário do código.
- **`tool_calls_normalized_count` já é incrementado** em task 3.0 (loop de eventos). Esta task **apenas** garante que o campo é persistido na seção do `execution_report.md` — sem dupla contagem.
- **Telemetria entries seguem padrão ADR-006**: `<chave>=<valor>` separado por whitespace; sem JSON.
- **Validador de evidence é permissivo**: a seção "Métricas Claude-2026" é opcional. Regex correspondente em `evidence.go` deve permitir presença OU ausência sem bloquear `execution_report.md` válido.
- **`EnrichReport`** já é o ponto único de produção do report — apêndice da seção é o único change-set ali.
- **Sem mudanças em `runner.go` além do `Summary` struct e do call site `extractClaudeMetrics`** — manter cirúrgico.

## Critérios de Sucesso

- T-CONV-CR-01..T-CONV-CR-03 verdes
- T-CONV-CR-03: payload sem `usage` → campos permanecem 0 sem erro
- `make test` verde (cobertura ≥ 70%)
- `make integration` verde (sem regressão F2/F3)
- Smoke manual: `execution_report.md` produzido por sessão com prompt caching ativo contém `cache_read_tokens > 0`
- `GOVERNANCE_TELEMETRY=1 ai-spec task-loop ...` resulta em `.agents/telemetry.log` com entries `claude.cache_read=N`
- Sessão sem `GOVERNANCE_TELEMETRY` não emite entries novas (opt-in preservado)
- 31 invariantes parity verdes
- Defaults preservam comportamento atual: sessão sem prompt caching emite seção "Métricas Claude-2026" com zeros — opcional renderizar ou omitir (decisão: **omitir** quando todos zero, evita poluir reports legados)

## Skills Necessárias

Nenhuma além das auto-carregadas (governance + linguagem).

## Testes da Tarefa

- [ ] Testes unitários: T-CONV-CR-01..T-CONV-CR-03 em `internal/runtime/events/convert_test.go`
- [ ] Teste de evidence: regex aceita presença E ausência da seção
- [ ] Teste de telemetria: entries `claude.*` aparecem com env setado; ausentes sem env
- [ ] Smoke manual documentado em `execution_report.md` da própria task
- [ ] Cobertura ≥ 70% global mantida

<critical>SEMPRE CRIAR E EXECUTAR TESTES DA TAREFA ANTES DE CONSIDERAR A TAREFA COMO `done`</critical>

## Arquivos Relevantes

- **Modificado** `internal/runtime/events/convert.go` (+`extractClaudeMetrics` ~40 LoC)
- **Modificado** `internal/runtime/events/convert_test.go` (+3 casos)
- **Modificado** `internal/runtime/runner.go` (+4 campos em `Summary`; call site `extractClaudeMetrics`)
- **Modificado** `internal/evidence/evidence.go` (+`renderClaudeMetricsSection`; validador permissivo)
- **Modificado** `internal/telemetry/telemetry.go` (+4 entries no append)
- **Modificado** `internal/evidence/evidence_test.go` (+caso validador permissivo)
- **Modificado** `internal/telemetry/telemetry_test.go` (+caso opt-in)
- **Leitor:** `coder/acp-go-sdk v0.6.3` types (verificar campo `thinking`/`reasoning`)
