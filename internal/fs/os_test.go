package fs_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/JailtonJunior94/ai-spec-harness/internal/fs"
)

func TestOS_WriteAndRead(t *testing.T) {
	dir := t.TempDir()
	f := fs.NewOSFileSystem()
	p := filepath.Join(dir, "sub", "file.txt")

	if err := f.WriteFile(p, []byte("hello")); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	data, err := f.ReadFile(p)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(data) != "hello" {
		t.Errorf("ReadFile = %q, want 'hello'", data)
	}
}

func TestOS_CopyFile_overwritesReadOnly(t *testing.T) {
	dir := t.TempDir()
	f := fs.NewOSFileSystem()
	src := filepath.Join(dir, "src.md")
	dst := filepath.Join(dir, "dst.md")

	if err := os.WriteFile(src, []byte("source"), 0o444); err != nil {
		t.Fatalf("seed src: %v", err)
	}
	if err := os.WriteFile(dst, []byte("old"), 0o444); err != nil {
		t.Fatalf("seed dst: %v", err)
	}
	if err := f.CopyFile(src, dst); err != nil {
		t.Fatalf("CopyFile over read-only: %v", err)
	}
	data, err := os.ReadFile(dst)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(data) != "source" {
		t.Errorf("content = %q, want 'source'", data)
	}
}

func TestOS_WriteFile_overwritesReadOnly(t *testing.T) {
	dir := t.TempDir()
	f := fs.NewOSFileSystem()
	p := filepath.Join(dir, "governance.md")

	if err := os.WriteFile(p, []byte("old"), 0o444); err != nil {
		t.Fatalf("seed: %v", err)
	}
	if err := f.WriteFile(p, []byte("new")); err != nil {
		t.Fatalf("WriteFile over read-only: %v", err)
	}
	data, err := os.ReadFile(p)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(data) != "new" {
		t.Errorf("content = %q, want 'new'", data)
	}
	info, err := os.Stat(p)
	if err != nil {
		t.Fatalf("Stat: %v", err)
	}
	if info.Mode().Perm()&0o200 == 0 {
		t.Errorf("file still not writable after WriteFile, mode = %v", info.Mode().Perm())
	}
}

func TestOS_ReadFile_missing(t *testing.T) {
	f := fs.NewOSFileSystem()
	_, err := f.ReadFile("/nonexistent/path/file.txt")
	if err == nil {
		t.Error("ReadFile on missing file should return error")
	}
}

func TestOS_Exists(t *testing.T) {
	dir := t.TempDir()
	f := fs.NewOSFileSystem()
	p := filepath.Join(dir, "e.txt")

	if f.Exists(p) {
		t.Error("Exists should be false before write")
	}
	_ = f.WriteFile(p, []byte("x"))
	if !f.Exists(p) {
		t.Error("Exists should be true after write")
	}
}

func TestOS_IsDir(t *testing.T) {
	dir := t.TempDir()
	f := fs.NewOSFileSystem()

	if !f.IsDir(dir) {
		t.Error("IsDir should be true for temp dir")
	}
	p := filepath.Join(dir, "file.txt")
	_ = f.WriteFile(p, []byte("x"))
	if f.IsDir(p) {
		t.Error("IsDir should be false for regular file")
	}
}

func TestOS_MkdirAll(t *testing.T) {
	dir := t.TempDir()
	f := fs.NewOSFileSystem()
	nested := filepath.Join(dir, "a", "b", "c")
	if err := f.MkdirAll(nested); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if !f.IsDir(nested) {
		t.Error("MkdirAll should create nested directories")
	}
}

func TestOS_Remove(t *testing.T) {
	dir := t.TempDir()
	f := fs.NewOSFileSystem()
	p := filepath.Join(dir, "r.txt")
	_ = f.WriteFile(p, []byte("x"))
	if err := f.Remove(p); err != nil {
		t.Fatalf("Remove: %v", err)
	}
	if f.Exists(p) {
		t.Error("file should not exist after Remove")
	}
}

func TestOS_RemoveAll(t *testing.T) {
	dir := t.TempDir()
	f := fs.NewOSFileSystem()
	sub := filepath.Join(dir, "tree")
	_ = f.WriteFile(filepath.Join(sub, "a.txt"), []byte("a"))
	_ = f.WriteFile(filepath.Join(sub, "b.txt"), []byte("b"))
	if err := f.RemoveAll(sub); err != nil {
		t.Fatalf("RemoveAll: %v", err)
	}
	if f.Exists(sub) {
		t.Error("directory should not exist after RemoveAll")
	}
}

func TestOS_CopyFile(t *testing.T) {
	dir := t.TempDir()
	f := fs.NewOSFileSystem()
	src := filepath.Join(dir, "src.txt")
	dst := filepath.Join(dir, "dst", "dst.txt")
	_ = os.WriteFile(src, []byte("copy me"), 0o644)

	if err := f.CopyFile(src, dst); err != nil {
		t.Fatalf("CopyFile: %v", err)
	}
	data, _ := os.ReadFile(dst)
	if string(data) != "copy me" {
		t.Errorf("CopyFile dst = %q, want 'copy me'", data)
	}
}

func TestOS_CopyFile_missing(t *testing.T) {
	f := fs.NewOSFileSystem()
	err := f.CopyFile("/no/src.txt", "/no/dst.txt")
	if err == nil {
		t.Error("CopyFile from missing file should return error")
	}
}

func TestOS_CopyDir(t *testing.T) {
	dir := t.TempDir()
	f := fs.NewOSFileSystem()
	src := filepath.Join(dir, "src")
	dst := filepath.Join(dir, "dst")
	_ = os.MkdirAll(src, 0o755)
	_ = os.WriteFile(filepath.Join(src, "file.txt"), []byte("hi"), 0o644)

	if err := f.CopyDir(src, dst); err != nil {
		t.Fatalf("CopyDir: %v", err)
	}
	data, _ := os.ReadFile(filepath.Join(dst, "file.txt"))
	if string(data) != "hi" {
		t.Errorf("CopyDir result = %q, want 'hi'", data)
	}
}

func TestOS_Symlink(t *testing.T) {
	dir := t.TempDir()
	f := fs.NewOSFileSystem()
	target := filepath.Join(dir, "target.txt")
	link := filepath.Join(dir, "link.txt")
	_ = os.WriteFile(target, []byte("t"), 0o644)

	if err := f.Symlink(target, link); err != nil {
		t.Fatalf("Symlink: %v", err)
	}
	if !f.IsSymlink(link) {
		t.Error("IsSymlink should be true after Symlink")
	}
}

func TestOS_FileHash(t *testing.T) {
	dir := t.TempDir()
	f := fs.NewOSFileSystem()
	p := filepath.Join(dir, "h.txt")
	_ = os.WriteFile(p, []byte("data"), 0o644)

	h, err := f.FileHash(p)
	if err != nil {
		t.Fatalf("FileHash: %v", err)
	}
	if h == "" {
		t.Error("FileHash should return non-empty string")
	}
}

func TestOS_FileHash_missing(t *testing.T) {
	f := fs.NewOSFileSystem()
	_, err := f.FileHash("/no/file")
	if err == nil {
		t.Error("FileHash on missing file should return error")
	}
}

func TestOS_DirHash(t *testing.T) {
	dir := t.TempDir()
	f := fs.NewOSFileSystem()
	_ = os.WriteFile(filepath.Join(dir, "a.txt"), []byte("a"), 0o644)
	_ = os.WriteFile(filepath.Join(dir, "b.txt"), []byte("b"), 0o644)

	h, err := f.DirHash(dir)
	if err != nil {
		t.Fatalf("DirHash: %v", err)
	}
	if h == "" {
		t.Error("DirHash should return non-empty hash")
	}
}

func TestOS_DirHash_notDir(t *testing.T) {
	f := fs.NewOSFileSystem()
	h, err := f.DirHash("/nonexistent/path")
	if err != nil {
		t.Fatalf("DirHash on non-dir should not error: %v", err)
	}
	if h != "" {
		t.Errorf("DirHash on non-dir should return empty, got %q", h)
	}
}

func TestOS_Writable(t *testing.T) {
	dir := t.TempDir()
	f := fs.NewOSFileSystem()
	p := filepath.Join(dir, "w.txt")
	_ = os.WriteFile(p, []byte("x"), 0o644)

	if !f.Writable(p) {
		t.Error("Writable should be true for 0644 file")
	}
}

func TestOS_Writable_missing(t *testing.T) {
	f := fs.NewOSFileSystem()
	if f.Writable("/no/such/path") {
		t.Error("Writable should be false for non-existent path")
	}
}

func TestOS_ReadDir(t *testing.T) {
	dir := t.TempDir()
	f := fs.NewOSFileSystem()
	_ = os.WriteFile(filepath.Join(dir, "a.txt"), []byte("a"), 0o644)
	_ = os.WriteFile(filepath.Join(dir, "b.txt"), []byte("b"), 0o644)

	entries, err := f.ReadDir(dir)
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	if len(entries) != 2 {
		t.Errorf("ReadDir count = %d, want 2", len(entries))
	}
}
