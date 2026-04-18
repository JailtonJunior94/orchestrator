package upgrade

import (
	"fmt"
	"path/filepath"
	"time"

	"github.com/JailtonJunior94/ai-spec-harness/internal/adapters"
	"github.com/JailtonJunior94/ai-spec-harness/internal/config"
	"github.com/JailtonJunior94/ai-spec-harness/internal/contextgen"
	"github.com/JailtonJunior94/ai-spec-harness/internal/fs"
	"github.com/JailtonJunior94/ai-spec-harness/internal/manifest"
	"github.com/JailtonJunior94/ai-spec-harness/internal/output"
	"github.com/JailtonJunior94/ai-spec-harness/internal/skills"
)

// SkillStatus representa o resultado da verificacao de uma skill.
type SkillStatus int

const (
	StatusOK SkillStatus = iota
	StatusOutdated
	StatusMissing
	StatusContentDivergent
	StatusRefsDivergent
	StatusNoVersion
)

// SkillCheck armazena o resultado da verificacao de uma skill.
type SkillCheck struct {
	Name          string
	Status        SkillStatus
	SourceVersion string
	TargetVersion string
}

// Service orquestra o fluxo de upgrade de governanca.
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

func (s *Service) Execute(opts config.UpgradeOptions) error {
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

	if !s.fs.IsDir(filepath.Join(projectDir, ".agents", "skills")) {
		return fmt.Errorf("governanca nao instalada em %s (pasta .agents/skills/ ausente). Execute ai-spec-harness install primeiro", projectDir)
	}

	s.printer.Info("Verificando skills em: %s", projectDir)
	s.printer.Info("Fonte: %s", sourceDir)
	s.printer.Info("")

	// Verificar cada skill
	checks := s.checkSkills(sourceDir, projectDir, opts.Langs)

	// Contadores
	var okCount, outdatedCount, missingCount, refsDivCount int
	for _, c := range checks {
		switch c.Status {
		case StatusOK:
			okCount++
		case StatusOutdated, StatusContentDivergent, StatusNoVersion:
			outdatedCount++
		case StatusMissing:
			missingCount++
		case StatusRefsDivergent:
			refsDivCount++
			outdatedCount++
		}
	}

	// Imprimir resultados
	for _, c := range checks {
		switch c.Status {
		case StatusOK:
			s.printer.Status("OK", c.Name, c.TargetVersion)
		case StatusOutdated:
			s.printer.Status("DESATUALIZADA", c.Name, fmt.Sprintf("fonte: %s, alvo: %s", c.SourceVersion, c.TargetVersion))
		case StatusContentDivergent:
			s.printer.Status("CONTEUDO DIVERGENTE", c.Name, fmt.Sprintf("%s, checksum diferente", c.TargetVersion))
		case StatusRefsDivergent:
			s.printer.Status("REFS DIVERGENTES", c.Name, fmt.Sprintf("%s, references/ checksum diferente", c.TargetVersion))
		case StatusMissing:
			s.printer.Status("AUSENTE", c.Name, fmt.Sprintf("fonte: %s", c.SourceVersion))
		case StatusNoVersion:
			s.printer.Status("SEM VERSAO", c.Name, fmt.Sprintf("fonte: %s, alvo: sem campo version", c.SourceVersion))
		}
	}

	s.printer.Info("")
	s.printer.Info("Resumo: %d atualizadas, %d desatualizadas (%d refs divergentes), %d ausentes",
		okCount, outdatedCount, refsDivCount, missingCount)

	if opts.CheckOnly {
		if outdatedCount+missingCount > 0 {
			s.printer.Info("")
			s.printer.Info("Execute sem --check para atualizar: ai-spec-harness upgrade %s --source %s", opts.ProjectDir, opts.SourceDir)
			return fmt.Errorf("%d skill(s) desatualizadas ou ausentes", outdatedCount+missingCount)
		}
		return nil
	}

	// Aplicar atualizacoes
	updated := 0
	for _, c := range checks {
		if c.Status == StatusOK {
			continue
		}
		if c.Status == StatusMissing {
			continue
		}

		skillDst := filepath.Join(projectDir, ".agents", "skills", c.Name)
		if s.fs.IsSymlink(skillDst) {
			s.printer.Debug("  %s: symlink detectado, pulando copia (atualiza automaticamente)", c.Name)
			continue
		}

		skillSrc := filepath.Join(sourceDir, ".agents", "skills", c.Name)
		_ = s.fs.RemoveAll(skillDst)
		if err := s.fs.CopyDir(skillSrc, skillDst); err != nil {
			s.printer.Warn("Falha ao atualizar %s: %v", c.Name, err)
			continue
		}
		s.printer.Info("    -> %s atualizado", c.Name)
		updated++
	}

	// Re-gerar adaptadores se houve atualizacoes
	if updated > 0 {
		s.regenerateAdapters(sourceDir, projectDir)
		s.regenerateGovernance(sourceDir, projectDir)
	}

	// Atualizar manifesto
	if updated > 0 && s.manifest.Exists(projectDir) {
		mf, err := s.manifest.Load(projectDir)
		if err == nil {
			mf.UpdatedAt = time.Now()
			allSkills := skills.AllSkills(mf.Langs)
			mf.Checksums = s.computeChecksums(sourceDir, allSkills)
			_ = s.manifest.Save(projectDir, mf)
		}
	}

	return nil
}

func (s *Service) checkSkills(sourceDir, projectDir string, langFilter []skills.Lang) []SkillCheck {
	var checks []SkillCheck

	sourceSkillsDir := filepath.Join(sourceDir, ".agents", "skills")
	entries, err := s.fs.ReadDir(sourceSkillsDir)
	if err != nil {
		return nil
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		skillName := entry.Name()

		if !shouldProcessSkill(skillName, langFilter) {
			continue
		}

		sourceSkillMD := filepath.Join(sourceSkillsDir, skillName, "SKILL.md")
		targetSkillMD := filepath.Join(projectDir, ".agents", "skills", skillName, "SKILL.md")

		sourceData, err := s.fs.ReadFile(sourceSkillMD)
		if err != nil {
			continue
		}
		sourceFM := skills.ParseFrontmatter(sourceData)
		if sourceFM.Version == "" {
			continue
		}

		if !s.fs.Exists(targetSkillMD) {
			checks = append(checks, SkillCheck{
				Name:          skillName,
				Status:        StatusMissing,
				SourceVersion: sourceFM.Version,
			})
			continue
		}

		targetData, _ := s.fs.ReadFile(targetSkillMD)
		targetFM := skills.ParseFrontmatter(targetData)

		if targetFM.Version == "" {
			checks = append(checks, SkillCheck{
				Name:          skillName,
				Status:        StatusNoVersion,
				SourceVersion: sourceFM.Version,
			})
			continue
		}

		if skills.SemverGreater(sourceFM.Version, targetFM.Version) {
			checks = append(checks, SkillCheck{
				Name:          skillName,
				Status:        StatusOutdated,
				SourceVersion: sourceFM.Version,
				TargetVersion: targetFM.Version,
			})
			continue
		}

		// Mesma versao — verificar checksum do SKILL.md
		sourceHash, _ := s.fs.FileHash(sourceSkillMD)
		targetHash, _ := s.fs.FileHash(targetSkillMD)
		if sourceHash != targetHash {
			checks = append(checks, SkillCheck{
				Name:          skillName,
				Status:        StatusContentDivergent,
				SourceVersion: sourceFM.Version,
				TargetVersion: targetFM.Version,
			})
			continue
		}

		// SKILL.md identico — verificar references/
		sourceRefsHash, _ := s.fs.DirHash(filepath.Join(sourceDir, ".agents", "skills", skillName, "references"))
		targetRefsHash, _ := s.fs.DirHash(filepath.Join(projectDir, ".agents", "skills", skillName, "references"))
		if sourceRefsHash != "" && sourceRefsHash != targetRefsHash {
			checks = append(checks, SkillCheck{
				Name:          skillName,
				Status:        StatusRefsDivergent,
				SourceVersion: sourceFM.Version,
				TargetVersion: targetFM.Version,
			})
			continue
		}

		checks = append(checks, SkillCheck{
			Name:          skillName,
			Status:        StatusOK,
			SourceVersion: sourceFM.Version,
			TargetVersion: targetFM.Version,
		})
	}

	return checks
}

func (s *Service) regenerateAdapters(sourceDir, projectDir string) {
	s.printer.Step("Re-gerando adaptadores...")

	if s.fs.IsDir(filepath.Join(projectDir, ".claude")) {
		s.adapters.GenerateClaude(sourceDir, projectDir)
	}
	if s.fs.IsDir(filepath.Join(projectDir, ".github")) {
		s.adapters.GenerateGitHub(sourceDir, projectDir)
	}
	if s.fs.IsDir(filepath.Join(projectDir, ".gemini")) {
		s.adapters.GenerateGemini(sourceDir, projectDir)
	}
}

func (s *Service) regenerateGovernance(sourceDir, projectDir string) {
	if !s.fs.Exists(filepath.Join(projectDir, "AGENTS.md")) {
		return
	}

	s.printer.Step("Re-gerando governanca contextual apos atualizacao de skills...")

	// Detectar ferramentas instaladas
	var tools []skills.Tool
	if s.fs.Exists(filepath.Join(projectDir, "CLAUDE.md")) {
		tools = append(tools, skills.ToolClaude)
	}
	if s.fs.Exists(filepath.Join(projectDir, "GEMINI.md")) {
		tools = append(tools, skills.ToolGemini)
	}
	if s.fs.Exists(filepath.Join(projectDir, ".codex", "config.toml")) {
		tools = append(tools, skills.ToolCodex)
	}
	if s.fs.Exists(filepath.Join(projectDir, ".github", "copilot-instructions.md")) {
		tools = append(tools, skills.ToolCopilot)
	}

	if err := s.ctxgen.Generate(sourceDir, projectDir, tools, nil); err != nil {
		s.printer.Warn("Falha ao re-gerar governanca contextual: %v", err)
	}
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

func shouldProcessSkill(skillName string, langFilter []skills.Lang) bool {
	if len(langFilter) == 0 {
		return true
	}

	langSkills := map[string]bool{
		"go-implementation":      true,
		"object-calisthenics-go": true,
		"node-implementation":    true,
		"python-implementation":  true,
	}

	if !langSkills[skillName] {
		return true // skill processual — sempre incluir
	}

	allowed := make(map[string]bool)
	for _, l := range langFilter {
		for _, s := range skills.LangSkills([]skills.Lang{l}) {
			allowed[s] = true
		}
	}
	return allowed[skillName]
}
