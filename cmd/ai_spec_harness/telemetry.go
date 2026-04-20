package aispecharness

import (
	"fmt"
	"os"
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

var (
	reportSince  string
	reportFormat string
)

var telemetryReportCmd = &cobra.Command{
	Use:   "report",
	Short: "Exibe relatório acionável de telemetria com métricas e alertas",
	RunE: func(cmd *cobra.Command, args []string) error {
		var since time.Duration
		if reportSince != "" {
			d, err := time.ParseDuration(reportSince)
			if err != nil {
				return fmt.Errorf("duração inválida %q: %w", reportSince, err)
			}
			since = d
		}

		data, err := telemetry.Report(".", since)
		if err != nil {
			return err
		}

		switch reportFormat {
		case "json":
			b, err := telemetry.FormatJSON(data)
			if err != nil {
				return err
			}
			_, err = fmt.Fprintf(os.Stdout, "%s\n", b)
			return err
		default:
			fmt.Print(telemetry.FormatText(data))
			return nil
		}
	},
}

func init() {
	telemetrySummaryCmd.Flags().StringVar(&telemetrySince, "since", "", "Filtrar por periodo (ex: 1h, 24h, 168h)")

	telemetryReportCmd.Flags().StringVar(&reportSince, "since", "", "Filtrar por período (ex: 24h, 168h)")
	telemetryReportCmd.Flags().StringVar(&reportFormat, "format", "text", "Formato de saída: text ou json")

	telemetryCmd.AddCommand(telemetryLogCmd)
	telemetryCmd.AddCommand(telemetrySummaryCmd)
	telemetryCmd.AddCommand(telemetryReportCmd)
	rootCmd.AddCommand(telemetryCmd)
}
