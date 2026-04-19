package aispecharness

import (
	"fmt"
	"time"

	"github.com/JailtonJunior94/ai-spec-harness/internal/telemetry"
	"github.com/spf13/cobra"
)

var telemetryCmd = &cobra.Command{
	Use:   "telemetry",
	Short: "Gerencia telemetria de uso de skills",
}

var telemetryLogCmd = &cobra.Command{
	Use:   "log <skill> [ref]",
	Short: "Registra uso de skill em telemetria (requer GOVERNANCE_TELEMETRY=1)",
	Args:  cobra.RangeArgs(1, 2),
	RunE: func(cmd *cobra.Command, args []string) error {
		skill := args[0]
		ref := ""
		if len(args) > 1 {
			ref = args[1]
		}
		return telemetry.Log(".", skill, ref)
	},
}

var telemetrySince string

var telemetrySummaryCmd = &cobra.Command{
	Use:   "summary",
	Short: "Exibe resumo de telemetria agrupado por skill e referencia",
	RunE: func(cmd *cobra.Command, args []string) error {
		var since time.Duration
		if telemetrySince != "" {
			d, err := time.ParseDuration(telemetrySince)
			if err != nil {
				return fmt.Errorf("duracao invalida %q: %w", telemetrySince, err)
			}
			since = d
		}
		result, err := telemetry.Summary(".", since)
		if err != nil {
			return err
		}
		fmt.Print(result)
		return nil
	},
}

func init() {
	telemetrySummaryCmd.Flags().StringVar(&telemetrySince, "since", "", "Filtrar por periodo (ex: 1h, 24h, 168h)")

	telemetryCmd.AddCommand(telemetryLogCmd)
	telemetryCmd.AddCommand(telemetrySummaryCmd)
	rootCmd.AddCommand(telemetryCmd)
}
