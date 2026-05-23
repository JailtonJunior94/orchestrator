package events_test

import (
	"errors"
	"testing"

	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/events"
)

func TestSessionStateTransition(t *testing.T) {
	t.Parallel()

	validCases := []struct {
		from events.SessionState
		to   events.SessionState
	}{
		{events.StateInit, events.StateRunning},
		{events.StateInit, events.StateCanceled},
		{events.StateRunning, events.StateAwaiting},
		{events.StateRunning, events.StateClosed},
		{events.StateRunning, events.StateCanceled},
		{events.StateAwaiting, events.StateCanceled},
	}

	for _, tc := range validCases {
		tc := tc
		t.Run("valid/"+string(tc.from)+"->"+string(tc.to), func(t *testing.T) {
			t.Parallel()
			got, err := tc.from.Transition(tc.to)
			if err != nil {
				t.Fatalf("Transition(%q → %q) erro inesperado: %v", tc.from, tc.to, err)
			}
			if got != tc.to {
				t.Errorf("Transition retornou %q; quero %q", got, tc.to)
			}
		})
	}

	invalidCases := []struct {
		from events.SessionState
		to   events.SessionState
	}{
		{events.StateInit, events.StateInit},
		{events.StateInit, events.StateAwaiting},
		{events.StateInit, events.StateClosed},
		{events.StateRunning, events.StateInit},
		{events.StateRunning, events.StateRunning},
		{events.StateAwaiting, events.StateRunning},
		{events.StateAwaiting, events.StateClosed},
		{events.StateAwaiting, events.StateInit},
		{events.StateClosed, events.StateRunning},
		{events.StateClosed, events.StateCanceled},
		{events.StateCanceled, events.StateRunning},
		{events.StateCanceled, events.StateClosed},
	}

	for _, tc := range invalidCases {
		tc := tc
		t.Run("invalid/"+string(tc.from)+"->"+string(tc.to), func(t *testing.T) {
			t.Parallel()
			got, err := tc.from.Transition(tc.to)
			if err == nil {
				t.Errorf("Transition(%q → %q) esperava erro; obteve nil (retornou %q)", tc.from, tc.to, got)
				return
			}
			if !errors.Is(err, events.ErrInvalidTransition) {
				t.Errorf("Transition(%q → %q) erro = %v; quero ErrInvalidTransition", tc.from, tc.to, err)
			}
			// estado não deve mudar em transição inválida
			if got != tc.from {
				t.Errorf("Transition(%q → %q) estado inválido retornou %q; quero estado anterior %q", tc.from, tc.to, got, tc.from)
			}
		})
	}
}
