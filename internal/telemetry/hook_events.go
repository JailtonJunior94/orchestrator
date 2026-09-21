package telemetry

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

type HookTelemetryWriter interface {
	WriteLine(rootDir, line string) error
}

type fileHookTelemetryWriter struct{}

func (fileHookTelemetryWriter) WriteLine(rootDir, line string) error {
	logDir := filepath.Join(rootDir, ".agents")
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return fmt.Errorf("create telemetry directory: %w", err)
	}

	logPath := filepath.Join(logDir, "telemetry.log")
	f, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("open telemetry log: %w", err)
	}
	defer f.Close()

	_, err = f.WriteString(line + "\n")
	return err
}

type HookTelemetry struct {
	writer HookTelemetryWriter
}

func NewHookTelemetry() HookTelemetry {
	return HookTelemetry{writer: fileHookTelemetryWriter{}}
}

func NewHookTelemetryWithWriter(writer HookTelemetryWriter) HookTelemetry {
	return HookTelemetry{writer: writer}
}

func (h HookTelemetry) RecordDuration(rootDir, hook, event, provider string, durationMs int64) {
	h.record(rootDir, fmt.Sprintf("hook.duration_ms hook=%s event=%s provider=%s value=%d", hook, event, provider, durationMs))
}

func (h HookTelemetry) RecordDecision(rootDir, hook, event, provider, decision string) {
	h.record(rootDir, fmt.Sprintf("hook.decision hook=%s event=%s provider=%s decision=%s", hook, event, provider, decision))
}

func (h HookTelemetry) RecordTimeout(rootDir, hook, event, provider string) {
	h.record(rootDir, fmt.Sprintf("hook.timeout hook=%s event=%s provider=%s", hook, event, provider))
}

func (h HookTelemetry) RecordAblation(rootDir, hook string, withComponent bool, durationMs int64) {
	baselineLabel := "sem"
	if withComponent {
		baselineLabel = "com"
	}
	h.record(rootDir, fmt.Sprintf("hook.ablation baseline=%s hook=%s duration_ms=%d", baselineLabel, hook, durationMs))
}

func (h HookTelemetry) record(rootDir, body string) {
	if os.Getenv("GOVERNANCE_TELEMETRY") != "1" {
		return
	}
	ts := time.Now().UTC().Format(time.RFC3339)
	_ = h.writer.WriteLine(rootDir, fmt.Sprintf("%s %s", ts, body))
}
