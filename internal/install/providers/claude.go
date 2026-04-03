package providers

import (
	"context"

	install "github.com/jailtonjunior/orchestrator/internal/install/domain"
	"github.com/jailtonjunior/orchestrator/internal/platform"
)

// ClaudeAdapter manages install targets for Claude Code.
type ClaudeAdapter struct {
	*managedAdapter
}

// NewClaudeAdapter creates a Claude target adapter.
func NewClaudeAdapter(fileSystem platform.FileSystem, projectRoot string, globalRoot string) *ClaudeAdapter {
	verifier := structuralVerifier{
		provider:          install.ProviderClaude,
		fs:                fileSystem,
		functionalMessage: "functional verification is not supported for Claude Code; structural verification only",
	}

	return &ClaudeAdapter{
		managedAdapter: &managedAdapter{
			provider:        install.ProviderClaude,
			toolDir:         ".claude",
			projectRoot:     projectRoot,
			globalRoot:      globalRoot,
			fs:              fileSystem,
			verifier:        verifier,
			backupQualifier: "claude",
		},
	}
}

// ResolveTarget returns the Claude destination for an asset.
func (a *ClaudeAdapter) ResolveTarget(ctx context.Context, scope install.Scope, asset install.Asset) (ResolvedTarget, error) {
	return a.managedAdapter.ResolveTarget(ctx, scope, asset)
}
