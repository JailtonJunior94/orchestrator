package install

import (
	"testing"

	"github.com/JailtonJunior94/ai-spec-harness/internal/config"
	"github.com/JailtonJunior94/ai-spec-harness/internal/fs"
	"github.com/JailtonJunior94/ai-spec-harness/internal/skills"
)

func TestVerify_NoLangsDetected_DoesNotReportLangSkillsAsMissing(t *testing.T) {
	t.Parallel()

	ffs := fs.NewFakeFileSystem()
	skillContent := []byte("---\nversion: 1.0.0\n---")

	for _, sk := range skills.NewCatalog().AllSkills([]skills.Lang{skills.LangGo, skills.LangNode, skills.LangPython, skills.LangDotNet}) {
		ffs.Files["/source/.agents/skills/"+sk+"/SKILL.md"] = skillContent
		ffs.Dirs["/source/.agents/skills/"+sk] = true
	}
	for _, sk := range skills.NewCatalog().AllSkills(nil) {
		ffs.Files["/project/.agents/skills/"+sk+"/SKILL.md"] = skillContent
		ffs.Dirs["/project/.agents/skills/"+sk] = true
	}
	ffs.Dirs["/project"] = true
	ffs.Files["/project/.ai_spec_harness.json"] = []byte(`{
  "version": "test",
  "tools": ["claude"],
  "langs": null,
  "skills": []
}`)

	svc := setupTestService(ffs)
	items, err := svc.Verify(config.InstallOptions{
		ProjectDir: "/project",
		SourceDir:  "/source",
		Tools:      []skills.Tool{skills.ToolClaude},
	})
	if err != nil {
		t.Fatalf("Verify falhou: %v", err)
	}

	for _, item := range items {
		if langImplementationSkills[item.Skill] {
			t.Errorf("skill de linguagem %q nao deve ser verificada quando nenhuma linguagem foi detectada (estado reportado: %s)", item.Skill, item.State)
		}
	}
}

func TestVerify_ManifestLangsSelectOnlyMatchingLangSkills(t *testing.T) {
	t.Parallel()

	ffs := fs.NewFakeFileSystem()
	skillContent := []byte("---\nversion: 1.0.0\n---")

	for _, sk := range skills.NewCatalog().AllSkills([]skills.Lang{skills.LangGo, skills.LangNode, skills.LangPython, skills.LangDotNet}) {
		ffs.Files["/source/.agents/skills/"+sk+"/SKILL.md"] = skillContent
		ffs.Dirs["/source/.agents/skills/"+sk] = true
	}
	for _, sk := range skills.NewCatalog().AllSkills([]skills.Lang{skills.LangGo}) {
		ffs.Files["/project/.agents/skills/"+sk+"/SKILL.md"] = skillContent
		ffs.Dirs["/project/.agents/skills/"+sk] = true
	}
	ffs.Dirs["/project"] = true
	ffs.Files["/project/.ai_spec_harness.json"] = []byte(`{
  "version": "test",
  "tools": ["claude"],
  "langs": ["go"],
  "skills": []
}`)

	svc := setupTestService(ffs)
	items, err := svc.Verify(config.InstallOptions{
		ProjectDir: "/project",
		SourceDir:  "/source",
		Tools:      []skills.Tool{skills.ToolClaude},
	})
	if err != nil {
		t.Fatalf("Verify falhou: %v", err)
	}

	seen := make(map[string]VerifyState)
	for _, item := range items {
		if langImplementationSkills[item.Skill] {
			seen[item.Skill] = item.State
		}
	}
	for _, want := range []string{"go-implementation", "object-calisthenics-go"} {
		if _, ok := seen[want]; !ok {
			t.Errorf("skill de go %q deve ser verificada quando o manifesto declara langs=[go]", want)
		}
	}
	for _, unwanted := range []string{"node-implementation", "python-implementation", "dotnet-csharp-implementation"} {
		if state, ok := seen[unwanted]; ok {
			t.Errorf("skill %q nao deve ser verificada com langs=[go] (estado: %s)", unwanted, state)
		}
	}
}
