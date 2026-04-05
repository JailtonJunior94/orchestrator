package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/jailtonjunior/orchestrator/internal/bootstrap"
	runtimeapp "github.com/jailtonjunior/orchestrator/internal/runtime/application"
	"github.com/jailtonjunior/orchestrator/internal/tui"
	"github.com/jailtonjunior/orchestrator/internal/tui/components"
	"github.com/spf13/cobra"
)

var (
	shouldUseTUI       = tui.ShouldUseTUI
	runWorkflowTUI     = tui.RunTUI
	runWorkflowListTUI = tui.RunListTUI
)

// NewRunCommand creates the `orq run` command.
func NewRunCommand(app *bootstrap.App) *cobra.Command {
	var inlineInput string
	var fileInput string

	cmd := &cobra.Command{
		Use:   "run <workflow>",
		Short: "Executa um workflow built-in",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if inlineInput != "" && fileInput != "" {
				return fmt.Errorf("--input e --file não podem ser usados juntos")
			}

			var input string
			switch {
			case inlineInput != "":
				input = inlineInput
			case fileInput != "":
				data, err := os.ReadFile(fileInput)
				if err != nil {
					return fmt.Errorf("falha ao ler arquivo de input %q: %w", fileInput, err)
				}
				input = string(data)
			}

			return translateError(executeWorkflowCommand(cmd, app, args[0], input))
		},
	}

	cmd.Flags().StringVar(&inlineInput, "input", "", "Input inline do workflow")
	cmd.Flags().StringVarP(&fileInput, "file", "f", "", "Arquivo com o input do workflow")
	return cmd
}

func executeWorkflowCommand(cmd *cobra.Command, app *bootstrap.App, workflowName string, input string) error {
	summary, err := findWorkflowSummary(cmd.Context(), app.Runtime, workflowName)
	if err != nil {
		return err
	}

	resolvedInput, err := resolveWorkflowInput(cmd, workflowName, summary, input)
	if err != nil {
		return err
	}

	if app.TUIWiring != nil {
		noAnimation, _ := cmd.Root().PersistentFlags().GetBool("no-animation")
		noAnimation = noAnimation || os.Getenv("ORQ_NO_ANIMATION") != ""
		return runWorkflowTUI(cmd.Context(), app.Runtime, app.TUIWiring, workflowName, resolvedInput, noAnimation)
	}

	_, err = app.Runtime.Run(cmd.Context(), workflowName, resolvedInput)
	return err
}

func resolveWorkflowInput(cmd *cobra.Command, _ string, summary *runtimeapp.WorkflowSummary, provided string) (string, error) {
	if strings.TrimSpace(provided) != "" {
		return provided, nil
	}

	if !workflowRequiresInput(summary) {
		return "", nil
	}

	noTUI, _ := cmd.Root().PersistentFlags().GetBool("no-tui")
	if shouldUseTUI(noTUI) {
		return collectInputInteractive(summary)
	}

	return "", fmt.Errorf("missing required inputs: %v", missingInputNames(summary))
}

func workflowRequiresInput(summary *runtimeapp.WorkflowSummary) bool {
	return summary != nil && (summary.RequiresInput || len(summary.Inputs) > 0)
}

func missingInputNames(summary *runtimeapp.WorkflowSummary) []string {
	if summary == nil {
		return []string{"input"}
	}

	names := make([]string, 0, len(summary.Inputs))
	for _, input := range summary.Inputs {
		if input.Required || len(summary.Inputs) == 1 {
			names = append(names, input.Name)
		}
	}
	if len(names) > 0 {
		return names
	}
	if summary.RequiresInput {
		return []string{"input"}
	}

	return nil
}

func findWorkflowSummary(ctx context.Context, svc runtimeapp.Service, workflowName string) (*runtimeapp.WorkflowSummary, error) {
	summaries, err := svc.ListWorkflowDetails(ctx)
	if err != nil {
		return nil, err
	}
	for _, summary := range summaries {
		if summary.Name == workflowName {
			match := summary
			return &match, nil
		}
	}

	return nil, nil
}

// collectInputInteractive runs a blocking huh form to collect workflow input.
func collectInputInteractive(summary *runtimeapp.WorkflowSummary) (string, error) {
	fields := workflowInputFields(summary)
	form, result := components.BuildInputForm(fields)
	if err := form.Run(); err != nil {
		return "", fmt.Errorf("input form cancelled: %w", err)
	}

	values := make(map[string]any, len(fields))
	for _, field := range fields {
		if val := result.StringValues[field.Name]; val != nil && strings.TrimSpace(*val) != "" {
			values[field.Name] = *val
		}
		if val := result.BoolValues[field.Name]; val != nil {
			values[field.Name] = *val
		}
	}

	input := renderWorkflowInput(summary, values)
	if strings.TrimSpace(input) == "" {
		return "", fmt.Errorf("missing required inputs: %v", missingInputNames(summary))
	}

	return input, nil
}

func workflowInputFields(summary *runtimeapp.WorkflowSummary) []components.InputField {
	if summary == nil || len(summary.Inputs) == 0 {
		return []components.InputField{{
			Name:        "input",
			Type:        "text",
			Label:       "Workflow Input",
			Placeholder: "Describe what you want to build...",
		}}
	}

	fields := make([]components.InputField, 0, len(summary.Inputs))
	for _, input := range summary.Inputs {
		fieldType := input.Type
		if fieldType == "" {
			fieldType = "text"
		}
		label := input.Label
		if label == "" {
			label = input.Name
		}
		fields = append(fields, components.InputField{
			Name:        input.Name,
			Type:        fieldType,
			Label:       label,
			Placeholder: input.Placeholder,
			Options:     append([]string(nil), input.Options...),
		})
	}

	return fields
}

func renderWorkflowInput(summary *runtimeapp.WorkflowSummary, values map[string]any) string {
	if summary != nil && len(summary.Inputs) == 1 {
		if value, ok := values[summary.Inputs[0].Name]; ok {
			switch typed := value.(type) {
			case string:
				return typed
			case bool:
				if typed {
					return "true"
				}
				return "false"
			}
		}
	}

	if len(values) == 1 {
		if value, ok := values["input"]; ok {
			if typed, ok := value.(string); ok {
				return typed
			}
		}
	}

	data, err := json.MarshalIndent(values, "", "  ")
	if err != nil {
		return ""
	}

	return string(data)
}
