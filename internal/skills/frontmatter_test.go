package skills

import "testing"

func TestParseFrontmatter(t *testing.T) {
	content := []byte(`---
version: 1.2.3
description: Analisa e classifica projetos.
---

# Skill
`)
	fm := ParseFrontmatter(content)

	if fm.Version != "1.2.3" {
		t.Errorf("version: got %q, want %q", fm.Version, "1.2.3")
	}
	if fm.Description != "Analisa e classifica projetos." {
		t.Errorf("description: got %q, want %q", fm.Description, "Analisa e classifica projetos.")
	}
}

func TestParseFrontmatter_Empty(t *testing.T) {
	fm := ParseFrontmatter([]byte("# Sem frontmatter"))
	if fm.Version != "" {
		t.Errorf("version: got %q, want empty", fm.Version)
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
