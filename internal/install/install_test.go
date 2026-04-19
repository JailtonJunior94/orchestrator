package install

import (
	"encoding/json"
	"strings"
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
	ffs.Files["/source/.claude/scripts/validate-task-evidence.sh"] = []byte("#!/usr/bin/env bash")
	ffs.Files["/source/.claude/hooks/validate-governance.sh"] = []byte("#!/usr/bin/env bash")
	ffs.Files["/source/.claude/hooks/validate-preload.sh"] = []byte("#!/usr/bin/env bash")
	ffs.Files["/source/scripts/lib/parse-hook-input.sh"] = []byte("#!/usr/bin/env bash")

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
	if !ffs.Exists("/project/.claude/hooks/validate-governance.sh") {
		t.Error("hook validate-governance nao copiado")
	}
	if !ffs.Exists("/project/.claude/hooks/validate-preload.sh") {
		t.Error("hook validate-preload nao copiado")
	}
	if !ffs.Exists("/project/scripts/lib/parse-hook-input.sh") {
		t.Error("parse-hook-input.sh nao copiado")
	}
	settings, err := ffs.ReadFile("/project/.claude/settings.local.json")
	if err != nil {
		t.Fatalf("settings.local.json nao criado: %v", err)
	}
	content := string(settings)
	if !strings.Contains(content, "validate-governance.sh") {
		t.Error("settings.local.json sem PostToolUse")
	}
	if !strings.Contains(content, "validate-preload.sh") {
		t.Error("settings.local.json sem PreToolUse")
	}
}

func TestInstall_Codex_LeanProfile(t *testing.T) {
	ffs := fs.NewFakeFileSystem()
	ffs.Dirs["/project"] = true
	ffs.Dirs["/source"] = true
	svc := setupTestService(ffs)

	err := svc.Execute(config.InstallOptions{
		ProjectDir:   "/project",
		SourceDir:    "/source",
		Tools:        []skills.Tool{skills.ToolCodex},
		Langs:        nil,
		LinkMode:     skills.LinkCopy,
		CodexProfile: "lean",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, err := ffs.ReadFile("/project/.codex/config.toml")
	if err != nil {
		t.Fatalf("config.toml not created: %v", err)
	}
	content := string(data)

	if strings.Contains(content, "analyze-project") {
		t.Error("lean profile should not include analyze-project")
	}
	if !strings.Contains(content, "agent-governance") {
		t.Error("lean profile should include agent-governance")
	}
}

func TestInstall_Codex_FullProfile(t *testing.T) {
	ffs := fs.NewFakeFileSystem()
	ffs.Dirs["/project"] = true
	ffs.Dirs["/source"] = true
	svc := setupTestService(ffs)

	err := svc.Execute(config.InstallOptions{
		ProjectDir:   "/project",
		SourceDir:    "/source",
		Tools:        []skills.Tool{skills.ToolCodex},
		Langs:        nil,
		LinkMode:     skills.LinkCopy,
		CodexProfile: "full",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, err := ffs.ReadFile("/project/.codex/config.toml")
	if err != nil {
		t.Fatalf("config.toml not created: %v", err)
	}
	content := string(data)

	if !strings.Contains(content, "analyze-project") {
		t.Error("full profile should include analyze-project")
	}
}

func TestInstall_Idempotent(t *testing.T) {
	ffs := fs.NewFakeFileSystem()
	ffs.Dirs["/project"] = true
	ffs.Dirs["/source"] = true
	ffs.Files["/source/AGENTS.md"] = []byte("# AGENTS")
	ffs.Files["/source/.claude/rules/governance.md"] = []byte("# governance")
	ffs.Files["/source/.claude/scripts/validate-task-evidence.sh"] = []byte("#!/usr/bin/env bash")
	ffs.Files["/source/.claude/hooks/validate-governance.sh"] = []byte("#!/usr/bin/env bash")
	ffs.Files["/source/.claude/hooks/validate-preload.sh"] = []byte("#!/usr/bin/env bash")
	ffs.Files["/source/scripts/lib/parse-hook-input.sh"] = []byte("#!/usr/bin/env bash")
	svc := setupTestService(ffs)

	opts := config.InstallOptions{
		ProjectDir: "/project",
		SourceDir:  "/source",
		Tools:      []skills.Tool{skills.ToolClaude},
		LinkMode:   skills.LinkCopy,
	}

	if err := svc.Execute(opts); err != nil {
		t.Fatalf("first execution failed: %v", err)
	}

	fileCountAfterFirst := len(ffs.Files)

	if err := svc.Execute(opts); err != nil {
		t.Fatalf("second execution failed: %v", err)
	}

	fileCountAfterSecond := len(ffs.Files)
	if fileCountAfterFirst != fileCountAfterSecond {
		t.Errorf("file count changed: first=%d second=%d", fileCountAfterFirst, fileCountAfterSecond)
	}

	settings, err := ffs.ReadFile("/project/.claude/settings.local.json")
	if err != nil {
		t.Fatalf("settings.local.json not found: %v", err)
	}
	content := string(settings)
	if strings.Count(content, "validate-governance.sh") > 1 {
		t.Error("settings.local.json has duplicate validate-governance.sh hooks")
	}
	if strings.Count(content, "validate-preload.sh") > 1 {
		t.Error("settings.local.json has duplicate validate-preload.sh hooks")
	}

	manifestData, err := ffs.ReadFile("/project/.ai_spec_harness.json")
	if err != nil {
		t.Fatalf("manifest not found: %v", err)
	}
	var m interface{}
	if err := json.Unmarshal(manifestData, &m); err != nil {
		t.Errorf("manifest is not valid JSON: %v", err)
	}
}

func TestInstall_NoCtx_CopiesAGENTSMD(t *testing.T) {
	ffs := fs.NewFakeFileSystem()
	ffs.Dirs["/project"] = true
	ffs.Dirs["/source"] = true
	ffs.Files["/source/AGENTS.md"] = []byte("# AGENTS static")
	ffs.Files["/source/.claude/hooks/validate-governance.sh"] = []byte("#!/usr/bin/env bash")
	ffs.Files["/source/.claude/hooks/validate-preload.sh"] = []byte("#!/usr/bin/env bash")
	svc := setupTestService(ffs)

	err := svc.Execute(config.InstallOptions{
		ProjectDir:  "/project",
		SourceDir:   "/source",
		Tools:       []skills.Tool{skills.ToolClaude},
		LinkMode:    skills.LinkCopy,
		GenerateCtx: false,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !ffs.Exists("/project/AGENTS.md") {
		t.Error("AGENTS.md should have been copied from source")
	}
	data, err := ffs.ReadFile("/project/AGENTS.md")
	if err != nil {
		t.Fatalf("could not read AGENTS.md: %v", err)
	}
	if string(data) != "# AGENTS static" {
		t.Errorf("AGENTS.md content mismatch: got %q", string(data))
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
