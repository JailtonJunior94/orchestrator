package skills

import (
	"strings"
	"testing"
)

// TestParseFrontmatterFields_FlatFields verifica que campos de nível raiz são mapeados corretamente.
func TestParseFrontmatterFields_FlatFields(t *testing.T) {
	content := []byte(`---
name: claude-revisor-rigoroso
description: Revisor de PR com viés conservador
version: 1.0.0
---

Corpo do agente.
`)
	fields := ParseFrontmatterFields(content)

	cases := map[string]string{
		"name":        "claude-revisor-rigoroso",
		"description": "Revisor de PR com viés conservador",
		"version":     "1.0.0",
	}
	for k, want := range cases {
		if got := fields[k]; got != want {
			t.Errorf("fields[%q] = %q, want %q", k, got, want)
		}
	}
}

// TestParseFrontmatterFields_NestedBlock verifica que campos indentados são mapeados via dot-notation.
func TestParseFrontmatterFields_NestedBlock(t *testing.T) {
	content := []byte(`---
name: claude-revisor-rigoroso
description: Revisor de PR
version: 1.0.0
runtime:
  ide: claude
  model: claude-opus-4-7
  reasoning_effort: high
  access_mode: bypass-permissions
---
`)
	fields := ParseFrontmatterFields(content)

	nested := map[string]string{
		"runtime.ide":              "claude",
		"runtime.model":            "claude-opus-4-7",
		"runtime.reasoning_effort": "high",
		"runtime.access_mode":      "bypass-permissions",
	}
	for k, want := range nested {
		if got := fields[k]; got != want {
			t.Errorf("fields[%q] = %q, want %q", k, got, want)
		}
	}
	// Campos raiz ainda devem estar presentes.
	if got := fields["name"]; got != "claude-revisor-rigoroso" {
		t.Errorf("fields[\"name\"] = %q, want %q", got, "claude-revisor-rigoroso")
	}
}

// TestParseFrontmatterFields_NoFrontmatter verifica que conteúdo sem frontmatter retorna mapa vazio.
func TestParseFrontmatterFields_NoFrontmatter(t *testing.T) {
	content := []byte("# Sem frontmatter\nApenas corpo.")
	fields := ParseFrontmatterFields(content)
	if len(fields) != 0 {
		t.Errorf("esperado mapa vazio, got %v", fields)
	}
}

// TestParseFrontmatterFields_EmptyFrontmatter verifica que frontmatter vazio retorna mapa vazio.
func TestParseFrontmatterFields_EmptyFrontmatter(t *testing.T) {
	content := []byte("---\n---\n# Corpo\n")
	fields := ParseFrontmatterFields(content)
	if len(fields) != 0 {
		t.Errorf("esperado mapa vazio, got %v", fields)
	}
}

// TestParseFrontmatterFields_SkillsCompatibility verifica que ParseFrontmatterFields
// preserva comportamento para campos de skills (T-22: zero regressão).
func TestParseFrontmatterFields_SkillsCompatibility(t *testing.T) {
	content := []byte(`---
name: execute-task
version: 1.2.3
description: Executa tarefas.
depends_on: [review, bugfix]
lang: go
link_mode: hard
max_depth: 3
---
`)
	fields := ParseFrontmatterFields(content)

	expected := map[string]string{
		"name":        "execute-task",
		"version":     "1.2.3",
		"description": "Executa tarefas.",
		"depends_on":  "[review, bugfix]",
		"lang":        "go",
		"link_mode":   "hard",
		"max_depth":   "3",
	}
	for k, want := range expected {
		if got := fields[k]; got != want {
			t.Errorf("fields[%q] = %q, want %q", k, got, want)
		}
	}
}

// TestParseFrontmatterFields_WrapperConsistency verifica que ParseFrontmatter (wrapper)
// produz o mesmo resultado que ParseFrontmatterFields para campos de skills.
func TestParseFrontmatterFields_WrapperConsistency(t *testing.T) {
	content := []byte(`---
name: my-skill
version: 2.0.0
description: Uma skill de teste.
triggers: [go-implementation]
depends_on: [review]
lang: go
link_mode: hard
max_depth: 5
---
`)
	fm := ParseFrontmatter(content)
	fields := ParseFrontmatterFields(content)

	if fm.Name != fields["name"] {
		t.Errorf("name divergente: wrapper=%q, fields=%q", fm.Name, fields["name"])
	}
	if fm.Version != fields["version"] {
		t.Errorf("version divergente: wrapper=%q, fields=%q", fm.Version, fields["version"])
	}
	if fm.Description != fields["description"] {
		t.Errorf("description divergente: wrapper=%q, fields=%q", fm.Description, fields["description"])
	}
	if fm.Lang != fields["lang"] {
		t.Errorf("lang divergente: wrapper=%q, fields=%q", fm.Lang, fields["lang"])
	}
	if fm.LinkMode != fields["link_mode"] {
		t.Errorf("link_mode divergente: wrapper=%q, fields=%q", fm.LinkMode, fields["link_mode"])
	}
}

func TestParseFrontmatter(t *testing.T) {
	content := []byte(`---
name: analyze-project
version: 1.2.3
description: Analisa e classifica projetos.
---

# Skill
`)
	fm := ParseFrontmatter(content)

	if fm.Version != "1.2.3" {
		t.Errorf("version: got %q, want %q", fm.Version, "1.2.3")
	}
	if fm.Name != "analyze-project" {
		t.Errorf("name: got %q, want %q", fm.Name, "analyze-project")
	}
	if fm.Description != "Analisa e classifica projetos." {
		t.Errorf("description: got %q, want %q", fm.Description, "Analisa e classifica projetos.")
	}
}

func TestParseFrontmatter_DependsOn(t *testing.T) {
	content := []byte(`---
name: execute-task
version: 1.2.3
depends_on: [review, bugfix]
description: Executa tarefas.
---
`)
	fm := ParseFrontmatter(content)

	if len(fm.DependsOn) != 2 {
		t.Fatalf("depends_on: got %#v", fm.DependsOn)
	}
	if fm.DependsOn[0] != "review" || fm.DependsOn[1] != "bugfix" {
		t.Fatalf("depends_on: got %#v", fm.DependsOn)
	}
}

func TestParseFrontmatter_Empty(t *testing.T) {
	fm := ParseFrontmatter([]byte("# Sem frontmatter"))
	if fm.Version != "" {
		t.Errorf("version: got %q, want empty", fm.Version)
	}
}

func TestValidateFrontmatter_MissingBlock(t *testing.T) {
	content := []byte("# Sem frontmatter\nAlgum conteudo.")
	err := ValidateFrontmatter(content, "", nil)
	if err == nil {
		t.Fatal("expected error for missing frontmatter block")
	}
	if !strings.Contains(err.Error(), "frontmatter") {
		t.Fatalf("expected error containing 'frontmatter', got: %v", err)
	}
}

func TestValidateFrontmatter_MissingDescription(t *testing.T) {
	content := []byte("---\nname: my-skill\nversion: 1.0.0\n---\n")
	err := ValidateFrontmatter(content, "", nil)
	if err == nil {
		t.Fatal("expected error for missing description")
	}
	if !strings.Contains(err.Error(), "description") {
		t.Fatalf("expected error containing 'description', got: %v", err)
	}
}

func TestValidateFrontmatter_InvalidSemver(t *testing.T) {
	content := []byte("---\nname: my-skill\nversion: not-semver\ndescription: A skill.\n---\n")
	err := ValidateFrontmatter(content, "", nil)
	if err == nil {
		t.Fatal("expected error for invalid semver")
	}
	if !strings.Contains(err.Error(), "version") && !strings.Contains(err.Error(), "semver") {
		t.Fatalf("expected error containing 'version' or 'semver', got: %v", err)
	}
}

func TestValidateFrontmatter_NameMismatch(t *testing.T) {
	content := []byte("---\nname: wrong-name\nversion: 1.0.0\ndescription: A skill.\n---\n")
	err := ValidateFrontmatter(content, "my-skill", nil)
	if err == nil {
		t.Fatal("expected error for name mismatch")
	}
	if !strings.Contains(err.Error(), "name") {
		t.Fatalf("expected error containing 'name', got: %v", err)
	}
}

func TestValidateFrontmatter_DependsOnMissing(t *testing.T) {
	content := []byte("---\nname: my-skill\nversion: 1.0.0\ndescription: A skill.\ndepends_on: [ghost-skill]\n---\n")
	err := ValidateFrontmatter(content, "my-skill", []string{"review", "bugfix"})
	if err == nil {
		t.Fatal("expected error for missing dependency")
	}
	if !strings.Contains(err.Error(), "depends_on") && !strings.Contains(err.Error(), "ghost-skill") {
		t.Fatalf("expected error containing 'depends_on' or 'ghost-skill', got: %v", err)
	}
}

func TestValidateFrontmatter_ValidSkill(t *testing.T) {
	content := []byte("---\nname: my-skill\nversion: 1.0.0\ndescription: Uma skill valida.\n---\n# My Skill\n")
	err := ValidateFrontmatter(content, "my-skill", nil)
	if err != nil {
		t.Fatalf("unexpected error for valid frontmatter: %v", err)
	}
}

func TestSemverGreater(t *testing.T) {
	tests := []struct {
		a, b string
		want bool
	}{
		{"1.0.0", "0.9.0", true},
		{"1.1.0", "1.0.0", true},
		{"1.0.1", "1.0.0", true},
		{"1.0.0", "1.0.0", false},
		{"0.9.0", "1.0.0", false},
		{"2.0.0-beta", "1.9.9", true},
		{"v1.1.0", "1.0.0", true},
	}

	for _, tt := range tests {
		got := SemverGreater(tt.a, tt.b)
		if got != tt.want {
			t.Errorf("SemverGreater(%q, %q) = %v, want %v", tt.a, tt.b, got, tt.want)
		}
	}
}

func TestIsValidSemver(t *testing.T) {
	tests := []struct {
		name    string
		version string
		want    bool
	}{
		{name: "major minor patch", version: "1.2.3", want: true},
		{name: "prefixed prerelease", version: "v1.2.3-beta.1", want: true},
		{name: "major only", version: "1", want: false},
		{name: "major minor", version: "1.2", want: false},
		{name: "empty prerelease", version: "1.2.3-", want: false},
		{name: "empty prerelease identifier", version: "1.2.3-alpha..1", want: false},
		{name: "dev", version: "dev", want: false},
		{name: "missing segment", version: "1..3", want: false},
		{name: "non numeric", version: "1.2.x", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsValidSemver(tt.version); got != tt.want {
				t.Fatalf("IsValidSemver(%q) = %v, want %v", tt.version, got, tt.want)
			}
		})
	}
}
