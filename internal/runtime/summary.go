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

	// Metrics é o conjunto unificado de métricas da sessão (ADR-021).
	// Substitui os campos planos Claude/Gemini por um único MetricSet por driver.
	// Zero-value (IsZero()==true) preserva comportamento F1 — nenhuma seção de métricas emitida.
	Metrics events.MetricSet

	// ToolCallsNormalizedCount é o número de tool-calls normalizadas acumuladas.
	// Incrementado por task 3.0 (counters.Record) — esta task garante persistência no report.
	ToolCallsNormalizedCount int `json:"tool_calls_normalized_count,omitempty"`

	// RetryAttempts é o número de tentativas extras realizadas pelo loop de retry (ADR-018, RF-04).
	// 0 = sessão bem-sucedida na primeira tentativa (comportamento F1 default).
	// Incrementado pelo acpInvoker para cada reexecução após falha transitória.
	RetryAttempts int `json:"retry_attempts,omitempty"`
}
