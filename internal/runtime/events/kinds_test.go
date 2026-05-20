package events_test

import (
	"testing"

	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/events"
)

func TestParseEventKind(t *testing.T) {
	t.Parallel()

	valid := []struct {
		input string
		want  events.EventKind
	}{
		{"agent_message", events.KindAgentMessage},
		{"agent_thought", events.KindAgentThought},
		{"tool_call_start", events.KindToolCallStart},
		{"tool_call_update", events.KindToolCallUpdate},
		{"session_end", events.KindSessionEnd},
		{"runtime_init", events.KindRuntimeInit},
		{"unknown", events.KindUnknown},
	}

	for _, tc := range valid {
		tc := tc
		t.Run("valid/"+tc.input, func(t *testing.T) {
			t.Parallel()
			got, err := events.ParseEventKind(tc.input)
			if err != nil {
				t.Fatalf("ParseEventKind(%q) erro inesperado: %v", tc.input, err)
			}
			if got != tc.want {
				t.Errorf("ParseEventKind(%q) = %q; quero %q", tc.input, got, tc.want)
			}
		})
	}

	invalid := []string{"", "message", "AGENT_MESSAGE", "session_start", "  agent_message"}
	for _, s := range invalid {
		s := s
		t.Run("invalid/"+s, func(t *testing.T) {
			t.Parallel()
			_, err := events.ParseEventKind(s)
			if err == nil {
				t.Errorf("ParseEventKind(%q) esperava erro; obteve nil", s)
			}
		})
	}
}
