package hookcontract

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

const SchemaVersionV1 = 1

var schemaVersionDescriptions = map[int]string{
	SchemaVersionV1: "v1: schema_version, event, provider, session_id, payload (initial canonical envelope)",
}

var (
	ErrSchemaVersion    = errors.New("unsupported payload schema version")
	ErrUnknownField     = errors.New("unknown field in payload envelope")
	ErrMalformedPayload = errors.New("malformed payload envelope")
	ErrProviderUnknown  = errors.New("provider not declared in capability registry")
)

type Envelope struct {
	SchemaVersion int
	Event         EventKind
	Provider      string
	SessionID     string
	Payload       json.RawMessage
}

type Decoder interface {
	Decode(raw []byte) (Envelope, error)
}

type envelopeWireFormat struct {
	SchemaVersion int             `json:"schema_version"`
	Event         string          `json:"event"`
	Provider      string          `json:"provider"`
	SessionID     string          `json:"session_id"`
	Payload       json.RawMessage `json:"payload"`
}

type strictDecoder struct{}

func NewDecoder() Decoder {
	return strictDecoder{}
}

func (strictDecoder) Decode(raw []byte) (Envelope, error) {
	jsonDecoder := json.NewDecoder(bytes.NewReader(raw))
	jsonDecoder.DisallowUnknownFields()

	var wire envelopeWireFormat
	if err := jsonDecoder.Decode(&wire); err != nil {
		if strings.Contains(err.Error(), "unknown field") {
			return Envelope{}, fmt.Errorf("%w: %v", ErrUnknownField, err)
		}
		return Envelope{}, fmt.Errorf("%w: %v", ErrMalformedPayload, err)
	}

	if !SchemaVersionSupported(wire.SchemaVersion) {
		return Envelope{}, fmt.Errorf("%w: %d", ErrSchemaVersion, wire.SchemaVersion)
	}

	event, err := ParseEventKind(wire.Event)
	if err != nil {
		return Envelope{}, err
	}

	return Envelope{
		SchemaVersion: wire.SchemaVersion,
		Event:         event,
		Provider:      wire.Provider,
		SessionID:     wire.SessionID,
		Payload:       wire.Payload,
	}, nil
}

func SchemaVersionSupported(version int) bool {
	_, ok := schemaVersionDescriptions[version]
	return ok
}

func SchemaVersionDescription(version int) (string, bool) {
	description, ok := schemaVersionDescriptions[version]
	return description, ok
}
