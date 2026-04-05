package output

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

// ExtractJSONL parses JSONL output emitted by Codex CLI with --json.
// It splits by newline, ignores blank lines, and extracts the textual content
// from the last event that contains a "type":"message" field or a "content" field.
// If no such event is found, it concatenates all "text" or "content" string values.
// A parse failure on any individual line returns a recoverable ProcessError.
func ExtractJSONL(raw string) (string, error) {
	lines := strings.Split(raw, "\n")

	var events []map[string]json.RawMessage
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		var event map[string]json.RawMessage
		if err := json.Unmarshal([]byte(line), &event); err != nil {
			return "", &ProcessError{
				Err:         fmt.Errorf("jsonl: parse error on line %q: %w", line, err),
				Recoverable: true,
			}
		}
		events = append(events, event)
	}

	if len(events) == 0 {
		return "", fmt.Errorf("jsonl: no events found in output")
	}

	// Search for the last event with "type":"message" or top-level content/message.
	for i := len(events) - 1; i >= 0; i-- {
		ev := events[i]
		if typeRaw, ok := ev["type"]; ok {
			var typVal string
			if err := json.Unmarshal(typeRaw, &typVal); err == nil && strings.EqualFold(typVal, "message") {
				if text := extractText(ev); text != "" {
					return text, nil
				}
			}
		}
		if _, ok := ev["content"]; ok {
			if text := extractText(ev); text != "" {
				return text, nil
			}
		}
		if _, ok := ev["message"]; ok {
			if text := extractText(ev); text != "" {
				return text, nil
			}
		}
	}

	// Fallback: concatenate all extracted text snippets across events.
	var parts []string
	for _, ev := range events {
		if text := extractText(ev); text != "" {
			parts = append(parts, text)
		}
	}
	if len(parts) > 0 {
		return strings.Join(parts, "\n"), nil
	}

	return "", fmt.Errorf("jsonl: no textual content found in events")
}

func extractText(ev map[string]json.RawMessage) string {
	for _, key := range []string{"content", "text", "message"} {
		if raw, ok := ev[key]; ok {
			if values := extractTextValues(raw); len(values) > 0 {
				return strings.Join(values, "\n")
			}
		}
	}

	keys := make([]string, 0, len(ev))
	for key := range ev {
		if key == "content" || key == "text" || key == "message" {
			continue
		}
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		if values := extractTextValues(ev[key]); len(values) > 0 {
			return strings.Join(values, "\n")
		}
	}

	return ""
}

func extractTextValues(raw json.RawMessage) []string {
	var value any
	if err := json.Unmarshal(raw, &value); err != nil {
		return nil
	}

	return collectTextValues(value)
}

func collectTextValues(value any) []string {
	switch typed := value.(type) {
	case string:
		if strings.TrimSpace(typed) == "" {
			return nil
		}
		return []string{typed}
	case []any:
		values := make([]string, 0)
		for _, item := range typed {
			values = append(values, collectTextValues(item)...)
		}
		return values
	case map[string]any:
		values := make([]string, 0)
		for _, key := range []string{"content", "text", "message"} {
			if nested, ok := typed[key]; ok {
				values = append(values, collectTextValues(nested)...)
			}
		}

		keys := make([]string, 0, len(typed))
		for key := range typed {
			if key == "content" || key == "text" || key == "message" {
				continue
			}
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			switch nested := typed[key].(type) {
			case map[string]any, []any:
				values = append(values, collectTextValues(nested)...)
			}
		}

		return values
	default:
		return nil
	}
}
