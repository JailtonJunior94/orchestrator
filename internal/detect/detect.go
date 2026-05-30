package detect

import (
	"path/filepath"
	"strings"

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

var _ Detector = (*FileDetector)(nil)

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

	if d.hasDotNet(projectDir) {
		langs = append(langs, skills.LangDotNet)
	}

	return langs
}

// hasDotNet detecta um projeto .NET/C# por marcadores de solucao/projeto.
// Verifica arquivos de nome fixo no nivel de solucao e, na ausencia destes,
// procura por qualquer arquivo *.sln ou *.csproj no diretorio raiz.
func (d *FileDetector) hasDotNet(projectDir string) bool {
	fixed := []string{"global.json", "Directory.Build.props", "Directory.Packages.props"}
	for _, f := range fixed {
		if d.fs.Exists(filepath.Join(projectDir, f)) {
			return true
		}
	}

	entries, err := d.fs.ReadDir(projectDir)
	if err != nil {
		return false
	}
	for _, e := range entries {
		name := e.Name()
		if strings.HasSuffix(name, ".sln") || strings.HasSuffix(name, ".csproj") {
			return true
		}
	}
	return false
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
