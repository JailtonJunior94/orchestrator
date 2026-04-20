package scaffold

import (
	"io"
	"strings"
	"testing"

	"github.com/JailtonJunior94/ai-spec-harness/internal/fs"
	"github.com/JailtonJunior94/ai-spec-harness/internal/output"
)

func TestScaffold_References(t *testing.T) {
	t.Parallel()
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
	t.Parallel()
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

func TestScaffold_ThreeArtifacts(t *testing.T) {
	t.Parallel()
	ffs := fs.NewFakeFileSystem()
	printer := &output.Printer{Out: io.Discard, Err: io.Discard}
	svc := NewService(ffs, printer)

	if err := svc.Execute("rust", "/root"); err != nil {
		t.Fatalf("Execute: %v", err)
	}

	skillDir := "/root/.agents/skills/rust-implementation"

	// artifact 1: SKILL.md
	if _, ok := ffs.Files[skillDir+"/SKILL.md"]; !ok {
		t.Error("SKILL.md not created")
	}

	// artifact 2: at least one reference stub
	if _, ok := ffs.Files[skillDir+"/references/conventions.md"]; !ok {
		t.Error("references/conventions.md not created")
	}

	// artifact 3: Gemini TOML
	tomlPath := "/root/.gemini/commands/rust-implementation.toml"
	data, ok := ffs.Files[tomlPath]
	if !ok {
		t.Fatal("Gemini TOML not created")
	}

	content := string(data)
	if !strings.Contains(content, "rust-implementation") {
		t.Error("TOML does not reference skill name")
	}
	if !strings.Contains(content, "{{args}}") {
		t.Error("TOML missing {{args}} placeholder")
	}
}

func TestScaffold_Idempotent(t *testing.T) {
	t.Parallel()
	ffs := fs.NewFakeFileSystem()
	printer := &output.Printer{Out: io.Discard, Err: io.Discard}
	svc := NewService(ffs, printer)

	if err := svc.Execute("rust", "/root"); err != nil {
		t.Fatalf("first Execute: %v", err)
	}
	if err := svc.Execute("rust", "/root"); err != nil {
		t.Fatalf("second Execute (idempotency): %v", err)
	}

	tomlPath := "/root/.gemini/commands/rust-implementation.toml"
	if _, ok := ffs.Files[tomlPath]; !ok {
		t.Error("Gemini TOML missing after second run")
	}
}
