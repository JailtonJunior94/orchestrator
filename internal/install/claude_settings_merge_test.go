package install

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/JailtonJunior94/ai-spec-harness/internal/config"
	"github.com/JailtonJunior94/ai-spec-harness/internal/fs"
	"github.com/JailtonJunior94/ai-spec-harness/internal/skills"
)

func claudeHookCommands(t *testing.T, raw []byte) []string {
	t.Helper()
	var doc struct {
		Hooks map[string][]struct {
			Matcher string `json:"matcher"`
			Hooks   []struct {
				Type    string `json:"type"`
				Command string `json:"command"`
			} `json:"hooks"`
		} `json:"hooks"`
	}
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatalf("settings.local.json invalido: %v\n%s", err, raw)
	}
	var out []string
	for _, entries := range doc.Hooks {
		for _, entry := range entries {
			for _, hook := range entry.Hooks {
				out = append(out, hook.Command)
			}
		}
	}
	return out
}

func TestInstall_Claude_MergesGovernanceHooksIntoExistingSettings(t *testing.T) {
	t.Parallel()
	ffs := fs.NewFakeFileSystem()
	ffs.Dirs["/project"] = true
	ffs.Dirs["/source"] = true
	ffs.Files["/source/.agents/skills/review/SKILL.md"] = []byte("---\nversion: 1.0.0\ndescription: Revisa codigo.\n---\n")
	ffs.Files["/project/.claude/settings.local.json"] = []byte("{\n  \"permissions\": {\n    \"allow\": [\n      \"Bash(ls:*)\"\n    ]\n  }\n}\n")

	svc := setupTestService(ffs)

	err := svc.Execute(config.InstallOptions{
		ProjectDir: "/project",
		SourceDir:  "/source",
		Tools:      []skills.Tool{skills.ToolClaude},
		LinkMode:   skills.LinkCopy,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	written := ffs.Files["/project/.claude/settings.local.json"]
	commands := strings.Join(claudeHookCommands(t, written), "\n")
	for _, want := range []string{"validate-preload.sh", "validate-governance.sh", "validate-session-end.sh"} {
		if !strings.Contains(commands, want) {
			t.Errorf("settings.local.json preexistente deve receber o hook %s; comandos: %q", want, commands)
		}
	}

	var doc map[string]any
	if err := json.Unmarshal(written, &doc); err != nil {
		t.Fatalf("settings.local.json invalido: %v", err)
	}
	if _, ok := doc["permissions"]; !ok {
		t.Errorf("merge deve preservar as chaves do usuario; obteve: %s", written)
	}
}

func TestInstall_Claude_MergeIsIdempotentAndPreservesUserHooks(t *testing.T) {
	t.Parallel()
	ffs := fs.NewFakeFileSystem()
	ffs.Dirs["/project"] = true
	ffs.Dirs["/source"] = true
	ffs.Files["/source/.agents/skills/review/SKILL.md"] = []byte("---\nversion: 1.0.0\ndescription: Revisa codigo.\n---\n")
	ffs.Files["/project/.claude/settings.local.json"] = []byte(`{
  "hooks": {
    "PreToolUse": [
      {
        "matcher": "Bash",
        "hooks": [
          {
            "type": "command",
            "command": "bash scripts/meu-hook.sh"
          }
        ]
      }
    ]
  }
}
`)

	svc := setupTestService(ffs)
	opts := config.InstallOptions{
		ProjectDir: "/project",
		SourceDir:  "/source",
		Tools:      []skills.Tool{skills.ToolClaude},
		LinkMode:   skills.LinkCopy,
	}

	if err := svc.Execute(opts); err != nil {
		t.Fatalf("primeira instalacao: %v", err)
	}
	first := string(ffs.Files["/project/.claude/settings.local.json"])

	if err := svc.Execute(opts); err != nil {
		t.Fatalf("segunda instalacao: %v", err)
	}
	second := string(ffs.Files["/project/.claude/settings.local.json"])

	if first != second {
		t.Errorf("merge nao e idempotente:\n--- primeira ---\n%s\n--- segunda ---\n%s", first, second)
	}
	if !strings.Contains(second, "scripts/meu-hook.sh") {
		t.Errorf("merge deve preservar hooks do usuario; obteve: %s", second)
	}
	if !strings.Contains(second, "validate-preload.sh") {
		t.Errorf("merge deve adicionar o hook de governanca; obteve: %s", second)
	}
}

func TestInstall_Claude_MalformedSettingsFailsLoudly(t *testing.T) {
	t.Parallel()
	ffs := fs.NewFakeFileSystem()
	ffs.Dirs["/project"] = true
	ffs.Dirs["/source"] = true
	ffs.Files["/source/.agents/skills/review/SKILL.md"] = []byte("---\nversion: 1.0.0\ndescription: Revisa codigo.\n---\n")
	ffs.Files["/project/.claude/settings.local.json"] = []byte("{ nao e json")

	svc := setupTestService(ffs)

	err := svc.Execute(config.InstallOptions{
		ProjectDir: "/project",
		SourceDir:  "/source",
		Tools:      []skills.Tool{skills.ToolClaude},
		LinkMode:   skills.LinkCopy,
	})
	if err == nil {
		t.Fatal("settings.local.json malformado deve falhar a instalacao em vez de deixar o Claude sem enforcement")
	}
	if !strings.Contains(err.Error(), "settings.local.json") {
		t.Errorf("erro deve citar o arquivo problematico, obteve: %v", err)
	}
}
