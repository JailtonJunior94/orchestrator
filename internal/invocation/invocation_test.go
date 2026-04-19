package invocation

import (
	"os"
	"testing"
)

func resetEnv() {
	os.Unsetenv(envDepth)
	os.Unsetenv(envMax)
}

func TestCheckDepth_WithinLimit(t *testing.T) {
	resetEnv()
	os.Setenv(envDepth, "0")
	os.Setenv(envMax, "2")
	defer resetEnv()

	if err := CheckDepth(); err != nil {
		t.Errorf("expected no error, got: %v", err)
	}
}

func TestCheckDepth_AtLimit(t *testing.T) {
	resetEnv()
	os.Setenv(envDepth, "2")
	os.Setenv(envMax, "2")
	defer resetEnv()

	if err := CheckDepth(); err == nil {
		t.Error("expected error when depth >= max, got nil")
	}
}

func TestCheckDepth_NoEnvVars_Defaults(t *testing.T) {
	resetEnv()

	if err := CheckDepth(); err != nil {
		t.Errorf("expected no error with defaults (depth=0, max=2), got: %v", err)
	}
}

func TestCheckDepth_MaxZero_AlwaysBlocks(t *testing.T) {
	resetEnv()
	os.Setenv(envMax, "0")
	defer resetEnv()

	if err := CheckDepth(); err == nil {
		t.Error("expected error when max=0, got nil")
	}
}

func TestCheckDepth_DepthExceedsMax(t *testing.T) {
	resetEnv()
	os.Setenv(envDepth, "5")
	os.Setenv(envMax, "3")
	defer resetEnv()

	if err := CheckDepth(); err == nil {
		t.Error("expected error when depth > max, got nil")
	}
}

func TestIncrementDepth(t *testing.T) {
	resetEnv()
	os.Setenv(envDepth, "1")
	defer resetEnv()

	IncrementDepth()

	got := os.Getenv(envDepth)
	if got != "2" {
		t.Errorf("expected AI_INVOCATION_DEPTH=2, got %s", got)
	}
}

func TestIncrementDepth_FromZero(t *testing.T) {
	resetEnv()
	defer resetEnv()

	IncrementDepth()

	got := os.Getenv(envDepth)
	if got != "1" {
		t.Errorf("expected AI_INVOCATION_DEPTH=1, got %s", got)
	}
}
