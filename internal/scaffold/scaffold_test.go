package scaffold

import (
	"testing"

	"github.com/JailtonJunior94/ai-spec-harness/internal/fs"
	"github.com/JailtonJunior94/ai-spec-harness/internal/output"
	"io"
)

func TestScaffold_References(t *testing.T) {
	ffs := fs.NewFakeFileSystem()
	printer := &output.Printer{Out: io.Discard, Err: io.Discard}
	svc := NewService(ffs, printer)

	if err := svc.Execute("node", "/root"); err != nil {
		t.Fatalf("Execute: %v", err)
	}

	requiredRefs := []string{"architecture", "messaging", "observability", "persistence", "security"}
	skillDir := "/root/.agents/skills/node-implementation"
	for _, ref := range requiredRefs {
		path := skillDir + "/references/" + ref + ".md"
		if _, ok := ffs.Files[path]; !ok {
			t.Errorf("reference file missing: %s", path)
		}
	}
}

func TestScaffold_AllRefs(t *testing.T) {
	ffs := fs.NewFakeFileSystem()
	printer := &output.Printer{Out: io.Discard, Err: io.Discard}
	svc := NewService(ffs, printer)

	if err := svc.Execute("python", "/root"); err != nil {
		t.Fatalf("Execute: %v", err)
	}

	skillDir := "/root/.agents/skills/python-implementation"
	allRefs := []string{
		"conventions", "architecture", "testing", "error-handling",
		"api", "patterns", "messaging", "observability", "concurrency",
		"resilience", "persistence", "security", "build",
		"examples-domain-flow",
	}
	for _, ref := range allRefs {
		path := skillDir + "/references/" + ref + ".md"
		if _, ok := ffs.Files[path]; !ok {
			t.Errorf("reference file missing: %s", path)
		}
	}
}
