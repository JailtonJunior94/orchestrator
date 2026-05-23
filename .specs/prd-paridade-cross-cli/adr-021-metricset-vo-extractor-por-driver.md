# Registro de Decisão Arquitetural (ADR)

## Metadados

- **Título:** Value Object `MetricSet` + `MetricsExtractor` por driver (Strategy) e render unificado mínimo-comum
- **Data:** 2026-05-22
- **Status:** Proposta
- **Decisores:** Time ai-spec-harness (owner: JailtonJunior94)
- **Relacionados:** PRD (RP-02, RIN-03); techspec; ADR-006 (telemetria opt-in); `internal/runtime/events/convert.go` (`ExtractClaudeMetrics`/`ClaudeMetrics`); `internal/runtime/events/gemini_metrics.go` (`ExtractGeminiMetrics`/`GeminiMetrics`); `internal/runtime/persistence/report.go` (`RenderClaudeMetricsSection`); `internal/runtime/summary.go`

## Contexto

A extração de métricas hoje é por-driver acoplada e duplicada:

- `eventLoopResult` e `Summary` carregam **11 campos planos** misturando dois drivers: `CacheReadTokens`, `CacheCreationTokens`, `ThinkingTokens`, `ToolCallsNormalizedCount` (Claude) + `GeminiCacheReadTokens`, `GeminiEffectiveContextTokens`, `GeminiPromptTokensBilled`, `GeminiThoughtsTokens` (Gemini).
- O `runEventLoop` acumula ambos os conjuntos a cada update, sempre, mesmo em sessões Codex/Copilot (que não emitem nada → ruído de campos zero).
- `report.go` só renderiza a seção Claude (`RenderClaudeMetricsSection`); Codex/Copilot/Gemini não têm render mínimo unificado. Viola RP-02 ("conjunto mínimo normalizado: `total_tokens`, `cache_read_tokens`, `thinking_tokens`; ausentes ⇒ omitidos, nunca divergentes").

Isso é uma struct anêmica crescendo por driver (anti-R-DDD-001) e uma coleção sem comportamento de primeira classe (OC #4). Adicionar Codex/Copilot hoje significa mais 4–8 campos planos.

## Decisão

1. Introduzir o Value Object **`MetricSet`** em `internal/runtime/events`:
   - Campos canônicos mínimos: `TotalTokens`, `CacheReadTokens`, `ThinkingTokens` + mapa `Extra map[string]int` para campos driver-específicos (ex.: `effective_context_tokens`, `prompt_tokens_billed`).
   - Imutável após construção; método `Merge(other) MetricSet` (soma campo a campo) para acumulação no loop; `IsZero() bool`.
   - Render só inclui campo presente (não-zero) — campos ausentes são **omitidos**, garantindo RP-02 ("nunca divergentes").
2. Definir a interface de domínio **`MetricsExtractor`** (Strategy por driver):
   ```go
   type MetricsExtractor interface {
       Extract(raw json.RawMessage) MetricSet
   }
   ```
   - Implementações: `claudeExtractor`, `geminiExtractor`, e um `nullExtractor` (zero-value) para `codex`/`copilot` enquanto não houver schema próprio.
   - Seleção por `DriverID` (ADR-020) numa factory `ExtractorFor(DriverID) MetricsExtractor`. Drivers sem extractor próprio recebem `nullExtractor` (sem ruído).
3. `runEventLoop` passa a acumular **um único** `MetricSet` via `Merge`, eliminando os 8 acumuladores paralelos.
4. `Summary` compõe `Metrics MetricSet` (substitui os campos planos de métrica). `report.go` ganha `RenderMetricsSection(MetricSet)` único, que renderiza só o que existe.
5. Telemetria (ADR-006) continua opt-in: `MetricSet.IsZero()` ⇒ nada logado/renderizado.

## Alternativas Consideradas

- **Aditivo (campos mínimos ao lado dos atuais).** Vantagem: menor risco imediato. Desvantagem: perpetua duplicação e divergência; cada nova CLI engorda `Summary`. Rejeitada (decisão de produto registrada no PRD: preferir modelo unificado).
- **Um `MetricSet` por driver, sem `Extra`.** Vantagem: tipagem forte por campo. Desvantagem: explode em N tipos; render precisa conhecer cada um. Rejeitada — `Extra` cobre campos raros sem proliferar tipos.
- **Reflection sobre `usage` JSON genérico.** Não-determinístico e opaco para debug. Rejeitada.

## Consequências

### Benefícios Esperados

- `Summary` deixa de crescer por driver; coleção de métricas vira primeira classe (OC #4).
- RP-02 garantido por construção: render omite ausentes ⇒ relatórios nunca divergem por campo zero.
- Adicionar métricas de Codex/Copilot = nova implementação de `MetricsExtractor`, sem tocar `runEventLoop`.

### Trade-offs e Custos

- `Summary` é struct interna mas consumida por `persistence` e `evidence`; refactor exige atualizar leitores. Mitigado: troca mecânica de campo por `summary.Metrics.X`.
- `Extra` (map) introduz acesso por chave string — encapsular atrás de métodos do VO para não vazar chaves cruas.

### Riscos e Mitigações

- **Risco:** report legado esperava a seção "Métricas Claude-2026" com header fixo. **Mitigação:** manter header compatível quando só houver métricas Claude; novos drivers usam header genérico. Testar idempotência de `EnrichReport`.
- **Rollback:** zero-value de `MetricSet` ⇒ seção omitida (igual a hoje quando contadores são zero).

## Plano de Implementação

1. `MetricSet` VO + `Merge`/`IsZero` + testes table-driven.
2. `MetricsExtractor` + implementações + `ExtractorFor`.
3. Refatorar `runEventLoop` para acumular `MetricSet` único.
4. `Summary.Metrics` + atualizar `persistence`/`evidence`.
5. `RenderMetricsSection` único e idempotente.

## Monitoramento e Validação

- Gate: `make test`; golden de report por driver (com e sem `usage`).
- Sucesso: `execution_report.md` mostra apenas campos presentes; sessões Codex/Copilot sem `usage` não emitem seção.

## Impacto em Documentação e Operação

- Atualizar `CLAUDE.md` seção "Métricas" e validador de evidence (`internal/evidence`) que detecta a seção por regex.

## Revisão Futura

- Revisitar quando Codex/Copilot expuserem `usage` próprio (criar extractor real) ou ao unificar telemetria.
