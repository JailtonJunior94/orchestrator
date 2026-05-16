package adapters

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/JailtonJunior94/ai-spec-harness/internal/fs"
	"github.com/JailtonJunior94/ai-spec-harness/internal/output"
	"github.com/JailtonJunior94/ai-spec-harness/internal/skills"
)

// Generator cria arquivos adaptadores para cada ferramenta de IA.
type Generator struct {
	fs      fs.FileSystem
	printer *output.Printer
}

func NewGenerator(fsys fs.FileSystem, printer *output.Printer) *Generator {
	return &Generator{fs: fsys, printer: printer}
}

// ProcessualSkills sao skills que geram agents/comandos.
var ProcessualSkills = []string{
	"bugfix", "create-prd", "analyze-project", "refactor",
	"review", "execute-task", "create-tasks", "create-technical-specification",
}

// executeTaskYAMLContract eh o bloco YAML literal que TODO subagent task-executor
// DEVE retornar. Mantido aqui (em vez de embutido na instruction) para garantir
// paridade textual entre Claude/Codex/Gemini/Copilot — execute-all-tasks valida
// formato canonico em cadeia de 4 passos (status, report_path, summary).
const executeTaskYAMLContract = "status: done | blocked | failed | needs_input\nreport_path: tasks/prd-<slug>/<id>_execution_report.md\nsummary: <1 linha>"

type skillMeta struct {
	claudeName  string
	claudeFile  string
	githubName  string
	githubFile  string
	instruction string
}

var skillRegistry = map[string]skillMeta{
	"bugfix": {
		claudeName: "bugfixer", claudeFile: "bugfixer.md",
		githubName: "Corretor de Bugs", githubFile: "bugfix.agent.md",
		instruction: "corrija os bugs no escopo acordado, rode validacao proporcional e retorne o relatorio de correcao mais o estado final",
	},
	"create-prd": {
		claudeName: "prd-writer", claudeFile: "prd-writer.md",
		githubName: "Redator de PRD", githubFile: "prd-writer.agent.md",
		instruction: "colete o contexto minimo de produto, escreva ou atualize o PRD e retorne o caminho final ou um resumo conciso de needs_input",
	},
	"analyze-project": {
		claudeName: "project-analyzer", claudeFile: "project-analyzer.md",
		githubName: "Analisador de Projeto", githubFile: "project-analyzer.agent.md",
		instruction: "analise o projeto alvo, classifique a arquitetura, detecte a stack e ferramentas de IA, e gere os arquivos de governanca apropriados",
	},
	"refactor": {
		claudeName: "refactorer", claudeFile: "refactorer.md",
		githubName: "Refatorador", githubFile: "refactorer.agent.md",
		instruction: "fique dentro do escopo de refatoracao solicitado, preserve o comportamento observavel e retorne o caminho do relatorio mais o estado final",
	},
	"review": {
		claudeName: "reviewer", claudeFile: "reviewer.md",
		githubName: "Revisor", githubFile: "reviewer.agent.md",
		instruction: "revise o diff solicitado, lidere com achados e retorne um veredito canonico",
	},
	"execute-task": {
		claudeName: "task-executor", claudeFile: "task-executor.md",
		githubName: "Executor de Tarefa", githubFile: "task-executor.agent.md",
		instruction: "execute uma tarefa elegivel, rode validacao proporcional e retorne o caminho do relatorio de execucao mais o estado final",
	},
	"create-tasks": {
		claudeName: "task-planner", claudeFile: "task-planner.md",
		githubName: "Planejador de Tarefas", githubFile: "task-planner.agent.md",
		instruction: "produza o plano de alto nivel para aprovacao e so entao gere tasks.md e os arquivos por tarefa quando a aprovacao estiver disponivel",
	},
	"create-technical-specification": {
		claudeName: "technical-specification-writer", claudeFile: "technical-specification-writer.md",
		githubName: "Redator de Especificacao Tecnica", githubFile: "technical-specification-writer.agent.md",
		instruction: "explore os caminhos de codigo relevantes, resolva bloqueios de arquitetura, escreva a especificacao tecnica e as ADRs e retorne os caminhos criados ou um resumo conciso de needs_input",
	},
}

func (g *Generator) GenerateClaude(sourceDir, projectDir string) {
	agentsDir := filepath.Join(projectDir, ".claude", "agents")
	_ = g.fs.MkdirAll(agentsDir)
	count := 0

	for _, skill := range ProcessualSkills {
		meta, ok := skillRegistry[skill]
		if !ok {
			continue
		}

		skillFile := filepath.Join(sourceDir, ".agents", "skills", skill, "SKILL.md")
		if !g.fs.Exists(skillFile) {
			continue
		}

		data, err := g.fs.ReadFile(skillFile)
		if err != nil {
			continue
		}

		fm := skills.ParseFrontmatter(data)
		if fm.Description == "" {
			continue
		}

		shortDesc := truncateAtSentence(fm.Description, 120)
		content := fmt.Sprintf(`---
name: %s
description: %s
skills:
  - %s
---

Use a habilidade pre-carregada `+"`%s`"+` como processo canonico.
Mantenha este subagente estreito: %s.
`, meta.claudeName, shortDesc, skill, skill, meta.instruction)

		if skill == "execute-task" {
			content += "\nAo concluir, retorne EXCLUSIVAMENTE um bloco YAML (sem diffs, codigo ou logs):\n\n```yaml\n" + executeTaskYAMLContract + "\n```\n"
		}

		_ = g.fs.WriteFile(filepath.Join(agentsDir, meta.claudeFile), []byte(content))
		count++
	}
	g.printer.Debug("Adaptadores Claude gerados: %d", count)
}

func (g *Generator) GenerateGitHub(sourceDir, projectDir string) {
	agentsDir := filepath.Join(projectDir, ".github", "agents")
	_ = g.fs.MkdirAll(agentsDir)
	count := 0

	for _, skill := range ProcessualSkills {
		meta, ok := skillRegistry[skill]
		if !ok {
			continue
		}

		skillFile := filepath.Join(sourceDir, ".agents", "skills", skill, "SKILL.md")
		if !g.fs.Exists(skillFile) {
			continue
		}

		data, err := g.fs.ReadFile(skillFile)
		if err != nil {
			continue
		}

		fm := skills.ParseFrontmatter(data)
		if fm.Description == "" {
			continue
		}

		shortDesc := truncateAtSentence(fm.Description, 120)
		content := fmt.Sprintf(`---
name: %s
description: %s
---

Use a habilidade `+"`%s`"+` como processo canonico.
Mantenha este agente estreito: %s.
`, meta.githubName, shortDesc, skill, meta.instruction)

		if skill == "execute-task" {
			content += "\nAo concluir, retorne EXCLUSIVAMENTE um bloco YAML (sem diffs, codigo ou logs):\n\n```yaml\n" + executeTaskYAMLContract + "\n```\n"
		}

		_ = g.fs.WriteFile(filepath.Join(agentsDir, meta.githubFile), []byte(content))
		count++
	}
	g.printer.Debug("Adaptadores GitHub gerados: %d", count)
}

// GenerateGeminiAgents cria definicoes de subagent em .gemini/agents/<name>.md
// espelhando ProcessualSkills + skillRegistry. Mantem paridade com Claude/Copilot.
func (g *Generator) GenerateGeminiAgents(sourceDir, projectDir string) {
	agentsDir := filepath.Join(projectDir, ".gemini", "agents")
	_ = g.fs.MkdirAll(agentsDir)
	count := 0

	for _, skill := range ProcessualSkills {
		meta, ok := skillRegistry[skill]
		if !ok {
			continue
		}

		skillFile := filepath.Join(sourceDir, ".agents", "skills", skill, "SKILL.md")
		if !g.fs.Exists(skillFile) {
			continue
		}

		data, err := g.fs.ReadFile(skillFile)
		if err != nil {
			continue
		}

		fm := skills.ParseFrontmatter(data)
		if fm.Description == "" {
			continue
		}

		shortDesc := truncateAtSentence(fm.Description, 120)
		content := fmt.Sprintf(`---
name: %s
description: %s
---

Use a skill canonica `+"`.agents/skills/%s/SKILL.md`"+` como processo de execucao desta tarefa.
Mantenha este subagente estreito: %s.
`, meta.claudeName, shortDesc, skill, meta.instruction)

		if skill == "execute-task" {
			content += "\nAo concluir, retorne EXCLUSIVAMENTE um bloco YAML (sem diffs, codigo ou logs):\n\n```yaml\n" + executeTaskYAMLContract + "\n```\n"
		}

		_ = g.fs.WriteFile(filepath.Join(agentsDir, meta.claudeName+".md"), []byte(content))
		count++
	}
	g.printer.Debug("Adaptadores Gemini agents gerados: %d", count)
}

// GenerateCodexAgents cria definicoes de subagent em .codex/agents/<name>.toml
// espelhando ProcessualSkills + skillRegistry. Mantem paridade com Claude/Copilot.
func (g *Generator) GenerateCodexAgents(sourceDir, projectDir string) {
	agentsDir := filepath.Join(projectDir, ".codex", "agents")
	_ = g.fs.MkdirAll(agentsDir)
	count := 0

	for _, skill := range ProcessualSkills {
		meta, ok := skillRegistry[skill]
		if !ok {
			continue
		}

		skillFile := filepath.Join(sourceDir, ".agents", "skills", skill, "SKILL.md")
		if !g.fs.Exists(skillFile) {
			continue
		}

		data, err := g.fs.ReadFile(skillFile)
		if err != nil {
			continue
		}

		fm := skills.ParseFrontmatter(data)
		if fm.Description == "" {
			continue
		}

		shortDesc := truncateAtSentence(fm.Description, 120)
		instructions := fmt.Sprintf(`Use a skill canonica .agents/skills/%s/SKILL.md como processo de execucao desta tarefa.
Mantenha este subagente estreito: %s.`, skill, meta.instruction)

		if skill == "execute-task" {
			instructions += "\n\nAo concluir, retorne EXCLUSIVAMENTE um bloco YAML (sem diffs, codigo ou logs):\n\n" + executeTaskYAMLContract
		}

		content := fmt.Sprintf(`name = %q
description = %q

instructions = """
%s
"""

[[skills.config]]
path = ".agents/skills/%s"
enabled = true
`, meta.claudeName, shortDesc, instructions, skill)

		_ = g.fs.WriteFile(filepath.Join(agentsDir, meta.claudeName+".toml"), []byte(content))
		count++
	}
	g.printer.Debug("Adaptadores Codex agents gerados: %d", count)
}

// reviewLoopSkills are skills that include a validation loop and need an explicit reminder in the prompt.
var reviewLoopSkills = map[string]bool{
	"execute-task": true,
	"refactor":     true,
}

func (g *Generator) GenerateGemini(sourceDir, projectDir string) {
	cmdDir := filepath.Join(projectDir, ".gemini", "commands")
	_ = g.fs.MkdirAll(cmdDir)
	count := 0

	skillsDir := filepath.Join(sourceDir, ".agents", "skills")
	entries, err := g.fs.ReadDir(skillsDir)
	if err != nil {
		return
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		skillName := entry.Name()
		if skillName == "agent-governance" {
			continue
		}

		skillFile := filepath.Join(skillsDir, skillName, "SKILL.md")
		if !g.fs.Exists(skillFile) {
			continue
		}

		data, err := g.fs.ReadFile(skillFile)
		if err != nil {
			continue
		}

		fm := skills.ParseFrontmatter(data)
		if fm.Description == "" {
			continue
		}

		shortDesc := truncateAtSentence(fm.Description, 120)
		prompt := g.buildGeminiPrompt(skillsDir, skillName)
		content := fmt.Sprintf("description = %q\nprompt = \"\"\"\n%s\n\"\"\"\n", shortDesc, prompt)
		_ = g.fs.WriteFile(filepath.Join(cmdDir, skillName+".toml"), []byte(content))
		count++
	}
	g.printer.Debug("Gemini commands gerados: %d", count)
}

func (g *Generator) buildGeminiPrompt(skillsDir, skillName string) string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "Use `.agents/skills/%s/SKILL.md` como fluxo canonico desta tarefa.\n", skillName)

	for _, a := range g.collectSkillAssets(skillsDir, skillName) {
		fmt.Fprintf(&sb, "Carregue `%s` antes de iniciar.\n", a)
	}

	sb.WriteString("Leia os assets e references sob demanda conforme descrito no SKILL.md.\n")
	sb.WriteString("Nao invente um processo paralelo neste comando.")

	if reviewLoopSkills[skillName] {
		sb.WriteString("\n\nAo concluir, rode validacao proporcional e retorne o relatorio com estado final.")
	}

	sb.WriteString("\n\nAplicar a habilidade a esta solicitacao:\n{{args}}")
	return sb.String()
}

func (g *Generator) collectSkillAssets(skillsDir, skillName string) []string {
	assetsDir := filepath.Join(skillsDir, skillName, "assets")
	entries, err := g.fs.ReadDir(assetsDir)
	if err != nil {
		return nil
	}
	var files []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".md") {
			files = append(files, fmt.Sprintf(".agents/skills/%s/assets/%s", skillName, e.Name()))
		}
	}
	return files
}

// BuildCodexConfig gera o conteudo do config.toml para Codex.
func (g *Generator) BuildCodexConfig(skillList []string) string {
	var b strings.Builder
	for _, skill := range skillList {
		fmt.Fprintf(&b, "[[skills.config]]\npath = \".agents/skills/%s\"\nenabled = true\n\n", skill)
	}
	return b.String()
}

func truncateAtSentence(s string, maxLen int) string {
	if idx := strings.Index(s, ". "); idx >= 0 && idx < maxLen {
		return s[:idx+1]
	}
	if len(s) > maxLen {
		return s[:maxLen]
	}
	return s
}
