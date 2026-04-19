package lint

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/JailtonJunior94/ai-spec-harness/internal/contextgen"
)

// setupProject cria um diretório temporário com a estrutura mínima de projeto.
func setupProject(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	return dir
}

// writeFile escreve conteúdo em um arquivo, criando diretórios intermediários.
func writeFile(t *testing.T, dir, rel, content string) {
	t.Helper()
	path := filepath.Join(dir, rel)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll %s: %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile %s: %v", path, err)
	}
}

func validSkillFrontmatter(name string) string {
	return "---\nname: " + name + "\nversion: 1.0.0\ndescription: Skill de teste\n---\n\n# Conteúdo\n"
}

func TestLint_Clean(t *testing.T) {
	dir := setupProject(t)

	writeFile(t, dir, "AGENTS.md", "<!-- governance-schema: "+contextgen.GovernanceSchemaVersion+" -->\n# Regras\n")
	writeFile(t, dir, ".agents/skills/agent-governance/references/bug-schema.json", `{"type":"object"}`)
	writeFile(t, dir, ".agents/skills/agent-governance/SKILL.md", validSkillFrontmatter("agent-governance"))

	svc := NewService()
	errs, err := svc.Execute(dir)
	if err != nil {
		t.Fatalf("Execute retornou erro inesperado: %v", err)
	}
	if len(errs) != 0 {
		t.Errorf("esperava 0 erros, obteve %d: %v", len(errs), errs)
	}
}

func TestLint_PlaceholderInAGENTSMD(t *testing.T) {
	dir := setupProject(t)

	writeFile(t, dir, "AGENTS.md", "<!-- governance-schema: "+contextgen.GovernanceSchemaVersion+" -->\n# Regras\n{{ TOOLCHAIN_COMMANDS }}\n")

	svc := NewService()
	errs, err := svc.Execute(dir)
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
	dir := setupProject(t)

	writeFile(t, dir, "AGENTS.md", "<!-- governance-schema: 0.0.0 -->\n# Regras\n")

	svc := NewService()
	errs, err := svc.Execute(dir)
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
	dir := setupProject(t)

	writeFile(t, dir, ".agents/skills/agent-governance/references/bug-schema.json", `{ invalid json }`)

	svc := NewService()
	errs, err := svc.Execute(dir)
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
	dir := setupProject(t)

	// SKILL.md sem description
	writeFile(t, dir, ".agents/skills/my-skill/SKILL.md", "---\nname: my-skill\nversion: 1.0.0\n---\n\n# Skill\n")

	svc := NewService()
	errs, err := svc.Execute(dir)
	if err != nil {
		t.Fatalf("Execute retornou erro inesperado: %v", err)
	}
	if len(errs) == 0 {
		t.Fatal("esperava erro de frontmatter inválido, obteve 0")
	}

	found := false
	for _, e := range errs {
		if filepath.Base(filepath.Dir(e.File)) == "my-skill" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("esperava erro referenciando my-skill/SKILL.md, erros: %v", errs)
	}
}

func TestLint_MultipleErrors(t *testing.T) {
	dir := setupProject(t)

	// AGENTS.md com placeholder e schema errado
	writeFile(t, dir, "AGENTS.md", "<!-- governance-schema: 0.0.0 -->\n# Regras\n{{ PLACEHOLDER }}\n")

	svc := NewService()
	errs, err := svc.Execute(dir)
	if err != nil {
		t.Fatalf("Execute retornou erro inesperado: %v", err)
	}
	if len(errs) < 2 {
		t.Errorf("esperava pelo menos 2 erros, obteve %d: %v", len(errs), errs)
	}
}
