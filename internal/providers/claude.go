package providers

import (
	"time"

	"github.com/jailtonjunior/orchestrator/internal/platform"
)

// NewClaudeProvider creates the Claude CLI adapter.
func NewClaudeProvider(runner platform.CommandRunner) Provider {
	return cliProvider{
		name:           ClaudeProviderName,
		binary:         ClaudeProviderName,
		defaultTimeout: 5 * time.Minute,
		runner:         runner,
		profiles: []invocationProfile{
			{
				name:        "prompt-arg-json",
				probeTokens: []string{"-p", "--output-format"},
				build: func(prompt string) invocation {
					return invocation{
						args: []string{"-p", prompt, "--output-format", "json"},
					}
				},
			},
			{
				name:        "stdin-json",
				probeTokens: []string{"--output-format"},
				build: func(prompt string) invocation {
					return invocation{
						args:  []string{"--output-format", "json"},
						stdin: prompt,
					}
				},
			},
		},
	}
}
