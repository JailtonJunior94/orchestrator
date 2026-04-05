package providers

import (
	"fmt"

	"github.com/jailtonjunior/orchestrator/internal/platform"
)

type providerFactory struct {
	providers map[string]Provider
}

// NewFactory creates a provider factory with the built-in V1 providers.
func NewFactory(runner platform.CommandRunner) Factory {
	return providerFactory{
		providers: map[string]Provider{
			ClaudeProviderName:  NewClaudeProvider(runner),
			CopilotProviderName: NewCopilotProvider(runner),
			GeminiProviderName:  NewGeminiProvider(runner),
			CodexProviderName:   NewCodexProvider(runner),
		},
	}
}

// Get resolves a provider by name.
func (f providerFactory) Get(name string) (Provider, error) {
	provider, ok := f.providers[name]
	if !ok {
		return nil, fmt.Errorf("unknown provider: %q", name)
	}
	return provider, nil
}
