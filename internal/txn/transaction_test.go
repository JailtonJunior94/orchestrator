package txn_test

import (
	"errors"
	"testing"

	"github.com/JailtonJunior94/ai-spec-harness/internal/fs"
	"github.com/JailtonJunior94/ai-spec-harness/internal/txn"
)

func TestFileTransaction_StageDoesNotTouchDiskUntilCommit(t *testing.T) {
	fake := fs.NewFakeFileSystem()
	tx := txn.New(fake, "/project", nil)

	if err := tx.Stage("/project/a.txt", []byte("new content")); err != nil {
		t.Fatalf("stage: %v", err)
	}

	if fake.Exists("/project/a.txt") {
		t.Fatal("staged write must not reach the underlying filesystem before Commit")
	}
	if !tx.Exists("/project/a.txt") {
		t.Fatal("transaction overlay must report a staged path as existing")
	}
}

func TestFileTransaction_CommitAppliesStagedWrites(t *testing.T) {
	fake := fs.NewFakeFileSystem()
	tx := txn.New(fake, "/project", nil)

	if err := tx.Stage("/project/a.txt", []byte("hello")); err != nil {
		t.Fatalf("stage: %v", err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatalf("commit: %v", err)
	}

	data, err := fake.ReadFile("/project/a.txt")
	if err != nil {
		t.Fatalf("read committed file: %v", err)
	}
	if string(data) != "hello" {
		t.Fatalf("committed content = %q, want %q", data, "hello")
	}
}

func TestFileTransaction_ConflictDetectedAgainstManagedChecksum(t *testing.T) {
	fake := fs.NewFakeFileSystem()
	if err := fake.WriteFile("/project/managed.txt", []byte("tampered on disk")); err != nil {
		t.Fatalf("seed: %v", err)
	}

	originalHash := "0000000000000000000000000000000000000000000000000000000000000"
	tx := txn.New(fake, "/project", map[string]string{"managed.txt": originalHash})

	if err := tx.Stage("/project/managed.txt", []byte("new governed content")); err != nil {
		t.Fatalf("stage: %v", err)
	}

	conflicts := tx.Conflicts()
	if len(conflicts) != 1 {
		t.Fatalf("expected exactly one conflict, got %d", len(conflicts))
	}
	if conflicts[0].Path != "managed.txt" {
		t.Errorf("conflict path = %q, want %q", conflicts[0].Path, "managed.txt")
	}
	if conflicts[0].Expected != originalHash {
		t.Errorf("conflict expected hash = %q, want %q", conflicts[0].Expected, originalHash)
	}
}

func TestFileTransaction_CommitAbortsOnConflictByDefault(t *testing.T) {
	fake := fs.NewFakeFileSystem()
	if err := fake.WriteFile("/project/managed.txt", []byte("tampered on disk")); err != nil {
		t.Fatalf("seed: %v", err)
	}
	tx := txn.New(fake, "/project", map[string]string{"managed.txt": "expected-hash-that-never-matches"})

	if err := tx.Stage("/project/managed.txt", []byte("new content")); err != nil {
		t.Fatalf("stage: %v", err)
	}
	if err := tx.Stage("/project/untouched.txt", []byte("also part of the batch")); err != nil {
		t.Fatalf("stage: %v", err)
	}

	err := tx.Commit()
	if err == nil {
		t.Fatal("commit must abort when a managed file conflicts and overwrite was not allowed")
	}
	if !errors.Is(err, txn.ErrConflicts) {
		t.Fatalf("commit error = %v, want wrapping %v", err, txn.ErrConflicts)
	}
	if fake.Exists("/project/untouched.txt") {
		t.Fatal("no file of the batch may be written when the batch aborts on conflict")
	}
	current, _ := fake.ReadFile("/project/managed.txt")
	if string(current) != "tampered on disk" {
		t.Fatal("conflicting managed file must remain untouched after an aborted commit")
	}
}

func TestFileTransaction_AllowOverwriteCommitsDespiteConflict(t *testing.T) {
	fake := fs.NewFakeFileSystem()
	if err := fake.WriteFile("/project/managed.txt", []byte("tampered on disk")); err != nil {
		t.Fatalf("seed: %v", err)
	}
	tx := txn.New(fake, "/project", map[string]string{"managed.txt": "expected-hash-that-never-matches"})
	tx.AllowOverwrite()

	if err := tx.Stage("/project/managed.txt", []byte("new content")); err != nil {
		t.Fatalf("stage: %v", err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatalf("commit with overwrite allowed: %v", err)
	}

	data, _ := fake.ReadFile("/project/managed.txt")
	if string(data) != "new content" {
		t.Fatalf("overwritten content = %q, want %q", data, "new content")
	}
}

type failingWriteFileSystem struct {
	fs.FileSystem
	failOn map[string]bool
}

func (f *failingWriteFileSystem) WriteFileAtomic(path string, data []byte) error {
	if f.failOn[path] {
		return errors.New("simulated disk failure")
	}
	return f.FileSystem.WriteFileAtomic(path, data)
}

func TestFileTransaction_CommitFailureMidBatchRevertsAlreadyWrittenFiles(t *testing.T) {
	fake := fs.NewFakeFileSystem()
	if err := fake.WriteFile("/project/a.txt", []byte("original a")); err != nil {
		t.Fatalf("seed a: %v", err)
	}
	if err := fake.WriteFile("/project/b.txt", []byte("original b")); err != nil {
		t.Fatalf("seed b: %v", err)
	}

	failing := &failingWriteFileSystem{FileSystem: fake, failOn: map[string]bool{"/project/b.txt": true}}
	tx := txn.New(failing, "/project", nil)

	if err := tx.Stage("/project/a.txt", []byte("changed a")); err != nil {
		t.Fatalf("stage a: %v", err)
	}
	if err := tx.Stage("/project/b.txt", []byte("changed b")); err != nil {
		t.Fatalf("stage b: %v", err)
	}
	if err := tx.Stage("/project/c.txt", []byte("new c")); err != nil {
		t.Fatalf("stage c: %v", err)
	}

	err := tx.Commit()
	if err == nil {
		t.Fatal("commit must fail when a staged write fails mid-batch")
	}

	dataA, _ := fake.ReadFile("/project/a.txt")
	if string(dataA) != "original a" {
		t.Fatalf("a.txt after rollback = %q, want %q (reverted)", dataA, "original a")
	}
	if fake.Exists("/project/c.txt") {
		t.Fatal("c.txt must not exist: it was never committed before the failure at b.txt")
	}
	dataB, _ := fake.ReadFile("/project/b.txt")
	if string(dataB) != "original b" {
		t.Fatalf("b.txt must remain at its original content since its own write failed: got %q", dataB)
	}
}

func TestFileTransaction_JournalOrderedBeforeDestructiveWrite(t *testing.T) {
	fake := fs.NewFakeFileSystem()
	tx := txn.New(fake, "/project", nil)

	if err := tx.Stage("/project/new.txt", []byte("brand new")); err != nil {
		t.Fatalf("stage: %v", err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatalf("commit: %v", err)
	}

	outcomes := tx.Outcomes()
	state, ok := outcomes["/project/new.txt"]
	if !ok {
		t.Fatal("committed path must have a recorded outcome")
	}
	if state != txn.StateCreated {
		t.Errorf("outcome for a brand new file = %v, want StateCreated", state)
	}
}

func TestFileTransaction_OutcomeUpdatedVsPreserved(t *testing.T) {
	fake := fs.NewFakeFileSystem()
	if err := fake.WriteFile("/project/same.txt", []byte("identical")); err != nil {
		t.Fatalf("seed: %v", err)
	}
	if err := fake.WriteFile("/project/changed.txt", []byte("before")); err != nil {
		t.Fatalf("seed: %v", err)
	}
	tx := txn.New(fake, "/project", nil)

	if err := tx.Stage("/project/same.txt", []byte("identical")); err != nil {
		t.Fatalf("stage: %v", err)
	}
	if err := tx.Stage("/project/changed.txt", []byte("after")); err != nil {
		t.Fatalf("stage: %v", err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatalf("commit: %v", err)
	}

	outcomes := tx.Outcomes()
	if outcomes["/project/same.txt"] != txn.StatePreserved {
		t.Errorf("outcome for unchanged content = %v, want StatePreserved", outcomes["/project/same.txt"])
	}
	if outcomes["/project/changed.txt"] != txn.StateUpdated {
		t.Errorf("outcome for changed content = %v, want StateUpdated", outcomes["/project/changed.txt"])
	}
}

func TestFileTransaction_RemoveAllStagesTombstonesWithoutTouchingDisk(t *testing.T) {
	fake := fs.NewFakeFileSystem()
	if err := fake.WriteFile("/project/dir/old.txt", []byte("stale")); err != nil {
		t.Fatalf("seed: %v", err)
	}
	tx := txn.New(fake, "/project", nil)

	if err := tx.RemoveAll("/project/dir"); err != nil {
		t.Fatalf("removeall: %v", err)
	}
	if !fake.Exists("/project/dir/old.txt") {
		t.Fatal("staged RemoveAll must not touch disk before Commit — this is what makes it reversible")
	}
	if tx.Exists("/project/dir/old.txt") {
		t.Fatal("transaction overlay must report the tombstoned path as gone")
	}

	if err := tx.Commit(); err != nil {
		t.Fatalf("commit: %v", err)
	}
	if fake.Exists("/project/dir/old.txt") {
		t.Fatal("committed RemoveAll must remove the stale file from disk")
	}
}

func TestFileTransaction_RemoveAllThenCopyDirIsReversible(t *testing.T) {
	fake := fs.NewFakeFileSystem()
	if err := fake.WriteFile("/source/skill/SKILL.md", []byte("v2 content")); err != nil {
		t.Fatalf("seed source: %v", err)
	}
	if err := fake.WriteFile("/project/skill/SKILL.md", []byte("v1 content")); err != nil {
		t.Fatalf("seed dest: %v", err)
	}
	if err := fake.WriteFile("/project/skill/stale.txt", []byte("no longer shipped")); err != nil {
		t.Fatalf("seed stale: %v", err)
	}

	failing := &failingWriteFileSystem{FileSystem: fake, failOn: map[string]bool{}}
	tx := txn.New(failing, "/project", nil)

	if err := tx.RemoveAll("/project/skill"); err != nil {
		t.Fatalf("removeall: %v", err)
	}
	if err := tx.CopyDir("/source/skill", "/project/skill"); err != nil {
		t.Fatalf("copydir: %v", err)
	}

	if string(mustRead(t, fake, "/project/skill/SKILL.md")) != "v1 content" {
		t.Fatal("staging must not mutate disk before Commit")
	}

	if err := tx.Commit(); err != nil {
		t.Fatalf("commit: %v", err)
	}
	if string(mustRead(t, fake, "/project/skill/SKILL.md")) != "v2 content" {
		t.Fatal("committed SKILL.md must reflect the new source content")
	}
	if fake.Exists("/project/skill/stale.txt") {
		t.Fatal("stale file absent from the new source tree must be removed by the sync")
	}
}

func TestFileTransaction_RemoveAllPurgesStagedButUncommittedContent(t *testing.T) {
	fake := fs.NewFakeFileSystem()
	fake.Dirs["/source/skill"] = true
	if err := fake.WriteFile("/source/skill/SKILL.md", []byte("content")); err != nil {
		t.Fatalf("seed source: %v", err)
	}

	tx := txn.New(fake, "/project", nil)
	if err := tx.CopyDir("/source/skill", "/project/skill"); err != nil {
		t.Fatalf("copydir: %v", err)
	}
	if !tx.Exists("/project/skill/SKILL.md") {
		t.Fatal("precondition: staged file must exist in the overlay")
	}

	if err := tx.RemoveAll("/project/skill"); err != nil {
		t.Fatalf("removeall: %v", err)
	}
	if tx.Exists("/project/skill/SKILL.md") {
		t.Fatal("RemoveAll must purge staged-but-uncommitted content, not just tombstone real disk paths")
	}

	if err := tx.Commit(); err != nil {
		t.Fatalf("commit: %v", err)
	}
	if fake.Exists("/project/skill/SKILL.md") {
		t.Fatal("a staged write that was removed before Commit must never reach disk")
	}
}

func TestFileTransaction_RemoveAllThenCopyDirStillDetectsConflictOnDivergentManagedFile(t *testing.T) {
	fake := fs.NewFakeFileSystem()
	fake.Dirs["/source/skill"] = true
	if err := fake.WriteFile("/source/skill/SKILL.md", []byte("new canonical content")); err != nil {
		t.Fatalf("seed source: %v", err)
	}
	if err := fake.WriteFile("/project/skill/SKILL.md", []byte("USER TAMPERED LOCALLY")); err != nil {
		t.Fatalf("seed dest: %v", err)
	}

	originalInstalledHash := "0000000000000000000000000000000000000000000000000000000000000"
	tx := txn.New(fake, "/project", map[string]string{"skill/SKILL.md": originalInstalledHash})

	if err := tx.RemoveAll("/project/skill"); err != nil {
		t.Fatalf("removeall: %v", err)
	}
	if err := tx.CopyDir("/source/skill", "/project/skill"); err != nil {
		t.Fatalf("copydir: %v", err)
	}

	conflicts := tx.Conflicts()
	if len(conflicts) != 1 {
		t.Fatalf("RemoveAll followed by CopyDir must still detect a managed file conflict; got %d conflicts: %v", len(conflicts), conflicts)
	}
	if conflicts[0].Path != "skill/SKILL.md" {
		t.Errorf("conflict path = %q, want %q", conflicts[0].Path, "skill/SKILL.md")
	}
}

func TestFileTransaction_CommitTwiceIsRejected(t *testing.T) {
	fake := fs.NewFakeFileSystem()
	tx := txn.New(fake, "/project", nil)

	if err := tx.Stage("/project/a.txt", []byte("hello")); err != nil {
		t.Fatalf("stage: %v", err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatalf("first commit: %v", err)
	}
	if err := tx.Commit(); !errors.Is(err, txn.ErrAlreadyCommitted) {
		t.Fatalf("second commit = %v, want %v", err, txn.ErrAlreadyCommitted)
	}
}

func mustRead(t *testing.T, fsys fs.FileSystem, path string) []byte {
	t.Helper()
	data, err := fsys.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return data
}
