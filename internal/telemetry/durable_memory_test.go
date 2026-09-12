package telemetry

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLogDurableMemorySession_OptInWritesFields(t *testing.T) {
	t.Setenv("GOVERNANCE_TELEMETRY", "1")
	dir := t.TempDir()

	evt := DurableMemorySessionEvent{
		SessionID:      "sess-1",
		CLI:            "claude",
		FactsWritten:   2,
		FactsArchived:  1,
		Redactions:     1,
		Compactions:    1,
		Contradictions: 1,
		BatonClaimed:   true,
	}

	if err := NewCatalog().LogDurableMemorySession(dir, evt); err != nil {
		t.Fatalf("LogDurableMemorySession: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, ".agents", "telemetry.log"))
	if err != nil {
		t.Fatalf("log file not created: %v", err)
	}
	content := string(data)

	for _, field := range []string{
		"ref=durable-memory-session",
		"session_id=sess-1",
		"cli=claude",
		"facts_written=2",
		"facts_archived=1",
		"redactions=1",
		"compactions=1",
		"contradictions=1",
		"baton_claimed=true",
	} {
		if !strings.Contains(content, field) {
			t.Errorf("campo %q ausente no log; log:\n%s", field, content)
		}
	}
}

func TestLogDurableMemorySession_OptOutWritesNothing(t *testing.T) {
	t.Setenv("GOVERNANCE_TELEMETRY", "0")
	dir := t.TempDir()

	if err := NewCatalog().LogDurableMemorySession(dir, DurableMemorySessionEvent{SessionID: "sess-1"}); err != nil {
		t.Fatalf("LogDurableMemorySession: %v", err)
	}

	if _, err := os.Stat(filepath.Join(dir, ".agents", "telemetry.log")); !os.IsNotExist(err) {
		t.Errorf("telemetry.log não deveria existir com GOVERNANCE_TELEMETRY desligado; err=%v", err)
	}
}
