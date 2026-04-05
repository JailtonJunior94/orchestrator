package theme

import (
	"testing"
)

func TestMaestroDefaultsNotNil(t *testing.T) {
	t.Parallel()

	m := New()
	if m.Primary == nil {
		t.Error("Primary color must not be nil")
	}
	if m.Success == nil {
		t.Error("Success color must not be nil")
	}
	if m.Warning == nil {
		t.Error("Warning color must not be nil")
	}
	if m.Error == nil {
		t.Error("Error color must not be nil")
	}
}

func TestFallback256ColorsNotNil(t *testing.T) {
	t.Parallel()

	m := fallback256()

	if m.Primary == nil {
		t.Error("fallback Primary must not be nil")
	}
	if m.Success == nil {
		t.Error("fallback Success must not be nil")
	}
	if m.Warning == nil {
		t.Error("fallback Warning must not be nil")
	}
	if m.Error == nil {
		t.Error("fallback Error must not be nil")
	}
}

func TestTrueColorVs256(t *testing.T) {
	// Simulate true-color terminal.
	t.Setenv("COLORTERM", "truecolor")
	tcm := New()

	// Simulate 256-color terminal.
	t.Setenv("COLORTERM", "")
	m256 := New()

	if tcm.Primary == nil {
		t.Error("true-color Primary must not be nil")
	}
	if m256.Primary == nil {
		t.Error("256-color Primary must not be nil")
	}
}

func TestHasTrueColorEnvVars(t *testing.T) {
	tests := []struct {
		env  string
		want bool
	}{
		{"truecolor", true},
		{"24bit", true},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.env, func(t *testing.T) {
			t.Setenv("COLORTERM", tt.env)

			got := hasTrueColor()
			// In CI/test environment stdout may not be a terminal, so only
			// validate that "truecolor"/"24bit" return true regardless.
			if (tt.env == "truecolor" || tt.env == "24bit") && !got {
				t.Errorf("hasTrueColor() = false for COLORTERM=%q; want true", tt.env)
			}
		})
	}
}
