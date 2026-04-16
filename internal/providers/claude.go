package providers

import (
	"os"
	"time"

	"github.com/jailtonjunior/orchestrator/internal/platform"
)

const claudeBinaryEnvVar = "ORQ_CLAUDE_BINARY"
const defaultClaudeBinary = ClaudeProviderName

// NewClaudeProvider creates the Claude CLI adapter.
func NewClaudeProvider(runner platform.CommandRunner) Provider {
	return cliProvider{
		name:           ClaudeProviderName,
		binary:         resolveClaudeBinary(),
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

func resolveClaudeBinary() string {
	if binary := os.Getenv(claudeBinaryEnvVar); binary != "" {
		return binary
	}

	return defaultClaudeBinary
}
