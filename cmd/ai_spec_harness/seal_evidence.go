package aispecharness

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/JailtonJunior94/ai-spec-harness/internal/sdd"
	"github.com/JailtonJunior94/ai-spec-harness/internal/taskloop"
)

func newSealEvidenceCmd() *cobra.Command {
	command := &cobra.Command{
		Use:   "seal-evidence <arquivo-resultado-json>",
		Short: "Vincula a evidência de uma tarefa ao commit que a contém (RF-14)",
		Long: `Grava commit_sha e o digest do patch recomputado no range base..commit.

A prova de fechamento é verificada contra a árvore de trabalho viva, que deixa de
existir assim que o trabalho é commitado — por isso ela não é re-auditável depois.
O selo torna a evidência permanente: qualquer auditor a reverifica a partir apenas
dos dois SHAs, sem depender do estado atual do repositório.

O selo não prova que o commit é byte-idêntico à árvore do fechamento: essa árvore
já não existe quando o selo é aplicado.

Exemplos:
  ai-spec seal-evidence .specs/prd-x/result.json --prd-dir .specs/prd-x
  ai-spec seal-evidence result.json --prd-dir .specs/prd-x --commit HEAD
  ai-spec seal-evidence result.json --prd-dir .specs/prd-x --verify`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			prdDir, _ := cmd.Flags().GetString("prd-dir")
			if prdDir == "" {
				return fmt.Errorf("--prd-dir e obrigatorio")
			}
			commit, _ := cmd.Flags().GetString("commit")
			verifyOnly, _ := cmd.Flags().GetBool("verify")

			resultPath, err := filepath.Abs(args[0])
			if err != nil {
				return fmt.Errorf("resolver arquivo de resultado: %w", err)
			}
			content, err := os.ReadFile(resultPath)
			if err != nil {
				return fmt.Errorf("ler resultado SDD: %w", err)
			}
			result, err := sdd.NewResultValidator().ValidateExecutionJSON(content)
			if err != nil {
				return err
			}

			orchestrator := taskloop.NewOrchestrator(sdd.NewStore())
			if verifyOnly {
				if err := orchestrator.VerifySealedEvidence(prdDir, result, resultPath); err != nil {
					return err
				}
				fmt.Fprintf(cmd.OutOrStdout(), "OK: evidencia da tarefa %s conferida no commit %s\n", result.TaskID, result.CommitSHA)
				return nil
			}

			sealed, err := orchestrator.SealEvidence(prdDir, result, commit, resultPath)
			if err != nil {
				return err
			}
			encoded, err := json.MarshalIndent(sealed, "", "  ")
			if err != nil {
				return fmt.Errorf("serializar resultado selado: %w", err)
			}
			if err := os.WriteFile(resultPath, append(encoded, '\n'), 0o644); err != nil {
				return fmt.Errorf("gravar resultado selado: %w", err)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "OK: evidencia da tarefa %s selada no commit %s (patch %s)\n",
				sealed.TaskID, sealed.CommitSHA, sealed.CommitPatchSHA256[:12])
			return nil
		},
	}
	command.Flags().String("prd-dir", "", "diretorio do PRD que contem o estado SDD")
	command.Flags().String("commit", "HEAD", "commit que contem o trabalho provado")
	command.Flags().Bool("verify", false, "apenas reverifica um selo existente, sem gravar")
	return command
}
