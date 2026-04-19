package wrapper

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/JailtonJunior94/ai-spec-harness/internal/fs"
	"github.com/JailtonJunior94/ai-spec-harness/internal/metrics"
	"github.com/JailtonJunior94/ai-spec-harness/internal/prerequisites"
)

// ValidTools é o conjunto de ferramentas aceitas pelo wrapper.
var ValidTools = map[string]bool{
	"codex":   true,
	"gemini":  true,
	"copilot": true,
}

// Execute valida as condições de governança e, se todas passarem, retorna a instrução
// de invocação formatada para a ferramenta solicitada.
//
// Verificações realizadas (em ordem):
//  1. AGENTS.md existe em projectDir
//  2. .agents/skills/agent-governance/ existe em projectDir
//  3. Pré-requisitos da skill estão satisfeitos (prerequisites.Verify)
//  4. Budget de tokens está dentro do limite (metrics.CheckBudget)
func Execute(tool, skill, projectDir string, args []string, fsys fs.FileSystem) (instruction string, err error) {
	if !ValidTools[tool] {
		return "", fmt.Errorf("ferramenta invalida: %q — tools aceitos: codex, gemini, copilot", tool)
	}

	absDir, err := filepath.Abs(projectDir)
	if err != nil {
		return "", fmt.Errorf("caminho invalido %q: %w", projectDir, err)
	}

	// Verificação 1: AGENTS.md existe
	agentsMD := filepath.Join(absDir, "AGENTS.md")
	if !fsys.Exists(agentsMD) {
		return "", fmt.Errorf("verificacao falhou: AGENTS.md nao encontrado em %s", absDir)
	}

	// Verificação 2: .agents/skills/agent-governance/ existe
	govSkillDir := filepath.Join(absDir, ".agents", "skills", "agent-governance")
	if !fsys.IsDir(govSkillDir) {
		return "", fmt.Errorf("verificacao falhou: diretorio .agents/skills/agent-governance nao encontrado em %s", absDir)
	}

	// Verificação 3: pré-requisitos da skill
	passed, results, err := prerequisites.Verify(skill, projectDir, fsys)
	if err != nil {
		return "", fmt.Errorf("verificacao de pre-requisitos falhou: %w", err)
	}
	if !passed {
		var missing []string
		for _, r := range results {
			if !r.Found && !r.Optional {
				missing = append(missing, r.Label)
			}
		}
		return "", fmt.Errorf("pre-requisitos ausentes para skill %q: %s", skill, strings.Join(missing, ", "))
	}

	// Verificação 4: budget de tokens — estima lendo o AGENTS.md
	content := ""
	if data, readErr := fsys.ReadFile(agentsMD); readErr == nil {
		content = string(data)
	}
	tokens, limit, ok := metrics.CheckBudget(content, tool)
	if !ok {
		return "", fmt.Errorf("budget excedido para %q: %d tokens estimados, limite %d", tool, tokens, limit)
	}

	instruction = buildInstruction(tool, skill, projectDir, args)
	return instruction, nil
}

func buildInstruction(tool, skill, projectDir string, args []string) string {
	extraArgs := ""
	if len(args) > 0 {
		extraArgs = " " + strings.Join(args, " ")
	}

	switch tool {
	case "codex":
		return fmt.Sprintf(
			"Invoke Codex with skill %q in project %s:\n  codex --skill %s --project %s%s",
			skill, projectDir, skill, projectDir, extraArgs,
		)
	case "gemini":
		return fmt.Sprintf(
			"Invoke Gemini with skill %q in project %s:\n  gemini run --skill %s --project %s%s",
			skill, projectDir, skill, projectDir, extraArgs,
		)
	case "copilot":
		return fmt.Sprintf(
			"Invoke Copilot with skill %q in project %s:\n  @copilot skill=%s project=%s%s",
			skill, projectDir, skill, projectDir, extraArgs,
		)
	default:
		return fmt.Sprintf("Run skill %q with tool %q in project %s%s", skill, tool, projectDir, extraArgs)
	}
}
