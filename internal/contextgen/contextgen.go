package contextgen

import (
	"fmt"
	"path/filepath"

	"github.com/JailtonJunior94/ai-spec-harness/internal/fs"
	"github.com/JailtonJunior94/ai-spec-harness/internal/output"
	"github.com/JailtonJunior94/ai-spec-harness/internal/skills"
)

// Generator orquestra a geracao de governanca contextual.
// Na Fase 1, delega para o script shell existente.
// Na Fase 2+, a logica sera portada diretamente para Go.
type Generator struct {
	fs      fs.FileSystem
	printer *output.Printer
}

func NewGenerator(fsys fs.FileSystem, printer *output.Printer) *Generator {
	return &Generator{fs: fsys, printer: printer}
}

// Generate copia os arquivos de governanca base para o projeto alvo.
// Premissa: na Fase 1, copia os .md estaticos; na Fase 2 gerara conteudo contextual.
func (g *Generator) Generate(sourceDir, projectDir string, tools []skills.Tool, langs []skills.Lang) error {
	toolSet := make(map[skills.Tool]bool)
	for _, t := range tools {
		toolSet[t] = true
	}

	// AGENTS.md — sempre copiado
	agentsSrc := filepath.Join(sourceDir, "AGENTS.md")
	if g.fs.Exists(agentsSrc) {
		if err := g.fs.CopyFile(agentsSrc, filepath.Join(projectDir, "AGENTS.md")); err != nil {
			return fmt.Errorf("copiar AGENTS.md: %w", err)
		}
	}

	// CLAUDE.md
	if toolSet[skills.ToolClaude] {
		src := filepath.Join(sourceDir, "CLAUDE.md")
		if g.fs.Exists(src) {
			if err := g.fs.CopyFile(src, filepath.Join(projectDir, "CLAUDE.md")); err != nil {
				return fmt.Errorf("copiar CLAUDE.md: %w", err)
			}
		}
	}

	// GEMINI.md
	if toolSet[skills.ToolGemini] {
		src := filepath.Join(sourceDir, "GEMINI.md")
		if g.fs.Exists(src) {
			if err := g.fs.CopyFile(src, filepath.Join(projectDir, "GEMINI.md")); err != nil {
				return fmt.Errorf("copiar GEMINI.md: %w", err)
			}
		}
	}

	// copilot-instructions.md
	if toolSet[skills.ToolCopilot] {
		src := filepath.Join(sourceDir, ".github", "copilot-instructions.md")
		if g.fs.Exists(src) {
			dst := filepath.Join(projectDir, ".github", "copilot-instructions.md")
			if err := g.fs.CopyFile(src, dst); err != nil {
				return fmt.Errorf("copiar copilot-instructions.md: %w", err)
			}
		}
	}

	g.printer.Debug("Governanca contextual gerada com sucesso")
	return nil
}
