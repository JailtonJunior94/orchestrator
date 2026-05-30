package events

import "fmt"

// EventKind representa o tipo de um evento de runtime (enum fechado).
type EventKind string

const (
	// KindAgentMessage representa uma mensagem do agente.
	KindAgentMessage EventKind = "agent_message"
	// KindAgentThought representa um pensamento interno do agente.
	KindAgentThought EventKind = "agent_thought"
	// KindToolCallStart representa o início de uma chamada de ferramenta.
	KindToolCallStart EventKind = "tool_call_start"
	// KindToolCallUpdate representa uma atualização de chamada de ferramenta.
	KindToolCallUpdate EventKind = "tool_call_update"
	// KindSessionEnd representa o fim de uma sessão.
	KindSessionEnd EventKind = "session_end"
	// KindRuntimeInit representa a inicialização do runtime.
	KindRuntimeInit EventKind = "runtime_init"
	// KindUnknown representa um kind desconhecido.
	KindUnknown EventKind = "unknown"
)

var _validKinds = map[EventKind]struct{}{
	KindAgentMessage:   {},
	KindAgentThought:   {},
	KindToolCallStart:  {},
	KindToolCallUpdate: {},
	KindSessionEnd:     {},
	KindRuntimeInit:    {},
	KindUnknown:        {},
}

// ParseEventKind valida e retorna um EventKind a partir de uma string.
// Retorna erro se o valor não pertencer ao enum fechado.
func (c *Catalog) ParseEventKind(s string) (EventKind, error) {
	k := EventKind(s)
	if _, ok := _validKinds[k]; !ok {
		return "", fmt.Errorf("kind desconhecido: %q", s)
	}
	return k, nil
}
