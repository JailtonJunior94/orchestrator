package aispecharness

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/JailtonJunior94/ai-spec-harness/internal/semver"
	"github.com/spf13/cobra"
)

var semverNextFormat string

var semverNextCmd = &cobra.Command{
	Use:   "semver-next [path]",
	Short: "Calcula a proxima versao SemVer a partir de conventional commits",
	Long: `Calcula a proxima versao SemVer analisando conventional commits desde o ultimo tag v*.

Exemplos:
  ai-spec-harness semver-next .
  ai-spec-harness semver-next ./meu-repo --format json`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		repoPath := "."
		if len(args) > 0 {
			repoPath = args[0]
		}

		d, err := semver.Evaluate(repoPath)
		if err != nil {
			return err
		}

		switch semverNextFormat {
		case "json":
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			return enc.Encode(d)
		default:
			fmt.Printf("action=%s\n", d.Action)
			fmt.Printf("bootstrap_required=%v\n", d.BootstrapRequired)
			fmt.Printf("release_required=%v\n", d.ReleaseRequired)
			fmt.Printf("last_tag=%s\n", d.LastTag)
			fmt.Printf("base_version=%s\n", d.BaseVersion)
			fmt.Printf("bump=%s\n", d.Bump)
			fmt.Printf("target_version=%s\n", d.TargetVersion)
			fmt.Printf("commit_range=%s\n", d.CommitRange)
			fmt.Printf("commit_count=%d\n", d.CommitCount)
		}
		return nil
	},
}

func init() {
	semverNextCmd.Flags().StringVar(&semverNextFormat, "format", "text", "Formato de saida: text ou json")
	rootCmd.AddCommand(semverNextCmd)
}
