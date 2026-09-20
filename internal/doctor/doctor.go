package doctor

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/JailtonJunior94/ai-spec-harness/internal/adapters"
	"github.com/JailtonJunior94/ai-spec-harness/internal/config"
	"github.com/JailtonJunior94/ai-spec-harness/internal/contextgen"
	"github.com/JailtonJunior94/ai-spec-harness/internal/fs"
	"github.com/JailtonJunior94/ai-spec-harness/internal/git"
	"github.com/JailtonJunior94/ai-spec-harness/internal/harness"
	"github.com/JailtonJunior94/ai-spec-harness/internal/install"
	"github.com/JailtonJunior94/ai-spec-harness/internal/manifest"
	"github.com/JailtonJunior94/ai-spec-harness/internal/output"
	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/precondition"
	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/specs"
	"github.com/JailtonJunior94/ai-spec-harness/internal/skills"
	"github.com/JailtonJunior94/ai-spec-harness/internal/skillscheck"
)

const codexTrustDoctorTimeout = 5 * time.Second

type FailureLayer string

const (
	LayerCore         FailureLayer = "core"
	LayerInstallation FailureLayer = "installation"
	LayerAdapter      FailureLayer = "adapter"
	LayerProvider     FailureLayer = "provider"
	LayerValidation   FailureLayer = "validation"
)

type Check struct {
	Name   string
	Status string
	Detail string
	Layer  FailureLayer
}

type ProviderBlock struct {
	Tool   skills.Tool
	Name   string
	Checks []Check
}

type Service struct {
	fs            fs.FileSystem
	printer       *output.Printer
	manifest      *manifest.Store
	git           git.Repository
	install       *install.Service
	skillcheck    *skillscheck.Service
	harnessLoader harness.Loader
	lookPath      func(name string) (string, error)
}

func NewService(fsys fs.FileSystem, printer *output.Printer, mfst *manifest.Store, gitRepo git.Repository) *Service {
	adpt := adapters.NewGenerator(fsys, printer)
	ctxg := contextgen.NewGenerator(fsys, printer)
	return &Service{
		fs:            fsys,
		printer:       printer,
		manifest:      mfst,
		git:           gitRepo,
		install:       install.NewService(fsys, printer, mfst, adpt, ctxg),
		skillcheck:    skillscheck.NewService(fsys, printer),
		harnessLoader: harness.NewDefaultLoader(fsys),
		lookPath:      exec.LookPath,
	}
}

func (s *Service) Execute(projectDir string) error {
	return s.ExecuteWithOptions(projectDir, false)
}

func (s *Service) ExecuteWithOptions(projectDir string, checkCodexTrust bool) error {
	absDir, err := filepath.Abs(projectDir)
	if err != nil {
		return err
	}

	if !s.fs.IsDir(absDir) {
		return fmt.Errorf("diretorio nao encontrado: %s", absDir)
	}

	s.printer.Info("Diagnosticando: %s", absDir)
	s.printer.Info("")

	items, verifyErr := s.install.Verify(config.InstallOptions{
		ProjectDir:      absDir,
		CheckCodexTrust: checkCodexTrust,
	})

	coreChecks := s.runCoreChecks(absDir, items, verifyErr)
	providerBlocks := s.buildProviderBlocks(absDir, items, checkCodexTrust)

	var failCount int

	s.printer.Info("Core:")
	failCount += s.printChecks(coreChecks)

	for _, block := range providerBlocks {
		s.printer.Info("")
		s.printer.Info("%s:", block.Name)
		failCount += s.printChecks(block.Checks)
	}

	s.printer.Info("")
	if failCount > 0 {
		s.printer.Info("Resultado: %d problema(s) encontrado(s)", failCount)
		return fmt.Errorf("%d problema(s) detectado(s)", failCount)
	}
	s.printer.Info("Resultado: tudo ok")
	return nil
}

func (s *Service) printChecks(checks []Check) int {
	var fail int
	for _, c := range checks {
		icon := "OK"
		switch c.Status {
		case "warn":
			icon = "AVISO"
		case "fail":
			icon = "FALHA"
			fail++
		}
		s.printer.Info("  [%-5s] %-40s %s (camada: %s)", icon, c.Name, c.Detail, c.Layer)
	}
	return fail
}

func (s *Service) runCoreChecks(projectDir string, items []install.VerifyItem, verifyErr error) []Check {
	var checks []Check
	checks = append(checks, s.checkGit(projectDir))
	checks = append(checks, s.checkSkillsDir(projectDir))
	checks = append(checks, s.checkManifest(projectDir))
	checks = append(checks, s.checkSymlinks(projectDir)...)
	checks = append(checks, s.checkPermissions(projectDir))
	checks = append(checks, s.checkGitBinary())
	checks = append(checks, s.checkHookInterpreter())
	checks = append(checks, s.checkContract(projectDir))
	checks = append(checks, s.checkSkillIntegrity(projectDir))
	checks = append(checks, s.checkCanonicalSync(items, verifyErr))
	return checks
}

func (s *Service) checkGit(projectDir string) Check {
	if s.git.IsRepo(projectDir) {
		return Check{Name: "Repositorio git", Status: "ok", Detail: "valido", Layer: LayerCore}
	}
	return Check{Name: "Repositorio git", Status: "warn", Detail: "nao e um repositorio git", Layer: LayerCore}
}

func (s *Service) checkSkillsDir(projectDir string) Check {
	skillsDir := filepath.Join(projectDir, ".agents", "skills")
	if s.fs.IsDir(skillsDir) {
		entries, err := s.fs.ReadDir(skillsDir)
		if err == nil {
			count := 0
			for _, e := range entries {
				if e.IsDir() {
					count++
				}
			}
			return Check{Name: "Diretorio de skills", Status: "ok", Detail: fmt.Sprintf("%d skills encontradas", count), Layer: LayerCore}
		}
	}
	return Check{Name: "Diretorio de skills", Status: "fail", Detail: ".agents/skills/ ausente", Layer: LayerInstallation}
}

func (s *Service) checkManifest(projectDir string) Check {
	if s.manifest.Exists(projectDir) {
		mf, err := s.manifest.Load(projectDir)
		if err != nil {
			return Check{Name: "Manifesto", Status: "warn", Detail: fmt.Sprintf("corrompido: %v", err), Layer: LayerCore}
		}
		return Check{Name: "Manifesto", Status: "ok", Detail: fmt.Sprintf("versao %s", mf.Version), Layer: LayerCore}
	}
	return Check{Name: "Manifesto", Status: "warn", Detail: ".ai_spec_harness.json nao encontrado", Layer: LayerCore}
}

func (s *Service) checkSymlinks(projectDir string) []Check {
	var checks []Check
	skillsDir := filepath.Join(projectDir, ".agents", "skills")
	if !s.fs.IsDir(skillsDir) {
		return checks
	}

	entries, err := s.fs.ReadDir(skillsDir)
	if err != nil {
		return checks
	}

	brokenCount := 0
	for _, e := range entries {
		path := filepath.Join(skillsDir, e.Name())
		if s.fs.IsSymlink(path) {
			if !s.fs.Exists(path) {
				brokenCount++
			}
		}
	}

	if brokenCount > 0 {
		checks = append(checks, Check{
			Name:   "Symlinks de skills",
			Status: "fail",
			Detail: fmt.Sprintf("%d symlink(s) quebrado(s)", brokenCount),
			Layer:  LayerInstallation,
		})
	} else {
		checks = append(checks, Check{
			Name:   "Symlinks de skills",
			Status: "ok",
			Detail: "todos os links validos",
			Layer:  LayerCore,
		})
	}

	return checks
}

func (s *Service) checkPermissions(projectDir string) Check {
	if s.fs.Writable(projectDir) {
		return Check{Name: "Permissoes de escrita", Status: "ok", Detail: "diretorio gravavel", Layer: LayerCore}
	}
	return Check{Name: "Permissoes de escrita", Status: "fail", Detail: "sem permissao de escrita", Layer: LayerInstallation}
}

func (s *Service) checkGitBinary() Check {
	if _, err := exec.LookPath("git"); err != nil {
		return Check{Name: "Git instalado", Status: "fail", Detail: "git nao encontrado no PATH", Layer: LayerInstallation}
	}
	return Check{Name: "Git instalado", Status: "ok", Detail: "disponivel", Layer: LayerCore}
}

func (s *Service) checkHookInterpreter() Check {
	_, python3Err := s.lookPath("python3")
	if python3Err == nil {
		return Check{Name: "Interpretador de payload de hook", Status: "ok", Detail: "python3 disponivel", Layer: LayerInstallation}
	}
	_, jqErr := s.lookPath("jq")
	if jqErr == nil {
		return Check{Name: "Interpretador de payload de hook", Status: "ok", Detail: "jq disponivel (python3 ausente)", Layer: LayerInstallation}
	}
	return Check{
		Name:   "Interpretador de payload de hook",
		Status: "fail",
		Detail: "python3 e jq ausentes no PATH — hook-payload.sh falha fechado (bloqueia) em toda operacao git e validacao de governanca",
		Layer:  LayerInstallation,
	}
}

func (s *Service) checkContract(projectDir string) Check {
	contract, source, err := s.harnessLoader.Load(projectDir)
	if err != nil {
		return Check{Name: "Contrato do harness", Status: "fail", Detail: fmt.Sprintf("invalido: %v", err), Layer: LayerCore}
	}
	if source == harness.SourceDefault {
		return Check{
			Name:   "Contrato do harness",
			Status: "warn",
			Detail: fmt.Sprintf(".agents/harness.yaml nao declarado; usando default embutido (versao %d)", contract.Version),
			Layer:  LayerCore,
		}
	}
	return Check{Name: "Contrato do harness", Status: "ok", Detail: fmt.Sprintf("valido, versao %d", contract.Version), Layer: LayerCore}
}

func (s *Service) checkSkillIntegrity(projectDir string) Check {
	failures, err := s.skillcheck.Verify(projectDir)
	if err != nil {
		return Check{
			Name:   "Integridade de skills (lock)",
			Status: "warn",
			Detail: fmt.Sprintf("skills-lock.json indisponivel: %v", err),
			Layer:  LayerValidation,
		}
	}
	if len(failures) > 0 {
		names := make([]string, 0, len(failures))
		for _, f := range failures {
			names = append(names, fmt.Sprintf("%s (%s)", f.Check.Name, f.Reason))
		}
		sort.Strings(names)
		return Check{
			Name:   "Integridade de skills (lock)",
			Status: "fail",
			Detail: fmt.Sprintf("%d divergencia(s): %s", len(failures), strings.Join(names, "; ")),
			Layer:  LayerValidation,
		}
	}
	return Check{
		Name:   "Integridade de skills (lock)",
		Status: "ok",
		Detail: "versao + hash SHA-256 conferem com skills-lock.json",
		Layer:  LayerValidation,
	}
}

func (s *Service) checkCanonicalSync(items []install.VerifyItem, verifyErr error) Check {
	if verifyErr != nil {
		return Check{
			Name:   "Sincronia canonica/derivados",
			Status: "warn",
			Detail: fmt.Sprintf("verificacao indisponivel: %v", verifyErr),
			Layer:  LayerCore,
		}
	}

	var drifted []string
	for _, it := range items {
		if it.State == install.VerifyStateDrifted {
			label := it.Skill
			if it.Tool != "" {
				label = fmt.Sprintf("%s[%s]", it.Skill, it.Tool)
			}
			drifted = append(drifted, label)
		}
	}
	if len(drifted) > 0 {
		sort.Strings(drifted)
		return Check{
			Name:   "Sincronia canonica/derivados",
			Status: "fail",
			Detail: fmt.Sprintf("%d artefato(s) divergente(s) da origem canonica: %s", len(drifted), strings.Join(drifted, ", ")),
			Layer:  LayerAdapter,
		}
	}
	return Check{
		Name:   "Sincronia canonica/derivados",
		Status: "ok",
		Detail: fmt.Sprintf("%d item(ns) verificados, nenhuma divergencia", len(items)),
		Layer:  LayerCore,
	}
}

func (s *Service) buildProviderBlocks(projectDir string, items []install.VerifyItem, checkCodexTrust bool) []ProviderBlock {
	byTool := make(map[skills.Tool][]install.VerifyItem)
	var validators []install.VerifyItem
	for _, it := range items {
		if it.Tool == "" {
			if it.Kind == install.VerifyKindArtifact {
				validators = append(validators, it)
			}
			continue
		}
		byTool[it.Tool] = append(byTool[it.Tool], it)
	}

	var blocks []ProviderBlock
	for _, tool := range skills.AllTools {
		toolItems, ok := byTool[tool]
		if !ok {
			continue
		}
		checks := s.buildProviderChecks(projectDir, tool, toolItems, validators)
		if tool == skills.ToolCodex && checkCodexTrust {
			checks = append(checks, s.checkCodexTrustedHash(projectDir))
		}
		blocks = append(blocks, ProviderBlock{
			Tool:   tool,
			Name:   providerDisplayName(tool),
			Checks: checks,
		})
	}
	return blocks
}

func providerDisplayName(tool skills.Tool) string {
	agent, err := specs.NewCatalog().AgentByID(string(tool))
	if err != nil {
		return string(tool)
	}
	return agent.Identity().DisplayName()
}

func (s *Service) buildProviderChecks(projectDir string, tool skills.Tool, items, validators []install.VerifyItem) []Check {
	var checks []Check
	checks = append(checks, s.checkProviderInstructions(projectDir, tool))
	checks = append(checks, s.checkProviderSkills(items))
	checks = append(checks, s.checkProviderPolicies(items))
	checks = append(checks, s.checkProviderEvidenceValidators(validators))
	checks = append(checks, s.checkProviderPreconditions(items))
	return checks
}

func (s *Service) checkProviderInstructions(projectDir string, tool skills.Tool) Check {
	paths := specs.NativeConfigPaths(string(tool))
	if len(paths) == 0 {
		return Check{Name: "Instrucoes descobriveis", Status: "warn", Detail: "nenhuma configuracao nativa declarada para este provedor", Layer: LayerAdapter}
	}

	var present, missing []string
	for _, p := range paths {
		if s.fs.Exists(filepath.Join(projectDir, filepath.FromSlash(p))) {
			present = append(present, p)
		} else {
			missing = append(missing, p)
		}
	}

	if len(present) == 0 {
		return Check{
			Name:   "Instrucoes descobriveis",
			Status: "fail",
			Detail: fmt.Sprintf("nenhuma configuracao nativa encontrada (esperado: %s)", strings.Join(paths, ", ")),
			Layer:  LayerInstallation,
		}
	}
	if len(missing) > 0 {
		return Check{
			Name:   "Instrucoes descobriveis",
			Status: "warn",
			Detail: fmt.Sprintf("parcialmente descobrivel: presente=%s ausente=%s", strings.Join(present, ","), strings.Join(missing, ",")),
			Layer:  LayerAdapter,
		}
	}
	return Check{
		Name:   "Instrucoes descobriveis",
		Status: "ok",
		Detail: fmt.Sprintf("%d configuracao(oes) nativa(s) presente(s)", len(present)),
		Layer:  LayerAdapter,
	}
}

func (s *Service) checkProviderSkills(items []install.VerifyItem) Check {
	var total, missing, drifted int
	for _, it := range items {
		if it.Kind != install.VerifyKindSkill {
			continue
		}
		total++
		switch it.State {
		case install.VerifyStateMissing:
			missing++
		case install.VerifyStateDrifted:
			drifted++
		}
	}
	if total == 0 {
		return Check{Name: "Skills disponiveis", Status: "warn", Detail: "nenhuma skill verificavel encontrada", Layer: LayerInstallation}
	}
	if missing > 0 {
		return Check{Name: "Skills disponiveis", Status: "fail", Detail: fmt.Sprintf("%d/%d skill(s) ausente(s)", missing, total), Layer: LayerInstallation}
	}
	if drifted > 0 {
		return Check{Name: "Skills disponiveis", Status: "warn", Detail: fmt.Sprintf("%d/%d skill(s) divergente(s) da origem canonica", drifted, total), Layer: LayerAdapter}
	}
	return Check{Name: "Skills disponiveis", Status: "ok", Detail: fmt.Sprintf("%d skill(s) atuais", total), Layer: LayerCore}
}

func (s *Service) checkProviderPolicies(items []install.VerifyItem) Check {
	var total, missing, drifted int
	for _, it := range items {
		if it.Kind != install.VerifyKindArtifact {
			continue
		}
		total++
		switch it.State {
		case install.VerifyStateMissing:
			missing++
		case install.VerifyStateDrifted:
			drifted++
		}
	}
	if total == 0 {
		return Check{Name: "Politicas aplicadas", Status: "warn", Detail: "nenhum artefato de governanca (hooks) verificavel", Layer: LayerAdapter}
	}
	if missing > 0 {
		return Check{Name: "Politicas aplicadas", Status: "fail", Detail: fmt.Sprintf("%d/%d artefato(s) de politica ausente(s)", missing, total), Layer: LayerInstallation}
	}
	if drifted > 0 {
		return Check{Name: "Politicas aplicadas", Status: "warn", Detail: fmt.Sprintf("%d/%d artefato(s) de politica divergente(s)", drifted, total), Layer: LayerAdapter}
	}
	return Check{Name: "Politicas aplicadas", Status: "ok", Detail: fmt.Sprintf("%d artefato(s) de politica aplicados", total), Layer: LayerAdapter}
}

func (s *Service) checkProviderEvidenceValidators(validators []install.VerifyItem) Check {
	var missing, drifted int
	for _, it := range validators {
		switch it.State {
		case install.VerifyStateMissing:
			missing++
		case install.VerifyStateDrifted:
			drifted++
		}
	}
	if missing > 0 || drifted > 0 {
		return Check{
			Name:   "Validadores de evidencia instalados",
			Status: "warn",
			Detail: fmt.Sprintf("compartilhados (.agents/{hooks,scripts,lib}): %d ausente(s), %d divergente(s) — ver bloco Core", missing, drifted),
			Layer:  LayerValidation,
		}
	}
	return Check{
		Name:   "Validadores de evidencia instalados",
		Status: "ok",
		Detail: fmt.Sprintf("%d validador(es) canonico(s) atuais", len(validators)),
		Layer:  LayerValidation,
	}
}

func (s *Service) checkProviderPreconditions(items []install.VerifyItem) Check {
	var unmet []string
	var binaryMissing bool
	for _, it := range items {
		switch it.Kind {
		case install.VerifyKindPrecondition:
			if it.State != install.VerifyStateCurrent && it.State != install.VerifyStateUnknown {
				unmet = append(unmet, fmt.Sprintf("%s(%s)", it.Skill, it.State))
			}
		case install.VerifyKindBinary:
			if it.State == install.VerifyStateMissing {
				binaryMissing = true
			}
		}
	}
	if binaryMissing {
		return Check{
			Name:   "Pre-condicoes de enforcement",
			Status: "warn",
			Detail: "binario ACP nao encontrado no PATH — enforcement nao verificavel em runtime",
			Layer:  LayerProvider,
		}
	}
	if len(unmet) > 0 {
		sort.Strings(unmet)
		return Check{
			Name:   "Pre-condicoes de enforcement",
			Status: "fail",
			Detail: fmt.Sprintf("pre-condicao(oes) nao atendida(s): %s", strings.Join(unmet, ", ")),
			Layer:  LayerProvider,
		}
	}
	return Check{Name: "Pre-condicoes de enforcement", Status: "ok", Detail: "nenhuma pre-condicao pendente conhecida", Layer: LayerProvider}
}

func (s *Service) checkCodexTrustedHash(projectDir string) Check {
	agent, err := specs.NewCatalog().AgentByID("codex")
	if err != nil {
		return Check{Name: "Trust de hooks do Codex", Status: "warn", Detail: "unknown — agente codex ausente no catalogo", Layer: LayerProvider}
	}

	required := make([]string, 0, len(agent.Enforcement().Coverage()))
	for _, cov := range agent.Enforcement().Coverage() {
		if cov.ScriptPath() == "" {
			continue
		}
		required = append(required, cov.NativeKey())
	}

	client := precondition.NewCodexAppServerClient("", projectDir)
	ctx, cancel := context.WithTimeout(context.Background(), codexTrustDoctorTimeout)
	defer cancel()

	report, err := precondition.EvaluateCodexTrustedHash(ctx, client, codexTrustDoctorTimeout, required)
	if err != nil {
		return Check{Name: "Trust de hooks do Codex", Status: "warn", Detail: fmt.Sprintf("unknown — RPC hooks/list falhou: %v", err), Layer: LayerProvider}
	}

	untrusted := make([]string, 0, len(report.Points))
	for _, point := range report.Points {
		if point.State != specs.PreconditionCurrent {
			untrusted = append(untrusted, point.EventName)
		}
	}

	switch report.State() {
	case specs.PreconditionCurrent:
		return Check{Name: "Trust de hooks do Codex", Status: "ok", Detail: fmt.Sprintf("todos os pontos canonicos confiados em %s (%s)", projectDir, strings.Join(required, ", ")), Layer: LayerProvider}
	case specs.PreconditionInert:
		return Check{Name: "Trust de hooks do Codex", Status: "fail", Detail: fmt.Sprintf("pontos sem trust de projeto em %s: %s — conceda via TUI interativa (/hooks) antes de orquestrar", projectDir, strings.Join(untrusted, ", ")), Layer: LayerProvider}
	default:
		return Check{Name: "Trust de hooks do Codex", Status: "warn", Detail: "unknown — sem informacao de trust", Layer: LayerProvider}
	}
}
