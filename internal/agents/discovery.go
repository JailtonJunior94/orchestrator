package agents

import (
	"fmt"
	"log"
	"path/filepath"
	"sort"

	"github.com/JailtonJunior94/ai-spec-harness/internal/fs"
)

const (
	// _agentsSubdir e o subdiretorio onde os agentes sao descobertos.
	_agentsSubdir = ".ai-harness/agents"
	// _agentFileName e o nome do arquivo de definicao do agente.
	_agentFileName = "AGENT.md"
)

// discoverAgents escaneia root/.ai-harness/agents/*/AGENT.md usando fsys e retorna
// os agentes validos encontrados com o escopo indicado. Subdiretórios sem AGENT.md valido
// sao silenciosamente ignorados. Falhas de leitura em um agente nao interrompem a descoberta
// dos demais — o erro e registrado em nivel info e a iteracao continua (decisao: erro agregado
// por robustez, conforme task 3.0 requirement).
func (c *Catalog) discoverAgents(fsys fs.FileSystem, scope Scope, root string) ([]ResolvedAgent, error) {
	agentsDir := filepath.Join(root, _agentsSubdir)
	if !fsys.Exists(agentsDir) {
		return nil, nil
	}

	entries, err := fsys.ReadDir(agentsDir)
	if err != nil {
		return nil, fmt.Errorf("ler diretorio de agentes %q: %w", agentsDir, err)
	}

	var agents []ResolvedAgent
	var errs []error

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		dirName := entry.Name()
		agentPath := filepath.Join(agentsDir, dirName, _agentFileName)

		if !fsys.Exists(agentPath) {
			continue
		}

		content, err := fsys.ReadFile(agentPath)
		if err != nil {
			// Falha de leitura: registra e continua (nao interrompe descoberta dos demais).
			log.Printf("info: agente %q ignorado: erro ao ler %q: %v", dirName, agentPath, err)
			errs = append(errs, fmt.Errorf("ler agente %q: %w", agentPath, err))
			continue
		}

		agent, err := NewCatalog().ValidateAgentFrontmatter(content, dirName)
		if err != nil {
			// Frontmatter invalido: registra e continua.
			log.Printf("info: agente %q ignorado: frontmatter invalido em %q: %v", dirName, agentPath, err)
			errs = append(errs, fmt.Errorf("validar agente %q: %w", agentPath, err))
			continue
		}

		agent.Scope = scope
		agent.Path = agentPath
		agents = append(agents, agent)
	}

	// Ordenar lexicograficamente por name para determinismo (RF-16, ordem do catalogo).
	sort.Slice(agents, func(i, j int) bool {
		return agents[i].Name < agents[j].Name
	})

	// Retornar erro agregado se houver falhas parciais (sem interromper os agentes validos).
	if len(errs) > 0 {
		// Retornar agentes validos junto com o primeiro erro para indicar falhas parciais.
		// O caller (Registry.Discover) decide como expor erros agregados.
		return agents, errs[0]
	}

	return agents, nil
}

// mergeWithShadowing combina agentes globais e workspace aplicando a regra de precedencia:
// workspace prevalece em colisao de nome; o agente global homônimo e marcado como shadowed
// e logado em nivel info (RF-03, D-07). Retorna a lista merged (com workspace ganhando)
// e a lista de agentes globais que foram ofuscados.
func (c *Catalog) mergeWithShadowing(global, workspace []ResolvedAgent) (merged, shadowed []ResolvedAgent) {
	// Construir mapa de nomes workspace para deteccao rapida de colisao.
	workspaceByName := make(map[string]struct{}, len(workspace))
	for _, a := range workspace {
		workspaceByName[a.Name] = struct{}{}
	}

	var result []ResolvedAgent

	// Adicionar agentes globais que nao colidem; marcar os que colidem como shadowed.
	for _, g := range global {
		if _, collision := workspaceByName[g.Name]; collision {
			shadowed = append(shadowed, g)
			log.Printf("info: agente global %q ofuscado pelo workspace (shadowing)", g.Name)
		} else {
			result = append(result, g)
		}
	}

	// Adicionar todos os agentes workspace (sempre prevalecem).
	result = append(result, workspace...)

	// Ordenar merged lexicograficamente por name para determinismo.
	sort.Slice(result, func(i, j int) bool {
		return result[i].Name < result[j].Name
	})

	return result, shadowed
}
