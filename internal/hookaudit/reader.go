package hookaudit

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

var ErrCorruptEntry = errors.New("corrupt hook audit log entry")

type Reader struct {
	path string
}

func NewReader(path string) *Reader {
	return &Reader{path: filepath.Clean(path)}
}

func (r *Reader) ReadAll() ([]Entry, error) {
	file, err := os.Open(r.path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("hookaudit: open %s: %w", r.path, err)
	}
	defer file.Close()

	var entries []Entry
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	lineNumber := 0
	for scanner.Scan() {
		lineNumber++
		line := scanner.Bytes()
		if len(bytes.TrimSpace(line)) == 0 {
			continue
		}
		var entry Entry
		if err := json.Unmarshal(line, &entry); err != nil {
			return nil, fmt.Errorf("%w: line %d in %s: %v", ErrCorruptEntry, lineNumber, r.path, err)
		}
		entries = append(entries, entry)
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("hookaudit: scan %s: %w", r.path, err)
	}
	return entries, nil
}
