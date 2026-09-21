package persistence

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"sync"

	"github.com/JailtonJunior94/ai-spec-harness/internal/fs"
	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/events"
	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/memory/durable"
)

var ErrCorruptedJSONL = errors.New("persistence: corrupted jsonl content")

type JSONLWriter struct {
	path string
	fsys fs.FileSystem
	mu   sync.Mutex
}

func NewJSONLWriter(path string, fsys fs.FileSystem) (*JSONLWriter, error) {
	clean := filepath.Clean(path)
	if clean == "" || clean == "." {
		return nil, fmt.Errorf("persistence: invalid path for JSONLWriter: %q", path)
	}
	dir := filepath.Dir(clean)
	if err := fsys.MkdirAll(dir); err != nil {
		return nil, fmt.Errorf("persistence: create directory %s: %w", dir, err)
	}
	if existing, err := fsys.ReadFile(clean); err == nil {
		if verifyErr := VerifyJSONLIntegrity(existing); verifyErr != nil {
			repaired, repairable := repairTrailingCorruption(existing)
			if !repairable {
				return nil, fmt.Errorf("persistence: %s: %w", clean, verifyErr)
			}
			if err := fsys.WriteFileAtomic(clean, repaired); err != nil {
				return nil, fmt.Errorf("persistence: repair %s: %w", clean, err)
			}
		}
	}
	return &JSONLWriter{path: clean, fsys: fsys}, nil
}

func repairTrailingCorruption(content []byte) ([]byte, bool) {
	if len(content) == 0 {
		return content, false
	}
	hasTrailingNewline := content[len(content)-1] == '\n'
	body := content
	if hasTrailingNewline {
		body = content[:len(content)-1]
	}
	lastNewline := bytes.LastIndexByte(body, '\n')
	var prefix, lastLine []byte
	if lastNewline == -1 {
		lastLine = body
	} else {
		prefix = body[:lastNewline+1]
		lastLine = body[lastNewline+1:]
	}
	if verifyErr := VerifyJSONLIntegrity(prefix); verifyErr != nil {
		return content, false
	}
	trimmedLast := bytes.TrimSpace(lastLine)
	if len(trimmedLast) == 0 {
		return prefix, true
	}
	var probe json.RawMessage
	if jsonErr := json.Unmarshal(lastLine, &probe); jsonErr != nil {
		return prefix, true
	}
	if !hasTrailingNewline {
		return append(append([]byte{}, content...), '\n'), true
	}
	return content, false
}

func (w *JSONLWriter) Append(evt events.Event) error {
	data, err := evt.MarshalJSON()
	if err != nil {
		return fmt.Errorf("appending event to %s: %w", w.path, err)
	}
	sanitized, err := durable.DefaultSanitizationPolicy.Sanitize(string(data), durable.SanitizationConfig{})
	if err != nil {
		return fmt.Errorf("appending event to %s: %w", w.path, err)
	}
	line := append([]byte(sanitized.Content), '\n')

	w.mu.Lock()
	defer w.mu.Unlock()

	if err := w.fsys.AppendFile(w.path, line); err != nil {
		return fmt.Errorf("appending event to %s: %w", w.path, err)
	}
	return nil
}

func VerifyJSONLIntegrity(content []byte) error {
	if len(content) == 0 {
		return nil
	}
	if content[len(content)-1] != '\n' {
		return fmt.Errorf("%w: last line has no terminator (truncated mid-write)", ErrCorruptedJSONL)
	}
	trimmed := bytes.TrimSuffix(content, []byte("\n"))
	lines := bytes.Split(trimmed, []byte("\n"))
	for i, line := range lines {
		if len(bytes.TrimSpace(line)) == 0 {
			return fmt.Errorf("%w: empty line at position %d", ErrCorruptedJSONL, i+1)
		}
		var probe json.RawMessage
		if err := json.Unmarshal(line, &probe); err != nil {
			return fmt.Errorf("%w: invalid json at line %d: %w", ErrCorruptedJSONL, i+1, err)
		}
	}
	return nil
}
