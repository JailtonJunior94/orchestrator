package events_test

import (
	"testing"

	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/events"
)

func TestParseCancelReason(t *testing.T) {
	t.Parallel()

	valid := []struct {
		input string
		want  events.CancelReason
	}{
		{"none", events.CancelReasonNone},
		{"activity_timeout", events.CancelReasonActivityTimeout},
		{"context_canceled", events.CancelReasonContextCanceled},
		{"tool_error", events.CancelReasonToolError},
		{"permission_denied", events.CancelReasonPermissionDenied},
	}

	for _, tc := range valid {
		tc := tc
		t.Run("valid/"+tc.input, func(t *testing.T) {
			t.Parallel()
			got, err := events.NewCatalog().ParseCancelReason(tc.input)
			if err != nil {
				t.Fatalf("ParseCancelReason(%q) erro inesperado: %v", tc.input, err)
			}
			if got != tc.want {
				t.Errorf("ParseCancelReason(%q) = %q; quero %q", tc.input, got, tc.want)
			}
		})
	}

	invalid := []string{"", "timeout", "NONE", "cancelled", "permission"}
	for _, s := range invalid {
		s := s
		t.Run("invalid/"+s, func(t *testing.T) {
			t.Parallel()
			_, err := events.NewCatalog().ParseCancelReason(s)
			if err == nil {
				t.Errorf("ParseCancelReason(%q) esperava erro; obteve nil", s)
			}
		})
	}
}
