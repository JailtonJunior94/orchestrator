package agents

import (
	"fmt"
	"sort"
	"strings"
)

// _catalogLimit e o numero maximo de entradas no bloco de catalogo (RF-16).
const _catalogLimit = 200

// BuildAgentBlocks produz dois blocos textuais para enriquecimento de prompt (RF-15, RF-16):
//   - metadata: bloco Markdown com informacoes do agente ativo.
//   - catalogBlock: bloco Markdown com lista dos agentes disponiveis, ordenada lexicograficamente
//     por name, com o agente ativo marcado com [active]. Entradas excedentes a 200 sao descartadas.
//
// Funcao pura: sem IO, sem cache, sem estado externo.
func (c *Catalog) BuildAgentBlocks(agent *ResolvedAgent, catalog []ResolvedAgent) (metadata, catalogBlock string) {
	metadata = NewCatalog().buildMetadataBlock(agent)
	catalogBlock = NewCatalog().buildCatalogBlock(agent, catalog)
	return metadata, catalogBlock
}

// buildMetadataBlock constroi o bloco de metadados do agente ativo.
// Retorna string vazia quando agent e nil.
func (c *Catalog) buildMetadataBlock(agent *ResolvedAgent) string {
	if agent == nil {
		return ""
	}
	var sb strings.Builder
	sb.WriteString("### Agente Ativo\n\n")
	fmt.Fprintf(&sb, "- **Nome**: %s\n", agent.Name)
	fmt.Fprintf(&sb, "- **Versao**: %s\n", agent.Metadata.Version)
	fmt.Fprintf(&sb, "- **Descricao**: %s\n", agent.Metadata.Description)

	runtimeParts := NewCatalog().buildRuntimeLine(agent.Runtime)
	fmt.Fprintf(&sb, "- **Runtime**: %s\n", runtimeParts)
	return sb.String()
}

// buildRuntimeLine constroi a linha de runtime no formato "ide / model / reasoning_effort / access_mode".
// Campos vazios sao omitidos.
func (c *Catalog) buildRuntimeLine(rt RuntimeDefaults) string {
	var parts []string
	if rt.IDE != "" {
		parts = append(parts, rt.IDE)
	}
	if rt.Model != "" {
		parts = append(parts, rt.Model)
	}
	if rt.ReasoningEffort != "" {
		parts = append(parts, rt.ReasoningEffort)
	}
	if rt.AccessMode != "" {
		parts = append(parts, rt.AccessMode)
	}
	if len(parts) == 0 {
		return "-"
	}
	return strings.Join(parts, " / ")
}

// buildCatalogBlock constroi o bloco de catalogo ordenado lexicograficamente por name.
// Aplica o limite de 200 entradas (RF-16) e marca o agente ativo com [active].
func (c *Catalog) buildCatalogBlock(active *ResolvedAgent, catalog []ResolvedAgent) string {
	// Copiar para nao mutar o slice original.
	entries := make([]ResolvedAgent, len(catalog))
	copy(entries, catalog)

	// Ordenar lexicograficamente por name.
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Name < entries[j].Name
	})

	// Aplicar limite de 200 entradas.
	if len(entries) > _catalogLimit {
		entries = entries[:_catalogLimit]
	}

	var sb strings.Builder
	sb.WriteString("### Agentes Disponiveis\n\n")
	for _, e := range entries {
		line := fmt.Sprintf("- `%s`", e.Name)
		if active != nil && e.Name == active.Name {
			line += " [active]"
		}
		if e.Metadata.Description != "" {
			line += " — " + e.Metadata.Description
		}
		sb.WriteString(line)
		sb.WriteByte('\n')
	}
	return sb.String()
}
