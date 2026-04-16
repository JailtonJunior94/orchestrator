package bootstrap

import (
	"fmt"
	"io"
	"log/slog"
	"os"

	"github.com/jailtonjunior/orchestrator/internal/hitl"
	installapp "github.com/jailtonjunior/orchestrator/internal/install/application"
	installcatalog "github.com/jailtonjunior/orchestrator/internal/install/catalog"
	install "github.com/jailtonjunior/orchestrator/internal/install/domain"
	installinventory "github.com/jailtonjunior/orchestrator/internal/install/inventory"
	installproviders "github.com/jailtonjunior/orchestrator/internal/install/providers"
	"github.com/jailtonjunior/orchestrator/internal/output"
	"github.com/jailtonjunior/orchestrator/internal/platform"
	"github.com/jailtonjunior/orchestrator/internal/providers"
	"github.com/jailtonjunior/orchestrator/internal/runtime"
	runtimeapp "github.com/jailtonjunior/orchestrator/internal/runtime/application"
	"github.com/jailtonjunior/orchestrator/internal/state"
	"github.com/jailtonjunior/orchestrator/internal/tui"
	"github.com/jailtonjunior/orchestrator/internal/workflows"
)

// App bundles the wired application services used by the CLI.
type App struct {
	Runtime runtimeapp.Service
	Install installapp.Service
	// TUIWiring is non-nil when the App was bootstrapped in TUI mode.
	// run.go uses this to determine whether to start the Bubbletea program.
	TUIWiring *tui.Wiring
}

// New wires the production dependencies.
// If prompter is nil, a terminal prompter backed by stdin/stdout is created.
func New(stdin io.Reader, stdout io.Writer, progress runtime.ProgressReporter, prompter hitl.Prompter) (*App, error) {
	return NewWithLoggerOutput(stdin, stdout, progress, prompter, os.Stderr)
}

// NewWithLoggerOutput wires the production dependencies and allows the caller
// to redirect operational logs away from the interactive terminal when needed.
func NewWithLoggerOutput(stdin io.Reader, stdout io.Writer, progress runtime.ProgressReporter, prompter hitl.Prompter, logOutput io.Writer) (*App, error) {
	commandRunner := platform.NewCommandRunner()
	editor := platform.NewEditor()
	clock := platform.NewClock()
	fileSystem := platform.NewFileSystem()
	dirResolver := platform.NewDirResolver()
	parser := workflows.NewParser()
	catalog := workflows.NewCatalog(parser)
	validator := workflows.NewValidator([]string{
		providers.ClaudeProviderName,
		providers.CopilotProviderName,
		providers.GeminiProviderName,
		providers.CodexProviderName,
	})
	resolver := workflows.NewTemplateResolver()
	providerFactory := providers.NewFactory(commandRunner)
	processor := output.NewProcessor()
	store := state.NewFileStore(".", fileSystem)
	if prompter == nil {
		prompter = hitl.NewTerminalPrompter(stdin, stdout, editor)
	}
	logger := newLogger(logOutput)

	engine := runtime.NewEngine(runtime.Dependencies{
		Catalog:    catalog,
		Validator:  validator,
		Resolver:   resolver,
		Providers:  providerFactory,
		Processor:  processor,
		Store:      store,
		Prompter:   prompter,
		Clock:      clock,
		FileSystem: fileSystem,
		Runner:     commandRunner,
		Logger:     logger,
		Progress:   progress,
	})

	projectRoot, err := dirResolver.ResolveProjectRoot(".")
	if err != nil {
		return nil, fmt.Errorf("resolve project root: %w", err)
	}
	homeRoot, err := dirResolver.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("resolve user home directory: %w", err)
	}
	projectInventoryPath, err := installinventory.ResolvePath(dirResolver, install.ScopeProject, projectRoot)
	if err != nil {
		return nil, fmt.Errorf("resolve project install inventory path: %w", err)
	}
	globalInventoryPath, err := installinventory.ResolvePath(dirResolver, install.ScopeGlobal, projectRoot)
	if err != nil {
		return nil, fmt.Errorf("resolve global install inventory path: %w", err)
	}

	installRegistry, err := installproviders.NewRegistry(
		installproviders.NewClaudeAdapter(fileSystem, projectRoot, homeRoot),
		installproviders.NewGeminiAdapter(fileSystem, commandRunner, projectRoot, homeRoot),
		installproviders.NewCodexAdapter(fileSystem, projectRoot, homeRoot),
		installproviders.NewCopilotAdapter(fileSystem, projectRoot, homeRoot),
	)
	if err != nil {
		return nil, fmt.Errorf("build install provider registry: %w", err)
	}

	return &App{
		Runtime: runtimeapp.NewService(engine, catalog),
		Install: installapp.NewService(
			projectRoot,
			projectInventoryPath,
			globalInventoryPath,
			installcatalog.New(),
			installapp.NewPlanner(fileSystem, installapp.NewRegistryTargetResolver(installRegistry), nil),
			installinventory.NewStore(fileSystem, clock),
			installRegistry,
			fileSystem,
			clock,
			logger,
		),
	}, nil
}

func newLogger(output io.Writer) *slog.Logger {
	if output == nil {
		output = io.Discard
	}

	return slog.New(slog.NewTextHandler(output, nil))
}
