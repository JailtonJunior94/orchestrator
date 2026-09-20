package qualitygate

import (
	"errors"
	"testing"
)

func TestCheckKind_StringAndParse(t *testing.T) {
	cases := []struct {
		kind CheckKind
		text string
	}{
		{CheckFmt, "fmt"},
		{CheckTest, "test"},
		{CheckLint, "lint"},
	}
	for _, tc := range cases {
		if got := tc.kind.String(); got != tc.text {
			t.Fatalf("String() = %q, want %q", got, tc.text)
		}
		parsed, err := ParseCheckKind(tc.text)
		if err != nil {
			t.Fatalf("ParseCheckKind(%q) unexpected error: %v", tc.text, err)
		}
		if parsed != tc.kind {
			t.Fatalf("ParseCheckKind(%q) = %v, want %v", tc.text, parsed, tc.kind)
		}
		if !tc.kind.Valid() {
			t.Fatalf("%v should be valid", tc.kind)
		}
	}
}

func TestParseCheckKind_Unknown(t *testing.T) {
	_, err := ParseCheckKind("coverage")
	if !errors.Is(err, ErrUnknownCheckKind) {
		t.Fatalf("expected ErrUnknownCheckKind, got %v", err)
	}
}

func TestCheckKind_InvalidZeroValue(t *testing.T) {
	var zero CheckKind
	if zero.Valid() {
		t.Fatalf("zero value must not be valid")
	}
	if zero.String() != "unknown" {
		t.Fatalf("zero value String() = %q, want unknown", zero.String())
	}
}
