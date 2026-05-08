package upgrade

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
	"github.com/JailtonJunior94/ai-spec-harness/internal/skills"
	"github.com/JailtonJunior94/ai-spec-harness/internal/version"
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
	ChangedRefs   []string
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
	// Se --source nao fornecido, extrair assets embutidos para temp dir.
	if opts.SourceDir == "" {
		tmpDir, cleanup, err := embedded.ExtractToTempDir()
		if err != nil {
			return fmt.Errorf("extrair assets embutidos: %w", err)
		}
		defer cleanup()
		opts.SourceDir = tmpDir
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

	if !s.fs.IsDir(filepath.Join(projectDir, ".agents", "skills")) {
		return fmt.Errorf("governanca nao instalada em %s (pasta .agents/skills/ ausente). Execute ai-spec-harness install primeiro", projectDir)
	}

	s.printer.Info("Verificando skills em: %s", projectDir)
	s.printer.Info("Fonte: %s", sourceDir)
	sourceVersion := version.ResolveFromExecutable()
	s.printer.Info("ai-spec %s", sourceVersion)
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
			for _, changed := range c.ChangedRefs {
				s.printer.Info("    %s", changed)
			}
		case StatusMissing:
			s.printer.Status("AUSENTE", c.Name, fmt.Sprintf("fonte: %s", c.SourceVersion))
		case StatusNoVersion:
			s.printer.Status("SEM VERSAO", c.Name, fmt.Sprintf("fonte: %s, alvo: sem campo version", c.SourceVersion))
		}
	}

	// Verificar divergencia de schema de governanca
	schemaDivergent := s.checkSchemaDivergence(sourceDir, projectDir)
	if schemaDivergent {
		outdatedCount++
	}

	s.printer.Info("")
	s.printer.Info("Resumo: %d atualizadas, %d desatualizadas (%d refs divergentes), %d ausentes",
		okCount, outdatedCount, refsDivCount, missingCount)

	if opts.CheckOnly {
		s.printCheckVersionInfo(projectDir)
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

	// Determinar perfil codex a partir do manifesto
	codexProfile := "full"
	if s.manifest.Exists(projectDir) {
		if mf, err := s.manifest.Load(projectDir); err == nil && mf.CodexProfile != "" {
			codexProfile = mf.CodexProfile
		}
	}

	// Re-gerar adaptadores se houve atualizacoes
	if updated > 0 {
		s.regenerateAdapters(sourceDir, projectDir, codexProfile)
	}
	if updated > 0 || schemaDivergent {
		s.regenerateGovernance(sourceDir, projectDir, codexProfile)
	}

	// Atualizar manifesto
	if s.manifest.Exists(projectDir) {
		mf, err := s.manifest.Load(projectDir)
		if err == nil {
			currentVersion := version.ResolveFromExecutable()
			versionChanged := mf.Version != currentVersion
			if updated == 0 && !versionChanged {
				return nil
			}

			mf.UpdatedAt = time.Now()
			mf.Version = currentVersion
			allSkills := skills.AllSkills(mf.Langs)
			mf.Checksums = s.computeChecksums(sourceDir, allSkills)
			mf.SkillVersions = s.collectSkillVersions(sourceDir, allSkills)
			_ = s.manifest.Save(projectDir, mf)
		}
	}

	return nil
}

func (s *Service) printCheckVersionInfo(projectDir string) {
	if !s.manifest.Exists(projectDir) {
		return
	}

	mf, err := s.manifest.Load(projectDir)
	if err != nil || mf == nil {
		return
	}

	binaryVersion := version.ResolveFromExecutable()
	if !skills.IsValidSemver(mf.Version) || !skills.IsValidSemver(binaryVersion) {
		return
	}

	if mf.Version != binaryVersion {
		s.printer.Info("CLI: %s (manifesto: %s)", binaryVersion, mf.Version)
	}
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
				ChangedRefs:   s.refsChangedFiles(filepath.Join(sourceDir, ".agents", "skills", skillName, "references"), filepath.Join(projectDir, ".agents", "skills", skillName, "references")),
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

func (s *Service) collectSkillVersions(sourceDir string, skillNames []string) map[string]string {
	versions := make(map[string]string, len(skillNames))
	for _, name := range skillNames {
		path := filepath.Join(sourceDir, ".agents", "skills", name, "SKILL.md")
		data, err := s.fs.ReadFile(path)
		if err != nil {
			continue
		}

		fm := skills.ParseFrontmatter(data)
		if fm.Version != "" {
			versions[name] = fm.Version
		}
	}

	return versions
}

func (s *Service) regenerateAdapters(sourceDir, projectDir, codexProfile string) {
	s.printer.Step("Re-gerando adaptadores...")

	if s.fs.IsDir(filepath.Join(projectDir, ".claude")) {
		s.adapters.GenerateClaude(sourceDir, projectDir)
		s.syncFileIfPresent(
			filepath.Join(sourceDir, ".claude", "rules", "governance.md"),
			filepath.Join(projectDir, ".claude", "rules", "governance.md"),
		)
		s.syncFileIfPresent(
			filepath.Join(sourceDir, ".claude", "scripts", "validate-task-evidence.sh"),
			filepath.Join(projectDir, ".claude", "scripts", "validate-task-evidence.sh"),
		)
		s.syncFileIfPresent(
			filepath.Join(sourceDir, ".claude", "scripts", "validate-bugfix-evidence.sh"),
			filepath.Join(projectDir, ".claude", "scripts", "validate-bugfix-evidence.sh"),
		)
		s.syncFileIfPresent(
			filepath.Join(sourceDir, ".claude", "scripts", "validate-refactor-evidence.sh"),
			filepath.Join(projectDir, ".claude", "scripts", "validate-refactor-evidence.sh"),
		)
		s.syncFileIfPresent(
			filepath.Join(sourceDir, ".claude", "hooks", "validate-preload.sh"),
			filepath.Join(projectDir, ".claude", "hooks", "validate-preload.sh"),
		)
		s.syncFileIfPresent(
			filepath.Join(sourceDir, ".claude", "hooks", "validate-governance.sh"),
			filepath.Join(projectDir, ".claude", "hooks", "validate-governance.sh"),
		)
		s.syncFileIfPresent(
			filepath.Join(sourceDir, "scripts", "lib", "check-invocation-depth.sh"),
			filepath.Join(projectDir, "scripts", "lib", "check-invocation-depth.sh"),
		)
		s.syncFileIfPresent(
			filepath.Join(sourceDir, "scripts", "lib", "parse-hook-input.sh"),
			filepath.Join(projectDir, "scripts", "lib", "parse-hook-input.sh"),
		)
	}
	if s.fs.IsDir(filepath.Join(projectDir, ".github")) {
		s.adapters.GenerateGitHub(sourceDir, projectDir)
	}
	if s.fs.IsDir(filepath.Join(projectDir, ".gemini")) {
		s.adapters.GenerateGemini(sourceDir, projectDir)
		s.syncFileIfPresent(
			filepath.Join(sourceDir, ".gemini", "hooks", "validate-preload.sh"),
			filepath.Join(projectDir, ".gemini", "hooks", "validate-preload.sh"),
		)
		s.syncFileIfPresent(
			filepath.Join(sourceDir, ".gemini", "hooks", "validate-governance.sh"),
			filepath.Join(projectDir, ".gemini", "hooks", "validate-governance.sh"),
		)
	}
	if s.fs.Exists(filepath.Join(projectDir, ".codex", "config.toml")) {
		content := s.adapters.BuildCodexConfig(s.installedCodexSkills(projectDir, codexProfile))
		_ = s.fs.WriteFile(filepath.Join(projectDir, ".codex", "config.toml"), []byte(content))
		s.syncFileIfPresent(
			filepath.Join(sourceDir, ".codex", "hooks", "validate-preload.sh"),
			filepath.Join(projectDir, ".codex", "hooks", "validate-preload.sh"),
		)
	}
}

func (s *Service) regenerateGovernance(sourceDir, projectDir, codexProfile string) {
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

	if err := s.ctxgen.Generate(sourceDir, projectDir, tools, nil, codexProfile, false); err != nil {
		s.printer.Warn("Falha ao re-gerar governanca contextual: %v", err)
	}
}

// checkSchemaDivergence compara governance-schema no AGENTS.md do projeto
// com a versao esperada pela fonte, avisando se houve edicao manual.
func (s *Service) checkSchemaDivergence(sourceDir, projectDir string) bool {
	projectAgents := filepath.Join(projectDir, "AGENTS.md")
	if !s.fs.Exists(projectAgents) {
		return false
	}

	data, err := s.fs.ReadFile(projectAgents)
	if err != nil {
		return false
	}

	projectSchema := extractSchemaVersion(string(data))
	if projectSchema == "" {
		return false
	}

	sourceSchema := resolveSourceSchema(s.fs, sourceDir)

	if sourceSchema != "" && sourceSchema != projectSchema {
		s.printer.Status("SCHEMA DIVERGENTE", "AGENTS.md",
			fmt.Sprintf("projeto: %s, fonte: %s", projectSchema, sourceSchema))
		return true
	}

	return false
}

// resolveSourceSchema retorna a versao de schema esperada pela fonte.
// O template `agents-template.md` carrega `{{GOVERNANCE_SCHEMA_VERSION}}` como
// placeholder substituido em tempo de geracao, entao um valor com `{{` indica
// que devemos cair de volta para a constante autoritativa em contextgen.
func resolveSourceSchema(filesystem fs.FileSystem, sourceDir string) string {
	sourceTemplate := filepath.Join(sourceDir, ".agents", "skills", "analyze-project", "assets", "agents-template.md")
	if filesystem.Exists(sourceTemplate) {
		if data, err := filesystem.ReadFile(sourceTemplate); err == nil {
			if v := extractSchemaVersion(string(data)); v != "" && !strings.Contains(v, "{{") {
				return v
			}
		}
	}
	return contextgen.GovernanceSchemaVersion
}

// extractSchemaVersion extrai o valor de governance-schema do comentario HTML no topo de AGENTS.md.
func extractSchemaVersion(content string) string {
	for _, line := range strings.SplitN(content, "\n", 5) {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "<!-- governance-schema:") {
			v := strings.TrimPrefix(line, "<!-- governance-schema:")
			v = strings.TrimSuffix(v, "-->")
			return strings.TrimSpace(v)
		}
	}
	return ""
}

func (s *Service) refsChangedFiles(sourceRefs, targetRefs string) []string {
	if !s.fs.IsDir(sourceRefs) || !s.fs.IsDir(targetRefs) {
		return nil
	}

	sourceFiles := s.collectRelativeFiles(sourceRefs)
	targetFiles := s.collectRelativeFiles(targetRefs)

	targetSet := make(map[string]bool, len(targetFiles))
	for _, rel := range targetFiles {
		targetSet[rel] = true
	}

	var changed []string
	for _, rel := range sourceFiles {
		sourcePath := filepath.Join(sourceRefs, rel)
		targetPath := filepath.Join(targetRefs, rel)

		if !targetSet[rel] {
			changed = append(changed, "+ "+rel+" (novo)")
			continue
		}

		sourceHash, err1 := s.fs.FileHash(sourcePath)
		targetHash, err2 := s.fs.FileHash(targetPath)
		if err1 == nil && err2 == nil && sourceHash != targetHash {
			changed = append(changed, "~ "+rel+" (modificado)")
		}
	}

	sourceSet := make(map[string]bool, len(sourceFiles))
	for _, rel := range sourceFiles {
		sourceSet[rel] = true
	}
	for _, rel := range targetFiles {
		if !sourceSet[rel] {
			changed = append(changed, "- "+rel+" (removido)")
		}
	}

	return changed
}

func (s *Service) collectRelativeFiles(root string) []string {
	if !s.fs.IsDir(root) {
		return nil
	}

	var files []string
	var walk func(dir, prefix string)
	walk = func(dir, prefix string) {
		entries, err := s.fs.ReadDir(dir)
		if err != nil {
			return
		}
		for _, entry := range entries {
			name := entry.Name()
			fullPath := filepath.Join(dir, name)
			relPath := name
			if prefix != "" {
				relPath = filepath.Join(prefix, name)
			}
			if entry.IsDir() {
				walk(fullPath, relPath)
				continue
			}
			files = append(files, relPath)
		}
	}

	walk(root, "")
	return files
}

func (s *Service) syncFileIfPresent(src, dst string) {
	if !s.fs.Exists(src) {
		return
	}
	_ = s.fs.CopyFile(src, dst)
}

var upgradePlanningSkills = []string{
	"analyze-project",
	"create-prd",
	"create-technical-specification",
	"create-tasks",
}

func (s *Service) installedCodexSkills(projectDir, codexProfile string) []string {
	baseSkills := []string{"agent-governance", "bugfix", "review", "refactor", "execute-task"}

	if codexProfile != "lean" {
		baseSkills = append(baseSkills, upgradePlanningSkills...)
	}

	if s.fs.Exists(filepath.Join(projectDir, ".agents", "skills", "go-implementation", "SKILL.md")) {
		baseSkills = append(baseSkills, "go-implementation", "object-calisthenics-go")
	}
	if s.fs.Exists(filepath.Join(projectDir, ".agents", "skills", "node-implementation", "SKILL.md")) {
		baseSkills = append(baseSkills, "node-implementation")
	}
	if s.fs.Exists(filepath.Join(projectDir, ".agents", "skills", "python-implementation", "SKILL.md")) {
		baseSkills = append(baseSkills, "python-implementation")
	}

	return baseSkills
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
