package catalog

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestSkillsLockDoesNotIncludeBubbletea(t *testing.T) {
	t.Parallel()

	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller() failed")
	}

	lockPath := filepath.Join(filepath.Dir(currentFile), "..", "..", "..", "skills-lock.json")
	contents, err := os.ReadFile(lockPath)
	if err != nil {
		t.Fatalf("ReadFile(%q) error = %v", lockPath, err)
	}

	var lock struct {
		Skills map[string]json.RawMessage `json:"skills"`
	}
	if err := json.Unmarshal(contents, &lock); err != nil {
		t.Fatalf("json.Unmarshal(%q) error = %v", lockPath, err)
	}

	if _, exists := lock.Skills["bubbletea"]; exists {
		t.Fatalf("skills-lock.json unexpectedly includes %q", "bubbletea")
	}
}
