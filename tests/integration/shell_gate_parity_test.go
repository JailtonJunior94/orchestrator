//go:build integration

package integration

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

const shellToolNameClaude = "Bash"

var shellGateMutatingClaudeTools = []string{"Bash", "Edit", "Write", "NotebookEdit", "apply_patch"}

type claudeHookMatcher struct {
	Matcher string `json:"matcher"`
	Hooks   []struct {
		Type    string `json:"type"`
		Command string `json:"command"`
	} `json:"hooks"`
}

type claudeSettingsDocument struct {
	Hooks map[string][]claudeHookMatcher `json:"hooks"`
}

func readClaudeSettings(t *testing.T, projectDir string) claudeSettingsDocument {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(projectDir, ".claude", "settings.local.json"))
	if err != nil {
		t.Fatalf("read claude settings: %v", err)
	}
	var doc claudeSettingsDocument
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatalf("decode claude settings: %v", err)
	}
	return doc
}

func TestClaudeToolHooksCoverShellTool(t *testing.T) {
	t.Parallel()
	projectDir := installMandatoryAgentsForDispatch(t)
	doc := readClaudeSettings(t, projectDir)

	for _, event := range []string{"PreToolUse", "PostToolUse"} {
		t.Run(event, func(t *testing.T) {
			entries, ok := doc.Hooks[event]
			if !ok || len(entries) == 0 {
				t.Fatalf("%s hook must be declared in .claude/settings.local.json", event)
			}
			covered := false
			for _, entry := range entries {
				pattern, err := regexp.Compile(entry.Matcher)
				if err != nil {
					t.Fatalf("%s matcher %q is not a valid regexp: %v", event, entry.Matcher, err)
				}
				if pattern.MatchString(shellToolNameClaude) {
					covered = true
				}
			}
			if !covered {
				t.Fatalf("%s matcher must cover the %s tool; a shell write (sed -i, python -c) routes around the gate otherwise: %+v",
					event, shellToolNameClaude, entries)
			}
		})
	}
}

func TestClaudeToolHooksCoverEveryMutatingTool(t *testing.T) {
	t.Parallel()
	projectDir := installMandatoryAgentsForDispatch(t)
	doc := readClaudeSettings(t, projectDir)

	for _, event := range []string{"PreToolUse", "PostToolUse"} {
		for _, tool := range shellGateMutatingClaudeTools {
			t.Run(event+"/"+tool, func(t *testing.T) {
				covered := false
				for _, entry := range doc.Hooks[event] {
					pattern, err := regexp.Compile(entry.Matcher)
					if err != nil {
						t.Fatalf("matcher %q is not a valid regexp: %v", entry.Matcher, err)
					}
					if pattern.MatchString(tool) {
						covered = true
					}
				}
				if !covered {
					t.Fatalf("%s matcher must cover the %s tool", event, tool)
				}
			})
		}
	}
}

func TestCodexAndCopilotToolHooksDeclareNoMatcher(t *testing.T) {
	t.Parallel()
	projectDir := installMandatoryAgentsForDispatch(t)

	codexConfig, err := os.ReadFile(filepath.Join(projectDir, ".codex", "config.toml"))
	if err != nil {
		t.Fatalf("read codex config: %v", err)
	}
	if strings.Contains(string(codexConfig), "matcher") {
		t.Fatalf("codex tool hooks must stay matcher-less so every tool (shell included) is gated; got:\n%s", codexConfig)
	}

	copilotHooks, err := os.ReadFile(filepath.Join(projectDir, ".github", "hooks", "governance.json"))
	if err != nil {
		t.Fatalf("read copilot governance.json: %v", err)
	}
	if strings.Contains(string(copilotHooks), "matcher") {
		t.Fatalf("copilot tool hooks must stay matcher-less so every tool (shell included) is gated; got:\n%s", copilotHooks)
	}
}

func TestOpenCodePluginTreatsShellAsMutating(t *testing.T) {
	t.Parallel()
	pluginPath := pluginAssetPath(t)
	raw, err := os.ReadFile(pluginPath)
	if err != nil {
		t.Fatalf("read plugin: %v", err)
	}
	if !strings.Contains(string(raw), `const MUTATING_TOOLS = new Set(["bash"`) {
		t.Fatalf("the opencode plugin must keep \"bash\" in MUTATING_TOOLS; got:\n%s", raw)
	}
}
