package aispecharness

import (
	"fmt"
	"sort"

	"github.com/JailtonJunior94/ai-spec-harness/internal/fs"
	"github.com/JailtonJunior94/ai-spec-harness/internal/output"
	"github.com/JailtonJunior94/ai-spec-harness/internal/skillscheck"
	"github.com/spf13/cobra"
)

type skillsCommand struct{}

func newSkillsCmd() *cobra.Command {
	var verify bool

	cmd := &cobra.Command{
		Use:   "skills",
		Short: "Gerencia skills externas e detecta mudancas de versao",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if !verify {
				return cmd.Help()
			}
			projectDir := "."
			if len(args) > 0 {
				projectDir = args[0]
			}
			return runSkillsVerify(cmd, projectDir)
		},
	}

	cmd.Flags().BoolVar(&verify, "verify", false,
		"Gate de integridade: versao sem breaking + hash SHA-256 igual ao lock (exit != 0 em divergencia)")
	cmd.AddCommand(newSkillsCheckCmd())
	return cmd
}

// runSkillsVerify executa o gate de integridade das skills externas (ADR-005).
// Emite uma linha por falha e retorna exitError(1) se houver divergencia;
// usado como gate bloqueante pelas skills execute-task/execute-all-tasks.
func runSkillsVerify(cmd *cobra.Command, projectDir string) error {
	printer := output.New(newCommandEnv().verbose(cmd))
	fsys := fs.NewOSFileSystem()
	svc := skillscheck.NewService(fsys, printer)

	failures, err := svc.Verify(projectDir)
	if err != nil {
		return err
	}

	if len(failures) > 0 {
		sort.Slice(failures, func(i, j int) bool {
			return failures[i].Check.Name < failures[j].Check.Name
		})
		for _, f := range failures {
			fmt.Printf("[!!] %s: %s\n", f.Check.Name, f.Reason)
		}
		fmt.Printf("\n%d divergencia(s) de integridade. Atualize skills-lock.json apos registrar a decisao em audit/.\n", len(failures))
		return newExitError(1)
	}

	fmt.Println("Integridade das skills OK (versao + hash SHA-256).")
	return nil
}

func newSkillsCheckCmd() *cobra.Command {
	handler := &skillsCommand{}
	var skillsCheckForce bool

	cmd := &cobra.Command{
		Use:   "check [path]",
		Short: "Verifica versoes de skills externas contra skills-lock.json",
		Long: `Compara a versao de cada skill instalada em .agents/skills/ com a versao registrada
em skills-lock.json. Detecta upgrades compativeis (minor/patch) e potencialmente
quebra de interface (major bump).

Exemplos:
  ai-spec-harness skills check .
  ai-spec-harness skills check . --force`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			projectDir := "."
			if len(args) > 0 {
				projectDir = args[0]
			}

			printer := output.New(newCommandEnv().verbose(cmd))
			fsys := fs.NewOSFileSystem()
			svc := skillscheck.NewService(fsys, printer)

			results, err := svc.Check(projectDir)
			if err != nil {
				return err
			}

			if len(results) == 0 {
				fmt.Println("Nenhuma skill externa registrada em skills-lock.json.")
				return nil
			}

			// Ordenar por nome para saida deterministica
			sort.Slice(results, func(i, j int) bool {
				return results[i].Name < results[j].Name
			})

			hasBreaking := false
			for _, r := range results {
				status := handler.statusIcon(r.Drift)
				switch r.Drift {
				case skillscheck.DriftNone:
					printer.Info("  %s %-40s v%s", status, r.Name, r.LockedVer)
				case skillscheck.DriftMinor:
					printer.Info("  %s %-40s lock=v%s installed=v%s (compativel)", status, r.Name, r.LockedVer, r.InstalledVer)
				case skillscheck.DriftBreaking:
					printer.Warn("[%s] %-40s lock=v%s installed=v%s — BREAKING: major version bump", status, r.Name, r.LockedVer, r.InstalledVer)
					hasBreaking = true
				case skillscheck.DriftNoSkill:
					printer.Warn("[%s] %-40s no lock entry v%s — skill nao instalada", status, r.Name, r.LockedVer)
				case skillscheck.DriftUnknown:
					printer.Info("  %s %-40s versao desconhecida (lock=v%s installed=v%s)", status, r.Name, r.LockedVer, r.InstalledVer)
				}
			}

			if hasBreaking && !skillsCheckForce {
				fmt.Println()
				fmt.Println("AVISO: skills com breaking changes detectadas.")
				fmt.Println("Use --force para aceitar e atualizar skills-lock.json.")
				return newExitError(1)
			}

			fmt.Printf("\n%d skill(s) verificadas.\n", len(results))
			return nil
		},
	}

	cmd.Flags().BoolVar(&skillsCheckForce, "force", false, "Aceita breaking changes sem bloquear (exit code 0)")
	return cmd
}

func (c *skillsCommand) statusIcon(drift skillscheck.VersionDrift) string {
	switch drift {
	case skillscheck.DriftNone:
		return "OK"
	case skillscheck.DriftMinor:
		return "UP"
	case skillscheck.DriftBreaking:
		return "!!"
	case skillscheck.DriftNoSkill:
		return "??"
	default:
		return "--"
	}
}
