package invocation

import (
	"os"
	"testing"
)

func resetEnv() {
	os.Unsetenv(_envDepth)
	os.Unsetenv(_envMax)
}

func TestCheckDepth_WithinLimit(t *testing.T) {
	resetEnv()
	os.Setenv(_envDepth, "0")
	os.Setenv(_envMax, "2")
	defer resetEnv()

	if err := NewGuard().CheckDepth(); err != nil {
		t.Errorf("expected no error, got: %v", err)
	}
}

func TestCheckDepth_AtLimit(t *testing.T) {
	resetEnv()
	os.Setenv(_envDepth, "2")
	os.Setenv(_envMax, "2")
	defer resetEnv()

	if err := NewGuard().CheckDepth(); err == nil {
		t.Error("expected error when depth >= max, got nil")
	}
}

func TestCheckDepth_NoEnvVars_Defaults(t *testing.T) {
	resetEnv()

	if err := NewGuard().CheckDepth(); err != nil {
		t.Errorf("expected no error with defaults (depth=0, max=2), got: %v", err)
	}
}

func TestCheckDepth_MaxZero_AlwaysBlocks(t *testing.T) {
	resetEnv()
	os.Setenv(_envMax, "0")
	defer resetEnv()

	if err := NewGuard().CheckDepth(); err == nil {
		t.Error("expected error when max=0, got nil")
	}
}

func TestCheckDepth_DepthExceedsMax(t *testing.T) {
	resetEnv()
	os.Setenv(_envDepth, "5")
	os.Setenv(_envMax, "3")
	defer resetEnv()

	if err := NewGuard().CheckDepth(); err == nil {
		t.Error("expected error when depth > max, got nil")
	}
}

func TestIncrementDepth(t *testing.T) {
	resetEnv()
	os.Setenv(_envDepth, "1")
	defer resetEnv()

	NewGuard().IncrementDepth()

	got := os.Getenv(_envDepth)
	if got != "2" {
		t.Errorf("expected AI_INVOCATION_DEPTH=2, got %s", got)
	}
}

func TestIncrementDepth_FromZero(t *testing.T) {
	resetEnv()
	defer resetEnv()

	NewGuard().IncrementDepth()

	got := os.Getenv(_envDepth)
	if got != "1" {
		t.Errorf("expected AI_INVOCATION_DEPTH=1, got %s", got)
	}
}

func TestResetDepthRestoresPreviousValue(t *testing.T) {
	resetEnv()
	os.Setenv(_envDepth, "1")
	defer resetEnv()

	restore := NewGuard().ResetDepth()
	if got := os.Getenv(_envDepth); got != "0" {
		t.Fatalf("expected AI_INVOCATION_DEPTH=0 during round, got %s", got)
	}

	restore()
	if got := os.Getenv(_envDepth); got != "1" {
		t.Fatalf("expected AI_INVOCATION_DEPTH restored to 1, got %s", got)
	}
}

func TestResetDepthUnsetsWhenAbsent(t *testing.T) {
	resetEnv()
	defer resetEnv()

	restore := NewGuard().ResetDepth()
	restore()

	if _, ok := os.LookupEnv(_envDepth); ok {
		t.Fatal("expected AI_INVOCATION_DEPTH to be unset after restore")
	}
}

func TestResetDepthDoesNotChangeDefaultMax(t *testing.T) {
	if _defaultMax != 2 {
		t.Fatalf("_defaultMax = %d, want 2 (reset must not raise the ceiling)", _defaultMax)
	}
}
