package acp_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/jailtonjunior/orchestrator/internal/acp"
)

// fakeLookup lets tests control which binaries appear to be in PATH.
type fakeLookup struct {
	available map[string]bool
}

func (f *fakeLookup) LookPath(file string) (string, error) {
	if f.available[file] {
		return "/usr/bin/" + file, nil
	}
	return "", errors.New("not found")
}

func TestRegistry_GetByValidName(t *testing.T) {
	t.Parallel()
	r := acp.NewRegistry(nil)

	for _, name := range []string{"claude", "codex", "gemini", "copilot"} {
		spec, err := r.Get(name)
		if err != nil {
			t.Errorf("Get(%q): unexpected error: %v", name, err)
		}
		if spec.Name != name {
			t.Errorf("Get(%q): got name %q", name, spec.Name)
		}
	}
}

func TestRegistry_GetByInvalidName(t *testing.T) {
	t.Parallel()
	r := acp.NewRegistry(nil)

	_, err := r.Get("unknown-provider")
	if err == nil {
		t.Fatal("expected error for unknown provider, got nil")
	}
	if !errors.Is(err, acp.ErrAgentNotAvailable) {
		t.Errorf("expected ErrAgentNotAvailable, got: %v", err)
	}
}

func TestRegistry_List(t *testing.T) {
	t.Parallel()
	r := acp.NewRegistry(nil)
	specs := r.List()
	if len(specs) != 4 {
		t.Errorf("expected 4 specs, got %d", len(specs))
	}
}

func TestRegistry_Resolve_NativeBinaryFound(t *testing.T) {
	t.Parallel()
	lookup := &fakeLookup{available: map[string]bool{"claude-agent-acp": true}}
	r := acp.NewRegistry(lookup)

	cmd, _, err := r.Resolve("claude")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cmd != "claude-agent-acp" {
		t.Errorf("expected native binary, got %q", cmd)
	}
}

func TestRegistry_Resolve_FallbackNpx(t *testing.T) {
	t.Parallel()
	// native binary absent, but npx is present
	lookup := &fakeLookup{available: map[string]bool{"npx": true}}
	r := acp.NewRegistry(lookup)

	cmd, args, err := r.Resolve("claude")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cmd != "npx" {
		t.Errorf("expected fallback command npx, got %q", cmd)
	}
	if len(args) == 0 {
		t.Error("expected fallback args to be non-empty")
	}
}

func TestRegistry_Resolve_NeitherAvailable(t *testing.T) {
	t.Parallel()
	lookup := &fakeLookup{available: map[string]bool{}} // nothing found
	r := acp.NewRegistry(lookup)

	_, _, err := r.Resolve("claude")
	if err == nil {
		t.Fatal("expected error when neither binary nor fallback available")
	}
	if !errors.Is(err, acp.ErrAgentNotAvailable) {
		t.Errorf("expected ErrAgentNotAvailable, got: %v", err)
	}
	if got := err.Error(); !strings.Contains(got, "install with: npx --yes @agentclientprotocol/claude-agent-acp") {
		t.Fatalf("unexpected install hint: %q", got)
	}
}

func TestRegistry_Available(t *testing.T) {
	t.Parallel()
	lookup := &fakeLookup{available: map[string]bool{"claude-agent-acp": true}}
	r := acp.NewRegistry(lookup)

	if !r.Available("claude") {
		t.Error("claude should be available")
	}
	if r.Available("codex") {
		t.Error("codex should not be available (no binary in fake lookup)")
	}
}
