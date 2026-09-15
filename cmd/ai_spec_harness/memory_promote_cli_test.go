package aispecharness

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/JailtonJunior94/ai-spec-harness/internal/fs"
	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/memory/durable"
)

func TestMemoryPromoteCommandCreatesMissingTargetLayerDirectory(t *testing.T) {
	root := t.TempDir()
	projectDir := filepath.Join(root, "project")
	tasksDir := filepath.Join(projectDir, ".specs", "prd-x")
	if err := os.MkdirAll(filepath.Join(tasksDir, "memory"), 0o755); err != nil {
		t.Fatalf("prepare source layer directory: %v", err)
	}

	osFS := fs.NewOSFileSystem()
	layer := durable.NewLayer(osFS)
	catalog := durable.NewCatalog()
	content := "usar backoff exponencial"
	fact := durable.Fact{
		Identity:   durable.Identity{Key: durable.SemanticKey("decision.retry"), Hash: catalog.HashContent(content)},
		Content:    content,
		Durability: durable.DurabilityDurable,
		Origin:     durable.FactOrigin{Session: "session-1", CLI: "claude", Task: "task-1.0.md", Date: "2026-09-11T00:00:00Z"},
	}
	source := durable.Scope{Layer: durable.TargetLayerPRD, TasksDir: tasksDir}
	if _, err := layer.Consolidate(context.Background(), source, []durable.Fact{fact}); err != nil {
		t.Fatalf("seed source fact: %v", err)
	}

	targetDir := filepath.Join(projectDir, ".aispec", "memory")
	if _, err := os.Stat(targetDir); !os.IsNotExist(err) {
		t.Fatalf("target layer directory must not exist before promote, stat err=%v", err)
	}

	cmd := newMemoryPromoteCmd()
	cmd.SetArgs([]string{
		"--project-dir", projectDir,
		"--tasks-dir", tasksDir,
		"--key", string(fact.Identity.Key),
		"--hash", string(fact.Identity.Hash),
	})
	cmd.SetOut(os.Stderr)
	cmd.SetErr(os.Stderr)

	if err := cmd.ExecuteContext(context.Background()); err != nil {
		t.Fatalf("memory promote failed on a target layer whose directory does not exist yet: %v", err)
	}

	promoted, _, err := layer.Read(context.Background(), durable.Scope{Layer: durable.TargetLayerProject, ProjectDir: projectDir})
	if err != nil {
		t.Fatalf("read promoted project layer: %v", err)
	}
	if len(promoted) != 1 || promoted[0].Identity != fact.Identity {
		t.Fatalf("project layer facts=%+v, want exactly the promoted fact %+v", promoted, fact.Identity)
	}
}
