//go:build hooks_live

package hooks_live

import (
	"os/exec"
	"testing"
)

func binaryForAgent(agent string) string {
	switch agent {
	case "claude":
		return "claude"
	case "codex":
		return "codex"
	case "copilot":
		return "copilot"
	case "opencode":
		return "opencode"
	default:
		return ""
	}
}

func TestHooksLiveMatrixNoCellSkipped(t *testing.T) {
	cells := LiveMatrix()

	results := make([]CellResult, 0, len(cells))
	for _, cell := range cells {
		bin := binaryForAgent(cell.Agent)
		if bin == "" {
			results = append(results, CellResult{Cell: cell, Skipped: true, Reason: "unknown agent"})
			continue
		}
		path, err := exec.LookPath(bin)
		if err != nil {
			results = append(results, CellResult{Cell: cell, Skipped: true, Reason: "binary " + bin + " not found in PATH — hooks_live requires the four CLIs installed and authenticated"})
			continue
		}
		if err := exec.Command(path, "--version").Run(); err != nil {
			results = append(results, CellResult{Cell: cell, Skipped: true, Reason: "binary " + bin + " present but not executable: " + err.Error()})
			continue
		}
		results = append(results, CellResult{Cell: cell, Skipped: false})
	}

	if err := AggregateCellResults(results); err != nil {
		t.Fatal(err)
	}
}
