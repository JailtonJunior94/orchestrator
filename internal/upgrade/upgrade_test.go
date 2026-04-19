package upgrade

import (
	"strings"
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

func TestUpgrade_RefsChangedFiles(t *testing.T) {
	ffs := fs.NewFakeFileSystem()
	printer := output.New(false)
	svc := NewService(ffs, printer, manifest.NewStore(ffs), adapters.NewGenerator(ffs, printer), contextgen.NewGenerator(ffs, printer))

	ffs.Files["/source/references/a.md"] = []byte("new")
	ffs.Files["/source/references/b.md"] = []byte("changed")
	ffs.Files["/project/references/b.md"] = []byte("old")
	ffs.Files["/project/references/c.md"] = []byte("removed")

	changed := svc.refsChangedFiles("/source/references", "/project/references")
	got := strings.Join(changed, "\n")

	for _, want := range []string{"+ a.md (novo)", "~ b.md (modificado)", "- c.md (removido)"} {
		if !strings.Contains(got, want) {
			t.Fatalf("changed refs missing %q in %q", want, got)
		}
	}
}

func TestUpgrade_RegeneratesGovernanceOnSchemaDivergence(t *testing.T) {
	ffs := fs.NewFakeFileSystem()
	ffs.Files["/source/.agents/skills/review/SKILL.md"] = []byte("---\nname: review\nversion: 1.0.0\ndescription: Review.\n---\n")
	ffs.Files["/project/.agents/skills/review/SKILL.md"] = []byte("---\nname: review\nversion: 1.0.0\ndescription: Review.\n---\n")
	ffs.Files["/source/.agents/skills/analyze-project/assets/agents-template.md"] = []byte("<!-- governance-schema: 1.0.0 -->")
	ffs.Files["/project/AGENTS.md"] = []byte("<!-- governance-schema: 0.9.0 -->\n# old")

	svc := setupTestService(ffs)
	err := svc.Execute(config.UpgradeOptions{
		ProjectDir: "/project",
		SourceDir:  "/source",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, err := ffs.ReadFile("/project/AGENTS.md")
	if err != nil {
		t.Fatalf("AGENTS.md not regenerated: %v", err)
	}
	if !strings.Contains(string(data), "governance-schema: 1.0.0") {
		t.Fatalf("expected regenerated schema, got %q", string(data))
	}
}

func TestUpgrade_AdaptersRegeneratedAfterSkillChange(t *testing.T) {
	ffs := fs.NewFakeFileSystem()
	ffs.Files["/source/.agents/skills/review/SKILL.md"] = []byte("---\nname: review\nversion: 2.0.0\ndescription: Revisa codigo.\n---\n")
	ffs.Files["/project/.agents/skills/review/SKILL.md"] = []byte("---\nname: review\nversion: 1.0.0\ndescription: Revisa codigo.\n---\n")
	ffs.Dirs["/project/.claude"] = true

	svc := setupTestService(ffs)
	err := svc.Execute(config.UpgradeOptions{
		ProjectDir: "/project",
		SourceDir:  "/source",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !ffs.Exists("/project/.claude/agents/reviewer.md") {
		t.Error("expected reviewer.md to be regenerated after skill update")
	}
}

func TestUpgrade_CodexRegeneratedAfterSkillChange(t *testing.T) {
	ffs := fs.NewFakeFileSystem()
	ffs.Files["/source/.agents/skills/review/SKILL.md"] = []byte("---\nname: review\nversion: 2.0.0\ndescription: Review.\n---\n")
	ffs.Files["/project/.agents/skills/review/SKILL.md"] = []byte("---\nname: review\nversion: 1.0.0\ndescription: Review.\n---\n")
	ffs.Files["/project/.codex/config.toml"] = []byte("stale content")

	svc := setupTestService(ffs)
	err := svc.Execute(config.UpgradeOptions{
		ProjectDir: "/project",
		SourceDir:  "/source",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, err := ffs.ReadFile("/project/.codex/config.toml")
	if err != nil {
		t.Fatalf("codex config should still exist: %v", err)
	}
	if strings.Contains(string(data), "stale content") {
		t.Error("codex config should have been regenerated, still contains stale content")
	}
	if !strings.Contains(string(data), ".agents/skills/agent-governance") {
		t.Errorf("codex config missing expected skills, got: %q", string(data))
	}
}

func TestUpgrade_PreservesCustomClaudeSettings(t *testing.T) {
	ffs := fs.NewFakeFileSystem()
	ffs.Files["/source/.agents/skills/review/SKILL.md"] = []byte("---\nname: review\nversion: 2.0.0\ndescription: Review.\n---\n")
	ffs.Files["/project/.agents/skills/review/SKILL.md"] = []byte("---\nname: review\nversion: 1.0.0\ndescription: Review.\n---\n")
	ffs.Dirs["/project/.claude"] = true

	customSettings := `{"hooks": {"PreToolUse": []}, "customOption": "value"}`
	ffs.Files["/project/.claude/settings.local.json"] = []byte(customSettings)

	svc := setupTestService(ffs)
	err := svc.Execute(config.UpgradeOptions{
		ProjectDir: "/project",
		SourceDir:  "/source",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, err := ffs.ReadFile("/project/.claude/settings.local.json")
	if err != nil {
		t.Fatalf("settings.local.json should be preserved: %v", err)
	}
	if string(data) != customSettings {
		t.Fatalf("settings.local.json was modified by upgrade, got: %q", string(data))
	}
}

func TestUpgrade_CrossVersionSchemaChange(t *testing.T) {
	ffs := fs.NewFakeFileSystem()
	ffs.Files["/source/.agents/skills/review/SKILL.md"] = []byte("---\nname: review\nversion: 1.0.0\ndescription: Review.\n---\n")
	ffs.Files["/project/.agents/skills/review/SKILL.md"] = []byte("---\nname: review\nversion: 1.0.0\ndescription: Review.\n---\n")
	ffs.Files["/source/.agents/skills/analyze-project/assets/agents-template.md"] = []byte("<!-- governance-schema: 1.0.0 -->")
	ffs.Files["/project/AGENTS.md"] = []byte("<!-- governance-schema: 0.1.0 -->\n# muito antigo")

	svc := setupTestService(ffs)
	err := svc.Execute(config.UpgradeOptions{
		ProjectDir: "/project",
		SourceDir:  "/source",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, err := ffs.ReadFile("/project/AGENTS.md")
	if err != nil {
		t.Fatalf("AGENTS.md not found after upgrade: %v", err)
	}
	if !strings.Contains(string(data), "governance-schema: 1.0.0") {
		t.Fatalf("expected governance-schema: 1.0.0 after upgrade, got: %q", string(data))
	}
	if strings.Contains(string(data), "muito antigo") {
		t.Fatal("old AGENTS.md content should have been replaced by regeneration")
	}
}

func TestUpgrade_PreservesAgentsLocal(t *testing.T) {
	ffs := fs.NewFakeFileSystem()
	ffs.Files["/source/.agents/skills/review/SKILL.md"] = []byte("---\nname: review\nversion: 1.0.0\ndescription: Review.\n---\n")
	ffs.Files["/project/.agents/skills/review/SKILL.md"] = []byte("---\nname: review\nversion: 1.0.0\ndescription: Review.\n---\n")
	ffs.Files["/source/.agents/skills/analyze-project/assets/agents-template.md"] = []byte("<!-- governance-schema: 1.0.0 -->")
	ffs.Files["/project/AGENTS.md"] = []byte("<!-- governance-schema: 0.9.0 -->\n# content")

	const localContent = "# Custom user section\n\nPersonal rules here.\n"
	ffs.Files["/project/AGENTS.local.md"] = []byte(localContent)

	svc := setupTestService(ffs)
	err := svc.Execute(config.UpgradeOptions{
		ProjectDir: "/project",
		SourceDir:  "/source",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, err := ffs.ReadFile("/project/AGENTS.local.md")
	if err != nil {
		t.Fatalf("AGENTS.local.md should be preserved: %v", err)
	}
	if string(data) != localContent {
		t.Fatalf("AGENTS.local.md was modified: got %q", string(data))
	}

	agentsData, err := ffs.ReadFile("/project/AGENTS.md")
	if err != nil {
		t.Fatalf("AGENTS.md not found after upgrade: %v", err)
	}
	if !strings.Contains(string(agentsData), "governance-schema: 1.0.0") {
		t.Fatalf("AGENTS.md should have been regenerated, got: %q", string(agentsData))
	}
}

func TestUpgrade_RegeneratesAdaptersAndSupportFiles(t *testing.T) {
	ffs := fs.NewFakeFileSystem()
	ffs.Files["/source/.agents/skills/review/SKILL.md"] = []byte("---\nname: review\nversion: 2.0.0\ndescription: Review.\n---\n")
	ffs.Files["/project/.agents/skills/review/SKILL.md"] = []byte("---\nname: review\nversion: 1.0.0\ndescription: Review.\n---\n")
	ffs.Files["/source/.claude/rules/governance.md"] = []byte("# new governance")
	ffs.Files["/source/.claude/scripts/validate-task-evidence.sh"] = []byte("#!/usr/bin/env bash\n# new")
	ffs.Files["/project/.claude/rules/governance.md"] = []byte("# old governance")
	ffs.Files["/project/.claude/scripts/validate-task-evidence.sh"] = []byte("#!/usr/bin/env bash\n# old")
	ffs.Files["/project/.codex/config.toml"] = []byte("stale")
	ffs.Dirs["/project/.claude"] = true

	svc := setupTestService(ffs)
	err := svc.Execute(config.UpgradeOptions{
		ProjectDir: "/project",
		SourceDir:  "/source",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	rules, err := ffs.ReadFile("/project/.claude/rules/governance.md")
	if err != nil || string(rules) != "# new governance" {
		t.Fatalf("expected governance.md synced, got %q err=%v", string(rules), err)
	}

	script, err := ffs.ReadFile("/project/.claude/scripts/validate-task-evidence.sh")
	if err != nil || !strings.Contains(string(script), "# new") {
		t.Fatalf("expected validate-task-evidence synced, got %q err=%v", string(script), err)
	}

	codex, err := ffs.ReadFile("/project/.codex/config.toml")
	if err != nil {
		t.Fatalf("expected codex config regenerated: %v", err)
	}
	if !strings.Contains(string(codex), ".agents/skills/agent-governance") {
		t.Fatalf("expected codex config regenerated, got %q", string(codex))
	}
}
