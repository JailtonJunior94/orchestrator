package aispecharness

import (
	"time"

	"github.com/JailtonJunior94/ai-spec-harness/internal/hookaudit"
	"github.com/spf13/cobra"
)

const hookAuditDefaultPath = ".aispec/hook-decisions.jsonl"

func newHookAuditCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "hookaudit", Short: "Registro canonico de decisoes de hooks de governanca"}
	cmd.AddCommand(newHookAuditAppendCmd())
	return cmd
}

func newHookAuditAppendCmd() *cobra.Command {
	var (
		path       string
		event      string
		provider   string
		hook       string
		decision   string
		reason     string
		policyID   string
		gateID     string
		durationMS int64
	)
	cmd := &cobra.Command{
		Use:   "append",
		Short: "Acrescenta uma decisao ao registro canonico de auditoria de hooks",
		RunE: func(cmd *cobra.Command, args []string) error {
			entry, err := hookaudit.NewEntry(time.Now().UTC(), event, provider, hook, decision, durationMS)
			if err != nil {
				return err
			}
			entry = entry.WithReason(reason, policyID, gateID)

			writer, err := hookaudit.NewWriter(path)
			if err != nil {
				return err
			}
			return writer.Append(entry)
		},
	}
	cmd.Flags().StringVar(&path, "path", hookAuditDefaultPath, "Caminho do arquivo JSONL de auditoria")
	cmd.Flags().StringVar(&event, "event", "", "Evento canonico (ex.: before_tool)")
	cmd.Flags().StringVar(&provider, "provider", "", "Provedor de CLI (claude, codex, copilot, opencode)")
	cmd.Flags().StringVar(&hook, "hook", "", "Nome do hook que produziu a decisao")
	cmd.Flags().StringVar(&decision, "decision", "", "Decisao tomada (ex.: BLOCK, ALLOW)")
	cmd.Flags().StringVar(&reason, "reason", "", "Motivo legivel da decisao")
	cmd.Flags().StringVar(&policyID, "policy-id", "", "Identificador da politica aplicada")
	cmd.Flags().StringVar(&gateID, "gate-id", "", "Identificador do gate que decidiu")
	cmd.Flags().Int64Var(&durationMS, "duration-ms", 0, "Duracao da avaliacao em milissegundos")
	return cmd
}
