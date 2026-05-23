package events_test

import (
	"encoding/json"
	"testing"

	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/events"
)

func TestToolCallID(t *testing.T) {
	t.Parallel()

	t.Run("zero value IsZero true", func(t *testing.T) {
		t.Parallel()
		var id events.ToolCallID
		if !id.IsZero() {
			t.Error("zero ToolCallID deve retornar IsZero() = true")
		}
	})

	t.Run("NewToolCallID vazio é zero", func(t *testing.T) {
		t.Parallel()
		id := events.NewToolCallID("")
		if !id.IsZero() {
			t.Error("NewToolCallID(\"\") deve ser zero")
		}
	})

	t.Run("NewToolCallID não-vazio não é zero", func(t *testing.T) {
		t.Parallel()
		id := events.NewToolCallID("tc_123")
		if id.IsZero() {
			t.Error("NewToolCallID(\"tc_123\") não deve ser zero")
		}
		if id.String() != "tc_123" {
			t.Errorf("String() = %q; quero %q", id.String(), "tc_123")
		}
	})

	t.Run("MarshalJSON zero produz null", func(t *testing.T) {
		t.Parallel()
		id := events.NewToolCallID("")
		b, err := json.Marshal(id)
		if err != nil {
			t.Fatalf("MarshalJSON erro: %v", err)
		}
		if string(b) != "null" {
			t.Errorf("MarshalJSON zero = %s; quero null", string(b))
		}
	})

	t.Run("MarshalJSON não-zero produz string", func(t *testing.T) {
		t.Parallel()
		id := events.NewToolCallID("tc_abc")
		b, err := json.Marshal(id)
		if err != nil {
			t.Fatalf("MarshalJSON erro: %v", err)
		}
		if string(b) != `"tc_abc"` {
			t.Errorf("MarshalJSON = %s; quero %q", string(b), `"tc_abc"`)
		}
	})
}
