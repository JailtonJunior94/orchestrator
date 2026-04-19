package telemetry

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// Summary le .agents/telemetry.log, filtra por timestamp >= now-since (zero = sem filtro),
// conta por skill e por ref, estima tokens adicionais (refs × 500) e retorna string formatada.
func Summary(rootDir string, since time.Duration) (string, error) {
	logPath := filepath.Join(rootDir, ".agents", "telemetry.log")
	f, err := os.Open(logPath)
	if err != nil {
		if os.IsNotExist(err) {
			return "Sem dados de telemetria.", nil
		}
		return "", fmt.Errorf("abrir log de telemetria: %w", err)
	}
	defer f.Close()

	var cutoff time.Time
	if since > 0 {
		cutoff = time.Now().UTC().Add(-since)
	}

	skillCounts := make(map[string]int)
	refCounts := make(map[string]int)
	totalLines := 0

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

		totalLines++
		for _, part := range parts[1:] {
			if strings.HasPrefix(part, "skill=") {
				skill := strings.TrimPrefix(part, "skill=")
				skillCounts[skill]++
			} else if strings.HasPrefix(part, "ref=") {
				ref := strings.TrimPrefix(part, "ref=")
				refCounts[ref]++
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("ler log de telemetria: %w", err)
	}

	if totalLines == 0 {
		return "Sem dados de telemetria no periodo especificado.", nil
	}

	var sb strings.Builder
	fmt.Fprintf(&sb, "Telemetria (%d entradas)\n\n", totalLines)

	fmt.Fprintf(&sb, "Skills:\n")
	skillKeys := sortedKeys(skillCounts)
	for _, skill := range skillKeys {
		fmt.Fprintf(&sb, "  %s: %d\n", skill, skillCounts[skill])
	}

	fmt.Fprintf(&sb, "\nReferencias:\n")
	refKeys := sortedKeys(refCounts)
	for _, ref := range refKeys {
		fmt.Fprintf(&sb, "  %s: %d\n", ref, refCounts[ref])
	}

	estimatedTokens := len(refCounts) * 500
	fmt.Fprintf(&sb, "\nTokens estimados adicionais: %d\n", estimatedTokens)

	return sb.String(), nil
}

func sortedKeys(m map[string]int) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
