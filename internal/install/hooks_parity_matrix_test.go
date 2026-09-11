package install_test

import (
	"encoding/json"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/JailtonJunior94/ai-spec-harness/internal/adapters"
	"github.com/JailtonJunior94/ai-spec-harness/internal/config"
	"github.com/JailtonJunior94/ai-spec-harness/internal/contextgen"
	"github.com/JailtonJunior94/ai-spec-harness/internal/embedded"
	"github.com/JailtonJunior94/ai-spec-harness/internal/fs"
	"github.com/JailtonJunior94/ai-spec-harness/internal/install"
	"github.com/JailtonJunior94/ai-spec-harness/internal/manifest"
	"github.com/JailtonJunior94/ai-spec-harness/internal/output"
	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/specs"
	"github.com/JailtonJunior94/ai-spec-harness/internal/skills"
)

var mandatoryMatrixAgents = []skills.Tool{
	skills.ToolClaude,
	skills.ToolCodex,
	skills.ToolCopilot,
	skills.ToolOpenCode,
}

func mandatoryNativeConfigPath(t skills.Tool) string {
	switch t {
	case skills.ToolClaude:
		return filepath.Join(".claude", "settings.local.json")
	case skills.ToolCodex:
		return filepath.Join(".codex", "hooks.json")
	case skills.ToolCopilot:
		return filepath.Join(".github", "hooks", "governance.json")
	case skills.ToolOpenCode:
		return filepath.Join(".opencode", "plugin", "governance.js")
	default:
		return ""
	}
}

func mandatoryNativeConfigIsJSON(t skills.Tool) bool {
	return t != skills.ToolOpenCode
}

func jsonHookKeyScope(t *testing.T, tool skills.Tool, content, nativeKey string) ([]string, bool) {
	t.Helper()
	var doc struct {
		Hooks map[string]json.RawMessage `json:"hooks"`
	}
	if err := json.Unmarshal([]byte(content), &doc); err != nil {
		t.Fatalf("agent %q: parse installed native config as JSON: %v", tool, err)
	}
	raw, ok := doc.Hooks[nativeKey]
	if !ok {
		return nil, false
	}
	return collectJSONStringValues(raw), true
}

func collectJSONStringValues(raw json.RawMessage) []string {
	var v any
	if err := json.Unmarshal(raw, &v); err != nil {
		return nil
	}
	var out []string
	var walk func(any)
	walk = func(node any) {
		switch typed := node.(type) {
		case string:
			out = append(out, typed)
		case []any:
			for _, item := range typed {
				walk(item)
			}
		case map[string]any:
			for _, item := range typed {
				walk(item)
			}
		}
	}
	walk(v)
	return out
}

var jsConstStringPattern = regexp.MustCompile(`const\s+([A-Za-z0-9_]+)\s*=\s*"([^"]*)"`)

var jsCallIdentifierPattern = regexp.MustCompile(`([A-Za-z_][A-Za-z0-9_]*)\(`)

var jsUpperIdentifierPattern = regexp.MustCompile(`\b[A-Z][A-Z0-9_]{2,}\b`)

var jsStringLiteralPattern = regexp.MustCompile(`"([^"]*)"`)

func extractJSConstStrings(content string) map[string]string {
	out := map[string]string{}
	for _, m := range jsConstStringPattern.FindAllStringSubmatch(content, -1) {
		out[m[1]] = m[2]
	}
	return out
}

func extractBalancedBraces(content string, openIdx int) string {
	depth := 0
	for i := openIdx; i < len(content); i++ {
		switch content[i] {
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return content[openIdx : i+1]
			}
		}
	}
	return content[openIdx:]
}

func findJSHandlerBody(content, key string) (string, bool) {
	pattern := regexp.MustCompile(`"` + regexp.QuoteMeta(key) + `":\s*async[^{]*\{`)
	loc := pattern.FindStringIndex(content)
	if loc == nil {
		return "", false
	}
	return extractBalancedBraces(content, loc[1]-1), true
}

func findJSFunctionBody(content, name string) (string, bool) {
	pattern := regexp.MustCompile(`function\s+` + regexp.QuoteMeta(name) + `\([^)]*\)\s*\{`)
	loc := pattern.FindStringIndex(content)
	if loc == nil {
		return "", false
	}
	return extractBalancedBraces(content, loc[1]-1), true
}

func extractJSCallIdentifiers(text string) []string {
	seen := map[string]bool{}
	var out []string
	for _, m := range jsCallIdentifierPattern.FindAllStringSubmatch(text, -1) {
		name := m[1]
		if seen[name] {
			continue
		}
		seen[name] = true
		out = append(out, name)
	}
	return out
}

func jsHookKeyScope(t *testing.T, tool skills.Tool, content, nativeKey string) ([]string, bool) {
	t.Helper()
	handlerBody, ok := findJSHandlerBody(content, nativeKey)
	if !ok {
		return nil, false
	}

	consts := extractJSConstStrings(content)
	scope := handlerBody
	visited := map[string]bool{}
	for _, called := range extractJSCallIdentifiers(handlerBody) {
		if visited[called] {
			continue
		}
		visited[called] = true
		if body, found := findJSFunctionBody(content, called); found {
			scope += "\n" + body
		}
	}

	var out []string
	for _, m := range jsStringLiteralPattern.FindAllStringSubmatch(scope, -1) {
		out = append(out, m[1])
	}
	for _, ident := range jsUpperIdentifierPattern.FindAllString(scope, -1) {
		if val, ok := consts[ident]; ok {
			out = append(out, val)
		}
	}
	return out, true
}

func TestMandatoryMatrixWrittenConfigContainsNativeKeyAndValidator(t *testing.T) {
	t.Parallel()

	sourceDir, cleanup, err := embedded.NewExtractor().ExtractToTempDir()
	if err != nil {
		t.Fatalf("extract embedded assets: %v", err)
	}
	t.Cleanup(cleanup)

	projectDir := t.TempDir()
	fsys := fs.NewOSFileSystem()
	printer := output.New(false)
	mfst := manifest.NewStore(fsys)
	adpt := adapters.NewGenerator(fsys, printer)
	ctxg := contextgen.NewGenerator(fsys, printer)
	svc := install.NewService(fsys, printer, mfst, adpt, ctxg)

	if err := svc.Execute(config.InstallOptions{
		ProjectDir: projectDir,
		SourceDir:  sourceDir,
		Tools:      mandatoryMatrixAgents,
		LinkMode:   skills.LinkCopy,
	}); err != nil {
		t.Fatalf("install: %v", err)
	}

	catalog := specs.NewCatalog()
	points := catalog.CanonicalPoints()
	if len(points) != 3 {
		t.Fatalf("expected 3 canonical points, got %d", len(points))
	}
	if len(mandatoryMatrixAgents) != 4 {
		t.Fatalf("expected 4 mandatory agents, got %d", len(mandatoryMatrixAgents))
	}

	acceptableValidators := map[specs.CanonicalPoint][]string{
		specs.PointPreTool:    {"validate-preload.sh", "hook-prereq-gate.sh"},
		specs.PointPostTool:   {"validate-governance.sh", "post-execute-task.sh"},
		specs.PointSessionEnd: {"validate-session-end.sh"},
	}

	nonBlockingPlaceholderCells := map[[2]string]bool{
		{"opencode", specs.PointPostTool.String()}: true,
	}

	cellsChecked := 0
	for _, tool := range mandatoryMatrixAgents {
		agent, err := catalog.AgentByID(string(tool))
		if err != nil {
			t.Fatalf("agent %q not in registry: %v", tool, err)
		}

		configRelPath := mandatoryNativeConfigPath(tool)
		if configRelPath == "" {
			t.Fatalf("no native config path mapped for tool %q", tool)
		}
		data, err := fsys.ReadFile(filepath.Join(projectDir, configRelPath))
		if err != nil {
			t.Fatalf("agent %q: read installed native config %q: %v", tool, configRelPath, err)
		}
		content := string(data)
		isJSON := mandatoryNativeConfigIsJSON(tool)

		for _, point := range points {
			cov, ok := agent.Enforcement().CoverageFor(point)
			if !ok {
				t.Errorf("agent %q: registry missing coverage for point %s", tool, point)
				continue
			}

			var scopedStrings []string
			var keyPresent bool
			if isJSON {
				scopedStrings, keyPresent = jsonHookKeyScope(t, tool, content, cov.NativeKey())
			} else {
				scopedStrings, keyPresent = jsHookKeyScope(t, tool, content, cov.NativeKey())
			}
			if !keyPresent {
				t.Errorf("agent %q point %s: installed config %q does not structurally declare native key %q", tool, point, configRelPath, cov.NativeKey())
			}

			if !nonBlockingPlaceholderCells[[2]string{string(tool), point.String()}] {
				found := false
				for _, candidate := range acceptableValidators[point] {
					for _, s := range scopedStrings {
						if strings.Contains(s, candidate) {
							found = true
							break
						}
					}
					if found {
						break
					}
				}
				if !found {
					t.Errorf("agent %q point %s: installed config %q key %q does not structurally invoke any declared validator %v", tool, point, configRelPath, cov.NativeKey(), acceptableValidators[point])
				}
			}
			cellsChecked++
		}
	}

	if cellsChecked != 12 {
		t.Fatalf("mandatory matrix must cover 4 agents x 3 canonical points = 12 cells; checked %d", cellsChecked)
	}
}
