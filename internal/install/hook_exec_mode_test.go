package install_test

import (
	"os"
	"path/filepath"
	"testing"
)

func TestInstalledValidationHooksAreExecutableForEveryAgent(t *testing.T) {
	t.Parallel()

	projectDir := installMandatoryMatrixProject(t, false)

	hookDirs := []string{".agents/hooks", ".claude/hooks", ".codex/hooks", ".github/hooks"}
	hooks := []string{"validate-preload.sh", "validate-governance.sh", "validate-session-end.sh"}

	checked := 0
	for _, dir := range hookDirs {
		for _, hook := range hooks {
			full := filepath.Join(projectDir, filepath.FromSlash(dir), hook)
			info, err := os.Stat(full)
			if err != nil {
				t.Errorf("%s/%s was not installed: %v", dir, hook, err)
				continue
			}
			checked++
			if info.Mode().Perm()&0o111 == 0 {
				t.Errorf("%s/%s installed without execute bit (mode %v)", dir, hook, info.Mode().Perm())
			}
		}
	}
	if checked != len(hookDirs)*len(hooks) {
		t.Fatalf("expected %d installed hooks, checked %d", len(hookDirs)*len(hooks), checked)
	}
}
