package specs

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

const (
	OpenCodeNpmPackage = "opencode-ai"

	OpenCodeNpmVersion = "1.18.30"

	OpenCodeSDKVersion = "v0.13.0"

	OpenCodeConservativeWindow = 16_000

	OpenCodeConfigFileName = "opencode.json"

	OpenCodePluginDir = ".opencode/plugin"

	OpenCodePluginRuntime = "bash"

	OpenCodeGovernanceSentinelEnvVar = "AISPEC_OPENCODE_GOVERNANCE_SENTINEL"

	OpenCodeOrchestratedEnvVar = "AISPEC_OPENCODE_ORCHESTRATED"
)

var OpenCodeKillSwitchVars = []string{
	"OPENCODE_PURE",
	"OPENCODE_DISABLE_PROJECT_CONFIG",
	"OPENCODE_DISABLE_EXTERNAL_SKILLS",
}

var ErrOpenCodePermissionForbidsAsk = errors.New("permission block must never use \"ask\" in orchestration (RF-20)")

var ErrOpenCodePermissionWildcardDeny = errors.New("blanket deny removes a required tool from the model's tool-set (RF-20)")

func ValidatePermissionBlock(perm map[string]any, toolsThatMustExist ...string) error {
	mustExist := make(map[string]bool, len(toolsThatMustExist))
	for _, t := range toolsThatMustExist {
		mustExist[t] = true
	}
	for tool, rule := range perm {
		if s, ok := rule.(string); ok {
			if s == "ask" {
				return ErrOpenCodePermissionForbidsAsk
			}
			if s == "deny" && mustExist[tool] {
				return fmt.Errorf("%w: tool %q", ErrOpenCodePermissionWildcardDeny, tool)
			}
			continue
		}
		if err := validatePermissionValue(rule); err != nil {
			return err
		}
	}
	return nil
}

func validatePermissionValue(v any) error {
	switch val := v.(type) {
	case string:
		if val == "ask" {
			return ErrOpenCodePermissionForbidsAsk
		}
	case map[string]any:
		for _, nested := range val {
			if err := validatePermissionValue(nested); err != nil {
				return err
			}
		}
	}
	return nil
}

func DefaultOpenCodePermission() map[string]any {
	return map[string]any{
		"bash": map[string]any{
			"rm -rf*":           "deny",
			"git push --force*": "deny",
			"git reset --hard*": "deny",
		},
	}
}

func MergeOpenCodeConfig(existing []byte, permission map[string]any, model string) ([]byte, error) {
	doc := map[string]any{}
	if len(existing) > 0 {
		if err := json.Unmarshal(existing, &doc); err != nil {
			return nil, fmt.Errorf("decode opencode.json: %w", err)
		}
	}
	if permission != nil {
		doc["permission"] = permission
	}
	if model != "" {
		doc["model"] = model
	}
	out, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("encode opencode.json: %w", err)
	}
	return append(out, '\n'), nil
}

func (c *Catalog) OpenCode() Spec {
	return NewCatalog().newSpecWithBootstrap(
		"opencode",
		"OpenCode (ACP)",
		"opencode",
		[]string{"acp"},
		[]FallbackLauncher{
			{
				Command:   "npx",
				FixedArgs: []string{"--yes", OpenCodeNpmPackage + "@" + OpenCodeNpmVersion, "acp"},
			},
		},
		"",
		OpenCodeSDKVersion,
		OpenCodeNpmVersion,
		OpenCodeNpmPackage,
		NewCatalog().openCodeBootstrapArgs,
		ContextWindow{},
		NewCatalog().resolveOpenCodeWindow,
	)
}

func (c *Catalog) openCodeBootstrapArgs(_, _ string, _ []string, _ AccessMode, workDir string) []string {
	args := make([]string, 0, 4)
	if workDir != "" {
		args = append(args, "--cwd", workDir)
	}
	args = append(args, "--log-level", "ERROR")
	return args
}

type modelWindowEntry struct {
	prefix string
	window ContextWindow
}

var openCodeWindowTable = []modelWindowEntry{
	{prefix: "claude-opus-4", window: ContextWindow{MaxTokens: 200_000}},
	{prefix: "claude-sonnet-4", window: ContextWindow{MaxTokens: 200_000}},
	{prefix: "claude-haiku-4", window: ContextWindow{MaxTokens: 200_000}},
	{prefix: "gpt-5", window: ContextWindow{MaxTokens: 400_000}},
	{prefix: "gemini-2.5-pro", window: ContextWindow{MaxTokens: 1_000_000}},
	{prefix: "gemini-2.5-flash", window: ContextWindow{MaxTokens: 1_000_000}},
}

func (c *Catalog) resolveOpenCodeWindow(model string) ContextWindow {
	if model == "" {
		return ContextWindow{MaxTokens: OpenCodeConservativeWindow}
	}
	for _, entry := range openCodeWindowTable {
		if entry.prefix == model {
			return entry.window
		}
	}
	bestPrefixLen := -1
	best := ContextWindow{MaxTokens: OpenCodeConservativeWindow}
	for _, entry := range openCodeWindowTable {
		if !strings.HasPrefix(model, entry.prefix) {
			continue
		}
		if len(entry.prefix) > bestPrefixLen {
			bestPrefixLen = len(entry.prefix)
			best = entry.window
		}
	}
	return best
}
