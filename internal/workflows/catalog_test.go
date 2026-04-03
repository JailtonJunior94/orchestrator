package workflows

import (
	"context"
	"errors"
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
