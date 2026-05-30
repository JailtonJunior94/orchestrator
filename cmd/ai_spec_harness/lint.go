package aispecharness

import (
	"fmt"

	"github.com/JailtonJunior94/ai-spec-harness/internal/detect"
	"github.com/JailtonJunior94/ai-spec-harness/internal/fs"
	"github.com/JailtonJunior94/ai-spec-harness/internal/lint"
	"github.com/JailtonJunior94/ai-spec-harness/internal/output"
	"github.com/JailtonJunior94/ai-spec-harness/internal/parity"
	"github.com/spf13/cobra"
)

func newLintCmd() *cobra.Command {
	var lintStrict bool
	var lintSchema bool

	cmd := &cobra.Command{
		Use:   "lint [path]",
		Short: "Verifica governança em arquivos gerados",
		Long: `Detecta problemas de governança no projeto:
  - Placeholders não renderizados {{ em AGENTS.md, CLAUDE.md, GEMINI.md, .codex/config.toml, copilot-instructions.md
  - Versão de governance-schema em AGENTS.md divergente da versão atual do CLI
  - bug-schema.json inválido
  - SKILL.md com frontmatter inválido
  - Invariantes BestEffort de paridade (avisos; erros com --strict)

Exemplos:
  ai-spec-harness lint .
  ai-spec-harness lint ./meu-projeto
  ai-spec-harness lint . --strict`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			projectDir := "."
			if len(args) > 0 {
				projectDir = args[0]
			}

			printer := output.New(newCommandEnv().verbose(cmd))
			svc := lint.NewService()
			errs, err := svc.Execute(projectDir)
			if err != nil {
				return err
			}

			// Verificar invariantes BestEffort de paridade
			fsys := fs.NewOSFileSystem()
			det := detect.NewFileDetector(fsys)
			tools := det.DetectTools(projectDir)
			langs := det.DetectLangs(projectDir)

			var parityWarnings []parity.CheckResult
			if len(tools) > 0 {
				snap, snapErr := parity.NewChecker().Generate(projectDir, tools, langs, "")
				if snapErr == nil {
					results := parity.NewChecker().Run(snap, parity.NewChecker().Invariants())
					parityWarnings = parity.NewChecker().Warnings(results)
				}
			}

			// Exibir avisos (BestEffort) — nao bloqueiam por padrao
			for _, w := range parityWarnings {
				printer.Warn("[%s] %s — %s", w.Invariant.ID, w.Invariant.Description, w.Result.Reason)
			}

			// --strict: promover warnings a erros
			if lintStrict && len(parityWarnings) > 0 {
				for _, w := range parityWarnings {
					errs = append(errs, lint.LintError{
						File:    projectDir,
						Message: fmt.Sprintf("[%s] %s — %s", w.Invariant.ID, w.Invariant.Description, w.Result.Reason),
					})
				}
			}

			if len(errs) == 0 {
				n := svc.CountChecks(projectDir)
				if len(parityWarnings) > 0 {
					fmt.Printf("Lint aprovado: %d verificacoes passaram (%d avisos de paridade)\n", n, len(parityWarnings))
				} else {
					fmt.Printf("Lint aprovado: %d verificacoes passaram\n", n)
				}
				return nil
			}

			for _, e := range errs {
				fmt.Println(e.String())
			}
			fmt.Printf("%d erro(s) encontrado(s)\n", len(errs))
			return newExitError(1)
		},
	}

	cmd.Flags().BoolVar(&lintStrict, "strict", false, "Promove avisos de paridade (BestEffort) a erros (exit code 1)")
	cmd.Flags().BoolVar(&lintSchema, "schema", false, "Valida skills contra docs/skill-schema.json")
	return cmd
}
