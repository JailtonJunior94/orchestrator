// Package events implementa o modelo de domínio de eventos do runtime ACP.
// Segue o padrão tagged union descrito em ADR-010.
package events

import (
	"encoding/json"
	"fmt"
	"time"
)

// Event é o value object central do domínio de eventos (tagged union — ADR-010).
// Exatamente um ponteiro de payload é não-nil por instância; invariante garantida
// pelos construtores New*. Nunca construa Event{} fora dos construtores ou de testes.
type Event struct {
	ts         time.Time
	kind       EventKind
	toolCallID ToolCallID
	// launcher representa o tipo de lançador: "binary" ou "npx".
	// Preenchido em runtime_init; vazio nos demais eventos.
	launcher string

	// payloads — apenas um não-nil por instância (invariante de construção)
	agentMessage   *AgentMessagePayload
	agentThought   *AgentThoughtPayload
	toolCallStart  *ToolCallStartPayload
	toolCallUpdate *ToolCallUpdatePayload
	sessionEnd     *SessionEndPayload
	runtimeInit    *RuntimeInitPayload
	unknown        *UnknownPayload

	// raw bruto do update ACP (RF-08)
	raw json.RawMessage
}

// NewAgentMessage cria um Event do kind agent_message.
func NewAgentMessage(ts time.Time, text string, raw json.RawMessage) (Event, error) {
	if text == "" {
		return Event{}, fmt.Errorf("%w: agent_message requer texto não vazio", ErrInvalidEvent)
	}
	return Event{
		ts:           ts,
		kind:         KindAgentMessage,
		agentMessage: &AgentMessagePayload{text: text},
		raw:          raw,
	}, nil
}

// NewAgentThought cria um Event do kind agent_thought.
func NewAgentThought(ts time.Time, text string, raw json.RawMessage) (Event, error) {
	if text == "" {
		return Event{}, fmt.Errorf("%w: agent_thought requer texto não vazio", ErrInvalidEvent)
	}
	return Event{
		ts:           ts,
		kind:         KindAgentThought,
		agentThought: &AgentThoughtPayload{text: text},
		raw:          raw,
	}, nil
}

// NewToolCallStart cria um Event do kind tool_call_start.
func NewToolCallStart(ts time.Time, id ToolCallID, name string, input string, raw json.RawMessage) (Event, error) {
	if name == "" {
		return Event{}, fmt.Errorf("%w: tool_call_start requer nome de ferramenta não vazio", ErrInvalidEvent)
	}
	return Event{
		ts:            ts,
		kind:          KindToolCallStart,
		toolCallID:    id,
		toolCallStart: &ToolCallStartPayload{name: name, input: input},
		raw:           raw,
	}, nil
}

// NewToolCallUpdate cria um Event do kind tool_call_update.
func NewToolCallUpdate(ts time.Time, id ToolCallID, output, status string, final bool, raw json.RawMessage) (Event, error) {
	return Event{
		ts:             ts,
		kind:           KindToolCallUpdate,
		toolCallID:     id,
		toolCallUpdate: &ToolCallUpdatePayload{output: output, status: status, final: final},
		raw:            raw,
	}, nil
}

// NewSessionEnd cria um Event do kind session_end.
func NewSessionEnd(ts time.Time, exitCode int, reason string, raw json.RawMessage) (Event, error) {
	return Event{
		ts:         ts,
		kind:       KindSessionEnd,
		sessionEnd: &SessionEndPayload{exitCode: exitCode, reason: reason},
		raw:        raw,
	}, nil
}

// NewRuntimeInit cria um Event do kind runtime_init.
// launcher deve ser "binary" ou "npx".
func NewRuntimeInit(ts time.Time, launcher, command string, args []string, sdkVersion, npmVersion string, raw json.RawMessage) (Event, error) {
	if launcher != "binary" && launcher != "npx" {
		return Event{}, fmt.Errorf("%w: launcher inválido %q; esperado \"binary\" ou \"npx\"", ErrInvalidEvent, launcher)
	}
	argsCopy := make([]string, len(args))
	copy(argsCopy, args)
	return Event{
		ts:       ts,
		kind:     KindRuntimeInit,
		launcher: launcher,
		runtimeInit: &RuntimeInitPayload{
			launcher:   launcher,
			command:    command,
			args:       argsCopy,
			sdkVersion: sdkVersion,
			npmVersion: npmVersion,
		},
		raw: raw,
	}, nil
}

// NewUnknown cria um Event do kind unknown para um kind não reconhecido.
func NewUnknown(ts time.Time, rawKind string, raw json.RawMessage) (Event, error) {
	return Event{
		ts:      ts,
		kind:    KindUnknown,
		unknown: &UnknownPayload{rawKind: rawKind},
		raw:     raw,
	}, nil
}

// Kind retorna o kind do evento.
func (e Event) Kind() EventKind { return e.kind }

// Timestamp retorna o timestamp do evento.
func (e Event) Timestamp() time.Time { return e.ts }

// ToolCallID retorna o ToolCallID do evento (zero value quando não aplicável).
func (e Event) ToolCallID() ToolCallID { return e.toolCallID }

// Launcher retorna o tipo de launcher ("binary", "npx" ou "" quando não aplicável).
func (e Event) Launcher() string { return e.launcher }

// IsTerminal retorna true quando o evento encerra a sessão.
func (e Event) IsTerminal() bool { return e.kind == KindSessionEnd }

// AgentMessage retorna o payload de agent_message ou nil se o kind não corresponder.
func (e Event) AgentMessage() *AgentMessagePayload { return e.agentMessage }

// AgentThought retorna o payload de agent_thought ou nil se o kind não corresponder.
func (e Event) AgentThought() *AgentThoughtPayload { return e.agentThought }

// ToolCallStart retorna o payload de tool_call_start ou nil se o kind não corresponder.
func (e Event) ToolCallStart() *ToolCallStartPayload { return e.toolCallStart }

// ToolCallUpdate retorna o payload de tool_call_update ou nil se o kind não corresponder.
func (e Event) ToolCallUpdate() *ToolCallUpdatePayload { return e.toolCallUpdate }

// SessionEnd retorna o payload de session_end ou nil se o kind não corresponder.
func (e Event) SessionEnd() *SessionEndPayload { return e.sessionEnd }

// RuntimeInit retorna o payload de runtime_init ou nil se o kind não corresponder.
func (e Event) RuntimeInit() *RuntimeInitPayload { return e.runtimeInit }

// Unknown retorna o payload de unknown ou nil se o kind não corresponder.
func (e Event) Unknown() *UnknownPayload { return e.unknown }

// envelopeJSON é o shape canônico do envelope RF-08 gravado no events.jsonl.
type envelopeJSON struct {
	TS         string          `json:"ts"`
	Kind       EventKind       `json:"kind"`
	ToolCallID ToolCallID      `json:"tool_call_id"`
	Launcher   string          `json:"launcher"`
	Raw        json.RawMessage `json:"raw"`
}

// MarshalJSON produz o envelope RF-08 conforme a techspec.
func (e Event) MarshalJSON() ([]byte, error) {
	env := envelopeJSON{
		TS:         e.ts.UTC().Format(time.RFC3339Nano),
		Kind:       e.kind,
		ToolCallID: e.toolCallID,
		Launcher:   e.launcher,
		Raw:        e.raw,
	}
	return json.Marshal(env)
}
