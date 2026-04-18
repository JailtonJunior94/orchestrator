package skills

// Tool representa uma ferramenta de IA suportada.
type Tool string

const (
	ToolClaude  Tool = "claude"
	ToolGemini  Tool = "gemini"
	ToolCodex   Tool = "codex"
	ToolCopilot Tool = "copilot"
)

var AllTools = []Tool{ToolClaude, ToolGemini, ToolCodex, ToolCopilot}

func ParseTool(s string) (Tool, bool) {
	switch Tool(s) {
	case ToolClaude, ToolGemini, ToolCodex, ToolCopilot:
		return Tool(s), true
	}
	return "", false
}

// Lang representa uma linguagem suportada.
type Lang string

const (
	LangGo     Lang = "go"
	LangNode   Lang = "node"
	LangPython Lang = "python"
)

var AllLangs = []Lang{LangGo, LangNode, LangPython}

func ParseLang(s string) (Lang, bool) {
	switch Lang(s) {
	case LangGo, LangNode, LangPython:
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

func ParseLinkMode(s string) (LinkMode, bool) {
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
	"refactor",
	"review",
	"analyze-project",
	"agent-governance",
	"bugfix",
}

// LangSkills retorna as skills de implementacao para as linguagens selecionadas.
func LangSkills(langs []Lang) []string {
	var out []string
	for _, l := range langs {
		switch l {
		case LangGo:
			out = append(out, "go-implementation", "object-calisthenics-go")
		case LangNode:
			out = append(out, "node-implementation")
		case LangPython:
			out = append(out, "python-implementation")
		}
	}
	return out
}

// AllSkills retorna a lista completa de skills a instalar.
func AllSkills(langs []Lang) []string {
	return append(BaseSkills, LangSkills(langs)...)
}
