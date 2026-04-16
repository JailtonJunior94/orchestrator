package workflows

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestCatalogLoad(t *testing.T) {
	t.Parallel()

	catalog := NewCatalog(NewParser())

	t.Run("load dev-workflow", func(t *testing.T) {
		t.Parallel()
		wf, err := catalog.Load(context.Background(), "dev-workflow")
		if err != nil {
			t.Fatal(err)
		}
		if wf.Name != "dev-workflow" {
			t.Errorf("name = %q", wf.Name)
		}
		if len(wf.Steps) != 4 {
			t.Errorf("steps = %d, want 4", len(wf.Steps))
		}
		wantProviders := []string{"claude", "claude", "claude", "copilot"}
		for i, want := range wantProviders {
			if wf.Steps[i].Provider != want {
				t.Fatalf("step %d provider = %q, want %q", i, wf.Steps[i].Provider, want)
			}
		}
	})

	t.Run("workflow not found", func(t *testing.T) {
		t.Parallel()
		_, err := catalog.Load(context.Background(), "nonexistent")
		if !errors.Is(err, ErrWorkflowNotFound) {
			t.Errorf("error = %v, want ErrWorkflowNotFound", err)
		}
	})
}

func TestCatalogList(t *testing.T) {
	t.Parallel()

	catalog := NewCatalog(NewParser())
	names, err := catalog.List()
	if err != nil {
		t.Fatal(err)
	}
	if len(names) == 0 {
		t.Error("expected at least one workflow")
	}

	found := false
	for _, n := range names {
		if n == "dev-workflow" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("dev-workflow not found in %v", names)
	}
}

func TestExternalDevWorkflowMatchesEmbeddedDefinition(t *testing.T) {
	t.Parallel()

	parser := NewParser()

	embeddedData, err := os.ReadFile(filepath.Join("embedded", "dev-workflow.yaml"))
	if err != nil {
		t.Fatalf("ReadFile(embedded) error = %v", err)
	}
	externalData, err := os.ReadFile(filepath.Join("..", "..", "workflows", "dev-workflow.yaml"))
	if err != nil {
		t.Fatalf("ReadFile(external) error = %v", err)
	}

	embeddedWorkflow, err := parser.Parse(context.Background(), embeddedData)
	if err != nil {
		t.Fatalf("Parse(embedded) error = %v", err)
	}
	externalWorkflow, err := parser.Parse(context.Background(), externalData)
	if err != nil {
		t.Fatalf("Parse(external) error = %v", err)
	}

	if !reflect.DeepEqual(embeddedWorkflow, externalWorkflow) {
		t.Fatal("external workflow diverged from embedded definition")
	}
}
