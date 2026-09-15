package durable_test

import (
	"testing"

	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/memory/durable"
)

func TestDurabilityPolicy_ClassifiesDeclaredSectionAsPRD(t *testing.T) {
	t.Parallel()
	got := durable.DefaultDurabilityPolicy.Classify("declared-section")
	if got != durable.DurabilityPRD {
		t.Fatalf("Classify(declared-section) = %v, want DurabilityPRD", got)
	}
}

func TestDurabilityPolicy_ClassifiesSessionSummaryAsEphemeral(t *testing.T) {
	t.Parallel()
	got := durable.DefaultDurabilityPolicy.Classify("session-summary")
	if got != durable.DurabilityEphemeral {
		t.Fatalf("Classify(session-summary) = %v, want DurabilityEphemeral", got)
	}
}

func TestDurabilityPolicy_ClassifiesUnknownKindAsEphemeral(t *testing.T) {
	t.Parallel()
	got := durable.DefaultDurabilityPolicy.Classify("unknown-kind")
	if got != durable.DurabilityEphemeral {
		t.Fatalf("Classify(unknown-kind) = %v, want DurabilityEphemeral (fail-closed default)", got)
	}
}
