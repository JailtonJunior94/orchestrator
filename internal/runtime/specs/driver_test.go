package specs

import (
	"errors"
	"testing"
)

func TestParseDriverID(t *testing.T) {
	t.Parallel()

	validDrivers := []string{"claude", "codex", "copilot", "gemini"}

	for _, name := range validDrivers {
		t.Run("valid/"+name, func(t *testing.T) {
			t.Parallel()
			got, err := ParseDriverID(name)
			if err != nil {
				t.Fatalf("ParseDriverID(%q) inesperado erro: %v", name, err)
			}
			if got.String() != name {
				t.Errorf("String() = %q; want %q", got.String(), name)
			}
		})
	}

	invalidCases := []struct {
		input string
	}{
		{""},
		{"CLAUDE"},
		{"Claude"},
		{"gpt-4"},
		{"unknown"},
		{" claude"},
		{"claude "},
	}

	for _, tc := range invalidCases {
		t.Run("invalid/"+tc.input, func(t *testing.T) {
			t.Parallel()
			_, err := ParseDriverID(tc.input)
			if err == nil {
				t.Fatalf("ParseDriverID(%q) esperava erro, obteve nil", tc.input)
			}
			if !errors.Is(err, ErrUnknownDriver) {
				t.Errorf("ParseDriverID(%q) erro = %v; want errors.Is ErrUnknownDriver", tc.input, err)
			}
		})
	}
}

func TestDriverID_String_Idempotent(t *testing.T) {
	t.Parallel()

	d, err := ParseDriverID("codex")
	if err != nil {
		t.Fatal(err)
	}

	first, second := d.String(), d.String()
	if first != second {
		t.Errorf("String() não idempotente: %q != %q", first, second)
	}
}

func TestDriverID_ZeroValue_NotValid(t *testing.T) {
	t.Parallel()

	// Zero-value de DriverID não representa um driver válido do catálogo.
	var zero DriverID
	if zero.String() != "" {
		t.Errorf("zero-value String() = %q; want empty string", zero.String())
	}
}
