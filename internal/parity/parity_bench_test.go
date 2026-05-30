package parity

import (
	"testing"

	"github.com/JailtonJunior94/ai-spec-harness/internal/skills"
)

func BenchmarkCheckParity_AllTools(b *testing.B) {
	snap, err := NewChecker().Generate("/bench-project", skills.AllTools, nil, "full")
	if err != nil {
		b.Fatalf("Generate: %v", err)
	}
	invariants := NewChecker().Invariants()
	b.ResetTimer()
	for b.Loop() {
		NewChecker().Run(snap, invariants)
	}
}
