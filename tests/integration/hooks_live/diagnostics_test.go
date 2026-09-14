package hooks_live_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	hookslive "github.com/JailtonJunior94/ai-spec-harness/tests/integration/hooks_live"
)

func repoRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("resolve working directory: %v", err)
	}
	root := filepath.Join(dir, "..", "..", "..")
	if _, err := os.Stat(filepath.Join(root, "go.mod")); err != nil {
		t.Fatalf("resolve repository root from %s: %v", dir, err)
	}
	return root
}

func TestCanonicalDiagnosticsStayInSyncWithTheirEmittingScript(t *testing.T) {
	t.Parallel()
	root := repoRoot(t)
	for _, source := range hookslive.CanonicalDiagnosticSources() {
		path := filepath.Join(root, source.SourceFile)
		data, err := os.ReadFile(path)
		if err != nil {
			t.Errorf("read canonical emitter %s: %v", source.SourceFile, err)
			continue
		}
		if !strings.Contains(string(data), source.Diagnostic) {
			t.Errorf("canonical emitter %s no longer emits %q — the live assertions would silently stop proving native dispatch; update the constant in diagnostics.go to the text the script emits today", source.SourceFile, source.Diagnostic)
		}
	}
}

func TestPreToolDenialDiagnosticsCoverEveryLiveAgent(t *testing.T) {
	t.Parallel()
	for _, agent := range hookslive.LiveAgentIDs {
		if len(hookslive.PreToolDenialDiagnostics(agent)) == 0 {
			t.Errorf("agent %s has no pre-tool denial diagnostic: its cell could pass without any hook running", agent)
		}
		if len(hookslive.PostToolDiagnostics(agent)) == 0 {
			t.Errorf("agent %s has no post-tool diagnostic", agent)
		}
		if len(hookslive.SessionEndDiagnostics(agent)) == 0 {
			t.Errorf("agent %s has no session-end diagnostic", agent)
		}
	}
}
