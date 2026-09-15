package specs_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/specs"
)

func overridingResolver(t *testing.T, overrides map[string][]byte, absent map[string]bool) specs.ScriptResolver {
	t.Helper()
	root := repoRoot(t)
	return func(relPath string) ([]byte, error) {
		if absent[relPath] {
			return nil, os.ErrNotExist
		}
		if data, ok := overrides[relPath]; ok {
			return data, nil
		}
		return os.ReadFile(filepath.Join(root, filepath.FromSlash(relPath)))
	}
}

func repoFile(t *testing.T, relPath string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(repoRoot(t), filepath.FromSlash(relPath)))
	if err != nil {
		t.Fatalf("read %s: %v", relPath, err)
	}
	return string(data)
}

func violationFor(violations []specs.ParityViolation, agent string, point specs.CanonicalPoint, fragment string) bool {
	for _, v := range violations {
		if v.Agent == agent && v.Point == point && strings.Contains(v.Reason, fragment) {
			return true
		}
	}
	return false
}

func TestParityGateFlagsCodexConfigThatRenamesTheNativeHookKey(t *testing.T) {
	t.Parallel()

	mutated := strings.ReplaceAll(repoFile(t, ".codex/config.toml"), "[[hooks.PreToolUse", "[[hooks.BogusPreToolUse")
	resolve := overridingResolver(t, map[string][]byte{".codex/config.toml": []byte(mutated)}, nil)

	violations := specs.ValidateParityMatrix(mandatoryParityCells(t), mandatoryParityAgents, allCellsDispatchProven, resolve)
	if !violationFor(violations, "codex", specs.PointPreTool, `does not declare native key "PreToolUse"`) {
		t.Fatalf("renaming the codex PreToolUse hook key must break the parity gate; got %v", violations)
	}
}

func TestParityGateFlagsOpenCodePluginThatRenamesTheNativeHookKey(t *testing.T) {
	t.Parallel()

	mutated := strings.ReplaceAll(repoFile(t, ".opencode/plugin/governance.js"), `"tool.execute.before":`, `"tool.execute.beforeX":`)
	resolve := overridingResolver(t, map[string][]byte{".opencode/plugin/governance.js": []byte(mutated)}, nil)

	violations := specs.ValidateParityMatrix(mandatoryParityCells(t), mandatoryParityAgents, allCellsDispatchProven, resolve)
	if !violationFor(violations, "opencode", specs.PointPreTool, `does not declare native key "tool.execute.before"`) {
		t.Fatalf("renaming the opencode pre-tool handler must break the parity gate; got %v", violations)
	}
}

func TestParityGateFlagsCopilotConfigThatRenamesTheNativeHookKey(t *testing.T) {
	t.Parallel()

	mutated := strings.ReplaceAll(repoFile(t, ".github/hooks/governance.json"), `"postToolUse"`, `"postToolUseX"`)
	resolve := overridingResolver(t, map[string][]byte{".github/hooks/governance.json": []byte(mutated)}, nil)

	violations := specs.ValidateParityMatrix(mandatoryParityCells(t), mandatoryParityAgents, allCellsDispatchProven, resolve)
	if !violationFor(violations, "copilot", specs.PointPostTool, `does not declare native key "postToolUse"`) {
		t.Fatalf("renaming the copilot postToolUse hook key must break the parity gate; got %v", violations)
	}
}

func TestParityGateFlagsNativeKeyWiredToANonCanonicalValidator(t *testing.T) {
	t.Parallel()

	mutated := strings.ReplaceAll(repoFile(t, ".codex/config.toml"),
		`command = "bash .codex/hooks/validate-preload.sh"`,
		`command = "bash .codex/hooks/nao-existe.sh"`)
	resolve := overridingResolver(t, map[string][]byte{".codex/config.toml": []byte(mutated)}, nil)

	violations := specs.ValidateParityMatrix(mandatoryParityCells(t), mandatoryParityAgents, allCellsDispatchProven, resolve)
	if !violationFor(violations, "codex", specs.PointPreTool, "without wiring it to the canonical validator") {
		t.Fatalf("a native key pointing at a foreign script must break the parity gate; got %v", violations)
	}
}

func TestParityGateFlagsMissingVersionedNativeConfig(t *testing.T) {
	t.Parallel()

	resolve := overridingResolver(t, nil, map[string]bool{".codex/config.toml": true})

	violations := specs.ValidateParityMatrix(mandatoryParityCells(t), mandatoryParityAgents, allCellsDispatchProven, resolve)
	if !violationFor(violations, "codex", specs.PointPreTool, "is missing, so native key") {
		t.Fatalf("a missing versioned CLI config must break the parity gate; got %v", violations)
	}
}

func TestParityGateConfrontsEveryMandatoryAgentWithAVersionedNativeConfig(t *testing.T) {
	t.Parallel()

	for _, agentID := range mandatoryParityAgents {
		paths := specs.NativeConfigPaths(agentID)
		if len(paths) == 0 {
			t.Errorf("agent %q declares no native CLI config, so its native keys are never confronted", agentID)
		}
	}
}
