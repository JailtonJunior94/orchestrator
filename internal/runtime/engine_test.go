package runtime

import (
	"bytes"
	"context"
	"errors"
	"io"
	"log/slog"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jailtonjunior/orchestrator/internal/acp"
	"github.com/jailtonjunior/orchestrator/internal/hitl"
	"github.com/jailtonjunior/orchestrator/internal/output"
	"github.com/jailtonjunior/orchestrator/internal/platform"
	"github.com/jailtonjunior/orchestrator/internal/providers"
	"github.com/jailtonjunior/orchestrator/internal/runtime/domain"
	"github.com/jailtonjunior/orchestrator/internal/state"
	"github.com/jailtonjunior/orchestrator/internal/workflows"
)

func TestEngineRunHappyPath(t *testing.T) {
	t.Parallel()

	engine := newTestEngine(t, []providerResponse{
		{stdout: "```json\n{\"doc\":\"prd\"}\n```"},
		{stdout: "```json\n{\"doc\":\"techspec\"}\n```"},
	}, hitl.NewFakePrompter(
		hitl.PromptResult{Action: hitl.ActionApprove},
		hitl.PromptResult{Action: hitl.ActionApprove},
	))

	result, err := engine.Run(context.Background(), "dev-workflow", "input")
	if err != nil {
		t.Fatal(err)
	}
	if result.Run.Status() != domain.RunCompleted {
		t.Fatalf("status = %s", result.Run.Status())
	}
}

func TestEnginePauseAndContinue(t *testing.T) {
	t.Parallel()

	engine := newTestEngine(t, []providerResponse{
		{stdout: "```json\n{\"doc\":\"prd\"}\n```"},
		{stdout: "```json\n{\"doc\":\"techspec\"}\n```"},
	}, hitl.NewFakePrompter(
		hitl.PromptResult{Action: hitl.ActionExit},
	))

	first, err := engine.Run(context.Background(), "dev-workflow", "input")
	if err != nil {
		t.Fatal(err)
	}
	if first.Run.Status() != domain.RunPaused {
		t.Fatalf("status = %s", first.Run.Status())
	}

	engine.prompter = hitl.NewFakePrompter(
		hitl.PromptResult{Action: hitl.ActionApprove},
		hitl.PromptResult{Action: hitl.ActionApprove},
		hitl.PromptResult{Action: hitl.ActionApprove},
		hitl.PromptResult{Action: hitl.ActionApprove},
	)

	second, err := engine.Continue(context.Background(), first.Run.ID())
	if err != nil {
		t.Fatal(err)
	}
	if second.Run.Status() != domain.RunCompleted {
		t.Fatalf("status = %s", second.Run.Status())
	}
}

func TestEngineRetriesInvalidJSONThenFallsBackToHITL(t *testing.T) {
	t.Parallel()

	engine := newTestEngine(t, []providerResponse{
		{stdout: "invalid"},
		{stdout: "still invalid"},
		{stdout: "again invalid"},
		{stdout: "```json\n{\"doc\":\"next\"}\n```"},
	}, hitl.NewFakePrompter(
		hitl.PromptResult{Action: hitl.ActionApprove},
		hitl.PromptResult{Action: hitl.ActionApprove},
		hitl.PromptResult{Action: hitl.ActionApprove},
		hitl.PromptResult{Action: hitl.ActionApprove},
	))

	result, err := engine.Run(context.Background(), "dev-workflow", "input")
	if err != nil {
		t.Fatal(err)
	}

	if result.Run.Status() != domain.RunCompleted {
		t.Fatalf("status = %s", result.Run.Status())
	}
	if result.Run.Steps()[0].Attempts() != 1 {
		t.Fatalf("attempts = %d", result.Run.Steps()[0].Attempts())
	}
}

func TestEngineRedoReexecutesStep(t *testing.T) {
	t.Parallel()

	responses := []providerResponse{
		{stdout: "```json\n{\"doc\":\"first\"}\n```"},
		{stdout: "```json\n{\"doc\":\"second\"}\n```"},
		{stdout: "```json\n{\"doc\":\"techspec\"}\n```"},
	}
	engine := newTestEngine(t, responses, hitl.NewFakePrompter(
		hitl.PromptResult{Action: hitl.ActionRedo},
		hitl.PromptResult{Action: hitl.ActionApprove},
		hitl.PromptResult{Action: hitl.ActionApprove},
	))

	result, err := engine.Run(context.Background(), "dev-workflow", "input")
	if err != nil {
		t.Fatal(err)
	}
	if result.Run.Steps()[0].Attempts() != 2 {
		t.Fatalf("attempts = %d", result.Run.Steps()[0].Attempts())
	}
}

func TestEngineProviderFailure(t *testing.T) {
	t.Parallel()

	engine := newTestEngine(t, []providerResponse{
		{err: errors.New("boom")},
	}, hitl.NewFakePrompter())

	_, err := engine.Run(context.Background(), "dev-workflow", "input")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestEnginePassesStepTimeoutAndSchema(t *testing.T) {
	t.Parallel()

	engine := newTestEngine(t, []providerResponse{
		{stdout: "```json\n{\"doc\":\"prd\"}\n```"},
	}, hitl.NewFakePrompter(
		hitl.PromptResult{Action: hitl.ActionApprove},
	))

	engine.catalog = fakeCatalog{workflow: &workflows.WorkflowDefinition{
		Name: "timed",
		Steps: []workflows.StepDefinition{{
			Name:     "prd",
			Provider: "claude",
			Input:    "do {{input}}",
			Timeout:  "45s",
			Schema:   `{"type":"object","required":["doc"]}`,
		}},
	}}

	result, err := engine.Run(context.Background(), "timed", "input")
	if err != nil {
		t.Fatal(err)
	}
	if result.Run.Status() != domain.RunCompleted {
		t.Fatalf("status = %s", result.Run.Status())
	}

	provider := engine.providers.(*fakeFactory).provider.(*fakeProvider)
	if provider.inputs[0].Timeout != 45*time.Second {
		t.Fatalf("timeout = %s", provider.inputs[0].Timeout)
	}
}

func TestEngineRunProcessesClaudeJSONEnvelope(t *testing.T) {
	t.Parallel()

	claudeProvider := &fakeProvider{
		responses: []providerResponse{
			// ACP returns plain content — no JSON envelope wrapping.
			{stdout: "PRD\n\n```json\n{\"doc\":\"prd\"}\n```"},
		},
	}
	engine := newTestEngineWithOptions(t, nil, hitl.NewFakePrompter(
		hitl.PromptResult{Action: hitl.ActionApprove},
	), nil, platform.FakeCommandRunner{}, t.TempDir(), false)
	engine.catalog = fakeCatalog{workflow: &workflows.WorkflowDefinition{
		Name: "dev-workflow",
		Steps: []workflows.StepDefinition{
			{
				Name:     "prd",
				Provider: providers.ClaudeProviderName,
				Input:    "generate {{input}}",
				Output: workflows.StepOutputDefinition{
					Markdown:   "required",
					JSONSchema: "prd/v1",
				},
				Schema: `{"type":"object","required":["doc"],"properties":{"doc":{"type":"string"}}}`,
			},
		},
	}}
	engine.providers = &fakeFactory{
		providerByName: map[string]providers.ACPProvider{
			providers.ClaudeProviderName: claudeProvider,
		},
	}

	result, err := engine.Run(context.Background(), "dev-workflow", "input")
	if err != nil {
		t.Fatalf("run error = %v", err)
	}
	if result.Run.Status() != domain.RunCompleted {
		t.Fatalf("status = %s", result.Run.Status())
	}

	steps := result.Run.Steps()
	if len(steps) != 1 {
		t.Fatalf("steps = %d", len(steps))
	}
	step := steps[0]
	if got := step.Output(); got != "PRD\n\n```json\n{\"doc\":\"prd\"}\n```" {
		t.Fatalf("output = %q", got)
	}
	if got := step.Result().ValidationStatus; got != domain.ValidationPassed {
		t.Fatalf("validation status = %q", got)
	}
}

func TestEngineExecutePlanWritesFilesAndAuditsCommands(t *testing.T) {
	t.Parallel()

	fileSystem, err := platform.NewFakeFileSystem()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = fileSystem.Close()
	})

	var executed [][]string
	runner := platform.FakeCommandRunner{
		RunFunc: func(_ context.Context, name string, args []string, stdin string) (platform.CommandResult, error) {
			executed = append(executed, append([]string{name}, args...))
			return platform.CommandResult{Duration: 5 * time.Millisecond}, nil
		},
	}

	storeDir := t.TempDir()
	engine := newTestEngineWithOptions(t, []providerResponse{
		{stdout: "PRD\n```json\n{\"doc\":\"prd\"}\n```"},
		{stdout: "TechSpec\n```json\n{\"doc\":\"techspec\"}\n```"},
		{stdout: "Tasks\n```json\n{\"doc\":\"tasks\"}\n```"},
		{stdout: "Execute\n```json\n{\"summary\":\"apply\",\"commands\":[{\"executable\":\"go\",\"args\":[\"test\",\"./...\"]}],\"files\":[{\"path\":\"notes.txt\",\"content\":\"done\"}]}\n```"},
	}, hitl.NewFakePrompter(
		hitl.PromptResult{Action: hitl.ActionApprove},
		hitl.PromptResult{Action: hitl.ActionApprove},
		hitl.PromptResult{Action: hitl.ActionApprove},
		hitl.PromptResult{Action: hitl.ActionApprove},
	), fileSystem, runner, storeDir, false)

	result, err := engine.Run(context.Background(), "dev-workflow", "input")
	if err != nil {
		t.Fatal(err)
	}
	if result.Run.Status() != domain.RunCompleted {
		t.Fatalf("status = %s", result.Run.Status())
	}

	data, err := fileSystem.ReadFile("notes.txt")
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "done" {
		t.Fatalf("file content = %q", string(data))
	}
	if len(executed) != 1 || executed[0][0] != "go" {
		t.Fatalf("executed = %#v", executed)
	}

	logData, err := platform.NewFileSystem().ReadFile(filepath.Join(storeDir, ".orq", "runs", result.Run.ID(), "logs", "run.log"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(logData), "execute_command_finished") {
		t.Fatalf("run log = %s", string(logData))
	}
}

func TestEngineExecutePlanRejectsForbiddenGitCommand(t *testing.T) {
	t.Parallel()

	engine := newTestEngineWithOptions(t, []providerResponse{
		{stdout: "PRD\n```json\n{\"doc\":\"prd\"}\n```"},
		{stdout: "TechSpec\n```json\n{\"doc\":\"techspec\"}\n```"},
		{stdout: "Tasks\n```json\n{\"doc\":\"tasks\"}\n```"},
		{stdout: "Execute\n```json\n{\"summary\":\"apply\",\"commands\":[{\"executable\":\"git\",\"args\":[\"commit\",\"-m\",\"x\"]}]}\n```"},
	}, hitl.NewFakePrompter(
		hitl.PromptResult{Action: hitl.ActionApprove},
		hitl.PromptResult{Action: hitl.ActionApprove},
		hitl.PromptResult{Action: hitl.ActionApprove},
		hitl.PromptResult{Action: hitl.ActionApprove},
	), nil, platform.FakeCommandRunner{}, t.TempDir(), false)

	_, err := engine.Run(context.Background(), "dev-workflow", "input")
	if err == nil || !strings.Contains(err.Error(), "forbidden") {
		t.Fatalf("error = %v", err)
	}
}

func TestEngineRunUsesWorkflowProviders(t *testing.T) {
	t.Parallel()

	engine := newTestEngineWithOptions(t, []providerResponse{
		{stdout: "PRD\n```json\n{\"doc\":\"prd\"}\n```"},
		{stdout: "TechSpec\n```json\n{\"doc\":\"techspec\"}\n```"},
		{stdout: "Tasks\n```json\n{\"doc\":\"tasks\"}\n```"},
		{stdout: "Execute\n```json\n{\"summary\":\"apply\",\"commands\":[]}\n```"},
	}, hitl.NewFakePrompter(
		hitl.PromptResult{Action: hitl.ActionApprove},
		hitl.PromptResult{Action: hitl.ActionApprove},
		hitl.PromptResult{Action: hitl.ActionApprove},
		hitl.PromptResult{Action: hitl.ActionApprove},
	), nil, platform.FakeCommandRunner{}, t.TempDir(), false)

	result, err := engine.Run(context.Background(), "dev-workflow", "input")
	if err != nil {
		factory := engine.providers.(*fakeFactory)
		provider := factory.provider.(*fakeProvider)
		t.Fatalf("run error = %v, requested=%v, calls=%d", err, factory.requested, len(provider.inputs))
	}
	if result.Run.Steps()[0].Provider().String() != providers.ClaudeProviderName {
		t.Fatalf("step prd provider = %q", result.Run.Steps()[0].Provider().String())
	}
	if result.Run.Steps()[1].Provider().String() != providers.ClaudeProviderName {
		t.Fatalf("step techspec provider = %q", result.Run.Steps()[1].Provider().String())
	}
	if result.Run.Steps()[2].Provider().String() != providers.ClaudeProviderName {
		t.Fatalf("step tasks provider = %q", result.Run.Steps()[2].Provider().String())
	}
	if result.Run.Steps()[3].Provider().String() != providers.CopilotProviderName {
		t.Fatalf("step execute provider = %q", result.Run.Steps()[3].Provider().String())
	}

	factory := engine.providers.(*fakeFactory)
	if !containsProviderRequest(factory.requested, providers.ClaudeProviderName) {
		t.Fatalf("missing claude lookup in %v", factory.requested)
	}
	if !containsProviderRequest(factory.requested, providers.CopilotProviderName) {
		t.Fatalf("missing copilot lookup in %v", factory.requested)
	}
}

func TestEngineContinuePreservesWorkflowProvider(t *testing.T) {
	t.Parallel()

	engine := newTestEngine(t, []providerResponse{
		{stdout: "PRD\n```json\n{\"doc\":\"prd\"}\n```"},
		{stdout: "TechSpec\n```json\n{\"doc\":\"techspec\"}\n```"},
		{stdout: "Tasks\n```json\n{\"doc\":\"tasks\"}\n```"},
		{stdout: "Execute\n```json\n{\"summary\":\"apply\",\"commands\":[]}\n```"},
	}, hitl.NewFakePrompter(
		hitl.PromptResult{Action: hitl.ActionExit},
	))

	first, err := engine.Run(context.Background(), "dev-workflow", "input")
	if err != nil {
		factory := engine.providers.(*fakeFactory)
		provider := factory.provider.(*fakeProvider)
		t.Fatalf("run error = %v, requested=%v, calls=%d", err, factory.requested, len(provider.inputs))
	}
	if first.Run.Status() != domain.RunPaused {
		t.Fatalf("status = %s", first.Run.Status())
	}

	engine.prompter = hitl.NewFakePrompter(
		hitl.PromptResult{Action: hitl.ActionApprove},
		hitl.PromptResult{Action: hitl.ActionApprove},
	)
	if _, err := engine.Continue(context.Background(), first.Run.ID()); err != nil {
		factory := engine.providers.(*fakeFactory)
		provider := factory.provider.(*fakeProvider)
		t.Fatalf("continue error = %v, requested=%v, calls=%d", err, factory.requested, len(provider.inputs))
	}

	factory := engine.providers.(*fakeFactory)
	if got := factory.requested[len(factory.requested)-1]; got != providers.ClaudeProviderName {
		t.Fatalf("continued provider lookup = %q", got)
	}
}

func TestEngineContinueIgnoresWaitingApprovalProviderAvailability(t *testing.T) {
	t.Parallel()

	claudeProvider := &fakeProvider{
		responses: []providerResponse{
			{stdout: "PRD\n```json\n{\"doc\":\"prd\"}\n```"},
		},
	}
	engine := newTestEngineWithOptions(t, nil, hitl.NewFakePrompter(
		hitl.PromptResult{Action: hitl.ActionExit},
	), nil, platform.FakeCommandRunner{}, t.TempDir(), false)
	engine.catalog = fakeCatalog{workflow: &workflows.WorkflowDefinition{
		Name: "dev-workflow",
		Steps: []workflows.StepDefinition{
			{
				Name:     "prd",
				Provider: providers.ClaudeProviderName,
				Input:    "generate {{input}}",
				Output: workflows.StepOutputDefinition{
					Markdown:   "required",
					JSONSchema: "prd/v1",
				},
				Schema: `{"type":"object","required":["doc"],"properties":{"doc":{"type":"string"}}}`,
			},
			{
				Name:     "techspec",
				Provider: providers.CopilotProviderName,
				Input:    "{{steps.prd.output}}",
				Output: workflows.StepOutputDefinition{
					Markdown:   "required",
					JSONSchema: "techspec/v1",
				},
				Schema: `{"type":"object","required":["doc"],"properties":{"doc":{"type":"string"}}}`,
			},
		},
	}}
	engine.providers = &fakeFactory{
		providerByName: map[string]providers.ACPProvider{
			providers.ClaudeProviderName:  claudeProvider,
			providers.CopilotProviderName: claudeProvider,
			providers.GeminiProviderName:  claudeProvider,
			providers.CodexProviderName:   claudeProvider,
		},
	}

	first, err := engine.Run(context.Background(), "dev-workflow", "input")
	if err != nil {
		t.Fatalf("run error = %v", err)
	}
	if first.Run.Status() != domain.RunPaused {
		t.Fatalf("status = %s", first.Run.Status())
	}

	claudeProvider.availableErr = errors.New("claude missing")
	copilotProvider := &fakeProvider{
		responses: []providerResponse{
			{stdout: "TechSpec\n```json\n{\"doc\":\"techspec\"}\n```"},
		},
	}
	engine.providers = &fakeFactory{
		providerByName: map[string]providers.ACPProvider{
			providers.ClaudeProviderName:  claudeProvider,
			providers.CopilotProviderName: copilotProvider,
			providers.GeminiProviderName:  copilotProvider,
			providers.CodexProviderName:   copilotProvider,
		},
	}
	engine.prompter = hitl.NewFakePrompter(
		hitl.PromptResult{Action: hitl.ActionApprove},
		hitl.PromptResult{Action: hitl.ActionApprove},
	)

	second, err := engine.Continue(context.Background(), first.Run.ID())
	if err != nil {
		t.Fatalf("continue error = %v", err)
	}
	if second.Run.Status() != domain.RunCompleted {
		t.Fatalf("status = %s", second.Run.Status())
	}
	if copilotProvider.availableCalls == 0 {
		t.Fatal("expected next provider availability check")
	}
}

func TestEngineContinueDoesNotPreflightFutureProvidersBeforePendingPrompt(t *testing.T) {
	t.Parallel()

	codexProvider := &fakeProvider{
		responses: []providerResponse{
			{stdout: "```json\n{\"doc\":\"tasks\"}\n```"},
		},
	}
	sharedProvider := &fakeProvider{
		responses: []providerResponse{
			{stdout: "```json\n{\"doc\":\"prd\"}\n```"},
			{stdout: "```json\n{\"doc\":\"techspec\"}\n```"},
		},
	}

	engine := newTestEngineWithOptions(t, nil, hitl.NewFakePrompter(
		hitl.PromptResult{Action: hitl.ActionExit},
	), nil, platform.FakeCommandRunner{}, t.TempDir(), false)
	engine.catalog = fakeCatalog{workflow: &workflows.WorkflowDefinition{
		Name: "dev-workflow",
		Steps: []workflows.StepDefinition{
			{
				Name:     "prd",
				Provider: providers.ClaudeProviderName,
				Input:    "generate {{input}}",
				Output: workflows.StepOutputDefinition{
					Markdown:   "required",
					JSONSchema: "prd/v1",
				},
				Schema: `{"type":"object","required":["doc"],"properties":{"doc":{"type":"string"}}}`,
			},
			{
				Name:     "techspec",
				Provider: providers.ClaudeProviderName,
				Input:    "{{steps.prd.output}}",
				Output: workflows.StepOutputDefinition{
					Markdown:   "required",
					JSONSchema: "techspec/v1",
				},
				Schema: `{"type":"object","required":["doc"],"properties":{"doc":{"type":"string"}}}`,
			},
			{
				Name:     "tasks",
				Provider: providers.CodexProviderName,
				Input:    "{{steps.techspec.output}}",
				Output: workflows.StepOutputDefinition{
					Markdown:   "required",
					JSONSchema: "tasks/v1",
				},
				Schema: `{"type":"object","required":["doc"],"properties":{"doc":{"type":"string"}}}`,
			},
		},
	}}
	engine.providers = &fakeFactory{
		providerByName: map[string]providers.ACPProvider{
			providers.CodexProviderName:  codexProvider,
			providers.ClaudeProviderName: sharedProvider,
		},
	}

	first, err := engine.Run(context.Background(), "dev-workflow", "input")
	if err != nil {
		t.Fatalf("run error = %v", err)
	}
	if first.Run.Status() != domain.RunPaused {
		t.Fatalf("status = %s", first.Run.Status())
	}

	codexProvider.availableErr = errors.New("codex missing")
	engine.prompter = hitl.NewFakePrompter(
		hitl.PromptResult{Action: hitl.ActionApprove},
		hitl.PromptResult{Action: hitl.ActionApprove},
	)

	second, err := engine.Continue(context.Background(), first.Run.ID())
	if err == nil || !strings.Contains(err.Error(), "codex missing") {
		t.Fatalf("continue error = %v", err)
	}
	if second.Run.Steps()[0].Status() != domain.StepApproved {
		t.Fatalf("step 0 status = %s", second.Run.Steps()[0].Status())
	}
	if sharedProvider.availableCalls == 0 {
		t.Fatal("expected claude availability check after approving pending step")
	}
	if codexProvider.availableCalls == 0 {
		t.Fatal("expected codex availability check during later execution")
	}
}

func TestEngineContinueRedoValidatesWaitingApprovalProviderAvailability(t *testing.T) {
	t.Parallel()

	claudeProvider := &fakeProvider{
		responses: []providerResponse{
			{stdout: "PRD\n```json\n{\"doc\":\"prd\"}\n```"},
		},
	}
	copilotProvider := &fakeProvider{
		responses: []providerResponse{
			{stdout: "TechSpec\n```json\n{\"doc\":\"techspec\"}\n```"},
		},
	}
	engine := newTestEngine(t, nil, hitl.NewFakePrompter(
		hitl.PromptResult{Action: hitl.ActionExit},
	))
	engine.catalog = fakeCatalog{workflow: &workflows.WorkflowDefinition{
		Name: "dev-workflow",
		Steps: []workflows.StepDefinition{
			{
				Name:     "prd",
				Provider: providers.ClaudeProviderName,
				Input:    "generate {{input}}",
				Output: workflows.StepOutputDefinition{
					Markdown:   "required",
					JSONSchema: "prd/v1",
				},
				Schema: `{"type":"object","required":["doc"],"properties":{"doc":{"type":"string"}}}`,
			},
			{
				Name:     "techspec",
				Provider: providers.CopilotProviderName,
				Input:    "{{steps.prd.output}}",
				Output: workflows.StepOutputDefinition{
					Markdown:   "required",
					JSONSchema: "techspec/v1",
				},
				Schema: `{"type":"object","required":["doc"],"properties":{"doc":{"type":"string"}}}`,
			},
		},
	}}
	engine.providers = &fakeFactory{
		providerByName: map[string]providers.ACPProvider{
			providers.ClaudeProviderName:  claudeProvider,
			providers.CopilotProviderName: copilotProvider,
			providers.GeminiProviderName:  copilotProvider,
			providers.CodexProviderName:   copilotProvider,
		},
	}

	first, err := engine.Run(context.Background(), "dev-workflow", "input")
	if err != nil {
		t.Fatalf("run error = %v", err)
	}
	if first.Run.Status() != domain.RunPaused {
		t.Fatalf("status = %s", first.Run.Status())
	}

	claudeProvider.availableErr = errors.New("claude missing")
	engine.prompter = hitl.NewFakePrompter(
		hitl.PromptResult{Action: hitl.ActionRedo},
	)

	_, err = engine.Continue(context.Background(), first.Run.ID())
	if err == nil || !strings.Contains(err.Error(), "claude missing") {
		t.Fatalf("error = %v", err)
	}
	if len(claudeProvider.inputs) != 1 {
		t.Fatalf("claude execute calls = %d", len(claudeProvider.inputs))
	}
}

// TestEngineRecoverableStructuredErrorFallsBackToHITL verifies that when a provider
// returns content with invalid JSON (unrecoverable after retries), the engine falls
// back to HITL with the raw content as-is.
func TestEngineRecoverableStructuredErrorFallsBackToHITL(t *testing.T) {
	t.Parallel()

	// All three attempts return invalid JSON — exhausts maxProviderRetries.
	engine := newTestEngine(t, []providerResponse{
		{stdout: "Summary\n```json\n{\"doc\":}\n```"},
		{stdout: "Summary\n```json\n{\"doc\":}\n```"},
		{stdout: "Summary\n```json\n{\"doc\":}\n```"},
	}, hitl.NewFakePrompter(
		hitl.PromptResult{Action: hitl.ActionExit},
	))
	engine.catalog = fakeCatalog{workflow: &workflows.WorkflowDefinition{
		Name: "dev-workflow",
		Steps: []workflows.StepDefinition{
			{
				Name:     "prd",
				Provider: providers.ClaudeProviderName,
				Input:    "generate {{input}}",
				Output: workflows.StepOutputDefinition{
					Markdown:   "required",
					JSONSchema: "prd/v1",
				},
				Schema: `{"type":"object","required":["doc"],"properties":{"doc":{"type":"string"}}}`,
			},
		},
	}}

	result, err := engine.Run(context.Background(), "dev-workflow", "input")
	if err != nil {
		t.Fatal(err)
	}

	step := result.Run.Steps()[0]
	want := "Summary\n```json\n{\"doc\":}\n```"
	if got := step.Output(); got != want {
		t.Fatalf("output = %q, want %q", got, want)
	}

	artifact, err := engine.store.LoadArtifact(context.Background(), result.Run.ID(), step.Name().String())
	if err != nil {
		t.Fatal(err)
	}
	if got := string(artifact.ApprovedMarkdown); got != want {
		t.Fatalf("approved markdown = %q, want %q", got, want)
	}
}

func TestEngineLogsProviderStartedBeforeExecute(t *testing.T) {
	t.Parallel()

	var logBuf bytes.Buffer
	provider := &fakeProvider{
		responses: []providerResponse{
			{stdout: "PRD\n```json\n{\"doc\":\"prd\"}\n```"},
			{stdout: "TechSpec\n```json\n{\"doc\":\"techspec\"}\n```"},
		},
	}

	engine := newTestEngine(t, nil, hitl.NewFakePrompter(
		hitl.PromptResult{Action: hitl.ActionApprove},
		hitl.PromptResult{Action: hitl.ActionApprove},
	))
	engine.providers = &fakeFactory{provider: provider}
	engine.logger = slog.New(slog.NewTextHandler(&logBuf, nil))
	provider.onExecute = func() {
		if !strings.Contains(logBuf.String(), "event=provider_execute_started") {
			t.Fatalf("provider_execute_started must be logged before Execute, logs=%s", logBuf.String())
		}
	}

	if _, err := engine.Run(context.Background(), "dev-workflow", "input"); err != nil {
		t.Fatal(err)
	}
}

type providerResponse struct {
	stdout string
	err    error
}

type fakeCatalog struct {
	workflow *workflows.WorkflowDefinition
}

func (f fakeCatalog) Load(_ context.Context, name string) (*workflows.WorkflowDefinition, error) {
	if name != f.workflow.Name {
		return nil, workflows.ErrWorkflowNotFound
	}
	return f.workflow, nil
}

func (f fakeCatalog) List() ([]string, error) {
	return []string{f.workflow.Name}, nil
}

type fakeFactory struct {
	provider       providers.ACPProvider
	providerByName map[string]providers.ACPProvider
	requested      []string
}

func (f *fakeFactory) Get(name string) (providers.ACPProvider, error) {
	f.requested = append(f.requested, name)
	if f.providerByName != nil {
		provider, ok := f.providerByName[name]
		if !ok {
			return nil, errors.New("provider not configured in fake factory")
		}
		return provider, nil
	}
	return f.provider, nil
}

type fakeProvider struct {
	responses      []providerResponse
	index          int
	inputs         []acp.ACPInput
	resumeInputs   []acp.ACPInput
	resumeResult   string
	resumeErr      error
	availableErr   error
	availableCalls int
	onExecute      func()
}

func (p *fakeProvider) Name() string { return providers.ClaudeProviderName }
func (p *fakeProvider) Close() error { return nil }
func (p *fakeProvider) Available() error {
	p.availableCalls++
	return p.availableErr
}

func (p *fakeProvider) Execute(_ context.Context, input acp.ACPInput) (acp.ACPOutput, error) {
	return p.ExecuteStream(context.Background(), input, nil)
}

func (p *fakeProvider) ResumeSession(_ context.Context, input acp.ACPInput) (string, error) {
	p.resumeInputs = append(p.resumeInputs, input)
	if p.resumeErr != nil {
		return "", p.resumeErr
	}
	if p.resumeResult != "" {
		return p.resumeResult, nil
	}
	return input.SessionID, nil
}

func (p *fakeProvider) ExecuteStream(_ context.Context, input acp.ACPInput, onUpdate func(acp.TypedUpdate)) (acp.ACPOutput, error) {
	if p.onExecute != nil {
		p.onExecute()
	}
	p.inputs = append(p.inputs, input)
	if p.index >= len(p.responses) {
		return acp.ACPOutput{}, io.EOF
	}
	response := p.responses[p.index]
	p.index++
	if onUpdate != nil && response.stdout != "" {
		onUpdate(acp.TypedUpdate{Kind: acp.UpdateMessage, Text: response.stdout})
	}
	return acp.ACPOutput{
		Content:   response.stdout,
		SessionID: "test-session-" + response.stdout[:min(8, len(response.stdout))],
		Duration:  10 * time.Millisecond,
	}, response.err
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func containsProviderRequest(requested []string, providerName string) bool {
	for _, requestedName := range requested {
		if requestedName == providerName {
			return true
		}
	}

	return false
}

func TestEngineRunPreflightsProvidersBeforeExecution(t *testing.T) {
	t.Parallel()

	provider := &fakeProvider{
		availableErr: errors.New("missing provider"),
		onExecute: func() {
			t.Fatal("execute must not run when provider preflight fails")
		},
	}
	engine := newTestEngine(t, nil, hitl.NewFakePrompter())
	engine.providers = &fakeFactory{provider: provider}

	_, err := engine.Run(context.Background(), "dev-workflow", "input")
	if err == nil || !strings.Contains(err.Error(), "missing provider") {
		t.Fatalf("error = %v", err)
	}
	if provider.availableCalls == 0 {
		t.Fatal("expected preflight availability check")
	}
	if len(provider.inputs) != 0 {
		t.Fatalf("provider executed unexpectedly: %d calls", len(provider.inputs))
	}
}

func newTestEngine(t *testing.T, responses []providerResponse, prompter hitl.Prompter) *DefaultEngine {
	t.Helper()

	return newTestEngineWithOptions(t, responses, prompter, nil, platform.FakeCommandRunner{}, t.TempDir(), true)
}

func newTestEngineWithOptions(t *testing.T, responses []providerResponse, prompter hitl.Prompter, fileSystem platform.FileSystem, runner platform.CommandRunner, storeDir string, truncateToTwoSteps bool) *DefaultEngine {
	t.Helper()

	parser := workflows.NewParser()
	catalog := workflows.NewCatalog(parser)
	loaded, err := catalog.Load(context.Background(), "dev-workflow")
	if err != nil {
		t.Fatal(err)
	}
	if truncateToTwoSteps {
		loaded.Steps = loaded.Steps[:2]
	}
	if fileSystem == nil {
		fileSystem = platform.NewFileSystem()
	}

	return NewEngine(Dependencies{
		Catalog: fakeCatalog{workflow: loaded},
		Validator: workflows.NewValidator([]string{
			providers.ClaudeProviderName,
			providers.CopilotProviderName,
			providers.GeminiProviderName,
			providers.CodexProviderName,
		}),
		Resolver:   workflows.NewTemplateResolver(),
		Providers:  &fakeFactory{provider: &fakeProvider{responses: responses}},
		Processor:  output.NewProcessor(),
		Store:      state.NewFileStore(storeDir, platform.NewFileSystem()),
		Prompter:   prompter,
		Clock:      platform.NewFakeClock(time.Date(2026, 4, 3, 12, 0, 0, 0, time.UTC)),
		FileSystem: fileSystem,
		Runner:     runner,
		Logger:     slog.New(slog.NewTextHandler(io.Discard, nil)),
	})
}

// --- Task 2.0 tests: ProgressReporter extended methods ---

type recordingProgressReporter struct {
	completed        []string
	failed           []string
	waitingApprovals []string
	typedUpdates     []string
}

func (r *recordingProgressReporter) StepStarted(ProgressStep)  {}
func (r *recordingProgressReporter) StepFinished(ProgressStep) {}
func (r *recordingProgressReporter) TypedUpdate(stepName string, _ acp.TypedUpdate) {
	r.typedUpdates = append(r.typedUpdates, stepName)
}
func (r *recordingProgressReporter) WaitingApproval(stepName string, _ string) {
	r.waitingApprovals = append(r.waitingApprovals, stepName)
}
func (r *recordingProgressReporter) RunCompleted(runID string, _ string) {
	r.completed = append(r.completed, runID)
}
func (r *recordingProgressReporter) RunFailed(runID string, _ error) {
	r.failed = append(r.failed, runID)
}

// Verify recordingProgressReporter implements ProgressReporter.
var _ ProgressReporter = (*recordingProgressReporter)(nil)

func TestEngineCallsRunCompletedOnHappyPath(t *testing.T) {
	t.Parallel()

	rec := &recordingProgressReporter{}
	engine := newTestEngine(t, []providerResponse{
		{stdout: "```json\n{\"doc\":\"prd\"}\n```"},
		{stdout: "```json\n{\"doc\":\"techspec\"}\n```"},
	}, hitl.NewFakePrompter(
		hitl.PromptResult{Action: hitl.ActionApprove},
		hitl.PromptResult{Action: hitl.ActionApprove},
	))
	engine.progress = rec

	result, err := engine.Run(context.Background(), "dev-workflow", "input")
	if err != nil {
		t.Fatal(err)
	}

	if len(rec.completed) != 1 || rec.completed[0] != result.Run.ID() {
		t.Errorf("RunCompleted calls = %v; want [%s]", rec.completed, result.Run.ID())
	}
	if len(rec.failed) != 0 {
		t.Errorf("unexpected RunFailed calls: %v", rec.failed)
	}
}

func TestEngineCallsRunFailedOnStepError(t *testing.T) {
	t.Parallel()

	rec := &recordingProgressReporter{}
	engine := newTestEngine(t, []providerResponse{
		{err: errors.New("provider boom")},
	}, hitl.NewFakePrompter())
	engine.progress = rec

	_, err := engine.Run(context.Background(), "dev-workflow", "input")
	if err == nil {
		t.Fatal("expected error")
	}

	if len(rec.failed) != 1 {
		t.Errorf("RunFailed calls = %v; want 1", rec.failed)
	}
	if len(rec.completed) != 0 {
		t.Errorf("unexpected RunCompleted calls: %v", rec.completed)
	}
}

func TestEngineCallsWaitingApprovalBeforeHITLPrompt(t *testing.T) {
	t.Parallel()

	rec := &recordingProgressReporter{}
	engine := newTestEngine(t, []providerResponse{
		{stdout: "```json\n{\"doc\":\"prd\"}\n```"},
		{stdout: "```json\n{\"doc\":\"techspec\"}\n```"},
	}, hitl.NewFakePrompter(
		hitl.PromptResult{Action: hitl.ActionApprove},
		hitl.PromptResult{Action: hitl.ActionApprove},
	))
	engine.progress = rec

	if _, err := engine.Run(context.Background(), "dev-workflow", "input"); err != nil {
		t.Fatal(err)
	}

	if len(rec.waitingApprovals) == 0 {
		t.Error("expected at least one WaitingApproval call")
	}
}

// --- Task 6.0 tests: TypedUpdate integration ---

func TestEngineEmitsTypedUpdateViaExecuteStream(t *testing.T) {
	t.Parallel()

	rec := &recordingProgressReporter{}
	engine := newTestEngine(t, []providerResponse{
		{stdout: "```json\n{\"doc\":\"prd\"}\n```"},
		{stdout: "```json\n{\"doc\":\"techspec\"}\n```"},
	}, hitl.NewFakePrompter(
		hitl.PromptResult{Action: hitl.ActionApprove},
		hitl.PromptResult{Action: hitl.ActionApprove},
	))
	engine.progress = rec

	if _, err := engine.Run(context.Background(), "dev-workflow", "input"); err != nil {
		t.Fatal(err)
	}

	if len(rec.typedUpdates) == 0 {
		t.Error("expected TypedUpdate to be called via ExecuteStream")
	}
}

func TestNoopProgressReporterImplementsFullInterface(t *testing.T) {
	t.Parallel()

	var reporter ProgressReporter = NoopProgressReporter{}
	reporter.StepStarted(ProgressStep{})
	reporter.StepFinished(ProgressStep{})
	reporter.TypedUpdate("step", acp.TypedUpdate{Kind: acp.UpdateMessage, Text: "data"})
	reporter.WaitingApproval("step", "output")
	reporter.RunCompleted("run-id", "completed")
	reporter.RunFailed("run-id", errors.New("err"))
	// All methods must complete without panic.
}

// TestEngineSessionIDPersistedAfterStep verifies that the engine saves the
// sessionID returned by the provider into the step and persists it to state.json.
func TestEngineSessionIDPersistedAfterStep(t *testing.T) {
	t.Parallel()

	storeDir := t.TempDir()
	engine := newTestEngineWithOptions(t, []providerResponse{
		{stdout: "```json\n{\"doc\":\"prd\"}\n```"},
		{stdout: "```json\n{\"doc\":\"techspec\"}\n```"},
	}, hitl.NewFakePrompter(
		hitl.PromptResult{Action: hitl.ActionApprove},
		hitl.PromptResult{Action: hitl.ActionApprove},
	), nil, platform.FakeCommandRunner{}, storeDir, true)

	result, err := engine.Run(context.Background(), "dev-workflow", "input")
	if err != nil {
		t.Fatal(err)
	}

	for _, step := range result.Run.Steps() {
		if step.SessionID() == "" {
			t.Errorf("step %q: sessionID not persisted", step.Name())
		}
	}
}

// TestEngineRetryResetsSessionID verifies that a retry operation clears the
// sessionID before re-executing the step, forcing a new ACP session.
func TestEngineRetryResetsSessionID(t *testing.T) {
	t.Parallel()

	provider := &fakeProvider{
		responses: []providerResponse{
			{stdout: "```json\n{\"doc\":\"prd\"}\n```"},
			{stdout: "```json\n{\"doc\":\"prd-retry\"}\n```"},
			{stdout: "```json\n{\"doc\":\"techspec\"}\n```"},
		},
	}
	engine := newTestEngine(t, nil, hitl.NewFakePrompter(
		hitl.PromptResult{Action: hitl.ActionRedo},
		hitl.PromptResult{Action: hitl.ActionApprove},
		hitl.PromptResult{Action: hitl.ActionApprove},
	))
	engine.providers = &fakeFactory{provider: provider}

	result, err := engine.Run(context.Background(), "dev-workflow", "input")
	if err != nil {
		t.Fatal(err)
	}
	if result.Run.Status() != domain.RunCompleted {
		t.Fatalf("status = %s", result.Run.Status())
	}

	// The first step was retried; verify the second call had an empty sessionID
	// (cleared by retry logic) before the provider assigned a new one.
	if len(provider.inputs) < 2 {
		t.Fatalf("expected at least 2 provider calls, got %d", len(provider.inputs))
	}
	if provider.inputs[1].SessionID != "" {
		t.Errorf("retry call should have empty sessionID, got %q", provider.inputs[1].SessionID)
	}
}

// TestEngineContinueUsesExistingSessionID verifies that when continuing a paused
// run, the engine passes the stored sessionID to the provider via ACPInput.
func TestEngineContinueUsesExistingSessionID(t *testing.T) {
	t.Parallel()

	storeDir := t.TempDir()
	provider := &fakeProvider{
		responses: []providerResponse{
			{stdout: "```json\n{\"doc\":\"prd\"}\n```"},
			{stdout: "```json\n{\"doc\":\"techspec\"}\n```"},
		},
	}
	engine := newTestEngineWithOptions(t, nil, hitl.NewFakePrompter(
		hitl.PromptResult{Action: hitl.ActionExit},
	), nil, platform.FakeCommandRunner{}, storeDir, true)
	engine.providers = &fakeFactory{provider: provider}

	first, err := engine.Run(context.Background(), "dev-workflow", "input")
	if err != nil {
		t.Fatal(err)
	}
	if first.Run.Status() != domain.RunPaused {
		t.Fatalf("expected paused, got %s", first.Run.Status())
	}

	// sessionID must be set on the completed step after first run.
	prdStep := first.Run.Steps()[0]
	if prdStep.SessionID() == "" {
		t.Fatal("expected sessionID on completed prd step")
	}
	savedSessionID := prdStep.SessionID()

	// Continue: second step should be executed; reload from store verifies sessionID
	// was persisted. The first step is already approved so only step 2 executes.
	engine.prompter = hitl.NewFakePrompter(
		hitl.PromptResult{Action: hitl.ActionApprove}, // approve step1 pending prompt
		hitl.PromptResult{Action: hitl.ActionApprove}, // approve step2 after execution
	)
	second, err := engine.Continue(context.Background(), first.Run.ID())
	if err != nil {
		t.Fatal(err)
	}
	if second.Run.Status() != domain.RunCompleted {
		t.Fatalf("expected completed, got %s", second.Run.Status())
	}

	// Verify the persisted sessionID on step 1 survived the reload.
	if got := second.Run.Steps()[0].SessionID(); got != savedSessionID {
		t.Errorf("sessionID after continue = %q, want %q", got, savedSessionID)
	}
	if len(provider.inputs) == 0 {
		t.Fatal("expected provider inputs to be recorded")
	}
	for i, input := range provider.inputs {
		if input.WorkDir == "" {
			t.Fatalf("provider input %d has empty WorkDir", i)
		}
	}
	if len(provider.resumeInputs) != 1 {
		t.Fatalf("resume calls = %d, want 1", len(provider.resumeInputs))
	}
	if got := provider.resumeInputs[0].SessionID; got != savedSessionID {
		t.Fatalf("resume sessionID = %q, want %q", got, savedSessionID)
	}
}

func TestEngineContinueUpdatesSessionIDWhenResumeFallsBackToNewSession(t *testing.T) {
	t.Parallel()

	storeDir := t.TempDir()
	provider := &fakeProvider{
		responses: []providerResponse{
			{stdout: "```json\n{\"doc\":\"prd\"}\n```"},
			{stdout: "```json\n{\"doc\":\"techspec\"}\n```"},
		},
		resumeResult: "replacement-session",
	}
	engine := newTestEngineWithOptions(t, nil, hitl.NewFakePrompter(
		hitl.PromptResult{Action: hitl.ActionExit},
	), nil, platform.FakeCommandRunner{}, storeDir, true)
	engine.providers = &fakeFactory{provider: provider}

	first, err := engine.Run(context.Background(), "dev-workflow", "input")
	if err != nil {
		t.Fatal(err)
	}

	engine.prompter = hitl.NewFakePrompter(
		hitl.PromptResult{Action: hitl.ActionApprove},
		hitl.PromptResult{Action: hitl.ActionApprove},
	)
	second, err := engine.Continue(context.Background(), first.Run.ID())
	if err != nil {
		t.Fatal(err)
	}

	if got := second.Run.Steps()[0].SessionID(); got != "replacement-session" {
		t.Fatalf("sessionID after resume fallback = %q, want %q", got, "replacement-session")
	}
}

func TestPermissionPolicyForStep_UsesWorkflowCapability(t *testing.T) {
	t.Parallel()

	policy := permissionPolicyForStep("security-review", &workflows.StepDefinition{
		Name:     "implement",
		Provider: providers.ClaudeProviderName,
		Input:    "do it",
		Capabilities: map[string]string{
			"permission_policy": "deny",
		},
	})

	if got := policy.WorkflowDecisions["security-review"]; got != hitl.PermissionDeny {
		t.Fatalf("workflow decision = %q, want %q", got, hitl.PermissionDeny)
	}
}
