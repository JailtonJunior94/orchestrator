package skills

import (
	"strings"
	"testing"
)

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
