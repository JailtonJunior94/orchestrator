//go:build integration

package integration

import (
	"os/exec"
	"path/filepath"
	"testing"
)

func TestHookInventoryGate(t *testing.T) {
	root := repoRootForDispatch(t)
	command := exec.Command("bash", filepath.Join(root, "scripts", "check-hooks-inventory.sh"))
	command.Dir = root
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("run hook inventory gate: %v: %s", err, output)
	}
}
