package aispecharness

import (
	"fmt"

	"github.com/JailtonJunior94/ai-spec-harness/internal/version"
	"github.com/spf13/cobra"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Exibe a versao do ai-spec-harness",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("ai-spec-harness %s (commit: %s, built: %s)\n", version.Version, version.Commit, version.Date)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
