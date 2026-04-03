package providers

import (
	"time"

	"github.com/jailtonjunior/orchestrator/internal/platform"
)

// NewCopilotProvider creates the Copilot CLI adapter.
func NewCopilotProvider(runner platform.CommandRunner) Provider {
	return cliProvider{
		name:           CopilotProviderName,
		binary:         CopilotProviderName,
		defaultTimeout: 10 * time.Minute,
		runner:         runner,
		profiles: []invocationProfile{
			{
				name:        "yolo-arg",
				probeTokens: []string{"--yolo"},
				build: func(prompt string) invocation {
					return invocation{
						args: []string{"--yolo", prompt},
					}
				},
			},
			{
				name:        "yolo-stdin",
				probeTokens: []string{"--yolo"},
				build: func(prompt string) invocation {
					return invocation{
						args:  []string{"--yolo"},
						stdin: prompt,
					}
				},
			},
		},
	}
}
