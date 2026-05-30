// Package events — extractor.go define a interface MetricsExtractor e as
// implementações por driver (ADR-021). A estratégia é selecionada por DriverID
// via ExtractorFor; domínio puro sem IO.
package events

import (
	"encoding/json"

	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/specs"
)

// MetricsExtractor extrai um MetricSet de um payload bruto ACP.
// Implementações: claudeExtractor, geminiExtractor, nullExtractor.
// Contratos:
//   - Extract nunca retorna erro; payload inválido → MetricSet{} zero-value.
//   - Extract não muta o slice raw recebido.
type MetricsExtractor interface {
	Extract(raw json.RawMessage) MetricSet
}

// ExtractorFor retorna o MetricsExtractor adequado para o driver.
// Driver desconhecido (zero-value de DriverID ou driver sem métricas) → nullExtractor.
func (c *Catalog) ExtractorFor(d specs.DriverID) MetricsExtractor {
	switch d.String() {
	case "claude":
		return claudeExtractor{}
	case "gemini":
		return geminiExtractor{}
	default:
		// codex, copilot e qualquer driver futuro → nullExtractor (conjunto mínimo zero).
		return nullExtractor{}
	}
}

// ── claudeExtractor ──────────────────────────────────────────────────────────

// claudeExtractor extrai métricas Claude-2026 de um payload ACP bruto.
// Reutiliza a lógica de ExtractClaudeMetrics preservando o contrato defensivo (F4-Claude).
type claudeExtractor struct{}

var _ MetricsExtractor = claudeExtractor{}

func (claudeExtractor) Extract(raw json.RawMessage) MetricSet {
	m := NewCatalog().ExtractClaudeMetrics(raw)
	if m.CacheReadTokens == 0 && m.CacheCreationTokens == 0 && m.ThinkingTokens == 0 {
		return MetricSet{}
	}
	extra := map[string]int{
		"cache_creation_tokens": m.CacheCreationTokens,
	}
	return NewMetricSet(0, m.CacheReadTokens, m.ThinkingTokens, extra)
}

// ── geminiExtractor ──────────────────────────────────────────────────────────

// geminiExtractor extrai métricas Gemini-2026 de um payload ACP bruto.
// Reutiliza ExtractGeminiMetrics preservando o contrato defensivo (RF-18/RF-20).
type geminiExtractor struct{}

var _ MetricsExtractor = geminiExtractor{}

func (geminiExtractor) Extract(raw json.RawMessage) MetricSet {
	m, _ := NewCatalog().ExtractGeminiMetrics(raw)
	if m.CacheReadTokens == 0 && m.EffectiveContextTokens == 0 &&
		m.PromptTokensBilled == 0 && m.ThoughtsTokens == 0 {
		return MetricSet{}
	}
	extra := map[string]int{
		"effective_context_tokens": m.EffectiveContextTokens,
		"prompt_tokens_billed":     m.PromptTokensBilled,
	}
	return NewMetricSet(0, m.CacheReadTokens, m.ThoughtsTokens, extra)
}

// ── nullExtractor ─────────────────────────────────────────────────────────────

// nullExtractor retorna MetricSet{} zero-value para qualquer payload.
// Usado para Codex, Copilot e drivers sem métricas definidas.
type nullExtractor struct{}

var _ MetricsExtractor = nullExtractor{}

func (nullExtractor) Extract(_ json.RawMessage) MetricSet {
	return MetricSet{}
}
