//go:build windows

package durable_test

import (
	"testing"

	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/memory/durable"
)

func TestCheckProcessLiveness_WindowsAlwaysUnreliable(t *testing.T) {
	t.Parallel()

	ref := durable.CurrentProcessRef()
	result := durable.DefaultLivenessProbe.Probe(ref)
	if result.Reliable {
		t.Fatal("expected windows probe to be unreliable so the lease deadline prevails (MD-003)")
	}
	if result.Alive {
		t.Fatal("expected windows probe to report not alive when unreliable")
	}
}
