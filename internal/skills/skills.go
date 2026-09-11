package skills

import (
	"fmt"
	"strings"

	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/specs"
)

// Tool representa uma ferramenta de IA suportada.
type Tool string

const (
	ToolClaude   Tool = "claude"
	ToolCodex    Tool = "codex"
	ToolCopilot  Tool = "copilot"
	ToolOpenCode Tool = "opencode"
)

var AllTools = NewCatalog().canonicalTools()

// RemovedAgentError e o erro tipado e explicativo emitido quando um agente
// descontinuado e invocado por nome (RF-03). Distinguivel do erro generico de
// valor invalido via errors.As.
type RemovedAgentError struct {
	Agent          string
	Supported      []Tool
	MigrationGuide string
}

func (e *RemovedAgentError) Error() string {
	names := make([]string, 0, len(e.Supported))
	for _, t := range e.Supported {
		names = append(names, string(t))
	}
	return fmt.Sprintf(
		"agente %q foi removido — conjunto suportado: {%s}; guia de migracao: %s",
		e.Agent, strings.Join(names, ", "), e.MigrationGuide,
	)
}

// removedAgentGuides mapeia agentes descontinuados ao guia de migracao correspondente.
var removedAgentGuides = map[string]string{
	"gemini": "docs/migracao-legacy-acp.md#gemini-removido",
}

func (catalog *Catalog) canonicalTools() []Tool {
	ids := specs.NewCatalog().CanonicalOrder()
	out := make([]Tool, 0, len(ids))
	for _, id := range ids {
		out = append(out, Tool(id))
	}
	return out
}

func (catalog *Catalog) ParseTool(s string) (Tool, bool) {
	for _, t := range NewCatalog().canonicalTools() {
		if string(t) == s {
			return t, true
		}
	}
	return "", false
}

// ResolveTool resolve um nome de ferramenta em Tool, distinguindo agente removido
// (RemovedAgentError, RF-03) de valor generico invalido.
func (catalog *Catalog) ResolveTool(s string) (Tool, error) {
	if t, ok := catalog.ParseTool(s); ok {
		return t, nil
	}
	if guide, isRemoved := removedAgentGuides[s]; isRemoved {
		return "", &RemovedAgentError{Agent: s, Supported: AllTools, MigrationGuide: guide}
	}
	names := make([]string, 0, len(AllTools))
	for _, t := range AllTools {
		names = append(names, string(t))
	}
	return "", fmt.Errorf("ferramenta invalida: %q — opcoes: %s", s, strings.Join(names, ", "))
}

// Lang representa uma linguagem suportada.
type Lang string

const (
	LangGo     Lang = "go"
	LangNode   Lang = "node"
	LangPython Lang = "python"
	LangDotNet Lang = "dotnet"
)

var AllLangs = []Lang{LangGo, LangNode, LangPython, LangDotNet}

func (catalog *Catalog) ParseLang(s string) (Lang, bool) {
	switch Lang(s) {
	case LangGo, LangNode, LangPython, LangDotNet:
		return Lang(s), true
	}
	return "", false
}

// LinkMode define como skills canonicas sao instaladas no projeto alvo.
type LinkMode string

const (
	LinkSymlink LinkMode = "symlink"
	LinkCopy    LinkMode = "copy"
)

func (catalog *Catalog) ParseLinkMode(s string) (LinkMode, bool) {
	switch LinkMode(s) {
	case LinkSymlink, LinkCopy:
		return LinkMode(s), true
	}
	return "", false
}

// BaseSkills sao skills processuais instaladas para qualquer combinacao de ferramentas.
var BaseSkills = []string{
	"create-prd",
	"create-technical-specification",
	"create-tasks",
	"execute-task",
	"execute-all-tasks",
	"refactor",
	"review",
	"analyze-project",
	"agent-governance",
	"bugfix",
}

// ComplementarySkills sao skills de integracao embarcadas no binario e instaladas junto com as base.
var ComplementarySkills = []string{
	"confluence-changelog-publisher",
	"github-diff-changelog-publisher",
	"github-pr-comment-triage",
	"github-release-publication-flow",
	"jira-tasks",
	"otel-grafana-dashboards",
	"postman-collection-generator",
	"prompt-enricher",
	"pull-request",
	"semantic-commit",
	"us-to-prd",
}

// LangSkills retorna as skills de implementacao para as linguagens selecionadas.
func (catalog *Catalog) LangSkills(langs []Lang) []string {
	var out []string
	for _, l := range langs {
		switch l {
		case LangGo:
			out = append(out, "go-implementation", "object-calisthenics-go")
		case LangNode:
			out = append(out, "node-implementation")
		case LangPython:
			out = append(out, "python-implementation")
		case LangDotNet:
			out = append(out, "dotnet-csharp-implementation")
		}
	}
	return out
}

// AllSkills retorna a lista completa de skills a instalar.
func (catalog *Catalog) AllSkills(langs []Lang) []string {
	base := append(BaseSkills, ComplementarySkills...)
	return append(base, NewCatalog().LangSkills(langs)...)
}
