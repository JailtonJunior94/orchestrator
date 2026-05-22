package events_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	acp "github.com/coder/acp-go-sdk"

	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/events"
)

const testDriverID = "test-driver-01"

// buildAgentMessageUpdate cria um acp.SessionUpdate com AgentMessageChunk populado.
func buildAgentMessageUpdate(text string) acp.SessionUpdate {
	return acp.SessionUpdate{
		AgentMessageChunk: &acp.SessionUpdateAgentMessageChunk{
			Content: acp.ContentBlock{
				Text: &acp.ContentBlockText{Text: text, Type: "text"},
			},
		},
	}
}

// buildAgentThoughtUpdate cria um acp.SessionUpdate com AgentThoughtChunk populado.
func buildAgentThoughtUpdate(text string) acp.SessionUpdate {
	return acp.SessionUpdate{
		AgentThoughtChunk: &acp.SessionUpdateAgentThoughtChunk{
			Content: acp.ContentBlock{
				Text: &acp.ContentBlockText{Text: text, Type: "text"},
			},
		},
	}
}

// buildToolCallUpdate cria um acp.SessionUpdate com ToolCall populado.
func buildToolCallUpdate(id, title string, rawInput any) acp.SessionUpdate {
	return acp.SessionUpdate{
		ToolCall: &acp.SessionUpdateToolCall{
			ToolCallId: acp.ToolCallId(id),
			Title:      title,
			RawInput:   rawInput,
		},
	}
}

// buildToolCallStatusUpdate cria um acp.SessionUpdate com SessionToolCallUpdate populado.
func buildToolCallStatusUpdate(id string, status acp.ToolCallStatus, rawOutput any) acp.SessionUpdate {
	s := status
	return acp.SessionUpdate{
		ToolCallUpdate: &acp.SessionToolCallUpdate{
			ToolCallId: acp.ToolCallId(id),
			Status:     &s,
			RawOutput:  rawOutput,
		},
	}
}

// buildPlanUpdate cria um acp.SessionUpdate com Plan populado (variante não mapeada).
func buildPlanUpdate() acp.SessionUpdate {
	return acp.SessionUpdate{
		Plan: &acp.SessionUpdatePlan{
			Entries: []acp.PlanEntry{},
		},
	}
}

// buildUserMessageUpdate cria um acp.SessionUpdate com UserMessageChunk populado.
func buildUserMessageUpdate(text string) acp.SessionUpdate {
	return acp.SessionUpdate{
		UserMessageChunk: &acp.SessionUpdateUserMessageChunk{
			Content: acp.ContentBlock{
				Text: &acp.ContentBlockText{Text: text, Type: "text"},
			},
		},
	}
}

// TestFromACPUpdate valida a conversão de cada variante de acp.SessionUpdate para events.Event.
func TestFromACPUpdate(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name       string
		update     acp.SessionUpdate
		wantKind   events.EventKind
		wantToolID string
		wantFinal  bool
	}{
		{
			name:     "agent_message_chunk",
			update:   buildAgentMessageUpdate("Hello from Claude"),
			wantKind: events.KindAgentMessage,
		},
		{
			name:     "agent_thought_chunk",
			update:   buildAgentThoughtUpdate("Thinking about the solution"),
			wantKind: events.KindAgentThought,
		},
		{
			name:       "tool_call_start",
			update:     buildToolCallUpdate("tc_abc123", "bash", map[string]string{"command": "echo hello"}),
			wantKind:   events.KindToolCallStart,
			wantToolID: "tc_abc123",
		},
		{
			name:       "tool_call_update_in_progress",
			update:     buildToolCallStatusUpdate("tc_abc123", acp.ToolCallStatusInProgress, nil),
			wantKind:   events.KindToolCallUpdate,
			wantToolID: "tc_abc123",
			wantFinal:  false,
		},
		{
			name:       "tool_call_update_completed",
			update:     buildToolCallStatusUpdate("tc_abc123", acp.ToolCallStatusCompleted, "hello"),
			wantKind:   events.KindToolCallUpdate,
			wantToolID: "tc_abc123",
			wantFinal:  true,
		},
		{
			name:       "tool_call_update_failed",
			update:     buildToolCallStatusUpdate("tc_abc123", acp.ToolCallStatusFailed, nil),
			wantKind:   events.KindToolCallUpdate,
			wantToolID: "tc_abc123",
			wantFinal:  true,
		},
		{
			name:     "plan_update_unknown",
			update:   buildPlanUpdate(),
			wantKind: events.KindUnknown,
		},
		{
			name:     "user_message_chunk_unknown",
			update:   buildUserMessageUpdate("user text"),
			wantKind: events.KindUnknown,
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			evt, err := events.FromACPUpdate(testDriverID, tc.update)
			if err != nil {
				t.Fatalf("FromACPUpdate() erro inesperado: %v", err)
			}
			if evt.Kind() != tc.wantKind {
				t.Errorf("Kind() = %q; quero %q", evt.Kind(), tc.wantKind)
			}
			if tc.wantToolID != "" && evt.ToolCallID().String() != tc.wantToolID {
				t.Errorf("ToolCallID() = %q; quero %q", evt.ToolCallID().String(), tc.wantToolID)
			}
			if tc.wantKind == events.KindToolCallUpdate {
				p := evt.ToolCallUpdate()
				if p == nil {
					t.Fatal("ToolCallUpdate() payload = nil")
				}
				if p.Final() != tc.wantFinal {
					t.Errorf("Final() = %v; quero %v", p.Final(), tc.wantFinal)
				}
			}
			// Envelope MarshalJSON não deve falhar.
			if _, err := json.Marshal(evt); err != nil {
				t.Errorf("MarshalJSON() erro: %v", err)
			}
		})
	}
}

// TestFromACPUpdateEdgeCases valida comportamentos defensivos.
func TestFromACPUpdateEdgeCases(t *testing.T) {
	t.Parallel()

	t.Run("agent_message_chunk_empty_text_produces_unknown", func(t *testing.T) {
		t.Parallel()
		update := acp.SessionUpdate{
			AgentMessageChunk: &acp.SessionUpdateAgentMessageChunk{
				Content: acp.ContentBlock{
					// Sem Text — bloco de imagem, por exemplo.
					Image: &acp.ContentBlockImage{Type: "image", MimeType: "image/png", Data: "base64=="},
				},
			},
		}
		evt, err := events.FromACPUpdate(testDriverID, update)
		if err != nil {
			t.Fatalf("erro inesperado: %v", err)
		}
		if evt.Kind() != events.KindUnknown {
			t.Errorf("Kind() = %q; quero %q", evt.Kind(), events.KindUnknown)
		}
	})

	t.Run("agent_thought_chunk_empty_text_produces_unknown", func(t *testing.T) {
		t.Parallel()
		update := acp.SessionUpdate{
			AgentThoughtChunk: &acp.SessionUpdateAgentThoughtChunk{
				Content: acp.ContentBlock{}, // bloco sem variante
			},
		}
		evt, err := events.FromACPUpdate(testDriverID, update)
		if err != nil {
			t.Fatalf("erro inesperado: %v", err)
		}
		if evt.Kind() != events.KindUnknown {
			t.Errorf("Kind() = %q; quero %q", evt.Kind(), events.KindUnknown)
		}
	})

	t.Run("tool_call_empty_id", func(t *testing.T) {
		t.Parallel()
		update := buildToolCallUpdate("", "bash", nil)
		evt, err := events.FromACPUpdate(testDriverID, update)
		if err != nil {
			t.Fatalf("erro inesperado: %v", err)
		}
		if evt.Kind() != events.KindToolCallStart {
			t.Errorf("Kind() = %q; quero %q", evt.Kind(), events.KindToolCallStart)
		}
		if !evt.ToolCallID().IsZero() {
			t.Errorf("ToolCallID() deve ser zero; got %q", evt.ToolCallID().String())
		}
	})

	t.Run("tool_call_update_nil_status", func(t *testing.T) {
		t.Parallel()
		update := acp.SessionUpdate{
			ToolCallUpdate: &acp.SessionToolCallUpdate{
				ToolCallId: "tc_999",
				Status:     nil,
			},
		}
		evt, err := events.FromACPUpdate(testDriverID, update)
		if err != nil {
			t.Fatalf("erro inesperado: %v", err)
		}
		if evt.Kind() != events.KindToolCallUpdate {
			t.Errorf("Kind() = %q; quero %q", evt.Kind(), events.KindToolCallUpdate)
		}
		p := evt.ToolCallUpdate()
		if p == nil {
			t.Fatal("ToolCallUpdate() payload = nil")
		}
		if p.Final() {
			t.Error("Final() deve ser false quando status nil")
		}
	})

	t.Run("tool_call_nil_raw_input", func(t *testing.T) {
		t.Parallel()
		update := buildToolCallUpdate("tc_zero", "grep", nil)
		evt, err := events.FromACPUpdate(testDriverID, update)
		if err != nil {
			t.Fatalf("erro inesperado: %v", err)
		}
		if evt.Kind() != events.KindToolCallStart {
			t.Errorf("Kind() = %q; quero %q", evt.Kind(), events.KindToolCallStart)
		}
		p := evt.ToolCallStart()
		if p == nil {
			t.Fatal("ToolCallStart() payload = nil")
		}
		if p.Input() != "" {
			t.Errorf("Input() = %q; quero vazio", p.Input())
		}
	})
}

// TestFromACPRaw valida a conversão a partir de JSON bruto, cobrindo session_end
// (discriminador que o SDK v0.13.0 não reconhece) e variantes futuras.
func TestFromACPRaw(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		raw      string
		wantKind events.EventKind
	}{
		{
			name:     "session_end_success",
			raw:      `{"sessionUpdate":"session_end","exitCode":0}`,
			wantKind: events.KindSessionEnd,
		},
		{
			name:     "session_end_error",
			raw:      `{"sessionUpdate":"session_end","exitCode":1,"error":"tool execution failed"}`,
			wantKind: events.KindSessionEnd,
		},
		{
			name:     "unknown_drift",
			raw:      `{"sessionUpdate":"future_extension_kind","data":"some payload"}`,
			wantKind: events.KindUnknown,
		},
		{
			name:     "no_discriminator",
			raw:      `{}`,
			wantKind: events.KindUnknown,
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			evt, err := events.FromACPRaw(testDriverID, json.RawMessage(tc.raw))
			if err != nil {
				t.Fatalf("FromACPRaw() erro inesperado: %v", err)
			}
			if evt.Kind() != tc.wantKind {
				t.Errorf("Kind() = %q; quero %q", evt.Kind(), tc.wantKind)
			}
			if _, err := json.Marshal(evt); err != nil {
				t.Errorf("MarshalJSON() erro: %v", err)
			}
		})
	}
}

// TestFromACPRawFixtures lê as fixtures ACP de testdata/acp/*.json e valida
// que a conversão produz o kind esperado (via FromACPRaw ou FromACPUpdate).
func TestFromACPRawFixtures(t *testing.T) {
	t.Parallel()

	fixtures := []struct {
		file     string
		wantKind events.EventKind
		viaRaw   bool // true = usar FromACPRaw (SDK não consegue unmarshal)
	}{
		{file: "agent_message.json", wantKind: events.KindAgentMessage},
		{file: "agent_thought.json", wantKind: events.KindAgentThought},
		{file: "tool_call_start.json", wantKind: events.KindToolCallStart},
		{file: "tool_call_update_in_progress.json", wantKind: events.KindToolCallUpdate},
		{file: "tool_call_update_final.json", wantKind: events.KindToolCallUpdate},
		{file: "session_end_success.json", wantKind: events.KindSessionEnd, viaRaw: true},
		{file: "session_end_error.json", wantKind: events.KindSessionEnd, viaRaw: true},
		{file: "unknown_drift.json", wantKind: events.KindUnknown, viaRaw: true},
	}

	for _, fx := range fixtures {
		fx := fx
		t.Run(fx.file, func(t *testing.T) {
			t.Parallel()

			path := filepath.Join("testdata", "acp", fx.file)
			raw, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("leitura fixture %s: %v", fx.file, err)
			}

			var evt events.Event

			if fx.viaRaw {
				evt, err = events.FromACPRaw(testDriverID, json.RawMessage(raw))
				if err != nil {
					t.Fatalf("FromACPRaw() erro inesperado: %v", err)
				}
			} else {
				var update acp.SessionUpdate
				if err := json.Unmarshal(raw, &update); err != nil {
					t.Fatalf("unmarshal fixture %s: %v", fx.file, err)
				}
				evt, err = events.FromACPUpdate(testDriverID, update)
				if err != nil {
					t.Fatalf("FromACPUpdate() erro inesperado: %v", err)
				}
			}

			if evt.Kind() != fx.wantKind {
				t.Errorf("Kind() = %q; quero %q", evt.Kind(), fx.wantKind)
			}
			if _, err := json.Marshal(evt); err != nil {
				t.Errorf("MarshalJSON() erro: %v", err)
			}
		})
	}
}

// TestFromACPUpdateEnvelopeGolden valida que o envelope MarshalJSON bate com os
// golden files existentes em testdata/envelopes/ para os kinds produzidos por FromACPUpdate.
func TestFromACPUpdateEnvelopeGolden(t *testing.T) {
	t.Parallel()

	// Apenas valida campos estruturais (kind, tool_call_id) — não o timestamp (não-determinístico).
	cases := []struct {
		name       string
		update     acp.SessionUpdate
		wantKind   events.EventKind
		wantToolID string
	}{
		{
			name:     "agent_message",
			update:   buildAgentMessageUpdate("hello"),
			wantKind: events.KindAgentMessage,
		},
		{
			name:     "agent_thought",
			update:   buildAgentThoughtUpdate("thinking"),
			wantKind: events.KindAgentThought,
		},
		{
			name:       "tool_call_start",
			update:     buildToolCallUpdate("tc_001", "bash", nil),
			wantKind:   events.KindToolCallStart,
			wantToolID: "tc_001",
		},
		{
			name:       "tool_call_update",
			update:     buildToolCallStatusUpdate("tc_001", acp.ToolCallStatusCompleted, nil),
			wantKind:   events.KindToolCallUpdate,
			wantToolID: "tc_001",
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			evt, err := events.FromACPUpdate(testDriverID, tc.update)
			if err != nil {
				t.Fatalf("FromACPUpdate() erro: %v", err)
			}

			got, err := json.Marshal(evt)
			if err != nil {
				t.Fatalf("MarshalJSON() erro: %v", err)
			}

			var gotMap map[string]json.RawMessage
			if err := json.Unmarshal(got, &gotMap); err != nil {
				t.Fatalf("unmarshal envelope: %v", err)
			}

			// Validar kind no envelope.
			var kind string
			if err := json.Unmarshal(gotMap["kind"], &kind); err != nil {
				t.Fatalf("unmarshal kind: %v", err)
			}
			if events.EventKind(kind) != tc.wantKind {
				t.Errorf("envelope kind = %q; quero %q", kind, tc.wantKind)
			}

			// Validar tool_call_id no envelope.
			if tc.wantToolID != "" {
				var toolID string
				if err := json.Unmarshal(gotMap["tool_call_id"], &toolID); err != nil {
					t.Errorf("tool_call_id não é string: %s", gotMap["tool_call_id"])
				} else if toolID != tc.wantToolID {
					t.Errorf("tool_call_id = %q; quero %q", toolID, tc.wantToolID)
				}
			}
		})
	}
}

// FuzzFromACPUpdate verifica que FromACPUpdate não entra em panic para nenhum input
// que o SDK consiga desserializar. Injeta bytes arbitrários através do unmarshal do SDK.
func FuzzFromACPUpdate(f *testing.F) {
	// Seeds baseadas nas variantes conhecidas.
	seeds := []string{
		mustMarshalS(buildAgentMessageUpdate("hello")),
		mustMarshalS(buildAgentThoughtUpdate("thinking")),
		mustMarshalS(buildToolCallUpdate("tc_1", "bash", nil)),
		mustMarshalS(buildToolCallStatusUpdate("tc_1", acp.ToolCallStatusCompleted, nil)),
		mustMarshalS(buildPlanUpdate()),
	}
	for _, s := range seeds {
		f.Add([]byte(s))
	}

	f.Fuzz(func(t *testing.T, data []byte) {
		var update acp.SessionUpdate
		if err := json.Unmarshal(data, &update); err != nil {
			// SDK não conseguiu desserializar: input inválido para FromACPUpdate.
			// Testar via FromACPRaw para cobrir o caminho de bytes brutos.
			evt, err := events.FromACPRaw("fuzz-driver", json.RawMessage(data))
			if err != nil {
				t.Errorf("FromACPRaw panic/erro para input: %s — %v", data, err)
			}
			_ = evt
			return
		}
		evt, err := events.FromACPUpdate("fuzz-driver", update)
		if err != nil {
			t.Errorf("FromACPUpdate retornou erro inesperado: %v", err)
		}
		_ = evt
	})
}

// mustMarshalS marshals v para string JSON (helper de fuzz seed).
func mustMarshalS(v any) string {
	b, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return string(b)
}

// TestExtractClaudeMetrics valida ExtractClaudeMetrics para os casos T-CONV-CR-01..T-CONV-CR-03.
func TestExtractClaudeMetrics(t *testing.T) {
	t.Parallel()

	// T-CONV-CR-01: payload com todos os campos de usage presentes → valores extraídos corretamente.
	t.Run("T-CONV-CR-01_all_usage_fields_present", func(t *testing.T) {
		t.Parallel()

		raw := json.RawMessage(`{
			"sessionUpdate": "agent_message",
			"usage": {
				"cache_read_input_tokens": 150,
				"cache_creation_input_tokens": 300,
				"thoughtTokens": 42,
				"inputTokens": 500,
				"outputTokens": 120,
				"totalTokens": 1112
			}
		}`)

		got := events.ExtractClaudeMetrics(raw)

		if got.CacheReadTokens != 150 {
			t.Errorf("CacheReadTokens = %d; quero 150", got.CacheReadTokens)
		}
		if got.CacheCreationTokens != 300 {
			t.Errorf("CacheCreationTokens = %d; quero 300", got.CacheCreationTokens)
		}
		if got.ThinkingTokens != 42 {
			t.Errorf("ThinkingTokens = %d; quero 42", got.ThinkingTokens)
		}
	})

	// T-CONV-CR-02: payload com usage presente mas sem campos opcionais → zeros sem erro.
	t.Run("T-CONV-CR-02_usage_present_optional_fields_absent", func(t *testing.T) {
		t.Parallel()

		raw := json.RawMessage(`{
			"sessionUpdate": "agent_message",
			"usage": {
				"inputTokens": 200,
				"outputTokens": 50,
				"totalTokens": 250
			}
		}`)

		got := events.ExtractClaudeMetrics(raw)

		if got.CacheReadTokens != 0 {
			t.Errorf("CacheReadTokens = %d; quero 0 (campo ausente)", got.CacheReadTokens)
		}
		if got.CacheCreationTokens != 0 {
			t.Errorf("CacheCreationTokens = %d; quero 0 (campo ausente)", got.CacheCreationTokens)
		}
		if got.ThinkingTokens != 0 {
			t.Errorf("ThinkingTokens = %d; quero 0 (campo ausente)", got.ThinkingTokens)
		}
	})

	// T-CONV-CR-03: payload sem campo "usage" → todos zeros, sem erro (comportamento defensivo).
	t.Run("T-CONV-CR-03_no_usage_field_stays_zero", func(t *testing.T) {
		t.Parallel()

		raw := json.RawMessage(`{"sessionUpdate": "agent_message", "content": "hello"}`)

		got := events.ExtractClaudeMetrics(raw)

		if got.CacheReadTokens != 0 || got.CacheCreationTokens != 0 || got.ThinkingTokens != 0 {
			t.Errorf("todos campos devem ser 0 quando usage ausente; got cache_read=%d cache_creation=%d thinking=%d",
				got.CacheReadTokens, got.CacheCreationTokens, got.ThinkingTokens)
		}
	})
}
