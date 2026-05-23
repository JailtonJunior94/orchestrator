// Package acpfake implementa um servidor ACP fake usando o coder/acp-go-sdk no lado servidor.
// Destinado exclusivamente a testes (não importado em código de produção).
// Ver techspec §"Testes de Integração" e RF-13.
package acpfake

import (
	"time"

	acp "github.com/coder/acp-go-sdk"
)

// ScriptedUpdate representa um update ACP roteirizado com delay opcional.
type ScriptedUpdate struct {
	// Delay é o tempo de espera antes de emitir este update.
	// Zero significa emissão imediata.
	Delay time.Duration

	// Update é o SessionUpdate a ser emitido ao cliente.
	Update acp.SessionUpdate
}

// Script é uma coleção de primeira classe (OC #4) de ScriptedUpdate.
// Construída via API fluente; imutável após Start().
type Script struct {
	updates []ScriptedUpdate
}

// NewScript cria um Script vazio.
func NewScript() *Script {
	return &Script{}
}

// append adiciona um ScriptedUpdate ao script.
func (s *Script) append(delay time.Duration, u acp.SessionUpdate) *Script {
	s.updates = append(s.updates, ScriptedUpdate{Delay: delay, Update: u})
	return s
}

// AppendAgentMessage adiciona uma mensagem do agente ao script.
func (s *Script) AppendAgentMessage(text string) *Script {
	return s.append(0, acp.UpdateAgentMessageText(text))
}

// AppendAgentMessageWithDelay adiciona uma mensagem do agente com delay.
func (s *Script) AppendAgentMessageWithDelay(text string, delay time.Duration) *Script {
	return s.append(delay, acp.UpdateAgentMessageText(text))
}

// AppendAgentThought adiciona um pensamento do agente ao script.
func (s *Script) AppendAgentThought(text string) *Script {
	return s.append(0, acp.UpdateAgentThoughtText(text))
}

// AppendToolCall adiciona o início de um tool call ao script.
func (s *Script) AppendToolCall(id, name string) *Script {
	return s.append(0, acp.StartToolCall(
		acp.ToolCallId(id),
		name,
		acp.WithStartStatus(acp.ToolCallStatusPending),
	))
}

// AppendToolCallUpdate adiciona uma atualização de tool call ao script.
func (s *Script) AppendToolCallUpdate(id, status string) *Script {
	var st acp.ToolCallStatus
	switch status {
	case "completed":
		st = acp.ToolCallStatusCompleted
	case "failed":
		st = acp.ToolCallStatusFailed
	default:
		st = acp.ToolCallStatus(status)
	}
	return s.append(0, acp.UpdateToolCall(
		acp.ToolCallId(id),
		acp.WithUpdateStatus(st),
	))
}

// AppendSessionEnd sinaliza o fim da sessão ao script.
// Nota: na implementação o agent retorna PromptResponse{StopReason: StopReasonEndTurn}
// para indicar session_end; não há update explícito de session_end no protocolo.
func (s *Script) AppendSessionEnd() *Script {
	// Marcador sentinela detectado pelo agente fake.
	s.updates = append(s.updates, ScriptedUpdate{
		Delay:  0,
		Update: acp.SessionUpdate{}, // update vazio = marcador de fim
	})
	return s
}

// AppendUnknown adiciona um update desconhecido (para testar RF-05).
// Usa UserMessageChunk pois é um tipo válido mas não mapeado para events conhecidos.
func (s *Script) AppendUnknown(rawKind string, payload any) *Script {
	// UserMessageChunk é tratado como "unknown" pelo convert.FromACPUpdate.
	return s.append(0, acp.UpdateUserMessageText(rawKind))
}

// AppendRequestPermission adiciona um requestPermission ao script (RF-16).
// No fake, RequestPermission é emitido via conn.RequestPermission do AgentSideConnection.
func (s *Script) AppendRequestPermission(toolName string) *Script {
	// Marcador especial: update com Plan (não mapeado) carregando o toolName como conteúdo.
	// O agente fake interpreta isso como sinal para chamar RequestPermission.
	s.updates = append(s.updates, ScriptedUpdate{
		Delay: 0,
		Update: acp.SessionUpdate{
			Plan: &acp.SessionUpdatePlan{
				Entries: []acp.PlanEntry{{Content: "requestPermission:" + toolName}},
			},
		},
	})
	return s
}

// Updates retorna uma cópia dos updates roteirizados.
func (s *Script) Updates() []ScriptedUpdate {
	cp := make([]ScriptedUpdate, len(s.updates))
	copy(cp, s.updates)
	return cp
}
