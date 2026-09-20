package upgrade

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"reflect"

	"github.com/JailtonJunior94/ai-spec-harness/internal/fs"
	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/specs"
)

const CopilotGovernanceHooksRelPath = ".github/hooks/governance.json"

const CopilotSessionEndHookKey = "agentStop"

var ObsoleteCopilotHookKeys = []string{"stop", "Stop"}

const DefaultCopilotGovernanceHooks = `{
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
    "agentStop": [
      {
        "type": "command",
        "bash": "bash .github/hooks/subagent-stop-wrapper.sh"
      },
      {
        "type": "command",
        "bash": "AISPEC_HOOK_DECISION_OUTPUT=json bash .github/hooks/validate-session-end.sh"
      }
    ]
  }
}
`

func (r1 *Helper) CopilotGovernanceHooksPath(projectDir string) string {
	return filepath.Join(projectDir, ".github", "hooks", "governance.json")
}

func (r1 *Helper) RepairCopilotGovernanceHooks(filesystem fs.FileSystem, projectDir string) (bool, error) {
	path := r1.CopilotGovernanceHooksPath(projectDir)
	if !filesystem.Exists(path) {
		if err := filesystem.WriteFile(path, []byte(DefaultCopilotGovernanceHooks)); err != nil {
			return false, err
		}
		return true, nil
	}

	raw, err := filesystem.ReadFile(path)
	if err != nil {
		return false, err
	}

	repaired, changed, err := r1.repairCopilotGovernanceDocument(raw)
	if err != nil {
		return false, err
	}
	if !changed {
		return false, nil
	}
	if err := filesystem.WriteFile(path, repaired); err != nil {
		return false, err
	}
	return true, nil
}

func (r1 *Helper) repairCopilotGovernanceDocument(raw []byte) ([]byte, bool, error) {
	if len(raw) == 0 {
		return []byte(DefaultCopilotGovernanceHooks), true, nil
	}

	doc := map[string]any{}
	if err := json.Unmarshal(raw, &doc); err != nil {
		return nil, false, fmt.Errorf("decode %s: %w", CopilotGovernanceHooksRelPath, err)
	}

	original, hadHooks := doc["hooks"]
	hooks, isObject := original.(map[string]any)
	if hadHooks && !isObject {
		return nil, false, fmt.Errorf("%s: hooks must be an object", CopilotGovernanceHooksRelPath)
	}

	merged := make(map[string]any, len(hooks))
	for event, value := range hooks {
		merged[event] = value
	}

	migrated := make([]any, 0, len(ObsoleteCopilotHookKeys))
	for _, obsolete := range ObsoleteCopilotHookKeys {
		entries, present := merged[obsolete]
		if !present {
			continue
		}
		delete(merged, obsolete)
		list, isList := entries.([]any)
		if !isList {
			return nil, false, fmt.Errorf("%s: hooks.%s must be an array", CopilotGovernanceHooksRelPath, obsolete)
		}
		migrated = append(migrated, list...)
	}

	var defaults struct {
		Hooks map[string][]any `json:"hooks"`
	}
	if err := json.Unmarshal([]byte(DefaultCopilotGovernanceHooks), &defaults); err != nil {
		return nil, false, err
	}

	merged[CopilotSessionEndHookKey] = r1.appendCopilotHookEntries(merged[CopilotSessionEndHookKey], migrated)
	for event, expected := range defaults.Hooks {
		merged[event] = r1.appendCopilotHookEntries(merged[event], expected)
	}

	if hadHooks && reflect.DeepEqual(original, merged) {
		return raw, false, nil
	}

	edited, editErr := specs.NewCatalog().SetJSONTopLevelKey(raw, "hooks", merged)
	if editErr == nil {
		return edited, true, nil
	}

	doc["hooks"] = merged
	encoded, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return nil, false, err
	}
	return append(encoded, '\n'), true, nil
}

func (r1 *Helper) appendCopilotHookEntries(existing any, extra []any) []any {
	list, _ := existing.([]any)
	result := make([]any, 0, len(list)+len(extra))
	result = append(result, list...)
	for _, entry := range extra {
		if r1.copilotHookEntryExists(result, entry) {
			continue
		}
		result = append(result, entry)
	}
	return result
}

func (r1 *Helper) copilotHookEntryExists(entries []any, expected any) bool {
	expectedJSON, err := json.Marshal(expected)
	if err != nil {
		return false
	}
	for _, entry := range entries {
		entryJSON, marshalErr := json.Marshal(entry)
		if marshalErr != nil {
			continue
		}
		if string(entryJSON) == string(expectedJSON) {
			return true
		}
	}
	return false
}
