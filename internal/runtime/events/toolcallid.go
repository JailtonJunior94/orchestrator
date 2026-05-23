package events

import "encoding/json"

// ToolCallID é um value object que representa o identificador de uma chamada de ferramenta.
// O zero value representa ausência de ID.
type ToolCallID struct {
	value string
}

// NewToolCallID cria um ToolCallID a partir de uma string.
// String vazia resulta em ToolCallID zero.
func NewToolCallID(s string) ToolCallID {
	return ToolCallID{value: s}
}

// IsZero retorna true quando o ID está ausente (zero value).
func (t ToolCallID) IsZero() bool {
	return t.value == ""
}

// String retorna a representação string do ID.
func (t ToolCallID) String() string {
	return t.value
}

// MarshalJSON produz null quando zero, ou a string com aspas caso contrário.
func (t ToolCallID) MarshalJSON() ([]byte, error) {
	if t.IsZero() {
		return []byte("null"), nil
	}
	return json.Marshal(t.value)
}
