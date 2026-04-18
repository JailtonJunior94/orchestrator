package upgrade

import (
	"testing"

	"github.com/JailtonJunior94/ai-spec-harness/internal/adapters"
	"github.com/JailtonJunior94/ai-spec-harness/internal/config"
	"github.com/JailtonJunior94/ai-spec-harness/internal/contextgen"
	"github.com/JailtonJunior94/ai-spec-harness/internal/fs"
	"github.com/JailtonJunior94/ai-spec-harness/internal/manifest"
	"github.com/JailtonJunior94/ai-spec-harness/internal/output"
)

func setupTestService(ffs *fs.FakeFileSystem) *Service {
	printer := output.New(false)
	mfst := manifest.NewStore(ffs)
	adpt := adapters.NewGenerator(ffs, printer)
	ctxg := contextgen.NewGenerator(ffs, printer)
	return NewService(ffs, printer, mfst, adpt, ctxg)
}

func TestUpgrade_NoSkillsDir(t *testing.T) {
	ffs := fs.NewFakeFileSystem()
	ffs.Dirs["/project"] = true
	ffs.Dirs["/source"] = true
	svc := setupTestService(ffs)

	err := svc.Execute(config.UpgradeOptions{
		ProjectDir: "/project",
		SourceDir:  "/source",
	})

	if err == nil {
		t.Fatal("expected error when .agents/skills/ is missing")
	}
}

func TestUpgrade_CheckOnly_Outdated(t *testing.T) {
	ffs := fs.NewFakeFileSystem()

	// Source com versao mais nova
	ffs.Files["/source/.agents/skills/review/SKILL.md"] = []byte("---\nversion: 2.0.0\ndescription: Review.\n---\n")

	// Target com versao antiga
	ffs.Files["/project/.agents/skills/review/SKILL.md"] = []byte("---\nversion: 1.0.0\ndescription: Review.\n---\n")

	svc := setupTestService(ffs)

	err := svc.Execute(config.UpgradeOptions{
		ProjectDir: "/project",
		SourceDir:  "/source",
		CheckOnly:  true,
	})

	if err == nil {
		t.Fatal("expected error for outdated skills in check-only mode")
	}
}

func TestUpgrade_CheckOnly_UpToDate(t *testing.T) {
	ffs := fs.NewFakeFileSystem()

	content := []byte("---\nversion: 1.0.0\ndescription: Review.\n---\n")
	ffs.Files["/source/.agents/skills/review/SKILL.md"] = content
	ffs.Files["/project/.agents/skills/review/SKILL.md"] = content

	svc := setupTestService(ffs)

	err := svc.Execute(config.UpgradeOptions{
		ProjectDir: "/project",
		SourceDir:  "/source",
		CheckOnly:  true,
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
