package specs

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"
)

const (
	OpenCodeNpmPackage = "opencode-ai"

	OpenCodeNpmVersion = "1.18.30"

	OpenCodeSDKVersion = "v0.13.0"

	OpenCodeConservativeWindow = 16_000

	OpenCodeConfigFileName = "opencode.json"

	OpenCodeSpecID = "opencode"

	OpenCodePluginDir = ".opencode/plugin"

	OpenCodePluginRuntime = "bash"

	OpenCodeGovernanceSentinelEnvVar = "AISPEC_OPENCODE_GOVERNANCE_SENTINEL"

	OpenCodeOrchestratedEnvVar = "AISPEC_OPENCODE_ORCHESTRATED"
)

var OpenCodeKillSwitchVars = []string{
	"OPENCODE_PURE",
	"OPENCODE_DISABLE_PROJECT_CONFIG",
	"OPENCODE_DISABLE_EXTERNAL_SKILLS",
	"OPENCODE_DISABLE_DEFAULT_PLUGINS",
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

	updates := make([][2]any, 0, 2)
	if permission != nil {
		merged := NewCatalog().mergePermission(doc, permission)
		if !reflect.DeepEqual(doc["permission"], merged) {
			updates = append(updates, [2]any{"permission", merged})
		}
		doc["permission"] = merged
	}
	if model != "" && doc["model"] != model {
		updates = append(updates, [2]any{"model", model})
		doc["model"] = model
	}

	if len(existing) == 0 {
		out, err := json.MarshalIndent(doc, "", "  ")
		if err != nil {
			return nil, fmt.Errorf("encode opencode.json: %w", err)
		}
		return append(out, '\n'), nil
	}

	if len(updates) == 0 {
		return existing, nil
	}

	out := existing
	for _, update := range updates {
		key, _ := update[0].(string)
		edited, err := NewCatalog().SetJSONTopLevelKey(out, key, update[1])
		if err != nil {
			encoded, encodeErr := json.MarshalIndent(doc, "", "  ")
			if encodeErr != nil {
				return nil, fmt.Errorf("encode opencode.json: %w", encodeErr)
			}
			return append(encoded, '\n'), nil
		}
		out = edited
	}
	return out, nil
}

func (c *Catalog) mergePermission(doc map[string]any, permission map[string]any) map[string]any {
	existingPermission, _ := doc["permission"].(map[string]any)
	merged := make(map[string]any, len(existingPermission)+len(permission))
	for tool, rule := range existingPermission {
		merged[tool] = rule
	}
	for tool, rule := range permission {
		requiredRule, ruleIsMap := rule.(map[string]any)
		currentRule, currentIsMap := merged[tool].(map[string]any)
		if ruleIsMap && currentIsMap {
			combined := make(map[string]any, len(currentRule)+len(requiredRule))
			for command, decision := range currentRule {
				combined[command] = decision
			}
			for command, decision := range requiredRule {
				combined[command] = decision
			}
			merged[tool] = combined
			continue
		}
		merged[tool] = rule
	}
	return merged
}

func (c *Catalog) OpenCode() Spec {
	return NewCatalog().newSpecWithBootstrap(
		OpenCodeSpecID,
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

func NormalizeModelID(model string) string {
	if idx := strings.LastIndex(model, "/"); idx >= 0 {
		return model[idx+1:]
	}
	return model
}

var ErrOpenCodeModelMismatch = errors.New("executor model diverges from the model declared in opencode.json (RF-17)")

func (c *Catalog) OpenCodeConfigModel(raw []byte) (string, error) {
	if len(raw) == 0 {
		return "", nil
	}
	doc := map[string]any{}
	if err := json.Unmarshal(raw, &doc); err != nil {
		return "", fmt.Errorf("decode %s: %w", OpenCodeConfigFileName, err)
	}
	model, _ := doc["model"].(string)
	return strings.TrimSpace(model), nil
}

func (c *Catalog) ResolveOpenCodeEffectiveModel(configModel, flagModel string) (string, error) {
	declared := strings.TrimSpace(configModel)
	requested := strings.TrimSpace(flagModel)
	if declared == "" {
		return requested, nil
	}
	if requested == "" {
		return declared, nil
	}
	if NormalizeModelID(requested) != NormalizeModelID(declared) {
		return "", fmt.Errorf("%w: requested %q, %s declares %q", ErrOpenCodeModelMismatch, requested, OpenCodeConfigFileName, declared)
	}
	return declared, nil
}

func OpenCodeModelPrefixes() []string {
	prefixes := make([]string, 0, len(openCodeWindowTable))
	for _, entry := range openCodeWindowTable {
		prefixes = append(prefixes, entry.prefix)
	}
	return prefixes
}

func (c *Catalog) MatchOpenCodeModel(model string) (ContextWindow, bool) {
	normalized := NormalizeModelID(model)
	if normalized == "" {
		return ContextWindow{}, false
	}
	for _, entry := range openCodeWindowTable {
		if entry.prefix == normalized {
			return entry.window, true
		}
	}
	bestPrefixLen := -1
	var best ContextWindow
	matched := false
	for _, entry := range openCodeWindowTable {
		if !strings.HasPrefix(normalized, entry.prefix) {
			continue
		}
		if len(entry.prefix) > bestPrefixLen {
			bestPrefixLen = len(entry.prefix)
			best = entry.window
			matched = true
		}
	}
	return best, matched
}

func (c *Catalog) resolveOpenCodeWindow(model string) ContextWindow {
	if window, matched := c.MatchOpenCodeModel(model); matched {
		return window
	}
	return ContextWindow{MaxTokens: OpenCodeConservativeWindow}
}
