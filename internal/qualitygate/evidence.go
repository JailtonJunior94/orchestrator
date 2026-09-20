package qualitygate

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"time"

	"github.com/JailtonJunior94/ai-spec-harness/internal/fs"
)

type CheckEvidence struct {
	Kind     string `json:"kind"`
	Command  string `json:"command"`
	Required bool   `json:"required"`
	ExitCode int    `json:"exit_code"`
	Output   string `json:"output"`
}

type GateEvidence struct {
	GateID      string          `json:"gate_id"`
	PolicyID    string          `json:"policy_id"`
	TaskID      string          `json:"task_id"`
	TaskType    string          `json:"task_type"`
	Risk        string          `json:"risk"`
	Fingerprint string          `json:"fingerprint"`
	Decision    string          `json:"decision"`
	Reason      string          `json:"reason"`
	Checks      []CheckEvidence `json:"checks"`
	Timestamp   string          `json:"timestamp"`
}

func (e GateEvidence) CommandLines() []string {
	lines := make([]string, 0, len(e.Checks))
	for _, check := range e.Checks {
		lines = append(lines, check.Command)
	}
	return lines
}

type EvidenceWriter interface {
	Persist(evidence GateEvidence) error
}

type fileEvidenceWriter struct {
	fs  fs.FileSystem
	dir string
	now func() time.Time
}

var _ EvidenceWriter = (*fileEvidenceWriter)(nil)

func NewFileEvidenceWriter(filesystem fs.FileSystem, dir string) EvidenceWriter {
	return &fileEvidenceWriter{fs: filesystem, dir: dir, now: time.Now}
}

func (w *fileEvidenceWriter) Persist(evidence GateEvidence) error {
	if evidence.Timestamp == "" {
		evidence.Timestamp = w.now().UTC().Format(time.RFC3339)
	}

	encoded, err := json.MarshalIndent(evidence, "", "  ")
	if err != nil {
		return fmt.Errorf("qualitygate: encode evidence: %w", err)
	}

	taskID := evidence.TaskID
	if taskID == "" {
		taskID = "adhoc"
	}
	path := filepath.Join(w.dir, taskID+".json")
	if err := w.fs.WriteFileAtomic(path, append(encoded, '\n')); err != nil {
		return fmt.Errorf("qualitygate: write evidence %s: %w", path, err)
	}
	return nil
}
