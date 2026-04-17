package providers

import (
	"testing"
)

func TestProviderNames(t *testing.T) {
	t.Parallel()

	names := []string{ClaudeProviderName, CopilotProviderName, GeminiProviderName, CodexProviderName}
	seen := make(map[string]struct{}, len(names))
	for _, name := range names {
		if name == "" {
			t.Fatal("provider name must not be empty")
		}
		if _, dup := seen[name]; dup {
			t.Fatalf("duplicate provider name: %q", name)
		}
		seen[name] = struct{}{}
	}
}
