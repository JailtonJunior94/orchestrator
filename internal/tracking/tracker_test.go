package tracking

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"os"
	"testing"

	"github.com/JailtonJunior94/ai-spec-harness/internal/fs"
)

func hashOf(data []byte) string {
	sum := sha256.Sum256(data)
	return fmt.Sprintf("%x", sum[:])
}

func TestTracker_WriteFileRecordsMatchingChecksum(t *testing.T) {
	ffs := fs.NewFakeFileSystem()
	ffs.Dirs["/project"] = true
	tr := New(ffs, "/project", ".ai_spec_harness.json")

	data := []byte("hello world")
	if err := tr.WriteFile("/project/a.txt", data); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	checksums := tr.Checksums()
	got, ok := checksums["a.txt"]
	if !ok {
		t.Fatal("checksum nao registrado para a.txt")
	}
	if want := hashOf(data); got != want {
		t.Errorf("checksum divergente: got %s, want %s", got, want)
	}
	if len(tr.CreatedPaths()) != 1 || tr.CreatedPaths()[0] != "a.txt" {
		t.Errorf("a.txt deveria estar em CreatedPaths, got %v", tr.CreatedPaths())
	}
}

func TestTracker_ExcludedManifestFileNeverTracked(t *testing.T) {
	ffs := fs.NewFakeFileSystem()
	ffs.Dirs["/project"] = true
	tr := New(ffs, "/project", ".ai_spec_harness.json")

	if err := tr.WriteFile("/project/.ai_spec_harness.json", []byte("{}")); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	if len(tr.CreatedPaths()) != 0 {
		t.Errorf("manifesto nao deve ser rastreado: %v", tr.CreatedPaths())
	}
	if len(tr.Checksums()) != 0 {
		t.Errorf("manifesto nao deve ter checksum: %v", tr.Checksums())
	}
}

func TestTracker_CopyDirRecordsChecksumPerFileNotDirectory(t *testing.T) {
	ffs := fs.NewFakeFileSystem()
	ffs.Dirs["/project"] = true
	ffs.Dirs["/source/skill"] = true
	ffs.Files["/source/skill/SKILL.md"] = []byte("conteudo")
	ffs.Files["/source/skill/nested/ref.md"] = []byte("ref")

	tr := New(ffs, "/project", ".ai_spec_harness.json")
	if err := tr.CopyDir("/source/skill", "/project/skill"); err != nil {
		t.Fatalf("CopyDir: %v", err)
	}

	checksums := tr.Checksums()
	if _, ok := checksums["skill/SKILL.md"]; !ok {
		t.Error("SKILL.md deveria ter checksum registrado")
	}
	if _, ok := checksums["skill/nested/ref.md"]; !ok {
		t.Error("nested/ref.md deveria ter checksum registrado")
	}
	if _, ok := checksums["skill"]; ok {
		t.Error("diretorio nao deve receber entrada de checksum")
	}
}

func TestTracker_SymlinkToDirectorySkipsChecksumButTracksPath(t *testing.T) {
	ffs := fs.NewFakeFileSystem()
	ffs.Dirs["/project"] = true
	ffs.Dirs["/source/skill"] = true
	ffs.Files["/source/skill/SKILL.md"] = []byte("conteudo")

	tr := New(ffs, "/project", ".ai_spec_harness.json")
	if err := tr.Symlink("../source/skill", "/project/skill"); err != nil {
		t.Fatalf("Symlink: %v", err)
	}

	if len(tr.Checksums()) != 0 {
		t.Errorf("symlink de diretorio nao deve gerar checksum: %v", tr.Checksums())
	}
	if len(tr.CreatedPaths()) != 1 || tr.CreatedPaths()[0] != "skill" {
		t.Errorf("symlink deveria ser rastreado como created path: %v", tr.CreatedPaths())
	}
}

func TestTracker_MarkMergedReclassifiesAndKeepsChecksum(t *testing.T) {
	ffs := fs.NewFakeFileSystem()
	ffs.Dirs["/project"] = true

	tr := New(ffs, "/project", ".ai_spec_harness.json")
	data := []byte("merged content")
	if err := tr.WriteFile("/project/settings.json", data); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	if err := tr.MarkMerged("/project/settings.json"); err != nil {
		t.Fatalf("MarkMerged: %v", err)
	}

	if len(tr.CreatedPaths()) != 0 {
		t.Errorf("path mesclado nao deve permanecer em CreatedPaths: %v", tr.CreatedPaths())
	}
	if len(tr.MergedPaths()) != 1 || tr.MergedPaths()[0] != "settings.json" {
		t.Errorf("path deveria estar em MergedPaths: %v", tr.MergedPaths())
	}
	if got, ok := tr.Checksums()["settings.json"]; !ok || got != hashOf(data) {
		t.Errorf("checksum deveria permanecer apos MarkMerged: %v", tr.Checksums())
	}
}

func TestTracker_MarkInstalledOnPreexistingFileComputesChecksum(t *testing.T) {
	ffs := fs.NewFakeFileSystem()
	ffs.Dirs["/project"] = true
	data := []byte("ja existente no disco")
	ffs.Files["/project/hooks/validate.sh"] = data

	tr := New(ffs, "/project", ".ai_spec_harness.json")
	if err := tr.MarkInstalled("/project/hooks/validate.sh"); err != nil {
		t.Fatalf("MarkInstalled: %v", err)
	}

	if got, ok := tr.Checksums()["hooks/validate.sh"]; !ok || got != hashOf(data) {
		t.Errorf("checksum deveria ser calculado a partir do conteudo em disco: %v", tr.Checksums())
	}
}

func TestTracker_HashReadFailurePropagatesErrorNeverSucceedsSilently(t *testing.T) {
	ffs := fs.NewFakeFileSystem()
	ffs.Dirs["/project"] = true

	tr := New(ffs, "/project", ".ai_spec_harness.json")
	err := tr.MarkInstalled("/project/nao-existe.txt")
	if err == nil {
		t.Fatal("falha de leitura de hash nao pode produzir sucesso")
	}
	if len(tr.Checksums()) != 0 {
		t.Errorf("nenhum checksum deve ser registrado quando o hash falha: %v", tr.Checksums())
	}
}

type hashFailingFS struct {
	fs.FileSystem
	failPath string
}

func (f *hashFailingFS) FileHash(path string) (string, error) {
	if path == f.failPath {
		return "", errors.New("permission denied")
	}
	return f.FileSystem.FileHash(path)
}

func TestTracker_MarkInstalledErrorsWhenHashOfPreexistingFileFails(t *testing.T) {
	ffs := fs.NewFakeFileSystem()
	ffs.Dirs["/project"] = true
	ffs.Files["/project/a.txt"] = []byte("ja existente no disco")
	decorated := &hashFailingFS{FileSystem: ffs, failPath: "/project/a.txt"}

	tr := New(decorated, "/project", ".ai_spec_harness.json")
	err := tr.MarkInstalled("/project/a.txt")
	if err == nil {
		t.Fatal("MarkInstalled deveria propagar a falha de hash, nunca reportar sucesso")
	}
	if len(tr.Checksums()) != 0 {
		t.Errorf("nenhum checksum deve ser registrado quando o hash falha: %v", tr.Checksums())
	}
}

type readDirFailingFS struct {
	fs.FileSystem
	failPath string
}

func (f *readDirFailingFS) ReadDir(path string) ([]os.DirEntry, error) {
	if path == f.failPath {
		return nil, errors.New("permission denied")
	}
	return f.FileSystem.ReadDir(path)
}

func TestTracker_CopyDirPropagatesReadDirFailureNeverSucceedsSilently(t *testing.T) {
	ffs := fs.NewFakeFileSystem()
	ffs.Dirs["/project"] = true
	ffs.Dirs["/source/skill"] = true
	ffs.Files["/source/skill/SKILL.md"] = []byte("conteudo")
	ffs.Dirs["/source/skill/nested"] = true
	ffs.Files["/source/skill/nested/ref.md"] = []byte("ref")

	decorated := &readDirFailingFS{FileSystem: ffs, failPath: "/source/skill/nested"}
	tr := New(decorated, "/project", ".ai_spec_harness.json")

	err := tr.CopyDir("/source/skill", "/project/skill")
	if err == nil {
		t.Fatal("falha de ReadDir ao percorrer a arvore copiada nao pode ser reportada como sucesso")
	}
	if _, ok := tr.Checksums()["skill/nested/ref.md"]; ok {
		t.Error("checksum nao deveria ter sido registrado para arquivo sob diretorio com falha de leitura")
	}
}

func TestTracker_StagedWritesDoNotReachDiskBeforeCommit(t *testing.T) {
	ffs := fs.NewFakeFileSystem()
	ffs.Dirs["/project"] = true
	tr := New(ffs, "/project", ".ai_spec_harness.json")

	if err := tr.WriteFile("/project/a.txt", []byte("data")); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	if ffs.Exists("/project/a.txt") {
		t.Fatal("install/upgrade batch must stay staged until Commit — RF-28")
	}
	if err := tr.Commit(); err != nil {
		t.Fatalf("Commit: %v", err)
	}
	if !ffs.Exists("/project/a.txt") {
		t.Fatal("Commit must persist staged writes to disk")
	}
}

func TestTracker_ConflictAbortsBatchByDefault(t *testing.T) {
	ffs := fs.NewFakeFileSystem()
	ffs.Dirs["/project"] = true
	ffs.Files["/project/managed.md"] = []byte("tampered by user")

	tr := NewTransactional(ffs, "/project", ".ai_spec_harness.json", map[string]string{
		"managed.md": "expected-hash-that-never-matches",
	})
	if err := tr.WriteFile("/project/managed.md", []byte("new governed content")); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	if err := tr.WriteFile("/project/other.md", []byte("part of the same batch")); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	if len(tr.Conflicts()) != 1 {
		t.Fatalf("expected exactly one conflict, got %d: %v", len(tr.Conflicts()), tr.Conflicts())
	}

	if err := tr.Commit(); err == nil {
		t.Fatal("Commit must abort the whole batch when a managed file conflicts")
	}
	if ffs.Exists("/project/other.md") {
		t.Fatal("no file in the batch may be written when the batch aborts on conflict")
	}
}

func TestTracker_AllowOverwriteCommitsDespiteConflict(t *testing.T) {
	ffs := fs.NewFakeFileSystem()
	ffs.Dirs["/project"] = true
	ffs.Files["/project/managed.md"] = []byte("tampered by user")

	tr := NewTransactional(ffs, "/project", ".ai_spec_harness.json", map[string]string{
		"managed.md": "expected-hash-that-never-matches",
	})
	tr.AllowOverwrite()

	if err := tr.WriteFile("/project/managed.md", []byte("new governed content")); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	if err := tr.Commit(); err != nil {
		t.Fatalf("Commit with overwrite allowed: %v", err)
	}

	data, err := ffs.ReadFile("/project/managed.md")
	if err != nil {
		t.Fatalf("read committed file: %v", err)
	}
	if string(data) != "new governed content" {
		t.Errorf("committed content = %q, want overwritten content", data)
	}
}

func TestTracker_OutcomesClassifyCreatedUpdatedPreserved(t *testing.T) {
	ffs := fs.NewFakeFileSystem()
	ffs.Dirs["/project"] = true
	ffs.Files["/project/same.md"] = []byte("identical")
	ffs.Files["/project/changed.md"] = []byte("before")

	tr := New(ffs, "/project", ".ai_spec_harness.json")
	if err := tr.WriteFile("/project/same.md", []byte("identical")); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	if err := tr.WriteFile("/project/changed.md", []byte("after")); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	if err := tr.WriteFile("/project/new.md", []byte("brand new")); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	if err := tr.Commit(); err != nil {
		t.Fatalf("Commit: %v", err)
	}

	outcomes := tr.Outcomes()
	if outcomes["same.md"].String() != "preserved" {
		t.Errorf("outcome[same.md] = %v, want preserved", outcomes["same.md"])
	}
	if outcomes["changed.md"].String() != "updated" {
		t.Errorf("outcome[changed.md] = %v, want updated", outcomes["changed.md"])
	}
	if outcomes["new.md"].String() != "created" {
		t.Errorf("outcome[new.md] = %v, want created", outcomes["new.md"])
	}
}

func TestTracker_RemoveAllForgetsChecksumsUnderPrefix(t *testing.T) {
	ffs := fs.NewFakeFileSystem()
	ffs.Dirs["/project"] = true
	ffs.Dirs["/source/skill"] = true
	ffs.Files["/source/skill/SKILL.md"] = []byte("conteudo")

	tr := New(ffs, "/project", ".ai_spec_harness.json")
	if err := tr.CopyDir("/source/skill", "/project/skill"); err != nil {
		t.Fatalf("CopyDir: %v", err)
	}
	if len(tr.Checksums()) == 0 {
		t.Fatal("pre-condicao: checksum deveria existir antes do RemoveAll")
	}

	if err := tr.RemoveAll("/project/skill"); err != nil {
		t.Fatalf("RemoveAll: %v", err)
	}
	if len(tr.Checksums()) != 0 {
		t.Errorf("checksums sob o prefixo removido deveriam ser esquecidos: %v", tr.Checksums())
	}
	if len(tr.CreatedPaths()) != 0 {
		t.Errorf("created paths sob o prefixo removido deveriam ser esquecidos: %v", tr.CreatedPaths())
	}
}
