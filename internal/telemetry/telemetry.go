package telemetry

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// Log registra uso de skill e referencia em .agents/telemetry.log apenas quando
// a variavel de ambiente GOVERNANCE_TELEMETRY estiver definida como "1".
func Log(rootDir, skill, ref string) error {
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
	line := fmt.Sprintf("%s skill=%s ref=%s\n", ts, skill, ref)
	_, err = f.WriteString(line)
	return err
}
