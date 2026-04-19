package detect

import (
	"fmt"
	"path/filepath"

	"github.com/JailtonJunior94/ai-spec-harness/internal/fs"
)

// ArchitectureType representa o tipo de arquitetura detectada.
type ArchitectureType string

const (
	ArchMonorepo        ArchitectureType = "monorepo"
	ArchModular         ArchitectureType = "monolito modular"
	ArchMicroservice    ArchitectureType = "microservico"
	ArchMonolith        ArchitectureType = "monolito"
)

// ArchitectureResult agrupa a deteccao de arquitetura e padrao.
type ArchitectureResult struct {
	Type    ArchitectureType
	Pattern string
}

// ArchitectureDetector detecta tipo e padrao arquitetural de um projeto.
type ArchitectureDetector struct {
	fs fs.FileSystem
}

func NewArchitectureDetector(fsys fs.FileSystem) *ArchitectureDetector {
	return &ArchitectureDetector{fs: fsys}
}

func (d *ArchitectureDetector) Detect(projectDir string) ArchitectureResult {
	return ArchitectureResult{
		Type:    d.detectType(projectDir),
		Pattern: d.detectPattern(projectDir),
	}
}

func (d *ArchitectureDetector) detectType(projectDir string) ArchitectureType {
	// Monorepo: sinais fortes de multiplos projetos independentes
	monorepoIndicators := []string{"go.work", "pnpm-workspace.yaml", "nx.json", "turbo.json", "lerna.json"}
	for _, f := range monorepoIndicators {
		if d.fs.Exists(filepath.Join(projectDir, f)) {
			return ArchMonorepo
		}
	}
	if d.hasAnyFiles(projectDir, "services") && d.hasAnyFiles(projectDir, "packages") {
		return ArchMonorepo
	}
	if d.hasAnyFiles(projectDir, "apps") && d.hasAnyFiles(projectDir, "packages") {
		return ArchMonorepo
	}

	// Monolito modular
	if d.hasAnyFiles(projectDir, "modules") || d.hasAnyFiles(projectDir, "domains") {
		return ArchModular
	}
	internalDir := filepath.Join(projectDir, "internal")
	if d.fs.IsDir(internalDir) {
		entries, err := d.fs.ReadDir(internalDir)
		if err == nil {
			subdirCount := 0
			for _, e := range entries {
				if e.IsDir() {
					subdirCount++
				}
			}
			if subdirCount >= 3 {
				return ArchModular
			}
		}
	}

	// Microservico: Dockerfile + sinais de deploy isolado
	if d.fs.Exists(filepath.Join(projectDir, "Dockerfile")) {
		deployDirs := []string{"deployments", "k8s", "helm"}
		deployFiles := []string{"skaffold.yaml", "kustomization.yaml"}
		for _, dir := range deployDirs {
			if d.hasAnyFiles(projectDir, dir) {
				return ArchMicroservice
			}
		}
		for _, f := range deployFiles {
			if d.fs.Exists(filepath.Join(projectDir, f)) {
				return ArchMicroservice
			}
		}
	}

	return ArchMonolith
}

func (d *ArchitectureDetector) detectPattern(projectDir string) string {
	cleanArchDirs := []string{"domain", "application", "infrastructure", "ports", "adapters"}
	for _, dir := range cleanArchDirs {
		if d.hasAnyFiles(projectDir, dir) {
			return "Predominio de Clean Architecture / Hexagonal com fronteiras explicitas entre dominio, aplicacao e infraestrutura."
		}
	}

	layeredDirs := []string{"controllers", "services", "repositories", "models"}
	for _, dir := range layeredDirs {
		if d.hasAnyFiles(projectDir, dir) {
			return "Predominio de arquitetura em camadas, com separacao entre transporte, servicos, persistencia e modelos."
		}
	}

	if d.hasAnyFiles(projectDir, "features") || d.hasAnyFiles(projectDir, "feature") {
		return "Predominio de organizacao por funcionalidade / fatiamento vertical, agrupando fluxo e dependencias por capacidade de negocio."
	}

	if d.hasAnyFiles(projectDir, "internal") {
		return "Predominio de packages internos coesos, com estrutura orientada por dominio ou componente."
	}

	return "Padrao arquitetural nao inferido com alta confianca; assumir composicao simples e dependencias explicitas."
}

func (d *ArchitectureDetector) hasAnyFiles(projectDir, subdir string) bool {
	dir := filepath.Join(projectDir, subdir)
	if !d.fs.IsDir(dir) {
		return false
	}
	entries, err := d.fs.ReadDir(dir)
	if err != nil {
		return false
	}
	return len(entries) > 0
}

// DescribeArchitecture retorna descricao formatada da arquitetura detectada.
func DescribeArchitecture(archType ArchitectureType, stack, frameworks string) string {
	switch archType {
	case ArchMonorepo:
		return fmt.Sprintf(`O projeto aparenta ser um monorepo, com multiplos componentes ou workspaces sob a mesma raiz. A governanca deve preservar fronteiras entre pacotes e validar apenas os workspaces afetados.

Stack detectada: %s.
Frameworks detectados: %s.`, stack, frameworks)
	case ArchModular:
		return fmt.Sprintf(`O projeto aparenta ser um monolito modular, com separacao relevante por modulos, dominios ou componentes internos. A governanca deve proteger essas fronteiras e evitar dependencias circulares.

Stack detectada: %s.
Frameworks detectados: %s.`, stack, frameworks)
	case ArchMicroservice:
		return fmt.Sprintf(`O projeto aparenta ser um microservico independente, com foco em contrato de API, inicializacao, dependencias externas e seguranca operacional. A governanca deve preservar o escopo do servico e o seu deploy independente.

Stack detectada: %s.
Frameworks detectados: %s.`, stack, frameworks)
	default:
		return fmt.Sprintf(`O projeto aparenta ser um monolito unico. A governanca deve privilegiar coesao local, limites de pacote claros e crescimento incremental da estrutura.

Stack detectada: %s.
Frameworks detectados: %s.`, stack, frameworks)
	}
}

// ArchitectureRules retorna regras especificas para o tipo de arquitetura.
func ArchitectureRules(archType ArchitectureType) string {
	switch archType {
	case ArchMonorepo:
		return `## Regras por Arquitetura

1. Limitar mudancas ao workspace, pacote ou servico afetado.
2. Nao criar dependencias internas cruzadas sem contrato explicito.
3. Validar primeiro apenas os workspaces impactados antes de ampliar o escopo.`
	case ArchModular:
		return `## Regras por Arquitetura

1. Respeitar fronteiras entre modulos e bounded contexts.
2. Evitar dependencia circular entre packages internos.
3. Nao extrair shared helpers sem demanda comprovada de mais de um modulo.`
	case ArchMicroservice:
		return `## Regras por Arquitetura

1. Preservar contratos publicados e compatibilidade de integracao.
2. Manter inicializacao, observabilidade e shutdown como parte do comportamento do servico.
3. Nao acoplar o servico a convencoes de outros servicos sem contrato explicito.`
	default:
		return `## Regras por Arquitetura

1. Preservar coesao local e dependencia unidirecional entre packages.
2. Evitar helpers transversais que escondam regra de negocio ou IO.
3. Crescer a estrutura apenas quando o codigo atual ja nao comportar a mudanca com clareza.`
	}
}

// ArchitectureRestrictions retorna restricoes extras por arquitetura.
func ArchitectureRestrictions(archType ArchitectureType) string {
	switch archType {
	case ArchMonorepo:
		return "\n5. Nao alterar contratos entre workspaces sem deixar o impacto explicito."
	case ArchMicroservice:
		return "\n5. Nao alterar contratos externos, readiness, observabilidade ou semantica operacional sem explicitar a mudanca."
	default:
		return ""
	}
}
