package hooks_live

import (
	"fmt"
	"sort"
	"strings"

	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/specs"
)

var LiveAgentIDs = []string{"claude", "codex", "copilot", "opencode"}

type MatrixCell struct {
	Agent string
	Point specs.CanonicalPoint
}

func LiveMatrix() []MatrixCell {
	catalog := specs.NewCatalog()
	cells := make([]MatrixCell, 0, len(LiveAgentIDs)*len(catalog.CanonicalPoints()))
	for _, agent := range LiveAgentIDs {
		for _, point := range catalog.CanonicalPoints() {
			cells = append(cells, MatrixCell{Agent: agent, Point: point})
		}
	}
	return cells
}

type CellResult struct {
	Cell    MatrixCell
	Skipped bool
	Reason  string
}

func AggregateCellResults(results []CellResult) error {
	skipped := make([]string, 0, len(results))
	for _, r := range results {
		if r.Skipped {
			skipped = append(skipped, fmt.Sprintf("%s/%s (%s)", r.Cell.Agent, r.Cell.Point, r.Reason))
		}
	}
	if len(skipped) == 0 {
		return nil
	}
	sort.Strings(skipped)
	return fmt.Errorf("hooks_live: %d cell(s) skipped — nightly matrix must not skip cells: %s", len(skipped), strings.Join(skipped, "; "))
}
