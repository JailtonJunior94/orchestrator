package taskloop

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/JailtonJunior94/ai-spec-harness/internal/config"
	"github.com/JailtonJunior94/ai-spec-harness/internal/fs"
)

func TestBuildRuntimeConfigRejectsNegativeMaxBugfixIterations(t *testing.T) {
	_, err := NewCatalog().BuildRuntimeConfig(config.Runtime{MaxBugfixIterations: -3})
	if !errors.Is(err, ErrInvalidMaxBugfixIterations) {
		t.Fatalf("erro = %v, quero ErrInvalidMaxBugfixIterations", err)
	}
}

func TestExecuteLegacyRuntimeReadsMaxBugfixIterationsFromConfigFile(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".claude"), 0o755); err != nil {
		t.Fatalf("setup: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, ".claude", "config.yaml"), []byte("max_bugfix_iterations: -3\n"), 0o644); err != nil {
		t.Fatalf("setup: %v", err)
	}

	prd := filepath.Join(root, "prd-x")
	if err := os.MkdirAll(prd, 0o755); err != nil {
		t.Fatalf("setup: %v", err)
	}
	for _, name := range []string{"tasks.md", "prd.md", "techspec.md"} {
		if err := os.WriteFile(filepath.Join(prd, name), []byte("# doc\n"), 0o644); err != nil {
			t.Fatalf("setup: %v", err)
		}
	}

	svc := NewService(fs.NewOSFileSystem(), newTestPrinter())
	svc.binaryChecker = noBinaryCheck

	err := svc.Execute(Options{
		PRDFolder:         prd,
		Tool:              "claude",
		Runtime:           "legacy",
		MaxIterations:     1,
		Timeout:           time.Second,
		ReportPath:        filepath.Join(prd, "report.md"),
		AllowUnknownModel: true,
	})
	if !errors.Is(err, ErrInvalidMaxBugfixIterations) {
		t.Fatalf("erro = %v, quero ErrInvalidMaxBugfixIterations no caminho legacy (RF-35)", err)
	}
}
