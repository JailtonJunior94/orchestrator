package fs

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// FakeFileSystem implementa FileSystem em memoria para testes.
type FakeFileSystem struct {
	Files   map[string][]byte
	Dirs    map[string]bool
	Links   map[string]string // link -> target
	NoWrite map[string]bool
}

func NewFakeFileSystem() *FakeFileSystem {
	return &FakeFileSystem{
		Files:   make(map[string][]byte),
		Dirs:    make(map[string]bool),
		Links:   make(map[string]string),
		NoWrite: make(map[string]bool),
	}
}

func (f *FakeFileSystem) MkdirAll(path string) error {
	f.Dirs[path] = true
	return nil
}

func (f *FakeFileSystem) CopyFile(src, dst string) error {
	data, ok := f.Files[src]
	if !ok {
		return fmt.Errorf("arquivo nao encontrado: %s", src)
	}
	f.Files[dst] = append([]byte(nil), data...)
	return nil
}

func (f *FakeFileSystem) CopyDir(src, dst string) error {
	f.Dirs[dst] = true
	for path, data := range f.Files {
		if strings.HasPrefix(path, src+"/") {
			rel, _ := filepath.Rel(src, path)
			newPath := filepath.Join(dst, rel)
			f.Files[newPath] = append([]byte(nil), data...)
		}
	}
	return nil
}

func (f *FakeFileSystem) Symlink(target, link string) error {
	f.Links[link] = target
	return nil
}

func (f *FakeFileSystem) Remove(path string) error {
	delete(f.Files, path)
	delete(f.Links, path)
	return nil
}

func (f *FakeFileSystem) RemoveAll(path string) error {
	delete(f.Dirs, path)
	for k := range f.Files {
		if k == path || strings.HasPrefix(k, path+"/") {
			delete(f.Files, k)
		}
	}
	for k := range f.Links {
		if k == path || strings.HasPrefix(k, path+"/") {
			delete(f.Links, k)
		}
	}
	return nil
}

func (f *FakeFileSystem) Exists(path string) bool {
	if _, ok := f.Files[path]; ok {
		return true
	}
	if _, ok := f.Dirs[path]; ok {
		return true
	}
	if _, ok := f.Links[path]; ok {
		return true
	}
	// Verificar se algum arquivo esta dentro desse path (implica diretorio)
	for k := range f.Files {
		if strings.HasPrefix(k, path+"/") {
			return true
		}
	}
	return false
}

func (f *FakeFileSystem) IsDir(path string) bool {
	if f.Dirs[path] {
		return true
	}
	for k := range f.Files {
		if strings.HasPrefix(k, path+"/") {
			return true
		}
	}
	return false
}

func (f *FakeFileSystem) IsSymlink(path string) bool {
	_, ok := f.Links[path]
	return ok
}

func (f *FakeFileSystem) ReadFile(path string) ([]byte, error) {
	data, ok := f.Files[path]
	if !ok {
		return nil, fmt.Errorf("arquivo nao encontrado: %s", path)
	}
	return data, nil
}

func (f *FakeFileSystem) WriteFile(path string, data []byte) error {
	f.Files[path] = append([]byte(nil), data...)
	return nil
}

func (f *FakeFileSystem) ReadDir(path string) ([]os.DirEntry, error) {
	seen := make(map[string]bool)
	var entries []os.DirEntry

	for k := range f.Files {
		if !strings.HasPrefix(k, path+"/") {
			continue
		}
		rest := strings.TrimPrefix(k, path+"/")
		parts := strings.SplitN(rest, "/", 2)
		name := parts[0]
		if seen[name] {
			continue
		}
		seen[name] = true
		isDir := len(parts) > 1
		entries = append(entries, &fakeDirEntry{name: name, dir: isDir})
	}

	for k := range f.Dirs {
		if !strings.HasPrefix(k, path+"/") {
			continue
		}
		rest := strings.TrimPrefix(k, path+"/")
		parts := strings.SplitN(rest, "/", 2)
		name := parts[0]
		if seen[name] {
			continue
		}
		seen[name] = true
		entries = append(entries, &fakeDirEntry{name: name, dir: true})
	}

	sort.Slice(entries, func(i, j int) bool { return entries[i].Name() < entries[j].Name() })
	return entries, nil
}

func (f *FakeFileSystem) FileHash(path string) (string, error) {
	data, ok := f.Files[path]
	if !ok {
		return "", fmt.Errorf("arquivo nao encontrado: %s", path)
	}
	h := sha256.Sum256(data)
	return fmt.Sprintf("%x", h[:]), nil
}

func (f *FakeFileSystem) DirHash(path string) (string, error) {
	if !f.IsDir(path) {
		return "", nil
	}
	var entries []string
	for k := range f.Files {
		if strings.HasPrefix(k, path+"/") {
			rel, _ := filepath.Rel(path, k)
			entries = append(entries, rel)
		}
	}
	sort.Strings(entries)
	h := sha256.New()
	for _, rel := range entries {
		fmt.Fprintf(h, "%s\n", rel)
		h.Write(f.Files[filepath.Join(path, rel)])
	}
	return fmt.Sprintf("%x", h.Sum(nil)), nil
}

func (f *FakeFileSystem) Writable(path string) bool {
	return !f.NoWrite[path]
}

type fakeDirEntry struct {
	name string
	dir  bool
}

func (e *fakeDirEntry) Name() string { return e.name }
func (e *fakeDirEntry) IsDir() bool  { return e.dir }
func (e *fakeDirEntry) Type() os.FileMode {
	if e.dir {
		return os.ModeDir
	}
	return 0
}
func (e *fakeDirEntry) Info() (os.FileInfo, error) { return nil, nil }
