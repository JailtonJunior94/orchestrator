package lint

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	"github.com/JailtonJunior94/ai-spec-harness/internal/contextgen"
	"github.com/JailtonJunior94/ai-spec-harness/internal/fs"
	"github.com/JailtonJunior94/ai-spec-harness/internal/skillscheck"
)

const _projectDir = "/proj"

func newFakeService() (*Service, *fs.FakeFileSystem) {
	fake := fs.NewFakeFileSystem()
	return NewService(fake), fake
}

func writeFile(t *testing.T, fake *fs.FakeFileSystem, rel, content string) {
	t.Helper()
	if err := fake.WriteFile(filepath.Join(_projectDir, rel), []byte(content)); err != nil {
		t.Fatalf("WriteFile %s: %v", rel, err)
	}
}

func writeLock(t *testing.T, fake *fs.FakeFileSystem, entries map[string]skillscheck.LockEntry) {
	t.Helper()
	lock := skillscheck.LockFile{Version: 1, Skills: entries}
	data, err := json.Marshal(lock)
	if err != nil {
		t.Fatalf("Marshal skills-lock.json: %v", err)
	}
	writeFile(t, fake, "skills-lock.json", string(data))
}

func validSkillFrontmatter(name string) string {
	return "---\nname: " + name + "\nversion: 1.0.0\ndescription: Skill de teste\n---\n\n# Conteúdo\n"
}

func skillFrontmatter(fields ...string) string {
	return "---\n" + strings.Join(fields, "\n") + "\n---\n\n# Skill\n"
}

func hasSkillError(errs []LintError, skillName string) bool {
	for _, e := range errs {
		if filepath.Base(filepath.Dir(e.File)) == skillName {
			return true
		}
	}
	return false
}

func TestLint_Clean(t *testing.T) {
	svc, fake := newFakeService()

	writeFile(t, fake, "AGENTS.md", "<!-- governance-schema: "+contextgen.GovernanceSchemaVersion+" -->\n# Regras\n")
	writeFile(t, fake, ".agents/skills/agent-governance/references/bug-schema.json", `{"type":"object"}`)
	writeFile(t, fake, ".agents/skills/agent-governance/SKILL.md", validSkillFrontmatter("agent-governance"))

	errs, err := svc.Execute(_projectDir)
	if err != nil {
		t.Fatalf("Execute retornou erro inesperado: %v", err)
	}
	if len(errs) != 0 {
		t.Errorf("esperava 0 erros, obteve %d: %v", len(errs), errs)
	}
}

func TestLint_PlaceholderInAGENTSMD(t *testing.T) {
	svc, fake := newFakeService()

	writeFile(t, fake, "AGENTS.md", "<!-- governance-schema: "+contextgen.GovernanceSchemaVersion+" -->\n# Regras\n{{ TOOLCHAIN_COMMANDS }}\n")

	errs, err := svc.Execute(_projectDir)
	if err != nil {
		t.Fatalf("Execute retornou erro inesperado: %v", err)
	}
	if len(errs) == 0 {
		t.Fatal("esperava pelo menos 1 erro de placeholder, obteve 0")
	}

	found := false
	for _, e := range errs {
		if e.File == "AGENTS.md" && e.Line == 3 {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("esperava erro em AGENTS.md linha 3, erros: %v", errs)
	}
}

func TestLint_SchemaVersionMismatch(t *testing.T) {
	svc, fake := newFakeService()

	writeFile(t, fake, "AGENTS.md", "<!-- governance-schema: 0.0.0 -->\n# Regras\n")

	errs, err := svc.Execute(_projectDir)
	if err != nil {
		t.Fatalf("Execute retornou erro inesperado: %v", err)
	}
	if len(errs) == 0 {
		t.Fatal("esperava erro de versão de schema, obteve 0")
	}

	found := false
	for _, e := range errs {
		if e.File == "AGENTS.md" && e.Line == 0 {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("esperava erro de versao em AGENTS.md, erros: %v", errs)
	}
}

func TestLint_InvalidBugSchema(t *testing.T) {
	svc, fake := newFakeService()

	writeFile(t, fake, ".agents/skills/agent-governance/references/bug-schema.json", `{ invalid json }`)

	errs, err := svc.Execute(_projectDir)
	if err != nil {
		t.Fatalf("Execute retornou erro inesperado: %v", err)
	}
	if len(errs) == 0 {
		t.Fatal("esperava erro de JSON inválido, obteve 0")
	}

	found := false
	for _, e := range errs {
		if filepath.Base(e.File) == "bug-schema.json" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("esperava erro referenciando bug-schema.json, erros: %v", errs)
	}
}

func TestLint_InvalidSkillFrontmatter(t *testing.T) {
	svc, fake := newFakeService()

	writeFile(t, fake, ".agents/skills/my-skill/SKILL.md", skillFrontmatter("name: my-skill", "version: 1.0.0"))

	errs, err := svc.Execute(_projectDir)
	if err != nil {
		t.Fatalf("Execute retornou erro inesperado: %v", err)
	}
	if len(errs) == 0 {
		t.Fatal("esperava erro de frontmatter inválido, obteve 0")
	}
	if !hasSkillError(errs, "my-skill") {
		t.Errorf("esperava erro referenciando my-skill/SKILL.md, erros: %v", errs)
	}
}

func TestLint_MultipleErrors(t *testing.T) {
	svc, fake := newFakeService()

	writeFile(t, fake, "AGENTS.md", "<!-- governance-schema: 0.0.0 -->\n# Regras\n{{ PLACEHOLDER }}\n")

	errs, err := svc.Execute(_projectDir)
	if err != nil {
		t.Fatalf("Execute retornou erro inesperado: %v", err)
	}
	if len(errs) < 2 {
		t.Errorf("esperava pelo menos 2 erros, obteve %d: %v", len(errs), errs)
	}
}

func TestLint_SkillVersionRequirementBySourceType(t *testing.T) {
	scenarios := []struct {
		name         string
		lockEntries  map[string]skillscheck.LockEntry
		skillContent string
		wantError    bool
	}{
		{
			name: "deve aceitar skill terceira sem version quando lock usa sourceType github",
			lockEntries: map[string]skillscheck.LockEntry{
				"external-skill": {SourceType: "github", Source: "owner/repo"},
			},
			skillContent: skillFrontmatter("name: external-skill", "description: Skill externa"),
			wantError:    false,
		},
		{
			name:         "deve rejeitar skill primeira sem version",
			skillContent: skillFrontmatter("name: external-skill", "description: Skill interna"),
			wantError:    true,
		},
		{
			name: "deve rejeitar skill terceira sem name",
			lockEntries: map[string]skillscheck.LockEntry{
				"external-skill": {SourceType: "github", Source: "owner/repo"},
			},
			skillContent: skillFrontmatter("description: Skill externa"),
			wantError:    true,
		},
		{
			name: "deve rejeitar skill terceira sem description",
			lockEntries: map[string]skillscheck.LockEntry{
				"external-skill": {SourceType: "github", Source: "owner/repo"},
			},
			skillContent: skillFrontmatter("name: external-skill"),
			wantError:    true,
		},
		{
			name:         "deve preservar comportamento estrito sem skills-lock",
			lockEntries:  nil,
			skillContent: skillFrontmatter("name: external-skill", "description: Skill sem lock"),
			wantError:    true,
		},
	}

	for _, scenario := range scenarios {
		t.Run(scenario.name, func(t *testing.T) {
			svc, fake := newFakeService()
			if scenario.lockEntries != nil {
				writeLock(t, fake, scenario.lockEntries)
			}
			writeFile(t, fake, ".agents/skills/external-skill/SKILL.md", scenario.skillContent)

			errs, err := svc.Execute(_projectDir)
			if err != nil {
				t.Fatalf("Execute retornou erro inesperado: %v", err)
			}
			gotError := hasSkillError(errs, "external-skill")
			if gotError != scenario.wantError {
				t.Fatalf("erro esperado=%v, obtido=%v: %v", scenario.wantError, gotError, errs)
			}
		})
	}
}
