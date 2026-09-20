package contextgen

import (
	"strings"
	"testing"

	"github.com/JailtonJunior94/ai-spec-harness/internal/fs"
	"github.com/JailtonJunior94/ai-spec-harness/internal/output"
)

func TestBuildDirectoryTree_ExcludesHarnessGeneratedArtifacts(t *testing.T) {
	t.Parallel()

	ffs := fs.NewFakeFileSystem()
	ffs.Dirs["/project"] = true
	ffs.Dirs["/project/.github"] = true
	ffs.Dirs["/project/.github/workflows"] = true
	ffs.Dirs["/project/.github/skills"] = true
	ffs.Dirs["/project/.github/agents"] = true
	ffs.Dirs["/project/.github/hooks"] = true
	ffs.Dirs["/project/.github/copilot"] = true
	ffs.Files["/project/main.go"] = []byte("package main\n")
	ffs.Files["/project/.github/workflows/ci.yml"] = []byte("name: ci\n")
	ffs.Files["/project/.github/copilot-instructions.md"] = []byte("gerado\n")
	ffs.Files["/project/.github/copilot/settings.json"] = []byte("{}\n")
	ffs.Files["/project/.ai_spec_harness.json"] = []byte("{}\n")

	tree := NewGenerator(ffs, output.New(false)).buildDirectoryTree("/project")

	for _, generated := range []string{
		".ai_spec_harness.json",
		".github/copilot-instructions.md",
		".github/copilot/settings.json",
		".github/skills",
		".github/agents",
		".github/hooks",
	} {
		if strings.Contains(tree, generated) {
			t.Errorf("artefato gerado pelo harness %q nao deve aparecer no snapshot de estrutura:\n%s", generated, tree)
		}
	}

	for _, authored := range []string{"main.go", ".github/workflows"} {
		if !strings.Contains(tree, authored) {
			t.Errorf("caminho autoral %q deve permanecer no snapshot:\n%s", authored, tree)
		}
	}
}

func TestBuildDirectoryTree_StableAcrossHarnessInstall(t *testing.T) {
	t.Parallel()

	ffs := fs.NewFakeFileSystem()
	ffs.Dirs["/project"] = true
	ffs.Dirs["/project/.github"] = true
	ffs.Dirs["/project/.github/workflows"] = true
	ffs.Files["/project/main.go"] = []byte("package main\n")
	ffs.Files["/project/.github/workflows/ci.yml"] = []byte("name: ci\n")

	gen := NewGenerator(ffs, output.New(false))
	before := gen.buildDirectoryTree("/project")

	ffs.Dirs["/project/.github/skills"] = true
	ffs.Files["/project/.github/skills/review/SKILL.md"] = []byte("x\n")
	ffs.Dirs["/project/.github/skills/review"] = true
	ffs.Files["/project/.github/copilot-instructions.md"] = []byte("gerado\n")
	ffs.Files["/project/.ai_spec_harness.json"] = []byte("{}\n")

	after := gen.buildDirectoryTree("/project")

	if before != after {
		t.Errorf("snapshot de estrutura deve ser estavel apos a instalacao do harness:\n--- antes ---\n%s\n--- depois ---\n%s", before, after)
	}
}
