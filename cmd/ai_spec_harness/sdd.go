package aispecharness

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/JailtonJunior94/ai-spec-harness/internal/sdd"
	"github.com/JailtonJunior94/ai-spec-harness/internal/taskloop"
	"github.com/spf13/cobra"
)

func newApproveCmd() *cobra.Command {
	return &cobra.Command{Use: "approve <prd|techspec|tasks> <diretorio-prd>", Short: "Aprova um artefato SDD e registra seu digest imutável", Args: cobra.ExactArgs(2), RunE: func(cmd *cobra.Command, args []string) error {
		store := sdd.NewStore()
		if _, err := os.Stat(store.StatePath(args[1])); os.IsNotExist(err) {
			if _, err := store.Initialize(args[1], "manual"); err != nil {
				return err
			}
		}
		if _, err := store.Approve(args[1], sdd.Artifact(args[0])); err != nil {
			return err
		}
		fmt.Printf("OK: %s aprovado em %s\n", args[0], args[1])
		return nil
	}}
}

func newInvalidateCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "invalidate <diretorio-prd> --from <prd|techspec>", Short: "Invalida artefatos downstream aprovados", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		from, _ := cmd.Flags().GetString("from")
		if from != string(sdd.ArtifactPRD) && from != string(sdd.ArtifactTechSpec) {
			return fmt.Errorf("--from deve ser prd ou techspec")
		}
		if _, err := sdd.NewStore().Invalidate(args[0], sdd.Artifact(from)); err != nil {
			return err
		}
		fmt.Printf("OK: downstream de %s invalidado em %s\n", from, args[0])
		return nil
	}}
	cmd.Flags().String("from", "", "artefato de origem")
	return cmd
}

func newOrchestrateCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "orchestrate <diretorio-prd> --run-id <id>", Short: "Valida estado SDD e inicia uma execução recuperável", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		runID, _ := cmd.Flags().GetString("run-id")
		store := sdd.NewStore()
		state, err := store.Load(args[0])
		if os.IsNotExist(err) {
			state, err = store.Initialize(args[0], runID)
		}
		if err != nil {
			return err
		}
		for _, artifact := range []sdd.Artifact{sdd.ArtifactPRD, sdd.ArtifactTechSpec, sdd.ArtifactTasks} {
			entry := state.Artifacts[artifact]
			if !entry.Approved || entry.Status != sdd.StatusApproved {
				return fmt.Errorf("orquestracao bloqueada: %s nao esta aprovado", artifact)
			}
		}
		fmt.Printf("OK: estado SDD validado para run_id=%s\n", state.RunID)
		return nil
	}}
	cmd.Flags().String("run-id", "manual", "identificador da execução")
	return cmd
}

func newValidateSDDCmd() *cobra.Command {
	return &cobra.Command{Use: "validate-sdd <diretorio-prd>", Short: "Valida o contrato SDD versionado de um PRD", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		state, err := sdd.NewStore().Load(args[0])
		if err != nil {
			return err
		}
		fmt.Printf("OK: estado SDD v%d valido para %s\n", state.SchemaVersion, filepath.Clean(args[0]))
		return nil
	}}
}

func newRuntimeCapabilitiesCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "runtime-capabilities <diretorio>",
		Short: "Detecta capacidades locais para orquestração segura",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			capabilities, err := taskloop.DetectRuntimeCapabilities(cmd.Context(), args[0])
			if err != nil {
				return fmt.Errorf("detectar capacidades de runtime: %w", err)
			}
			if err := json.NewEncoder(cmd.OutOrStdout()).Encode(capabilities); err != nil {
				return fmt.Errorf("serializar capacidades de runtime: %w", err)
			}
			return nil
		},
	}
}
