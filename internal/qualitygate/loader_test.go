package qualitygate

import (
	"errors"
	"path/filepath"
	"testing"

	"github.com/JailtonJunior94/ai-spec-harness/internal/fs"
)

func TestDefaultPolicyLoader_FallsBackToEmbeddedDefault(t *testing.T) {
	fakeFS := fs.NewFakeFileSystem()
	loader := NewDefaultPolicyLoader(fakeFS)

	policySet, err := loader.Load("/project")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, err := policySet.Lookup(DefaultTaskType, DefaultRisk); err != nil {
		t.Fatalf("expected default policy set to cover default combination: %v", err)
	}
}

func TestDefaultPolicyLoader_ReadsProjectArtifact(t *testing.T) {
	fakeFS := fs.NewFakeFileSystem()
	path := filepath.Join("/project", filepath.FromSlash(PolicyArtifactRelativePath))
	fakeFS.Files[path] = []byte(`
version: 1
policies:
  - task_type: custom
    risk: low
    required: [test]
`)
	loader := NewDefaultPolicyLoader(fakeFS)

	policySet, err := loader.Load("/project")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	taskType, err := NewTaskType("custom")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, err := policySet.Lookup(taskType, RiskLow); err != nil {
		t.Fatalf("expected declared combination to resolve: %v", err)
	}
}

func TestDefaultPolicyLoader_RejectsInvalidProjectArtifact(t *testing.T) {
	fakeFS := fs.NewFakeFileSystem()
	path := filepath.Join("/project", filepath.FromSlash(PolicyArtifactRelativePath))
	fakeFS.Files[path] = []byte(`
version: 1
policies:
  - task_type: custom
    risk: extreme
    required: [test]
`)
	loader := NewDefaultPolicyLoader(fakeFS)

	_, err := loader.Load("/project")
	if !errors.Is(err, ErrInvalidPolicy) {
		t.Fatalf("expected ErrInvalidPolicy, got %v", err)
	}
}
