package hookcontract

import (
	"errors"
	"testing"
)

func TestEventKind_ParseAndString(t *testing.T) {
	tests := []struct {
		kind EventKind
		text string
	}{
		{EventSessionStart, "session_start"},
		{EventBeforeTool, "before_tool"},
		{EventAfterTool, "after_tool"},
		{EventBeforeComplete, "before_complete"},
		{EventSessionEnd, "session_end"},
	}

	for _, tt := range tests {
		t.Run(tt.text, func(t *testing.T) {
			if got := tt.kind.String(); got != tt.text {
				t.Fatalf("String() = %q, want %q", got, tt.text)
			}
			parsed, err := ParseEventKind(tt.text)
			if err != nil {
				t.Fatalf("ParseEventKind(%q) unexpected error: %v", tt.text, err)
			}
			if parsed != tt.kind {
				t.Fatalf("ParseEventKind(%q) = %v, want %v", tt.text, parsed, tt.kind)
			}
			if !tt.kind.Valid() {
				t.Fatalf("%v should be Valid()", tt.kind)
			}
		})
	}
}

func TestEventKind_ParseUnknownReturnsSentinel(t *testing.T) {
	_, err := ParseEventKind("mid_tool_panic")
	if !errors.Is(err, ErrUnknownEvent) {
		t.Fatalf("ParseEventKind(unknown) error = %v, want ErrUnknownEvent", err)
	}
}

func TestEventKind_ZeroValueIsInvalid(t *testing.T) {
	var kind EventKind
	if kind.Valid() {
		t.Fatal("zero-value EventKind must not be Valid()")
	}
}

func TestEventKinds_ReturnsAllFive(t *testing.T) {
	kinds := EventKinds()
	if len(kinds) != 5 {
		t.Fatalf("EventKinds() len = %d, want 5", len(kinds))
	}
}
