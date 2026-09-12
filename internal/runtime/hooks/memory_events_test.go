package hooks_test

import (
	"testing"

	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/hooks"
)

func TestMemoryEvents_KindMatchesPoint(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name  string
		event hooks.Event
		want  string
	}{
		{"FactRecorded", hooks.MemoryFactRecordedEvent{}, hooks.PointMemoryFactRecorded},
		{"FactArchived", hooks.MemoryFactArchivedEvent{}, hooks.PointMemoryFactArchived},
		{"FactPromoted", hooks.MemoryFactPromotedEvent{}, hooks.PointMemoryFactPromoted},
		{"ContradictionDetected", hooks.MemoryContradictionDetectedEvent{}, hooks.PointMemoryContradictionDetected},
		{"SecretRedacted", hooks.MemorySecretRedactedEvent{}, hooks.PointMemorySecretRedacted},
		{"CompactionExecuted", hooks.MemoryCompactionExecutedEvent{}, hooks.PointMemoryCompactionExecuted},
		{"BatonTransferred", hooks.MemoryBatonTransferredEvent{}, hooks.PointMemoryBatonTransferred},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := tc.event.Kind(); got != tc.want {
				t.Errorf("Kind() = %q; want %q", got, tc.want)
			}
		})
	}
}

func TestMemoryEvents_PointsAreDistinct(t *testing.T) {
	t.Parallel()

	points := []string{
		hooks.PointMemoryFactRecorded,
		hooks.PointMemoryFactArchived,
		hooks.PointMemoryFactPromoted,
		hooks.PointMemoryContradictionDetected,
		hooks.PointMemorySecretRedacted,
		hooks.PointMemoryCompactionExecuted,
		hooks.PointMemoryBatonTransferred,
	}

	seen := make(map[string]bool, len(points))
	for _, p := range points {
		if seen[p] {
			t.Fatalf("ponto de hook duplicado: %s", p)
		}
		seen[p] = true
	}
	if len(seen) != 7 {
		t.Fatalf("esperava 7 pontos distintos, obtido %d", len(seen))
	}
}
