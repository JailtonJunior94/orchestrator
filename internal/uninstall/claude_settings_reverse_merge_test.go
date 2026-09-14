package uninstall

import (
	"strings"
	"testing"

	"github.com/JailtonJunior94/ai-spec-harness/internal/fs"
	"github.com/JailtonJunior94/ai-spec-harness/internal/output"
)

func seedMergedClaudeManifest(ffs *fs.FakeFileSystem) {
	ffs.Files["/project/.ai_spec_harness.json"] = []byte(`{
  "version": "test",
  "tools": ["claude"],
  "installed_files": [],
  "merged_files": [".claude/settings.local.json"]
}`)
}

func TestUninstall_ReverseMergesClaudeSettingsRestoringUserFile(t *testing.T) {
	t.Parallel()
	ffs := fs.NewFakeFileSystem()
	seedMergedClaudeManifest(ffs)
	original := "{\"permissions\":{\"allow\":[\"Bash(ls:*)\"]},\"env\":{\"FOO\":\"bar\"},\n  \"hooks\": {\n    \"PreToolUse\": [\n      {\n        \"hooks\": [\n          {\n            \"command\": \"bash .claude/hooks/validate-preload.sh\",\n            \"type\": \"command\"\n          }\n        ],\n        \"matcher\": \"Bash|Edit|Write\"\n      }\n    ]\n  }}"
	ffs.Files["/project/.claude/settings.local.json"] = []byte(original)

	if err := NewService(ffs, output.New(false)).Execute("/project", false); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, err := ffs.ReadFile("/project/.claude/settings.local.json")
	if err != nil {
		t.Fatalf("arquivo do usuario nao pode ser removido: %v", err)
	}
	content := string(data)
	if strings.Contains(content, ".claude/hooks/") {
		t.Errorf("hooks de governanca deveriam ter sido removidos; obteve: %s", content)
	}
	for _, userKey := range []string{"permissions", "Bash(ls:*)", "FOO"} {
		if !strings.Contains(content, userKey) {
			t.Errorf("chave do usuario %q deveria ser preservada; obteve: %s", userKey, content)
		}
	}
}

func TestUninstall_ReverseMergeClaudeSettingsKeepsUserHooks(t *testing.T) {
	t.Parallel()
	ffs := fs.NewFakeFileSystem()
	seedMergedClaudeManifest(ffs)
	ffs.Files["/project/.claude/settings.local.json"] = []byte(`{
  "hooks": {
    "PreToolUse": [
      {
        "matcher": "Bash",
        "hooks": [
          {"type": "command", "command": "bash scripts/meu-hook.sh"}
        ]
      },
      {
        "matcher": "Bash|Edit|Write",
        "hooks": [
          {"type": "command", "command": "bash .claude/hooks/validate-preload.sh"}
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
		t.Fatalf("arquivo com hook do usuario nao pode ser removido: %v", err)
	}
	content := string(data)
	if !strings.Contains(content, "scripts/meu-hook.sh") {
		t.Errorf("hook do usuario deveria ser preservado; obteve: %s", content)
	}
	if strings.Contains(content, ".claude/hooks/") {
		t.Errorf("hook de governanca deveria ser removido; obteve: %s", content)
	}
}
