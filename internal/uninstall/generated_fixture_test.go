package uninstall

import "fmt"

const generatedAgentsMarkdown = `<!-- governance-schema: 1.3.0 -->
# Regras para Agentes de IA

Este diretorio centraliza regras para uso com agentes de IA em tarefas reais de analise, alteracao e validacao de codigo.
`

const generatedCodexConfig = `[[skills.config]]
path = ".agents/skills/agent-governance"
enabled = true

[[skills.config]]
path = ".agents/skills/bugfix"
enabled = true
`

func generatedToolMarkdown(toolName string) string {
	return fmt.Sprintf("# %s\n\nUse `AGENTS.md` como fonte canonica das regras deste repositorio.\n\n"+
		"## Instrucoes\n\n1. Ler `AGENTS.md` no inicio da sessao.\n2. `.agents/skills/` e a fonte de verdade.\n", toolName)
}
