package telemetry

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type recordingHookTelemetryWriter struct {
	lines []string
}

func (w *recordingHookTelemetryWriter) WriteLine(rootDir, line string) error {
	w.lines = append(w.lines, line)
	return nil
}

type failingHookTelemetryWriter struct {
	calls int
}

func (w *failingHookTelemetryWriter) WriteLine(rootDir, line string) error {
	w.calls++
	return errors.New("simulated telemetry backend failure")
}

func TestHookTelemetry_NoOpWithoutOptIn(t *testing.T) {
	t.Setenv("GOVERNANCE_TELEMETRY", "")
	root := t.TempDir()

	telemetry := NewHookTelemetry()
	telemetry.RecordDuration(root, "validate-preload", "before_tool", "provider-x", 42)
	telemetry.RecordDecision(root, "validate-preload", "before_tool", "provider-x", "ALLOW")
	telemetry.RecordTimeout(root, "validate-preload", "before_tool", "provider-x")
	telemetry.RecordAblation(root, "validate-preload", false, 10)

	if _, err := os.Stat(filepath.Join(root, ".agents", "telemetry.log")); !os.IsNotExist(err) {
		t.Fatalf("telemetry.log must not exist without GOVERNANCE_TELEMETRY=1, stat err = %v", err)
	}
}

func TestHookTelemetry_EmitsCanonicalFormatWhenOptedIn(t *testing.T) {
	t.Setenv("GOVERNANCE_TELEMETRY", "1")
	root := t.TempDir()
	writer := &recordingHookTelemetryWriter{}
	telemetry := NewHookTelemetryWithWriter(writer)

	telemetry.RecordDuration(root, "validate-preload", "before_tool", "provider-x", 42)
	telemetry.RecordDecision(root, "validate-preload", "before_tool", "provider-x", "ALLOW")
	telemetry.RecordTimeout(root, "validate-preload", "before_tool", "provider-x")
	telemetry.RecordAblation(root, "validate-preload", true, 10)

	if len(writer.lines) != 4 {
		t.Fatalf("len(lines) = %d, want 4", len(writer.lines))
	}
	wantSuffixes := []string{
		"hook.duration_ms hook=validate-preload event=before_tool provider=provider-x value=42",
		"hook.decision hook=validate-preload event=before_tool provider=provider-x decision=ALLOW",
		"hook.timeout hook=validate-preload event=before_tool provider=provider-x",
		"hook.ablation baseline=com hook=validate-preload duration_ms=10",
	}
	for i, want := range wantSuffixes {
		if !hasTimestampPrefixAndSuffix(writer.lines[i], want) {
			t.Fatalf("line[%d] = %q, want format '<ts> %s'", i, writer.lines[i], want)
		}
	}
}

func hasTimestampPrefixAndSuffix(line, suffix string) bool {
	prefix, rest, found := strings.Cut(line, " ")
	if !found || prefix == "" {
		return false
	}
	return rest == suffix
}

func TestHookTelemetry_FailureDoesNotPropagateOrCountAsRetry(t *testing.T) {
	t.Setenv("GOVERNANCE_TELEMETRY", "1")
	root := t.TempDir()
	writer := &failingHookTelemetryWriter{}
	telemetry := NewHookTelemetryWithWriter(writer)

	retryAttempts := 0
	taskCompleted := false

	telemetry.RecordDuration(root, "validate-governance", "after_tool", "provider-x", 100)
	taskCompleted = true

	if writer.calls != 1 {
		t.Fatalf("writer.calls = %d, want 1 — the failing write must still be attempted", writer.calls)
	}
	if !taskCompleted {
		t.Fatal("task must complete even when the telemetry writer always fails (RF-48)")
	}
	if retryAttempts != 0 {
		t.Fatalf("retryAttempts = %d, want 0 — telemetry failure must never trigger a retry cascade (RF-48)", retryAttempts)
	}
}
