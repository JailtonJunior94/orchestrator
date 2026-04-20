package aispecharness

import (
	"fmt"
	"os"
	"sort"

	"github.com/JailtonJunior94/ai-spec-harness/internal/fs"
	"github.com/JailtonJunior94/ai-spec-harness/internal/output"
	"github.com/JailtonJunior94/ai-spec-harness/internal/skillscheck"
	"github.com/spf13/cobra"
)

var skillsCmd = &cobra.Command{
	Use:   "skills",
	Short: "Gerencia skills externas e detecta mudancas de versao",
}

var skillsCheckForce bool

var skillsCheckCmd = &cobra.Command{
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

		printer := output.New(verbose)
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
			status := statusIcon(r.Drift)
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
			os.Exit(1)
		}

		fmt.Printf("\n%d skill(s) verificadas.\n", len(results))
		return nil
	},
}

func statusIcon(drift skillscheck.VersionDrift) string {
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

func init() {
	skillsCheckCmd.Flags().BoolVar(&skillsCheckForce, "force", false, "Aceita breaking changes sem bloquear (exit code 0)")
	skillsCmd.AddCommand(skillsCheckCmd)
	rootCmd.AddCommand(skillsCmd)
}
