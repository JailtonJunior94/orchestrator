package specs

import (
	"errors"
	"fmt"
)

// ErrUnknownDriver é retornado por ParseDriverID quando o valor não pertence ao conjunto canônico.
// Usar errors.Is para comparação.
var ErrUnknownDriver = errors.New("driver desconhecido")

// DriverID é um Value Object que representa um dos 4 drivers canônicos do ai-spec-harness.
// Imutável; campo interno não exportado (R-DDD-001 §Entidades).
// Construtor: ParseDriverID. Zero-value não é válido — use apenas valores criados pelo construtor.
type DriverID struct{ value string }

// ParseDriverID valida s contra o conjunto canônico {claude, codex, copilot, gemini}
// e retorna o DriverID correspondente.
// Retorna ErrUnknownDriver (envolvido) quando s não pertence ao conjunto.
func ParseDriverID(s string) (DriverID, error) {
	switch s {
	case "claude", "codex", "copilot", "gemini":
		return DriverID{value: s}, nil
	default:
		return DriverID{}, fmt.Errorf("%w: %q", ErrUnknownDriver, s)
	}
}

// String retorna a representação canônica do driver (ex: "claude").
func (d DriverID) String() string { return d.value }
