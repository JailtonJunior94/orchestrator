package telemetry

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"time"
)

type logEntry struct {
	Timestamp time.Time
	Skill     string
	Ref       string // vazio se a linha não tem campo ref=
}

// parseLogEntries lê o arquivo de log em logPath e retorna as entradas
// dentro da janela de tempo definida por since (0 = sem filtro).
// Linhas malformadas são ignoradas silenciosamente.
// Retorna slice vazio e err=nil se o arquivo não existir.
func parseLogEntries(logPath string, since time.Duration) ([]logEntry, error) {
	f, err := os.Open(logPath)
	if err != nil {
		if os.IsNotExist(err) {
			return []logEntry{}, nil
		}
		return nil, fmt.Errorf("abrir log de telemetria: %w", err)
	}
	defer f.Close()

	var cutoff time.Time
	if since > 0 {
		cutoff = time.Now().UTC().Add(-since)
	}

	entries := []logEntry{}
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.Fields(line)
		if len(parts) < 2 {
			continue
		}
		ts, err := time.Parse(time.RFC3339, parts[0])
		if err != nil {
			continue
		}
		if !cutoff.IsZero() && ts.Before(cutoff) {
			continue
		}

		entry := logEntry{Timestamp: ts}
		for _, part := range parts[1:] {
			switch {
			case strings.HasPrefix(part, "skill="):
				entry.Skill = strings.TrimPrefix(part, "skill=")
			case strings.HasPrefix(part, "ref="):
				entry.Ref = strings.TrimPrefix(part, "ref=")
			}
		}
		entries = append(entries, entry)
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("ler log de telemetria: %w", err)
	}
	return entries, nil
}
