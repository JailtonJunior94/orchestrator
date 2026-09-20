package qualitygate

import (
	"errors"
	"testing"
)

func TestRisk_StringAndParse(t *testing.T) {
	cases := []struct {
		risk Risk
		text string
	}{
		{RiskLow, "low"},
		{RiskMedium, "medium"},
		{RiskHigh, "high"},
	}
	for _, tc := range cases {
		if got := tc.risk.String(); got != tc.text {
			t.Fatalf("String() = %q, want %q", got, tc.text)
		}
		parsed, err := ParseRisk(tc.text)
		if err != nil {
			t.Fatalf("ParseRisk(%q) unexpected error: %v", tc.text, err)
		}
		if parsed != tc.risk {
			t.Fatalf("ParseRisk(%q) = %v, want %v", tc.text, parsed, tc.risk)
		}
	}
}

func TestParseRisk_Unknown(t *testing.T) {
	_, err := ParseRisk("critical")
	if !errors.Is(err, ErrUnknownRisk) {
		t.Fatalf("expected ErrUnknownRisk, got %v", err)
	}
}

func TestNewTaskType_RejectsEmpty(t *testing.T) {
	_, err := NewTaskType("   ")
	if !errors.Is(err, ErrEmptyTaskType) {
		t.Fatalf("expected ErrEmptyTaskType, got %v", err)
	}
}

func TestNewTaskType_Trims(t *testing.T) {
	taskType, err := NewTaskType("  feature  ")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if taskType != "feature" {
		t.Fatalf("got %q, want feature", taskType)
	}
}
