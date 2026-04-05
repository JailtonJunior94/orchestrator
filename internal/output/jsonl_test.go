package output

import (
	"errors"
	"testing"
)

func TestExtractJSONL(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		raw         string
		wantText    string
		wantErr     bool
		recoverable bool
	}{
		{
			name:     "single message event",
			raw:      `{"type":"message","content":"Hello from Codex"}`,
			wantText: "Hello from Codex",
		},
		{
			name: "multiple events — last message wins",
			raw: `{"type":"step","text":"step1"}
{"type":"message","content":"final answer"}`,
			wantText: "final answer",
		},
		{
			name: "blank lines ignored",
			raw: `{"type":"message","content":"answer"}

`,
			wantText: "answer",
		},
		{
			name:     "whitespace-only lines ignored",
			raw:      "   \n{\"type\":\"message\",\"content\":\"ok\"}\n  \n",
			wantText: "ok",
		},
		{
			name: "fallback concatenation when no message event",
			raw: `{"type":"step","text":"part1"}
{"type":"step","text":"part2"}`,
			wantText: "part1\npart2",
		},
		{
			name:     "content field without type message",
			raw:      `{"content":"direct content"}`,
			wantText: "direct content",
		},
		{
			name:        "invalid JSON line — recoverable error",
			raw:         `not-json`,
			wantErr:     true,
			recoverable: true,
		},
		{
			name:    "empty raw — no events",
			raw:     "",
			wantErr: true,
		},
		{
			name:    "only blank lines",
			raw:     "\n\n   \n",
			wantErr: true,
		},
		{
			name: "last message event with text field",
			raw: `{"type":"message","text":"text content"}
{"type":"done"}`,
			wantText: "text content",
		},
		{
			name:     "nested content array is extracted",
			raw:      `{"type":"response.completed","response":{"output":[{"type":"message","content":[{"type":"output_text","text":"final answer"}]}]}}`,
			wantText: "final answer",
		},
		{
			name:     "message field object is extracted",
			raw:      `{"event":"completed","message":{"role":"assistant","content":[{"type":"output_text","text":"done"}]}}`,
			wantText: "done",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := ExtractJSONL(tc.raw)
			if tc.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				var pe *ProcessError
				if tc.recoverable && !errors.As(err, &pe) {
					t.Fatalf("expected ProcessError, got %T: %v", err, err)
				}
				if tc.recoverable && !pe.Recoverable {
					t.Fatal("expected recoverable=true")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tc.wantText {
				t.Fatalf("got %q, want %q", got, tc.wantText)
			}
		})
	}
}
