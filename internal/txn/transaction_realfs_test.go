package txn_test

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/JailtonJunior94/ai-spec-harness/internal/fs"
	"github.com/JailtonJunior94/ai-spec-harness/internal/txn"
)

func treeSnapshot(t *testing.T, root string) string {
	t.Helper()
	var entries []string
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		rel, relErr := filepath.Rel(root, path)
		if relErr != nil {
			return relErr
		}
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		entries = append(entries, fmt.Sprintf("%s:%x", rel, sha256.Sum256(data)))
		return nil
	})
	if err != nil {
		t.Fatalf("walk %s: %v", root, err)
	}
	sort.Strings(entries)
	h := sha256.New()
	for _, e := range entries {
		fmt.Fprintln(h, e)
	}
	return fmt.Sprintf("%x", h.Sum(nil))
}

func requireNoTempFiles(t *testing.T, root string) {
	t.Helper()
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		if strings.HasPrefix(filepath.Base(path), ".tmp-") {
			t.Errorf("leftover temp file after failed commit: %s", path)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk %s: %v", root, err)
	}
}

func TestFileTransaction_RealFilesystem_BatchFailureLeavesTreeByteIdentical(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("root bypassa restricoes de permissao")
	}

	scenarios := []struct {
		name       string
		failTarget int
	}{
		{"fails_on_first_file", 0},
		{"fails_on_middle_file", 1},
		{"fails_on_last_file", 2},
	}

	for _, scenario := range scenarios {
		t.Run(scenario.name, func(t *testing.T) {
			root := t.TempDir()
			realFS := fs.NewOSFileSystem()

			dirs := []string{"alpha", "beta", "gamma"}
			paths := make([]string, len(dirs))
			for i, dir := range dirs {
				dirPath := filepath.Join(root, dir)
				if err := os.MkdirAll(dirPath, 0o755); err != nil {
					t.Fatalf("mkdir %s: %v", dirPath, err)
				}
				paths[i] = filepath.Join(dirPath, "SKILL.md")
				if err := os.WriteFile(paths[i], []byte(fmt.Sprintf("original content %d", i)), 0o644); err != nil {
					t.Fatalf("seed %s: %v", paths[i], err)
				}
			}

			before := treeSnapshot(t, root)

			lockedDir := filepath.Join(root, dirs[scenario.failTarget])
			if err := os.Chmod(lockedDir, 0o555); err != nil {
				t.Fatalf("chmod read-only %s: %v", lockedDir, err)
			}
			t.Cleanup(func() {
				_ = os.Chmod(lockedDir, 0o755)
			})

			tx := txn.New(realFS, root, nil)
			for i, path := range paths {
				if err := tx.Stage(path, []byte(fmt.Sprintf("updated content %d", i))); err != nil {
					t.Fatalf("stage %s: %v", path, err)
				}
			}

			err := tx.Commit()
			if err == nil {
				t.Fatal("commit must fail: one of the target directories is read-only")
			}

			if err := os.Chmod(lockedDir, 0o755); err != nil {
				t.Fatalf("restore permissions for verification: %v", err)
			}

			after := treeSnapshot(t, root)
			if before != after {
				t.Fatalf("tree hash diverged after a failed batch: before=%s after=%s — the batch must be all-or-nothing", before, after)
			}

			requireNoTempFiles(t, root)
		})
	}
}

func TestFileTransaction_RealFilesystem_SuccessfulCommitPersistsToDisk(t *testing.T) {
	root := t.TempDir()
	realFS := fs.NewOSFileSystem()

	tx := txn.New(realFS, root, nil)
	target := filepath.Join(root, "nested", "AGENTS.md")
	if err := tx.Stage(target, []byte("governance content")); err != nil {
		t.Fatalf("stage: %v", err)
	}
	if err := tx.MkdirAll(filepath.Dir(target)); err != nil {
		t.Fatalf("mkdirall: %v", err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatalf("commit: %v", err)
	}

	data, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("read committed file: %v", err)
	}
	if string(data) != "governance content" {
		t.Fatalf("committed content = %q, want %q", data, "governance content")
	}
	requireNoTempFiles(t, root)
}
