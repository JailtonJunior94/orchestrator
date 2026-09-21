package upgrade

import (
	"errors"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/JailtonJunior94/ai-spec-harness/internal/adapters"
	"github.com/JailtonJunior94/ai-spec-harness/internal/batchreport"
	"github.com/JailtonJunior94/ai-spec-harness/internal/config"
	"github.com/JailtonJunior94/ai-spec-harness/internal/contextgen"
	"github.com/JailtonJunior94/ai-spec-harness/internal/embedded"
	"github.com/JailtonJunior94/ai-spec-harness/internal/fs"
	"github.com/JailtonJunior94/ai-spec-harness/internal/hooksync"
	"github.com/JailtonJunior94/ai-spec-harness/internal/manifest"
	"github.com/JailtonJunior94/ai-spec-harness/internal/output"
	"github.com/JailtonJunior94/ai-spec-harness/internal/skills"
	"github.com/JailtonJunior94/ai-spec-harness/internal/tracking"
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
		tmpDir, cleanup, err := embedded.NewExtractor().ExtractToTempDir()
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

	skillsDir := filepath.Join(projectDir, ".agents", "skills")
	if err := fs.RefuseExternalSymlink(s.fs, projectDir, skillsDir, opts.FollowExternalSymlinks); err != nil {
		return err
	}
	if !s.fs.IsDir(skillsDir) {
		return fmt.Errorf("governanca nao instalada em %s (pasta .agents/skills/ ausente). Execute ai-spec-harness install primeiro", projectDir)
	}

	s.printer.Info("Verificando skills em: %s", projectDir)
	s.printer.Info("Fonte: %s", sourceDir)
	sourceVersion := version.NewProvider().ResolveFromExecutable()
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

	copilotHooksObsolete := s.copilotGovernanceNeedsRepair(projectDir)
	if copilotHooksObsolete {
		s.printer.Status("DESATUALIZADO", CopilotGovernanceHooksRelPath, "gate de encerramento do Copilot com chave obsoleta ou ausente")
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

	var expectedChecksums map[string]string
	if s.manifest.Exists(projectDir) {
		if previous, err := s.manifest.Load(projectDir); err == nil {
			expectedChecksums = previous.FileChecksums
		}
	}

	tracker := tracking.NewTransactional(s.fs, projectDir, manifest.ManifestFile, expectedChecksums)
	previousFS, previousAdapters, previousCtxgen := s.fs, s.adapters, s.ctxgen
	s.fs = tracker
	s.adapters = adapters.NewGenerator(tracker, s.printer)
	s.ctxgen = contextgen.NewGenerator(tracker, s.printer)
	defer func() {
		s.fs = previousFS
		s.adapters = previousAdapters
		s.ctxgen = previousCtxgen
	}()

	if copilotHooksObsolete {
		repaired, err := NewHelper().RepairCopilotGovernanceHooks(s.fs, projectDir)
		if err != nil {
			s.printer.Warn("Falha ao reparar %s: %v", CopilotGovernanceHooksRelPath, err)
		} else if repaired {
			s.printer.Info("    -> %s reparado (chave obsoleta migrada para %q)", CopilotGovernanceHooksRelPath, CopilotSessionEndHookKey)
		}
	}

	if err := s.syncManagedArtifacts(sourceDir, projectDir); err != nil {
		s.printer.Warn("Falha ao sincronizar hooks/scripts/lib gerenciados: %v", err)
	}

	// Aplicar atualizacoes
	updated := 0
	for _, c := range checks {
		if c.Status == StatusOK {
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

	if len(tracker.Conflicts()) > 0 && !opts.OverwriteConflicts {
		return batchreport.ConflictError(tracker.Conflicts(), sourceDir)
	}
	if opts.OverwriteConflicts {
		tracker.AllowOverwrite()
	}
	if err := tracker.Commit(); err != nil {
		return fmt.Errorf("apply synchronization batch: %w", err)
	}

	batchreport.Print(s.printer, "upgrade", batchreport.Build(tracker))

	if s.manifest.Exists(projectDir) {
		mf, err := s.manifest.Load(projectDir)
		if err == nil {
			currentVersion := version.NewProvider().ResolveFromExecutable()

			mf.UpdatedAt = time.Now()
			mf.Version = currentVersion
			allSkills := skills.NewCatalog().AllSkills(mf.Langs)
			mf.Checksums = s.computeChecksums(sourceDir, allSkills)
			mf.SkillVersions = s.collectSkillVersions(sourceDir, allSkills)
			mergeFileTracking(mf, tracker, projectDir)
			_ = s.manifest.Save(projectDir, mf)
		}
	}

	return nil
}

func (s *Service) copilotGovernanceNeedsRepair(projectDir string) bool {
	path := NewHelper().CopilotGovernanceHooksPath(projectDir)
	if !s.fs.Exists(path) {
		return false
	}
	raw, err := s.fs.ReadFile(path)
	if err != nil {
		return false
	}
	_, changed, err := NewHelper().repairCopilotGovernanceDocument(raw)
	if err != nil {
		return false
	}
	return changed
}

func (s *Service) printCheckVersionInfo(projectDir string) {
	if !s.manifest.Exists(projectDir) {
		return
	}

	mf, err := s.manifest.Load(projectDir)
	if err != nil || mf == nil {
		return
	}

	binaryVersion := version.NewProvider().ResolveFromExecutable()
	if !skills.NewCatalog().IsValidSemver(mf.Version) || !skills.NewCatalog().IsValidSemver(binaryVersion) {
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

		if !NewHelper().shouldProcessSkill(skillName, langFilter) {
			continue
		}

		sourceSkillMD := filepath.Join(sourceSkillsDir, skillName, "SKILL.md")
		targetSkillMD := filepath.Join(projectDir, ".agents", "skills", skillName, "SKILL.md")

		sourceData, err := s.fs.ReadFile(sourceSkillMD)
		if err != nil {
			continue
		}
		sourceFM := skills.NewCatalog().ParseFrontmatter(sourceData)
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
		targetFM := skills.NewCatalog().ParseFrontmatter(targetData)

		if targetFM.Version == "" {
			checks = append(checks, SkillCheck{
				Name:          skillName,
				Status:        StatusNoVersion,
				SourceVersion: sourceFM.Version,
			})
			continue
		}

		if skills.NewCatalog().SemverGreater(sourceFM.Version, targetFM.Version) {
			checks = append(checks, SkillCheck{
				Name:          skillName,
				Status:        StatusOutdated,
				SourceVersion: sourceFM.Version,
				TargetVersion: targetFM.Version,
			})
			continue
		}

		sourceHash, sourceHashErr := s.fs.FileHash(sourceSkillMD)
		targetHash, targetHashErr := s.fs.FileHash(targetSkillMD)
		if sourceHashErr != nil || targetHashErr != nil || sourceHash != targetHash {
			checks = append(checks, SkillCheck{
				Name:          skillName,
				Status:        StatusContentDivergent,
				SourceVersion: sourceFM.Version,
				TargetVersion: targetFM.Version,
			})
			continue
		}

		sourceRefsDir := filepath.Join(sourceDir, ".agents", "skills", skillName, "references")
		targetRefsDir := filepath.Join(projectDir, ".agents", "skills", skillName, "references")
		sourceRefsHash, sourceRefsErr := s.fs.DirHash(sourceRefsDir)
		if sourceRefsErr != nil && !errors.Is(sourceRefsErr, os.ErrNotExist) {
			checks = append(checks, SkillCheck{
				Name:          skillName,
				Status:        StatusRefsDivergent,
				SourceVersion: sourceFM.Version,
				TargetVersion: targetFM.Version,
			})
			continue
		}
		if sourceRefsErr == nil {
			targetRefsHash, targetRefsErr := s.fs.DirHash(targetRefsDir)
			if targetRefsErr != nil || sourceRefsHash != targetRefsHash {
				checks = append(checks, SkillCheck{
					Name:          skillName,
					Status:        StatusRefsDivergent,
					SourceVersion: sourceFM.Version,
					TargetVersion: targetFM.Version,
					ChangedRefs:   s.refsChangedFiles(sourceRefsDir, targetRefsDir),
				})
				continue
			}
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

		fm := skills.NewCatalog().ParseFrontmatter(data)
		if fm.Version != "" {
			versions[name] = fm.Version
		}
	}

	return versions
}

func mergeFileTracking(mf *manifest.Manifest, tracker *tracking.Tracker, projectDir string) {
	legacyManifest := !mf.HasFileTracking()
	trackedInstalled := mf.InstalledFiles != nil
	trackedMerged := mf.MergedFiles != nil

	created := toPathSet(mf.InstalledFiles)
	merged := toPathSet(mf.MergedFiles)

	for _, path := range tracker.MergedPaths() {
		delete(created, path)
		merged[path] = true
		trackedMerged = true
	}
	for _, path := range tracker.CreatedPaths() {
		if !merged[path] {
			created[path] = true
			trackedInstalled = true
		}
	}

	if trackedInstalled {
		mf.InstalledFiles = sortedPathSet(created)
	}
	if trackedMerged {
		mf.MergedFiles = sortedPathSet(merged)
	}

	newChecksums := tracker.Checksums()
	checksums := make(map[string]string, len(mf.FileChecksums)+len(newChecksums))
	maps.Copy(checksums, mf.FileChecksums)
	maps.Copy(checksums, newChecksums)

	managedPaths := make([]string, 0, len(mf.InstalledFiles)+len(mf.MergedFiles))
	managedPaths = append(managedPaths, mf.InstalledFiles...)
	managedPaths = append(managedPaths, mf.MergedFiles...)
	if legacyManifest {
		managedPaths = append(managedPaths, legacyManagedPathCandidates(tracker, projectDir)...)
	}
	mf.FileChecksums = tracking.BackfillChecksums(tracker, projectDir, managedPaths, checksums)
}

func legacyManagedPathCandidates(fsys fs.FileSystem, projectDir string) []string {
	var out []string
	for _, dir := range legacyManagedRootDirs {
		abs := filepath.Join(projectDir, dir)
		if fsys.IsDir(abs) {
			out = append(out, tracking.WalkManagedPaths(fsys, projectDir, dir)...)
		}
	}
	for _, name := range legacyManagedRootFiles {
		if fsys.Exists(filepath.Join(projectDir, name)) {
			out = append(out, name)
		}
	}
	return out
}

var legacyManagedRootDirs = []string{
	filepath.Join(".agents", "skills"),
	filepath.Join(".agents", "hooks"),
	filepath.Join(".agents", "scripts"),
	filepath.Join(".agents", "lib"),
	filepath.Join(".agents", "policies"),
	".claude",
	".codex",
	".github",
	".opencode",
	filepath.Join("scripts", "lib"),
}

var legacyManagedRootFiles = []string{
	"AGENTS.md",
	"CLAUDE.md",
	"CODEX.md",
	"COPILOT.md",
}

func toPathSet(paths []string) map[string]bool {
	set := make(map[string]bool, len(paths))
	for _, path := range paths {
		set[path] = true
	}
	return set
}

func sortedPathSet(set map[string]bool) []string {
	out := make([]string, 0, len(set))
	for path := range set {
		out = append(out, path)
	}
	sort.Strings(out)
	return out
}

func (s *Service) regenerateAdapters(sourceDir, projectDir, codexProfile string) {
	s.printer.Step("Re-gerando adaptadores...")

	if s.fs.IsDir(filepath.Join(projectDir, ".claude")) {
		s.adapters.GenerateClaude(sourceDir, projectDir)
		for _, ruleFile := range skills.UniversalRuleFiles {
			s.syncFileIfPresent(
				filepath.Join(sourceDir, ".claude", "rules", ruleFile),
				filepath.Join(projectDir, ".claude", "rules", ruleFile),
			)
		}
		for _, script := range []string{
			"validate-task-evidence.sh",
			"validate-bugfix-evidence.sh",
			"validate-refactor-evidence.sh",
			"validate-review-evidence.sh",
		} {
			s.syncFileIfPresent(
				filepath.Join(sourceDir, ".claude", "scripts", script),
				filepath.Join(projectDir, ".claude", "scripts", script),
			)
		}
		for _, lib := range hooksync.AgentsLibFiles {
			s.syncFileIfPresent(
				filepath.Join(sourceDir, "scripts", "lib", lib),
				filepath.Join(projectDir, "scripts", "lib", lib),
			)
		}
	}
	if s.fs.IsDir(filepath.Join(projectDir, ".github")) {
		s.adapters.GenerateGitHub(sourceDir, projectDir)
	}
	if s.fs.Exists(filepath.Join(projectDir, ".codex", "config.toml")) {
		content := s.adapters.BuildCodexConfig(s.installedCodexSkills(projectDir, codexProfile))
		_ = s.fs.WriteFile(filepath.Join(projectDir, ".codex", "config.toml"), []byte(content))
	}
}

// syncManagedArtifacts sincroniza hooks/scripts/lib gerenciados (orquestrador,
// validadores de tool e vendor canonico .agents/) com paridade real entre os 4
// tools, espelhando exatamente o conjunto que `install` escreve na primeira
// instalacao. Roda sempre — independente de haver skill desatualizada — porque
// esses artefatos tem ciclo de vida proprio (nao acompanham o campo `version`
// do frontmatter de skill) e `verify` os compara por hash contra a fonte.
func (s *Service) syncManagedArtifacts(sourceDir, projectDir string) error {
	var toolHookDirs []string
	if s.fs.IsDir(filepath.Join(projectDir, ".claude")) {
		toolHookDirs = append(toolHookDirs, filepath.Join(".claude", "hooks"))
	}
	if s.fs.Exists(filepath.Join(projectDir, ".codex", "config.toml")) {
		toolHookDirs = append(toolHookDirs, filepath.Join(".codex", "hooks"))
	}
	if s.fs.IsDir(filepath.Join(projectDir, ".github")) {
		toolHookDirs = append(toolHookDirs, filepath.Join(".github", "hooks"))
	}
	toolHookDirs = append(toolHookDirs, filepath.Join(".agents", "hooks"))

	return hooksync.SyncAll(s.fs, sourceDir, projectDir, toolHookDirs)
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

	projectSchema := NewHelper().extractSchemaVersion(string(data))
	if projectSchema == "" {
		return false
	}

	sourceSchema := NewHelper().resolveSourceSchema(s.fs, sourceDir)

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
func (r1 *Helper) resolveSourceSchema(filesystem fs.FileSystem, sourceDir string) string {
	sourceTemplate := filepath.Join(sourceDir, ".agents", "skills", "analyze-project", "assets", "agents-template.md")
	if filesystem.Exists(sourceTemplate) {
		if data, err := filesystem.ReadFile(sourceTemplate); err == nil {
			if v := NewHelper().extractSchemaVersion(string(data)); v != "" && !strings.Contains(v, "{{") {
				return v
			}
		}
	}
	return contextgen.GovernanceSchemaVersion
}

// extractSchemaVersion extrai o valor de governance-schema do comentario HTML no topo de AGENTS.md.
func (r1 *Helper) extractSchemaVersion(content string) string {
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
		if err1 != nil || err2 != nil || sourceHash != targetHash {
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

var _upgradePlanningSkills = []string{
	"analyze-project",
	"create-prd",
	"create-technical-specification",
	"create-tasks",
}

func (s *Service) installedCodexSkills(projectDir, codexProfile string) []string {
	baseSkills := []string{"agent-governance", "bugfix", "review", "refactor", "execute-task", "execute-all-tasks"}

	if codexProfile != "lean" {
		baseSkills = append(baseSkills, _upgradePlanningSkills...)
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
	if s.fs.Exists(filepath.Join(projectDir, ".agents", "skills", "dotnet-csharp-implementation", "SKILL.md")) {
		baseSkills = append(baseSkills, "dotnet-csharp-implementation")
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

func (r1 *Helper) shouldProcessSkill(skillName string, langFilter []skills.Lang) bool {
	if len(langFilter) == 0 {
		return true
	}

	langSkills := map[string]bool{
		"go-implementation":            true,
		"object-calisthenics-go":       true,
		"node-implementation":          true,
		"python-implementation":        true,
		"dotnet-csharp-implementation": true,
	}

	if !langSkills[skillName] {
		return true // skill processual — sempre incluir
	}

	allowed := make(map[string]bool)
	for _, l := range langFilter {
		for _, s := range skills.NewCatalog().LangSkills([]skills.Lang{l}) {
			allowed[s] = true
		}
	}
	return allowed[skillName]
}
