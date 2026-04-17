package providers

import (
	"fmt"
	"log/slog"

	"github.com/jailtonjunior/orchestrator/internal/acp"
)

// ACPProviderFactory resolves ACPProvider instances by workflow name.
// It is the contract the engine uses to obtain ACP-backed providers.
type ACPProviderFactory interface {
	Get(name string) (ACPProvider, error)
}

// ACPFactory is a provider factory backed by the ACP registry.
type ACPFactory struct {
	registry *acp.Registry
	logger   *slog.Logger
	connOpts []acp.ConnectionOption
}

// NewACPFactory creates an ACPFactory using the given registry.
func NewACPFactory(registry *acp.Registry, logger *slog.Logger, connOpts ...acp.ConnectionOption) *ACPFactory {
	return &ACPFactory{
		registry: registry,
		logger:   logger,
		connOpts: append([]acp.ConnectionOption{}, connOpts...),
	}
}

// Get resolves an ACPProvider by provider name.
func (f *ACPFactory) Get(name string) (ACPProvider, error) {
	spec, err := f.registry.Get(name)
	if err != nil {
		return nil, fmt.Errorf("acp factory: %w", err)
	}
	return NewACPProvider(spec, f.registry, f.logger, f.connOpts...), nil
}

// NewACPProviderFromSpec creates an ACPProvider directly from a spec and registry.
func NewACPProviderFromSpec(spec acp.AgentSpec, registry *acp.Registry, logger *slog.Logger, connOpts ...acp.ConnectionOption) ACPProvider {
	return NewACPProvider(spec, registry, logger, connOpts...)
}
