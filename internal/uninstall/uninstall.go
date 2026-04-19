package uninstall

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/JailtonJunior94/ai-spec-harness/internal/fs"
	"github.com/JailtonJunior94/ai-spec-harness/internal/output"
)

// Service orquestra a remocao de governanca de um projeto.
type Service struct {
	fs      fs.FileSystem
	printer *output.Printer
}

func NewService(fsys fs.FileSystem, printer *output.Printer) *Service {
	return &Service{fs: fsys, printer: printer}
}

// Execute remove artefatos de governanca do projeto alvo.
func (s *Service) Execute(projectDir string, dryRun bool) error {
	absDir, err := filepath.Abs(projectDir)
	if err != nil {
		return err
	}

	if !s.fs.IsDir(absDir) {
		return fmt.Errorf("diretorio alvo nao encontrado: %s", absDir)
	}

	skillsDir := filepath.Join(absDir, ".agents", "skills")
	if !s.fs.IsDir(skillsDir) {
		return fmt.Errorf("governanca nao instalada em %s (pasta .agents/skills/ ausente)", absDir)
	}

	s.printer.Info("Removendo governanca de: %s", absDir)
	s.printer.Info("")

	removed := 0

	safeRm := func(target string) {
		if !s.fs.Exists(target) {
			return
		}
		if dryRun {
			s.printer.DryRun("rm %s", target)
		} else {
			_ = s.fs.RemoveAll(target)
		}
		removed++
	}

	safeRmdirIfEmpty := func(dir string) {
		if !s.fs.IsDir(dir) || dryRun {
			return
		}
		entries, err := s.fs.ReadDir(dir)
		if err != nil || len(entries) > 0 {
			return
		}
		_ = s.fs.RemoveAll(dir)
	}

	// Skills (symlinks ou copias)
	if s.fs.IsDir(skillsDir) {
		entries, _ := s.fs.ReadDir(skillsDir)
		for _, e := range entries {
			safeRm(filepath.Join(skillsDir, e.Name()))
		}
	}
	safeRmdirIfEmpty(skillsDir)
	safeRmdirIfEmpty(filepath.Join(absDir, ".agents"))

	// Claude
	claudeSkills := filepath.Join(absDir, ".claude", "skills")
	if s.fs.IsDir(claudeSkills) {
		entries, _ := s.fs.ReadDir(claudeSkills)
		for _, e := range entries {
			safeRm(filepath.Join(claudeSkills, e.Name()))
		}
	}
	safeRmdirIfEmpty(claudeSkills)

	claudeAgents := filepath.Join(absDir, ".claude", "agents")
	if s.fs.IsDir(claudeAgents) {
		entries, _ := s.fs.ReadDir(claudeAgents)
		for _, e := range entries {
			if strings.HasSuffix(e.Name(), ".md") {
				safeRm(filepath.Join(claudeAgents, e.Name()))
			}
		}
	}
	safeRmdirIfEmpty(claudeAgents)

	safeRm(filepath.Join(absDir, ".claude", "rules", "governance.md"))
	safeRmdirIfEmpty(filepath.Join(absDir, ".claude", "rules"))

	safeRm(filepath.Join(absDir, ".claude", "scripts", "validate-task-evidence.sh"))
	safeRmdirIfEmpty(filepath.Join(absDir, ".claude", "scripts"))

	safeRm(filepath.Join(absDir, ".claude", "hooks", "validate-governance.sh"))
	safeRm(filepath.Join(absDir, ".claude", "hooks", "validate-preload.sh"))
	safeRmdirIfEmpty(filepath.Join(absDir, ".claude", "hooks"))

	// Remove settings.local.json apenas quando ele coincide com o arquivo gerado pela CLI.
	settingsFile := filepath.Join(absDir, ".claude", "settings.local.json")
	if s.fs.Exists(settingsFile) {
		data, err := s.fs.ReadFile(settingsFile)
		if err == nil {
			content := string(data)
			if isGeneratedClaudeSettings(content) {
				safeRm(settingsFile)
			} else if strings.Contains(content, "validate-governance") || strings.Contains(content, "validate-preload") {
				s.printer.Warn(".claude/settings.local.json contem configuracoes alem dos hooks de governanca — mantido.")
			}
		}
	}
	safeRmdirIfEmpty(filepath.Join(absDir, ".claude"))

	// Gemini
	geminiCmds := filepath.Join(absDir, ".gemini", "commands")
	if s.fs.IsDir(geminiCmds) {
		entries, _ := s.fs.ReadDir(geminiCmds)
		for _, e := range entries {
			if strings.HasSuffix(e.Name(), ".toml") {
				safeRm(filepath.Join(geminiCmds, e.Name()))
			}
		}
	}
	safeRmdirIfEmpty(geminiCmds)
	safeRmdirIfEmpty(filepath.Join(absDir, ".gemini"))

	// Codex
	safeRm(filepath.Join(absDir, ".codex", "config.toml"))
	safeRmdirIfEmpty(filepath.Join(absDir, ".codex"))

	// Shared helper scripts
	safeRm(filepath.Join(absDir, "scripts", "lib", "parse-hook-input.sh"))
	safeRmdirIfEmpty(filepath.Join(absDir, "scripts", "lib"))
	safeRmdirIfEmpty(filepath.Join(absDir, "scripts"))

	// GitHub/Copilot
	githubSkills := filepath.Join(absDir, ".github", "skills")
	if s.fs.IsDir(githubSkills) {
		entries, _ := s.fs.ReadDir(githubSkills)
		for _, e := range entries {
			safeRm(filepath.Join(githubSkills, e.Name()))
		}
	}
	safeRmdirIfEmpty(githubSkills)

	githubAgents := filepath.Join(absDir, ".github", "agents")
	if s.fs.IsDir(githubAgents) {
		entries, _ := s.fs.ReadDir(githubAgents)
		for _, e := range entries {
			if strings.HasSuffix(e.Name(), ".agent.md") {
				safeRm(filepath.Join(githubAgents, e.Name()))
			}
		}
	}
	safeRmdirIfEmpty(githubAgents)
	safeRm(filepath.Join(absDir, ".github", "copilot-instructions.md"))
	safeRmdirIfEmpty(filepath.Join(absDir, ".github"))

	// Root files
	safeRm(filepath.Join(absDir, "AGENTS.md"))
	safeRm(filepath.Join(absDir, "CLAUDE.md"))
	safeRm(filepath.Join(absDir, "GEMINI.md"))

	// Manifesto
	safeRm(filepath.Join(absDir, ".ai_spec_harness.json"))

	// Preservar AGENTS.local.md
	localFile := filepath.Join(absDir, "AGENTS.local.md")
	if s.fs.Exists(localFile) {
		s.printer.Info("")
		s.printer.Info("AGENTS.local.md preservado (extensao local do usuario).")
	}

	s.printer.Info("")
	if dryRun {
		s.printer.Info("[dry-run] %d arquivo(s) seriam removidos. Nenhuma alteracao feita.", removed)
	} else {
		s.printer.Info("Governanca removida: %d arquivo(s).", removed)
	}

	return nil
}

func isGeneratedClaudeSettings(content string) bool {
	normalize := func(v string) string {
		return strings.Join(strings.Fields(v), "")
	}
	return normalize(content) == normalize(defaultClaudeSettings())
}

func defaultClaudeSettings() string {
	return `{
  "hooks": {
    "PreToolUse": [
      {
        "matcher": "Edit|Write",
        "hooks": [
          {
            "type": "command",
            "command": "bash .claude/hooks/validate-preload.sh"
          }
        ]
      }
    ],
    "PostToolUse": [
      {
        "matcher": "Edit|Write",
        "hooks": [
          {
            "type": "command",
            "command": "bash .claude/hooks/validate-governance.sh"
          }
        ]
      }
    ]
  }
}
`
}
