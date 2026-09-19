package upgrade

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/JailtonJunior94/ai-spec-harness/internal/adapters"
	"github.com/JailtonJunior94/ai-spec-harness/internal/config"
	"github.com/JailtonJunior94/ai-spec-harness/internal/contextgen"
	"github.com/JailtonJunior94/ai-spec-harness/internal/fs"
	"github.com/JailtonJunior94/ai-spec-harness/internal/manifest"
	"github.com/JailtonJunior94/ai-spec-harness/internal/output"
	"github.com/JailtonJunior94/ai-spec-harness/internal/skills"
)

type refsFileHashFailingFS struct {
	fs.FileSystem
	failPath string
}

func (f *refsFileHashFailingFS) FileHash(path string) (string, error) {
	if path == f.failPath {
		return "", errors.New("permission denied")
	}
	return f.FileSystem.FileHash(path)
}

func TestUpgrade_RefsChangedFilesNeverSilentlyIgnoresHashReadFailure(t *testing.T) {
	t.Parallel()
	ffs := fs.NewFakeFileSystem()
	ffs.Files["/source/references/b.md"] = []byte("changed")
	ffs.Files["/project/references/b.md"] = []byte("old")

	decorated := &refsFileHashFailingFS{FileSystem: ffs, failPath: "/project/references/b.md"}
	printer := output.New(false)
	svc := NewService(decorated, printer, manifest.NewStore(decorated), adapters.NewGenerator(decorated, printer), contextgen.NewGenerator(decorated, printer))

	changed := svc.refsChangedFiles("/source/references", "/project/references")
	got := strings.Join(changed, "\n")

	if !strings.Contains(got, "~ b.md (modificado)") {
		t.Fatalf("uma falha de leitura de hash nao pode ser tratada como ausencia de mudanca (fail-open); changed=%q", got)
	}
}

func hashOfContent(data []byte) string {
	sum := sha256.Sum256(data)
	return fmt.Sprintf("%x", sum[:])
}

func TestUpgrade_RecordsFileChecksumForUpdatedSkillPath(t *testing.T) {
	t.Parallel()

	ffs := fs.NewFakeFileSystem()
	newContent := []byte("---\nname: review\nversion: 2.0.0\ndescription: Review.\n---\n")
	ffs.Files["/source/.agents/skills/review/SKILL.md"] = newContent
	ffs.Files["/project/.agents/skills/review/SKILL.md"] = []byte("---\nname: review\nversion: 1.0.0\ndescription: Review.\n---\n")

	store := manifest.NewStore(ffs)
	if err := store.Save("/project", &manifest.Manifest{
		Version:        "0.11.2",
		CreatedAt:      time.Unix(1700000000, 0),
		UpdatedAt:      time.Unix(1700000000, 0),
		Langs:          []skills.Lang{},
		InstalledFiles: []string{},
	}); err != nil {
		t.Fatalf("falha ao salvar manifesto inicial: %v", err)
	}

	svc := setupTestService(ffs)
	if err := svc.Execute(config.UpgradeOptions{
		ProjectDir: "/project",
		SourceDir:  "/source",
	}); err != nil {
		t.Fatalf("upgrade falhou: %v", err)
	}

	mf := readManifestFromFakeFS(t, ffs, "/project/.ai_spec_harness.json")
	const rel = ".agents/skills/review/SKILL.md"
	got, ok := mf.FileChecksums[rel]
	if !ok {
		t.Fatalf("esperava checksum registrado para %s, manifesto: %+v", rel, mf.FileChecksums)
	}
	if want := hashOfContent(newContent); got != want {
		t.Errorf("checksum divergente para %s: got %s, want %s", rel, got, want)
	}

	found := false
	for _, p := range mf.InstalledFiles {
		if p == rel {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("esperava %s em InstalledFiles apos upgrade, obteve: %v", rel, mf.InstalledFiles)
	}
}

func TestUpgrade_MergeFileTrackingPreservesUntouchedPreviousEntries(t *testing.T) {
	t.Parallel()

	ffs := fs.NewFakeFileSystem()
	newContent := []byte("---\nname: review\nversion: 2.0.0\ndescription: Review.\n---\n")
	ffs.Files["/source/.agents/skills/review/SKILL.md"] = newContent
	ffs.Files["/project/.agents/skills/review/SKILL.md"] = []byte("---\nname: review\nversion: 1.0.0\ndescription: Review.\n---\n")

	store := manifest.NewStore(ffs)
	if err := store.Save("/project", &manifest.Manifest{
		Version:        "0.11.2",
		CreatedAt:      time.Unix(1700000000, 0),
		UpdatedAt:      time.Unix(1700000000, 0),
		Langs:          []skills.Lang{},
		InstalledFiles: []string{"AGENTS.md"},
		FileChecksums:  map[string]string{"AGENTS.md": "deadbeefdeadbeef"},
	}); err != nil {
		t.Fatalf("falha ao salvar manifesto inicial: %v", err)
	}

	svc := setupTestService(ffs)
	if err := svc.Execute(config.UpgradeOptions{
		ProjectDir: "/project",
		SourceDir:  "/source",
	}); err != nil {
		t.Fatalf("upgrade falhou: %v", err)
	}

	mf := readManifestFromFakeFS(t, ffs, "/project/.ai_spec_harness.json")
	if got := mf.FileChecksums["AGENTS.md"]; got != "deadbeefdeadbeef" {
		t.Errorf("checksum previamente rastreado nao pode ser perdido por upgrade parcial: got %q", got)
	}

	found := false
	for _, p := range mf.InstalledFiles {
		if p == "AGENTS.md" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("AGENTS.md deveria permanecer em InstalledFiles apos upgrade parcial, obteve: %v", mf.InstalledFiles)
	}
}

func TestUpgrade_LegacyManifestWithoutTrackingBackfillsChecksumsViaSkillsWalkOnVersionOnlyBump(t *testing.T) {
	t.Parallel()

	ffs := fs.NewFakeFileSystem()
	content := []byte("---\nname: review\nversion: 1.4.0\ndescription: Review.\n---\n")
	ffs.Files["/source/.agents/skills/review/SKILL.md"] = content
	ffs.Files["/project/.agents/skills/review/SKILL.md"] = content

	setVersionForTest(t, "0.12.0")

	store := manifest.NewStore(ffs)
	if err := store.Save("/project", &manifest.Manifest{
		Version:   "0.11.2",
		CreatedAt: time.Unix(1690000000, 0),
		UpdatedAt: time.Unix(1700000000, 0),
		Langs:     []skills.Lang{},
	}); err != nil {
		t.Fatalf("falha ao salvar manifesto inicial: %v", err)
	}

	svc := setupTestService(ffs)
	if err := svc.Execute(config.UpgradeOptions{
		ProjectDir: "/project",
		SourceDir:  "/source",
	}); err != nil {
		t.Fatalf("upgrade falhou: %v", err)
	}

	mf := readManifestFromFakeFS(t, ffs, "/project/.ai_spec_harness.json")
	if mf.Version != "0.12.0" {
		t.Fatalf("version do manifesto nao atualizada: got %q want %q", mf.Version, "0.12.0")
	}
	if mf.HasFileTracking() {
		t.Errorf("manifesto legado sem rastreio nao pode passar a rastrear InstalledFiles/MergedFiles so por causa de bump de versao sem escrita real: InstalledFiles=%v", mf.InstalledFiles)
	}
	wantHash, err := ffs.FileHash("/project/.agents/skills/review/SKILL.md")
	if err != nil {
		t.Fatalf("falha ao calcular hash esperado: %v", err)
	}
	gotHash, ok := mf.FileChecksums[".agents/skills/review/SKILL.md"]
	if !ok {
		t.Fatal("manifesto legado sem InstalledFiles deve ganhar FileChecksums via varredura de .agents/skills/ — sem isso, a instalacao nunca ganha protecao de conflito (RF-22/24/27)")
	}
	if gotHash != wantHash {
		t.Errorf("checksum de backfill legado incorreto: got %s, want %s", gotHash, wantHash)
	}
}
