package events_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/events"
)

// TestPayloadsGetters cobre os getters de payload não exercitados pelos outros testes.
func TestPayloadsGetters(t *testing.T) {
	t.Parallel()

	t.Run("ToolCallStartPayload.Input", func(t *testing.T) {
		t.Parallel()
		evt, err := events.NewToolCallStart(fixedTS, events.NewToolCallID("tc_x"), "bash", `{"cmd":"ls"}`, json.RawMessage(`{}`))
		if err != nil {
			t.Fatalf("NewToolCallStart: %v", err)
		}
		p := evt.ToolCallStart()
		if p == nil {
			t.Fatal("ToolCallStart() nil")
		}
		if p.Input() != `{"cmd":"ls"}` {
			t.Errorf("Input() = %q; quero %q", p.Input(), `{"cmd":"ls"}`)
		}
	})

	t.Run("SessionEndPayload.Reason", func(t *testing.T) {
		t.Parallel()
		evt, err := events.NewSessionEnd(fixedTS, 0, "ok", json.RawMessage(`{}`))
		if err != nil {
			t.Fatalf("NewSessionEnd: %v", err)
		}
		p := evt.SessionEnd()
		if p == nil {
			t.Fatal("SessionEnd() nil")
		}
		if p.Reason() != "ok" {
			t.Errorf("Reason() = %q; quero ok", p.Reason())
		}
	})

	t.Run("RuntimeInitPayload getters", func(t *testing.T) {
		t.Parallel()
		args := []string{"--yes", "@acp/claude@0.1.0"}
		evt, err := events.NewRuntimeInit(
			fixedTS, "npx", "npx", args, "v0.2.0", "0.1.0",
			json.RawMessage(`{}`),
		)
		if err != nil {
			t.Fatalf("NewRuntimeInit: %v", err)
		}
		p := evt.RuntimeInit()
		if p == nil {
			t.Fatal("RuntimeInit() nil")
		}
		if p.Command() != "npx" {
			t.Errorf("Command() = %q; quero npx", p.Command())
		}
		gotArgs := p.Args()
		if len(gotArgs) != 2 || gotArgs[0] != "--yes" {
			t.Errorf("Args() = %v; quero %v", gotArgs, args)
		}
		if p.SDKVersion() != "v0.2.0" {
			t.Errorf("SDKVersion() = %q; quero v0.2.0", p.SDKVersion())
		}
		if p.NpmVersion() != "0.1.0" {
			t.Errorf("NpmVersion() = %q; quero 0.1.0", p.NpmVersion())
		}
	})
}

// TestSessionStateTransitionUnknownState garante que estado inválido retorna erro.
func TestSessionStateTransitionUnknownState(t *testing.T) {
	t.Parallel()
	var s events.SessionState = "nonexistent"
	got, err := s.Transition(events.StateRunning)
	if err == nil {
		t.Fatal("esperava erro para estado desconhecido; obteve nil")
	}
	if got != s {
		t.Errorf("estado deve ser preservado em erro: got %q; quero %q", got, s)
	}
}

// TestCountersRecordIgnoresUnknownWithoutRawKind garante que unknown sem rawKind não cria entrada.
func TestCountersRecordIgnoresUnknownWithoutRawKind(t *testing.T) {
	t.Parallel()
	c := events.NewToolCallCounters()
	ts := time.Date(2026, 5, 20, 10, 0, 0, 0, time.UTC)
	// NewUnknown com rawKind vazio.
	evt, err := events.NewUnknown(ts, "", json.RawMessage(`{}`))
	if err != nil {
		t.Fatalf("NewUnknown: %v", err)
	}
	c.Record(evt)
	if len(c.UnknownKinds()) != 0 {
		t.Errorf("UnknownKinds() len = %d; quero 0 para rawKind vazio", len(c.UnknownKinds()))
	}
}
