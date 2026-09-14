package agents_test

import (
	"context"
	"errors"
	"testing"

	"github.com/JailtonJunior94/ai-spec-harness/internal/agents"
	"github.com/JailtonJunior94/ai-spec-harness/internal/fs"
	"github.com/JailtonJunior94/ai-spec-harness/internal/skills"
)

const removedIDEAgentMD = "---\nname: foo\ndescription: agente de teste\nversion: 1.0.0\nruntime:\n  ide: gemini\n---\n\ncorpo\n"

func registryWithRemovedIDEAgent() agents.Registry {
	fake := fs.NewFakeFileSystem()
	fake.Files["/workspace/.ai-harness/agents/foo/AGENT.md"] = []byte(removedIDEAgentMD)
	return agents.NewDefaultRegistry(fake, "/workspace", "/home/user")
}

func TestRegistryDiscoverPropagatesRemovedAgentError(t *testing.T) {
	_, err := registryWithRemovedIDEAgent().Discover(context.Background())
	if err == nil {
		t.Fatal("Discover deveria falhar para runtime.ide gemini")
	}
	var removed *skills.RemovedAgentError
	if !errors.As(err, &removed) {
		t.Fatalf("erro deve ser *skills.RemovedAgentError na superficie, obteve %T: %v", err, err)
	}
	if removed.Agent != "gemini" {
		t.Errorf("RemovedAgentError.Agent = %q; want gemini", removed.Agent)
	}
}

func TestRegistryResolvePropagatesRemovedAgentError(t *testing.T) {
	_, err := registryWithRemovedIDEAgent().Resolve("foo")
	if err == nil {
		t.Fatal("Resolve deveria falhar para runtime.ide gemini")
	}
	var removed *skills.RemovedAgentError
	if !errors.As(err, &removed) {
		t.Fatalf("erro deve ser *skills.RemovedAgentError na superficie, obteve %T: %v", err, err)
	}
}
