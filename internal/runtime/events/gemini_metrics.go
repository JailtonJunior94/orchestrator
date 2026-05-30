// Package events — gemini_metrics.go captura métricas Gemini-2026 do payload
// acp.SessionUpdate e expõe extração defensiva alinhada com TD-02 (schema defensivo).
package events

import (
	"context"
	"encoding/json"

	"github.com/JailtonJunior94/ai-spec-harness/internal/telemetry"
)

// GeminiMetrics captura métricas Gemini-2026 do payload acp.SessionUpdate.
// Todos os campos são opcionais — ausência retorna zero-value silencioso (RF-18).
//
// Diferenças vs Claude-2026 (compozy-adaptation-gemini-2026.md §"Mecânica"):
//   - cache_creation_tokens não existe em Gemini (modelo de cache implícito)
//   - thoughts_tokens (Gemini) ↔ thinking_tokens (Claude) — semântica distinta
//   - prompt_tokens_billed e effective_context_tokens são exclusivos de Gemini
type GeminiMetrics struct {
	CacheReadTokens        int `json:"cache_read_tokens,omitempty"`
	EffectiveContextTokens int `json:"effective_context_tokens,omitempty"`
	PromptTokensBilled     int `json:"prompt_tokens_billed,omitempty"`
	ThoughtsTokens         int `json:"thoughts_tokens,omitempty"`
}

// geminiMetricsEnvelope é o shape mínimo do JSON para extração de usage Gemini.
type geminiMetricsEnvelope struct {
	Usage GeminiMetrics `json:"usage"`
}

// ExtractGeminiMetrics lê os campos Gemini-específicos do raw payload do
// acp.SessionUpdate. Retorna GeminiMetrics{} zero-value quando o payload não
// contém os campos (silencioso, alinhado com TD-02 — schema defensivo).
//
// Chamada por internal/runtime/events/convert.go quando driver_id == "gemini".
// Nunca retorna erro para campos ausentes; só falha em JSON syntactically inválido.
func (c *Catalog) ExtractGeminiMetrics(raw json.RawMessage) (GeminiMetrics, error) {
	if len(raw) == 0 {
		return GeminiMetrics{}, nil
	}
	var envelope geminiMetricsEnvelope
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return GeminiMetrics{}, err
	}
	return envelope.Usage, nil
}

// LogGeminiMetrics escreve no telemetria opt-in (GOVERNANCE_TELEMETRY=1)
// entries com prefixo "gemini." quando os campos são não-zero. Espelha
// LogClaudeMetrics em events/metrics.go (via telemetry package).
//
// ctx é reservado para expansão futura (cancelamento, trace); não usado atualmente.
func (c *Catalog) LogGeminiMetrics(ctx context.Context, rootDir string, m GeminiMetrics) error {
	_ = ctx
	sm := telemetry.GeminiSessionMetrics{
		CacheReadTokens:        m.CacheReadTokens,
		EffectiveContextTokens: m.EffectiveContextTokens,
		PromptTokensBilled:     m.PromptTokensBilled,
		ThoughtsTokens:         m.ThoughtsTokens,
	}
	return telemetry.NewCatalog().LogGeminiMetrics(rootDir, sm)
}
