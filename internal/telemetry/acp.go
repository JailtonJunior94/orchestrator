package telemetry

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// ACPSessionEvent representa os campos opcionais de telemetria para uma sessão ACP.
// Emitido apenas quando runtime=acp; o caminho legacy não emite esses campos.
type ACPSessionEvent struct {
	// Runtime é o tipo de runtime usado: "acp" ou "legacy".
	Runtime string
	// Launcher é o tipo de launcher resolvido: "binary" ou "npx".
	Launcher string
	// EventsCount é o número de eventos mapeados recebidos (exclui unknown).
	EventsCount int
	// UnknownEventsCount é o número de eventos de kind desconhecido.
	UnknownEventsCount int
	// CancelReason é o motivo de cancelamento: none, activity_timeout, context_canceled,
	// tool_error, permission_denied.
	CancelReason string
}

// LogACPSession registra campos de sessão ACP em .agents/telemetry.log apenas quando
// GOVERNANCE_TELEMETRY=1. O caminho legacy não deve chamar esta função.
// Compatibilidade com leitores atuais: campos novos são adicionados como atributos extras
// na linha de log; parsers que não entendem esses campos simplesmente os ignoram.
func LogACPSession(rootDir string, evt ACPSessionEvent) error {
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
		"%s skill=task-loop ref=acp-session runtime=%s launcher=%s events_count=%d unknown_events_count=%d cancel_reason=%s\n",
		ts,
		evt.Runtime,
		evt.Launcher,
		evt.EventsCount,
		evt.UnknownEventsCount,
		evt.CancelReason,
	)
	_, err = f.WriteString(line)
	return err
}
