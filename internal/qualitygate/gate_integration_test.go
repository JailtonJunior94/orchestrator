//go:build integration

package qualitygate

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/JailtonJunior94/ai-spec-harness/internal/detect"
	"github.com/JailtonJunior94/ai-spec-harness/internal/fs"
)

func TestGate_Evaluate_RealShellExecutorAgainstGoFixture(t *testing.T) {
	dir := t.TempDir()

	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module fixture\n\ngo 1.21\n"), 0o644); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "main.go"), []byte("package fixture\n"), 0o644); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	osFS := fs.NewOSFileSystem()
	gate := NewGate(
		NewDefaultPolicyLoader(osFS),
		detect.NewToolchainDetector(osFS),
		NewShellExecutor(),
		NewFileCache(osFS, filepath.Join(dir, ".agents/generated/quality-gate-cache.json")),
		NewFileEvidenceWriter(osFS, filepath.Join(dir, ".agents/generated/quality-gate-evidence")),
	)

	result := gate.Evaluate(context.Background(), EvaluationInput{
		ProjectDir: dir,
		TaskID:     "integration-fixture",
		TaskType:   DefaultTaskType,
		Risk:       RiskLow,
	})

	if !result.Valid() {
		t.Fatalf("expected a valid result, got %+v", result)
	}

	evidencePath := filepath.Join(dir, ".agents/generated/quality-gate-evidence", "integration-fixture.json")
	if _, err := os.Stat(evidencePath); err != nil {
		t.Fatalf("expected evidence file at %s: %v", evidencePath, err)
	}
}
