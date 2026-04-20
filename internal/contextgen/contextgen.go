package contextgen

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/JailtonJunior94/ai-spec-harness/internal/detect"
	"github.com/JailtonJunior94/ai-spec-harness/internal/fs"
	"github.com/JailtonJunior94/ai-spec-harness/internal/output"
	"github.com/JailtonJunior94/ai-spec-harness/internal/skills"
)

// GovernanceSchemaVersion é a versão de schema de governança embutida nos arquivos gerados.
const GovernanceSchemaVersion = "1.0.0"

// Generator orquestra a geracao de governanca contextual.
type Generator struct {
	fs      fs.FileSystem
	printer *output.Printer
}

func NewGenerator(fsys fs.FileSystem, printer *output.Printer) *Generator {
	return &Generator{fs: fsys, printer: printer}
}

// Generate cria governanca contextual para o projeto alvo.
// focusPaths e opcional: quando fornecido, o ToolchainDetector prioriza manifests
// mais proximos dos arquivos listados (util em monorepos).
func (g *Generator) Generate(sourceDir, projectDir string, tools []skills.Tool, langs []skills.Lang, codexProfile string, dryRun bool, focusPaths ...string) error {
	toolSet := make(map[skills.Tool]bool)
	for _, t := range tools {
		toolSet[t] = true
	}

	// Auto-detect compact profile: apenas Codex presente sem outras ferramentas
	governanceProfile := "standard"
	if len(tools) == 1 && toolSet[skills.ToolCodex] {
		governanceProfile = "compact"
	}

	if dryRun {
		g.printer.DryRun("Geraria AGENTS.md (schema %s, profile %s)", GovernanceSchemaVersion, governanceProfile)
		if toolSet[skills.ToolClaude] {
			g.printer.DryRun("Geraria CLAUDE.md")
		}
		if toolSet[skills.ToolGemini] {
			g.printer.DryRun("Geraria GEMINI.md")
		}
		if toolSet[skills.ToolCopilot] {
			g.printer.DryRun("Geraria .github/copilot-instructions.md")
		}
		if toolSet[skills.ToolCodex] {
			g.printer.DryRun("Geraria .codex/config.toml")
		}
		return nil
	}

	archDetector := detect.NewArchitectureDetector(g.fs)
	archResult := archDetector.Detect(projectDir)

	fwDetector := detect.NewFrameworkDetector(g.fs)
	frameworks := fwDetector.Detect(projectDir)
	frameworksStr := detect.JoinFrameworks(frameworks)

	stacks := detect.DetectPrimaryStack(g.fs, projectDir)
	stackStr := strings.Join(stacks, ",")

	tcDetector := detect.NewToolchainDetector(g.fs)
	if len(focusPaths) > 0 {
		tcDetector.FocusPaths = focusPaths
	}
	toolchain := tcDetector.Detect(projectDir)

	dirTree := g.buildDirectoryTree(projectDir)
	archDescription := detect.DescribeArchitecture(archResult.Type, stackStr, frameworksStr)
	depFlow := g.buildDependencyFlow(projectDir)
	archRules := detect.ArchitectureRules(archResult.Type)
	langRules := g.buildLanguageRules(projectDir)
	validationCmds := g.buildValidationCommands(toolchain)
	archRestrictions := detect.ArchitectureRestrictions(archResult.Type)

	// Gerar AGENTS.md
	agentsContent := g.renderAgentsTemplate(
		archResult.Type, archDescription, dirTree,
		archResult.Pattern, depFlow, archRules,
		langRules, validationCmds, archRestrictions,
	)

	if governanceProfile == "compact" {
		agentsContent = stripCompactSections(agentsContent)
	}

	if err := g.fs.WriteFile(filepath.Join(projectDir, "AGENTS.md"), []byte(agentsContent)); err != nil {
		return fmt.Errorf("escrever AGENTS.md: %w", err)
	}

	// Append AGENTS.local.md se presente
	localPath := filepath.Join(projectDir, "AGENTS.local.md")
	if g.fs.Exists(localPath) {
		localData, err := g.fs.ReadFile(localPath)
		if err == nil {
			agentsData, _ := g.fs.ReadFile(filepath.Join(projectDir, "AGENTS.md"))
			combined := string(agentsData) + "\n" + string(localData)
			_ = g.fs.WriteFile(filepath.Join(projectDir, "AGENTS.md"), []byte(combined))
		}
	}

	// Stack section para AI tool templates
	stackSection := g.buildStackSection(projectDir)

	// CLAUDE.md
	if toolSet[skills.ToolClaude] {
		content := g.renderAIToolTemplate(
			"Claude Code",
			"fonte canonica das regras",
			"`.claude/skills/` sao symlinks para `.agents/skills/` — a fonte de verdade e sempre `.agents/skills/`.",
			"`.claude/agents/` sao wrappers leves que delegam para a habilidade canonica.",
			stackSection,
		)
		if err := g.fs.WriteFile(filepath.Join(projectDir, "CLAUDE.md"), []byte(content)); err != nil {
			return fmt.Errorf("escrever CLAUDE.md: %w", err)
		}
	}

	// GEMINI.md
	if toolSet[skills.ToolGemini] {
		content := g.renderAIToolTemplate(
			"Gemini CLI",
			"fonte canonica das regras",
			"`.agents/skills/` e a fonte de verdade dos fluxos procedurais.",
			"`.gemini/commands/` sao adaptadores finos que apontam para a habilidade correta.",
			stackSection,
		)
		content += geminiExtraGuidance
		if err := g.fs.WriteFile(filepath.Join(projectDir, "GEMINI.md"), []byte(content)); err != nil {
			return fmt.Errorf("escrever GEMINI.md: %w", err)
		}
	}

	// Copilot instructions
	if toolSet[skills.ToolCopilot] {
		content := g.renderAIToolTemplate(
			"GitHub Copilot CLI",
			"instrucao principal",
			"`.agents/skills/` e a fonte de verdade dos fluxos procedurais.",
			"`.github/agents/` sao wrappers leves que apontam para a habilidade correta.",
			stackSection,
		)
		content += copilotExtraGuidance
		copilotPath := filepath.Join(projectDir, ".github", "copilot-instructions.md")
		_ = g.fs.MkdirAll(filepath.Dir(copilotPath))
		if err := g.fs.WriteFile(copilotPath, []byte(content)); err != nil {
			return fmt.Errorf("escrever copilot-instructions.md: %w", err)
		}
	}

	// Codex config
	if toolSet[skills.ToolCodex] {
		codexDir := filepath.Join(projectDir, ".codex")
		_ = g.fs.MkdirAll(codexDir)
		content := g.buildCodexConfig(projectDir, codexProfile)
		if err := g.fs.WriteFile(filepath.Join(codexDir, "config.toml"), []byte(content)); err != nil {
			return fmt.Errorf("escrever config.toml: %w", err)
		}
	}

	g.printer.Debug("Governanca contextual gerada com sucesso")
	g.printer.Debug("Arquitetura detectada: %s", archResult.Type)
	g.printer.Debug("Stack detectada: %s", stackStr)
	g.printer.Debug("Frameworks detectados: %s", frameworksStr)

	return nil
}

func (g *Generator) buildDirectoryTree(projectDir string) string {
	ignoreDirs := map[string]bool{
		".git": true, ".agents": true, ".claude": true, ".codex": true,
		".gemini": true, "node_modules": true, "vendor": true, "dist": true,
		"build": true, "bin": true, "target": true, "__pycache__": true,
	}
	ignoreFiles := map[string]bool{".gitkeep": true}

	var lines []string
	lines = append(lines, ".")

	g.walkTree(projectDir, projectDir, ignoreDirs, ignoreFiles, &lines, 80)

	return strings.Join(lines, "\n")
}

func (g *Generator) walkTree(baseDir, dir string, ignoreDirs, ignoreFiles map[string]bool, lines *[]string, maxLines int) {
	if len(*lines) >= maxLines {
		return
	}

	entries, err := g.fs.ReadDir(dir)
	if err != nil {
		return
	}

	for _, e := range entries {
		if len(*lines) >= maxLines {
			return
		}

		name := e.Name()
		if e.IsDir() && ignoreDirs[name] {
			continue
		}
		if !e.IsDir() && ignoreFiles[name] {
			continue
		}

		rel, _ := filepath.Rel(baseDir, filepath.Join(dir, name))
		*lines = append(*lines, rel)

		if e.IsDir() {
			g.walkTree(baseDir, filepath.Join(dir, name), ignoreDirs, ignoreFiles, lines, maxLines)
		}
	}
}

func (g *Generator) buildDependencyFlow(projectDir string) string {
	hasGo := g.fs.Exists(filepath.Join(projectDir, "go.mod")) || g.fs.Exists(filepath.Join(projectDir, "go.work"))
	hasNode := g.fs.Exists(filepath.Join(projectDir, "package.json")) || g.fs.Exists(filepath.Join(projectDir, "tsconfig.json"))
	hasPython := g.fs.Exists(filepath.Join(projectDir, "pyproject.toml")) || g.fs.Exists(filepath.Join(projectDir, "requirements.txt"))

	activeCount := 0
	if hasGo {
		activeCount++
	}
	if hasNode {
		activeCount++
	}
	if hasPython {
		activeCount++
	}

	if activeCount > 1 {
		return `- Cada stack deve expor contratos por fronteiras estaveis (HTTP/gRPC/eventos/arquivos), sem assumir detalhes internos de runtime de outra linguagem.
- Mudancas em contratos compartilhados devem atualizar produtores e consumidores da stack afetada e validar cada runtime com seu proprio toolchain.
- Compartilhar schemas, payloads e semantica operacional e aceitavel; compartilhar convencoes de framework, helpers de runtime ou acoplamento de deploy entre linguagens nao e.`
	}

	if hasGo {
		return `- Transporte e adapters devem depender de casos de uso ou servicos explicitos, nao do contrario.
- Dominio nao deve conhecer detalhes de HTTP, banco, filas, serializacao ou drivers.
- Infraestrutura pode implementar contratos consumidos pela aplicacao, preservando dependencia para dentro.`
	}

	if hasNode {
		return `- Controllers e routers devem depender de services ou use cases, nao do contrario.
- Dominio nao deve importar detalhes de framework (Express, Fastify, NestJS), ORM ou drivers.
- Infraestrutura implementa interfaces consumidas pela camada de aplicacao, preservando dependencia para dentro.`
	}

	if hasPython {
		return `- Routers e handlers devem depender de services ou use cases, nao do contrario.
- Dominio nao deve importar detalhes de framework (FastAPI, Django, Flask), ORM ou drivers.
- Infraestrutura implementa contratos consumidos pela camada de aplicacao, preservando dependencia para dentro.`
	}

	return `- Dependencias devem apontar de bordas externas para o nucleo do negocio.
- Detalhes de framework, IO e persistencia nao devem vazar para o centro do sistema.`
}

func (g *Generator) buildLanguageRules(projectDir string) string {
	var parts []string
	hasGo := g.fs.Exists(filepath.Join(projectDir, "go.mod")) || g.fs.Exists(filepath.Join(projectDir, "go.work"))
	hasNode := g.fs.Exists(filepath.Join(projectDir, "package.json")) || g.fs.Exists(filepath.Join(projectDir, "tsconfig.json"))
	hasPython := g.fs.Exists(filepath.Join(projectDir, "pyproject.toml")) || g.fs.Exists(filepath.Join(projectDir, "requirements.txt"))

	if hasGo {
		parts = append(parts, `Para tarefas que alteram codigo Go, carregar tambem:

- `+"`"+`.agents/skills/go-implementation/SKILL.md`+"`"+`

Para tarefas de revisao ou refatoracao incremental de design em Go guiadas por heuristicas de object calisthenics, carregar tambem:

- `+"`"+`.agents/skills/object-calisthenics-go/SKILL.md`+"`")
	}

	if hasNode {
		parts = append(parts, `Para tarefas que alteram codigo Node/TypeScript, carregar tambem:

- `+"`"+`.agents/skills/node-implementation/SKILL.md`+"`")
	}

	if hasPython {
		parts = append(parts, `Para tarefas que alteram codigo Python, carregar tambem:

- `+"`"+`.agents/skills/python-implementation/SKILL.md`+"`")
	}

	return strings.Join(parts, "\n\n")
}

func (g *Generator) buildValidationCommands(toolchain detect.ToolchainResult) string {
	var lines []string
	lines = append(lines, "Seguir Etapa 4 de `.agents/skills/agent-governance/SKILL.md` como base canonica.")
	lines = append(lines, "")

	labels := map[string]string{"go": "Go", "node": "Node", "python": "Python"}
	for _, lang := range []string{"go", "node", "python"} {
		entry, ok := toolchain[lang]
		if !ok {
			continue
		}
		label := labels[lang]
		lines = append(lines, fmt.Sprintf("Comandos detectados no projeto (%s):", label))
		idx := 1
		if entry.Fmt != "" {
			lines = append(lines, fmt.Sprintf("%d. Rodar fmt: `%s`.", idx, entry.Fmt))
			idx++
		}
		if entry.Test != "" {
			lines = append(lines, fmt.Sprintf("%d. Rodar test: `%s`.", idx, entry.Test))
			idx++
		}
		if entry.Lint != "" {
			lines = append(lines, fmt.Sprintf("%d. Rodar lint: `%s`.", idx, entry.Lint))
		}
	}

	return strings.Join(lines, "\n")
}

func (g *Generator) buildStackSection(projectDir string) string {
	var lines []string

	if g.fs.Exists(filepath.Join(projectDir, "go.mod")) {
		lines = append(lines, "- Projeto com contexto Go detectado: carregar `.agents/skills/go-implementation/SKILL.md` ao alterar codigo Go.")
		lines = append(lines, "- Validar a versao declarada em `go.mod` antes de introduzir APIs da linguagem ou novas dependencias.")
	}
	if g.fs.Exists(filepath.Join(projectDir, "package.json")) || g.fs.Exists(filepath.Join(projectDir, "tsconfig.json")) {
		lines = append(lines, "- Projeto com contexto Node/TypeScript detectado: carregar `.agents/skills/node-implementation/SKILL.md` ao alterar codigo Node/TS.")
		lines = append(lines, "- Validar versao de Node em `engines` ou `.nvmrc` antes de usar APIs recentes.")
	}
	if g.fs.Exists(filepath.Join(projectDir, "pyproject.toml")) || g.fs.Exists(filepath.Join(projectDir, "requirements.txt")) {
		lines = append(lines, "- Projeto com contexto Python detectado: carregar `.agents/skills/python-implementation/SKILL.md` ao alterar codigo Python.")
		lines = append(lines, "- Validar versao de Python em `pyproject.toml` ou `.python-version` antes de usar APIs recentes.")
	}

	if len(lines) == 0 {
		return ""
	}
	return "## Stack\n\n" + strings.Join(lines, "\n") + "\n"
}

var planningSkills = []string{
	"analyze-project",
	"create-prd",
	"create-technical-specification",
	"create-tasks",
}

func (g *Generator) buildCodexConfig(projectDir, codexProfile string) string {
	baseSkills := []string{"agent-governance", "bugfix", "review", "refactor", "execute-task"}

	if codexProfile != "lean" {
		baseSkills = append(baseSkills, planningSkills...)
	}

	if g.fs.Exists(filepath.Join(projectDir, ".agents", "skills", "go-implementation", "SKILL.md")) {
		baseSkills = append(baseSkills, "go-implementation", "object-calisthenics-go")
	}
	if g.fs.Exists(filepath.Join(projectDir, ".agents", "skills", "node-implementation", "SKILL.md")) {
		baseSkills = append(baseSkills, "node-implementation")
	}
	if g.fs.Exists(filepath.Join(projectDir, ".agents", "skills", "python-implementation", "SKILL.md")) {
		baseSkills = append(baseSkills, "python-implementation")
	}

	var b strings.Builder
	for _, skill := range baseSkills {
		fmt.Fprintf(&b, "[[skills.config]]\npath = \".agents/skills/%s\"\nenabled = true\n\n", skill)
	}
	return b.String()
}

func (g *Generator) renderAgentsTemplate(
	archType detect.ArchitectureType, archDescription, dirTree,
	archPattern, depFlow, archRules, langRules, validationCmds, archRestrictions string,
) string {
	return fmt.Sprintf(`<!-- governance-schema: %s -->
# Regras para Agentes de IA

Este diretorio centraliza regras para uso com agentes de IA em tarefas reais de analise, alteracao e validacao de codigo.

## Objetivo

Use estas instrucoes para manter consistencia, seguranca e qualidade ao trabalhar com codigo, configuracao, validacao e evolucao de sistemas.

## Arquitetura: %s

%s

## Estrutura de Pastas

%s`+"```"+`

## Padrao Arquitetural

%s

### Fluxo de Dependencias

%s

## Modo de trabalho

1. Entender o contexto antes de editar qualquer arquivo.
2. Preferir a menor mudanca segura que resolva a causa raiz.
3. Preservar arquitetura, convencoes e fronteiras ja existentes no contexto analisado.
4. Nao introduzir abstracoes, camadas ou dependencias sem demanda concreta.
5. Atualizar ou adicionar testes quando houver mudanca de comportamento.
6. Rodar validacoes proporcionais a mudanca.
7. Registrar bloqueios e suposicoes explicitamente quando o contexto estiver incompleto.

## Diretrizes de Estrutura

1. Priorize entendimento do codigo e do contexto atual antes de propor refatoracoes.
2. Respeite padroes existentes de nomenclatura, organizacao e tratamento de erro.
3. Defina estrutura simples, evolutiva e com defaults explicitos.
4. Evite reescritas amplas quando uma alteracao localizada resolver o problema.
5. Estabeleca contratos, testes e comandos de validacao cedo quando eles ainda nao existirem.
6. Considere risco de regressao como restricao principal.
7. Evite overengineering disfarcado de arquitetura futura.

%s

## Regras por Linguagem

Para tarefas que alteram codigo, carregar a skill:

- `+"`"+`.agents/skills/agent-governance/SKILL.md`+"`"+`

%s

Para tarefas de correcao de bugs com remediacao e teste de regressao, carregar tambem:

- `+"`"+`.agents/skills/bugfix/SKILL.md`+"`"+`

### Composicao Multi-Linguagem

Em projetos com mais de uma linguagem (ex: monorepo Go + Node), carregar apenas a skill da linguagem afetada pela mudanca. Se a tarefa cruzar linguagens, carregar ambas e aplicar a validacao de cada stack nos arquivos correspondentes. Nao misturar convencoes de uma linguagem em arquivos de outra.

## Referencias

Cada skill lista suas proprias referencias em `+"`"+`references/`+"`"+` com gatilhos de carregamento no respectivo `+"`"+`SKILL.md`+"`"+`. Nao duplicar a listagem aqui — consultar o SKILL.md da skill ativa para saber quais referencias carregar e em que condicao.

## Notas por Ferramenta

- **Claude Code**: skills pre-carregadas via `+"`"+`.claude/skills/`+"`"+`, hooks via `+"`"+`.claude/hooks/`+"`"+`, agents delegados via `+"`"+`.claude/agents/`+"`"+`.
- **Gemini CLI**: commands em `+"`"+`.gemini/commands/*.toml`+"`"+` apontam para skills canonicas. Sem hooks ou agents nativos — o modelo deve seguir as instrucoes procedurais do SKILL.md carregado.
- **Codex**: le `+"`"+`AGENTS.md`+"`"+` como instrucao de sessao. Entradas em `+"`"+`.codex/config.toml`+"`"+` sao metadados para `+"`"+`upgrade.sh`+"`"+`, nao spec oficial do Codex CLI. O agente deve seguir as instrucoes de `+"`"+`AGENTS.md`+"`"+` para descobrir e carregar skills.
- **Copilot**: `+"`"+`.github/copilot-instructions.md`+"`"+` como instrucao principal. `+"`"+`.github/agents/`+"`"+` sao wrappers. Sem hooks nativos — compliance depende do modelo seguir as instrucoes.

### Matrix de Enforcement

| Capacidade | Claude Code | Gemini CLI | Codex | Copilot |
|---|---|---|---|---|
| Carga base automatica | hook PreToolUse | procedural | procedural | procedural |
| Protecao de governanca | hook PostToolUse | procedural | procedural | procedural |
| Skills pre-carregadas | sim (symlinks) | sim (commands) | nao | sim (agents) |
| Enforcement programatico | sim (hooks) | nao | nao | nao |
| Validacao de evidencias | script | procedural | procedural | procedural |

Ferramentas sem enforcement programatico dependem do modelo seguir instrucoes procedurais. A compliance nessas ferramentas e best-effort.

## Economia de Contexto

Carregar o minimo necessario para a tarefa reduz custo de tokens em 35-50%%:

| Complexidade | Criterio | O que carregar |
|---|---|---|
| `+"`"+`trivial`+"`"+` | Rename, typo, import, formatacao | Apenas AGENTS.md |
| `+"`"+`standard`+"`"+` | Bug fix, novo metodo, refactor local | AGENTS.md + TL;DR das references afetadas |
| `+"`"+`complex`+"`"+` | Nova feature, interface publica, migracao | AGENTS.md + referencias completas |

- Classificar a complexidade **antes** de carregar qualquer referencia.
- Quando a reference tiver bloco `+"`"+`<!-- TL;DR ... -->`+"`"+`, preferir o TL;DR ao documento completo em tarefas standard.
- Override explicito via `+"`"+`--complexity=<nivel>`+"`"+` prevalece sobre classificacao automatica.

## Validacao

Antes de concluir uma alteracao:

%s

## Restricoes

1. Nao inventar contexto ausente.
2. Nao assumir versao de linguagem, framework ou runtime sem verificar.
3. Nao alterar comportamento publico sem deixar isso explicito.
4. Nao usar exemplos como copia cega; adaptar ao contexto real.
%s

### Controle de profundidade de invocacao

- Skills que invocam outros skills (execute-task, refactor) devem verificar profundidade via `+"`"+`scripts/lib/check-invocation-depth.sh`+"`"+`.
- Limite padrao: 2 niveis. Configuravel via `+"`"+`AI_INVOCATION_MAX`+"`"+`.
- Variaveis de ambiente: `+"`"+`AI_INVOCATION_DEPTH`+"`"+` (corrente), `+"`"+`AI_INVOCATION_MAX`+"`"+` (limite).
`,
		GovernanceSchemaVersion,
		string(archType),
		archDescription,
		"```\n"+dirTree+"\n",
		archPattern,
		depFlow,
		archRules,
		langRules,
		validationCmds,
		archRestrictions,
	)
}

func (g *Generator) renderAIToolTemplate(
	toolName, toolInstruction, configLine2, configLine3, stackSection string,
) string {
	content := fmt.Sprintf(`# %s

Use `+"`"+`AGENTS.md`+"`"+` como %s deste repositorio.

## Instrucoes

1. Ler `+"`"+`AGENTS.md`+"`"+` no inicio da sessao.
2. %s
3. %s
4. Em tarefas de execucao, carregar apenas `+"`"+`AGENTS.md`+"`"+`, `+"`"+`agent-governance`+"`"+` e a skill operacional da linguagem ou atividade afetada.
5. Skills de planejamento (`+"`"+`analyze-project`+"`"+`, `+"`"+`create-prd`+"`"+`, `+"`"+`create-technical-specification`+"`"+`, `+"`"+`create-tasks`+"`"+`) entram apenas quando a tarefa pedir esse fluxo explicitamente.
6. Carregar referencias adicionais apenas quando a tarefa exigir.
7. Preservar estilo, arquitetura e fronteiras existentes antes de propor mudancas.
8. Validar mudancas com comandos proporcionais ao risco.

%s`, toolName, toolInstruction, configLine2, configLine3, stackSection)

	return content
}

const geminiExtraGuidance = `

## Orientacoes Especificas para Gemini

O Gemini CLI nao suporta hooks, agents ou rules nativos. Para modelar o fluxo de governanca:

1. Ao iniciar uma tarefa, ler ` + "`" + `AGENTS.md` + "`" + ` e ` + "`" + `.agents/skills/agent-governance/SKILL.md` + "`" + ` como contexto base antes de editar codigo.
2. Usar ` + "`" + `@<command>` + "`" + ` para invocar o comando TOML correspondente a skill desejada.
3. Seguir as etapas procedurais do SKILL.md carregado pelo comando como se fossem instrucoes sequenciais.
4. Ao final da tarefa, executar os comandos de validacao descritos na secao Validacao do ` + "`" + `AGENTS.md` + "`" + `.
5. Nao confiar em enforcement automatico — a compliance depende de seguir as instrucoes procedurais manualmente.
`

const copilotExtraGuidance = `

## Orientacoes Especificas para Copilot

O GitHub Copilot suporta agents em ` + "`" + `.github/agents/` + "`" + ` e carrega ` + "`" + `copilot-instructions.md` + "`" + ` automaticamente, mas nao suporta hooks de enforcement. Para manter compliance:

1. Usar agents disponiveis em ` + "`" + `.github/agents/` + "`" + ` para delegar tarefas processuais (review, bugfix, execute-task, etc.).
2. Cada agent aponta para a skill canonica em ` + "`" + `.agents/skills/` + "`" + ` — seguir as etapas procedurais do SKILL.md referenciado.
3. Ao iniciar uma tarefa, confirmar que ` + "`" + `AGENTS.md` + "`" + ` e ` + "`" + `agent-governance/SKILL.md` + "`" + ` foram lidos.
4. Ao final da tarefa, executar os comandos de validacao descritos na secao Validacao acima.
5. Enforcement depende do modelo seguir as instrucoes — nao ha bloqueio automatico.
`

// stripCompactSections remove secoes verbose do AGENTS.md para profile compact.
// Remove os blocos "## Diretrizes de Estrutura" e "### Composicao Multi-Linguagem"
// ate o inicio do proximo cabecalho de mesmo ou maior nivel.
func stripCompactSections(content string) string {
	content = stripSection(content, "## Diretrizes de Estrutura", "## ")
	content = stripSection(content, "### Composicao Multi-Linguagem", "## ")
	return content
}

// stripSection remove o bloco que começa em startHeader ate (exclusive) a proxima
// linha que comece com nextPrefix (mesmo nivel ou superior).
func stripSection(content, startHeader, nextPrefix string) string {
	startIdx := strings.Index(content, "\n"+startHeader)
	if startIdx == -1 {
		// tenta no inicio do arquivo
		if strings.HasPrefix(content, startHeader) {
			startIdx = -1
		} else {
			return content
		}
	}

	var blockStart int
	if startIdx == -1 {
		blockStart = 0
	} else {
		blockStart = startIdx + 1 // pula o '\n' inicial
	}

	// encontrar proximo cabecalho do mesmo nivel apos blockStart
	rest := content[blockStart+len(startHeader):]
	endOffset := strings.Index(rest, "\n"+nextPrefix)
	if endOffset == -1 {
		// nao ha proximo cabecalho — remover ate o final
		return content[:blockStart]
	}

	endIdx := blockStart + len(startHeader) + endOffset + 1 // +1 para incluir o '\n' antes do proximo header
	return content[:blockStart] + content[endIdx:]
}
