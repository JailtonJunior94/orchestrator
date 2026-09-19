package install

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"time"

	"github.com/JailtonJunior94/ai-spec-harness/internal/adapters"
	"github.com/JailtonJunior94/ai-spec-harness/internal/batchreport"
	"github.com/JailtonJunior94/ai-spec-harness/internal/config"
	"github.com/JailtonJunior94/ai-spec-harness/internal/contextgen"
	"github.com/JailtonJunior94/ai-spec-harness/internal/detect"
	"github.com/JailtonJunior94/ai-spec-harness/internal/embedded"
	"github.com/JailtonJunior94/ai-spec-harness/internal/fs"
	"github.com/JailtonJunior94/ai-spec-harness/internal/manifest"
	"github.com/JailtonJunior94/ai-spec-harness/internal/output"
	"github.com/JailtonJunior94/ai-spec-harness/internal/platform"
	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/precondition"
	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/probe"
	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/specs"
	"github.com/JailtonJunior94/ai-spec-harness/internal/skills"
	"github.com/JailtonJunior94/ai-spec-harness/internal/tracking"
	"github.com/JailtonJunior94/ai-spec-harness/internal/upgrade"
	"github.com/JailtonJunior94/ai-spec-harness/internal/version"
)

const probeTimeout = 3 * time.Second

const codexTrustRPCTimeout = 5 * time.Second

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
	VerifyStateInert   VerifyState = "inert"
	VerifyStateUnknown VerifyState = "unknown"
)

// VerifyKind classifica o tipo de item verificado.
// ADR-024: "skill" para skills de governanca; "binary" para binario ACP por CLI.
type VerifyKind string

const (
	// VerifyKindSkill indica verificacao de uma skill de governanca.
	VerifyKindSkill VerifyKind = "skill"
	// VerifyKindBinary indica verificacao de disponibilidade do binario ACP por CLI.
	VerifyKindBinary       VerifyKind = "binary"
	VerifyKindRuntime      VerifyKind = "runtime"
	VerifyKindPrecondition VerifyKind = "precondition"
)

// VerifyItem representa o resultado de verificacao de uma skill ou binario para um agente.
// ADR-024: campo Kind distingue skill vs binario ACP.
type VerifyItem struct {
	Tool   skills.Tool
	Skill  string
	State  VerifyState
	Kind   VerifyKind
	Remedy string
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
		detectedLangs := s.detectLangs(opts.ProjectDir, opts.FocusPaths)
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

	var expectedChecksums map[string]string
	if s.manifest.Exists(projectDir) {
		if previous, err := s.manifest.Load(projectDir); err == nil {
			expectedChecksums = previous.FileChecksums
		}
	}

	var tracker *tracking.Tracker
	if !opts.DryRun {
		tracker = tracking.NewTransactional(s.fs, projectDir, manifest.ManifestFile, expectedChecksums)
		previousFS, previousAdapters, previousCtxgen := s.fs, s.adapters, s.ctxgen
		s.fs = tracker
		s.adapters = adapters.NewGenerator(tracker, s.printer)
		s.ctxgen = contextgen.NewGenerator(tracker, s.printer)
		defer func() {
			s.fs = previousFS
			s.adapters = previousAdapters
			s.ctxgen = previousCtxgen
		}()
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
	if err := s.installBaseSkills(sourceDir, projectDir, allSkills, linkMode, opts.DryRun, opts.FollowExternalSymlinks); err != nil {
		return fmt.Errorf("instalar skills base: %w", err)
	}

	// 2. Instalar adaptadores por ferramenta
	for _, tool := range opts.Tools {
		if err := s.installTool(sourceDir, projectDir, tool, allSkills, linkMode, opts.DryRun, opts.CodexProfile, opts.Model); err != nil {
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
		if err := s.copyToolValidationHooks(sourceDir, projectDir, filepath.Join(".agents", "hooks")); err != nil {
			s.printer.Warn("falha ao copiar validadores canonicos: %v", err)
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
	// -> `scripts/` resolve os gates mesmo em projetos sem Claude (so Codex/Copilot/OpenCode).
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

	if !opts.DryRun {
		if len(tracker.Conflicts()) > 0 && !opts.OverwriteConflicts {
			return batchreport.ConflictError(tracker.Conflicts(), sourceDir)
		}
		if opts.OverwriteConflicts {
			tracker.AllowOverwrite()
		}
		if err := tracker.Commit(); err != nil {
			return fmt.Errorf("apply installation batch: %w", err)
		}
		for _, path := range tracker.ExecutablePaths() {
			if err := os.Chmod(path, 0o755); err != nil {
				s.printer.Warn("nao foi possivel preservar +x em %s: %v", path, err)
			}
		}

		batchreport.Print(s.printer, "install", batchreport.Build(tracker))

		checksums := s.computeChecksums(sourceDir, allSkills)
		mf := &manifest.Manifest{
			Version:        version.NewProvider().ResolveFromExecutable(),
			CreatedAt:      time.Now(),
			UpdatedAt:      time.Now(),
			SourceDir:      sourceDir,
			LinkMode:       linkMode,
			Tools:          opts.Tools,
			Langs:          opts.Langs,
			Skills:         allSkills,
			Checksums:      checksums,
			CodexProfile:   opts.CodexProfile,
			SkillVersions:  s.collectSkillVersions(sourceDir, allSkills),
			InstalledFiles: tracker.CreatedPaths(),
			MergedFiles:    tracker.MergedPaths(),
			FileChecksums:  tracker.Checksums(),
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
		langs = s.detectLangs(installDir, opts.FocusPaths)
	}

	allSkills := s.sourceSkills(absSource, langs)

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

		if tool == skills.ToolOpenCode {
			runtimeState := VerifyStateCurrent
			if _, lookErr := s.lookPather.LookPath(specs.OpenCodePluginRuntime); lookErr != nil {
				runtimeState = VerifyStateMissing
			}
			items = append(items, VerifyItem{
				Tool:  tool,
				Skill: "governance-plugin-runtime(" + specs.OpenCodePluginRuntime + ")",
				State: runtimeState,
				Kind:  VerifyKindRuntime,
			})
		}

		items = append(items, s.verifyGovernanceArtifacts(absSource, installDir, tool)...)

		items = append(items, s.verifyPreconditions(tool, installDir, opts.CheckCodexTrust)...)
	}

	items = append(items, s.verifyCanonicalValidators(absSource, installDir)...)

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
type focusLangDetector interface {
	DetectLangsIn(projectDir string, focusPaths []string) []skills.Lang
}

func (s *Service) detectLangs(projectDir string, focusPaths []string) []skills.Lang {
	if len(focusPaths) > 0 {
		if detector, ok := s.langDetect.(focusLangDetector); ok {
			return detector.DetectLangsIn(projectDir, focusPaths)
		}
	}
	return s.langDetect.DetectLangs(projectDir)
}

func (s *Service) sourceSkills(sourceDir string, langs []skills.Lang) []string {
	sourceSkillsDir := filepath.Join(sourceDir, ".agents", "skills")
	entries, err := s.fs.ReadDir(sourceSkillsDir)
	if err != nil {
		return skills.NewCatalog().AllSkills(langs)
	}

	out := make([]string, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() || !NewHelper().skillSelectedForLangs(entry.Name(), langs) {
			continue
		}
		// Reconcilia com upgrade.checkSkills: so e skill o diretorio que
		// contem SKILL.md. Diretorios auxiliares (ex.: tests/ com pytest) sao
		// ignorados para evitar divergencia entre verify e upgrade --check.
		if !s.fs.Exists(filepath.Join(sourceSkillsDir, entry.Name(), "SKILL.md")) {
			continue
		}
		out = append(out, entry.Name())
	}
	return out
}

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
	spec, err := specs.NewCatalog().ResolveACPSpec(string(tool))
	if err != nil {
		return specs.Spec{}, false
	}
	return spec, true
}

// probeBinaryAvailable testa se o binario ACP de uma tool esta disponivel.
// Retorna VerifyStateCurrent se disponivel, VerifyStateMissing caso contrario.
// ADR-024 RI-04: timeout curto para nao violar RF-11.
func (s *Service) probeBinaryAvailable(tool skills.Tool) VerifyState {
	spec, ok := NewHelper().specForTool(tool)
	if !ok {
		return VerifyStateMissing
	}
	ctx, cancel := context.WithTimeout(context.Background(), probeTimeout)
	defer cancel()
	_, err := probe.NewCatalog().Resolve(ctx, spec, s.lookPather)
	if err != nil {
		return VerifyStateMissing
	}
	return VerifyStateCurrent
}

func (s *Service) verifyPreconditions(tool skills.Tool, installDir string, checkCodexTrust bool) []VerifyItem {
	agent, err := specs.NewCatalog().AgentByID(string(tool))
	if err != nil {
		return nil
	}

	absInstallDir, err := filepath.Abs(installDir)
	if err != nil {
		absInstallDir = installDir
	}

	var codexCheck precondition.CodexTrustChecker
	if checkCodexTrust {
		required := enforcementNativeKeys(agent.Enforcement())
		codexCheck = func() (precondition.CodexTrustReport, error) {
			client := precondition.NewCodexAppServerClient("", absInstallDir)
			ctx, cancel := context.WithTimeout(context.Background(), codexTrustRPCTimeout)
			defer cancel()
			return precondition.EvaluateCodexTrustedHash(ctx, client, codexTrustRPCTimeout, required)
		}
	}

	configPath, _ := precondition.DefaultCopilotConfigPath()
	reader := precondition.NewFileCopilotConfigReader(configPath)

	items := make([]VerifyItem, 0, len(agent.Enforcement().Preconditions()))
	for _, pre := range agent.Enforcement().Preconditions() {
		items = append(items, preconditionItems(tool, pre, precondition.Evaluate(pre, absInstallDir, reader, codexCheck))...)
	}
	return items
}

func preconditionItems(tool skills.Tool, pre specs.EnforcementPrecondition, report precondition.Report) []VerifyItem {
	if len(report.Points) == 0 {
		return []VerifyItem{{
			Tool:   tool,
			Skill:  "precondition(" + pre.Kind().String() + ")",
			State:  preconditionVerifyState(report.State),
			Kind:   VerifyKindPrecondition,
			Remedy: report.Remedy,
		}}
	}
	items := make([]VerifyItem, 0, len(report.Points))
	for _, point := range report.Points {
		items = append(items, VerifyItem{
			Tool:   tool,
			Skill:  "precondition(" + pre.Kind().String() + ":" + point.EventName + ")",
			State:  preconditionVerifyState(point.State),
			Kind:   VerifyKindPrecondition,
			Remedy: report.Remedy,
		})
	}
	return items
}

func preconditionVerifyState(state specs.PreconditionState) VerifyState {
	switch state {
	case specs.PreconditionInert:
		return VerifyStateInert
	case specs.PreconditionUnknown:
		return VerifyStateUnknown
	default:
		return VerifyStateCurrent
	}
}

func enforcementNativeKeys(enf specs.Enforcement) []string {
	keys := make([]string, 0, len(enf.Coverage()))
	for _, cov := range enf.Coverage() {
		keys = append(keys, cov.NativeKey())
	}
	return keys
}

// probeBinariesWarn executa probe por CLI e emite Warn para binarios ausentes.
// Nao-fatal: install continua mesmo sem binario disponivel. ADR-024 RI-02.
func (s *Service) probeBinariesWarn(tools []skills.Tool) {
	for _, tool := range tools {
		spec, ok := NewHelper().specForTool(tool)
		if !ok {
			continue
		}
		ctx, cancel := context.WithTimeout(context.Background(), probeTimeout)
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

func (s *Service) installBaseSkills(sourceDir, projectDir string, skillList []string, mode skills.LinkMode, dryRun bool, followExternalSymlinks bool) error {
	s.printer.Step("Instalando skills canonicas...")
	skillsDir := filepath.Join(projectDir, ".agents", "skills")

	if err := fs.RefuseExternalSymlink(s.fs, projectDir, skillsDir, followExternalSymlinks); err != nil {
		return err
	}

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

func (s *Service) installTool(sourceDir, projectDir string, tool skills.Tool, skillList []string, mode skills.LinkMode, dryRun bool, codexProfile, model string) error {
	s.printer.Step("Instalando %s...", tool)

	switch tool {
	case skills.ToolClaude:
		return s.installClaude(sourceDir, projectDir, skillList, mode, dryRun)
	case skills.ToolCodex:
		return s.installCodex(sourceDir, projectDir, skillList, codexProfile, dryRun)
	case skills.ToolCopilot:
		return s.installCopilot(sourceDir, projectDir, skillList, mode, dryRun)
	case skills.ToolOpenCode:
		return s.installOpenCode(sourceDir, projectDir, dryRun, model)
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
		for _, ruleFile := range skills.UniversalRuleFiles {
			s.printer.DryRun("copiar .claude/rules/%s", ruleFile)
		}
		s.printer.DryRun("copiar .claude/scripts/validate-task-evidence.sh")
		s.printer.DryRun("copiar .claude/scripts/validate-bugfix-evidence.sh")
		s.printer.DryRun("copiar .claude/scripts/validate-refactor-evidence.sh")
		s.printer.DryRun("copiar .claude/scripts/validate-review-evidence.sh")
		s.printer.DryRun("copiar .claude/hooks/validate-governance.sh")
		s.printer.DryRun("copiar .claude/hooks/validate-preload.sh")
		s.printer.DryRun("copiar .claude/hooks/validate-session-end.sh")
		s.printer.DryRun("copiar scripts/lib/parse-hook-input.sh")
		s.printer.DryRun("copiar scripts/lib/check-invocation-depth.sh")
		s.printer.DryRun("configurar hooks PreToolUse e PostToolUse em .claude/settings.local.json")
		s.printer.DryRun("gerar .claude/agents/*.md via adaptadores")
		return nil
	}

	for _, ruleFile := range skills.UniversalRuleFiles {
		src := filepath.Join(sourceDir, ".claude", "rules", ruleFile)
		if !s.fs.Exists(src) {
			continue
		}
		if err := s.fs.CopyFile(src, filepath.Join(projectDir, ".claude", "rules", ruleFile)); err != nil {
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

	if err := s.copyToolValidationHooks(sourceDir, projectDir, filepath.Join(".claude", "hooks")); err != nil {
		return err
	}

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

	if err := s.writeMergedMarkdownFromSource(filepath.Join(sourceDir, "AGENTS.md"), filepath.Join(projectDir, "AGENTS.md")); err != nil {
		return err
	}

	if err := s.writeClaudeSettings(projectDir); err != nil {
		return err
	}

	s.adapters.GenerateClaude(sourceDir, projectDir)
	return nil
}

func (s *Service) writeClaudeSettings(projectDir string) error {
	settingsFile := filepath.Join(projectDir, ".claude", "settings.local.json")

	var existing []byte
	if s.fs.Exists(settingsFile) {
		data, err := s.fs.ReadFile(settingsFile)
		if err != nil {
			return err
		}
		existing = data
	}

	if len(bytes.TrimSpace(existing)) == 0 {
		return s.fs.WriteFile(settingsFile, []byte(NewHelper().defaultClaudeSettings()))
	}

	merged, err := NewHelper().mergeClaudeSettings(existing)
	if err != nil {
		return fmt.Errorf("merge .claude/settings.local.json: %w", err)
	}
	if err := s.fs.WriteFile(settingsFile, merged); err != nil {
		return err
	}
	if err := s.trackMerged(settingsFile); err != nil {
		return err
	}
	return nil
}

var orchestratorHooks = []string{
	"post-execute-task.sh",
	"pre-execute-all-tasks.sh",
	"post-wave.sh",
	"subagent-stop-wrapper.sh",
}

var agentsScriptsFiles = []string{
	"validate-task-evidence.sh",
	"validate-bugfix-evidence.sh",
	"validate-refactor-evidence.sh",
	"validate-review-evidence.sh",
	"hook-prereq-gate.sh",
	"resolve-references.sh",
	"validate-skill-prerequisites.sh",
	"validate-governance-references.sh",
	"validate-session-end.sh",
	"git-operation-gate.sh",
}

// copyAgentsScripts copia os validadores canonicos de evidencia para .agents/scripts/ do
// projeto destino, de forma tool-neutra. Fonte preferencial: sourceDir/.agents/scripts/;
// fallback: sourceDir/.claude/scripts/ (bundles que so espelham o mirror Claude).
func (s *Service) copyAgentsScripts(sourceDir, projectDir string) error {
	dstDir := filepath.Join(projectDir, ".agents", "scripts")
	primarySrc := filepath.Join(sourceDir, ".agents", "scripts")
	fallbackSrc := filepath.Join(sourceDir, ".claude", "scripts")

	for _, name := range agentsScriptsFiles {
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
		s.markExecutable(dst)
	}
	return nil
}

var agentsLibFiles = []string{
	"check-invocation-depth.sh",
	"parse-hook-input.sh",
}

// copyAgentsLib copia shell libs canonicas de .agents/lib/ da fonte para o
// projeto destino. Falha silenciosamente quando o arquivo nao existe na fonte
// (compatibilidade com installs partindo de bundles antigos sem o vendor).
func (s *Service) copyAgentsLib(sourceDir, projectDir string) error {
	dstDir := filepath.Join(projectDir, ".agents", "lib")
	srcDir := filepath.Join(sourceDir, ".agents", "lib")

	for _, lib := range agentsLibFiles {
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
		s.markExecutable(dst)
	}
	return nil
}

var toolValidationHooks = []string{
	"validate-preload.sh",
	"validate-governance.sh",
	"validate-session-end.sh",
}

// copyToolValidationHooks copia hooks de validacao especificos do tool (preload e
// governanca) quando existirem na fonte. Mantem paridade entre Claude/Codex/Copilot.
// Falha silenciosamente para hooks ausentes — cada tool pode adotar apenas o subset
// que faz sentido para sua mecanica de invocacao.
func (s *Service) copyToolValidationHooks(sourceDir, projectDir, toolHookDir string) error {
	dstDir := filepath.Join(projectDir, toolHookDir)
	srcDir := filepath.Join(sourceDir, toolHookDir)

	for _, hook := range toolValidationHooks {
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
		s.markExecutable(dst)
	}
	return nil
}

// copyOrchestratorHooks copia os hooks do execute-all-tasks/execute-task para o
// diretorio de hooks do tool especificado e preserva permissao +x.
// Falha silenciosamente para hooks ausentes na fonte (compatibilidade legada).
func (s *Service) copyOrchestratorHooks(sourceDir, projectDir, toolHookDir string) error {
	dstDir := filepath.Join(projectDir, toolHookDir)
	srcDir := filepath.Join(sourceDir, toolHookDir)

	for _, hook := range orchestratorHooks {
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
		s.markExecutable(dst)
	}
	return nil
}

var codexPlanningSkills = map[string]bool{
	"analyze-project":                true,
	"create-prd":                     true,
	"create-technical-specification": true,
	"create-tasks":                   true,
}

func (r1 *Helper) filterCodexSkills(skillList []string) []string {
	out := make([]string, 0, len(skillList))
	for _, s := range skillList {
		if !codexPlanningSkills[s] {
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

	helper := NewHelper()
	content := helper.codexGovernancePreambleTOML()
	content += s.adapters.BuildCodexConfig(list)
	content += helper.codexGovernanceHooksTOML()

	configPath := filepath.Join(codexDir, "config.toml")
	merged := false
	if existing, err := s.fs.ReadFile(configPath); err == nil {
		content, merged = contextgen.MergeCodexInstallConfig(content, string(existing))
	}
	if err := s.fs.WriteFile(configPath, []byte(content)); err != nil {
		return err
	}
	if merged {
		if err := s.trackMerged(configPath); err != nil {
			return err
		}
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
		governanceHooks := upgrade.NewHelper().CopilotGovernanceHooksPath(projectDir)
		if _, err := upgrade.NewHelper().RepairCopilotGovernanceHooks(s.fs, projectDir); err != nil {
			return err
		}
		if err := s.trackInstalled(governanceHooks); err != nil {
			return err
		}
		settings := filepath.Join(projectDir, ".github", "settings.json")
		var existing []byte
		if s.fs.Exists(settings) {
			data, err := s.fs.ReadFile(settings)
			if err != nil {
				return err
			}
			existing = data
		}
		merged, err := NewHelper().mergeCopilotSettings(existing)
		if err != nil {
			return fmt.Errorf("merge Copilot repository settings: %w", err)
		}
		if err := s.fs.WriteFile(settings, merged); err != nil {
			return err
		}
		if err := s.trackMerged(settings); err != nil {
			return err
		}
	} else {
		s.printer.DryRun("gerar .github/agents/*.agent.md via adaptadores")
		s.printer.DryRun("copiar .github/hooks/{validate-preload,validate-governance,post-execute-task,pre-execute-all-tasks,post-wave}.sh")
	}

	return nil
}

var ErrOpenCodePluginRuntimeMissing = fmt.Errorf("opencode governance plugin runtime %q not found on PATH — install it before installing/verifying the opencode agent", specs.OpenCodePluginRuntime)

var openCodeRequiredTools = []string{"bash", "edit", "write", "multiedit", "patch"}

var openCodePermissionFactory = specs.DefaultOpenCodePermission

func (s *Service) installOpenCode(sourceDir, projectDir string, dryRun bool, model string) error {
	if _, err := s.lookPather.LookPath(specs.OpenCodePluginRuntime); err != nil {
		return ErrOpenCodePluginRuntimeMissing
	}

	pluginDir := filepath.Join(projectDir, ".opencode", "plugin")
	configPath := filepath.Join(projectDir, specs.OpenCodeConfigFileName)

	if dryRun {
		s.printer.DryRun("mkdir -p %s", pluginDir)
		s.printer.DryRun("merge permission block into %s", configPath)
		return nil
	}

	if err := s.fs.MkdirAll(pluginDir); err != nil {
		return err
	}

	sourcePluginDir := filepath.Join(sourceDir, ".opencode", "plugin")
	if s.fs.IsDir(sourcePluginDir) {
		if err := s.fs.CopyDir(sourcePluginDir, pluginDir); err != nil {
			return err
		}
	}

	var existing []byte
	if s.fs.Exists(configPath) {
		data, err := s.fs.ReadFile(configPath)
		if err != nil {
			return err
		}
		existing = data
	}

	permission := openCodePermissionFactory()
	if err := specs.ValidatePermissionBlock(permission, openCodeRequiredTools...); err != nil {
		return fmt.Errorf("opencode permission block: %w", err)
	}

	merged, err := specs.MergeOpenCodeConfig(existing, permission, model)
	if err != nil {
		return err
	}
	if err := specs.ValidatePermissionBlock(decodeWrittenPermission(merged), openCodeRequiredTools...); err != nil {
		return fmt.Errorf("opencode permission block written to disk: %w", err)
	}
	if err := s.fs.WriteFile(configPath, merged); err != nil {
		return err
	}
	if err := s.trackMerged(configPath); err != nil {
		return err
	}
	return nil
}

func (s *Service) trackInstalled(path string) error {
	if tracker, ok := s.fs.(*tracking.Tracker); ok {
		return tracker.MarkInstalled(path)
	}
	return nil
}

func (s *Service) trackMerged(path string) error {
	if tracker, ok := s.fs.(*tracking.Tracker); ok {
		return tracker.MarkMerged(path)
	}
	return nil
}

func (s *Service) markExecutable(path string) {
	if tracker, ok := s.fs.(*tracking.Tracker); ok {
		tracker.MarkExecutable(path)
	}
}

func decodeWrittenPermission(configBytes []byte) map[string]any {
	var doc map[string]any
	if err := json.Unmarshal(configBytes, &doc); err != nil {
		return nil
	}
	perm, _ := doc["permission"].(map[string]any)
	return perm
}

var langImplementationSkills = map[string]bool{
	"go-implementation":            true,
	"object-calisthenics-go":       true,
	"node-implementation":          true,
	"python-implementation":        true,
	"dotnet-csharp-implementation": true,
}

func (r1 *Helper) skillSelectedForLangs(skillName string, langFilter []skills.Lang) bool {
	if !langImplementationSkills[skillName] {
		return true
	}
	for _, selected := range skills.NewCatalog().LangSkills(langFilter) {
		if selected == skillName {
			return true
		}
	}
	return false
}

func (r1 *Helper) defaultClaudeSettings() string {
	return `{
  "hooks": {
    "PreToolUse": [
      {
        "matcher": "Bash|Edit|Write|NotebookEdit|apply_patch",
        "hooks": [
          {
            "type": "command",
            "command": "bash \"$CLAUDE_PROJECT_DIR/.claude/hooks/validate-preload.sh\""
          }
        ]
      }
    ],
    "PostToolUse": [
      {
        "matcher": "Bash|Edit|Write|NotebookEdit|apply_patch",
        "hooks": [
          {
            "type": "command",
            "command": "bash \"$CLAUDE_PROJECT_DIR/.claude/hooks/validate-governance.sh\""
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
            "command": "bash \"$CLAUDE_PROJECT_DIR/.claude/hooks/subagent-stop-wrapper.sh\""
          }
        ]
      }
    ],
    "Stop": [
      {
        "hooks": [
          {
            "type": "command",
            "command": "bash \"$CLAUDE_PROJECT_DIR/.claude/hooks/validate-session-end.sh\""
          }
        ]
      }
    ]
  }
}
`
}

func (r1 *Helper) mergeClaudeSettings(existing []byte) ([]byte, error) {
	settings := map[string]any{}
	if len(existing) > 0 {
		if err := json.Unmarshal(existing, &settings); err != nil {
			return nil, err
		}
	}

	var governance struct {
		Hooks map[string][]any `json:"hooks"`
	}
	if err := json.Unmarshal([]byte(r1.defaultClaudeSettings()), &governance); err != nil {
		return nil, err
	}

	original, hadHooks := settings["hooks"]
	hooks, ok := original.(map[string]any)
	if !ok {
		if hadHooks {
			return nil, fmt.Errorf("hooks must be an object")
		}
		hooks = map[string]any{}
	}

	merged := make(map[string]any, len(hooks)+len(governance.Hooks))
	for event, value := range hooks {
		merged[event] = value
	}

	for event, expected := range governance.Hooks {
		existingHooks, exists := merged[event]
		if !exists {
			merged[event] = expected
			continue
		}
		existingList, isList := existingHooks.([]any)
		if !isList {
			return nil, fmt.Errorf("hooks.%s must be an array", event)
		}
		for _, hook := range expected {
			if r1.hookEntryExists(existingList, hook) {
				continue
			}
			existingList = append(existingList, hook)
		}
		merged[event] = existingList
	}
	settings["hooks"] = merged

	if hadHooks && reflect.DeepEqual(original, merged) {
		return existing, nil
	}

	edited, err := specs.NewCatalog().SetJSONTopLevelKey(existing, "hooks", merged)
	if err != nil {
		return json.MarshalIndent(settings, "", "  ")
	}
	return edited, nil
}

func (r1 *Helper) defaultCopilotHooks() string {
	return upgrade.DefaultCopilotGovernanceHooks
}

func (r1 *Helper) mergeCopilotSettings(existing []byte) ([]byte, error) {
	settings := map[string]any{}
	if len(existing) > 0 {
		if err := json.Unmarshal(existing, &settings); err != nil {
			return nil, err
		}
	}

	var governance struct {
		Hooks map[string][]any `json:"hooks"`
	}
	if err := json.Unmarshal([]byte(r1.defaultCopilotHooks()), &governance); err != nil {
		return nil, err
	}

	original, hadHooks := settings["hooks"]
	hooks, ok := original.(map[string]any)
	if !ok {
		if hadHooks {
			return nil, fmt.Errorf("hooks must be an object")
		}
		hooks = map[string]any{}
	}

	merged := make(map[string]any, len(hooks)+len(governance.Hooks))
	for event, value := range hooks {
		merged[event] = value
	}

	migrated := make([]any, 0, len(upgrade.ObsoleteCopilotHookKeys))
	for _, obsolete := range upgrade.ObsoleteCopilotHookKeys {
		entries, present := merged[obsolete]
		if !present {
			continue
		}
		delete(merged, obsolete)
		list, isList := entries.([]any)
		if !isList {
			return nil, fmt.Errorf("hooks.%s must be an array", obsolete)
		}
		migrated = append(migrated, list...)
	}
	if len(migrated) > 0 {
		merged[upgrade.CopilotSessionEndHookKey] = r1.appendCopilotHooks(merged[upgrade.CopilotSessionEndHookKey], migrated)
	}

	for event, expected := range governance.Hooks {
		existingHooks, exists := merged[event]
		if !exists {
			merged[event] = expected
			continue
		}
		existingList, ok := existingHooks.([]any)
		if !ok {
			return nil, fmt.Errorf("hooks.%s must be an array", event)
		}
		for _, hook := range expected {
			if r1.hookEntryExists(existingList, hook) {
				continue
			}
			existingList = append(existingList, hook)
		}
		merged[event] = existingList
	}
	settings["hooks"] = merged

	if len(existing) == 0 {
		encoded, err := json.MarshalIndent(settings, "", "  ")
		if err != nil {
			return nil, err
		}
		return encoded, nil
	}

	if hadHooks && reflect.DeepEqual(original, merged) {
		return existing, nil
	}

	edited, err := specs.NewCatalog().SetJSONTopLevelKey(existing, "hooks", merged)
	if err != nil {
		return json.MarshalIndent(settings, "", "  ")
	}
	return edited, nil
}

func (r1 *Helper) appendCopilotHooks(existing any, extra []any) []any {
	list, _ := existing.([]any)
	result := make([]any, 0, len(list)+len(extra))
	result = append(result, list...)
	for _, entry := range extra {
		if r1.hookEntryExists(result, entry) {
			continue
		}
		result = append(result, entry)
	}
	return result
}

func (r1 *Helper) hookEntryExists(hooks []any, expected any) bool {
	expectedJSON, err := json.Marshal(expected)
	if err != nil {
		return false
	}
	for _, hook := range hooks {
		hookJSON, err := json.Marshal(hook)
		if err == nil && string(hookJSON) == string(expectedJSON) {
			return true
		}
	}
	return false
}

func (r1 *Helper) codexGovernancePreambleTOML() string {
	return `# Governanca (paridade cross-CLI): o hook PreToolUse do Codex tem lacuna de
# route-around documentada. sandbox_mode + approval_policy garantem que mudancas
# de filesystem e comandos passem por aprovacao, fechando a lacuna (ADR-002).
# Chaves de nivel raiz precisam preceder qualquer tabela TOML.
sandbox_mode = "workspace-write"
approval_policy = "on-request"

`
}

func (r1 *Helper) codexGovernanceHooksTOML() string {
	return `[[hooks.PreToolUse]]
[[hooks.PreToolUse.hooks]]
type = "command"
command = "bash .codex/hooks/validate-preload.sh"

[[hooks.PostToolUse]]
[[hooks.PostToolUse.hooks]]
type = "command"
command = "bash .codex/hooks/validate-governance.sh"

[[hooks.Stop]]
[[hooks.Stop.hooks]]
type = "command"
command = "bash .codex/hooks/validate-session-end.sh"
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

const VerifyKindArtifact VerifyKind = "artifact"

var toolHookDirs = map[skills.Tool]string{
	skills.ToolClaude:  filepath.Join(".claude", "hooks"),
	skills.ToolCodex:   filepath.Join(".codex", "hooks"),
	skills.ToolCopilot: filepath.Join(".github", "hooks"),
}

func (s *Service) verifyGovernanceArtifacts(sourceDir, installDir string, tool skills.Tool) []VerifyItem {
	var items []VerifyItem

	if hookDir, ok := toolHookDirs[tool]; ok {
		for _, hook := range NewHelper().governanceHookFiles() {
			src := filepath.Join(sourceDir, hookDir, hook)
			if !s.fs.Exists(src) {
				continue
			}
			items = append(items, VerifyItem{
				Tool:   tool,
				Skill:  filepath.ToSlash(filepath.Join(hookDir, hook)),
				State:  s.verifyArtifactFile(src, filepath.Join(installDir, hookDir, hook)),
				Kind:   VerifyKindArtifact,
				Remedy: verifyArtifactRemedy,
			})
		}
	}

	if tool != skills.ToolOpenCode {
		return items
	}

	pluginDir := filepath.Join(".opencode", "plugin")
	entries, err := s.fs.ReadDir(filepath.Join(sourceDir, pluginDir))
	if err != nil {
		return items
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		rel := filepath.Join(pluginDir, entry.Name())
		items = append(items, VerifyItem{
			Tool:   tool,
			Skill:  filepath.ToSlash(rel),
			State:  s.verifyArtifactFile(filepath.Join(sourceDir, rel), filepath.Join(installDir, rel)),
			Kind:   VerifyKindArtifact,
			Remedy: verifyArtifactRemedy,
		})
	}
	return items
}

var canonicalValidatorDirs = []string{
	filepath.Join(".agents", "hooks"),
	filepath.Join(".agents", "scripts"),
	filepath.Join(".agents", "lib"),
}

func (s *Service) verifyCanonicalValidators(sourceDir, installDir string) []VerifyItem {
	var items []VerifyItem
	for _, dir := range canonicalValidatorDirs {
		entries, err := s.fs.ReadDir(filepath.Join(sourceDir, dir))
		if err != nil {
			continue
		}
		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			rel := filepath.Join(dir, entry.Name())
			items = append(items, VerifyItem{
				Tool:   "",
				Skill:  filepath.ToSlash(rel),
				State:  s.verifyArtifactFile(filepath.Join(sourceDir, rel), filepath.Join(installDir, rel)),
				Kind:   VerifyKindArtifact,
				Remedy: verifyArtifactRemedy,
			})
		}
	}
	return items
}

const verifyArtifactRemedy = "execute 'ai-spec-harness install' para reinstalar o artefato de governanca"

func (s *Service) verifyArtifactFile(sourcePath, installedPath string) VerifyState {
	if !s.fs.Exists(installedPath) {
		return VerifyStateMissing
	}
	sourceHash, srcErr := s.fs.FileHash(sourcePath)
	installedHash, dstErr := s.fs.FileHash(installedPath)
	if srcErr != nil || dstErr != nil {
		return VerifyStateUnknown
	}
	if sourceHash != installedHash {
		return VerifyStateDrifted
	}
	return VerifyStateCurrent
}

func (r1 *Helper) governanceHookFiles() []string {
	out := make([]string, 0, len(toolValidationHooks)+len(orchestratorHooks))
	out = append(out, toolValidationHooks...)
	out = append(out, orchestratorHooks...)
	return out
}

func (s *Service) writeMergedMarkdownFromSource(sourcePath, targetPath string) error {
	if !s.fs.Exists(sourcePath) {
		return nil
	}
	generated, err := s.fs.ReadFile(sourcePath)
	if err != nil {
		return err
	}
	content := string(generated)
	merged := false
	if existing, readErr := s.fs.ReadFile(targetPath); readErr == nil {
		content, merged = contextgen.MergeUserContentMarkdown(content, string(existing))
	}
	if err := s.fs.WriteFile(targetPath, []byte(content)); err != nil {
		return err
	}
	if merged {
		if err := s.trackMerged(targetPath); err != nil {
			return err
		}
	}
	return nil
}
