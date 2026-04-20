package detect

import (
	"testing"

	"github.com/JailtonJunior94/ai-spec-harness/internal/fs"
	"github.com/JailtonJunior94/ai-spec-harness/internal/skills"
)

func TestDetectLangs(t *testing.T) {
	t.Parallel()
	ffs := fs.NewFakeFileSystem()
	ffs.Files["/project/go.mod"] = []byte("module example")
	ffs.Files["/project/package.json"] = []byte("{}")

	det := NewFileDetector(ffs)
	langs := det.DetectLangs("/project")

	if len(langs) != 2 {
		t.Fatalf("expected 2 langs, got %d", len(langs))
	}
	if langs[0] != skills.LangGo || langs[1] != skills.LangNode {
		t.Errorf("unexpected langs: %v", langs)
	}
}

func TestDetectTools(t *testing.T) {
	t.Parallel()
	ffs := fs.NewFakeFileSystem()
	ffs.Files["/project/CLAUDE.md"] = []byte("# Claude")
	ffs.Files["/project/GEMINI.md"] = []byte("# Gemini")

	det := NewFileDetector(ffs)
	tools := det.DetectTools("/project")

	if len(tools) != 2 {
		t.Fatalf("expected 2 tools, got %d", len(tools))
	}
}

func TestDetectLangs_Empty(t *testing.T) {
	t.Parallel()
	ffs := fs.NewFakeFileSystem()
	ffs.Dirs["/project"] = true

	det := NewFileDetector(ffs)
	langs := det.DetectLangs("/project")

	if len(langs) != 0 {
		t.Errorf("expected 0 langs, got %v", langs)
	}
}
