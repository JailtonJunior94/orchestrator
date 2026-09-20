package txn

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/JailtonJunior94/ai-spec-harness/internal/fs"
)

type Conflict struct {
	Path     string
	Expected string
	Actual   string
}

type Transaction interface {
	Stage(path string, content []byte) error
	Conflicts() []Conflict
	Commit() error
	Rollback() error
}

type FileState int

const (
	StateCreated FileState = iota
	StateUpdated
	StatePreserved
)

func (s FileState) String() string {
	switch s {
	case StateCreated:
		return "created"
	case StateUpdated:
		return "updated"
	case StatePreserved:
		return "preserved"
	default:
		return "unknown"
	}
}

var ErrConflicts = errors.New("staged batch has managed files with unresolved conflicts")

type journalEntry struct {
	path     string
	existed  bool
	previous []byte
}

type FileTransaction struct {
	fs       fs.FileSystem
	root     string
	expected map[string]string

	stagedContent map[string][]byte
	stagedSource  map[string]string
	stagedOrder   []string
	removed       map[string]bool

	conflicts       []Conflict
	overwriteForced bool

	journal   []journalEntry
	outcomes  map[string]FileState
	committed bool
}

var _ Transaction = (*FileTransaction)(nil)
var _ fs.FileSystem = (*FileTransaction)(nil)

func New(fsys fs.FileSystem, root string, expectedChecksums map[string]string) *FileTransaction {
	expected := make(map[string]string, len(expectedChecksums))
	for k, v := range expectedChecksums {
		expected[k] = v
	}
	return &FileTransaction{
		fs:            fsys,
		root:          filepath.Clean(root),
		expected:      expected,
		stagedContent: make(map[string][]byte),
		stagedSource:  make(map[string]string),
		removed:       make(map[string]bool),
		outcomes:      make(map[string]FileState),
	}
}

func canonicalPath(path string) string {
	abs := path
	if !filepath.IsAbs(abs) {
		if resolved, err := filepath.Abs(abs); err == nil {
			abs = resolved
		}
	}
	return filepath.Clean(abs)
}

func (t *FileTransaction) relativeToRoot(path string) (string, bool) {
	abs := canonicalPath(path)
	rel, err := filepath.Rel(t.root, abs)
	if err != nil {
		return "", false
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", false
	}
	return rel, true
}

func (t *FileTransaction) AllowOverwrite() {
	t.overwriteForced = true
}

func (t *FileTransaction) Stage(path string, content []byte) error {
	return t.stage(path, content)
}

func (t *FileTransaction) stage(path string, content []byte) error {
	delete(t.stagedSource, canonicalPath(path))
	return t.stageInternal(path, content)
}

func (t *FileTransaction) stageFromSource(path, source string, content []byte) error {
	if err := t.stageInternal(path, content); err != nil {
		return err
	}
	t.stagedSource[canonicalPath(path)] = source
	return nil
}

func (t *FileTransaction) stageInternal(path string, content []byte) error {
	key := canonicalPath(path)
	t.checkConflict(key, path)
	if _, exists := t.stagedContent[key]; !exists {
		t.stagedOrder = append(t.stagedOrder, key)
	}
	t.stagedContent[key] = append([]byte(nil), content...)
	delete(t.removed, key)
	return nil
}

func (t *FileTransaction) checkConflict(key, originalPath string) {
	rel, managed := t.relativeToRoot(originalPath)
	if !managed {
		return
	}
	expected, isManaged := t.expected[rel]
	if !isManaged {
		return
	}
	if _, alreadyStaged := t.stagedContent[key]; alreadyStaged {
		return
	}
	if t.removed[key] {
		return
	}
	for _, existing := range t.conflicts {
		if existing.Path == rel {
			return
		}
	}
	if !t.fs.Exists(originalPath) {
		return
	}
	actual, err := t.fs.FileHash(originalPath)
	if err != nil {
		t.conflicts = append(t.conflicts, Conflict{Path: rel, Expected: expected, Actual: "unreadable: " + err.Error()})
		return
	}
	if actual != expected {
		t.conflicts = append(t.conflicts, Conflict{Path: rel, Expected: expected, Actual: actual})
	}
}

func (t *FileTransaction) Conflicts() []Conflict {
	out := make([]Conflict, len(t.conflicts))
	copy(out, t.conflicts)
	sort.Slice(out, func(i, j int) bool { return out[i].Path < out[j].Path })
	return out
}

var ErrAlreadyCommitted = errors.New("transaction already committed")

func (t *FileTransaction) Commit() error {
	if t.committed {
		return ErrAlreadyCommitted
	}
	if len(t.conflicts) > 0 && !t.overwriteForced {
		return fmt.Errorf("%w: %d file(s), abort-on-conflict default", ErrConflicts, len(t.conflicts))
	}

	for _, key := range t.stagedOrder {
		content := t.stagedContent[key]
		existed := t.fs.Exists(key)
		var previous []byte
		if existed {
			data, err := t.fs.ReadFile(key)
			if err != nil {
				rollbackErr := t.rollback()
				return t.combineCommitError(key, fmt.Errorf("snapshot before write: %w", err), rollbackErr)
			}
			previous = data
		}

		t.journal = append(t.journal, journalEntry{path: key, existed: existed, previous: previous})

		writeErr := t.writeCommitted(key, content)
		if writeErr != nil {
			rollbackErr := t.rollback()
			return t.combineCommitError(key, writeErr, rollbackErr)
		}

		if !existed {
			t.outcomes[key] = StateCreated
		} else if previous != nil && string(previous) == string(content) {
			t.outcomes[key] = StatePreserved
		} else {
			t.outcomes[key] = StateUpdated
		}
	}

	for _, key := range t.tombstonedKeys() {
		if !t.fs.Exists(key) {
			continue
		}
		previous, err := t.fs.ReadFile(key)
		if err != nil {
			rollbackErr := t.rollback()
			return t.combineCommitError(key, fmt.Errorf("snapshot before removal: %w", err), rollbackErr)
		}
		t.journal = append(t.journal, journalEntry{path: key, existed: true, previous: previous})
		if err := t.fs.Remove(key); err != nil && t.fs.Exists(key) {
			rollbackErr := t.rollback()
			return t.combineCommitError(key, err, rollbackErr)
		}
	}

	t.committed = true
	return nil
}

func (t *FileTransaction) writeCommitted(key string, content []byte) error {
	if source, ok := t.stagedSource[key]; ok && t.fs.Exists(source) {
		if err := t.fs.CopyFile(source, key); err == nil {
			return nil
		}
	}
	return t.fs.WriteFileAtomic(key, content)
}

func (t *FileTransaction) tombstonedKeys() []string {
	out := make([]string, 0, len(t.removed))
	for key, removed := range t.removed {
		if !removed {
			continue
		}
		if _, staged := t.stagedContent[key]; staged {
			continue
		}
		out = append(out, key)
	}
	sort.Strings(out)
	return out
}

func (t *FileTransaction) combineCommitError(failedPath string, commitErr, rollbackErr error) error {
	if rollbackErr != nil {
		return fmt.Errorf("commit failed at %s: %w; rollback incomplete: %v", failedPath, commitErr, rollbackErr)
	}
	return fmt.Errorf("commit failed at %s, batch fully reverted to prior state: %w", failedPath, commitErr)
}

func (t *FileTransaction) Rollback() error {
	return t.rollback()
}

func (t *FileTransaction) rollback() error {
	var notReverted []string
	for i := len(t.journal) - 1; i >= 0; i-- {
		entry := t.journal[i]
		var err error
		if entry.existed {
			err = t.fs.WriteFileAtomic(entry.path, entry.previous)
		} else {
			err = t.fs.Remove(entry.path)
			if err != nil && !t.fs.Exists(entry.path) {
				err = nil
			}
		}
		if err != nil {
			notReverted = append(notReverted, entry.path)
		}
	}
	t.journal = nil
	if len(notReverted) > 0 {
		sort.Strings(notReverted)
		return fmt.Errorf("rollback incomplete, not reverted: %s", strings.Join(notReverted, ", "))
	}
	return nil
}

func (t *FileTransaction) Outcomes() map[string]FileState {
	out := make(map[string]FileState, len(t.outcomes))
	for k, v := range t.outcomes {
		out[k] = v
	}
	return out
}

func (t *FileTransaction) MkdirAll(path string) error {
	return t.fs.MkdirAll(path)
}

func (t *FileTransaction) CopyFile(src, dst string) error {
	data, err := t.ReadFile(src)
	if err != nil {
		return fmt.Errorf("read source %s: %w", src, err)
	}
	return t.stageFromSource(dst, src, data)
}

func (t *FileTransaction) CopyDir(src, dst string) error {
	entries, err := t.fs.ReadDir(src)
	if err != nil {
		return fmt.Errorf("read source dir %s: %w", src, err)
	}
	for _, entry := range entries {
		childSrc := filepath.Join(src, entry.Name())
		childDst := filepath.Join(dst, entry.Name())
		if entry.IsDir() {
			if err := t.CopyDir(childSrc, childDst); err != nil {
				return err
			}
			continue
		}
		data, err := t.ReadFile(childSrc)
		if err != nil {
			return fmt.Errorf("read %s: %w", childSrc, err)
		}
		if err := t.stageFromSource(childDst, childSrc, data); err != nil {
			return err
		}
	}
	return nil
}

func (t *FileTransaction) Symlink(target, link string) error {
	return t.fs.Symlink(target, link)
}

func (t *FileTransaction) Remove(path string) error {
	key := canonicalPath(path)
	delete(t.stagedContent, key)
	delete(t.stagedSource, key)
	t.removed[key] = true
	return nil
}

func (t *FileTransaction) RemoveAll(path string) error {
	prefix := canonicalPath(path)
	dirPrefix := prefix + string(filepath.Separator)
	for key := range t.stagedContent {
		if key == prefix || strings.HasPrefix(key, dirPrefix) {
			delete(t.stagedContent, key)
			delete(t.stagedSource, key)
			t.removed[key] = true
		}
	}

	if !t.fs.Exists(path) {
		return nil
	}
	if !t.fs.IsDir(path) {
		t.checkConflict(prefix, path)
		t.removed[prefix] = true
		return nil
	}
	entries, err := t.fs.ReadDir(path)
	if err != nil {
		return fmt.Errorf("read directory %s for staged removal: %w", path, err)
	}
	for _, entry := range entries {
		child := filepath.Join(path, entry.Name())
		if err := t.RemoveAll(child); err != nil {
			return err
		}
	}
	return nil
}

func (t *FileTransaction) Exists(path string) bool {
	key := canonicalPath(path)
	if t.removed[key] {
		return false
	}
	if _, staged := t.stagedContent[key]; staged {
		return true
	}
	return t.fs.Exists(path)
}

func (t *FileTransaction) IsDir(path string) bool {
	key := canonicalPath(path)
	if t.removed[key] {
		return false
	}
	if _, staged := t.stagedContent[key]; staged {
		return false
	}
	return t.fs.IsDir(path)
}

func (t *FileTransaction) IsSymlink(path string) bool {
	return t.fs.IsSymlink(path)
}

func (t *FileTransaction) EvalSymlinks(path string) (string, error) {
	return t.fs.EvalSymlinks(path)
}

func (t *FileTransaction) ReadFile(path string) ([]byte, error) {
	key := canonicalPath(path)
	if t.removed[key] {
		return nil, fmt.Errorf("%s: %w", path, os.ErrNotExist)
	}
	if data, staged := t.stagedContent[key]; staged {
		return append([]byte(nil), data...), nil
	}
	return t.fs.ReadFile(path)
}

func (t *FileTransaction) WriteFile(path string, data []byte) error {
	return t.stage(path, data)
}

func (t *FileTransaction) WriteFileAtomic(path string, data []byte) error {
	return t.stage(path, data)
}

func (t *FileTransaction) AppendFile(path string, data []byte) error {
	existing, err := t.ReadFile(path)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("read %s before append: %w", path, err)
	}
	combined := make([]byte, 0, len(existing)+len(data))
	combined = append(combined, existing...)
	combined = append(combined, data...)
	return t.stage(path, combined)
}

func (t *FileTransaction) ReadDir(path string) ([]os.DirEntry, error) {
	return t.fs.ReadDir(path)
}

func (t *FileTransaction) FileHash(path string) (string, error) {
	key := canonicalPath(path)
	if t.removed[key] {
		return "", fmt.Errorf("%s: %w", path, os.ErrNotExist)
	}
	if data, staged := t.stagedContent[key]; staged {
		sum := sha256.Sum256(data)
		return fmt.Sprintf("%x", sum[:]), nil
	}
	return t.fs.FileHash(path)
}

func (t *FileTransaction) DirHash(path string) (string, error) {
	return t.fs.DirHash(path)
}

func (t *FileTransaction) Writable(path string) bool {
	return t.fs.Writable(path)
}
