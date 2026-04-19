package skills

import "testing"

func TestMaxDepth_WithValue(t *testing.T) {
	content := []byte(`---
name: my-skill
version: 1.0.0
description: Uma skill com max_depth.
max_depth: 3
---
`)
	fm := ParseFrontmatter(content)
	if fm.MaxDepth != 3 {
		t.Errorf("MaxDepth: got %d, want 3", fm.MaxDepth)
	}
}

func TestMaxDepth_Default(t *testing.T) {
	content := []byte(`---
name: my-skill
version: 1.0.0
description: Uma skill sem max_depth.
---
`)
	fm := ParseFrontmatter(content)
	if fm.MaxDepth != 0 {
		t.Errorf("MaxDepth: got %d, want 0", fm.MaxDepth)
	}
}
