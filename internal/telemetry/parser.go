package telemetry

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"time"
)

var rf44MetricNameToProductionLogKey = map[string]string{
	"provider":     "provider",
	"model":        "model",
	"duration":     "duration_ms",
	"tool_calls":   "tool_calls",
	"final_status": "final_status",
	"retries":      "retry_attempts",
}

func RF44MetricNames() []string {
	return []string{
		"provider", "model", "task_type", "risk", "tokens_in", "tokens_out",
		"duration", "retries", "tool_calls", "context_loaded", "skills_loaded",
		"review_rounds", "tests_passed", "human_intervention", "final_status",
	}
}

func RF44MetricsWithProductionWriter() []string {
	out := make([]string, 0, len(rf44MetricNameToProductionLogKey))
	for _, name := range RF44MetricNames() {
		if _, ok := rf44MetricNameToProductionLogKey[name]; ok {
			out = append(out, name)
		}
	}
	return out
}

type logEntry struct {
	Timestamp time.Time
	Skill     string
	Ref       string
	Fields    map[string]string
}

func (e logEntry) Metric(name string) (string, bool) {
	rawKey, known := rf44MetricNameToProductionLogKey[name]
	if !known {
		return "", false
	}
	v, ok := e.Fields[rawKey]
	return v, ok
}

func buildRecognizedRawKeys() map[string]bool {
	out := make(map[string]bool, len(rf44MetricNameToProductionLogKey))
	for _, rawKey := range rf44MetricNameToProductionLogKey {
		out[rawKey] = true
	}
	return out
}

var recognizedRawKeys = buildRecognizedRawKeys()

func (c *Catalog) parseLogEntries(logPath string, since time.Duration) ([]logEntry, error) {
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
			key, value, hasEq := strings.Cut(part, "=")
			if !hasEq {
				continue
			}
			switch key {
			case "skill":
				entry.Skill = value
			case "ref":
				entry.Ref = value
			default:
				if !recognizedRawKeys[key] {
					continue
				}
				if entry.Fields == nil {
					entry.Fields = make(map[string]string)
				}
				entry.Fields[key] = value
			}
		}
		entries = append(entries, entry)
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("ler log de telemetria: %w", err)
	}
	return entries, nil
}
