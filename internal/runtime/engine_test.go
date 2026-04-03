package runtime

import (
	"context"
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

	provider := engine.providers.(fakeFactory).provider.(*fakeProvider)
	if provider.inputs[0].Timeout != 45*time.Second {
		t.Fatalf("timeout = %s", provider.inputs[0].Timeout)
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
	provider providers.Provider
}

func (f fakeFactory) Get(_ string) (providers.Provider, error) {
	return f.provider, nil
}

type fakeProvider struct {
	responses []providerResponse
	index     int
	inputs    []providers.ProviderInput
}

func (p *fakeProvider) Name() string { return providers.ClaudeProviderName }
func (p *fakeProvider) Available() error {
	return nil
}
func (p *fakeProvider) Execute(_ context.Context, input providers.ProviderInput) (providers.ProviderOutput, error) {
	p.inputs = append(p.inputs, input)
	if p.index >= len(p.responses) {
		return providers.ProviderOutput{}, io.EOF
	}
	response := p.responses[p.index]
	p.index++
	return providers.ProviderOutput{
		Stdout:   response.stdout,
		ExitCode: 0,
		Duration: 10 * time.Millisecond,
	}, response.err
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
		Catalog:    fakeCatalog{workflow: loaded},
		Validator:  workflows.NewValidator([]string{providers.ClaudeProviderName, providers.CopilotProviderName}),
		Resolver:   workflows.NewTemplateResolver(),
		Providers:  fakeFactory{provider: &fakeProvider{responses: responses}},
		Processor:  output.NewProcessor(),
		Store:      state.NewFileStore(storeDir, platform.NewFileSystem()),
		Prompter:   prompter,
		Clock:      platform.NewFakeClock(time.Date(2026, 4, 3, 12, 0, 0, 0, time.UTC)),
		FileSystem: fileSystem,
		Runner:     runner,
		Logger:     slog.New(slog.NewTextHandler(io.Discard, nil)),
	})
}
