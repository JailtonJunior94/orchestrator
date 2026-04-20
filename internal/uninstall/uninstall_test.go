package uninstall

import (
	"testing"

	"github.com/JailtonJunior94/ai-spec-harness/internal/fs"
	"github.com/JailtonJunior94/ai-spec-harness/internal/output"
)

func TestUninstall_RemovesSkills(t *testing.T) {
	t.Parallel()
	ffs := fs.NewFakeFileSystem()
	ffs.Files["/project/.agents/skills/review/SKILL.md"] = []byte("---\nversion: 1.0.0\n---\n")
	ffs.Files["/project/.agents/skills/bugfix/SKILL.md"] = []byte("---\nversion: 1.0.0\n---\n")
	ffs.Files["/project/AGENTS.md"] = []byte("# AGENTS")
	ffs.Files["/project/CLAUDE.md"] = []byte("# CLAUDE")
	ffs.Files["/project/.ai_spec_harness.json"] = []byte("{}")
	ffs.Files["/project/.claude/hooks/validate-governance.sh"] = []byte("gov")
	ffs.Files["/project/.claude/hooks/validate-preload.sh"] = []byte("pre")
	ffs.Files["/project/.claude/settings.local.json"] = []byte(defaultClaudeSettings())
	ffs.Files["/project/.claude/scripts/validate-bugfix-evidence.sh"] = []byte("bugfix")
	ffs.Files["/project/.claude/scripts/validate-refactor-evidence.sh"] = []byte("refactor")
	ffs.Files["/project/scripts/lib/parse-hook-input.sh"] = []byte("helper")
	ffs.Files["/project/scripts/lib/check-invocation-depth.sh"] = []byte("depth")

	printer := output.New(false)
	svc := NewService(ffs, printer)

	err := svc.Execute("/project", false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if ffs.Exists("/project/.agents/skills/review/SKILL.md") {
		t.Error("skill review should be removed")
	}
	if ffs.Exists("/project/AGENTS.md") {
		t.Error("AGENTS.md should be removed")
	}
	if ffs.Exists("/project/CLAUDE.md") {
		t.Error("CLAUDE.md should be removed")
	}
	if ffs.Exists("/project/.ai_spec_harness.json") {
		t.Error("manifest should be removed")
	}
	if ffs.Exists("/project/.claude/hooks/validate-preload.sh") {
		t.Error("validate-preload hook should be removed")
	}
	if ffs.Exists("/project/.claude/settings.local.json") {
		t.Error("generated settings.local.json should be removed")
	}
	if ffs.Exists("/project/scripts/lib/parse-hook-input.sh") {
		t.Error("parse-hook-input helper should be removed")
	}
	if ffs.Exists("/project/.claude/scripts/validate-bugfix-evidence.sh") {
		t.Error("validate-bugfix-evidence.sh should be removed")
	}
	if ffs.Exists("/project/.claude/scripts/validate-refactor-evidence.sh") {
		t.Error("validate-refactor-evidence.sh should be removed")
	}
	if ffs.Exists("/project/scripts/lib/check-invocation-depth.sh") {
		t.Error("check-invocation-depth.sh should be removed")
	}
}

func TestUninstall_RemovesGeminiHook(t *testing.T) {
	t.Parallel()
	ffs := fs.NewFakeFileSystem()
	ffs.Files["/project/.agents/skills/review/SKILL.md"] = []byte("---\nversion: 1.0.0\n---\n")
	ffs.Files["/project/.gemini/commands/review.toml"] = []byte("[command]")
	ffs.Files["/project/.gemini/hooks/validate-preload.sh"] = []byte("#!/usr/bin/env bash")

	printer := output.New(false)
	svc := NewService(ffs, printer)

	err := svc.Execute("/project", false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if ffs.Exists("/project/.gemini/hooks/validate-preload.sh") {
		t.Error("gemini hook validate-preload.sh should be removed")
	}
}

func TestUninstall_DryRunDoesNotRemove(t *testing.T) {
	t.Parallel()
	ffs := fs.NewFakeFileSystem()
	ffs.Files["/project/.agents/skills/review/SKILL.md"] = []byte("---\nversion: 1.0.0\n---\n")
	ffs.Files["/project/AGENTS.md"] = []byte("# AGENTS")

	printer := output.New(false)
	svc := NewService(ffs, printer)

	err := svc.Execute("/project", true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// In dry-run, files should still exist
	if !ffs.Exists("/project/.agents/skills/review/SKILL.md") {
		t.Error("skill should NOT be removed in dry-run")
	}
	if !ffs.Exists("/project/AGENTS.md") {
		t.Error("AGENTS.md should NOT be removed in dry-run")
	}
}

func TestUninstall_NoSkillsDir(t *testing.T) {
	t.Parallel()
	ffs := fs.NewFakeFileSystem()
	ffs.Dirs["/project"] = true

	printer := output.New(false)
	svc := NewService(ffs, printer)

	err := svc.Execute("/project", false)
	if err == nil {
		t.Fatal("expected error for missing .agents/skills/")
	}
}

func TestUninstall_PreservesAgentsLocal(t *testing.T) {
	t.Parallel()
	ffs := fs.NewFakeFileSystem()
	ffs.Files["/project/.agents/skills/review/SKILL.md"] = []byte("---\nversion: 1.0.0\n---\n")
	ffs.Files["/project/AGENTS.local.md"] = []byte("# Local extensions")

	printer := output.New(false)
	svc := NewService(ffs, printer)

	err := svc.Execute("/project", false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !ffs.Exists("/project/AGENTS.local.md") {
		t.Error("AGENTS.local.md should be preserved")
	}
}
