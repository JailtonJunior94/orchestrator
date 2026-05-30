package skills

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/suite"
)

type FrontmatterSuite struct {
	suite.Suite
}

func TestFrontmatterSuite(t *testing.T) {
	suite.Run(t, new(FrontmatterSuite))
}

// TestParseFrontmatterFieldsFlatFields verifica que campos de nível raiz são mapeados corretamente.
func (s *FrontmatterSuite) TestParseFrontmatterFieldsFlatFields() {
	content := []byte(`---
name: claude-revisor-rigoroso
description: Revisor de PR com viés conservador
version: 1.0.0
---

Corpo do agente.
`)
	fields := NewCatalog().ParseFrontmatterFields(content)

	expected := map[string]string{
		"name":        "claude-revisor-rigoroso",
		"description": "Revisor de PR com viés conservador",
		"version":     "1.0.0",
	}
	for key, want := range expected {
		s.Equal(want, fields[key], "fields[%q]", key)
	}
}

// TestParseFrontmatterFieldsNestedBlock verifica que campos indentados são mapeados via dot-notation.
func (s *FrontmatterSuite) TestParseFrontmatterFieldsNestedBlock() {
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
	fields := NewCatalog().ParseFrontmatterFields(content)

	nested := map[string]string{
		"runtime.ide":              "claude",
		"runtime.model":            "claude-opus-4-7",
		"runtime.reasoning_effort": "high",
		"runtime.access_mode":      "bypass-permissions",
	}
	for key, want := range nested {
		s.Equal(want, fields[key], "fields[%q]", key)
	}
	// Campos raiz ainda devem estar presentes.
	s.Equal("claude-revisor-rigoroso", fields["name"], "fields[\"name\"]")
}

func (s *FrontmatterSuite) TestParseFrontmatterFieldsEmptyInputs() {
	scenarios := []struct {
		name    string
		content []byte
	}{
		{name: "conteudo sem frontmatter retorna mapa vazio", content: []byte("# Sem frontmatter\nApenas corpo.")},
		{name: "frontmatter vazio retorna mapa vazio", content: []byte("---\n---\n# Corpo\n")},
	}

	for _, scenario := range scenarios {
		s.Run(scenario.name, func() {
			fields := NewCatalog().ParseFrontmatterFields(scenario.content)
			s.Empty(fields, "esperado mapa vazio")
		})
	}
}

// TestParseFrontmatterFieldsSkillsCompatibility verifica que ParseFrontmatterFields
// preserva comportamento para campos de skills (T-22: zero regressão).
func (s *FrontmatterSuite) TestParseFrontmatterFieldsSkillsCompatibility() {
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
	fields := NewCatalog().ParseFrontmatterFields(content)

	expected := map[string]string{
		"name":        "execute-task",
		"version":     "1.2.3",
		"description": "Executa tarefas.",
		"depends_on":  "[review, bugfix]",
		"lang":        "go",
		"link_mode":   "hard",
		"max_depth":   "3",
	}
	for key, want := range expected {
		s.Equal(want, fields[key], "fields[%q]", key)
	}
}

// TestParseFrontmatterFieldsWrapperConsistency verifica que ParseFrontmatter (wrapper)
// produz o mesmo resultado que ParseFrontmatterFields para campos de skills.
func (s *FrontmatterSuite) TestParseFrontmatterFieldsWrapperConsistency() {
	content := []byte(`---
name: my-skill
version: 2.0.0
description: Uma skill de teste.
category: language
triggers: [go-implementation]
depends_on: [review]
lang: go
link_mode: hard
max_depth: 5
---
`)
	fm := NewCatalog().ParseFrontmatter(content)
	fields := NewCatalog().ParseFrontmatterFields(content)

	s.Equal(fields["name"], fm.Name, "name divergente")
	s.Equal(fields["version"], fm.Version, "version divergente")
	s.Equal(fields["description"], fm.Description, "description divergente")
	s.Equal(fields["category"], fm.Category, "category divergente")
	s.Equal(fields["lang"], fm.Lang, "lang divergente")
	s.Equal(fields["link_mode"], fm.LinkMode, "link_mode divergente")
}

func (s *FrontmatterSuite) TestParseFrontmatter() {
	content := []byte(`---
name: analyze-project
version: 1.2.3
description: Analisa e classifica projetos.
---

# Skill
`)
	fm := NewCatalog().ParseFrontmatter(content)

	s.Equal("1.2.3", fm.Version, "version")
	s.Equal("analyze-project", fm.Name, "name")
	s.Equal("Analisa e classifica projetos.", fm.Description, "description")
}

func (s *FrontmatterSuite) TestParseFrontmatterDependsOn() {
	content := []byte(`---
name: execute-task
version: 1.2.3
depends_on: [review, bugfix]
description: Executa tarefas.
---
`)
	fm := NewCatalog().ParseFrontmatter(content)

	s.Len(fm.DependsOn, 2, "depends_on")
	s.Equal("review", fm.DependsOn[0], "depends_on[0]")
	s.Equal("bugfix", fm.DependsOn[1], "depends_on[1]")
}

func (s *FrontmatterSuite) TestParseFrontmatterEmpty() {
	fm := NewCatalog().ParseFrontmatter([]byte("# Sem frontmatter"))
	s.Empty(fm.Version, "version")
}

func (s *FrontmatterSuite) TestValidateFrontmatterInvalid() {
	scenarios := []struct {
		name         string
		content      []byte
		skillName    string
		dependencies []string
		wantContains []string
	}{
		{
			name:         "deve rejeitar bloco ausente",
			content:      []byte("# Sem frontmatter\nAlgum conteudo."),
			wantContains: []string{"frontmatter"},
		},
		{
			name:         "deve rejeitar description ausente",
			content:      []byte("---\nname: my-skill\nversion: 1.0.0\n---\n"),
			wantContains: []string{"description"},
		},
		{
			name:         "deve rejeitar semver invalido",
			content:      []byte("---\nname: my-skill\nversion: not-semver\ndescription: A skill.\n---\n"),
			wantContains: []string{"version", "semver"},
		},
		{
			name:         "deve rejeitar name divergente",
			content:      []byte("---\nname: wrong-name\nversion: 1.0.0\ndescription: A skill.\n---\n"),
			skillName:    "my-skill",
			wantContains: []string{"name"},
		},
		{
			name:         "deve rejeitar depends_on ausente",
			content:      []byte("---\nname: my-skill\nversion: 1.0.0\ndescription: A skill.\ndepends_on: [ghost-skill]\n---\n"),
			skillName:    "my-skill",
			dependencies: []string{"review", "bugfix"},
			wantContains: []string{"depends_on", "ghost-skill"},
		},
	}

	for _, scenario := range scenarios {
		s.Run(scenario.name, func() {
			err := NewCatalog().ValidateFrontmatter(scenario.content, scenario.skillName, scenario.dependencies)
			s.Error(err)
			if err == nil {
				return
			}
			if len(scenario.wantContains) == 1 {
				s.Contains(err.Error(), scenario.wantContains[0])
				return
			}

			matched := false
			for _, want := range scenario.wantContains {
				if strings.Contains(err.Error(), want) {
					matched = true
				}
			}
			s.True(matched, "expected error containing one of %v, got: %v", scenario.wantContains, err)
		})
	}
}

func (s *FrontmatterSuite) TestValidateFrontmatterValidSkill() {
	content := []byte("---\nname: my-skill\nversion: 1.0.0\ndescription: Uma skill valida.\n---\n# My Skill\n")
	err := NewCatalog().ValidateFrontmatter(content, "my-skill", nil)
	s.NoError(err, "unexpected error for valid frontmatter")
}

func (s *FrontmatterSuite) TestSemverGreater() {
	scenarios := []struct {
		name string
		a    string
		b    string
		want bool
	}{
		{name: "1.0.0 maior que 0.9.0", a: "1.0.0", b: "0.9.0", want: true},
		{name: "1.1.0 maior que 1.0.0", a: "1.1.0", b: "1.0.0", want: true},
		{name: "1.0.1 maior que 1.0.0", a: "1.0.1", b: "1.0.0", want: true},
		{name: "1.0.0 igual a 1.0.0", a: "1.0.0", b: "1.0.0", want: false},
		{name: "0.9.0 menor que 1.0.0", a: "0.9.0", b: "1.0.0", want: false},
		{name: "2.0.0-beta maior que 1.9.9", a: "2.0.0-beta", b: "1.9.9", want: true},
		{name: "v1.1.0 maior que 1.0.0", a: "v1.1.0", b: "1.0.0", want: true},
	}

	for _, scenario := range scenarios {
		s.Run(scenario.name, func() {
			got := NewCatalog().SemverGreater(scenario.a, scenario.b)
			s.Equal(scenario.want, got, "NewCatalog().SemverGreater(%q, %q)", scenario.a, scenario.b)
		})
	}
}

func (s *FrontmatterSuite) TestIsValidSemver() {
	scenarios := []struct {
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

	for _, scenario := range scenarios {
		s.Run(scenario.name, func() {
			got := NewCatalog().IsValidSemver(scenario.version)
			s.Equal(scenario.want, got, "NewCatalog().IsValidSemver(%q)", scenario.version)
		})
	}
}
