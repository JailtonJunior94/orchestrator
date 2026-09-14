package specs_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func dispatchProofSource(t *testing.T) string {
	t.Helper()
	path := filepath.Join(repoRoot(t), "internal", "runtime", "specs", "parity_dispatch_proof_test.go")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(data)
}

func TestDispatchProofHasNoFabricableEvidenceEscape(t *testing.T) {
	t.Parallel()

	source := dispatchProofSource(t)
	forbidden := []string{"AISPEC_DISPATCH_PROOF_LOG", "dispatchProofLogEnvVar", "os.Getenv("}
	for _, escape := range forbidden {
		if strings.Contains(source, escape) {
			t.Fatalf("RF-56: parity_dispatch_proof_test.go contains %q; dispatch evidence must come from a real run of the integration suite, never from a file or variable a reviewer can supply", escape)
		}
	}
}

func TestDispatchProofCollectsEvidenceFromIntegrationSubprocess(t *testing.T) {
	t.Parallel()

	source := dispatchProofSource(t)
	required := []string{`exec.Command("go", "test", "-tags=integration"`, `"-count=1"`}
	for _, fragment := range required {
		if !strings.Contains(source, fragment) {
			t.Fatalf("RF-56/O-01: dispatch evidence collection must execute the integration suite; missing %q", fragment)
		}
	}
}
