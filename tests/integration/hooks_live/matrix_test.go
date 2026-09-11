package hooks_live_test

import (
	"testing"

	hookslive "github.com/JailtonJunior94/ai-spec-harness/tests/integration/hooks_live"
)

func TestLiveMatrixHasTwelveCells(t *testing.T) {
	t.Parallel()
	cells := hookslive.LiveMatrix()
	if len(cells) != 12 {
		t.Fatalf("expected 4 agents x 3 canonical points = 12 cells, got %d", len(cells))
	}
}

func TestAggregateCellResultsFailsWhenAnyCellSkipped(t *testing.T) {
	t.Parallel()
	cells := hookslive.LiveMatrix()

	results := make([]hookslive.CellResult, 0, len(cells))
	for i, cell := range cells {
		skipped := i == 0
		reason := ""
		if skipped {
			reason = "binary not found in PATH"
		}
		results = append(results, hookslive.CellResult{Cell: cell, Skipped: skipped, Reason: reason})
	}

	if err := hookslive.AggregateCellResults(results); err == nil {
		t.Fatalf("expected error when at least one cell is skipped — nightly must not become an empty test")
	}
}

func TestAggregateCellResultsPassesWhenNoCellSkipped(t *testing.T) {
	t.Parallel()
	cells := hookslive.LiveMatrix()

	results := make([]hookslive.CellResult, 0, len(cells))
	for _, cell := range cells {
		results = append(results, hookslive.CellResult{Cell: cell, Skipped: false})
	}

	if err := hookslive.AggregateCellResults(results); err != nil {
		t.Fatalf("expected no error when no cell is skipped: %v", err)
	}
}
