package fs

import (
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"sort"
)

// FileSystem abstrai operacoes de arquivo para testabilidade.
type FileSystem interface {
	MkdirAll(path string) error
	CopyFile(src, dst string) error
	CopyDir(src, dst string) error
	Symlink(target, link string) error
	Remove(path string) error
	RemoveAll(path string) error
	Exists(path string) bool
	IsDir(path string) bool
	IsSymlink(path string) bool
	EvalSymlinks(path string) (string, error)
	ReadFile(path string) ([]byte, error)
	WriteFile(path string, data []byte) error
	ReadDir(path string) ([]os.DirEntry, error)
	FileHash(path string) (string, error)
	DirHash(path string) (string, error)
	Writable(path string) bool
}

// OSFileSystem implementa FileSystem usando o sistema operacional real.
type OSFileSystem struct{}

var _ FileSystem = (*OSFileSystem)(nil)

func NewOSFileSystem() *OSFileSystem { return &OSFileSystem{} }

func (f *OSFileSystem) MkdirAll(path string) error {
	return os.MkdirAll(path, 0o755)
}

func (f *OSFileSystem) CopyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("abrir origem %s: %w", src, err)
	}
	defer in.Close()

	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}

	if info, err := os.Lstat(dst); err == nil && info.Mode().IsRegular() && info.Mode().Perm()&0o200 == 0 {
		_ = os.Chmod(dst, 0o644)
	}

	out, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("criar destino %s: %w", dst, err)
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		return err
	}

	info, err := os.Stat(src)
	if err != nil {
		return err
	}
	return os.Chmod(dst, f.writableFileMode(info.Mode()))
}

func (f *OSFileSystem) CopyDir(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)

		if info.IsDir() {
			mode := f.writableDirMode(info.Mode())
			if err := os.MkdirAll(target, mode); err != nil {
				return err
			}
			return os.Chmod(target, mode)
		}
		return f.CopyFile(path, target)
	})
}

func (f *OSFileSystem) Symlink(target, link string) error {
	if err := os.MkdirAll(filepath.Dir(link), 0o755); err != nil {
		return err
	}
	_ = os.Remove(link)
	return os.Symlink(target, link)
}

func (f *OSFileSystem) Remove(path string) error {
	return os.Remove(path)
}

func (f *OSFileSystem) RemoveAll(path string) error {
	if path == "" {
		return nil
	}

	root, err := os.OpenRoot(filepath.Dir(path))
	if err != nil {
		return os.RemoveAll(path)
	}
	defer func() { _ = root.Close() }()

	name := filepath.Base(path)
	info, err := root.Lstat(name)
	if err == nil && info.IsDir() {
		childRoot, err := root.OpenRoot(name)
		if err == nil {
			_ = f.chmodTreeWritable(childRoot, ".")
			_ = childRoot.Close()
		}
	}

	return root.RemoveAll(name)
}

func (f *OSFileSystem) Exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func (f *OSFileSystem) IsDir(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

func (f *OSFileSystem) IsSymlink(path string) bool {
	info, err := os.Lstat(path)
	return err == nil && info.Mode()&os.ModeSymlink != 0
}

func (f *OSFileSystem) EvalSymlinks(path string) (string, error) {
	resolved, err := filepath.EvalSymlinks(path)
	if err != nil {
		return "", fmt.Errorf("resolver symlinks de %s: %w", path, err)
	}
	return resolved, nil
}

func (f *OSFileSystem) ReadFile(path string) ([]byte, error) {
	return os.ReadFile(path)
}

func (f *OSFileSystem) WriteFile(path string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	if info, err := os.Lstat(path); err == nil && info.Mode().IsRegular() && info.Mode().Perm()&0o200 == 0 {
		_ = os.Chmod(path, 0o644)
	}
	return os.WriteFile(path, data, 0o644)
}

func (f *OSFileSystem) ReadDir(path string) ([]os.DirEntry, error) {
	return os.ReadDir(path)
}

func (f *OSFileSystem) FileHash(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	h := sha256.New()
	if _, err := io.Copy(h, file); err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", h.Sum(nil)), nil
}

func (f *OSFileSystem) DirHash(path string) (string, error) {
	if !f.IsDir(path) {
		return "", nil
	}

	var entries []string
	err := filepath.Walk(path, func(p string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return err
		}
		rel, _ := filepath.Rel(path, p)
		entries = append(entries, rel)
		return nil
	})
	if err != nil {
		return "", err
	}

	sort.Strings(entries)
	h := sha256.New()
	for _, rel := range entries {
		full := filepath.Join(path, rel)
		fmt.Fprintf(h, "%s\n", rel)
		data, err := os.ReadFile(full)
		if err != nil {
			return "", err
		}
		h.Write(data)
	}
	return fmt.Sprintf("%x", h.Sum(nil)), nil
}

func (f *OSFileSystem) Writable(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.Mode().Perm()&0o200 != 0
}

func (f *OSFileSystem) chmodTreeWritable(root *os.Root, name string) error {
	info, err := root.Lstat(name)
	if err != nil || info.Mode()&os.ModeSymlink != 0 {
		return err
	}

	file, err := root.Open(name)
	if err != nil {
		return err
	}
	defer func() { _ = file.Close() }()

	if !info.IsDir() {
		if info.Mode().IsRegular() {
			return file.Chmod(f.writableFileMode(info.Mode()))
		}
		return nil
	}

	if err := file.Chmod(f.writableDirMode(info.Mode())); err != nil {
		return err
	}
	entries, err := file.ReadDir(-1)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if err := f.chmodTreeWritable(root, path.Join(name, entry.Name())); err != nil {
			return err
		}
	}
	return nil
}

func (f *OSFileSystem) writableDirMode(mode os.FileMode) os.FileMode {
	return mode.Perm() | 0o700
}

func (f *OSFileSystem) writableFileMode(mode os.FileMode) os.FileMode {
	return mode.Perm() | 0o600
}
