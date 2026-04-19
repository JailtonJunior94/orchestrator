package fs_test

import (
	"testing"

	"github.com/JailtonJunior94/ai-spec-harness/internal/fs"
)

func TestFake_WriteAndRead(t *testing.T) {
	f := fs.NewFakeFileSystem()
	if err := f.WriteFile("/a/b.txt", []byte("hello")); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	data, err := f.ReadFile("/a/b.txt")
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(data) != "hello" {
		t.Errorf("ReadFile = %q, want 'hello'", data)
	}
}

func TestFake_ReadFile_missing(t *testing.T) {
	f := fs.NewFakeFileSystem()
	_, err := f.ReadFile("/no/such/file")
	if err == nil {
		t.Error("ReadFile on missing file should return error")
	}
}

func TestFake_Exists(t *testing.T) {
	f := fs.NewFakeFileSystem()
	if f.Exists("/x") {
		t.Error("Exists should be false before write")
	}
	_ = f.WriteFile("/x", []byte("data"))
	if !f.Exists("/x") {
		t.Error("Exists should be true after write")
	}
}

func TestFake_Exists_dir(t *testing.T) {
	f := fs.NewFakeFileSystem()
	_ = f.MkdirAll("/mydir")
	if !f.Exists("/mydir") {
		t.Error("Exists should be true for created directory")
	}
}

func TestFake_IsDir(t *testing.T) {
	f := fs.NewFakeFileSystem()
	_ = f.MkdirAll("/d")
	if !f.IsDir("/d") {
		t.Error("IsDir should be true after MkdirAll")
	}
	_ = f.WriteFile("/d/file.txt", []byte("x"))
	// A path with children is also treated as a dir
	if !f.IsDir("/d") {
		t.Error("IsDir should be true for path with child files")
	}
}

func TestFake_IsDir_file(t *testing.T) {
	f := fs.NewFakeFileSystem()
	_ = f.WriteFile("/f.txt", []byte("x"))
	if f.IsDir("/f.txt") {
		t.Error("IsDir should be false for a regular file")
	}
}

func TestFake_IsSymlink(t *testing.T) {
	f := fs.NewFakeFileSystem()
	_ = f.Symlink("/target", "/link")
	if !f.IsSymlink("/link") {
		t.Error("IsSymlink should be true after Symlink")
	}
	if f.IsSymlink("/target") {
		t.Error("IsSymlink should be false for non-link path")
	}
}

func TestFake_Remove(t *testing.T) {
	f := fs.NewFakeFileSystem()
	_ = f.WriteFile("/r.txt", []byte("x"))
	if err := f.Remove("/r.txt"); err != nil {
		t.Fatalf("Remove: %v", err)
	}
	if f.Exists("/r.txt") {
		t.Error("file should not exist after Remove")
	}
}

func TestFake_RemoveAll(t *testing.T) {
	f := fs.NewFakeFileSystem()
	_ = f.WriteFile("/tree/a.txt", []byte("a"))
	_ = f.WriteFile("/tree/b.txt", []byte("b"))
	if err := f.RemoveAll("/tree"); err != nil {
		t.Fatalf("RemoveAll: %v", err)
	}
	if f.Exists("/tree/a.txt") || f.Exists("/tree/b.txt") {
		t.Error("files should not exist after RemoveAll")
	}
}

func TestFake_CopyFile(t *testing.T) {
	f := fs.NewFakeFileSystem()
	_ = f.WriteFile("/src.txt", []byte("content"))
	if err := f.CopyFile("/src.txt", "/dst.txt"); err != nil {
		t.Fatalf("CopyFile: %v", err)
	}
	data, _ := f.ReadFile("/dst.txt")
	if string(data) != "content" {
		t.Errorf("CopyFile dst = %q, want 'content'", data)
	}
}

func TestFake_CopyFile_missing(t *testing.T) {
	f := fs.NewFakeFileSystem()
	if err := f.CopyFile("/missing.txt", "/dst.txt"); err == nil {
		t.Error("CopyFile from missing file should return error")
	}
}

func TestFake_CopyDir(t *testing.T) {
	f := fs.NewFakeFileSystem()
	_ = f.WriteFile("/src/a.txt", []byte("a"))
	_ = f.WriteFile("/src/b.txt", []byte("b"))
	if err := f.CopyDir("/src", "/dst"); err != nil {
		t.Fatalf("CopyDir: %v", err)
	}
	data, _ := f.ReadFile("/dst/a.txt")
	if string(data) != "a" {
		t.Errorf("CopyDir /dst/a.txt = %q, want 'a'", data)
	}
}

func TestFake_FileHash(t *testing.T) {
	f := fs.NewFakeFileSystem()
	_ = f.WriteFile("/h.txt", []byte("data"))
	h1, err := f.FileHash("/h.txt")
	if err != nil {
		t.Fatalf("FileHash: %v", err)
	}
	if h1 == "" {
		t.Error("FileHash should return non-empty hash")
	}
	// Same content same hash
	_ = f.WriteFile("/h2.txt", []byte("data"))
	h2, _ := f.FileHash("/h2.txt")
	if h1 != h2 {
		t.Error("FileHash should be deterministic for same content")
	}
}

func TestFake_FileHash_missing(t *testing.T) {
	f := fs.NewFakeFileSystem()
	_, err := f.FileHash("/missing")
	if err == nil {
		t.Error("FileHash on missing file should return error")
	}
}

func TestFake_DirHash(t *testing.T) {
	f := fs.NewFakeFileSystem()
	_ = f.WriteFile("/dir/a.txt", []byte("a"))
	_ = f.WriteFile("/dir/b.txt", []byte("b"))
	h, err := f.DirHash("/dir")
	if err != nil {
		t.Fatalf("DirHash: %v", err)
	}
	if h == "" {
		t.Error("DirHash should return non-empty hash")
	}
}

func TestFake_DirHash_notDir(t *testing.T) {
	f := fs.NewFakeFileSystem()
	h, err := f.DirHash("/nonexistent")
	if err != nil {
		t.Fatalf("DirHash on non-dir should not error: %v", err)
	}
	if h != "" {
		t.Errorf("DirHash on non-dir should return empty, got %q", h)
	}
}

func TestFake_Writable(t *testing.T) {
	f := fs.NewFakeFileSystem()
	_ = f.WriteFile("/w.txt", []byte("x"))
	if !f.Writable("/w.txt") {
		t.Error("Writable should be true by default")
	}
	f.NoWrite["/w.txt"] = true
	if f.Writable("/w.txt") {
		t.Error("Writable should be false when NoWrite is set")
	}
}

func TestFake_ReadDir(t *testing.T) {
	f := fs.NewFakeFileSystem()
	_ = f.WriteFile("/mydir/a.txt", []byte("a"))
	_ = f.WriteFile("/mydir/b.txt", []byte("b"))
	entries, err := f.ReadDir("/mydir")
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	if len(entries) != 2 {
		t.Errorf("ReadDir count = %d, want 2", len(entries))
	}
}
