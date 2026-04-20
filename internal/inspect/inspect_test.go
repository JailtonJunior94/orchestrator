package inspect

import (
	"io"
	"testing"

	"github.com/JailtonJunior94/ai-spec-harness/internal/fs"
	"github.com/JailtonJunior94/ai-spec-harness/internal/manifest"
	"github.com/JailtonJunior94/ai-spec-harness/internal/output"
	"github.com/JailtonJunior94/ai-spec-harness/internal/skills"
)

func silentPrinter() *output.Printer {
	return &output.Printer{Out: io.Discard, Err: io.Discard}
}

type fakeDetector struct {
	tools []skills.Tool
	langs []skills.Lang
}

func (d *fakeDetector) DetectLangs(projectDir string) []skills.Lang { return d.langs }
func (d *fakeDetector) DetectTools(projectDir string) []skills.Tool { return d.tools }

func setupService(fake *fs.FakeFileSystem, det *fakeDetector) *Service {
	mfst := manifest.NewStore(fake)
	return NewService(fake, silentPrinter(), mfst, det)
}

func TestExecute_WithManifestAndSkills(t *testing.T) {
	fake := fs.NewFakeFileSystem()
	fake.Dirs["/project"] = true
	fake.Dirs["/project/.agents/skills"] = true
	fake.Dirs["/project/.agents/skills/go-implementation"] = true
	fake.Files["/project/.agents/skills/go-implementation/SKILL.md"] = []byte(`---
name: go-implementation
version: 1.2.0
---
# Go Implementation
`)
	fake.Files["/project/.ai_spec_harness.json"] = []byte(`{
		"version": "0.5.0",
		"created_at": "2026-01-01T00:00:00Z",
		"updated_at": "2026-04-01T00:00:00Z",
		"source_dir": "/governance",
		"link_mode": "copy",
		"tools": ["claude", "gemini"],
		"langs": ["go"],
		"skills": ["go-implementation"],
		"checksums": {}
	}`)

	det := &fakeDetector{
		tools: []skills.Tool{skills.ToolClaude},
		langs: []skills.Lang{skills.LangGo},
	}

	svc := setupService(fake, det)
	err := svc.Execute("/project")
	if err != nil {
		t.Errorf("Execute() returned error: %v", err)
	}
}

func TestExecute_NoManifest(t *testing.T) {
	fake := fs.NewFakeFileSystem()
	fake.Dirs["/project"] = true

	det := &fakeDetector{
		tools: []skills.Tool{},
		langs: []skills.Lang{},
	}

	svc := setupService(fake, det)
	err := svc.Execute("/project")
	if err != nil {
		t.Errorf("Execute() returned error: %v", err)
	}
}

func TestExecute_DirectoryNotFound(t *testing.T) {
	fake := fs.NewFakeFileSystem()
	det := &fakeDetector{}
	svc := setupService(fake, det)
	err := svc.Execute("/nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent directory")
	}
}

func TestExecute_CorruptedManifest(t *testing.T) {
	fake := fs.NewFakeFileSystem()
	fake.Dirs["/project"] = true
	fake.Files["/project/.ai_spec_harness.json"] = []byte(`{invalid json}`)

	det := &fakeDetector{tools: []skills.Tool{}, langs: []skills.Lang{}}
	svc := setupService(fake, det)

	err := svc.Execute("/project")
	if err != nil {
		t.Errorf("Execute() returned error: %v (should warn, not fail)", err)
	}
}

func TestExecute_SkillsWithSymlink(t *testing.T) {
	fake := fs.NewFakeFileSystem()
	fake.Dirs["/project"] = true
	fake.Dirs["/project/.agents/skills"] = true
	fake.Dirs["/project/.agents/skills/review"] = true
	fake.Links["/project/.agents/skills/review"] = "/governance/skills/review"
	fake.Files["/project/.agents/skills/review/SKILL.md"] = []byte(`---
name: review
version: 2.0.0
---
# Review
`)

	det := &fakeDetector{
		tools: []skills.Tool{skills.ToolClaude, skills.ToolCopilot},
		langs: []skills.Lang{skills.LangNode},
	}

	svc := setupService(fake, det)
	err := svc.Execute("/project")
	if err != nil {
		t.Errorf("Execute() returned error: %v", err)
	}
}

func TestToolNames(t *testing.T) {
	tools := []skills.Tool{skills.ToolClaude, skills.ToolGemini}
	names := toolNames(tools)
	if len(names) != 2 {
		t.Fatalf("len = %d, want 2", len(names))
	}
	if names[0] != "claude" {
		t.Errorf("names[0] = %q, want claude", names[0])
	}
	if names[1] != "gemini" {
		t.Errorf("names[1] = %q, want gemini", names[1])
	}
}

func TestLangNames(t *testing.T) {
	langs := []skills.Lang{skills.LangGo, skills.LangPython}
	names := langNames(langs)
	if len(names) != 2 {
		t.Fatalf("len = %d, want 2", len(names))
	}
	if names[0] != "go" {
		t.Errorf("names[0] = %q, want go", names[0])
	}
}
