package uninstall

import (
	"encoding/json"
	"testing"

	"github.com/JailtonJunior94/ai-spec-harness/internal/fs"
	"github.com/JailtonJunior94/ai-spec-harness/internal/output"
)

func claudeHookEntries(t *testing.T, data []byte, event string) []any {
	t.Helper()
	doc := map[string]any{}
	if err := json.Unmarshal(data, &doc); err != nil {
		t.Fatalf("parse settings: %v", err)
	}
	hooks, ok := doc["hooks"].(map[string]any)
	if !ok {
		return nil
	}
	entries, _ := hooks[event].([]any)
	return entries
}

func hasClaudeHookEntry(t *testing.T, data []byte, event, matcher, command string) bool {
	t.Helper()
	for _, rawEntry := range claudeHookEntries(t, data, event) {
		entry, ok := rawEntry.(map[string]any)
		if !ok {
			continue
		}
		pattern, _ := entry["matcher"].(string)
		if pattern != matcher {
			continue
		}
		nested, _ := entry["hooks"].([]any)
		for _, rawHook := range nested {
			hook, isHook := rawHook.(map[string]any)
			if !isHook {
				continue
			}
			if text, _ := hook["command"].(string); text == command {
				return true
			}
		}
	}
	return false
}

func TestUninstallKeepsAuthoredHookThatReusesGovernanceCommandUnderForeignMatcher(t *testing.T) {
	t.Parallel()
	ffs := fs.NewFakeFileSystem()
	seedMergedClaudeManifest(ffs)
	ffs.Files["/project/.claude/settings.local.json"] = []byte(`{
  "hooks": {
    "PostToolUse": [
      {
        "matcher": "Read",
        "hooks": [
          {"type": "command", "command": "bash .claude/hooks/validate-governance.sh"}
        ]
      },
      {
        "matcher": "Bash|Edit|Write|NotebookEdit|apply_patch",
        "hooks": [
          {"type": "command", "command": "bash .claude/hooks/validate-governance.sh"}
        ]
      }
    ]
  }
}
`)

	if err := NewService(ffs, output.New(false)).Execute("/project", false); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, err := ffs.ReadFile("/project/.claude/settings.local.json")
	if err != nil {
		t.Fatalf("authored settings file must survive uninstall: %v", err)
	}
	if !hasClaudeHookEntry(t, data, "PostToolUse", "Read", "bash .claude/hooks/validate-governance.sh") {
		t.Errorf("authored PostToolUse matcher %q was deleted; got: %s", "Read", data)
	}
	if hasClaudeHookEntry(t, data, "PostToolUse", "Bash|Edit|Write|NotebookEdit|apply_patch", "bash .claude/hooks/validate-governance.sh") {
		t.Errorf("harness-installed PostToolUse entry survived uninstall; got: %s", data)
	}
}

func TestUninstallKeepsAuthoredScriptStoredUnderClaudeHooksDirectory(t *testing.T) {
	t.Parallel()
	ffs := fs.NewFakeFileSystem()
	seedMergedClaudeManifest(ffs)
	original := `{
  "hooks": {
    "PreToolUse": [
      {
        "matcher": "Bash",
        "hooks": [
          {"type": "command", "command": "bash .claude/hooks/meu-hook-autoral.sh"}
        ]
      }
    ]
  }
}
`
	ffs.Files["/project/.claude/settings.local.json"] = []byte(original)
	ffs.Files["/project/.claude/hooks/meu-hook-autoral.sh"] = []byte("#!/usr/bin/env bash\nexit 0\n")

	if err := NewService(ffs, output.New(false)).Execute("/project", false); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, err := ffs.ReadFile("/project/.claude/settings.local.json")
	if err != nil {
		t.Fatalf("settings file holding only an authored hook must survive: %v", err)
	}
	if string(data) != original {
		t.Errorf("authored settings file was rewritten;\nwant:\n%s\ngot:\n%s", original, data)
	}
}

func TestUninstallRemovesHarnessInstalledClaudeHooksAcrossEveryCanonicalPoint(t *testing.T) {
	t.Parallel()
	ffs := fs.NewFakeFileSystem()
	seedMergedClaudeManifest(ffs)
	ffs.Files["/project/.claude/settings.local.json"] = []byte(`{
  "permissions": {"allow": ["Bash(ls:*)"]},
` + installClaudeSettingsTemplate[len("{\n"):])

	if err := NewService(ffs, output.New(false)).Execute("/project", false); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, err := ffs.ReadFile("/project/.claude/settings.local.json")
	if err != nil {
		t.Fatalf("user file must survive: %v", err)
	}
	doc := map[string]any{}
	if err := json.Unmarshal(data, &doc); err != nil {
		t.Fatalf("parse settings: %v", err)
	}
	if _, stillThere := doc["hooks"]; stillThere {
		t.Errorf("every harness hook should have been removed; got: %s", data)
	}
	if _, kept := doc["permissions"]; !kept {
		t.Errorf("user permissions were dropped; got: %s", data)
	}
}
