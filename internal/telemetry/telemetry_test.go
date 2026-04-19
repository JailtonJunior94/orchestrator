package telemetry

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestLog_NoWriteWhenTelemetryDisabled(t *testing.T) {
	os.Unsetenv("GOVERNANCE_TELEMETRY")
	dir := t.TempDir()

	if err := Log(dir, "bugfix", "bug.md"); err != nil {
		t.Fatalf("expected nil error, got: %v", err)
	}

	logPath := filepath.Join(dir, ".agents", "telemetry.log")
	if _, err := os.Stat(logPath); !os.IsNotExist(err) {
		t.Error("log file should not exist when GOVERNANCE_TELEMETRY != 1")
	}
}

func TestLog_WritesLineWhenEnabled(t *testing.T) {
	t.Setenv("GOVERNANCE_TELEMETRY", "1")
	dir := t.TempDir()

	before := time.Now().UTC().Add(-time.Second)
	if err := Log(dir, "bugfix", "bug-schema.json"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	logPath := filepath.Join(dir, ".agents", "telemetry.log")
	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("log file not created: %v", err)
	}

	content := string(data)
	if !strings.Contains(content, "skill=bugfix") {
		t.Errorf("expected skill=bugfix in log, got: %s", content)
	}
	if !strings.Contains(content, "ref=bug-schema.json") {
		t.Errorf("expected ref=bug-schema.json in log, got: %s", content)
	}

	// Verify timestamp is valid RFC3339
	line := strings.TrimSpace(content)
	parts := strings.Fields(line)
	if len(parts) < 1 {
		t.Fatal("empty log line")
	}
	ts, err := time.Parse(time.RFC3339, parts[0])
	if err != nil {
		t.Fatalf("invalid RFC3339 timestamp: %v", err)
	}
	if ts.Before(before) {
		t.Error("timestamp is before test start")
	}
}

func TestSummary_AggregatesCorrectly(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, ".agents", "telemetry.log")
	if err := os.MkdirAll(filepath.Dir(logPath), 0755); err != nil {
		t.Fatalf("failed to create dir: %v", err)
	}

	now := time.Now().UTC()
	content := fmt.Sprintf(
		"%s skill=bugfix ref=bug.md\n%s skill=review ref=schema.json\n%s skill=bugfix ref=bug.md\n",
		now.Format(time.RFC3339),
		now.Format(time.RFC3339),
		now.Format(time.RFC3339),
	)
	if err := os.WriteFile(logPath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write log: %v", err)
	}

	result, err := Summary(dir, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(result, "bugfix: 2") {
		t.Errorf("expected bugfix: 2 in summary, got:\n%s", result)
	}
	if !strings.Contains(result, "review: 1") {
		t.Errorf("expected review: 1 in summary, got:\n%s", result)
	}
	if !strings.Contains(result, "Custo Estimado") {
		t.Errorf("expected cost breakdown in summary, got:\n%s", result)
	}
	if !strings.Contains(result, "loaded") {
		t.Errorf("expected loaded axis in summary, got:\n%s", result)
	}
	if !strings.Contains(result, "incremental-ref") {
		t.Errorf("expected incremental-ref axis in summary, got:\n%s", result)
	}
}
