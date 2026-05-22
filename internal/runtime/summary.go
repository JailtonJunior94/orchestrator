// Package runtime implementa o application service ACPRunner e tipos relacionados.
package runtime

import "github.com/JailtonJunior94/ai-spec-harness/internal/runtime/events"

// Summary é o resumo imutável produzido ao final de uma sessão ACP.
// Consumido por persistence.EnrichReport e pela camada de telemetria.
type Summary struct {
	// Launcher é o tipo de launcher usado: "binary" ou "npx".
	Launcher string
	// EventsCount é o número de eventos mapeados recebidos (exclui unknown).
	EventsCount int
	// UnknownEventsCount é o número de eventos de kind desconhecido.
	UnknownEventsCount int
	// CancelReason é o motivo de cancelamento (none quando sessão encerrou normalmente).
	CancelReason events.CancelReason
	// ToolCalls é o snapshot dos tool calls registrados durante a sessão.
	ToolCalls []events.ToolCallSummary
	// UnknownKinds são os raw kinds desconhecidos encontrados, deduplicados.
	UnknownKinds []string

	// Contadores de backpressure do canal ACP (ADR-018, RF-03).
	// Populados a partir dos contadores atômicos de acpClient ao final da sessão.
	// Default 0 quando publishTimeout=0 e o canal nunca ficou cheio (F1 default).

	// SlowPublishes é o número de publicações que aguardaram publishTimeout>0 antes
	// de entregar o evento ao canal. Indica pressão no canal durante a sessão.
	SlowPublishes uint64 `json:"slow_publishes,omitempty"`
	// DroppedUpdates é o número de eventos descartados por canal cheio.
	// Com timeout=0 (F1 default), qualquer canal cheio resulta em descarte imediato.
	DroppedUpdates uint64 `json:"dropped_updates,omitempty"`

	// F5-Claude: campos de auto-review (populados apenas quando AutoReview=true).
	// ReviewStatus é "" quando auto-review não executou, "ok" ou "blocked" caso contrário.
	ReviewStatus string
	// ReviewPath é o caminho de evidence/<task>/review.md (apontador conveniente).
	ReviewPath string

	// Campos Claude-2026 — opcionais (default 0). Populados por extractClaudeMetrics (F4-Claude).
	// Ausência de campos no payload ACP mantém os valores em 0 sem erro (comportamento defensivo).

	// CacheReadTokens é o total acumulado de tokens lidos do cache ao longo da sessão.
	// Fonte: campo "usage.cache_read_input_tokens" no payload JSON do update ACP.
	CacheReadTokens int `json:"cache_read_tokens,omitempty"`
	// CacheCreationTokens é o total acumulado de tokens escritos no cache (cache-warming).
	// Fonte: campo "usage.cache_creation_input_tokens" no payload JSON do update ACP.
	CacheCreationTokens int `json:"cache_creation_tokens,omitempty"`
	// ThinkingTokens é o total acumulado de tokens de raciocínio (extended thinking).
	// Fonte: acp.Usage.ThoughtTokens (campo "thoughtTokens") no payload JSON do update ACP.
	ThinkingTokens int `json:"thinking_tokens,omitempty"`
	// ToolCallsNormalizedCount é o número de tool-calls normalizadas acumuladas.
	// Incrementado por task 3.0 (counters.Record) — esta task garante persistência no report.
	ToolCallsNormalizedCount int `json:"tool_calls_normalized_count,omitempty"`

	// Métricas Gemini-2026 (F4-Gemini, RF-19) — opcionais (default 0).
	// Campos com omitempty garantem que sessões Claude/Codex/Copilot não emitam ruído no JSON.
	// Persistência (internal/runtime/persistence/) continua agnóstica — apenas serializa o Summary recebido.

	// GeminiCacheReadTokens é o total acumulado de tokens lidos do cache Gemini ao longo da sessão.
	GeminiCacheReadTokens int `json:"gemini_cache_read_tokens,omitempty"`
	// GeminiEffectiveContextTokens é o total acumulado de tokens de contexto efetivo Gemini.
	GeminiEffectiveContextTokens int `json:"gemini_effective_context_tokens,omitempty"`
	// GeminiPromptTokensBilled é o total acumulado de tokens de prompt faturados pelo Gemini.
	GeminiPromptTokensBilled int `json:"gemini_prompt_tokens_billed,omitempty"`
	// GeminiThoughtsTokens é o total acumulado de tokens de raciocínio (thoughts) do Gemini 2.5.
	// Pode ser sempre zero quando thoughts não estão expostos por default (caveat Q8 do PRD).
	GeminiThoughtsTokens int `json:"gemini_thoughts_tokens,omitempty"`

	// RetryAttempts é o número de tentativas extras realizadas pelo loop de retry (ADR-018, RF-04).
	// 0 = sessão bem-sucedida na primeira tentativa (comportamento F1 default).
	// Incrementado pelo acpInvoker para cada reexecução após falha transitória.
	RetryAttempts int `json:"retry_attempts,omitempty"`
}
