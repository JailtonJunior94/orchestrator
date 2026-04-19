package install

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/JailtonJunior94/ai-spec-harness/internal/adapters"
	"github.com/JailtonJunior94/ai-spec-harness/internal/config"
	"github.com/JailtonJunior94/ai-spec-harness/internal/contextgen"
	"github.com/JailtonJunior94/ai-spec-harness/internal/embedded"
	"github.com/JailtonJunior94/ai-spec-harness/internal/fs"
	"github.com/JailtonJunior94/ai-spec-harness/internal/manifest"
	"github.com/JailtonJunior94/ai-spec-harness/internal/output"
	"github.com/JailtonJunior94/ai-spec-harness/internal/platform"
	"github.com/JailtonJunior94/ai-spec-harness/internal/skills"
	"github.com/JailtonJunior94/ai-spec-harness/internal/version"
)

// Service orquestra o fluxo de instalacao de governanca.
type Service struct {
	fs       fs.FileSystem
	printer  *output.Printer
	manifest *manifest.Store
	adapters *adapters.Generator
	ctxgen   *contextgen.Generator
}

func NewService(
	fsys fs.FileSystem,
	printer *output.Printer,
	mfst *manifest.Store,
	adpt *adapters.Generator,
	ctxg *contextgen.Generator,
) *Service {
	return &Service{
		fs:       fsys,
		printer:  printer,
		manifest: mfst,
		adapters: adpt,
		ctxgen:   ctxg,
	}
}

func (s *Service) Execute(opts config.InstallOptions) error {
	if err := s.validate(opts); err != nil {
		return err
	}

	// Se --source nao fornecido, extrair assets embutidos para temp dir.
	if opts.SourceDir == "" {
		tmpDir, cleanup, err := embedded.ExtractToTempDir()
		if err != nil {
			return fmt.Errorf("extrair assets embutidos: %w", err)
		}
		defer cleanup()
		opts.SourceDir = tmpDir
		// Modo embutido sempre usa copy (sem symlinks para temp dir)
		if opts.LinkMode == skills.LinkSymlink {
			opts.LinkMode = skills.LinkCopy
		}
	}

	sourceDir, err := filepath.Abs(opts.SourceDir)
	if err != nil {
		return fmt.Errorf("resolver caminho fonte: %w", err)
	}
	projectDir, err := filepath.Abs(opts.ProjectDir)
	if err != nil {
		return fmt.Errorf("resolver caminho projeto: %w", err)
	}

	if sourceDir == projectDir {
		return fmt.Errorf("o diretorio alvo nao pode ser o proprio repositorio de regras")
	}

	linkMode := opts.LinkMode
	if !platform.Current().SupportsSymlinks() && linkMode == skills.LinkSymlink {
		s.printer.Warn("Plataforma %s nao suporta symlinks nativamente, usando modo copy", platform.Current().OS)
		linkMode = skills.LinkCopy
	}

	allSkills := skills.AllSkills(opts.Langs)

	s.printer.Info("Ferramentas: %v", toolNames(opts.Tools))
	s.printer.Info("Linguagens:  %v", langNames(opts.Langs))
	s.printer.Info("")

	// 1. Instalar skills canonicas em .agents/skills/
	if err := s.installBaseSkills(sourceDir, projectDir, allSkills, linkMode, opts.DryRun); err != nil {
		return fmt.Errorf("instalar skills base: %w", err)
	}

	// 2. Instalar adaptadores por ferramenta
	for _, tool := range opts.Tools {
		if err := s.installTool(sourceDir, projectDir, tool, allSkills, linkMode, opts.DryRun, opts.CodexProfile); err != nil {
			return fmt.Errorf("instalar %s: %w", tool, err)
		}
	}

	// 3. Gerar governanca contextual
	if opts.GenerateCtx {
		s.printer.Step("Gerando governanca contextual...")
		if err := s.ctxgen.Generate(sourceDir, projectDir, opts.Tools, opts.Langs, opts.CodexProfile, opts.DryRun, opts.FocusPaths...); err != nil {
			s.printer.Warn("Falha ao gerar governanca contextual: %v", err)
		}
	}

	// 4. Persistir manifesto
	if !opts.DryRun {
		checksums := s.computeChecksums(sourceDir, allSkills)
		mf := &manifest.Manifest{
			Version:      version.Version,
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
			SourceDir:    sourceDir,
			LinkMode:     linkMode,
			Tools:        opts.Tools,
			Langs:        opts.Langs,
			Skills:       allSkills,
			Checksums:    checksums,
			CodexProfile: opts.CodexProfile,
		}
		if err := s.manifest.Save(projectDir, mf); err != nil {
			return fmt.Errorf("salvar manifesto: %w", err)
		}
	}

	if opts.DryRun {
		s.printer.Info("")
		s.printer.Info("[dry-run] Nenhum arquivo foi alterado.")
	} else {
		s.printer.Info("")
		s.printer.Info("Governanca para IA instalada em: %s", projectDir)
		s.printer.Info("Modo de instalacao: %s", linkMode)
	}

	return nil
}

func (s *Service) validate(opts config.InstallOptions) error {
	if opts.ProjectDir == "" {
		return fmt.Errorf("diretorio alvo e obrigatorio")
	}
	if !s.fs.IsDir(opts.ProjectDir) {
		return fmt.Errorf("diretorio alvo nao encontrado: %s", opts.ProjectDir)
	}
	if !s.fs.Writable(opts.ProjectDir) {
		return fmt.Errorf("sem permissao de escrita em: %s", opts.ProjectDir)
	}
	if opts.SourceDir != "" && !s.fs.IsDir(opts.SourceDir) {
		return fmt.Errorf("diretorio fonte nao encontrado: %s", opts.SourceDir)
	}
	if len(opts.Tools) == 0 {
		return fmt.Errorf("nenhuma ferramenta selecionada")
	}
	return nil
}

func (s *Service) installBaseSkills(sourceDir, projectDir string, skillList []string, mode skills.LinkMode, dryRun bool) error {
	s.printer.Step("Instalando skills canonicas...")
	skillsDir := filepath.Join(projectDir, ".agents", "skills")

	if dryRun {
		s.printer.DryRun("mkdir -p %s", skillsDir)
	} else {
		if err := s.fs.MkdirAll(skillsDir); err != nil {
			return err
		}
	}

	for _, skill := range skillList {
		src := filepath.Join(sourceDir, ".agents", "skills", skill)
		dst := filepath.Join(skillsDir, skill)

		if !s.fs.IsDir(src) {
			s.printer.Debug("Skill %s nao encontrada na fonte, pulando", skill)
			continue
		}

		if dryRun {
			s.printer.DryRun("link_or_copy %s -> %s", src, dst)
			continue
		}

		if err := s.linkOrCopy(src, dst, mode); err != nil {
			return fmt.Errorf("skill %s: %w", skill, err)
		}
		s.printer.Debug("Skill %s instalada", skill)
	}
	return nil
}

func (s *Service) installTool(sourceDir, projectDir string, tool skills.Tool, skillList []string, mode skills.LinkMode, dryRun bool, codexProfile string) error {
	s.printer.Step("Instalando %s...", tool)

	switch tool {
	case skills.ToolClaude:
		return s.installClaude(sourceDir, projectDir, skillList, mode, dryRun)
	case skills.ToolGemini:
		return s.installGemini(sourceDir, projectDir, dryRun)
	case skills.ToolCodex:
		return s.installCodex(projectDir, skillList, codexProfile, dryRun)
	case skills.ToolCopilot:
		return s.installCopilot(sourceDir, projectDir, skillList, mode, dryRun)
	}
	return nil
}

func (s *Service) installClaude(sourceDir, projectDir string, skillList []string, mode skills.LinkMode, dryRun bool) error {
	dirs := []string{
		filepath.Join(projectDir, ".claude", "skills"),
		filepath.Join(projectDir, ".claude", "agents"),
		filepath.Join(projectDir, ".claude", "rules"),
		filepath.Join(projectDir, ".claude", "scripts"),
		filepath.Join(projectDir, ".claude", "hooks"),
		filepath.Join(projectDir, "scripts", "lib"),
	}

	for _, d := range dirs {
		if dryRun {
			s.printer.DryRun("mkdir -p %s", d)
		} else if err := s.fs.MkdirAll(d); err != nil {
			return err
		}
	}

	// Symlinks para skills
	for _, skill := range skillList {
		src := filepath.Join(sourceDir, ".agents", "skills", skill)
		dst := filepath.Join(projectDir, ".claude", "skills", skill)
		relTarget := filepath.Join("..", "..", ".agents", "skills", skill)

		if !s.fs.IsDir(src) {
			continue
		}

		if dryRun {
			s.printer.DryRun("link %s -> %s", relTarget, dst)
			continue
		}

		if mode == skills.LinkCopy {
			if err := s.fs.RemoveAll(dst); err != nil {
				return err
			}
			if err := s.fs.CopyDir(src, dst); err != nil {
				return err
			}
		} else {
			if err := s.fs.Symlink(relTarget, dst); err != nil {
				return err
			}
		}
	}

	if dryRun {
		s.printer.DryRun("copiar .claude/rules/governance.md")
		s.printer.DryRun("copiar .claude/scripts/validate-task-evidence.sh")
		s.printer.DryRun("copiar .claude/scripts/validate-bugfix-evidence.sh")
		s.printer.DryRun("copiar .claude/scripts/validate-refactor-evidence.sh")
		s.printer.DryRun("copiar .claude/hooks/validate-governance.sh")
		s.printer.DryRun("copiar .claude/hooks/validate-preload.sh")
		s.printer.DryRun("copiar scripts/lib/parse-hook-input.sh")
		s.printer.DryRun("copiar scripts/lib/check-invocation-depth.sh")
		s.printer.DryRun("configurar hooks PreToolUse e PostToolUse em .claude/settings.local.json")
		s.printer.DryRun("gerar .claude/agents/*.md via adaptadores")
		return nil
	}

	rulesGov := filepath.Join(sourceDir, ".claude", "rules", "governance.md")
	if s.fs.Exists(rulesGov) {
		if err := s.fs.CopyFile(rulesGov, filepath.Join(projectDir, ".claude", "rules", "governance.md")); err != nil {
			return err
		}
	}

	validateScript := filepath.Join(sourceDir, ".claude", "scripts", "validate-task-evidence.sh")
	if s.fs.Exists(validateScript) {
		if err := s.fs.CopyFile(validateScript, filepath.Join(projectDir, ".claude", "scripts", "validate-task-evidence.sh")); err != nil {
			return err
		}
	}

	bugfixScript := filepath.Join(sourceDir, ".claude", "scripts", "validate-bugfix-evidence.sh")
	if s.fs.Exists(bugfixScript) {
		if err := s.fs.CopyFile(bugfixScript, filepath.Join(projectDir, ".claude", "scripts", "validate-bugfix-evidence.sh")); err != nil {
			return err
		}
	}

	refactorScript := filepath.Join(sourceDir, ".claude", "scripts", "validate-refactor-evidence.sh")
	if s.fs.Exists(refactorScript) {
		if err := s.fs.CopyFile(refactorScript, filepath.Join(projectDir, ".claude", "scripts", "validate-refactor-evidence.sh")); err != nil {
			return err
		}
	}

	hookDir := filepath.Join(projectDir, ".claude", "hooks")
	govHook := filepath.Join(sourceDir, ".claude", "hooks", "validate-governance.sh")
	if s.fs.Exists(govHook) {
		if err := s.fs.CopyFile(govHook, filepath.Join(hookDir, "validate-governance.sh")); err != nil {
			return err
		}
	}

	preloadHook := filepath.Join(sourceDir, ".claude", "hooks", "validate-preload.sh")
	if s.fs.Exists(preloadHook) {
		if err := s.fs.CopyFile(preloadHook, filepath.Join(hookDir, "validate-preload.sh")); err != nil {
			return err
		}
	}

	parseHookInput := filepath.Join(sourceDir, "scripts", "lib", "parse-hook-input.sh")
	if s.fs.Exists(parseHookInput) {
		if err := s.fs.CopyFile(parseHookInput, filepath.Join(projectDir, "scripts", "lib", "parse-hook-input.sh")); err != nil {
			return err
		}
	}

	depthGuard := filepath.Join(sourceDir, "scripts", "lib", "check-invocation-depth.sh")
	if s.fs.Exists(depthGuard) {
		if err := s.fs.CopyFile(depthGuard, filepath.Join(projectDir, "scripts", "lib", "check-invocation-depth.sh")); err != nil {
			return err
		}
	}

	agentsMD := filepath.Join(sourceDir, "AGENTS.md")
	if s.fs.Exists(agentsMD) {
		if err := s.fs.CopyFile(agentsMD, filepath.Join(projectDir, "AGENTS.md")); err != nil {
			return err
		}
	}

	settingsFile := filepath.Join(projectDir, ".claude", "settings.local.json")
	if !s.fs.Exists(settingsFile) {
		if err := s.fs.WriteFile(settingsFile, []byte(defaultClaudeSettings())); err != nil {
			return err
		}
	} else if data, err := s.fs.ReadFile(settingsFile); err == nil {
		content := string(data)
		if !strings.Contains(content, "validate-governance.sh") || !strings.Contains(content, "validate-preload.sh") {
			s.printer.Warn(".claude/settings.local.json ja existe. Adicione os hooks manualmente para validate-preload e validate-governance.")
		}
	}

	s.adapters.GenerateClaude(sourceDir, projectDir)
	return nil
}

func (s *Service) installGemini(sourceDir, projectDir string, dryRun bool) error {
	cmdDir := filepath.Join(projectDir, ".gemini", "commands")
	if dryRun {
		s.printer.DryRun("mkdir -p %s", cmdDir)
		s.printer.DryRun("gerar .gemini/commands/*.toml via adaptadores")
		s.printer.DryRun("copiar .gemini/hooks/validate-preload.sh")
		return nil
	}

	if err := s.fs.MkdirAll(cmdDir); err != nil {
		return err
	}
	s.adapters.GenerateGemini(sourceDir, projectDir)

	geminiPreload := filepath.Join(sourceDir, ".gemini", "hooks", "validate-preload.sh")
	if s.fs.Exists(geminiPreload) {
		hookDir := filepath.Join(projectDir, ".gemini", "hooks")
		if err := s.fs.MkdirAll(hookDir); err != nil {
			return err
		}
		if err := s.fs.CopyFile(geminiPreload, filepath.Join(hookDir, "validate-preload.sh")); err != nil {
			return err
		}
	}
	return nil
}

var codexPlanningSkills = map[string]bool{
	"analyze-project":               true,
	"create-prd":                    true,
	"create-technical-specification": true,
	"create-tasks":                  true,
}

func filterCodexSkills(skillList []string) []string {
	out := make([]string, 0, len(skillList))
	for _, s := range skillList {
		if !codexPlanningSkills[s] {
			out = append(out, s)
		}
	}
	return out
}

func (s *Service) installCodex(projectDir string, skillList []string, codexProfile string, dryRun bool) error {
	codexDir := filepath.Join(projectDir, ".codex")
	if dryRun {
		s.printer.DryRun("mkdir -p %s", codexDir)
		s.printer.DryRun("gerar .codex/config.toml")
		return nil
	}

	if err := s.fs.MkdirAll(codexDir); err != nil {
		return err
	}

	list := skillList
	if codexProfile == "lean" {
		list = filterCodexSkills(skillList)
	}

	content := s.adapters.BuildCodexConfig(list)
	return s.fs.WriteFile(filepath.Join(codexDir, "config.toml"), []byte(content))
}

func (s *Service) installCopilot(sourceDir, projectDir string, skillList []string, mode skills.LinkMode, dryRun bool) error {
	dirs := []string{
		filepath.Join(projectDir, ".github", "skills"),
		filepath.Join(projectDir, ".github", "agents"),
	}

	for _, d := range dirs {
		if dryRun {
			s.printer.DryRun("mkdir -p %s", d)
		} else if err := s.fs.MkdirAll(d); err != nil {
			return err
		}
	}

	for _, skill := range skillList {
		src := filepath.Join(sourceDir, ".agents", "skills", skill)
		dst := filepath.Join(projectDir, ".github", "skills", skill)
		relTarget := filepath.Join("..", "..", ".agents", "skills", skill)

		if !s.fs.IsDir(src) {
			continue
		}

		if dryRun {
			s.printer.DryRun("link %s -> %s", relTarget, dst)
			continue
		}

		if mode == skills.LinkCopy {
			if err := s.fs.RemoveAll(dst); err != nil {
				return err
			}
			if err := s.fs.CopyDir(src, dst); err != nil {
				return err
			}
		} else {
			if err := s.fs.Symlink(relTarget, dst); err != nil {
				return err
			}
		}
	}

	if !dryRun {
		s.adapters.GenerateGitHub(sourceDir, projectDir)
	} else {
		s.printer.DryRun("gerar .github/agents/*.agent.md via adaptadores")
	}

	return nil
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

func (s *Service) linkOrCopy(src, dst string, mode skills.LinkMode) error {
	if mode == skills.LinkCopy {
		_ = s.fs.RemoveAll(dst)
		return s.fs.CopyDir(src, dst)
	}
	return s.fs.Symlink(src, dst)
}

func (s *Service) computeChecksums(sourceDir string, skillList []string) map[string]string {
	checksums := make(map[string]string)
	for _, skill := range skillList {
		skillMD := filepath.Join(sourceDir, ".agents", "skills", skill, "SKILL.md")
		hash, err := s.fs.FileHash(skillMD)
		if err != nil {
			continue
		}
		checksums[skill] = hash
	}
	return checksums
}

func toolNames(tools []skills.Tool) []string {
	out := make([]string, len(tools))
	for i, t := range tools {
		out[i] = string(t)
	}
	return out
}

func langNames(langs []skills.Lang) []string {
	out := make([]string, len(langs))
	for i, l := range langs {
		out[i] = string(l)
	}
	return out
}
