package runtime_test

import (
	"errors"
	"fmt"
	"testing"
	"time"

	airuntime "github.com/JailtonJunior94/ai-spec-harness/internal/runtime"
)

// TestRetryClassifier_IsTransient verifica a classificação transitório/fatal.
func TestRetryClassifier_IsTransient(t *testing.T) {
	t.Parallel()

	c := airuntime.NewRetryClassifier()

	tests := []struct {
		name      string
		err       error
		wantTrans bool
	}{
		// Erros fatais — não são transitórios.
		{name: "nil", err: nil, wantTrans: false},
		{name: "ErrPermissionDenied", err: airuntime.ErrPermissionDenied, wantTrans: false},
		{name: "ErrUnsupportedTool", err: airuntime.ErrUnsupportedTool, wantTrans: false},
		{name: "ErrSessionAborted", err: airuntime.ErrSessionAborted, wantTrans: false},
		{name: "ErrLauncherUnavailable", err: airuntime.ErrLauncherUnavailable, wantTrans: false},
		// Erros wrappados de permissão — ainda fatais.
		{name: "wrapped ErrPermissionDenied", err: fmt.Errorf("runner: %w", airuntime.ErrPermissionDenied), wantTrans: false},
		// Erros genéricos — transitórios.
		{name: "io error", err: errors.New("connection reset"), wantTrans: true},
		{name: "timeout error", err: errors.New("i/o timeout"), wantTrans: true},
		{name: "generic error", err: errors.New("unexpected exit"), wantTrans: true},
		// ErrActivityTimeout — transitório (inatividade, pode ser retentado).
		{name: "ErrActivityTimeout", err: airuntime.ErrActivityTimeout, wantTrans: true},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := c.IsTransient(tc.err)
			if got != tc.wantTrans {
				t.Errorf("IsTransient(%v) = %v, want %v", tc.err, got, tc.wantTrans)
			}
		})
	}
}

// TestBackoffDuration verifica cálculo exponencial do backoff.
func TestBackoffDuration(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		base        time.Duration
		multiplier  float64
		attempt     int
		wantAtLeast time.Duration
		wantZero    bool
	}{
		{
			name:       "zero base retorna zero",
			base:       0, multiplier: 2, attempt: 1,
			wantZero: true,
		},
		{
			name:       "multiplier zero retorna zero",
			base:       100 * time.Millisecond, multiplier: 0, attempt: 1,
			wantZero: true,
		},
		{
			name:       "multiplier negativo retorna zero",
			base:       100 * time.Millisecond, multiplier: -1, attempt: 1,
			wantZero: true,
		},
		{
			name:       "attempt zero retorna zero",
			base:       100 * time.Millisecond, multiplier: 2, attempt: 0,
			wantZero: true,
		},
		{
			name:        "attempt 1 retorna base*multiplier",
			base:        100 * time.Millisecond, multiplier: 2, attempt: 1,
			wantAtLeast: 200 * time.Millisecond,
		},
		{
			name:        "attempt 2 retorna base*multiplier^2",
			base:        100 * time.Millisecond, multiplier: 2, attempt: 2,
			wantAtLeast: 400 * time.Millisecond,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := airuntime.BackoffDuration(tc.base, tc.multiplier, tc.attempt)
			if tc.wantZero {
				if got != 0 {
					t.Errorf("BackoffDuration(%v, %v, %d) = %v, want 0", tc.base, tc.multiplier, tc.attempt, got)
				}
			} else {
				if got < tc.wantAtLeast {
					t.Errorf("BackoffDuration(%v, %v, %d) = %v, want >= %v", tc.base, tc.multiplier, tc.attempt, got, tc.wantAtLeast)
				}
			}
		})
	}
}

// TestRetryClassifier_FatalWrapped garante que erros fatais wrappados permanecem fatais.
func TestRetryClassifier_FatalWrapped(t *testing.T) {
	t.Parallel()

	c := airuntime.NewRetryClassifier()

	fatalErrs := []error{
		airuntime.ErrPermissionDenied,
		airuntime.ErrUnsupportedTool,
		airuntime.ErrSessionAborted,
		airuntime.ErrLauncherUnavailable,
	}

	for _, fatal := range fatalErrs {
		fatal := fatal
		t.Run(fatal.Error(), func(t *testing.T) {
			t.Parallel()
			// Wrappado em dois níveis.
			wrapped := fmt.Errorf("outer: %w", fmt.Errorf("inner: %w", fatal))
			if c.IsTransient(wrapped) {
				t.Errorf("IsTransient(wrapped %v) = true, want false (fatal deve ser não-transitório mesmo wrappado)", fatal)
			}
		})
	}
}
