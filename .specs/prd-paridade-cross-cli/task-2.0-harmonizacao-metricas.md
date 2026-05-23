# Tarefa 2.0: Harmonização de métricas (extractor por driver + render unificado)

<critical>Ler prd.md e techspec.md desta pasta — sua tarefa será invalidada se você pular</critical>

## Visão Geral

Unificar a extração de métricas: substituir os 11 campos planos de métrica do `Summary` e os 8 acumuladores paralelos do `runEventLoop` por um único `MetricSet`, alimentado por uma estratégia `MetricsExtractor` selecionada por `DriverID`. Render do report passa a omitir campos ausentes (RP-02 — nunca divergentes).

<requirements>
- Interface `MetricsExtractor { Extract(raw json.RawMessage) MetricSet }` em `internal/runtime/events`.
- Implementações: `claudeExtractor` (migra `ExtractClaudeMetrics`), `geminiExtractor` (migra `ExtractGeminiMetrics`), `nullExtractor` (zero-value para `codex`/`copilot`).
- Factory `ExtractorFor(DriverID) MetricsExtractor`.
- `runEventLoop` acumula um único `MetricSet` via `Merge` (remove os 8 acumuladores Claude/Gemini paralelos).
- `Summary` compõe `Metrics MetricSet` (remove campos planos de métrica).
- `RenderMetricsSection(MetricSet)` único e idempotente em `persistence/report.go`; omite campos zero; mantém compatibilidade do header quando só houver métricas Claude.
- Telemetria opt-in (RG-04/ADR-006) preservada: `IsZero()` ⇒ nada logado.
</requirements>

## Subtarefas

- [ ] 2.1 `events/extractor.go`: interface + 3 implementações + `ExtractorFor`.
- [ ] 2.2 Refatorar `runEventLoop` (`runner.go`) para `MetricSet` único por `Merge`.
- [ ] 2.3 `Summary.Metrics` (`summary.go`) + atualizar leitores em `persistence`/`evidence`.
- [ ] 2.4 `RenderMetricsSection` unificado (`report.go`) + idempotência.
- [ ] 2.5 Testes: extractors (payload completo/parcial/ausente), report golden por driver, telemetria opt-in.

## Detalhes de Implementação

Ver techspec.md §"Design de Implementação" (interfaces) e §"Modelos de Dados" (`Summary` modificado). ADR: [021](adr-021-metricset-vo-extractor-por-driver.md). Reusar `claudeMetricsEnvelope` (`convert.go`) e `geminiMetricsEnvelope` (`gemini_metrics.go`) na borda.

## Critérios de Sucesso

- Sessão sem `usage` (Codex/Copilot) não emite seção de métricas no `execution_report.md`.
- `EnrichReport` idempotente (chamadas sucessivas produzem o mesmo arquivo).
- Nenhum campo de métrica divergente entre drivers (só presentes são renderizados — RP-02).
- `make test` verde; validador de evidence (`internal/evidence`) continua detectando a seção.

## Skills Necessárias

<!-- MANDATÓRIO: preenchido por `create-tasks` Etapa 4.1 via descoberta agnóstica em `.agents/skills/`.
     NÃO inclua aqui skills auto-carregadas em runtime: `agent-governance`, `execute-task`, `bugfix`,
     `review`, `refactor`, nem skills `*-implementation` (linguagem, inferida pelo diff).
     Use o conteúdo único `Nenhuma além das auto-carregadas (governance + linguagem).` se a tarefa
     não exigir skill processual extra. -->

Nenhuma além das auto-carregadas (governance + linguagem).

## Testes da Tarefa

- [ ] Testes unitários
- [ ] Testes de integração

<critical>SEMPRE CRIAR E EXECUTAR TESTES DA TAREFA ANTES DE CONSIDERAR A TAREFA COMO `done`</critical>

## Arquivos Relevantes
- `internal/runtime/events/extractor.go` (novo) + `extractor_test.go`
- `internal/runtime/events/convert.go`, `internal/runtime/events/gemini_metrics.go` (migração)
- `internal/runtime/runner.go`, `internal/runtime/summary.go`
- `internal/runtime/persistence/report.go` + `report_test.go`
- `internal/evidence/*` (leitores da seção de métricas)
