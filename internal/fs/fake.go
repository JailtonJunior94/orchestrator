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
	resolved := f.resolvePath(path)
	f.Dirs[resolved] = true
	return nil
}

func (f *FakeFileSystem) CopyFile(src, dst string) error {
	resolvedSrc := f.resolvePath(src)
	data, ok := f.Files[resolvedSrc]
	if !ok {
		return fmt.Errorf("arquivo nao encontrado: %s", src)
	}
	f.Files[f.resolvePath(dst)] = append([]byte(nil), data...)
	return nil
}

func (f *FakeFileSystem) CopyDir(src, dst string) error {
	resolvedSrc := f.resolvePath(src)
	resolvedDst := f.resolvePath(dst)
	f.Dirs[resolvedDst] = true
	for path, data := range f.Files {
		if strings.HasPrefix(path, resolvedSrc+"/") {
			rel, _ := filepath.Rel(resolvedSrc, path)
			newPath := filepath.Join(resolvedDst, rel)
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
	resolved := filepath.Clean(path)
	if !f.IsSymlink(resolved) {
		resolved = f.resolvePath(resolved)
	}
	delete(f.Dirs, resolved)
	for k := range f.Files {
		if k == resolved || strings.HasPrefix(k, resolved+"/") {
			delete(f.Files, k)
		}
	}
	for k := range f.Links {
		if k == resolved || strings.HasPrefix(k, resolved+"/") {
			delete(f.Links, k)
		}
	}
	return nil
}

func (f *FakeFileSystem) Exists(path string) bool {
	if _, ok := f.Links[path]; ok {
		return true
	}
	resolved := f.resolvePath(path)
	if _, ok := f.Files[resolved]; ok {
		return true
	}
	if _, ok := f.Dirs[resolved]; ok {
		return true
	}
	// Verificar se algum arquivo esta dentro desse path (implica diretorio)
	for k := range f.Files {
		if strings.HasPrefix(k, resolved+"/") {
			return true
		}
	}
	return false
}

func (f *FakeFileSystem) IsDir(path string) bool {
	resolved := f.resolvePath(path)
	if f.Dirs[resolved] {
		return true
	}
	for k := range f.Files {
		if strings.HasPrefix(k, resolved+"/") {
			return true
		}
	}
	return false
}

func (f *FakeFileSystem) IsSymlink(path string) bool {
	_, ok := f.Links[path]
	return ok
}

func (f *FakeFileSystem) EvalSymlinks(path string) (string, error) {
	return f.resolvePath(path), nil
}

func (f *FakeFileSystem) ReadFile(path string) ([]byte, error) {
	resolved := f.resolvePath(path)
	data, ok := f.Files[resolved]
	if !ok {
		return nil, fmt.Errorf("arquivo nao encontrado: %s", path)
	}
	return data, nil
}

func (f *FakeFileSystem) WriteFile(path string, data []byte) error {
	f.Files[f.resolvePath(path)] = append([]byte(nil), data...)
	return nil
}

func (f *FakeFileSystem) ReadDir(path string) ([]os.DirEntry, error) {
	resolved := f.resolvePath(path)
	seen := make(map[string]bool)
	var entries []os.DirEntry

	for k := range f.Files {
		if !strings.HasPrefix(k, resolved+"/") {
			continue
		}
		rest := strings.TrimPrefix(k, resolved+"/")
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
		if !strings.HasPrefix(k, resolved+"/") {
			continue
		}
		rest := strings.TrimPrefix(k, resolved+"/")
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
	resolved := f.resolvePath(path)
	data, ok := f.Files[resolved]
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
	resolved := f.resolvePath(path)
	var entries []string
	for k := range f.Files {
		if strings.HasPrefix(k, resolved+"/") {
			rel, _ := filepath.Rel(resolved, k)
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
	return !f.NoWrite[f.resolvePath(path)]
}

func (f *FakeFileSystem) resolvePath(path string) string {
	current := filepath.Clean(path)
	seen := make(map[string]bool)
	for {
		if seen[current] {
			return current
		}
		seen[current] = true

		matched := ""
		matchedTarget := ""
		for link, target := range f.Links {
			cleanLink := filepath.Clean(link)
			if current == cleanLink || strings.HasPrefix(current, cleanLink+string(filepath.Separator)) {
				if len(cleanLink) > len(matched) {
					matched = cleanLink
					matchedTarget = target
				}
			}
		}
		if matched == "" {
			return current
		}

		if !filepath.IsAbs(matchedTarget) {
			matchedTarget = filepath.Join(filepath.Dir(matched), matchedTarget)
		}
		rest := strings.TrimPrefix(current, matched)
		current = filepath.Clean(matchedTarget + rest)
	}
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
