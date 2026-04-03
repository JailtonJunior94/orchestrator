package runtime

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jailtonjunior/orchestrator/internal/hitl"
	"github.com/jailtonjunior/orchestrator/internal/output"
	"github.com/jailtonjunior/orchestrator/internal/platform"
	"github.com/jailtonjunior/orchestrator/internal/providers"
	"github.com/jailtonjunior/orchestrator/internal/runtime/domain"
	"github.com/jailtonjunior/orchestrator/internal/state"
	"github.com/jailtonjunior/orchestrator/internal/workflows"
)

const maxProviderRetries = 2

var errExecutionPaused = errors.New("execution paused")

// Engine orchestrates workflow execution and resumption.
type Engine interface {
	Run(ctx context.Context, workflowName string, input string) (*RunResult, error)
	Continue(ctx context.Context, runID string) (*RunResult, error)
}

// RunResult summarizes the final run state.
type RunResult struct {
	Run *domain.Run
}

// ProgressReporter receives step progress notifications.
type ProgressReporter interface {
	StepStarted(step ProgressStep)
	StepFinished(step ProgressStep)
}

// ProgressStep is emitted when a step starts or finishes.
type ProgressStep struct {
	Index    int
	Total    int
	Name     string
	Provider string
	Status   string
	Duration time.Duration
}

// NoopProgressReporter ignores progress updates.
type NoopProgressReporter struct{}

func (NoopProgressReporter) StepStarted(ProgressStep)  {}
func (NoopProgressReporter) StepFinished(ProgressStep) {}

// Catalog exposes workflow loading and listing.
type Catalog interface {
	Load(ctx context.Context, name string) (*workflows.WorkflowDefinition, error)
	List() ([]string, error)
}

// DefaultEngine is the production workflow orchestrator.
type DefaultEngine struct {
	catalog   Catalog
	validator workflows.Validator
	resolver  workflows.TemplateResolver
	providers providers.Factory
	processor output.Processor
	store     state.Store
	prompter  hitl.Prompter
	clock     platform.Clock
	fs        platform.FileSystem
	runner    platform.CommandRunner
	logger    *slog.Logger
	progress  ProgressReporter
}

// Dependencies groups engine collaborators for construction.
type Dependencies struct {
	Catalog    Catalog
	Validator  workflows.Validator
	Resolver   workflows.TemplateResolver
	Providers  providers.Factory
	Processor  output.Processor
	Store      state.Store
	Prompter   hitl.Prompter
	Clock      platform.Clock
	FileSystem platform.FileSystem
	Runner     platform.CommandRunner
	Logger     *slog.Logger
	Progress   ProgressReporter
}

// NewEngine creates the default engine implementation.
func NewEngine(deps Dependencies) *DefaultEngine {
	progress := deps.Progress
	if progress == nil {
		progress = NoopProgressReporter{}
	}

	logger := deps.Logger
	if logger == nil {
		logger = slog.Default()
	}

	fileSystem := deps.FileSystem
	if fileSystem == nil {
		fileSystem = platform.NewFileSystem()
	}

	runner := deps.Runner
	if runner == nil {
		runner = platform.NewCommandRunner()
	}

	return &DefaultEngine{
		catalog:   deps.Catalog,
		validator: deps.Validator,
		resolver:  deps.Resolver,
		providers: deps.Providers,
		processor: deps.Processor,
		store:     deps.Store,
		prompter:  deps.Prompter,
		clock:     deps.Clock,
		fs:        fileSystem,
		runner:    runner,
		logger:    logger,
		progress:  progress,
	}
}

// Run executes a built-in workflow from the beginning.
func (e *DefaultEngine) Run(ctx context.Context, workflowName string, input string) (*RunResult, error) {
	workflow, err := e.catalog.Load(ctx, workflowName)
	if err != nil {
		return nil, err
	}
	if err := e.validator.Validate(ctx, workflow); err != nil {
		return nil, err
	}

	stepDefs, err := workflowToDomainSteps(workflow)
	if err != nil {
		return nil, err
	}

	now := e.clock.Now()
	run, err := domain.NewRun(uuid.NewString(), mustWorkflowName(workflow.Name), input, stepDefs, now)
	if err != nil {
		return nil, err
	}

	if err := run.Start(now); err != nil {
		return nil, err
	}

	if err := e.store.SaveRun(ctx, run); err != nil {
		return nil, err
	}

	if err := e.execute(ctx, workflow, run); err != nil {
		return &RunResult{Run: run}, err
	}

	return &RunResult{Run: run}, nil
}

// Continue resumes a paused or pending run.
func (e *DefaultEngine) Continue(ctx context.Context, runID string) (*RunResult, error) {
	var (
		run *domain.Run
		err error
	)

	if runID == "" {
		run, err = e.store.FindLatestPending(ctx)
	} else {
		run, err = e.store.LoadRun(ctx, runID)
	}
	if err != nil {
		return nil, err
	}

	workflow, err := e.catalog.Load(ctx, run.Workflow().String())
	if err != nil {
		return nil, err
	}
	if err := e.validator.Validate(ctx, workflow); err != nil {
		return nil, err
	}

	switch run.Status() {
	case domain.RunPaused:
		if err := run.Resume(e.clock.Now()); err != nil {
			return nil, err
		}
	case domain.RunPending:
		if err := run.Start(e.clock.Now()); err != nil {
			return nil, err
		}
	}

	if err := e.store.SaveRun(ctx, run); err != nil {
		return nil, err
	}

	if err := e.execute(ctx, workflow, run); err != nil {
		return &RunResult{Run: run}, err
	}

	return &RunResult{Run: run}, nil
}

func (e *DefaultEngine) execute(ctx context.Context, workflow *workflows.WorkflowDefinition, run *domain.Run) error {
	total := len(workflow.Steps)
	for {
		step, err := run.CurrentStep()
		if err != nil {
			if errors.Is(err, domain.ErrNoCurrentStep) {
				return nil
			}
			return err
		}

		index := indexOfStep(workflow, step.Name().String()) + 1
		progressStep := ProgressStep{
			Index:    index,
			Total:    total,
			Name:     step.Name().String(),
			Provider: step.Provider().String(),
		}

		if step.Status() == domain.StepWaitingApproval {
			e.logStepEvent(ctx, run, step, "waiting_approval", 0, nil, nil)
			if err := e.handlePrompt(ctx, run, step); err != nil {
				if errors.Is(err, errExecutionPaused) {
					return nil
				}
				return err
			}
			continue
		}

		if err := run.StartStep(step.Name(), e.clock.Now()); err != nil {
			return err
		}
		if err := e.store.SaveRun(ctx, run); err != nil {
			return err
		}

		e.progress.StepStarted(progressStep)

		stepDef, err := findWorkflowStep(workflow, step.Name().String())
		if err != nil {
			return err
		}

		resolvedInput, err := e.resolveStepInput(ctx, run, workflow, step.Name().String())
		if err != nil {
			return err
		}
		step.SetInput(resolvedInput)

		provider, err := e.providers.Get(step.Provider().String())
		if err != nil {
			return err
		}
		if err := provider.Available(); err != nil {
			return err
		}

		stepTimeout, err := parseStepTimeout(stepDef)
		if err != nil {
			return err
		}

		e.logStepEvent(ctx, run, step, "provider_execute_started", 0, nil, nil)

		outputValue, rawOutput, providerDuration, err := e.runProviderWithRetries(
			ctx,
			provider,
			resolvedInput,
			stepTimeout,
			[]byte(stepDef.Schema),
			output.ProcessOptions{
				RequireStructured: stepDef.RequiresStructuredOutput(),
				SchemaName:        stepDef.Output.JSONSchema,
			},
		)
		progressStep.Duration = providerDuration
		artifactRefs := artifactRefsForStep(step.Name().String())
		if err != nil {
			if output.IsRecoverable(err) && rawOutput != "" {
				report := []byte(`{"validation_status":"failed","structured":true}`)
				if saveErr := e.store.SaveArtifact(ctx, run.ID(), step.Name().String(), state.Artifact{
					RawOutput:        []byte(rawOutput),
					ApprovedMarkdown: []byte(rawOutput),
					ValidationReport: report,
				}); saveErr != nil {
					return saveErr
				}

				if markErr := run.MarkStepCompleted(step.Name(), domain.StepResult{
					Output:              rawOutput,
					RawOutputRef:        artifactRefs.rawOutput,
					ApprovedMarkdownRef: artifactRefs.approvedMarkdown,
					ValidationReportRef: artifactRefs.validationReport,
					SchemaName:          stepDef.Output.JSONSchema,
					SchemaVersion:       "1",
					ValidationStatus:    domain.ValidationFailed,
				}, e.clock.Now()); markErr != nil {
					return markErr
				}
				if saveErr := e.store.SaveRun(ctx, run); saveErr != nil {
					return saveErr
				}
				e.logStepEvent(ctx, run, step, "provider_output_requires_hitl", providerDuration, err, nil)
				e.progress.StepFinished(ProgressStep{
					Index:    index,
					Total:    total,
					Name:     step.Name().String(),
					Provider: step.Provider().String(),
					Status:   string(domain.StepWaitingApproval),
					Duration: providerDuration,
				})
				if promptErr := e.handlePrompt(ctx, run, step); promptErr != nil {
					if errors.Is(promptErr, errExecutionPaused) {
						return nil
					}
					return promptErr
				}
				continue
			}

			_ = run.MarkStepFailed(step.Name(), err.Error(), e.clock.Now())
			_ = e.store.SaveRun(ctx, run)
			e.logStepEvent(ctx, run, step, "failed", providerDuration, err, nil)
			e.progress.StepFinished(ProgressStep{
				Index:    index,
				Total:    total,
				Name:     step.Name().String(),
				Provider: step.Provider().String(),
				Status:   string(domain.StepFailed),
				Duration: providerDuration,
			})
			return err
		}

		if err := e.store.SaveArtifact(ctx, run.ID(), step.Name().String(), state.Artifact{
			RawOutput:        []byte(rawOutput),
			ApprovedMarkdown: []byte(outputValue.Markdown),
			StructuredJSON:   outputValue.JSON,
			ValidationReport: outputValue.ValidationReport,
		}); err != nil {
			return err
		}

		if err := run.MarkStepCompleted(step.Name(), domain.StepResult{
			Output:              outputValue.Markdown,
			RawOutputRef:        artifactRefs.rawOutput,
			ApprovedMarkdownRef: artifactRefs.approvedMarkdown,
			StructuredJSONRef:   artifactRefs.structuredJSON,
			ValidationReportRef: artifactRefs.validationReport,
			SchemaName:          stepDef.Output.JSONSchema,
			SchemaVersion:       "1",
			ValidationStatus:    domain.ValidationStatus(outputValue.ValidationStatus),
			EditedByHuman:       false,
		}, e.clock.Now()); err != nil {
			return err
		}
		if err := e.store.SaveRun(ctx, run); err != nil {
			return err
		}
		e.logStepEvent(ctx, run, step, "provider_execute_finished", providerDuration, nil, map[string]any{
			"validation_status": outputValue.ValidationStatus,
		})

		e.progress.StepFinished(ProgressStep{
			Index:    index,
			Total:    total,
			Name:     step.Name().String(),
			Provider: step.Provider().String(),
			Status:   string(domain.StepWaitingApproval),
			Duration: providerDuration,
		})

		if err := e.handlePrompt(ctx, run, step); err != nil {
			if errors.Is(err, errExecutionPaused) {
				return nil
			}
			return err
		}
	}
}

func (e *DefaultEngine) runProviderWithRetries(ctx context.Context, provider providers.Provider, prompt string, timeout time.Duration, schema []byte, options output.ProcessOptions) (*output.Result, string, time.Duration, error) {
	var (
		lastErr    error
		lastOutput string
		duration   time.Duration
	)

	for attempt := 0; attempt <= maxProviderRetries; attempt++ {
		result, err := provider.Execute(ctx, providers.ProviderInput{Prompt: prompt, Timeout: timeout})
		duration = result.Duration
		lastOutput = result.Stdout
		if err != nil {
			return nil, lastOutput, duration, err
		}

		processed, err := e.processor.Process(ctx, result.Stdout, schema, options)
		if err == nil {
			return processed, lastOutput, duration, nil
		}

		lastErr = err
		if !output.IsRecoverable(err) || attempt == maxProviderRetries {
			return nil, lastOutput, duration, lastErr
		}
	}

	return nil, lastOutput, duration, lastErr
}

func (e *DefaultEngine) handlePrompt(ctx context.Context, run *domain.Run, step *domain.StepExecution) error {
	promptResult, err := e.prompter.Prompt(ctx, step.Output())
	if err != nil {
		return err
	}

	switch promptResult.Action {
	case hitl.ActionApprove:
		if err := run.ApproveStep(step.Name(), e.clock.Now()); err != nil {
			return err
		}
		e.logStepEvent(ctx, run, step, "approved", 0, nil, nil)
		if step.Name().String() == "execute" {
			if err := e.executePlan(ctx, run, step); err != nil {
				return err
			}
		}
	case hitl.ActionEdit:
		edited := promptResult.Output
		if err := run.UpdateStepOutput(step.Name(), edited, e.clock.Now()); err != nil {
			return err
		}
		result := step.Result()
		result.Output = edited
		result.EditedByHuman = true
		if err := run.UpdateStepResult(step.Name(), result, e.clock.Now()); err != nil {
			return err
		}
		if err := e.store.SaveArtifact(ctx, run.ID(), step.Name().String(), state.Artifact{
			ApprovedMarkdown: []byte(edited),
		}); err != nil {
			return err
		}
		if err := run.ApproveStep(step.Name(), e.clock.Now()); err != nil {
			return err
		}
		e.logStepEvent(ctx, run, step, "edited", 0, nil, nil)
		if step.Name().String() == "execute" {
			if err := e.executePlan(ctx, run, step); err != nil {
				return err
			}
		}
	case hitl.ActionRedo:
		if err := run.RetryStep(step.Name(), e.clock.Now()); err != nil {
			return err
		}
		e.logStepEvent(ctx, run, step, "redo_requested", 0, nil, nil)
	case hitl.ActionExit:
		if err := run.Pause(e.clock.Now()); err != nil {
			return err
		}
		if err := e.store.SaveRun(ctx, run); err != nil {
			return err
		}
		e.logStepEvent(ctx, run, step, "paused", 0, nil, nil)
		return errExecutionPaused
	default:
		return fmt.Errorf("unsupported hitl action: %d", promptResult.Action)
	}

	return e.store.SaveRun(ctx, run)
}

func (e *DefaultEngine) logStepEvent(ctx context.Context, run *domain.Run, step *domain.StepExecution, event string, duration time.Duration, err error, extra map[string]any) {
	attrs := []any{
		"run_id", run.ID(),
		"workflow", run.Workflow().String(),
		"step", step.Name().String(),
		"provider", step.Provider().String(),
		"duration_ms", duration.Milliseconds(),
		"event", event,
	}
	if err != nil {
		attrs = append(attrs, "error", err.Error())
		e.logger.Error("workflow step", attrs...)
	} else {
		e.logger.Info("workflow step", attrs...)
	}

	entry := map[string]any{
		"run_id":      run.ID(),
		"workflow":    run.Workflow().String(),
		"step":        step.Name().String(),
		"provider":    step.Provider().String(),
		"event":       event,
		"duration_ms": duration.Milliseconds(),
	}
	for key, value := range extra {
		entry[key] = value
	}
	if err != nil {
		entry["error"] = err.Error()
	}

	data, marshalErr := json.Marshal(entry)
	if marshalErr == nil {
		_ = e.store.AppendLog(ctx, run.ID(), data)
	}
}

func findWorkflowStep(workflow *workflows.WorkflowDefinition, stepName string) (*workflows.StepDefinition, error) {
	for idx := range workflow.Steps {
		if workflow.Steps[idx].Name == stepName {
			return &workflow.Steps[idx], nil
		}
	}

	return nil, fmt.Errorf("workflow step %q not found", stepName)
}

func parseStepTimeout(step *workflows.StepDefinition) (time.Duration, error) {
	if step.Timeout == "" {
		return 0, nil
	}

	timeout, err := time.ParseDuration(step.Timeout)
	if err != nil {
		return 0, fmt.Errorf("step %q: %w: %v", step.Name, workflows.ErrInvalidTimeout, err)
	}

	return timeout, nil
}

func (e *DefaultEngine) resolveStepInput(ctx context.Context, run *domain.Run, workflow *workflows.WorkflowDefinition, stepName string) (string, error) {
	template := ""
	for _, step := range workflow.Steps {
		if step.Name == stepName {
			template = step.Input
			break
		}
	}
	if template == "" {
		return "", fmt.Errorf("workflow step %q not found", stepName)
	}

	outputs := make(map[string]string)
	for _, step := range run.Steps() {
		if step.Output() != "" {
			outputs[step.Name().String()] = step.Output()
		}
	}

	return e.resolver.Resolve(ctx, template, workflows.TemplateVars{
		Input:       run.Input(),
		StepOutputs: outputs,
	})
}

func workflowToDomainSteps(workflow *workflows.WorkflowDefinition) ([]domain.StepDefinition, error) {
	steps := make([]domain.StepDefinition, 0, len(workflow.Steps))
	for _, step := range workflow.Steps {
		stepName, err := domain.NewStepName(step.Name)
		if err != nil {
			return nil, err
		}

		providerName, err := domain.NewProviderName(step.Provider)
		if err != nil {
			return nil, err
		}

		steps = append(steps, domain.StepDefinition{
			Name:     stepName,
			Provider: providerName,
			Input:    step.Input,
		})
	}

	return steps, nil
}

func mustWorkflowName(name string) domain.WorkflowName {
	value, err := domain.NewWorkflowName(name)
	if err != nil {
		panic(err)
	}
	return value
}

func indexOfStep(workflow *workflows.WorkflowDefinition, stepName string) int {
	for idx, step := range workflow.Steps {
		if step.Name == stepName {
			return idx
		}
	}
	return -1
}

type executePlan struct {
	Summary  string              `json:"summary"`
	Commands []executeCommand    `json:"commands"`
	Files    []executeFileChange `json:"files,omitempty"`
}

type executeCommand struct {
	Executable string   `json:"executable"`
	Args       []string `json:"args"`
}

type executeFileChange struct {
	Path    string `json:"path"`
	Content string `json:"content"`
}

type stepArtifactRefs struct {
	rawOutput        string
	approvedMarkdown string
	structuredJSON   string
	validationReport string
}

func artifactRefsForStep(stepName string) stepArtifactRefs {
	base := filepath.Join("artifacts", stepName)
	return stepArtifactRefs{
		rawOutput:        filepath.Join(base, "raw.md"),
		approvedMarkdown: filepath.Join(base, "approved.md"),
		structuredJSON:   filepath.Join(base, "structured.json"),
		validationReport: filepath.Join(base, "validation.json"),
	}
}

func (e *DefaultEngine) executePlan(ctx context.Context, run *domain.Run, step *domain.StepExecution) error {
	artifact, err := e.store.LoadArtifact(ctx, run.ID(), step.Name().String())
	if err != nil {
		return err
	}
	if len(artifact.StructuredJSON) == 0 {
		return nil
	}

	var plan executePlan
	if err := json.Unmarshal(artifact.StructuredJSON, &plan); err != nil {
		return fmt.Errorf("decoding execute plan: %w", err)
	}

	for _, file := range plan.Files {
		targetPath, err := normalizeExecutePath(file.Path)
		if err != nil {
			return err
		}
		if err := e.fs.WriteFile(targetPath, []byte(file.Content), 0o644); err != nil {
			return fmt.Errorf("writing execute file %q: %w", targetPath, err)
		}
		e.logStepEvent(ctx, run, step, "execute_file_written", 0, nil, map[string]any{"path": targetPath})
	}

	for _, command := range plan.Commands {
		if err := validateExecuteCommand(command); err != nil {
			return err
		}

		result, err := e.runner.Run(ctx, command.Executable, command.Args, "")
		e.logStepEvent(ctx, run, step, "execute_command_finished", result.Duration, err, map[string]any{
			"executable": command.Executable,
			"args":       strings.Join(command.Args, " "),
		})
		if err != nil {
			return fmt.Errorf("executing approved command %q: %w", command.Executable, err)
		}
	}

	return nil
}

func normalizeExecutePath(path string) (string, error) {
	cleaned := filepath.Clean(path)
	if filepath.IsAbs(cleaned) {
		return "", fmt.Errorf("execute file path must be relative: %q", path)
	}
	if strings.HasPrefix(cleaned, "..") {
		return "", fmt.Errorf("execute file path escapes project root: %q", path)
	}
	return cleaned, nil
}

func validateExecuteCommand(command executeCommand) error {
	if command.Executable == "" {
		return fmt.Errorf("execute command executable must not be empty")
	}

	if command.Executable == "git" && len(command.Args) > 0 {
		switch command.Args[0] {
		case "commit", "push":
			return fmt.Errorf("git operation %q is forbidden", command.Args[0])
		}
	}

	if command.Executable == "gh" && len(command.Args) >= 2 && command.Args[0] == "pr" && command.Args[1] == "create" {
		return fmt.Errorf("gh pr create is forbidden")
	}

	return nil
}
