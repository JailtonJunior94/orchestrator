package aispecharness

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/JailtonJunior94/ai-spec-harness/internal/telemetry"
	"github.com/spf13/cobra"
)

func newTelemetryCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "telemetry",
		Short: "Gerencia telemetria de uso de skills",
	}

	cmd.AddCommand(newTelemetryLogCmd())
	cmd.AddCommand(newTelemetrySummaryCmd())
	cmd.AddCommand(newTelemetryReportCmd())
	return cmd
}

func newTelemetryLogCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "log <skill> [ref]",
		Short: "Registra uso de skill em telemetria (requer GOVERNANCE_TELEMETRY=1)",
		Args:  cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			skill := args[0]
			ref := ""
			if len(args) > 1 {
				ref = args[1]
			}
			return telemetry.NewCatalog().Log(".", skill, ref)
		},
	}
	return cmd
}

func newTelemetrySummaryCmd() *cobra.Command {
	var telemetrySince string

	cmd := &cobra.Command{
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
			result, err := telemetry.NewCatalog().Summary(".", since)
			if err != nil {
				return err
			}
			fmt.Print(result)
			return nil
		},
	}

	cmd.Flags().StringVar(&telemetrySince, "since", "", "Filtrar por periodo (ex: 1h, 24h, 168h)")
	return cmd
}

func newTelemetryReportCmd() *cobra.Command {
	var reportSince string
	var reportFormat string
	var reportTrend bool
	var reportTopSkills bool
	var reportBudget bool

	cmd := &cobra.Command{
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

			// Modo --trend: exibe evolucao semanal de invocacoes
			if reportTrend {
				trend, err := telemetry.NewCatalog().Trend(".")
				if err != nil {
					return err
				}
				if reportFormat == "json" {
					b, err := json.MarshalIndent(trend, "", "  ")
					if err != nil {
						return fmt.Errorf("serializar trend: %w", err)
					}
					_, err = fmt.Fprintf(os.Stdout, "%s\n", b)
					return err
				}
				fmt.Print(telemetry.NewCatalog().FormatTrend(trend))
				return nil
			}

			// Modo --budget-check: verifica budget de invocacoes por skill
			if reportBudget {
				budgetData, err := telemetry.NewCatalog().BudgetCheck(".", since)
				if err != nil {
					return err
				}
				if reportFormat == "json" {
					b, err := json.MarshalIndent(budgetData, "", "  ")
					if err != nil {
						return fmt.Errorf("serializar budget-check: %w", err)
					}
					_, err = fmt.Fprintf(os.Stdout, "%s\n", b)
					return err
				}
				fmt.Print(telemetry.NewCatalog().FormatBudgetCheck(budgetData))
				return nil
			}

			data, err := telemetry.NewCatalog().Report(".", since)
			if err != nil {
				return err
			}

			// Modo --top-skills: exibe apenas ranking de skills
			if reportTopSkills {
				if reportFormat == "json" {
					b, err := json.MarshalIndent(data.Skills, "", "  ")
					if err != nil {
						return fmt.Errorf("serializar top-skills: %w", err)
					}
					_, err = fmt.Fprintf(os.Stdout, "%s\n", b)
					return err
				}
				fmt.Print(telemetry.NewCatalog().FormatTopSkills(data.Skills))
				return nil
			}

			switch reportFormat {
			case "json":
				b, err := telemetry.NewCatalog().FormatJSON(data)
				if err != nil {
					return err
				}
				_, err = fmt.Fprintf(os.Stdout, "%s\n", b)
				return err
			default:
				fmt.Print(telemetry.NewCatalog().FormatText(data))
				return nil
			}
		},
	}

	cmd.Flags().StringVar(&reportSince, "since", "", "Filtrar por período (ex: 24h, 168h)")
	cmd.Flags().StringVar(&reportFormat, "format", "text", "Formato de saída: text ou json")
	cmd.Flags().BoolVar(&reportTrend, "trend", false, "Exibe evolucao de invocacoes por semana (ultimas 4 semanas)")
	cmd.Flags().BoolVar(&reportTopSkills, "top-skills", false, "Exibe ranking de skills por frequencia de uso")
	cmd.Flags().BoolVar(&reportBudget, "budget-check", false, "Alerta se alguma skill excedeu o budget de invocacoes esperado")
	return cmd
}
