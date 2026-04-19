package aispecharness

import (
	"fmt"
	"os"

	"github.com/JailtonJunior94/ai-spec-harness/internal/fs"
	"github.com/JailtonJunior94/ai-spec-harness/internal/wrapper"
	"github.com/spf13/cobra"
)

var wrapperCmd = &cobra.Command{
	Use:   "wrapper <tool> <skill> [path] [args...]",
	Short: "Valida governanca e emite instrucao de invocacao para Codex/Gemini/Copilot",
	Long: `Verifica as condicoes de governanca antes de emitir instrucoes de invocacao.

Verificacoes realizadas (em ordem):
  1. AGENTS.md existe no projeto
  2. .agents/skills/agent-governance/ existe no projeto
  3. Pre-requisitos da skill estao satisfeitos
  4. Budget de tokens esta dentro do limite da ferramenta

Exit codes:
  0 — todas as verificacoes passaram, instrucao emitida na stdout
  1 — uma ou mais verificacoes falharam
  2 — uso incorreto (tool invalido ou argumentos insuficientes)

Tools aceitos: codex, gemini, copilot
  (Claude usa hooks de governanca proprios e nao precisa deste wrapper)

Exemplos:
  ai-spec wrapper codex go-implementation .
  ai-spec wrapper gemini create-tasks ./meu-projeto
  ai-spec wrapper copilot execute-task . --verbose`,
	Args: cobra.MinimumNArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		tool := args[0]
		skill := args[1]

		// Valida ferramenta antes de qualquer coisa
		if !wrapper.ValidTools[tool] {
			fmt.Fprintf(os.Stderr, "Erro: ferramenta invalida %q — tools aceitos: codex, gemini, copilot\n", tool)
			os.Exit(2)
		}

		projectDir := "."
		extraArgs := []string{}
		if len(args) >= 3 {
			projectDir = args[2]
			extraArgs = args[3:]
		}

		fsys := fs.NewOSFileSystem()
		instruction, err := wrapper.Execute(tool, skill, projectDir, extraArgs, fsys)
		if err != nil {
			fmt.Fprintln(os.Stderr, "Erro:", err)
			os.Exit(1)
		}

		fmt.Println(instruction)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(wrapperCmd)
}
