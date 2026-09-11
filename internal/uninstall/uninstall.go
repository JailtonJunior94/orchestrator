package uninstall

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/JailtonJunior94/ai-spec-harness/internal/fs"
	"github.com/JailtonJunior94/ai-spec-harness/internal/manifest"
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
//
// Fonte de verdade da remocao (RF-05, RF-60): o manifesto persistido pela
// instalacao. Quando ele rastreia arquivos individualmente (campo aditivo
// InstalledFiles, presente a partir desta tarefa), cada caminho listado e
// removido apos checar existencia — nunca um arquivo nao rastreado. Manifesto
// ausente ou anterior a este campo cai no caminho conservador anunciado: a
// lista fixa historica, preservando o comportamento pre-existente sem
// inventar remocao de arquivo desconhecido.
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

	// Skills e adaptadores gerados por skill: enumeracao dinamica do disco, nao
	// lista estatica — reflete o conjunto real de skills instaladas, inclusive
	// as adicionadas apos esta versao do harness.
	if s.fs.IsDir(skillsDir) {
		entries, _ := s.fs.ReadDir(skillsDir)
		for _, e := range entries {
			safeRm(filepath.Join(skillsDir, e.Name()))
		}
	}
	safeRmdirIfEmpty(skillsDir)
	safeRmdirIfEmpty(filepath.Join(absDir, ".agents"))

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

	codexAgents := filepath.Join(absDir, ".codex", "agents")
	if s.fs.IsDir(codexAgents) {
		entries, _ := s.fs.ReadDir(codexAgents)
		for _, e := range entries {
			if strings.HasSuffix(e.Name(), ".toml") {
				safeRm(filepath.Join(codexAgents, e.Name()))
			}
		}
	}
	safeRmdirIfEmpty(codexAgents)

	// Arquivos estaticos (hooks, scripts, libs, configs nucleo por ferramenta):
	// fonte de verdade e o manifesto.
	mfst := manifest.NewStore(s.fs)
	mf, loadErr := mfst.Load(absDir)
	if loadErr == nil && mf.HasFileTracking() {
		for _, rel := range mf.InstalledFiles {
			safeRm(filepath.Join(absDir, rel))
		}
	} else {
		s.printer.Warn("manifesto sem rastreamento por arquivo (versao anterior a esta funcionalidade, ou ausente) — removendo apenas o conjunto conservador conhecido; reinstale para habilitar remocao completa por arquivo.")
		s.legacyStaticRemoval(absDir, safeRm)
	}

	for _, dir := range []string{
		filepath.Join(absDir, ".claude", "hooks"),
		filepath.Join(absDir, ".claude", "scripts"),
		filepath.Join(absDir, ".claude", "rules"),
		filepath.Join(absDir, ".codex", "hooks"),
		filepath.Join(absDir, ".codex"),
		filepath.Join(absDir, ".github", "hooks"),
		filepath.Join(absDir, "scripts", "lib"),
		filepath.Join(absDir, "scripts"),
		filepath.Join(absDir, ".opencode", "plugin"),
		filepath.Join(absDir, ".opencode"),
		filepath.Join(absDir, ".agents", "scripts"),
		filepath.Join(absDir, ".agents", "lib"),
		filepath.Join(absDir, ".agents", "hooks"),
	} {
		safeRmdirIfEmpty(dir)
	}

	// Remove settings.local.json apenas quando ele coincide com o arquivo gerado pela CLI.
	settingsFile := filepath.Join(absDir, ".claude", "settings.local.json")
	if s.fs.Exists(settingsFile) {
		data, err := s.fs.ReadFile(settingsFile)
		if err == nil {
			content := string(data)
			if s.isGeneratedClaudeSettings(content) {
				safeRm(settingsFile)
			} else if strings.Contains(content, "validate-governance") || strings.Contains(content, "validate-preload") {
				s.printer.Warn(".claude/settings.local.json contem configuracoes alem dos hooks de governanca — mantido.")
			}
		}
	}
	safeRmdirIfEmpty(filepath.Join(absDir, ".claude"))
	safeRmdirIfEmpty(filepath.Join(absDir, ".github"))

	// Residuo legado de projetos que instalaram o agente removido (Gemini,
	// tarefa 10.0): limpo incondicionalmente, independente do manifesto, para
	// que a desinstalacao tambem sirva de migracao.
	safeRm(filepath.Join(absDir, "GEMINI.md"))
	geminiDir := filepath.Join(absDir, ".gemini")
	if s.fs.IsDir(geminiDir) {
		if dryRun {
			s.printer.DryRun("rm -r %s (residuo legado do agente removido)", geminiDir)
			removed++
		} else {
			_ = s.fs.RemoveAll(geminiDir)
			removed++
		}
	}

	// Root files
	safeRm(filepath.Join(absDir, "AGENTS.md"))
	safeRm(filepath.Join(absDir, "CLAUDE.md"))

	// Manifesto
	safeRm(filepath.Join(absDir, manifest.ManifestFile))

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

// legacyStaticRemoval reproduz a lista fixa historica de arquivos estaticos
// removidos pela desinstalacao, usada exclusivamente quando o manifesto nao
// rastreia arquivos individualmente (ausente ou anterior a este campo). E
// deliberadamente a mesma lista conhecidamente incompleta de antes desta
// tarefa (RF-60 documentou a lacuna) — o caminho de rastreamento por manifesto
// e que a corrige; este caminho e o fallback conservador anunciado.
func (s *Service) legacyStaticRemoval(absDir string, safeRm func(string)) {
	safeRm(filepath.Join(absDir, ".claude", "rules", "governance.md"))
	safeRm(filepath.Join(absDir, ".claude", "scripts", "validate-task-evidence.sh"))
	safeRm(filepath.Join(absDir, ".claude", "scripts", "validate-bugfix-evidence.sh"))
	safeRm(filepath.Join(absDir, ".claude", "scripts", "validate-refactor-evidence.sh"))
	safeRm(filepath.Join(absDir, ".claude", "hooks", "validate-governance.sh"))
	safeRm(filepath.Join(absDir, ".claude", "hooks", "validate-preload.sh"))
	safeRm(filepath.Join(absDir, ".codex", "config.toml"))
	safeRm(filepath.Join(absDir, "scripts", "lib", "parse-hook-input.sh"))
	safeRm(filepath.Join(absDir, "scripts", "lib", "check-invocation-depth.sh"))
	safeRm(filepath.Join(absDir, ".github", "copilot-instructions.md"))
}

func (s *Service) isGeneratedClaudeSettings(content string) bool {
	normalize := func(v string) string {
		return strings.Join(strings.Fields(v), "")
	}
	return normalize(content) == normalize(s.defaultClaudeSettings())
}

func (s *Service) defaultClaudeSettings() string {
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
