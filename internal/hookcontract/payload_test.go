package hookcontract

import (
	"errors"
	"testing"
)

func TestEnvelope_RejectsUnknownField(t *testing.T) {
	raw := []byte(`{"schema_version":1,"event":"before_tool","provider":"provider-x","session_id":"s-1","payload":{},"unexpected_field":true}`)

	_, err := NewDecoder().Decode(raw)
	if !errors.Is(err, ErrUnknownField) {
		t.Fatalf("error = %v, want ErrUnknownField", err)
	}
}

func TestEnvelope_RejectsUnsupportedSchemaVersion(t *testing.T) {
	raw := []byte(`{"schema_version":99,"event":"before_tool","provider":"provider-x","session_id":"s-1","payload":{}}`)

	_, err := NewDecoder().Decode(raw)
	if !errors.Is(err, ErrSchemaVersion) {
		t.Fatalf("error = %v, want ErrSchemaVersion", err)
	}
}

func TestEnvelope_RejectsUnknownEvent(t *testing.T) {
	raw := []byte(`{"schema_version":1,"event":"mid_tool_panic","provider":"provider-x","session_id":"s-1","payload":{}}`)

	_, err := NewDecoder().Decode(raw)
	if !errors.Is(err, ErrUnknownEvent) {
		t.Fatalf("error = %v, want ErrUnknownEvent", err)
	}
}

func TestEnvelope_RejectsMalformedJSON(t *testing.T) {
	raw := []byte(`{"schema_version":1,`)

	_, err := NewDecoder().Decode(raw)
	if !errors.Is(err, ErrMalformedPayload) {
		t.Fatalf("error = %v, want ErrMalformedPayload", err)
	}
}

func TestEnvelope_AcceptsWellFormedV1(t *testing.T) {
	raw := []byte(`{"schema_version":1,"event":"before_tool","provider":"provider-x","session_id":"s-1","payload":{"tool_name":"Bash"}}`)

	envelope, err := NewDecoder().Decode(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if envelope.Event != EventBeforeTool {
		t.Fatalf("Event = %v, want EventBeforeTool", envelope.Event)
	}
	if envelope.SchemaVersion != SchemaVersionV1 {
		t.Fatalf("SchemaVersion = %d, want %d", envelope.SchemaVersion, SchemaVersionV1)
	}
	if envelope.Provider != "provider-x" {
		t.Fatalf("Provider = %q", envelope.Provider)
	}
}

func TestSchemaVersionMigration(t *testing.T) {
	if !SchemaVersionSupported(SchemaVersionV1) {
		t.Fatal("SchemaVersionV1 must be supported")
	}
	description, ok := SchemaVersionDescription(SchemaVersionV1)
	if !ok || description == "" {
		t.Fatal("SchemaVersionV1 must carry a non-empty migration description")
	}
	if SchemaVersionSupported(2) {
		t.Fatal("unreleased schema version 2 must not be supported yet")
	}
	if _, ok := SchemaVersionDescription(2); ok {
		t.Fatal("unreleased schema version 2 must not have a migration description")
	}
}

func FuzzDecode(f *testing.F) {
	f.Add([]byte(`{"schema_version":1,"event":"before_tool","provider":"p","session_id":"s","payload":{}}`))
	f.Add([]byte(`{}`))
	f.Add([]byte(``))
	f.Add([]byte(`{"schema_version":1,"event":"unknown","provider":"p","session_id":"s","payload":{}}`))
	f.Add([]byte(`{"schema_version":2,"event":"before_tool","provider":"p","session_id":"s","payload":{}}`))
	f.Add([]byte(`{"schema_version":1,"event":"before_tool","provider":"p","session_id":"s","payload":{},"extra":1}`))
	f.Add([]byte(`not json`))
	f.Add([]byte(`null`))

	decoder := NewDecoder()
	f.Fuzz(func(t *testing.T, data []byte) {
		envelope, err := decoder.Decode(data)
		if err != nil {
			return
		}
		if !envelope.Event.Valid() {
			t.Fatalf("Decode succeeded with invalid EventKind %d for input %q", int(envelope.Event), data)
		}
		if !SchemaVersionSupported(envelope.SchemaVersion) {
			t.Fatalf("Decode succeeded with unsupported schema version %d for input %q", envelope.SchemaVersion, data)
		}
	})
}
