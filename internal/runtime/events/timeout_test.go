package events_test

import (
	"testing"
	"time"

	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/events"
)

func TestNewActivityTimeout(t *testing.T) {
	t.Parallel()

	t.Run("zero é válido e desabilitado", func(t *testing.T) {
		t.Parallel()
		to, err := events.NewActivityTimeout(0)
		if err != nil {
			t.Fatalf("NewActivityTimeout(0) erro inesperado: %v", err)
		}
		if !to.Disabled() {
			t.Error("timeout zero deve ser Disabled() = true")
		}
	})

	t.Run("positivo é válido e habilitado", func(t *testing.T) {
		t.Parallel()
		to, err := events.NewActivityTimeout(120 * time.Second)
		if err != nil {
			t.Fatalf("NewActivityTimeout(120s) erro inesperado: %v", err)
		}
		if to.Disabled() {
			t.Error("timeout positivo não deve ser Disabled()")
		}
		if to.Duration() != 120*time.Second {
			t.Errorf("Duration() = %v; quero 120s", to.Duration())
		}
	})

	t.Run("negativo retorna erro", func(t *testing.T) {
		t.Parallel()
		_, err := events.NewActivityTimeout(-1 * time.Second)
		if err == nil {
			t.Error("NewActivityTimeout(-1s) esperava erro; obteve nil")
		}
	})
}
