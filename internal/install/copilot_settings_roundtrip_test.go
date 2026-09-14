package install

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/JailtonJunior94/ai-spec-harness/internal/config"
	"github.com/JailtonJunior94/ai-spec-harness/internal/fs"
	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/specs"
	"github.com/JailtonJunior94/ai-spec-harness/internal/skills"
)

func installCopilotSettings(t *testing.T, ffs *fs.FakeFileSystem) []byte {
	t.Helper()

	svc := setupTestService(ffs)
	if err := svc.Execute(config.InstallOptions{
		ProjectDir: "/project",
		SourceDir:  "/source",
		Tools:      []skills.Tool{skills.ToolCopilot},
		LinkMode:   skills.LinkCopy,
	}); err != nil {
		t.Fatalf("install copilot: %v", err)
	}
	data, err := ffs.ReadFile("/project/.github/settings.json")
	if err != nil {
		t.Fatalf("read .github/settings.json: %v", err)
	}
	return data
}

func withoutHooksKey(t *testing.T, raw []byte) []byte {
	t.Helper()

	stripped, err := specs.NewCatalog().DeleteJSONTopLevelKey(raw, "hooks")
	if err != nil {
		t.Fatalf("DeleteJSONTopLevelKey(%q): %v", raw, err)
	}
	return stripped
}

func TestInstallCopilotPreservesAuthoredSettingsFormatting(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		authored string
	}{
		{
			name:     "four space indent without trailing newline",
			authored: "{\n    \"zeta\": 1,\n    \"alpha\": 2\n}",
		},
		{
			name:     "four space indent with trailing newline",
			authored: "{\n    \"zeta\": 1,\n    \"alpha\": 2\n}\n",
		},
		{
			name:     "four space indent with authored hooks block",
			authored: "{\n    \"zeta\": 1,\n    \"hooks\": {\n        \"preToolUse\": [\n            {\"type\": \"command\", \"bash\": \"bash user-hook.sh\"}\n        ]\n    },\n    \"alpha\": 2\n}\n",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ffs := fs.NewFakeFileSystem()
			ffs.Dirs["/project"] = true
			ffs.Dirs["/source"] = true
			ffs.Files["/project/.github/settings.json"] = []byte(tc.authored)

			merged := installCopilotSettings(t, ffs)

			wantOutside := withoutHooksKey(t, []byte(tc.authored))
			gotOutside := withoutHooksKey(t, merged)
			if !bytes.Equal(wantOutside, gotOutside) {
				t.Fatalf("install reformatted content outside the governance hooks block:\nwant %q\ngot  %q", wantOutside, gotOutside)
			}

			var settings map[string]any
			if err := json.Unmarshal(merged, &settings); err != nil {
				t.Fatalf("merged settings is not valid JSON: %v\n%s", err, merged)
			}
			hooks, ok := settings["hooks"].(map[string]any)
			if !ok {
				t.Fatalf("merged settings has no hooks object: %s", merged)
			}
			for _, event := range []string{"preToolUse", "postToolUse", "agentStop"} {
				if _, exists := hooks[event]; !exists {
					t.Errorf("governance hook %q was not installed: %s", event, merged)
				}
			}
			if strings.Count(string(merged), "validate-preload.sh") != 1 {
				t.Errorf("preload hook duplicated: %s", merged)
			}

			second := installCopilotSettings(t, ffs)
			if !bytes.Equal(merged, second) {
				t.Fatalf("second install is not byte-identical:\nfirst  %q\nsecond %q", merged, second)
			}
		})
	}
}

func TestInstallCopilotMigratesObsoleteSessionEndKeysInSettings(t *testing.T) {
	t.Parallel()

	legacyEntry := map[string]any{"type": "command", "bash": "bash .github/hooks/legacy-session-end.sh"}

	for _, obsolete := range []string{"stop", "Stop", "sessionEnd", "SessionEnd"} {
		t.Run(obsolete, func(t *testing.T) {
			t.Parallel()

			authored, err := json.MarshalIndent(map[string]any{
				"hooks": map[string]any{obsolete: []any{legacyEntry}},
			}, "", "  ")
			if err != nil {
				t.Fatalf("marshal authored settings: %v", err)
			}

			ffs := fs.NewFakeFileSystem()
			ffs.Dirs["/project"] = true
			ffs.Dirs["/source"] = true
			ffs.Files["/project/.github/settings.json"] = authored

			merged := installCopilotSettings(t, ffs)

			var settings map[string]any
			if err := json.Unmarshal(merged, &settings); err != nil {
				t.Fatalf("merged settings is not valid JSON: %v\n%s", err, merged)
			}
			hooks, ok := settings["hooks"].(map[string]any)
			if !ok {
				t.Fatalf("merged settings has no hooks object: %s", merged)
			}
			if _, exists := hooks[obsolete]; exists {
				t.Fatalf("obsolete hook key %q survived the merge: %s", obsolete, merged)
			}
			agentStop, ok := hooks["agentStop"].([]any)
			if !ok {
				t.Fatalf("merged settings has no agentStop array: %s", merged)
			}
			found := false
			for _, entry := range agentStop {
				encoded, marshalErr := json.Marshal(entry)
				if marshalErr == nil && strings.Contains(string(encoded), "legacy-session-end.sh") {
					found = true
				}
			}
			if !found {
				t.Fatalf("entry migrated from %q was dropped instead of moved to agentStop: %s", obsolete, merged)
			}
		})
	}
}
