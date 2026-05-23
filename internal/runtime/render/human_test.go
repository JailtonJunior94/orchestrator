package render_test

import (
	"bytes"
	"encoding/json"
	"io"
	"testing"
	"time"

	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/events"
	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/render"
)

func newRaw(t *testing.T, v any) json.RawMessage {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("marshal raw: %v", err)
	}
	return b
}

func TestHumanRenderer_PerKind(t *testing.T) {
	t.Parallel()

	now := time.Now()

	agentMsg, err := events.NewAgentMessage(now, "Olá do agente", newRaw(t, map[string]string{"text": "Olá do agente"}))
	if err != nil {
		t.Fatal(err)
	}

	agentThought, err := events.NewAgentThought(now, "Estou pensando", newRaw(t, map[string]string{"text": "Estou pensando"}))
	if err != nil {
		t.Fatal(err)
	}

	toolStart, err := events.NewToolCallStart(now, events.NewToolCallID("tc_001"), "read_file", `{"path":"/tmp/x"}`, newRaw(t, ""))
	if err != nil {
		t.Fatal(err)
	}

	toolUpdateFinal, err := events.NewToolCallUpdate(now, events.NewToolCallID("tc_001"), "ok", "completed", true, newRaw(t, ""))
	if err != nil {
		t.Fatal(err)
	}

	toolUpdateNotFinal, err := events.NewToolCallUpdate(now, events.NewToolCallID("tc_002"), "", "running", false, newRaw(t, ""))
	if err != nil {
		t.Fatal(err)
	}

	sessionEnd, err := events.NewSessionEnd(now, 0, "end_turn", newRaw(t, ""))
	if err != nil {
		t.Fatal(err)
	}

	runtimeInit, err := events.NewRuntimeInit(now, "binary", "claude-agent-acp", nil, "v0.13.0", "", newRaw(t, ""))
	if err != nil {
		t.Fatal(err)
	}

	unknown, err := events.NewUnknown(now, "weird_kind", newRaw(t, ""))
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name string
		evt  events.Event
		want string
	}{
		{
			name: "agent_message",
			evt:  agentMsg,
			want: "[agent] Olá do agente\n",
		},
		{
			name: "agent_thought",
			evt:  agentThought,
			want: "[thought] Estou pensando\n",
		},
		{
			name: "tool_call_start",
			evt:  toolStart,
			want: "[tool] read_file tc_001\n",
		},
		{
			name: "tool_call_update_final",
			evt:  toolUpdateFinal,
			want: "[tool:done] tc_001 completed\n",
		},
		{
			name: "tool_call_update_not_final",
			evt:  toolUpdateNotFinal,
			want: "", // não final: sem output
		},
		{
			name: "session_end",
			evt:  sessionEnd,
			want: "", // sem output para session_end
		},
		{
			name: "runtime_init",
			evt:  runtimeInit,
			want: "", // sem output para runtime_init
		},
		{
			name: "unknown",
			evt:  unknown,
			want: "", // sem output para unknown
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var buf bytes.Buffer
			r := render.NewHumanRenderer(&buf)
			r.Render(tc.evt)
			if got := buf.String(); got != tc.want {
				t.Errorf("Render(%s) = %q; want %q", tc.name, got, tc.want)
			}
		})
	}
}

func TestHumanRenderer_QuietBehavior(t *testing.T) {
	t.Parallel()

	now := time.Now()
	agentMsg, err := events.NewAgentMessage(now, "Mensagem silenciosa", newRaw(t, ""))
	if err != nil {
		t.Fatal(err)
	}

	r := render.NewHumanRenderer(io.Discard)
	// Não deve entrar em pânico nem retornar erro.
	r.Render(agentMsg)
}

func TestHumanRenderer_NilWriter_NoOutput(t *testing.T) {
	t.Parallel()

	now := time.Now()
	agentMsg, err := events.NewAgentMessage(now, "Texto", newRaw(t, ""))
	if err != nil {
		t.Fatal(err)
	}

	// Simula --quiet com io.Discard.
	r := render.NewHumanRenderer(io.Discard)
	r.Render(agentMsg)
	// Se chegou aqui, passou.
}
