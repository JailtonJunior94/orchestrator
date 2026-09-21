package doctor

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"

	"github.com/JailtonJunior94/ai-spec-harness/internal/hookcontract"
	"github.com/JailtonJunior94/ai-spec-harness/internal/install"
	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/specs"
	"github.com/JailtonJunior94/ai-spec-harness/internal/skills"
)

func (s *Service) runHookChecks(projectDir string, items []install.VerifyItem) []Check {
	tools := installedToolsFrom(items)
	if len(tools) == 0 {
		return []Check{{
			Name:   "Adapters instalados",
			Status: "warn",
			Detail: "nenhum provedor instalado neste projeto; diagnostico de hooks pulado",
			Layer:  LayerAdapter,
		}}
	}

	violations := s.hookParityViolations(projectDir, tools)

	var checks []Check
	checks = append(checks, s.checkHookContractVersion(tools))
	checks = append(checks, checkHookAdapterMissing(violations))
	checks = append(checks, s.checkHookExecutable(projectDir, tools))
	checks = append(checks, checkHookNativeConfigInvalid(violations))
	checks = append(checks, checkHookAdapterDivergence(violations))
	return checks
}

func installedToolsFrom(items []install.VerifyItem) []skills.Tool {
	seen := make(map[skills.Tool]bool)
	var tools []skills.Tool
	for _, it := range items {
		if it.Tool == "" || seen[it.Tool] {
			continue
		}
		seen[it.Tool] = true
		tools = append(tools, it.Tool)
	}
	sort.Slice(tools, func(i, j int) bool { return tools[i] < tools[j] })
	return tools
}

func (s *Service) checkHookContractVersion(tools []skills.Tool) Check {
	wantEvents := len(hookcontract.EventKinds())
	var incompatible []string
	for _, tool := range tools {
		agent, err := specs.NewCatalog().AgentByID(string(tool))
		if err != nil {
			incompatible = append(incompatible, fmt.Sprintf("%s (agente ausente do catalogo)", tool))
			continue
		}
		gotPoints := len(agent.Enforcement().Coverage())
		if gotPoints != wantEvents {
			incompatible = append(incompatible, fmt.Sprintf("%s (%d/%d pontos canonicos)", tool, gotPoints, wantEvents))
		}
	}
	if len(incompatible) > 0 {
		sort.Strings(incompatible)
		return Check{
			Name:   "Versao do contrato de hooks",
			Status: "fail",
			Detail: fmt.Sprintf("contrato declara %d eventos canonicos; adapter(s) divergente(s): %s", wantEvents, strings.Join(incompatible, ", ")),
			Layer:  LayerAdapter,
		}
	}
	return Check{
		Name:   "Versao do contrato de hooks",
		Status: "ok",
		Detail: fmt.Sprintf("todos os %d adapter(s) instalado(s) cobrem os %d eventos canonicos do contrato", len(tools), wantEvents),
		Layer:  LayerAdapter,
	}
}

func (s *Service) hookParityViolations(projectDir string, tools []skills.Tool) []specs.ParityViolation {
	catalog := specs.NewCatalog()
	var cells []specs.AgentEnforcement
	requiredAgents := make([]string, 0, len(tools))
	for _, tool := range tools {
		agent, err := catalog.AgentByID(string(tool))
		if err != nil {
			continue
		}
		cells = append(cells, specs.AgentEnforcement{Agent: string(tool), Enforcement: agent.Enforcement()})
		requiredAgents = append(requiredAgents, string(tool))
	}

	resolve := func(relPath string) ([]byte, error) {
		if relPath == "" {
			return nil, fmt.Errorf("empty path")
		}
		return os.ReadFile(filepath.Join(projectDir, filepath.FromSlash(relPath)))
	}
	alwaysDispatchProven := func(string, specs.CanonicalPoint) bool { return true }

	return specs.ValidateParityMatrix(cells, requiredAgents, alwaysDispatchProven, resolve)
}

func checkHookAdapterMissing(violations []specs.ParityViolation) Check {
	var missing []string
	for _, v := range violations {
		if strings.Contains(v.Reason, "installed artifact") && strings.Contains(v.Reason, "does not exist on disk") {
			missing = append(missing, v.String())
		}
	}
	if len(missing) > 0 {
		sort.Strings(missing)
		return Check{
			Name:   "Adapter ausente",
			Status: "fail",
			Detail: fmt.Sprintf("%d adapter(s) ausente(s): %s", len(missing), strings.Join(missing, "; ")),
			Layer:  LayerAdapter,
		}
	}
	return Check{Name: "Adapter ausente", Status: "ok", Detail: "todos os adapters declarados estao instalados", Layer: LayerAdapter}
}

func checkHookNativeConfigInvalid(violations []specs.ParityViolation) Check {
	var invalid []string
	for _, v := range violations {
		if strings.Contains(v.Reason, "is missing, so native key") || strings.Contains(v.Reason, "does not declare native key") {
			invalid = append(invalid, v.String())
		}
	}
	if len(invalid) > 0 {
		sort.Strings(invalid)
		return Check{
			Name:   "Configuracao do hook invalida",
			Status: "fail",
			Detail: fmt.Sprintf("%d configuracao(oes) nativa(s) invalida(s): %s", len(invalid), strings.Join(invalid, "; ")),
			Layer:  LayerAdapter,
		}
	}
	return Check{Name: "Configuracao do hook invalida", Status: "ok", Detail: "configuracao nativa declara os pontos canonicos esperados", Layer: LayerAdapter}
}

func checkHookAdapterDivergence(violations []specs.ParityViolation) Check {
	var diverged []string
	for _, v := range violations {
		switch {
		case v.Reason == "missing canonical point coverage":
			diverged = append(diverged, v.String())
		case strings.Contains(v.Reason, "neither mirrors nor executes the canonical validator"):
			diverged = append(diverged, v.String())
		case strings.Contains(v.Reason, "without wiring it to the canonical validator"):
			diverged = append(diverged, v.String())
		}
	}
	if len(diverged) > 0 {
		sort.Strings(diverged)
		return Check{
			Name:   "Divergencia entre contrato e adapter",
			Status: "fail",
			Detail: fmt.Sprintf("%d divergencia(s): %s", len(diverged), strings.Join(diverged, "; ")),
			Layer:  LayerAdapter,
		}
	}
	return Check{Name: "Divergencia entre contrato e adapter", Status: "ok", Detail: "nenhuma divergencia entre o contrato e os adapters instalados", Layer: LayerAdapter}
}

func (s *Service) checkHookExecutable(projectDir string, tools []skills.Tool) Check {
	if runtime.GOOS == "windows" {
		return Check{Name: "Hook executavel", Status: "ok", Detail: "bit de execucao nao e aplicavel neste sistema operacional", Layer: LayerAdapter}
	}

	catalog := specs.NewCatalog()
	var nonExecutable []string
	seen := make(map[string]bool)
	for _, tool := range tools {
		agent, err := catalog.AgentByID(string(tool))
		if err != nil {
			continue
		}
		for _, cov := range agent.Enforcement().Coverage() {
			artifact := cov.ArtifactPath()
			if artifact == "" || seen[artifact] {
				continue
			}
			seen[artifact] = true
			fullPath := filepath.Join(projectDir, filepath.FromSlash(artifact))
			info, statErr := os.Stat(fullPath)
			if statErr != nil {
				continue
			}
			if info.Mode()&0o111 == 0 {
				nonExecutable = append(nonExecutable, artifact)
			}
		}
	}
	if len(nonExecutable) > 0 {
		sort.Strings(nonExecutable)
		return Check{
			Name:   "Hook executavel",
			Status: "fail",
			Detail: fmt.Sprintf("%d hook(s) instalado(s) sem bit de execucao: %s", len(nonExecutable), strings.Join(nonExecutable, ", ")),
			Layer:  LayerAdapter,
		}
	}
	return Check{Name: "Hook executavel", Status: "ok", Detail: "todos os hooks instalados sao executaveis", Layer: LayerAdapter}
}
