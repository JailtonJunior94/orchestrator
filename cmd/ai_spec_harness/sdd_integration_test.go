//go:build integration

package aispecharness

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/JailtonJunior94/ai-spec-harness/internal/sdd"
)

func TestSDDCommandsLifecycleIntegration(t *testing.T) {
	prdDir := t.TempDir()
	for _, file := range []string{"prd.md", "techspec.md", "tasks.md"} {
		if err := os.WriteFile(filepath.Join(prdDir, file), []byte(file), 0o644); err != nil {
			t.Fatalf("criar %s: %v", file, err)
		}
	}

	for _, args := range [][]string{
		{"approve", "prd", prdDir},
		{"approve", "techspec", prdDir},
		{"approve", "tasks", prdDir},
		{"validate-sdd", prdDir},
		{"invalidate", prdDir, "--from", "prd"},
		{"validate-sdd", prdDir},
	} {
		t.Run(args[0], func(t *testing.T) {
			t.Setenv("AI_INVOCATION_DEPTH", "0")
			root := newRootCmd()
			root.SetArgs(args)
			if err := root.Execute(); err != nil {
				t.Fatalf("executar %v: %v", args, err)
			}
		})
	}

	state, err := sdd.NewStore().Load(prdDir)
	if err != nil {
		t.Fatalf("carregar estado SDD: %v", err)
	}
	for _, artifact := range []sdd.Artifact{sdd.ArtifactTechSpec, sdd.ArtifactTasks} {
		entry := state.Artifacts[artifact]
		if entry.Status != sdd.StatusStale || entry.Approved {
			t.Fatalf("%s deveria estar stale e sem aprovacao: %#v", artifact, entry)
		}
	}
}

func TestRuntimeCapabilitiesCmdIntegration(t *testing.T) {
	dir := t.TempDir()
	command := exec.Command("git", "init", dir)
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("inicializar repositorio Git: %v: %s", err, output)
	}

	var output bytes.Buffer
	t.Setenv("AI_INVOCATION_DEPTH", "0")
	root := newRootCmd()
	root.SetOut(&output)
	root.SetArgs([]string{"runtime-capabilities", dir})
	if err := root.Execute(); err != nil {
		t.Fatalf("executar runtime-capabilities: %v", err)
	}

	var capabilities struct {
		SupportsWrite     bool `json:"supports_write"`
		SupportsWorktree  bool `json:"supports_worktree"`
		IsolatedWorktrees bool `json:"isolated_worktrees"`
	}
	if err := json.Unmarshal(output.Bytes(), &capabilities); err != nil {
		t.Fatalf("decodificar capacidades: %v", err)
	}
	if !capabilities.SupportsWrite || !capabilities.SupportsWorktree || capabilities.IsolatedWorktrees {
		t.Fatalf("capacidades inesperadas: %#v", capabilities)
	}
}

func TestSDDCommandsRejectOutOfOrderApprovalIntegration(t *testing.T) {
	prdDir := t.TempDir()
	for _, file := range []string{"prd.md", "techspec.md", "tasks.md"} {
		if err := os.WriteFile(filepath.Join(prdDir, file), []byte(file), 0o644); err != nil {
			t.Fatalf("criar %s: %v", file, err)
		}
	}

	root := newRootCmd()
	root.SetArgs([]string{"approve", "techspec", prdDir})
	t.Setenv("AI_INVOCATION_DEPTH", "0")
	if err := root.Execute(); err == nil {
		t.Fatal("aprovacao fora da ordem deveria falhar")
	}
}
