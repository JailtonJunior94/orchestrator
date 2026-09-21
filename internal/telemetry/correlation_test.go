package telemetry

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/JailtonJunior94/ai-spec-harness/internal/hookcontract"
)

func TestCorrelationFromEnvelope_RequiresSessionID(t *testing.T) {
	envelope := hookcontract.Envelope{SessionID: ""}
	if _, err := CorrelationFromEnvelope(envelope, "task-1"); !errors.Is(err, ErrCorrelationMissingSessionID) {
		t.Fatalf("error = %v, want ErrCorrelationMissingSessionID", err)
	}
}

func TestCorrelationFromEnvelope_UsesSessionIDAndTaskID(t *testing.T) {
	envelope := hookcontract.Envelope{SessionID: "sess-1"}
	corr, err := CorrelationFromEnvelope(envelope, "task-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if corr.SessionID != "sess-1" || corr.TaskID != "task-1" {
		t.Fatalf("Correlation = %+v, want SessionID=sess-1 TaskID=task-1", corr)
	}
	if corr.Key() != "sess-1:task-1" {
		t.Fatalf("Key() = %q, want %q", corr.Key(), "sess-1:task-1")
	}
}

func TestCorrelation_NeverDerivedFromPromptContent(t *testing.T) {
	envelopeA := hookcontract.Envelope{SessionID: "sess-1", Payload: json.RawMessage(`{"prompt":"do X"}`)}
	envelopeB := hookcontract.Envelope{SessionID: "sess-1", Payload: json.RawMessage(`{"prompt":"do a completely different unrelated thing"}`)}

	corrA, err := CorrelationFromEnvelope(envelopeA, "task-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	corrB, err := CorrelationFromEnvelope(envelopeB, "task-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if corrA.Key() != corrB.Key() {
		t.Fatalf("Key() differed across different prompt payloads (%q vs %q): correlation must not depend on prompt content",
			corrA.Key(), corrB.Key())
	}
}
