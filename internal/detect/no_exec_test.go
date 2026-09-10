package detect

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDetectionNeverExecutesBinaries(t *testing.T) {
	t.Parallel()

	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	forbidden := []string{"exec.Command", "exec.CommandContext", "cmd.Run(", "cmd.Start(", "cmd.Output(", "cmd.CombinedOutput("}
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		data, err := os.ReadFile(filepath.Clean(name))
		if err != nil {
			t.Fatalf("ReadFile %s: %v", name, err)
		}
		src := string(data)
		for _, bad := range forbidden {
			if strings.Contains(src, bad) {
				t.Errorf("%s contains %q — detection must not execute binaries (RF-26)", name, bad)
			}
		}
	}
}
