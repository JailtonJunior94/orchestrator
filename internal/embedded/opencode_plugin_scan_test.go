package embedded_test

import (
	"strings"
	"testing"

	"github.com/JailtonJunior94/ai-spec-harness/internal/embedded"
)

func TestOpenCodeGovernancePluginNeverRegistersPermissionHook(t *testing.T) {
	t.Parallel()

	data, err := embedded.Assets.ReadFile("assets/.opencode/plugin/governance.js")
	if err != nil {
		t.Fatalf("read embedded governance plugin: %v", err)
	}
	content := strings.ToLower(string(data))
	if strings.Contains(content, "permission") {
		t.Fatalf("governance.js must never register the permission hook — it is dead code in the current OpenCode build (V-06) and a gate based on it would fail silently; found %q", "permission")
	}
}

func TestOpenCodeGovernancePluginCoversExactlyThreeCanonicalHooks(t *testing.T) {
	t.Parallel()

	data, err := embedded.Assets.ReadFile("assets/.opencode/plugin/governance.js")
	if err != nil {
		t.Fatalf("read embedded governance plugin: %v", err)
	}
	content := string(data)
	for _, want := range []string{"tool.execute.before", "tool.execute.after", "session.idle"} {
		if !strings.Contains(content, want) {
			t.Errorf("governance.js does not cover canonical point %q", want)
		}
	}
}

func TestOpenCodeGovernancePluginSessionIdleNeverThrows(t *testing.T) {
	t.Parallel()

	data, err := embedded.Assets.ReadFile("assets/.opencode/plugin/governance.js")
	if err != nil {
		t.Fatalf("read embedded governance plugin: %v", err)
	}
	content := string(data)
	idx := strings.Index(content, "\"session.idle\": async")
	if idx < 0 {
		t.Fatalf("governance.js does not register session.idle")
	}
	handlerEnd := strings.Index(content[idx:], "\"tool.execute.after\"")
	body := content[idx:]
	if handlerEnd >= 0 {
		body = content[idx : idx+handlerEnd]
	}
	if strings.Contains(body, "throw ") {
		t.Fatalf("session.idle must be observational only (RF-19/8.7) — only tool.execute.before blocks; found a throw in its handler: %s", body)
	}
}

func TestOpenCodeGovernancePluginMutatingToolsListIsLockedToInvestigatedSet(t *testing.T) {
	t.Parallel()

	data, err := embedded.Assets.ReadFile("assets/.opencode/plugin/governance.js")
	if err != nil {
		t.Fatalf("read embedded governance plugin: %v", err)
	}
	content := string(data)
	const wantDeclaration = `const MUTATING_TOOLS = new Set(["bash", "write", "edit", "multiedit", "patch"])`
	if !strings.Contains(content, wantDeclaration) {
		t.Fatalf("MUTATING_TOOLS declaration changed from the investigated closed set (see 8.0_execution_report.md) — "+
			"any addition of a new OpenCode native mutating tool must update this test deliberately; want declaration %q not found", wantDeclaration)
	}
	if !strings.Contains(content, "not in the known mutating-tools set") {
		t.Fatalf("governance.js must warn when a tool outside MUTATING_TOOLS bypasses validation, to avoid a silent gate bypass for unrecognized tools")
	}
}
