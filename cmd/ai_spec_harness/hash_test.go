package aispecharness

import (
	"os"
	"path/filepath"
	"testing"
)

func TestHashFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "input.txt")
	if err := os.WriteFile(path, []byte("abc"), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	got, err := hashFile(path)
	if err != nil {
		t.Fatalf("hashFile returned error: %v", err)
	}
	const want = "ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad"
	if got != want {
		t.Fatalf("hashFile() = %q, want %q", got, want)
	}
}
