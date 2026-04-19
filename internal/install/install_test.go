package install

import (
	"encoding/json"
	"os"
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
	ffs.Files["/source/.claude/scripts/validate-bugfix-evidence.sh"] = []byte("#!/usr/bin/env bash")
	ffs.Files["/source/.claude/scripts/validate-refactor-evidence.sh"] = []byte("#!/usr/bin/env bash")
	ffs.Files["/source/.claude/hooks/validate-governance.sh"] = []byte("#!/usr/bin/env bash")
	ffs.Files["/source/.claude/hooks/validate-preload.sh"] = []byte("#!/usr/bin/env bash")
	ffs.Files["/source/scripts/lib/parse-hook-input.sh"] = []byte("#!/usr/bin/env bash")
	ffs.Files["/source/scripts/lib/check-invocation-depth.sh"] = []byte("#!/usr/bin/env bash")

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
	if !ffs.Exists("/project/.claude/scripts/validate-bugfix-evidence.sh") {
		t.Error("validate-bugfix-evidence.sh nao copiado")
	}
	if !ffs.Exists("/project/.claude/scripts/validate-refactor-evidence.sh") {
		t.Error("validate-refactor-evidence.sh nao copiado")
	}
	if !ffs.Exists("/project/scripts/lib/check-invocation-depth.sh") {
		t.Error("check-invocation-depth.sh nao copiado")
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

func TestInstall_Gemini_CopiesHook(t *testing.T) {
	ffs := fs.NewFakeFileSystem()
	ffs.Dirs["/project"] = true
	ffs.Dirs["/source"] = true
	ffs.Files["/source/.gemini/hooks/validate-preload.sh"] = []byte("#!/usr/bin/env bash\n# gemini hook")

	svc := setupTestService(ffs)

	err := svc.Execute(config.InstallOptions{
		ProjectDir: "/project",
		SourceDir:  "/source",
		Tools:      []skills.Tool{skills.ToolGemini},
		LinkMode:   skills.LinkCopy,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !ffs.Exists("/project/.gemini/hooks/validate-preload.sh") {
		t.Error("gemini hook validate-preload.sh nao copiado")
	}
}

func TestInstall_Gemini_NoHookInSource_NoError(t *testing.T) {
	ffs := fs.NewFakeFileSystem()
	ffs.Dirs["/project"] = true
	ffs.Dirs["/source"] = true

	svc := setupTestService(ffs)

	err := svc.Execute(config.InstallOptions{
		ProjectDir: "/project",
		SourceDir:  "/source",
		Tools:      []skills.Tool{skills.ToolGemini},
		LinkMode:   skills.LinkCopy,
	})
	if err != nil {
		t.Fatalf("unexpected error when hook absent from source: %v", err)
	}

	if ffs.Exists("/project/.gemini/hooks/validate-preload.sh") {
		t.Error("hook should not be created when absent from source")
	}
}

func TestInstall_Gemini_DryRun_NoTomlCreated(t *testing.T) {
	ffs := fs.NewFakeFileSystem()
	ffs.Dirs["/project"] = true
	ffs.Dirs["/source"] = true
	ffs.Files["/source/.agents/skills/review/SKILL.md"] = []byte("---\nversion: 1.0.0\ndescription: Revisa codigo.\n---\n")

	svc := setupTestService(ffs)

	err := svc.Execute(config.InstallOptions{
		ProjectDir: "/project",
		SourceDir:  "/source",
		Tools:      []skills.Tool{skills.ToolGemini},
		LinkMode:   skills.LinkCopy,
		DryRun:     true,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	tomlFile := "/project/.gemini/commands/review.toml"
	if ffs.Exists(tomlFile) {
		t.Errorf("dry-run should not create %s", tomlFile)
	}
}

func TestInstall_Copilot_GeneratesAgents(t *testing.T) {
	ffs := fs.NewFakeFileSystem()
	ffs.Dirs["/project"] = true
	ffs.Dirs["/source"] = true
	ffs.Files["/source/.agents/skills/review/SKILL.md"] = []byte("---\nversion: 1.0.0\ndescription: Revisa codigo.\n---\n")

	svc := setupTestService(ffs)

	err := svc.Execute(config.InstallOptions{
		ProjectDir: "/project",
		SourceDir:  "/source",
		Tools:      []skills.Tool{skills.ToolCopilot},
		LinkMode:   skills.LinkCopy,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	agentFile := "/project/.github/agents/reviewer.agent.md"
	if !ffs.Exists(agentFile) {
		t.Errorf("expected %s to exist after copilot install", agentFile)
	}

	data, err := ffs.ReadFile(agentFile)
	if err != nil {
		t.Fatalf("could not read reviewer.agent.md: %v", err)
	}
	if !strings.Contains(string(data), ".agent.md") && !strings.HasSuffix(agentFile, ".agent.md") {
		t.Error("reviewer.agent.md should have .agent.md suffix")
	}
	if !strings.Contains(string(data), "review") {
		t.Errorf("reviewer.agent.md should reference skill 'review', got: %s", string(data))
	}
}

func TestInstall_Copilot_DryRun_NoAgentsCreated(t *testing.T) {
	ffs := fs.NewFakeFileSystem()
	ffs.Dirs["/project"] = true
	ffs.Dirs["/source"] = true
	ffs.Files["/source/.agents/skills/review/SKILL.md"] = []byte("---\nversion: 1.0.0\ndescription: Revisa codigo.\n---\n")

	svc := setupTestService(ffs)

	err := svc.Execute(config.InstallOptions{
		ProjectDir: "/project",
		SourceDir:  "/source",
		Tools:      []skills.Tool{skills.ToolCopilot},
		LinkMode:   skills.LinkCopy,
		DryRun:     true,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	agentFile := "/project/.github/agents/reviewer.agent.md"
	if ffs.Exists(agentFile) {
		t.Errorf("dry-run should not create %s", agentFile)
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

// setupOSTestService cria um Service que usa o filesystem real (necessario para testes com embutidos).
func setupOSTestService() *Service {
	printer := output.New(false)
	fsys := fs.NewOSFileSystem()
	mfst := manifest.NewStore(fsys)
	adpt := adapters.NewGenerator(fsys, printer)
	ctxg := contextgen.NewGenerator(fsys, printer)
	return NewService(fsys, printer, mfst, adpt, ctxg)
}

func TestInstall_EmbeddedSource_NoSourceFlag(t *testing.T) {
	projectDir := t.TempDir()
	svc := setupOSTestService()

	err := svc.Execute(config.InstallOptions{
		ProjectDir: projectDir,
		SourceDir:  "", // sem --source: usa embutido
		Tools:      []skills.Tool{skills.ToolClaude},
		Langs:      nil,
		LinkMode:   skills.LinkCopy,
	})
	if err != nil {
		t.Fatalf("install sem --source falhou: %v", err)
	}

	// Skills canonicas devem ter sido instaladas
	for _, skill := range []string{"agent-governance", "bugfix", "review", "execute-task"} {
		path := projectDir + "/.agents/skills/" + skill + "/SKILL.md"
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("skill embutida %s nao instalada", skill)
		}
	}

	// Settings Claude devem ter sido criados
	if _, err := os.Stat(projectDir + "/.claude/settings.local.json"); os.IsNotExist(err) {
		t.Error(".claude/settings.local.json nao criado")
	}
}

func TestInstall_EmbeddedSource_AllToolsAllLangs(t *testing.T) {
	projectDir := t.TempDir()
	svc := setupOSTestService()

	err := svc.Execute(config.InstallOptions{
		ProjectDir: projectDir,
		SourceDir:  "", // usa embutido
		Tools:      skills.AllTools,
		Langs:      skills.AllLangs,
		LinkMode:   skills.LinkCopy,
	})
	if err != nil {
		t.Fatalf("install --tools all --langs all sem --source falhou: %v", err)
	}

	// Verificar skills de linguagem
	for _, skill := range []string{"go-implementation", "node-implementation", "python-implementation", "object-calisthenics-go"} {
		path := projectDir + "/.agents/skills/" + skill + "/SKILL.md"
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("skill de linguagem embutida %s nao instalada", skill)
		}
	}

	// Manifesto deve ter sido criado
	if _, err := os.Stat(projectDir + "/.ai_spec_harness.json"); os.IsNotExist(err) {
		t.Error("manifesto nao criado apos install com embutido")
	}
}

// TestInstall_RefResolved_UsesCopyMode validates that install works correctly when
// invoked with the parameters produced by --ref: a resolved temp dir as SourceDir
// and LinkMode=copy (symlinks are not allowed since the tempdir will be removed).
func TestInstall_RefResolved_UsesCopyMode(t *testing.T) {
	sourceDir := t.TempDir()
	projectDir := t.TempDir()

	// Simulate a source directory as gitref.Resolve would produce
	skillDir := sourceDir + "/.agents/skills/review"
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(skillDir+"/SKILL.md", []byte("---\nversion: 1.0.0\ndescription: Review.\n---\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	svc := setupOSTestService()

	err := svc.Execute(config.InstallOptions{
		ProjectDir: projectDir,
		SourceDir:  sourceDir,
		Tools:      []skills.Tool{skills.ToolClaude},
		LinkMode:   skills.LinkCopy, // --ref always forces copy
	})
	if err != nil {
		t.Fatalf("install with ref-resolved source failed: %v", err)
	}

	path := projectDir + "/.agents/skills/review/SKILL.md"
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Error("skill from ref-resolved source was not installed")
	}
}

func TestInstall_ExternalSource_Override(t *testing.T) {
	projectDir := t.TempDir()
	sourceDir := t.TempDir()

	// Criar fonte externa com uma skill customizada
	skillDir := sourceDir + "/.agents/skills/review"
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatal(err)
	}
	customContent := []byte("---\nversion: 99.0.0\ndescription: Custom Review.\n---\n# Custom Review\n")
	if err := os.WriteFile(skillDir+"/SKILL.md", customContent, 0o644); err != nil {
		t.Fatal(err)
	}

	svc := setupOSTestService()

	err := svc.Execute(config.InstallOptions{
		ProjectDir: projectDir,
		SourceDir:  sourceDir, // usa fonte externa
		Tools:      []skills.Tool{skills.ToolClaude},
		Langs:      nil,
		LinkMode:   skills.LinkCopy,
	})
	if err != nil {
		t.Fatalf("install com --source externo falhou: %v", err)
	}

	// A skill instalada deve ser a da fonte externa (versao 99.0.0)
	data, err := os.ReadFile(projectDir + "/.agents/skills/review/SKILL.md")
	if err != nil {
		t.Fatalf("skill review nao instalada: %v", err)
	}
	if !strings.Contains(string(data), "99.0.0") {
		t.Error("fonte externa nao teve precedencia sobre embutida")
	}
}
