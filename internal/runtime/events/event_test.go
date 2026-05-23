package events_test

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/events"
)

// fixedTS é o timestamp fixo usado em todos os testes de golden file.
var fixedTS = time.Date(2026, 5, 20, 12, 0, 0, 0, time.UTC)

func TestNewAgentMessage(t *testing.T) {
	t.Parallel()

	t.Run("válido", func(t *testing.T) {
		t.Parallel()
		evt, err := events.NewAgentMessage(fixedTS, "hello", json.RawMessage(`{"text":"hello"}`))
		if err != nil {
			t.Fatalf("NewAgentMessage erro inesperado: %v", err)
		}
		if evt.Kind() != events.KindAgentMessage {
			t.Errorf("Kind() = %q; quero %q", evt.Kind(), events.KindAgentMessage)
		}
		if evt.AgentMessage() == nil {
			t.Fatal("AgentMessage() = nil; esperava payload")
		}
		if evt.AgentMessage().Text() != "hello" {
			t.Errorf("Text() = %q; quero %q", evt.AgentMessage().Text(), "hello")
		}
		if !evt.Timestamp().Equal(fixedTS) {
			t.Errorf("Timestamp() = %v; quero %v", evt.Timestamp(), fixedTS)
		}
		if evt.AgentThought() != nil || evt.ToolCallStart() != nil {
			t.Error("payloads de outros kinds devem ser nil")
		}
	})

	t.Run("texto vazio retorna erro", func(t *testing.T) {
		t.Parallel()
		_, err := events.NewAgentMessage(fixedTS, "", nil)
		if err == nil {
			t.Fatal("esperava erro; obteve nil")
		}
		if !errors.Is(err, events.ErrInvalidEvent) {
			t.Errorf("erro = %v; quero ErrInvalidEvent", err)
		}
	})
}

func TestNewAgentThought(t *testing.T) {
	t.Parallel()

	t.Run("válido", func(t *testing.T) {
		t.Parallel()
		evt, err := events.NewAgentThought(fixedTS, "thinking", json.RawMessage(`{"text":"thinking"}`))
		if err != nil {
			t.Fatalf("NewAgentThought erro inesperado: %v", err)
		}
		if evt.Kind() != events.KindAgentThought {
			t.Errorf("Kind() = %q; quero %q", evt.Kind(), events.KindAgentThought)
		}
		if evt.AgentThought() == nil || evt.AgentThought().Text() != "thinking" {
			t.Error("AgentThought payload incorreto")
		}
	})

	t.Run("texto vazio retorna erro", func(t *testing.T) {
		t.Parallel()
		_, err := events.NewAgentThought(fixedTS, "", nil)
		if err == nil {
			t.Fatal("esperava erro; obteve nil")
		}
	})
}

func TestNewToolCallStart(t *testing.T) {
	t.Parallel()

	id := events.NewToolCallID("tc_001")

	t.Run("válido", func(t *testing.T) {
		t.Parallel()
		evt, err := events.NewToolCallStart(fixedTS, id, "bash", "echo hi", json.RawMessage(`{"name":"bash"}`))
		if err != nil {
			t.Fatalf("NewToolCallStart erro inesperado: %v", err)
		}
		if evt.Kind() != events.KindToolCallStart {
			t.Errorf("Kind() = %q; quero %q", evt.Kind(), events.KindToolCallStart)
		}
		if evt.ToolCallID().String() != "tc_001" {
			t.Errorf("ToolCallID() = %q; quero %q", evt.ToolCallID(), "tc_001")
		}
		if evt.ToolCallStart() == nil || evt.ToolCallStart().Name() != "bash" {
			t.Error("ToolCallStart payload incorreto")
		}
	})

	t.Run("nome vazio retorna erro", func(t *testing.T) {
		t.Parallel()
		_, err := events.NewToolCallStart(fixedTS, id, "", "", nil)
		if err == nil {
			t.Fatal("esperava erro; obteve nil")
		}
	})
}

func TestNewToolCallUpdate(t *testing.T) {
	t.Parallel()
	id := events.NewToolCallID("tc_001")

	evt, err := events.NewToolCallUpdate(fixedTS, id, "out", "running", false, json.RawMessage(`{"output":"done"}`))
	if err != nil {
		t.Fatalf("NewToolCallUpdate erro inesperado: %v", err)
	}
	if evt.Kind() != events.KindToolCallUpdate {
		t.Errorf("Kind() = %q", evt.Kind())
	}
	p := evt.ToolCallUpdate()
	if p == nil {
		t.Fatal("ToolCallUpdate() nil")
	}
	if p.Output() != "out" || p.Status() != "running" || p.Final() {
		t.Errorf("payload incorreto: %+v", p)
	}
}

func TestNewSessionEnd(t *testing.T) {
	t.Parallel()

	evt, err := events.NewSessionEnd(fixedTS, 0, "ok", json.RawMessage(`{"exit_code":0}`))
	if err != nil {
		t.Fatalf("NewSessionEnd erro inesperado: %v", err)
	}
	if !evt.IsTerminal() {
		t.Error("session_end deve ser IsTerminal() = true")
	}
	if evt.SessionEnd() == nil || evt.SessionEnd().ExitCode() != 0 {
		t.Error("SessionEnd payload incorreto")
	}
}

func TestNewRuntimeInit(t *testing.T) {
	t.Parallel()

	t.Run("launcher binary", func(t *testing.T) {
		t.Parallel()
		evt, err := events.NewRuntimeInit(fixedTS, "binary", "claude-agent-acp", nil, "v0.1.0", "", json.RawMessage(`{"launcher":"binary"}`))
		if err != nil {
			t.Fatalf("NewRuntimeInit erro inesperado: %v", err)
		}
		if evt.Launcher() != "binary" {
			t.Errorf("Launcher() = %q; quero binary", evt.Launcher())
		}
		if evt.RuntimeInit() == nil || evt.RuntimeInit().Launcher() != "binary" {
			t.Error("RuntimeInit payload incorreto")
		}
	})

	t.Run("launcher npx", func(t *testing.T) {
		t.Parallel()
		evt, err := events.NewRuntimeInit(fixedTS, "npx", "npx", []string{"--yes", "@acp/claude@0.1.0"}, "v0.1.0", "0.1.0", json.RawMessage(`{"launcher":"npx"}`))
		if err != nil {
			t.Fatalf("NewRuntimeInit erro inesperado: %v", err)
		}
		if evt.Launcher() != "npx" {
			t.Errorf("Launcher() = %q; quero npx", evt.Launcher())
		}
	})

	t.Run("launcher inválido retorna erro", func(t *testing.T) {
		t.Parallel()
		_, err := events.NewRuntimeInit(fixedTS, "docker", "docker", nil, "", "", nil)
		if err == nil {
			t.Fatal("esperava erro; obteve nil")
		}
		if !errors.Is(err, events.ErrInvalidEvent) {
			t.Errorf("erro = %v; quero ErrInvalidEvent", err)
		}
	})
}

func TestNewUnknown(t *testing.T) {
	t.Parallel()
	evt, err := events.NewUnknown(fixedTS, "future_kind", json.RawMessage(`{"kind":"future_kind"}`))
	if err != nil {
		t.Fatalf("NewUnknown erro inesperado: %v", err)
	}
	if evt.Kind() != events.KindUnknown {
		t.Errorf("Kind() = %q; quero %q", evt.Kind(), events.KindUnknown)
	}
	if evt.Unknown() == nil || evt.Unknown().RawKind() != "future_kind" {
		t.Error("Unknown payload incorreto")
	}
	if evt.IsTerminal() {
		t.Error("unknown não deve ser terminal")
	}
}

// TestEventMarshalJSONGolden valida que MarshalJSON produz o envelope RF-08 exato.
// Os arquivos golden estão em testdata/envelopes/*.json.
// Para regenerar: delete o arquivo e rode com -update flag (ou simplesmente corrija manual).
func TestEventMarshalJSONGolden(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name  string
		build func() (events.Event, error)
		file  string
	}{
		{
			name: "agent_message",
			build: func() (events.Event, error) {
				return events.NewAgentMessage(fixedTS, "hello", json.RawMessage(`{"text":"hello"}`))
			},
			file: "agent_message.json",
		},
		{
			name: "agent_thought",
			build: func() (events.Event, error) {
				return events.NewAgentThought(fixedTS, "thinking", json.RawMessage(`{"text":"thinking"}`))
			},
			file: "agent_thought.json",
		},
		{
			name: "tool_call_start",
			build: func() (events.Event, error) {
				return events.NewToolCallStart(fixedTS, events.NewToolCallID("tc_001"), "bash", "echo", json.RawMessage(`{"name":"bash"}`))
			},
			file: "tool_call_start.json",
		},
		{
			name: "tool_call_update",
			build: func() (events.Event, error) {
				return events.NewToolCallUpdate(fixedTS, events.NewToolCallID("tc_001"), "done", "ok", true, json.RawMessage(`{"output":"done"}`))
			},
			file: "tool_call_update.json",
		},
		{
			name: "session_end",
			build: func() (events.Event, error) {
				return events.NewSessionEnd(fixedTS, 0, "ok", json.RawMessage(`{"exit_code":0}`))
			},
			file: "session_end.json",
		},
		{
			name: "runtime_init_binary",
			build: func() (events.Event, error) {
				return events.NewRuntimeInit(fixedTS, "binary", "claude-agent-acp", nil, "v0.1.0", "", json.RawMessage(`{"launcher":"binary"}`))
			},
			file: "runtime_init_binary.json",
		},
		{
			name: "runtime_init_npx",
			build: func() (events.Event, error) {
				return events.NewRuntimeInit(fixedTS, "npx", "npx", []string{"--yes"}, "v0.1.0", "0.1.0", json.RawMessage(`{"launcher":"npx"}`))
			},
			file: "runtime_init_npx.json",
		},
		{
			name: "unknown",
			build: func() (events.Event, error) {
				return events.NewUnknown(fixedTS, "future_kind", json.RawMessage(`{"kind":"future_kind"}`))
			},
			file: "unknown.json",
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			evt, err := tc.build()
			if err != nil {
				t.Fatalf("construtor erro: %v", err)
			}

			got, err := json.Marshal(evt)
			if err != nil {
				t.Fatalf("MarshalJSON erro: %v", err)
			}

			goldenPath := filepath.Join("testdata", "envelopes", tc.file)

			// Se o arquivo não existe, criar com o conteúdo atual (bootstrap).
			if _, statErr := os.Stat(goldenPath); os.IsNotExist(statErr) {
				if writeErr := os.WriteFile(goldenPath, got, 0o644); writeErr != nil {
					t.Fatalf("erro ao criar golden file: %v", writeErr)
				}
				t.Logf("golden file criado: %s", goldenPath)
				return
			}

			want, err := os.ReadFile(goldenPath)
			if err != nil {
				t.Fatalf("leitura golden file: %v", err)
			}

			// Normalizar ambos via unmarshal+marshal para comparação semântica.
			var gotMap, wantMap map[string]json.RawMessage
			if err := json.Unmarshal(got, &gotMap); err != nil {
				t.Fatalf("unmarshal got: %v", err)
			}
			if err := json.Unmarshal(want, &wantMap); err != nil {
				t.Fatalf("unmarshal want: %v", err)
			}

			gotNorm, _ := json.Marshal(gotMap)
			wantNorm, _ := json.Marshal(wantMap)

			if string(gotNorm) != string(wantNorm) {
				t.Errorf("MarshalJSON != golden:\ngot:  %s\nwant: %s", got, want)
			}
		})
	}
}
