package upgrade

import (
	"encoding/json"
	"testing"

	"github.com/JailtonJunior94/ai-spec-harness/internal/fs"
)

const obsoleteCopilotGovernanceJSON = `{
  "version": 1,
  "hooks": {
    "preToolUse": [
      {
        "type": "command",
        "bash": "bash .github/hooks/validate-preload.sh"
      }
    ],
    "postToolUse": [
      {
        "type": "command",
        "bash": "bash .github/hooks/validate-governance.sh"
      }
    ],
    "stop": [
      {
        "type": "command",
        "bash": "bash .github/hooks/subagent-stop-wrapper.sh"
      }
    ]
  }
}
`

func decodeCopilotHooks(t *testing.T, raw []byte) map[string][]map[string]any {
	t.Helper()
	var doc struct {
		Hooks map[string][]map[string]any `json:"hooks"`
	}
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatalf("decode governance.json: %v", err)
	}
	return doc.Hooks
}

func hookCommands(entries []map[string]any) []string {
	commands := make([]string, 0, len(entries))
	for _, entry := range entries {
		value, _ := entry["bash"].(string)
		commands = append(commands, value)
	}
	return commands
}

func containsCommand(commands []string, want string) bool {
	for _, command := range commands {
		if command == want {
			return true
		}
	}
	return false
}

func TestRepairCopilotGovernanceHooksMigratesObsoleteStopKey(t *testing.T) {
	ffs := fs.NewFakeFileSystem()
	path := "/project/.github/hooks/governance.json"
	ffs.Files[path] = []byte(obsoleteCopilotGovernanceJSON)

	changed, err := NewHelper().RepairCopilotGovernanceHooks(ffs, "/project")
	if err != nil {
		t.Fatalf("repair: %v", err)
	}
	if !changed {
		t.Fatalf("a governance.json carrying the dead %q key must be repaired", "stop")
	}

	hooks := decodeCopilotHooks(t, ffs.Files[path])
	if _, present := hooks["stop"]; present {
		t.Fatalf("the dead stop key must be removed; got %v", hooks)
	}
	commands := hookCommands(hooks[CopilotSessionEndHookKey])
	if !containsCommand(commands, "bash .github/hooks/subagent-stop-wrapper.sh") {
		t.Fatalf("the migrated entry must survive under %q; got %v", CopilotSessionEndHookKey, commands)
	}
	if !containsCommand(commands, "AISPEC_HOOK_DECISION_OUTPUT=json bash .github/hooks/validate-session-end.sh") {
		t.Fatalf("the session-end gate must be present under %q after repair; got %v", CopilotSessionEndHookKey, commands)
	}
}

func TestRepairCopilotGovernanceHooksPreservesUnrelatedContent(t *testing.T) {
	ffs := fs.NewFakeFileSystem()
	path := "/project/.github/hooks/governance.json"
	ffs.Files[path] = []byte(`{
  "version": 1,
  "customField": "keep-me",
  "hooks": {
    "stop": [
      {
        "type": "command",
        "bash": "bash .github/hooks/custom.sh"
      }
    ]
  }
}
`)

	if _, err := NewHelper().RepairCopilotGovernanceHooks(ffs, "/project"); err != nil {
		t.Fatalf("repair: %v", err)
	}

	var doc map[string]any
	if err := json.Unmarshal(ffs.Files[path], &doc); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if doc["customField"] != "keep-me" {
		t.Fatalf("unrelated top-level content must be preserved; got %v", doc)
	}
	if doc["version"] != float64(1) {
		t.Fatalf("version must be preserved; got %v", doc["version"])
	}
	commands := hookCommands(decodeCopilotHooks(t, ffs.Files[path])[CopilotSessionEndHookKey])
	if !containsCommand(commands, "bash .github/hooks/custom.sh") {
		t.Fatalf("a custom hook declared under the dead key must be migrated, not dropped; got %v", commands)
	}
}

func TestRepairCopilotGovernanceHooksIsIdempotent(t *testing.T) {
	ffs := fs.NewFakeFileSystem()
	path := "/project/.github/hooks/governance.json"
	ffs.Files[path] = []byte(obsoleteCopilotGovernanceJSON)

	if _, err := NewHelper().RepairCopilotGovernanceHooks(ffs, "/project"); err != nil {
		t.Fatalf("first repair: %v", err)
	}
	first := string(ffs.Files[path])

	changed, err := NewHelper().RepairCopilotGovernanceHooks(ffs, "/project")
	if err != nil {
		t.Fatalf("second repair: %v", err)
	}
	if changed {
		t.Fatalf("a repaired governance.json must converge; the second pass reported a change")
	}
	if string(ffs.Files[path]) != first {
		t.Fatalf("the second pass rewrote the file:\n%s\n---\n%s", first, ffs.Files[path])
	}
}

func TestRepairCopilotGovernanceHooksLeavesCurrentFileUntouched(t *testing.T) {
	ffs := fs.NewFakeFileSystem()
	path := "/project/.github/hooks/governance.json"
	ffs.Files[path] = []byte(DefaultCopilotGovernanceHooks)

	changed, err := NewHelper().RepairCopilotGovernanceHooks(ffs, "/project")
	if err != nil {
		t.Fatalf("repair: %v", err)
	}
	if changed {
		t.Fatalf("an already-current governance.json must not be rewritten")
	}
	if string(ffs.Files[path]) != DefaultCopilotGovernanceHooks {
		t.Fatalf("file was modified:\n%s", ffs.Files[path])
	}
}

func TestRepairCopilotGovernanceHooksCreatesMissingFile(t *testing.T) {
	ffs := fs.NewFakeFileSystem()

	changed, err := NewHelper().RepairCopilotGovernanceHooks(ffs, "/project")
	if err != nil {
		t.Fatalf("repair: %v", err)
	}
	if !changed {
		t.Fatalf("a missing governance.json must be created")
	}
	if string(ffs.Files["/project/.github/hooks/governance.json"]) != DefaultCopilotGovernanceHooks {
		t.Fatalf("unexpected content:\n%s", ffs.Files["/project/.github/hooks/governance.json"])
	}
}

func TestRepairCopilotGovernanceHooksRejectsMalformedDocument(t *testing.T) {
	ffs := fs.NewFakeFileSystem()
	ffs.Files["/project/.github/hooks/governance.json"] = []byte("{not json")

	if _, err := NewHelper().RepairCopilotGovernanceHooks(ffs, "/project"); err == nil {
		t.Fatalf("a malformed governance.json must surface an explicit error, never a silent skip")
	}
}

func TestDefaultCopilotGovernanceHooksDeclaresSessionEndGate(t *testing.T) {
	hooks := decodeCopilotHooks(t, []byte(DefaultCopilotGovernanceHooks))
	if _, present := hooks["stop"]; present {
		t.Fatalf("the shipped default must never declare the dead stop key")
	}
	commands := hookCommands(hooks[CopilotSessionEndHookKey])
	if !containsCommand(commands, "AISPEC_HOOK_DECISION_OUTPUT=json bash .github/hooks/validate-session-end.sh") {
		t.Fatalf("the shipped default must wire the session-end gate under %q; got %v", CopilotSessionEndHookKey, commands)
	}
}
