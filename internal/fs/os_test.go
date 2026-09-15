package fs_test

import (
	"os"
	"path"
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

func TestOS_WriteFileAtomic_writeAndRead(t *testing.T) {
	dir := t.TempDir()
	f := fs.NewOSFileSystem()
	p := filepath.Join(dir, "sub", "atomic.txt")

	if err := f.WriteFileAtomic(p, []byte("hello")); err != nil {
		t.Fatalf("WriteFileAtomic: %v", err)
	}
	data, err := f.ReadFile(p)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(data) != "hello" {
		t.Errorf("ReadFile = %q, want 'hello'", data)
	}
}

func TestOS_WriteFileAtomic_overwritesReadOnly(t *testing.T) {
	dir := t.TempDir()
	f := fs.NewOSFileSystem()
	p := filepath.Join(dir, "atomic.txt")

	if err := os.WriteFile(p, []byte("old"), 0o444); err != nil {
		t.Fatalf("seed: %v", err)
	}
	if err := f.WriteFileAtomic(p, []byte("new")); err != nil {
		t.Fatalf("WriteFileAtomic over read-only: %v", err)
	}
	data, err := os.ReadFile(p)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(data) != "new" {
		t.Errorf("content = %q, want 'new'", data)
	}
}

func TestOS_WriteFileAtomic_noPartialFileOnInterruption(t *testing.T) {
	dir := t.TempDir()
	f := fs.NewOSFileSystem()
	p := filepath.Join(dir, "atomic.txt")

	if err := os.MkdirAll(p, 0o755); err != nil {
		t.Fatalf("seed final path as a pre-existing non-empty directory: %v", err)
	}
	sentinel := filepath.Join(p, "sentinel.txt")
	if err := os.WriteFile(sentinel, []byte("pre-existing"), 0o644); err != nil {
		t.Fatalf("seed sentinel file: %v", err)
	}

	writeErr := f.WriteFileAtomic(p, []byte("new-content-that-must-never-land-partially"))
	if writeErr == nil {
		t.Fatal("WriteFileAtomic must fail when Rename cannot replace a non-empty directory at the final path — Write/Sync/Close all succeed, only Rename fails")
	}

	info, err := os.Stat(p)
	if err != nil {
		t.Fatalf("Stat final path after failed attempt: %v", err)
	}
	if !info.IsDir() {
		t.Fatal("final path must remain the original directory after a failed Rename — never partially replaced with the new content")
	}
	data, err := os.ReadFile(sentinel)
	if err != nil {
		t.Fatalf("ReadFile sentinel after failed attempt: %v", err)
	}
	if string(data) != "pre-existing" {
		t.Errorf("pre-existing content at final path changed = %q, want 'pre-existing' (byte-identical to before the failed attempt)", data)
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	for _, e := range entries {
		if e.Name() != filepath.Base(p) {
			t.Errorf("temp file leaked in directory after failed WriteFileAtomic: %s", e.Name())
		}
	}
}

func TestOS_WriteFileAtomic_cleansUpTempOnWriteFailure(t *testing.T) {
	dir := t.TempDir()
	f := fs.NewOSFileSystem()
	unwritable := filepath.Join(dir, "unwritable")
	p := filepath.Join(unwritable, "file.txt")

	if err := os.MkdirAll(unwritable, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.Chmod(unwritable, 0o555); err != nil {
		t.Fatalf("chmod read-only: %v", err)
	}
	t.Cleanup(func() { makeWritableForCleanup(unwritable) })

	err := f.WriteFileAtomic(p, []byte("data"))
	if err == nil {
		t.Fatal("WriteFileAtomic into read-only directory should fail")
	}

	makeWritableForCleanup(unwritable)
	entries, readErr := os.ReadDir(unwritable)
	if readErr != nil {
		t.Fatalf("ReadDir: %v", readErr)
	}
	for _, e := range entries {
		t.Errorf("temp file leaked after failed WriteFileAtomic: %s", e.Name())
	}
}

func TestOS_WriteFileAtomicVsWriteFile_symlinkSemantics(t *testing.T) {
	scenarios := []struct {
		name       string
		write      func(f *fs.OSFileSystem, link string, data []byte) error
		wantTarget string
		wantLink   bool
	}{
		{
			name: "WriteFileAtomic replaces the symlink itself via Rename",
			write: func(f *fs.OSFileSystem, link string, data []byte) error {
				return f.WriteFileAtomic(link, data)
			},
			wantTarget: "atomic-written",
			wantLink:   false,
		},
		{
			name: "WriteFile follows the symlink and writes through it via os.WriteFile",
			write: func(f *fs.OSFileSystem, link string, data []byte) error {
				return f.WriteFile(link, data)
			},
			wantTarget: "followed-written",
			wantLink:   true,
		},
	}

	for _, sc := range scenarios {
		t.Run(sc.name, func(t *testing.T) {
			dir := t.TempDir()
			f := fs.NewOSFileSystem()
			target := filepath.Join(dir, "target.txt")
			link := filepath.Join(dir, "link.txt")

			if err := os.WriteFile(target, []byte("original"), 0o644); err != nil {
				t.Fatalf("seed target: %v", err)
			}
			if err := os.Symlink(target, link); err != nil {
				t.Fatalf("Symlink: %v", err)
			}

			var data []byte
			if sc.wantTarget == "atomic-written" {
				data = []byte("atomic-written")
			} else {
				data = []byte("followed-written")
			}
			if err := sc.write(f, link, data); err != nil {
				t.Fatalf("write: %v", err)
			}

			linkInfo, err := os.Lstat(link)
			if err != nil {
				t.Fatalf("Lstat link: %v", err)
			}
			isSymlink := linkInfo.Mode()&os.ModeSymlink != 0
			if isSymlink != sc.wantLink {
				t.Errorf("link is symlink after write = %v, want %v", isSymlink, sc.wantLink)
			}

			linkContent, err := os.ReadFile(link)
			if err != nil {
				t.Fatalf("ReadFile link: %v", err)
			}
			if string(linkContent) != string(data) {
				t.Errorf("link content = %q, want %q", linkContent, data)
			}

			targetContent, err := os.ReadFile(target)
			if err != nil {
				t.Fatalf("ReadFile target: %v", err)
			}
			if sc.wantLink {
				if string(targetContent) != string(data) {
					t.Errorf("WriteFile must follow the symlink and write through to target: target content = %q, want %q", targetContent, data)
				}
			} else {
				if string(targetContent) != "original" {
					t.Errorf("WriteFileAtomic must never write through the symlink to target: target content = %q, want 'original'", targetContent)
				}
			}
		})
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

func TestOS_RemoveAll_removesReadOnlyTree(t *testing.T) {
	dir := t.TempDir()
	f := fs.NewOSFileSystem()
	sub := filepath.Join(dir, "tree")
	nested := filepath.Join(sub, "nested")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(filepath.Join(nested, "skill.md"), []byte("old"), 0o444); err != nil {
		t.Fatalf("seed file: %v", err)
	}
	if err := os.Chmod(nested, 0o555); err != nil {
		t.Fatalf("chmod nested: %v", err)
	}
	if err := os.Chmod(sub, 0o555); err != nil {
		t.Fatalf("chmod sub: %v", err)
	}
	t.Cleanup(func() { makeWritableForCleanup(sub) })

	if err := f.RemoveAll(sub); err != nil {
		t.Fatalf("RemoveAll read-only tree: %v", err)
	}
	if f.Exists(sub) {
		t.Error("directory should not exist after RemoveAll")
	}
}

func TestOS_RemoveAll_doesNotFollowSymlinkOutsideTree(t *testing.T) {
	dir := t.TempDir()
	f := fs.NewOSFileSystem()
	tree := filepath.Join(dir, "tree")
	outside := filepath.Join(dir, "outside.txt")
	link := filepath.Join(tree, "outside-link")

	if err := os.MkdirAll(tree, 0o755); err != nil {
		t.Fatalf("MkdirAll tree: %v", err)
	}
	if err := os.WriteFile(outside, []byte("preservar"), 0o600); err != nil {
		t.Fatalf("seed outside: %v", err)
	}
	if err := os.Symlink(outside, link); err != nil {
		t.Fatalf("Symlink: %v", err)
	}
	if err := os.Chmod(tree, 0o555); err != nil {
		t.Fatalf("chmod tree: %v", err)
	}
	t.Cleanup(func() { makeWritableForCleanup(tree) })

	if err := f.RemoveAll(tree); err != nil {
		t.Fatalf("RemoveAll tree with symlink: %v", err)
	}
	content, err := os.ReadFile(outside)
	if err != nil {
		t.Fatalf("ReadFile outside: %v", err)
	}
	if string(content) != "preservar" {
		t.Errorf("outside content = %q, want preserved content", content)
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

func TestOS_CopyDir_AllowsUpgradeFromReadOnlySource(t *testing.T) {
	dir := t.TempDir()
	f := fs.NewOSFileSystem()
	src := filepath.Join(dir, "src")
	dst := filepath.Join(dir, "dst")
	if err := os.MkdirAll(filepath.Join(src, "skill"), 0o755); err != nil {
		t.Fatalf("MkdirAll src: %v", err)
	}
	skillFile := filepath.Join(src, "skill", "SKILL.md")
	if err := os.WriteFile(skillFile, []byte("v1"), 0o444); err != nil {
		t.Fatalf("seed source: %v", err)
	}
	if err := os.Chmod(filepath.Join(src, "skill"), 0o555); err != nil {
		t.Fatalf("chmod source skill: %v", err)
	}
	if err := os.Chmod(src, 0o555); err != nil {
		t.Fatalf("chmod source root: %v", err)
	}
	t.Cleanup(func() { makeWritableForCleanup(src) })

	if err := f.CopyDir(src, dst); err != nil {
		t.Fatalf("first CopyDir: %v", err)
	}
	if err := os.Chmod(src, 0o755); err != nil {
		t.Fatalf("chmod source root writable: %v", err)
	}
	if err := os.Chmod(filepath.Join(src, "skill"), 0o755); err != nil {
		t.Fatalf("chmod source skill writable: %v", err)
	}
	if err := os.Chmod(skillFile, 0o644); err != nil {
		t.Fatalf("chmod source file writable: %v", err)
	}
	if err := os.WriteFile(skillFile, []byte("v2"), 0o444); err != nil {
		t.Fatalf("update source: %v", err)
	}
	if err := os.Chmod(skillFile, 0o444); err != nil {
		t.Fatalf("chmod source file read-only: %v", err)
	}
	if err := os.Chmod(filepath.Join(src, "skill"), 0o555); err != nil {
		t.Fatalf("chmod source skill read-only: %v", err)
	}
	if err := os.Chmod(src, 0o555); err != nil {
		t.Fatalf("chmod source root read-only: %v", err)
	}
	if err := f.CopyDir(src, dst); err != nil {
		t.Fatalf("second CopyDir over read-only-derived dst: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dst, "skill", "SKILL.md"))
	if err != nil {
		t.Fatalf("ReadFile dst: %v", err)
	}
	if string(data) != "v2" {
		t.Errorf("CopyDir result = %q, want 'v2'", data)
	}
	info, err := os.Stat(filepath.Join(dst, "skill"))
	if err != nil {
		t.Fatalf("Stat dst dir: %v", err)
	}
	if info.Mode().Perm()&0o200 == 0 {
		t.Errorf("copied directory is not writable, mode = %v", info.Mode().Perm())
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

func makeWritableForCleanup(path string) {
	root, err := os.OpenRoot(path)
	if err != nil {
		return
	}
	defer func() { _ = root.Close() }()
	makeRootWritable(root, ".")
}

func makeRootWritable(root *os.Root, name string) {
	info, err := root.Lstat(name)
	if err != nil || info.Mode()&os.ModeSymlink != 0 {
		return
	}

	file, err := root.Open(name)
	if err != nil {
		return
	}
	defer func() { _ = file.Close() }()

	if !info.IsDir() {
		_ = file.Chmod(0o644)
		return
	}

	_ = file.Chmod(0o755)
	entries, err := file.ReadDir(-1)
	if err != nil {
		return
	}
	for _, entry := range entries {
		makeRootWritable(root, path.Join(name, entry.Name()))
	}
}
