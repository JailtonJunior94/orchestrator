package skills

import "github.com/JailtonJunior94/ai-spec-harness/internal/runtime/specs"

// Tool representa uma ferramenta de IA suportada.
type Tool string

const (
	ToolClaude  Tool = "claude"
	ToolGemini  Tool = "gemini"
	ToolCodex   Tool = "codex"
	ToolCopilot Tool = "copilot"
)

var AllTools = NewCatalog().canonicalTools()

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
