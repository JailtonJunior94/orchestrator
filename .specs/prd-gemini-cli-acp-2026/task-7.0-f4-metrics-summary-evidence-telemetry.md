# Tarefa 7.0: F4-Gemini — `gemini_metrics.go` + `Summary` + `convert.go` + `evidence.go` + telemetry

<critical>Ler prd.md e techspec.md desta pasta — sua tarefa será invalidada se você pular</critical>

## Visão Geral

Capturar métricas Gemini-2026 distintivas (`cache_read_tokens`, `effective_context_tokens`, `prompt_tokens_billed`, `thoughts_tokens`) do payload `acp.SessionUpdate` e persistir em `Summary` + `execution_report.md` + telemetria opt-in. Implementação **defensiva** com `omitempty`: ausência de campos no payload retorna zero-value silencioso (TD-02 da techspec). Esta task é **independente** das demais Wave 2 (não depende de F4-Claude) — Gemini metrics são distintas das Claude metrics.

<requirements>
- Novo `internal/runtime/events/gemini_metrics.go` com tipo `GeminiMetrics` (4 campos `int` com `omitempty`) + função `ExtractGeminiMetrics(raw json.RawMessage) (GeminiMetrics, error)` defensiva.
- `internal/runtime/runner.go::Summary` ganha 4 campos opcionais (`GeminiCacheReadTokens` etc.) com `omitempty`.
- `internal/runtime/events/convert.go` chama `ExtractGeminiMetrics` apenas quando `driver_id == "gemini"`. Não introduz novo kind de evento (preserva ADR-010).
- `internal/evidence/evidence.go` aceita seção opcional "Métricas Gemini-2026" no `execution_report.md`. Ausência **não bloqueia** validação (RF-20).
- `internal/telemetry/telemetry.go` registra entries `gemini.cache_read=N`, `gemini.effective_context=N`, `gemini.prompt_billed=N`, `gemini.thoughts=N` quando `GOVERNANCE_TELEMETRY=1` e valor > 0.
- Diff zero em `internal/runtime/persistence/` (apenas serialização de Summary recebido).
- Schema JSON tag: `json:"cache_read_tokens,omitempty"` etc. exato como técspec §"Interface Pública — Extração de Métricas".
</requirements>

## Subtarefas

- [ ] 7.1 Criar `internal/runtime/events/gemini_metrics.go` com `GeminiMetrics struct` + `ExtractGeminiMetrics(raw)` + `LogGeminiMetrics(ctx, m)` (delegação para telemetry).
- [ ] 7.2 Estender `internal/runtime/runner.go::Summary` com 4 campos `Gemini*Tokens int` + tags `json:",omitempty"`.
- [ ] 7.3 Em `internal/runtime/events/convert.go`, adicionar chamada a `ExtractGeminiMetrics(raw)` quando `driver_id == "gemini"`; popular `Summary.Gemini*` correspondentes.
- [ ] 7.4 Estender `internal/evidence/evidence.go` para aceitar seção opcional `"## Métricas Gemini-2026"` com tabela markdown 4-linhas. Ausência silenciosa.
- [ ] 7.5 Em `internal/telemetry/telemetry.go`, adicionar entries `gemini.*` quando `Summary.Gemini*` > 0 e `GOVERNANCE_TELEMETRY=1`.
- [ ] 7.6 Estender `runtime_init` event para incluir `tool=gemini`, `launcher`, `npm_version=0.43.0`, `sdk_version=v0.13.0` (RF-31).
- [ ] 7.7 Adicionar testes T-36 (`ExtractGeminiMetricsFromACPPayload` com 5 cenários), T-37 (`EvidenceRendersGeminiMetricsSection`), T-38 (`EvidenceMissingGeminiMetricsDoesNotBlock`).
- [ ] 7.8 Estender `TestSummarySerialization` para validar que zero-values `Gemini*` são omitidos do JSON (omitempty).
- [ ] 7.9 Estender `TestTelemetryRecordsRuntimeInit` para incluir entries `gemini.*`.

## Detalhes de Implementação

Ver techspec.md:
- §"Interface Pública — Extração de Métricas Gemini-2026" (linhas ~172-205) — código fonte exato de `GeminiMetrics` e `ExtractGeminiMetrics`.
- §"Modelos de Dados — Extensão de `Summary`" (linhas ~207-225) — schema exato dos novos campos.
- §"Considerações Técnicas / TD-02" — política defensiva (omitempty + zero-value silencioso).
- §"Monitoramento e Observabilidade" — entries telemetry específicos.
- §"Mapeamento RF → Componente → Teste" — RF-18..RF-21, RF-31.

Compozy não tem precedente direto (Compozy não extrai métricas Gemini-2026). Referência arquitetural: `internal/runtime/events/metrics.go::ExtractClaudeMetrics` (paralelo F4-Claude).

## Critérios de Sucesso

- `internal/runtime/events/gemini_metrics.go` existe com tipo + função + LogGeminiMetrics.
- `internal/runtime/runner.go::Summary` ganha 4 campos com `omitempty` validados via JSON serialization test.
- `internal/runtime/events/convert.go` extrai métricas Gemini apenas quando driver=gemini (cobertura de teste).
- `internal/evidence/evidence.go` aceita seção opcional sem bloquear.
- `internal/telemetry/telemetry.go` registra entries `gemini.*` quando aplicável.
- `go test -run 'TestExtractGeminiMetrics|TestEvidenceRendersGeminiMetricsSection|TestEvidenceMissingGeminiMetricsDoesNotBlock|TestSummarySerialization' ./internal/...` retorna `PASS` em todos.
- `go test -run TestTelemetryRecordsRuntimeInit ./internal/telemetry/...` retorna `PASS` com cobertura Gemini.
- `git diff --stat internal/runtime/persistence/` retorna 0 linhas (RF-32 — persistência inalterada).
- Suite regressão Claude metrics (F4-Claude) verde.

### Definition of Done

1. `GeminiMetrics` tipo + função defensiva implementados e testados (5 cenários T-36).
2. Summary com 4 campos `omitempty` validados.
3. `convert.go` extrai apenas quando driver=gemini; outros drivers inalterados.
4. Evidence aceita seção opcional; ausência não bloqueia (T-38).
5. Telemetry registra entries `gemini.*` opt-in.
6. Diff zero em persistence (apenas Summary serialization).
7. Regressão F4-Claude metrics verde.

## Skills Necessárias

<!-- MANDATÓRIO: preenchido por `create-tasks` Etapa 4.1 via descoberta agnóstica em `.agents/skills/`.
     NÃO inclua aqui skills auto-carregadas em runtime: `agent-governance`, `execute-task`, `bugfix`,
     `review`, `refactor`, nem skills `*-implementation` (linguagem, inferida pelo diff).
     Use o conteúdo único `Nenhuma além das auto-carregadas (governance + linguagem).` se a tarefa
     não exigir skill processual extra. -->

Nenhuma além das auto-carregadas (governance + linguagem).

## Testes da Tarefa

- [ ] T-36 `TestExtractGeminiMetricsFromACPPayload` — 5 cenários: payload vazio, payload parcial, payload completo, chaves inesperadas, JSON inválido
- [ ] T-37 `TestEvidenceRendersGeminiMetricsSection` — Summary com Gemini* não-zero renderiza tabela
- [ ] T-38 `TestEvidenceMissingGeminiMetricsDoesNotBlock` — Summary com Gemini* zero permite ausência sem erro
- [ ] Extensão de `TestSummarySerialization` — omitempty para Gemini* zero
- [ ] Extensão de `TestTelemetryRecordsRuntimeInit` — entries `gemini.*` quando tool=gemini
- [ ] Regressão F4-Claude metrics verde

<critical>SEMPRE CRIAR E EXECUTAR TESTES DA TAREFA ANTES DE CONSIDERAR A TAREFA COMO `done`</critical>

## Arquivos Relevantes

- **NOVO**: `internal/runtime/events/gemini_metrics.go`
- **NOVO**: `internal/runtime/events/gemini_metrics_test.go`
- **EDIÇÃO**: `internal/runtime/runner.go` (Summary struct + 4 campos)
- **EDIÇÃO**: `internal/runtime/events/convert.go` (chamada a ExtractGeminiMetrics quando driver=gemini)
- **EDIÇÃO**: `internal/evidence/evidence.go` (seção opcional Métricas Gemini-2026)
- **EDIÇÃO**: `internal/evidence/evidence_test.go` (T-37, T-38)
- **EDIÇÃO**: `internal/telemetry/telemetry.go` (entries gemini.*)
- **EDIÇÃO**: `internal/telemetry/telemetry_test.go` (extensão de runtime_init)
- **REFERÊNCIA**: `internal/runtime/events/metrics.go::ExtractClaudeMetrics` (paralelo F4-Claude)
