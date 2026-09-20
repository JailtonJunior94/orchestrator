package qualitygate

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/JailtonJunior94/ai-spec-harness/internal/fs"
)

func TestFileEvidenceWriter_PersistsRecognizableTestCommand(t *testing.T) {
	fakeFS := fs.NewFakeFileSystem()
	writer := NewFileEvidenceWriter(fakeFS, "/project/.agents/generated/quality-gate-evidence")

	evidence := GateEvidence{
		GateID:      GateID,
		PolicyID:    "quality-gate:feature:low",
		TaskID:      "10.0",
		TaskType:    "feature",
		Risk:        "low",
		Fingerprint: "abc123",
		Decision:    "BLOCK",
		Reason:      "required checks failed: test (go test ./...)",
		Checks: []CheckEvidence{
			{Kind: "test", Command: "go test ./...", Required: true, ExitCode: 1, Output: "FAIL"},
		},
	}

	if err := writer.Persist(evidence); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	written, ok := fakeFS.Files["/project/.agents/generated/quality-gate-evidence/10.0.json"]
	if !ok {
		t.Fatalf("expected evidence file to be written")
	}

	var decoded GateEvidence
	if err := json.Unmarshal(written, &decoded); err != nil {
		t.Fatalf("unexpected error decoding evidence: %v", err)
	}
	if decoded.Fingerprint != "abc123" {
		t.Fatalf("Fingerprint = %q, want abc123", decoded.Fingerprint)
	}
	if decoded.Timestamp == "" {
		t.Fatalf("expected timestamp to be filled")
	}

	lines := decoded.CommandLines()
	found := false
	for _, line := range lines {
		if strings.Contains(line, "go test") {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected a recognizable test command among %v", lines)
	}
}

func TestGateEvidence_CommandLines(t *testing.T) {
	evidence := GateEvidence{
		Checks: []CheckEvidence{
			{Command: "go test ./..."},
			{Command: "golangci-lint run"},
		},
	}
	lines := evidence.CommandLines()
	if len(lines) != 2 {
		t.Fatalf("CommandLines() = %v, want 2 entries", lines)
	}
}
