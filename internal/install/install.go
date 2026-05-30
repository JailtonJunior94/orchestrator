package install

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/JailtonJunior94/ai-spec-harness/internal/adapters"
	"github.com/JailtonJunior94/ai-spec-harness/internal/config"
	"github.com/JailtonJunior94/ai-spec-harness/internal/contextgen"
	"github.com/JailtonJunior94/ai-spec-harness/internal/detect"
	"github.com/JailtonJunior94/ai-spec-harness/internal/embedded"
	"github.com/JailtonJunior94/ai-spec-harness/internal/fs"
	"github.com/JailtonJunior94/ai-spec-harness/internal/manifest"
	"github.com/JailtonJunior94/ai-spec-harness/internal/output"
	"github.com/JailtonJunior94/ai-spec-harness/internal/platform"
	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/probe"
	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/specs"
	"github.com/JailtonJunior94/ai-spec-harness/internal/skills"
	"github.com/JailtonJunior94/ai-spec-harness/internal/upgrade"
	"github.com/JailtonJunior94/ai-spec-harness/internal/version"
)

// _probeTimeout e o timeout maximo por CLI para probe de binario ACP.
// Curto para nao violar RF-11 (bootstrap < 30s com N CLIs). ADR-024.
const _probeTimeout = 3 * time.Second

// VerifyState representa o estado de uma skill/agente apos verificacao.
// ADR-019: file-first (le hash do disco, nao recalcula da fonte).
type VerifyState string

const (
	// VerifyStateCurrent indica que o arquivo instalado corresponde ao esperado.
	VerifyStateCurrent VerifyState = "current"
	// VerifyStateMissing indica que o arquivo nao esta instalado.
	VerifyStateMissing VerifyState = "missing"
	// VerifyStateDrifted indica que o arquivo instalado diverge do esperado.
	VerifyStateDrifted VerifyState = "drifted"
)

// VerifyKind classifica o tipo de item verificado.
// ADR-024: "skill" para skills de governanca; "binary" para binario ACP por CLI.
type VerifyKind string

const (
	// VerifyKindSkill indica verificacao de uma skill de governanca.
	VerifyKindSkill VerifyKind = "skill"
	// VerifyKindBinary indica verificacao de disponibilidade do binario ACP por CLI.
	VerifyKindBinary VerifyKind = "binary"
)

// VerifyItem representa o resultado de verificacao de uma skill ou binario para um agente.
// ADR-024: campo Kind distingue skill vs binario ACP.
type VerifyItem struct {
	Tool  skills.Tool
	Skill string
	State VerifyState
	Kind  VerifyKind // "skill" (default) ou "binary" (ADR-024 RI-04)
}

// Service orquestra o fluxo de instalacao de governanca.
type Service struct {
	fs          fs.FileSystem
	printer     *output.Printer
	manifest    *manifest.Store
	adapters    *adapters.Generator
	ctxgen      *contextgen.Generator
	agentDetect detect.AgentDetector // para auto-deteccao quando Tools vazio (ADR-019)
	langDetect  detect.Detector      // para deteccao de stack quando Langs vazio (ADR-024 RI-01)
	lookPather  probe.LookPather     // para probe de binario ACP (ADR-024 RI-02/RI-04)
}

func NewService(
	fsys fs.FileSystem,
	printer *output.Printer,
	mfst *manifest.Store,
	adpt *adapters.Generator,
	ctxg *contextgen.Generator,
) *Service {
	return &Service{
		fs:          fsys,
		printer:     printer,
		manifest:    mfst,
		adapters:    adpt,
		ctxgen:      ctxg,
		agentDetect: detect.NewBinaryAgentDetector(detect.OSLookPather{}, detect.OSHomeDir{}, detect.NewFileDetector(fsys)),
		langDetect:  detect.NewFileDetector(fsys),
		lookPather:  probe.NewCatalog().OsLookPather(),
	}
}

// NewServiceWithDetector cria um Service com AgentDetector customizado (para testes).
func NewServiceWithDetector(
	fsys fs.FileSystem,
	printer *output.Printer,
	mfst *manifest.Store,
	adpt *adapters.Generator,
	ctxg *contextgen.Generator,
	det detect.AgentDetector,
) *Service {
	svc := NewService(fsys, printer, mfst, adpt, ctxg)
	svc.agentDetect = det
	return svc
}

// NewServiceWithOptions cria um Service com dependencias customizadas (para testes de integracao).
// Permite injetar AgentDetector, Detector de linguagem e LookPather.
func NewServiceWithOptions(
	fsys fs.FileSystem,
	printer *output.Printer,
	mfst *manifest.Store,
	adpt *adapters.Generator,
	ctxg *contextgen.Generator,
	det detect.AgentDetector,
	langDet detect.Detector,
	lp probe.LookPather,
) *Service {
	svc := NewService(fsys, printer, mfst, adpt, ctxg)
	svc.agentDetect = det
	svc.langDetect = langDet
	svc.lookPather = lp
	return svc
}

func (s *Service) Execute(opts config.InstallOptions) error {
	// Auto-deteccao quando Tools nao especificado (ADR-019 + RF-06).
	if len(opts.Tools) == 0 {
		detected, err := s.agentDetect.Detect(context.Background(), detect.DetectOptions{
			ProjectDir: opts.ProjectDir,
		})
		if err != nil {
			return fmt.Errorf("auto-detectar agentes: %w", err)
		}
		if len(detected) == 0 {
			s.printer.Warn("Nenhum agente detectado no ambiente. Use --tools para especificar explicitamente.")
			return nil
		}
		s.printer.Info("Agentes detectados automaticamente: %v", NewHelper().toolNames(detected))
		opts.Tools = detected
	}

	// RI-01: Derivar Langs de DetectLangs quando nao especificado explicitamente.
	// Permite install transparente sem flag --langs em repos Go/Node/Python.
	// ADR-024: stack-aware via FileDetector.
	if len(opts.Langs) == 0 {
		detectedLangs := s.langDetect.DetectLangs(opts.ProjectDir)
		if len(detectedLangs) > 0 {
			s.printer.Info("Linguagens detectadas automaticamente: %v", NewHelper().langNames(detectedLangs))
			opts.Langs = detectedLangs
		}
	}

	if err := s.validate(opts); err != nil {
		return err
	}

	// Se --source nao fornecido, extrair assets embutidos para temp dir.
	if opts.SourceDir == "" {
		tmpDir, cleanup, err := embedded.NewExtractor().ExtractToTempDir()
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
	// Resolver projectDir: escopo global usa ~/.aispec; projeto usa ProjectDir.
	// ADR-019, R-SEC-001: paths via os.UserHomeDir, nunca hardcoded.
	var projectDir string
	if opts.Scope == config.ScopeGlobal {
		globalDir, err := NewHelper().globalInstallDir()
		if err != nil {
			return fmt.Errorf("escopo global: %w", err)
		}
		projectDir = globalDir
		// Criar dir global se nao existir.
		if err := s.fs.MkdirAll(projectDir); err != nil {
			return fmt.Errorf("criar diretorio global %s: %w", projectDir, err)
		}
		s.printer.Info("Escopo: global (%s)", projectDir)
	} else {
		abs, err := filepath.Abs(opts.ProjectDir)
		if err != nil {
			return fmt.Errorf("resolver caminho projeto: %w", err)
		}
		projectDir = abs
	}

	if sourceDir == projectDir {
		return fmt.Errorf("o diretorio alvo nao pode ser o proprio repositorio de regras")
	}

	linkMode := opts.LinkMode
	plat := platform.NewDetector().Current()
	if !plat.SupportsSymlinks() && linkMode == skills.LinkSymlink {
		s.printer.Warn("Plataforma %s nao suporta symlinks nativamente, usando modo copy", plat.OS)
		linkMode = skills.LinkCopy
	}

	allSkills := skills.NewCatalog().AllSkills(opts.Langs)

	s.printer.Info("Ferramentas: %v", NewHelper().toolNames(opts.Tools))
	s.printer.Info("Linguagens:  %v", NewHelper().langNames(opts.Langs))
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

	// 2.25 RI-02: Probe de binario ACP por CLI (warning nao-fatal).
	// Executado apos instalar adaptadores, antes de hooks e manifesto.
	// Timeout curto (3s por CLI) para nao violar RF-11 (bootstrap < 30s).
	// ADR-024: ausencia = Warn, install nao aborta.
	if !opts.DryRun {
		s.probeBinariesWarn(opts.Tools)
	}

	// 2.5 Instalar hooks canonicos do orquestrador em .agents/hooks/ (source-of-truth).
	if !opts.DryRun {
		if err := s.copyOrchestratorHooks(sourceDir, projectDir, filepath.Join(".agents", "hooks")); err != nil {
			s.printer.Warn("falha ao copiar hooks canonicos: %v", err)
		}
	}

	// 2.6 Instalar vendor canonico .agents/lib/ (libs shell consumidas pelos
	// hooks e skills com fallback cascata .agents/lib/ -> scripts/lib/).
	// Mantem paridade entre vendor canonico e mirror legado em scripts/lib/.
	if !opts.DryRun {
		if err := s.copyAgentsLib(sourceDir, projectDir); err != nil {
			s.printer.Warn("falha ao copiar .agents/lib/: %v", err)
		}
	}

	// 2.7 Instalar validadores de evidencia canonicos em .agents/scripts/ (tool-neutro,
	// SEMPRE). Garante paridade cross-CLI: a cascata `.agents/scripts/` -> `.claude/scripts/`
	// -> `scripts/` resolve os gates mesmo em projetos sem Claude (so Gemini/Codex/Copilot).
	if !opts.DryRun {
		if err := s.copyAgentsScripts(sourceDir, projectDir); err != nil {
			s.printer.Warn("falha ao copiar .agents/scripts/: %v", err)
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
			Version:       version.NewProvider().ResolveFromExecutable(),
			CreatedAt:     time.Now(),
			UpdatedAt:     time.Now(),
			SourceDir:     sourceDir,
			LinkMode:      linkMode,
			Tools:         opts.Tools,
			Langs:         opts.Langs,
			Skills:        allSkills,
			Checksums:     checksums,
			CodexProfile:  opts.CodexProfile,
			SkillVersions: s.collectSkillVersions(sourceDir, allSkills),
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

// globalInstallDir retorna o diretorio de instalacao global (~/.aispec).
// Usa os.UserHomeDir para seguranca (R-SEC-001); retorna erro explicito quando
// $HOME ausente (ex: CI sem home configurado).
func (r1 *Helper) globalInstallDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("$HOME nao disponivel (necessario para escopo global): %w", err)
	}
	return filepath.Join(home, ".aispec"), nil
}

// Verify verifica o estado de instalacao das skills para cada ferramenta configurada.
// Reusa o comparador de checksum do internal/upgrade (file-first: le hash do disco).
// ADR-019: StatusOK->current, StatusMissing->missing, StatusOutdated|ContentDivergent->drifted.
//
// opts.Scope determina o diretorio de instalacao (project ou global).
// opts.SourceDir e usado como fonte; se vazio, assets embutidos sao extraidos.
func (s *Service) Verify(opts config.InstallOptions) ([]VerifyItem, error) {
	// Resolver diretorio de instalacao conforme escopo.
	var installDir string
	if opts.Scope == config.ScopeGlobal {
		globalDir, err := NewHelper().globalInstallDir()
		if err != nil {
			return nil, fmt.Errorf("escopo global: %w", err)
		}
		installDir = globalDir
	} else {
		abs, err := filepath.Abs(opts.ProjectDir)
		if err != nil {
			return nil, fmt.Errorf("resolver caminho projeto: %w", err)
		}
		installDir = abs
	}

	// Resolver diretorio fonte (ou extrair embutido).
	sourceDir := opts.SourceDir
	var cleanup func()
	if sourceDir == "" {
		tmpDir, cl, err := embedded.NewExtractor().ExtractToTempDir()
		if err != nil {
			return nil, fmt.Errorf("extrair assets embutidos: %w", err)
		}
		cleanup = cl
		sourceDir = tmpDir
	}
	if cleanup != nil {
		defer cleanup()
	}

	absSource, err := filepath.Abs(sourceDir)
	if err != nil {
		return nil, fmt.Errorf("resolver caminho fonte: %w", err)
	}

	// Determinar tools a verificar.
	tools := opts.Tools
	langs := opts.Langs
	if s.manifest.Exists(installDir) {
		mf, err := s.manifest.Load(installDir)
		if err == nil {
			if len(tools) == 0 && len(mf.Tools) > 0 {
				tools = mf.Tools
			}
			if len(langs) == 0 && len(mf.Langs) > 0 {
				langs = mf.Langs
			}
		}
	}
	if len(langs) == 0 {
		langs = s.langDetect.DetectLangs(installDir)
	}

	allSkills := skills.NewCatalog().AllSkills(langs)

	// Usar checkSkillsForVerify que reutiliza logica do upgrade.
	var items []VerifyItem
	for _, tool := range tools {
		toolItems := s.verifyToolSkills(absSource, installDir, tool, allSkills)
		items = append(items, toolItems...)

		// RI-04: Adicionar item "binary" por CLI (current/missing).
		// Reusa probeBinaryAvailable para consistencia com o probe do install.
		binaryState := s.probeBinaryAvailable(tool)
		spec, ok := NewHelper().specForTool(tool)
		binaryLabel := string(tool) + "-acp"
		if ok {
			binaryLabel = spec.Command
		}
		items = append(items, VerifyItem{
			Tool:  tool,
			Skill: binaryLabel,
			State: binaryState,
			Kind:  VerifyKindBinary,
		})
	}

	// Se nenhuma tool foi determinada, verificar skills base (sem items de binary).
	if len(tools) == 0 {
		for _, skill := range allSkills {
			state := s.verifySkillFile(absSource, installDir, skill)
			items = append(items, VerifyItem{
				Tool:  "",
				Skill: skill,
				State: state,
				Kind:  VerifyKindSkill,
			})
		}
	}

	return items, nil
}

// verifyToolSkills verifica as skills de uma ferramenta especifica no diretorio de instalacao.
// Mapeia upgrade.SkillStatus para VerifyState (ADR-019).
func (s *Service) verifyToolSkills(sourceDir, installDir string, tool skills.Tool, skillList []string) []VerifyItem {
	var items []VerifyItem

	// Verificar skills base (.agents/skills/)
	for _, skill := range skillList {
		state := s.verifySkillFile(sourceDir, installDir, skill)
		items = append(items, VerifyItem{
			Tool:  tool,
			Skill: skill,
			State: state,
			Kind:  VerifyKindSkill,
		})
	}

	return items
}

// specForTool retorna a Spec do runtime ACP para a ferramenta, ou zero-value e false.
// ADR-024: mapeamento canonico Tool -> Spec para probe de binario.
func (r1 *Helper) specForTool(tool skills.Tool) (specs.Spec, bool) {
	switch tool {
	case skills.ToolClaude:
		return specs.NewCatalog().Claude(), true
	case skills.ToolCodex:
		return specs.NewCatalog().Codex(), true
	case skills.ToolGemini:
		return specs.NewCatalog().Gemini(), true
	case skills.ToolCopilot:
		return specs.NewCatalog().Copilot(), true
	default:
		return specs.Spec{}, false
	}
}

// probeBinaryAvailable testa se o binario ACP de uma tool esta disponivel.
// Retorna VerifyStateCurrent se disponivel, VerifyStateMissing caso contrario.
// ADR-024 RI-04: timeout curto para nao violar RF-11.
func (s *Service) probeBinaryAvailable(tool skills.Tool) VerifyState {
	spec, ok := NewHelper().specForTool(tool)
	if !ok {
		return VerifyStateMissing
	}
	ctx, cancel := context.WithTimeout(context.Background(), _probeTimeout)
	defer cancel()
	probe.
		// Limpar cache entre chamadas de verify para garantir resultado fresco.
		NewCatalog().
		ResetCache()
	_, err := probe.NewCatalog().EnsureAvailable(ctx, spec, s.lookPather)
	if err != nil {
		return VerifyStateMissing
	}
	return VerifyStateCurrent
}

// probeBinariesWarn executa probe por CLI e emite Warn para binarios ausentes.
// Nao-fatal: install continua mesmo sem binario disponivel. ADR-024 RI-02.
func (s *Service) probeBinariesWarn(tools []skills.Tool) {
	for _, tool := range tools {
		spec, ok := NewHelper().specForTool(tool)
		if !ok {
			continue
		}
		ctx, cancel := context.WithTimeout(context.Background(), _probeTimeout)
		probe.NewCatalog().
			ResetCache()
		_, err := probe.NewCatalog().EnsureAvailable(ctx, spec, s.lookPather)
		cancel()
		if err != nil {
			s.printer.Warn("[%s] binario ACP '%s' nao encontrado no PATH. Install sera funcional, mas execucao ACP falhara ate o binario estar disponivel.", tool, spec.Command)
		} else {
			s.printer.Debug("[%s] binario ACP '%s' disponivel.", tool, spec.Command)
		}
	}
}

// verifySkillFile verifica o estado de um arquivo de skill (SKILL.md) no diretorio alvo.
// Implementa file-first: le o hash do disco, nao recalcula da fonte.
// Mapeia upgrade.SkillStatus para VerifyState.
func (s *Service) verifySkillFile(sourceDir, installDir, skill string) VerifyState {
	targetSkillMD := filepath.Join(installDir, ".agents", "skills", skill, "SKILL.md")

	// Verificar existencia primeiro (file-first, ADR-019).
	if !s.fs.Exists(targetSkillMD) {
		return VerifyStateMissing
	}

	sourceSkillMD := filepath.Join(sourceDir, ".agents", "skills", skill, "SKILL.md")
	if !s.fs.Exists(sourceSkillMD) {
		// Skill nao existe na fonte; considerar current se instalada.
		return VerifyStateCurrent
	}

	// Mapear resultado do comparador de checksum (upgrade) para VerifyState.
	// StatusOK -> current; outros -> drifted.
	status := s.compareSkillChecksum(sourceSkillMD, targetSkillMD)
	switch status {
	case upgrade.StatusOK:
		return VerifyStateCurrent
	case upgrade.StatusMissing:
		return VerifyStateMissing
	default:
		// StatusOutdated, StatusContentDivergent, StatusRefsDivergent, StatusNoVersion
		return VerifyStateDrifted
	}
}

// compareSkillChecksum compara o hash do SKILL.md da fonte com o do alvo.
// Retorna upgrade.StatusOK se identicos, upgrade.StatusMissing se alvo ausente,
// upgrade.StatusContentDivergent se divergentes.
// Reusa a logica de comparacao do internal/upgrade (file-first).
func (s *Service) compareSkillChecksum(sourceFile, targetFile string) upgrade.SkillStatus {
	if !s.fs.Exists(targetFile) {
		return upgrade.StatusMissing
	}

	sourceHash, err1 := s.fs.FileHash(sourceFile)
	targetHash, err2 := s.fs.FileHash(targetFile)

	if err1 != nil || err2 != nil {
		return upgrade.StatusContentDivergent
	}

	if sourceHash == targetHash {
		return upgrade.StatusOK
	}

	return upgrade.StatusContentDivergent
}

func (s *Service) validate(opts config.InstallOptions) error {
	// Escopo global: ProjectDir pode estar vazio (resolvido antes via globalInstallDir).
	// O diretorio ja foi criado e validado em Execute antes de validate ser chamado.
	if opts.Scope != config.ScopeGlobal {
		if opts.ProjectDir == "" {
			return fmt.Errorf("diretorio alvo e obrigatorio")
		}
		if !s.fs.IsDir(opts.ProjectDir) {
			return fmt.Errorf("diretorio alvo nao encontrado: %s", opts.ProjectDir)
		}
		if !s.fs.Writable(opts.ProjectDir) {
			return fmt.Errorf("sem permissao de escrita em: %s", opts.ProjectDir)
		}
	}
	if opts.SourceDir != "" && !s.fs.IsDir(opts.SourceDir) {
		return fmt.Errorf("diretorio fonte nao encontrado: %s", opts.SourceDir)
	}
	// Tools pode ser vazio aqui apenas se chegou via path alternativo; em Execute
	// a auto-deteccao ja preencheu ou retornou antes. validate permanece defensivo.
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
		return s.installCodex(sourceDir, projectDir, skillList, codexProfile, dryRun)
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
		s.printer.DryRun("copiar .claude/scripts/validate-review-evidence.sh")
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

	reviewScript := filepath.Join(sourceDir, ".claude", "scripts", "validate-review-evidence.sh")
	if s.fs.Exists(reviewScript) {
		if err := s.fs.CopyFile(reviewScript, filepath.Join(projectDir, ".claude", "scripts", "validate-review-evidence.sh")); err != nil {
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

	// Hooks de enforcement programatico do orquestrador (execute-all-tasks + execute-task).
	if err := s.copyOrchestratorHooks(sourceDir, projectDir, filepath.Join(".claude", "hooks")); err != nil {
		return err
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
		if err := s.fs.WriteFile(settingsFile, []byte(NewHelper().defaultClaudeSettings())); err != nil {
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
		s.printer.DryRun("gerar .gemini/agents/*.md via adaptadores")
		s.printer.DryRun("copiar .gemini/hooks/validate-preload.sh")
		s.printer.DryRun("copiar .gemini/hooks/validate-governance.sh")
		s.printer.DryRun("copiar GEMINI.md")
		return nil
	}

	if err := s.fs.MkdirAll(cmdDir); err != nil {
		return err
	}
	s.adapters.GenerateGemini(sourceDir, projectDir)
	s.adapters.GenerateGeminiAgents(sourceDir, projectDir)

	hookDir := filepath.Join(projectDir, ".gemini", "hooks")
	geminiPreload := filepath.Join(sourceDir, ".gemini", "hooks", "validate-preload.sh")
	if s.fs.Exists(geminiPreload) {
		if err := s.fs.MkdirAll(hookDir); err != nil {
			return err
		}
		if err := s.fs.CopyFile(geminiPreload, filepath.Join(hookDir, "validate-preload.sh")); err != nil {
			return err
		}
	}

	geminiGovernance := filepath.Join(sourceDir, ".gemini", "hooks", "validate-governance.sh")
	if s.fs.Exists(geminiGovernance) {
		if err := s.fs.MkdirAll(hookDir); err != nil {
			return err
		}
		if err := s.fs.CopyFile(geminiGovernance, filepath.Join(hookDir, "validate-governance.sh")); err != nil {
			return err
		}
	}

	if err := s.copyOrchestratorHooks(sourceDir, projectDir, filepath.Join(".gemini", "hooks")); err != nil {
		return err
	}

	geminiSettings := filepath.Join(projectDir, ".gemini", "settings.json")
	if !s.fs.Exists(geminiSettings) {
		if err := s.fs.WriteFile(geminiSettings, []byte(NewHelper().defaultGeminiSettings())); err != nil {
			return err
		}
	} else if data, err := s.fs.ReadFile(geminiSettings); err == nil {
		if !strings.Contains(string(data), "validate-preload.sh") {
			s.printer.Warn(".gemini/settings.json ja existe. Adicione os hooks BeforeTool/AfterAgent manualmente (validate-preload, validate-governance).")
		}
	}

	geminiMD := filepath.Join(sourceDir, "GEMINI.md")
	if s.fs.Exists(geminiMD) {
		if err := s.fs.CopyFile(geminiMD, filepath.Join(projectDir, "GEMINI.md")); err != nil {
			return err
		}
	}

	return nil
}

// _orchestratorHooks lista hooks de enforcement programatico do execute-all-tasks/execute-task.
// Distribuidos para todos os tools (.claude, .gemini, .codex, .github) para permitir
// invocacao consistente independente de qual CLI o usuario esteja rodando.
//
// O subagent-stop-wrapper.sh eh especifico do Claude Code (registrado em
// settings.local.json como SubagentStop hook) mas distribuido para todos os
// tools como utilitario invocavel.
var _orchestratorHooks = []string{
	"post-execute-task.sh",
	"pre-execute-all-tasks.sh",
	"post-wave.sh",
	"subagent-stop-wrapper.sh",
}

// _agentsScriptsFiles lista validadores de evidencia canonicos em .agents/scripts/.
// Sao tool-neutros e instalados SEMPRE (independente dos tools selecionados), pois a
// cascata de resolucao das skills e `.agents/scripts/` -> `.claude/scripts/` -> `scripts/`.
// Instalar em .agents/scripts/ garante paridade: um projeto so-Gemini/Codex/Copilot tem os
// mesmos gates de evidencia que um projeto Claude.
var _agentsScriptsFiles = []string{
	"validate-task-evidence.sh",
	"validate-bugfix-evidence.sh",
	"validate-refactor-evidence.sh",
	"validate-review-evidence.sh",
}

// copyAgentsScripts copia os validadores canonicos de evidencia para .agents/scripts/ do
// projeto destino, de forma tool-neutra. Fonte preferencial: sourceDir/.agents/scripts/;
// fallback: sourceDir/.claude/scripts/ (bundles que so espelham o mirror Claude).
func (s *Service) copyAgentsScripts(sourceDir, projectDir string) error {
	dstDir := filepath.Join(projectDir, ".agents", "scripts")
	primarySrc := filepath.Join(sourceDir, ".agents", "scripts")
	fallbackSrc := filepath.Join(sourceDir, ".claude", "scripts")

	for _, name := range _agentsScriptsFiles {
		src := filepath.Join(primarySrc, name)
		if !s.fs.Exists(src) {
			src = filepath.Join(fallbackSrc, name)
		}
		if !s.fs.Exists(src) {
			continue
		}
		if err := s.fs.MkdirAll(dstDir); err != nil {
			return err
		}
		dst := filepath.Join(dstDir, name)
		if err := s.fs.CopyFile(src, dst); err != nil {
			return err
		}
		if err := os.Chmod(dst, 0o755); err != nil {
			s.printer.Warn("nao foi possivel preservar +x em %s: %v", dst, err)
		}
	}
	return nil
}

// _agentsLibFiles lista shell libs vendoradas em .agents/lib/ que skills e hooks
// consomem via cascata `.agents/lib/` -> `scripts/lib/`. Distribuir o vendor
// canonico evita dependencia exclusiva do mirror legado scripts/lib/.
var _agentsLibFiles = []string{
	"check-invocation-depth.sh",
	"parse-hook-input.sh",
}

// copyAgentsLib copia shell libs canonicas de .agents/lib/ da fonte para o
// projeto destino. Falha silenciosamente quando o arquivo nao existe na fonte
// (compatibilidade com installs partindo de bundles antigos sem o vendor).
func (s *Service) copyAgentsLib(sourceDir, projectDir string) error {
	dstDir := filepath.Join(projectDir, ".agents", "lib")
	srcDir := filepath.Join(sourceDir, ".agents", "lib")

	for _, lib := range _agentsLibFiles {
		src := filepath.Join(srcDir, lib)
		if !s.fs.Exists(src) {
			continue
		}
		if err := s.fs.MkdirAll(dstDir); err != nil {
			return err
		}
		dst := filepath.Join(dstDir, lib)
		if err := s.fs.CopyFile(src, dst); err != nil {
			return err
		}
		if err := os.Chmod(dst, 0o755); err != nil {
			s.printer.Warn("nao foi possivel preservar +x em %s: %v", dst, err)
		}
	}
	return nil
}

// _toolValidationHooks lista hooks de validacao por-tool (preload + governanca pos-edicao).
// Distribuidos opcionalmente para Codex/Gemini/Copilot quando presentes na fonte;
// Claude tem caminho dedicado em installClaude por suportar PreToolUse/PostToolUse nativos.
var _toolValidationHooks = []string{
	"validate-preload.sh",
	"validate-governance.sh",
}

// copyToolValidationHooks copia hooks de validacao especificos do tool (preload e
// governanca) quando existirem na fonte. Mantem paridade entre Claude/Codex/Gemini/Copilot.
// Falha silenciosamente para hooks ausentes — cada tool pode adotar apenas o subset
// que faz sentido para sua mecanica de invocacao.
func (s *Service) copyToolValidationHooks(sourceDir, projectDir, toolHookDir string) error {
	dstDir := filepath.Join(projectDir, toolHookDir)
	srcDir := filepath.Join(sourceDir, toolHookDir)

	for _, hook := range _toolValidationHooks {
		src := filepath.Join(srcDir, hook)
		if !s.fs.Exists(src) {
			continue
		}
		if err := s.fs.MkdirAll(dstDir); err != nil {
			return err
		}
		dst := filepath.Join(dstDir, hook)
		if err := s.fs.CopyFile(src, dst); err != nil {
			return err
		}
		if err := os.Chmod(dst, 0o755); err != nil {
			s.printer.Warn("nao foi possivel preservar +x em %s: %v", dst, err)
		}
	}
	return nil
}

// copyOrchestratorHooks copia os hooks do execute-all-tasks/execute-task para o
// diretorio de hooks do tool especificado e preserva permissao +x.
// Falha silenciosamente para hooks ausentes na fonte (compatibilidade legada).
func (s *Service) copyOrchestratorHooks(sourceDir, projectDir, toolHookDir string) error {
	dstDir := filepath.Join(projectDir, toolHookDir)
	srcDir := filepath.Join(sourceDir, toolHookDir)

	for _, hook := range _orchestratorHooks {
		src := filepath.Join(srcDir, hook)
		if !s.fs.Exists(src) {
			continue
		}
		if err := s.fs.MkdirAll(dstDir); err != nil {
			return err
		}
		dst := filepath.Join(dstDir, hook)
		if err := s.fs.CopyFile(src, dst); err != nil {
			return err
		}
		// Preservar permissao executavel (CopyFile do harness nao preserva mode bits).
		// Usar os.Chmod direto ao filesystem real porque s.fs.Chmod pode nao existir.
		if err := os.Chmod(dst, 0o755); err != nil {
			// Nao bloqueante: hook ainda funciona via `bash hook.sh`. Apenas log.
			s.printer.Warn("nao foi possivel preservar +x em %s: %v", dst, err)
		}
	}
	return nil
}

var _codexPlanningSkills = map[string]bool{
	"analyze-project":                true,
	"create-prd":                     true,
	"create-technical-specification": true,
	"create-tasks":                   true,
}

func (r1 *Helper) filterCodexSkills(skillList []string) []string {
	out := make([]string, 0, len(skillList))
	for _, s := range skillList {
		if !_codexPlanningSkills[s] {
			out = append(out, s)
		}
	}
	return out
}

func (s *Service) installCodex(sourceDir, projectDir string, skillList []string, codexProfile string, dryRun bool) error {
	codexDir := filepath.Join(projectDir, ".codex")
	if dryRun {
		s.printer.DryRun("mkdir -p %s", codexDir)
		s.printer.DryRun("gerar .codex/config.toml")
		s.printer.DryRun("gerar .codex/agents/*.toml via adaptadores")
		s.printer.DryRun("copiar .codex/hooks/validate-preload.sh")
		return nil
	}

	if err := s.fs.MkdirAll(codexDir); err != nil {
		return err
	}

	list := skillList
	if codexProfile == "lean" {
		list = NewHelper().filterCodexSkills(skillList)
	}

	content := s.adapters.BuildCodexConfig(list)
	content += NewHelper().codexGovernanceTOML()
	if err := s.fs.WriteFile(filepath.Join(codexDir, "config.toml"), []byte(content)); err != nil {
		return err
	}

	if err := s.fs.WriteFile(filepath.Join(codexDir, "hooks.json"), []byte(NewHelper().defaultCodexHooks())); err != nil {
		return err
	}

	s.adapters.GenerateCodexAgents(sourceDir, projectDir)

	if err := s.copyToolValidationHooks(sourceDir, projectDir, filepath.Join(".codex", "hooks")); err != nil {
		return err
	}

	if err := s.copyOrchestratorHooks(sourceDir, projectDir, filepath.Join(".codex", "hooks")); err != nil {
		return err
	}

	return nil
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
		if err := s.copyToolValidationHooks(sourceDir, projectDir, filepath.Join(".github", "hooks")); err != nil {
			return err
		}
		if err := s.copyOrchestratorHooks(sourceDir, projectDir, filepath.Join(".github", "hooks")); err != nil {
			return err
		}
		governance := filepath.Join(projectDir, ".github", "hooks", "governance.json")
		if !s.fs.Exists(governance) {
			if err := s.fs.WriteFile(governance, []byte(NewHelper().defaultCopilotHooks())); err != nil {
				return err
			}
		}
	} else {
		s.printer.DryRun("gerar .github/agents/*.agent.md via adaptadores")
		s.printer.DryRun("copiar .github/hooks/{validate-preload,validate-governance,post-execute-task,pre-execute-all-tasks,post-wave}.sh")
	}

	return nil
}

func (r1 *Helper) defaultClaudeSettings() string {
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
    ],
    "SubagentStop": [
      {
        "matcher": "task-executor",
        "hooks": [
          {
            "type": "command",
            "command": "bash .claude/hooks/subagent-stop-wrapper.sh"
          }
        ]
      }
    ]
  }
}
`
}

// defaultGeminiSettings devolve o conteudo de .gemini/settings.json com hooks nativos
// 2026 (BeforeTool/AfterAgent) apontando para os validadores compartilhados. Paridade
// com Claude: preload de governanca antes de editar, validacao apos editar.
func (r1 *Helper) defaultGeminiSettings() string {
	return `{
  "hooks": {
    "BeforeTool": [
      {
        "matcher": "edit|write|replace",
        "command": "bash .gemini/hooks/validate-preload.sh"
      }
    ],
    "AfterTool": [
      {
        "matcher": "edit|write|replace",
        "command": "bash .gemini/hooks/validate-governance.sh"
      }
    ],
    "AfterAgent": [
      {
        "command": "bash .gemini/hooks/subagent-stop-wrapper.sh"
      }
    ]
  }
}
`
}

// defaultCopilotHooks devolve o conteudo de .github/hooks/governance.json no formato
// nativo do Copilot CLI 2026 (version:1, preToolUse/agentStop) apontando para os
// validadores compartilhados. Paridade com Claude/Gemini.
func (r1 *Helper) defaultCopilotHooks() string {
	return `{
  "version": 1,
  "preToolUse": [
    {
      "match": { "tool": "str_replace_editor|write|bash" },
      "bash": "bash .github/hooks/validate-preload.sh"
    }
  ],
  "postToolUse": [
    {
      "match": { "tool": "str_replace_editor|write" },
      "bash": "bash .github/hooks/validate-governance.sh"
    }
  ],
  "agentStop": [
    {
      "bash": "bash .github/hooks/subagent-stop-wrapper.sh"
    }
  ]
}
`
}

// defaultCodexHooks devolve o conteudo de .codex/hooks.json no formato nativo do
// Codex CLI 2026 (PreToolUse). O Codex tem lacuna documentada de route-around, por
// isso o enforcement por hook e suplementado por sandbox_mode/approval_policy no
// config.toml (ver codexGovernanceTOML e enforcement-matrix.md, caveat Codex).
func (r1 *Helper) defaultCodexHooks() string {
	return `{
  "PreToolUse": [
    {
      "matcher": "shell|apply_patch|edit",
      "command": "bash .codex/hooks/validate-preload.sh"
    }
  ],
  "PostToolUse": [
    {
      "matcher": "apply_patch|edit",
      "command": "bash .codex/hooks/validate-governance.sh"
    }
  ]
}
`
}

// codexGovernanceTOML devolve o bloco TOML de governanca apendado ao .codex/config.toml:
// sandbox_mode + approval_policy fecham a lacuna de route-around dos hooks do Codex
// (o hook sozinho nao e suficiente — ver ADR-002 e enforcement-matrix.md).
func (r1 *Helper) codexGovernanceTOML() string {
	return `
# Governanca (paridade cross-CLI): o hook PreToolUse do Codex tem lacuna de
# route-around documentada. sandbox_mode + approval_policy garantem que mudancas
# de filesystem e comandos passem por aprovacao, fechando a lacuna (ADR-002).
sandbox_mode = "workspace-write"
approval_policy = "on-request"

[hooks]
PreToolUse = "bash .codex/hooks/validate-preload.sh"
PostToolUse = "bash .codex/hooks/validate-governance.sh"
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

func (r1 *Helper) toolNames(tools []skills.Tool) []string {
	out := make([]string, len(tools))
	for i, t := range tools {
		out[i] = string(t)
	}
	return out
}

func (r1 *Helper) langNames(langs []skills.Lang) []string {
	out := make([]string, len(langs))
	for i, l := range langs {
		out[i] = string(l)
	}
	return out
}
