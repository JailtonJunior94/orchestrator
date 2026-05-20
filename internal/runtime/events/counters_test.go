package events_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/events"
)

func mustNewToolCallStart(t *testing.T, ts time.Time, id events.ToolCallID, name, input string, raw json.RawMessage) events.Event {
	t.Helper()
	evt, err := events.NewToolCallStart(ts, id, name, input, raw)
	if err != nil {
		t.Fatalf("NewToolCallStart erro: %v", err)
	}
	return evt
}

func mustNewToolCallUpdate(t *testing.T, ts time.Time, id events.ToolCallID, output, status string, final bool, raw json.RawMessage) events.Event {
	t.Helper()
	evt, err := events.NewToolCallUpdate(ts, id, output, status, final, raw)
	if err != nil {
		t.Fatalf("NewToolCallUpdate erro: %v", err)
	}
	return evt
}

func mustNewUnknown(t *testing.T, ts time.Time, rawKind string, raw json.RawMessage) events.Event {
	t.Helper()
	evt, err := events.NewUnknown(ts, rawKind, raw)
	if err != nil {
		t.Fatalf("NewUnknown erro: %v", err)
	}
	return evt
}

func TestToolCallCounters(t *testing.T) {
	t.Parallel()

	t.Run("zero eventos", func(t *testing.T) {
		t.Parallel()
		c := events.NewToolCallCounters()
		if c.MappedCount() != 0 {
			t.Errorf("MappedCount() = %d; quero 0", c.MappedCount())
		}
		if len(c.ToolCalls()) != 0 {
			t.Errorf("ToolCalls() len = %d; quero 0", len(c.ToolCalls()))
		}
		if len(c.UnknownKinds()) != 0 {
			t.Errorf("UnknownKinds() len = %d; quero 0", len(c.UnknownKinds()))
		}
	})

	t.Run("vários tool calls ordenados por startedAt", func(t *testing.T) {
		t.Parallel()
		c := events.NewToolCallCounters()

		ts1 := time.Date(2026, 5, 20, 10, 0, 0, 0, time.UTC)
		ts2 := time.Date(2026, 5, 20, 10, 0, 1, 0, time.UTC)
		ts3 := time.Date(2026, 5, 20, 10, 0, 2, 0, time.UTC)

		id1 := events.NewToolCallID("tc_001")
		id2 := events.NewToolCallID("tc_002")

		// Adiciona tc_002 primeiro para testar ordenação.
		c.Record(mustNewToolCallStart(t, ts2, id2, "grep", "", json.RawMessage(`{}`)))
		c.Record(mustNewToolCallStart(t, ts1, id1, "bash", "", json.RawMessage(`{}`)))
		c.Record(mustNewToolCallUpdate(t, ts3, id1, "out", "done", true, json.RawMessage(`{}`)))

		if c.MappedCount() != 2 {
			t.Errorf("MappedCount() = %d; quero 2", c.MappedCount())
		}

		calls := c.ToolCalls()
		if len(calls) != 2 {
			t.Fatalf("ToolCalls() len = %d; quero 2", len(calls))
		}

		// Primeiro deve ser tc_001 (ts1 < ts2).
		if calls[0].ID.String() != "tc_001" {
			t.Errorf("ToolCalls()[0].ID = %q; quero tc_001", calls[0].ID)
		}
		if calls[0].Name != "bash" {
			t.Errorf("ToolCalls()[0].Name = %q; quero bash", calls[0].Name)
		}
		if calls[0].Updates != 1 {
			t.Errorf("ToolCalls()[0].Updates = %d; quero 1", calls[0].Updates)
		}
		if !calls[0].Final {
			t.Error("ToolCalls()[0].Final = false; quero true")
		}

		if calls[1].ID.String() != "tc_002" {
			t.Errorf("ToolCalls()[1].ID = %q; quero tc_002", calls[1].ID)
		}
	})

	t.Run("eventos unknown acumulam kinds únicos", func(t *testing.T) {
		t.Parallel()
		c := events.NewToolCallCounters()

		ts := time.Date(2026, 5, 20, 10, 0, 0, 0, time.UTC)
		c.Record(mustNewUnknown(t, ts, "kind_a", json.RawMessage(`{}`)))
		c.Record(mustNewUnknown(t, ts, "kind_b", json.RawMessage(`{}`)))
		c.Record(mustNewUnknown(t, ts, "kind_a", json.RawMessage(`{}`)))

		kinds := c.UnknownKinds()
		if len(kinds) != 2 {
			t.Errorf("UnknownKinds() len = %d; quero 2 (deduplicado)", len(kinds))
		}
		if kinds[0] != "kind_a" || kinds[1] != "kind_b" {
			t.Errorf("UnknownKinds() = %v; quero [kind_a kind_b]", kinds)
		}
	})

	t.Run("tool_call_update sem start cria registro parcial", func(t *testing.T) {
		t.Parallel()
		c := events.NewToolCallCounters()

		ts := time.Date(2026, 5, 20, 10, 0, 0, 0, time.UTC)
		id := events.NewToolCallID("tc_orphan")
		c.Record(mustNewToolCallUpdate(t, ts, id, "out", "done", true, json.RawMessage(`{}`)))

		if c.MappedCount() != 1 {
			t.Errorf("MappedCount() = %d; quero 1", c.MappedCount())
		}
	})

	t.Run("tool_call_start com ID zero ignorado", func(t *testing.T) {
		t.Parallel()
		c := events.NewToolCallCounters()

		ts := time.Date(2026, 5, 20, 10, 0, 0, 0, time.UTC)
		zeroID := events.NewToolCallID("")
		c.Record(mustNewToolCallStart(t, ts, zeroID, "bash", "", json.RawMessage(`{}`)))

		if c.MappedCount() != 0 {
			t.Errorf("MappedCount() = %d; quero 0 (ID zero ignorado)", c.MappedCount())
		}
	})
}
