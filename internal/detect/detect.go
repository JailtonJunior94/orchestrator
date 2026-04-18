package detect

import (
	"path/filepath"

	"github.com/JailtonJunior94/ai-spec-harness/internal/fs"
	"github.com/JailtonJunior94/ai-spec-harness/internal/skills"
)

// Detector identifica linguagens e ferramentas ja presentes em um projeto.
type Detector interface {
	DetectLangs(projectDir string) []skills.Lang
	DetectTools(projectDir string) []skills.Tool
}

// FileDetector detecta pelo layout do filesystem.
type FileDetector struct {
	fs fs.FileSystem
}

func NewFileDetector(fsys fs.FileSystem) *FileDetector {
	return &FileDetector{fs: fsys}
}

func (d *FileDetector) DetectLangs(projectDir string) []skills.Lang {
	var langs []skills.Lang

	goIndicators := []string{"go.mod", "go.work"}
	for _, f := range goIndicators {
		if d.fs.Exists(filepath.Join(projectDir, f)) {
			langs = append(langs, skills.LangGo)
			break
		}
	}

	nodeIndicators := []string{"package.json", "pnpm-workspace.yaml"}
	for _, f := range nodeIndicators {
		if d.fs.Exists(filepath.Join(projectDir, f)) {
			langs = append(langs, skills.LangNode)
			break
		}
	}

	pythonIndicators := []string{"pyproject.toml", "setup.py", "requirements.txt"}
	for _, f := range pythonIndicators {
		if d.fs.Exists(filepath.Join(projectDir, f)) {
			langs = append(langs, skills.LangPython)
			break
		}
	}

	return langs
}

func (d *FileDetector) DetectTools(projectDir string) []skills.Tool {
	var tools []skills.Tool

	if d.fs.Exists(filepath.Join(projectDir, "CLAUDE.md")) || d.fs.IsDir(filepath.Join(projectDir, ".claude")) {
		tools = append(tools, skills.ToolClaude)
	}
	if d.fs.Exists(filepath.Join(projectDir, "GEMINI.md")) || d.fs.IsDir(filepath.Join(projectDir, ".gemini")) {
		tools = append(tools, skills.ToolGemini)
	}
	if d.fs.IsDir(filepath.Join(projectDir, ".codex")) {
		tools = append(tools, skills.ToolCodex)
	}
	if d.fs.Exists(filepath.Join(projectDir, ".github", "copilot-instructions.md")) {
		tools = append(tools, skills.ToolCopilot)
	}

	return tools
}
