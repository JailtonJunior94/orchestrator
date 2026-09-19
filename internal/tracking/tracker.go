package tracking

import (
	"fmt"
	"maps"
	"path/filepath"
	"sort"
	"strings"

	"github.com/JailtonJunior94/ai-spec-harness/internal/fs"
	"github.com/JailtonJunior94/ai-spec-harness/internal/txn"
)

type Tracker struct {
	fs.FileSystem
	txn        *txn.FileTransaction
	root       string
	exclude    string
	created    map[string]bool
	merged     map[string]bool
	checksums  map[string]string
	executable []string
}

var _ fs.FileSystem = (*Tracker)(nil)

func New(inner fs.FileSystem, root, exclude string) *Tracker {
	return NewTransactional(inner, root, exclude, nil)
}

func NewTransactional(inner fs.FileSystem, root, exclude string, expectedChecksums map[string]string) *Tracker {
	tx := txn.New(inner, root, expectedChecksums)
	return &Tracker{
		FileSystem: tx,
		txn:        tx,
		root:       filepath.Clean(root),
		exclude:    exclude,
		created:    make(map[string]bool),
		merged:     make(map[string]bool),
		checksums:  make(map[string]string),
	}
}

func (t *Tracker) AllowOverwrite() {
	t.txn.AllowOverwrite()
}

func (t *Tracker) Conflicts() []txn.Conflict {
	return t.txn.Conflicts()
}

func (t *Tracker) Commit() error {
	return t.txn.Commit()
}

func (t *Tracker) Rollback() error {
	return t.txn.Rollback()
}

func (t *Tracker) MarkExecutable(path string) {
	t.executable = append(t.executable, path)
}

func (t *Tracker) ExecutablePaths() []string {
	return append([]string(nil), t.executable...)
}

func (t *Tracker) Outcomes() map[string]txn.FileState {
	raw := t.txn.Outcomes()
	out := make(map[string]txn.FileState, len(raw))
	for path, state := range raw {
		rel, ok := t.relativize(path)
		if !ok || rel == t.exclude {
			continue
		}
		out[rel] = state
	}
	return out
}

func (t *Tracker) relativize(path string) (string, bool) {
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

func (t *Tracker) storeChecksum(path, rel string) error {
	hash, err := t.FileHash(path)
	if err != nil {
		return fmt.Errorf("compute checksum for tracked path %s: %w", rel, err)
	}
	t.checksums[rel] = hash
	return nil
}

func (t *Tracker) record(path string) error {
	rel, ok := t.relativize(path)
	if !ok || rel == t.exclude {
		return nil
	}
	if err := t.storeChecksum(path, rel); err != nil {
		return err
	}
	if !t.merged[rel] {
		t.created[rel] = true
	}
	return nil
}

func (t *Tracker) recordLink(path string) error {
	rel, ok := t.relativize(path)
	if !ok || rel == t.exclude {
		return nil
	}
	if !t.IsDir(path) {
		if err := t.storeChecksum(path, rel); err != nil {
			return err
		}
	}
	if !t.merged[rel] {
		t.created[rel] = true
	}
	return nil
}

func (t *Tracker) MarkInstalled(path string) error {
	return t.record(path)
}

func (t *Tracker) MarkMerged(path string) error {
	rel, ok := t.relativize(path)
	if !ok || rel == t.exclude {
		return nil
	}
	if err := t.storeChecksum(path, rel); err != nil {
		return err
	}
	delete(t.created, rel)
	t.merged[rel] = true
	return nil
}

func (t *Tracker) forget(path string) {
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
	for tracked := range t.checksums {
		if tracked == rel || strings.HasPrefix(tracked, prefix) {
			delete(t.checksums, tracked)
		}
	}
}

func (t *Tracker) recordCopiedTree(src, dst string) error {
	entries, err := t.ReadDir(src)
	if err != nil {
		return fmt.Errorf("read copied tree %s: %w", src, err)
	}
	for _, entry := range entries {
		childSrc := filepath.Join(src, entry.Name())
		childDst := filepath.Join(dst, entry.Name())
		if entry.IsDir() {
			if err := t.recordCopiedTree(childSrc, childDst); err != nil {
				return err
			}
			continue
		}
		if err := t.record(childDst); err != nil {
			return err
		}
	}
	return nil
}

func (t *Tracker) WriteFile(path string, data []byte) error {
	if err := t.FileSystem.WriteFile(path, data); err != nil {
		return err
	}
	return t.record(path)
}

func (t *Tracker) WriteFileAtomic(path string, data []byte) error {
	if err := t.FileSystem.WriteFileAtomic(path, data); err != nil {
		return err
	}
	return t.record(path)
}

func (t *Tracker) CopyFile(src, dst string) error {
	if err := t.FileSystem.CopyFile(src, dst); err != nil {
		return err
	}
	return t.record(dst)
}

func (t *Tracker) CopyDir(src, dst string) error {
	if err := t.FileSystem.CopyDir(src, dst); err != nil {
		return err
	}
	return t.recordCopiedTree(src, dst)
}

func (t *Tracker) Symlink(target, link string) error {
	if err := t.FileSystem.Symlink(target, link); err != nil {
		return err
	}
	return t.recordLink(link)
}

func (t *Tracker) Remove(path string) error {
	err := t.FileSystem.Remove(path)
	if err == nil {
		t.forget(path)
	}
	return err
}

func (t *Tracker) RemoveAll(path string) error {
	err := t.FileSystem.RemoveAll(path)
	if err == nil {
		t.forget(path)
	}
	return err
}

func (t *Tracker) CreatedPaths() []string {
	return t.sortedKeys(t.created)
}

func (t *Tracker) MergedPaths() []string {
	return t.sortedKeys(t.merged)
}

func (t *Tracker) Checksums() map[string]string {
	out := make(map[string]string, len(t.checksums))
	for k, v := range t.checksums {
		out[k] = v
	}
	return out
}

func BackfillChecksums(fsys fs.FileSystem, root string, managedPaths []string, existing map[string]string) map[string]string {
	out := make(map[string]string, len(existing)+len(managedPaths))
	maps.Copy(out, existing)
	for _, rel := range managedPaths {
		if _, ok := out[rel]; ok {
			continue
		}
		abs := filepath.Join(root, rel)
		if !fsys.Exists(abs) {
			continue
		}
		hash, err := fsys.FileHash(abs)
		if err != nil {
			continue
		}
		out[rel] = hash
	}
	return out
}

func WalkManagedPaths(fsys fs.FileSystem, root, dir string) []string {
	var out []string
	var walk func(rel string)
	walk = func(rel string) {
		abs := filepath.Join(root, rel)
		entries, err := fsys.ReadDir(abs)
		if err != nil {
			return
		}
		for _, entry := range entries {
			childRel := filepath.Join(rel, entry.Name())
			childAbs := filepath.Join(root, childRel)
			if fsys.IsDir(childAbs) {
				walk(childRel)
				continue
			}
			out = append(out, childRel)
		}
	}
	walk(dir)
	sort.Strings(out)
	return out
}

func (t *Tracker) sortedKeys(set map[string]bool) []string {
	out := make([]string, 0, len(set))
	for key := range set {
		out = append(out, key)
	}
	sort.Strings(out)
	return out
}
