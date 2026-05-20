// Package events — convert.go implementa a tradução de acp.SessionUpdate para events.Event.
// Esta é a ÚNICA camada autorizada a importar github.com/coder/acp-go-sdk além de
// internal/runtime/client/. Regra R-DDD-001: domínio não importa infra, mas a
// fronteira de conversão pode tocar o SDK como tipo de entrada sem expô-lo na borda pública.
package events

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	acp "github.com/coder/acp-go-sdk"
)

// rawDiscriminator lê o campo "sessionUpdate" do JSON bruto sem unmarshal completo.
// Retorna "" quando o campo não existe ou não é string.
func rawDiscriminator(b json.RawMessage) string {
	var m map[string]json.RawMessage
	if err := json.Unmarshal(b, &m); err != nil {
		return ""
	}
	v, ok := m["sessionUpdate"]
	if !ok {
		return ""
	}
	var disc string
	if err := json.Unmarshal(v, &disc); err != nil {
		return ""
	}
	return disc
}

// extractExitCode tenta extrair o campo "exitCode" do JSON bruto.
// Retorna 0 quando ausente ou inválido.
func extractExitCode(b json.RawMessage) int {
	var m map[string]json.RawMessage
	if err := json.Unmarshal(b, &m); err != nil {
		return 0
	}
	v, ok := m["exitCode"]
	if !ok {
		return 0
	}
	var code int
	if err := json.Unmarshal(v, &code); err != nil {
		return 0
	}
	return code
}

// extractErrorReason tenta extrair os campos "error" ou "reason" do JSON bruto.
func extractErrorReason(b json.RawMessage) string {
	var m map[string]json.RawMessage
	if err := json.Unmarshal(b, &m); err != nil {
		return ""
	}
	for _, key := range []string{"error", "reason"} {
		if v, ok := m[key]; ok {
			var s string
			if err := json.Unmarshal(v, &s); err == nil {
				return s
			}
		}
	}
	return ""
}

// contentBlockText extrai o texto de um acp.ContentBlock.
// Retorna "" quando o bloco não contém texto (image, audio, resource, etc.).
func contentBlockText(cb acp.ContentBlock) string {
	if cb.Text != nil {
		return cb.Text.Text
	}
	return ""
}

// isFinalStatus retorna true quando o ToolCallStatus indica conclusão da chamada.
func isFinalStatus(s acp.ToolCallStatus) bool {
	return s == acp.ToolCallStatusCompleted || s == acp.ToolCallStatusFailed
}

// FromACPRaw traduz bytes JSON brutos de um update ACP para events.Event.
// É chamado quando o SDK não consegue fazer unmarshal (discriminadores futuros como
// "session_end" que o SDK v0.13.0 não conhece), OU quando o chamador tem acesso
// direto ao JSON antes do parse.
//
// Nunca retorna erro — variantes desconhecidas viram events.NewUnknown.
func FromACPRaw(driverID string, raw json.RawMessage) (Event, error) {
	now := time.Now().UTC()

	disc := rawDiscriminator(raw)
	switch disc {
	case "session_end":
		exitCode := extractExitCode(raw)
		reason := extractErrorReason(raw)
		return NewSessionEnd(now, exitCode, reason, raw)

	case "":
		return NewUnknown(now, "acp.SessionUpdate", raw)

	default:
		return NewUnknown(now, disc, raw)
	}
}

// FromACPUpdate traduz um acp.SessionUpdate (tipo externo do coder/acp-go-sdk) para o
// tipo de domínio events.Event. É a função principal de conversão do pipeline ACP.
//
// Regras:
//   - Variantes reconhecidas pelo SDK são mapeadas para os construtores events.New*.
//   - Variantes não mapeadas produzem events.NewUnknown sem erro (RF-05).
//   - Payloads nil são tratados defensivamente; nunca há panic para input válido.
//   - O campo raw é preenchido com o JSON serializado do update (RF-08).
//
// driverID é o identificador da sessão (correlação em logs); reservado para uso futuro.
func FromACPUpdate(driverID string, update acp.SessionUpdate) (Event, error) {
	now := time.Now().UTC()

	// Serializa o update para raw (RF-08: "raw: <acp.SessionUpdate JSON inteiro>").
	// Erros de marshal são não-críticos: raw fica nil, envelope continua válido.
	raw, _ := json.Marshal(update)

	// Discriminador extraído do JSON serializado para diagnóstico em unknowns.
	disc := rawDiscriminator(raw)

	// --- AgentMessageChunk → agent_message ---
	if update.AgentMessageChunk != nil {
		text := contentBlockText(update.AgentMessageChunk.Content)
		if text == "" {
			// Chunk de texto vazio: não é um evento de mensagem válido.
			// RF-05: produzir unknown sem erro.
			return NewUnknown(now, "agent_message_chunk_empty", raw)
		}
		return NewAgentMessage(now, text, raw)
	}

	// --- AgentThoughtChunk → agent_thought ---
	if update.AgentThoughtChunk != nil {
		text := contentBlockText(update.AgentThoughtChunk.Content)
		if text == "" {
			return NewUnknown(now, "agent_thought_chunk_empty", raw)
		}
		return NewAgentThought(now, text, raw)
	}

	// --- ToolCall → tool_call_start ---
	if update.ToolCall != nil {
		tc := update.ToolCall
		id := NewToolCallID(string(tc.ToolCallId))
		name := tc.Title
		if name == "" {
			// Fallback: gerar nome genérico a partir do ID (não deve ocorrer em prod).
			name = fmt.Sprintf("tool_%s", strings.TrimSpace(string(tc.ToolCallId)))
		}
		var inputStr string
		if tc.RawInput != nil {
			if b, err := json.Marshal(tc.RawInput); err == nil {
				inputStr = string(b)
			}
		}
		return NewToolCallStart(now, id, name, inputStr, raw)
	}

	// --- ToolCallUpdate → tool_call_update ---
	if update.ToolCallUpdate != nil {
		tcu := update.ToolCallUpdate
		id := NewToolCallID(string(tcu.ToolCallId))

		var status acp.ToolCallStatus
		if tcu.Status != nil {
			status = *tcu.Status
		}
		final := isFinalStatus(status)

		var outputStr string
		if tcu.RawOutput != nil {
			if b, err := json.Marshal(tcu.RawOutput); err == nil {
				outputStr = string(b)
			}
		}
		return NewToolCallUpdate(now, id, outputStr, string(status), final, raw)
	}

	// --- UserMessageChunk: update do usuário, não do agente → unknown ---
	if update.UserMessageChunk != nil {
		return NewUnknown(now, "user_message_chunk", raw)
	}

	// --- Variantes não mapeadas (Plan, AvailableCommandsUpdate, CurrentModeUpdate,
	//     ConfigOptionUpdate, SessionInfoUpdate, UsageUpdate) → unknown (RF-05) ---
	rawKind := disc
	if rawKind == "" {
		rawKind = "acp.SessionUpdate"
	}
	return NewUnknown(now, rawKind, raw)
}
