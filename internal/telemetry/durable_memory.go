package telemetry

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

type DurableMemorySessionEvent struct {
	SessionID      string
	CLI            string
	FactsWritten   int
	FactsArchived  int
	Redactions     int
	Compactions    int
	Contradictions int
	BatonClaimed   bool
}

func (c *Catalog) LogDurableMemorySession(rootDir string, evt DurableMemorySessionEvent) error {
	if os.Getenv("GOVERNANCE_TELEMETRY") != "1" {
		return nil
	}

	logDir := filepath.Join(rootDir, ".agents")
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return fmt.Errorf("criar diretorio de telemetria: %w", err)
	}

	logPath := filepath.Join(logDir, "telemetry.log")
	f, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("abrir log de telemetria: %w", err)
	}
	defer f.Close()

	ts := time.Now().UTC().Format(time.RFC3339)
	line := fmt.Sprintf(
		"%s skill=task-loop ref=durable-memory-session session_id=%s cli=%s facts_written=%d facts_archived=%d redactions=%d compactions=%d contradictions=%d baton_claimed=%v",
		ts,
		evt.SessionID,
		evt.CLI,
		evt.FactsWritten,
		evt.FactsArchived,
		evt.Redactions,
		evt.Compactions,
		evt.Contradictions,
		evt.BatonClaimed,
	)
	line += "\n"
	_, err = f.WriteString(line)
	return err
}
