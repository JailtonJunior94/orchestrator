package install

import (
	"path/filepath"
	"sort"
	"strings"

	"github.com/JailtonJunior94/ai-spec-harness/internal/fs"
	"github.com/JailtonJunior94/ai-spec-harness/internal/manifest"
)

type writeTracker struct {
	fs.FileSystem
	root    string
	created map[string]bool
	merged  map[string]bool
}

var _ fs.FileSystem = (*writeTracker)(nil)

func newWriteTracker(inner fs.FileSystem, root string) *writeTracker {
	return &writeTracker{
		FileSystem: inner,
		root:       filepath.Clean(root),
		created:    make(map[string]bool),
		merged:     make(map[string]bool),
	}
}

func (t *writeTracker) relativize(path string) (string, bool) {
	abs := path
	if !filepath.IsAbs(abs) {
		resolved, err := filepath.Abs(abs)
		if err != nil {
			return "", false
		}
		abs = resolved
	}
	rel, err := filepath.Rel(t.root, filepath.Clean(abs))
	if err != nil {
		return "", false
	}
	if rel == "." || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", false
	}
	return rel, true
}

func (t *writeTracker) record(path string) {
	rel, ok := t.relativize(path)
	if !ok || rel == manifest.ManifestFile {
		return
	}
	if t.merged[rel] {
		return
	}
	t.created[rel] = true
}

func (t *writeTracker) MarkInstalled(path string) {
	t.record(path)
}

func (t *writeTracker) MarkMerged(path string) {
	rel, ok := t.relativize(path)
	if !ok {
		return
	}
	delete(t.created, rel)
	t.merged[rel] = true
}

func (t *writeTracker) forget(path string) {
	rel, ok := t.relativize(path)
	if !ok {
		return
	}
	prefix := rel + string(filepath.Separator)
	for _, set := range []map[string]bool{t.created, t.merged} {
		for tracked := range set {
			if tracked == rel || strings.HasPrefix(tracked, prefix) {
				delete(set, tracked)
			}
		}
	}
}

func (t *writeTracker) recordCopiedTree(src, dst string) {
	entries, err := t.ReadDir(src)
	if err != nil {
		return
	}
	for _, entry := range entries {
		childSrc := filepath.Join(src, entry.Name())
		childDst := filepath.Join(dst, entry.Name())
		if entry.IsDir() {
			t.recordCopiedTree(childSrc, childDst)
			continue
		}
		t.record(childDst)
	}
}

func (t *writeTracker) WriteFile(path string, data []byte) error {
	if err := t.FileSystem.WriteFile(path, data); err != nil {
		return err
	}
	t.record(path)
	return nil
}

func (t *writeTracker) WriteFileAtomic(path string, data []byte) error {
	if err := t.FileSystem.WriteFileAtomic(path, data); err != nil {
		return err
	}
	t.record(path)
	return nil
}

func (t *writeTracker) CopyFile(src, dst string) error {
	if err := t.FileSystem.CopyFile(src, dst); err != nil {
		return err
	}
	t.record(dst)
	return nil
}

func (t *writeTracker) CopyDir(src, dst string) error {
	if err := t.FileSystem.CopyDir(src, dst); err != nil {
		return err
	}
	t.recordCopiedTree(src, dst)
	return nil
}

func (t *writeTracker) Symlink(target, link string) error {
	if err := t.FileSystem.Symlink(target, link); err != nil {
		return err
	}
	t.record(link)
	return nil
}

func (t *writeTracker) Remove(path string) error {
	err := t.FileSystem.Remove(path)
	if err == nil {
		t.forget(path)
	}
	return err
}

func (t *writeTracker) RemoveAll(path string) error {
	err := t.FileSystem.RemoveAll(path)
	if err == nil {
		t.forget(path)
	}
	return err
}

func (t *writeTracker) createdPaths() []string {
	return t.sortedKeys(t.created)
}

func (t *writeTracker) mergedPaths() []string {
	return t.sortedKeys(t.merged)
}

func (t *writeTracker) sortedKeys(set map[string]bool) []string {
	out := make([]string, 0, len(set))
	for key := range set {
		out = append(out, key)
	}
	sort.Strings(out)
	return out
}
