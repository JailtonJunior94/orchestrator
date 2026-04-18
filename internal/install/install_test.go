package install

import (
	"testing"

	"github.com/JailtonJunior94/ai-spec-harness/internal/adapters"
	"github.com/JailtonJunior94/ai-spec-harness/internal/config"
	"github.com/JailtonJunior94/ai-spec-harness/internal/contextgen"
	"github.com/JailtonJunior94/ai-spec-harness/internal/fs"
	"github.com/JailtonJunior94/ai-spec-harness/internal/manifest"
	"github.com/JailtonJunior94/ai-spec-harness/internal/output"
	"github.com/JailtonJunior94/ai-spec-harness/internal/skills"
)

func setupTestService(ffs *fs.FakeFileSystem) *Service {
	printer := output.New(false)
	mfst := manifest.NewStore(ffs)
	adpt := adapters.NewGenerator(ffs, printer)
	ctxg := contextgen.NewGenerator(ffs, printer)
	return NewService(ffs, printer, mfst, adpt, ctxg)
}

func TestInstall_Validate_MissingProjectDir(t *testing.T) {
	ffs := fs.NewFakeFileSystem()
	svc := setupTestService(ffs)

	err := svc.Execute(config.InstallOptions{
		ProjectDir: "/nonexistent",
		SourceDir:  "/source",
		Tools:      []skills.Tool{skills.ToolClaude},
	})

	if err == nil {
		t.Fatal("expected error for nonexistent project dir")
	}
}

func TestInstall_Validate_NoTools(t *testing.T) {
	ffs := fs.NewFakeFileSystem()
	ffs.Dirs["/project"] = true
	ffs.Dirs["/source"] = true
	svc := setupTestService(ffs)

	err := svc.Execute(config.InstallOptions{
		ProjectDir: "/project",
		SourceDir:  "/source",
		Tools:      nil,
	})

	if err == nil {
		t.Fatal("expected error for no tools")
	}
}

func TestInstall_Validate_SameDir(t *testing.T) {
	ffs := fs.NewFakeFileSystem()
	ffs.Dirs["/project"] = true
	svc := setupTestService(ffs)

	err := svc.Execute(config.InstallOptions{
		ProjectDir: "/project",
		SourceDir:  "/project",
		Tools:      []skills.Tool{skills.ToolClaude},
		LinkMode:   skills.LinkCopy,
	})

	if err == nil {
		t.Fatal("expected error for same dir")
	}
}

func TestInstall_CopyMode(t *testing.T) {
	ffs := fs.NewFakeFileSystem()
	ffs.Dirs["/project"] = true
	ffs.Dirs["/source"] = true

	// Criar uma skill de teste na fonte
	ffs.Files["/source/.agents/skills/review/SKILL.md"] = []byte(`---
version: 1.0.0
description: Revisa codigo.
---

# Review
`)
	ffs.Files["/source/AGENTS.md"] = []byte("# AGENTS")
	ffs.Files["/source/CLAUDE.md"] = []byte("# CLAUDE")
	ffs.Files["/source/.claude/rules/governance.md"] = []byte("# governance")

	svc := setupTestService(ffs)

	err := svc.Execute(config.InstallOptions{
		ProjectDir:  "/project",
		SourceDir:   "/source",
		Tools:       []skills.Tool{skills.ToolClaude},
		Langs:       nil,
		LinkMode:    skills.LinkCopy,
		GenerateCtx: true,
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verificar manifesto criado
	if !ffs.Exists("/project/.ai_spec_harness.json") {
		t.Error("manifesto nao criado")
	}

	// Verificar skill copiada
	if !ffs.Exists("/project/.agents/skills/review/SKILL.md") {
		t.Error("skill review nao copiada para .agents/skills/")
	}

	// Verificar AGENTS.md copiado
	if !ffs.Exists("/project/AGENTS.md") {
		t.Error("AGENTS.md nao copiado")
	}
}

func TestInstall_DryRun(t *testing.T) {
	ffs := fs.NewFakeFileSystem()
	ffs.Dirs["/project"] = true
	ffs.Dirs["/source"] = true
	ffs.Files["/source/.agents/skills/review/SKILL.md"] = []byte("---\nversion: 1.0.0\n---\n")

	svc := setupTestService(ffs)

	err := svc.Execute(config.InstallOptions{
		ProjectDir:  "/project",
		SourceDir:   "/source",
		Tools:       []skills.Tool{skills.ToolClaude},
		LinkMode:    skills.LinkCopy,
		DryRun:      true,
		GenerateCtx: true,
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Em dry-run, manifesto NAO deve ser criado
	if ffs.Exists("/project/.ai_spec_harness.json") {
		t.Error("manifesto criado em dry-run")
	}
}
