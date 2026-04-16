package runtime

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"path/filepath"
	"strings"
	"testing"
	"time"

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

func TestEnginePassesStructuredProviderOptions(t *testing.T) {
	t.Parallel()

	claudeOpts, claudeProcess := buildProviderOptions(providers.ClaudeProviderName, true, "prd/v1")
	if claudeOpts != nil {
		t.Fatalf("claude options = %v, want nil", claudeOpts)
	}
	if claudeProcess.OutputFormat != "json" {
		t.Fatalf("claude process output format = %q", claudeProcess.OutputFormat)
	}

	geminiOpts, geminiProcess := buildProviderOptions(providers.GeminiProviderName, true, "prd/v1")
	if got := geminiOpts["output_format"]; got != "json" {
		t.Fatalf("gemini output_format = %q", got)
	}
	if geminiProcess.OutputFormat != "json" {
		t.Fatalf("gemini process output format = %q", geminiProcess.OutputFormat)
	}

	codexOpts, codexProcess := buildProviderOptions(providers.CodexProviderName, true, "tasks/v1")
	if got := codexOpts["output_format"]; got != "jsonl" {
		t.Fatalf("codex output_format = %q", got)
	}
	if got := codexOpts["sandbox"]; got != "read-only" {
		t.Fatalf("codex sandbox = %q", got)
	}
	if codexProcess.OutputFormat != "jsonl" {
		t.Fatalf("codex process output format = %q", codexProcess.OutputFormat)
	}
}

func TestEngineRunProcessesClaudeJSONEnvelope(t *testing.T) {
	t.Parallel()

	claudeProvider := &fakeProvider{
		responses: []providerResponse{
			{
				stdout: "{\"type\":\"result\",\"subtype\":\"success\",\"is_error\":false,\"result\":\"PRD\\n\\n```json\\n{\\\"doc\\\":\\\"prd\\\"}\\n```\"}",
			},
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
		providerByName: map[string]providers.Provider{
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
		providerByName: map[string]providers.Provider{
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
		providerByName: map[string]providers.Provider{
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
		providerByName: map[string]providers.Provider{
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
		providerByName: map[string]providers.Provider{
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

func TestEngineRecoverableStructuredErrorUsesExtractedGeminiMarkdown(t *testing.T) {
	t.Parallel()

	engine := newTestEngine(t, []providerResponse{
		{stdout: "{\"response\":\"Summary\\n```json\\n{\\\"doc\\\":}\\n```\",\"stats\":{\"session\":{\"duration\":1}},\"error\":null}"},
		{stdout: "{\"response\":\"Summary\\n```json\\n{\\\"doc\\\":}\\n```\",\"stats\":{\"session\":{\"duration\":1}},\"error\":null}"},
		{stdout: "{\"response\":\"Summary\\n```json\\n{\\\"doc\\\":}\\n```\",\"stats\":{\"session\":{\"duration\":1}},\"error\":null}"},
	}, hitl.NewFakePrompter(
		hitl.PromptResult{Action: hitl.ActionExit},
	))
	engine.catalog = fakeCatalog{workflow: &workflows.WorkflowDefinition{
		Name: "dev-workflow",
		Steps: []workflows.StepDefinition{
			{
				Name:     "prd",
				Provider: providers.GeminiProviderName,
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
	if got, want := step.Output(), "Summary\n```json\n{\"doc\":}\n```"; got != want {
		t.Fatalf("output = %q, want %q", got, want)
	}

	artifact, err := engine.store.LoadArtifact(context.Background(), result.Run.ID(), step.Name().String())
	if err != nil {
		t.Fatal(err)
	}
	if got, want := string(artifact.ApprovedMarkdown), "Summary\n```json\n{\"doc\":}\n```"; got != want {
		t.Fatalf("approved markdown = %q, want %q", got, want)
	}
}

func TestEngineRecoverableStructuredErrorUsesExtractedCodexMarkdown(t *testing.T) {
	t.Parallel()

	engine := newTestEngineWithOptions(t, []providerResponse{
		{stdout: "{\"response\":\"PRD\\n```json\\n{\\\"doc\\\":\\\"prd\\\"}\\n```\",\"stats\":{\"session\":{\"duration\":1}},\"error\":null}"},
		{stdout: "```json\n{\"doc\":\"techspec\"}\n```"},
		{stdout: "{\"type\":\"message\",\"content\":\"Tasks\\n```json\\n{\\\"doc\\\":}\\n```\"}\n"},
		{stdout: "{\"type\":\"message\",\"content\":\"Tasks\\n```json\\n{\\\"doc\\\":}\\n```\"}\n"},
		{stdout: "{\"type\":\"message\",\"content\":\"Tasks\\n```json\\n{\\\"doc\\\":}\\n```\"}\n"},
	}, hitl.NewFakePrompter(
		hitl.PromptResult{Action: hitl.ActionApprove},
		hitl.PromptResult{Action: hitl.ActionApprove},
		hitl.PromptResult{Action: hitl.ActionExit},
	), nil, platform.FakeCommandRunner{}, t.TempDir(), false)
	engine.catalog = fakeCatalog{workflow: &workflows.WorkflowDefinition{
		Name: "dev-workflow",
		Steps: []workflows.StepDefinition{
			{
				Name:     "prd",
				Provider: providers.GeminiProviderName,
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

	result, err := engine.Run(context.Background(), "dev-workflow", "input")
	if err != nil {
		t.Fatal(err)
	}

	step := result.Run.Steps()[2]
	if got, want := step.Output(), "Tasks\n```json\n{\"doc\":}\n```"; got != want {
		t.Fatalf("output = %q, want %q", got, want)
	}

	artifact, err := engine.store.LoadArtifact(context.Background(), result.Run.ID(), step.Name().String())
	if err != nil {
		t.Fatal(err)
	}
	if got, want := string(artifact.ApprovedMarkdown), "Tasks\n```json\n{\"doc\":}\n```"; got != want {
		t.Fatalf("approved markdown = %q, want %q", got, want)
	}
}

func TestEngineLogsProviderProfile(t *testing.T) {
	t.Parallel()

	var logBuf bytes.Buffer
	engine := newTestEngine(t, []providerResponse{
		{stdout: "PRD\n```json\n{\"doc\":\"prd\"}\n```", profile: "exec-yolo-stdin"},
		{stdout: "TechSpec\n```json\n{\"doc\":\"techspec\"}\n```", profile: "exec-yolo-stdin"},
	}, hitl.NewFakePrompter(
		hitl.PromptResult{Action: hitl.ActionApprove},
		hitl.PromptResult{Action: hitl.ActionApprove},
	))
	engine.logger = slog.New(slog.NewTextHandler(&logBuf, nil))

	if _, err := engine.Run(context.Background(), "dev-workflow", "input"); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(logBuf.String(), "profile=exec-yolo-stdin") {
		t.Fatalf("log output = %s", logBuf.String())
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
		profile: "exec-yolo-stdin",
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
		if !strings.Contains(logBuf.String(), "profile=exec-yolo-stdin") {
			t.Fatalf("provider_execute_started must log profile, logs=%s", logBuf.String())
		}
	}

	if _, err := engine.Run(context.Background(), "dev-workflow", "input"); err != nil {
		t.Fatal(err)
	}
}

type providerResponse struct {
	stdout  string
	err     error
	profile string
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
	provider       providers.Provider
	providerByName map[string]providers.Provider
	requested      []string
}

func (f *fakeFactory) Get(name string) (providers.Provider, error) {
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
	inputs         []providers.ProviderInput
	profile        string
	availableErr   error
	availableCalls int
	onExecute      func()
}

func (p *fakeProvider) Name() string { return providers.ClaudeProviderName }
func (p *fakeProvider) Available() error {
	p.availableCalls++
	return p.availableErr
}

func (p *fakeProvider) ResolveProfile(context.Context) (string, error) {
	return firstNonEmpty(p.profile, "prompt-arg-json"), nil
}

func (p *fakeProvider) Execute(_ context.Context, input providers.ProviderInput) (providers.ProviderOutput, error) {
	if p.onExecute != nil {
		p.onExecute()
	}
	p.inputs = append(p.inputs, input)
	if p.index >= len(p.responses) {
		return providers.ProviderOutput{}, io.EOF
	}
	response := p.responses[p.index]
	p.index++
	stdout := normalizeProviderStdout(response.stdout, input.Options)
	return providers.ProviderOutput{
		Stdout:   stdout,
		ExitCode: 0,
		Duration: 10 * time.Millisecond,
		Profile:  firstNonEmpty(response.profile, p.profile, "prompt-arg-json"),
	}, response.err
}

func (p *fakeProvider) ExecuteStream(ctx context.Context, input providers.ProviderInput, onChunk func([]byte)) (providers.ProviderOutput, error) {
	out, err := p.Execute(ctx, input)
	if onChunk != nil && out.Stdout != "" {
		onChunk([]byte(out.Stdout))
	}
	return out, err
}

func normalizeProviderStdout(stdout string, options map[string]string) string {
	switch options["output_format"] {
	case "json":
		if looksLikeProviderJSON(stdout) {
			return stdout
		}
		payload, _ := json.Marshal(map[string]any{
			"response": stdout,
			"error":    nil,
		})
		return string(payload)
	case "jsonl":
		if looksLikeJSONL(stdout) {
			return stdout
		}
		payload, _ := json.Marshal(map[string]any{
			"type":    "message",
			"content": stdout,
		})
		return string(payload) + "\n"
	default:
		return stdout
	}
}

func looksLikeProviderJSON(stdout string) bool {
	trimmed := strings.TrimSpace(stdout)
	return strings.HasPrefix(trimmed, "{") && strings.Contains(trimmed, `"response"`)
}

func looksLikeJSONL(stdout string) bool {
	trimmed := strings.TrimSpace(stdout)
	return strings.HasPrefix(trimmed, "{") && strings.Contains(trimmed, `"type"`)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}

	return ""
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
	outputChunks     []string
}

func (r *recordingProgressReporter) StepStarted(ProgressStep)  {}
func (r *recordingProgressReporter) StepFinished(ProgressStep) {}
func (r *recordingProgressReporter) OutputChunk(stepName string, _ []byte) {
	r.outputChunks = append(r.outputChunks, stepName)
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

// --- Task 4.0 tests: ExecuteStream integration ---

func TestEngineEmitsOutputChunkViaExecuteStream(t *testing.T) {
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

	if len(rec.outputChunks) == 0 {
		t.Error("expected OutputChunk to be called via ExecuteStream")
	}
}

func TestNoopProgressReporterImplementsFullInterface(t *testing.T) {
	t.Parallel()

	var reporter ProgressReporter = NoopProgressReporter{}
	reporter.StepStarted(ProgressStep{})
	reporter.StepFinished(ProgressStep{})
	reporter.OutputChunk("step", []byte("data"))
	reporter.WaitingApproval("step", "output")
	reporter.RunCompleted("run-id", "completed")
	reporter.RunFailed("run-id", errors.New("err"))
	// All methods must complete without panic.
}
