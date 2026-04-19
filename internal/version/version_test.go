package version

import (
	"os"
	"path/filepath"
	"testing"
)

func TestReadVersionFile_Exists(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "VERSION"), []byte("  1.2.3\n"), 0644); err != nil {
		t.Fatal(err)
	}
	got := ReadVersionFile(dir)
	if got != "1.2.3" {
		t.Errorf("ReadVersionFile: got %q, want %q", got, "1.2.3")
	}
}

func TestReadVersionFile_Missing(t *testing.T) {
	dir := t.TempDir()
	got := ReadVersionFile(dir)
	if got != "unknown" {
		t.Errorf("ReadVersionFile: got %q, want %q", got, "unknown")
	}
}
