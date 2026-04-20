package skills

import (
	"testing"
)

func BenchmarkValidateFrontmatterSchema(b *testing.B) {
	validYAML := []byte(`---
name: test-skill
version: 1.0.0
description: skill de teste para benchmark
---
# Test Skill
Conteudo do SKILL.md para benchmark.
`)
	for b.Loop() {
		_ = ValidateFrontmatterSchema(validYAML, "test-skill")
	}
}
