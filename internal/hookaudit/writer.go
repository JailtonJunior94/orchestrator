package hookaudit

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

type Writer struct {
	path string
	mu   sync.Mutex
}

func NewWriter(path string) (*Writer, error) {
	clean := filepath.Clean(path)
	if clean == "" || clean == "." {
		return nil, fmt.Errorf("hookaudit: invalid writer path %q", path)
	}
	dir := filepath.Dir(clean)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("hookaudit: create directory %s: %w", dir, err)
	}
	return &Writer{path: clean}, nil
}

func (w *Writer) Append(entry Entry) error {
	line, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("hookaudit: marshal entry: %w", err)
	}

	w.mu.Lock()
	defer w.mu.Unlock()

	file, err := os.OpenFile(w.path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("hookaudit: open %s: %w", w.path, err)
	}
	defer file.Close()

	if _, err := file.Write(append(line, '\n')); err != nil {
		return fmt.Errorf("hookaudit: append entry to %s: %w", w.path, err)
	}
	return nil
}

func (w *Writer) Path() string {
	return w.path
}
