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
}
