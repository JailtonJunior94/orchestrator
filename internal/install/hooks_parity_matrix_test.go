package install_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
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
		return filepath.Join(".codex", "config.toml")
	case skills.ToolCopilot:
		return filepath.Join(".github", "settings.json")
	case skills.ToolOpenCode:
		return filepath.Join(".opencode", "plugin", "governance.js")
	default:
		return ""
	}
}

func mandatoryNativeConfigIsJSON(t skills.Tool) bool {
	return t == skills.ToolClaude || t == skills.ToolCopilot
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
	if key == "session.idle" {
		const eventHandler = "event: async ({ event }) => {"
		idx := strings.Index(content, eventHandler)
		if idx < 0 {
			return "", false
		}
		body := extractBalancedBraces(content, idx+len(eventHandler)-1)
		if !strings.Contains(body, "event.type !== \"session.idle\"") {
			return "", false
		}
		return body, true
	}
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

func codexHookKeyScope(content, nativeKey string) ([]string, bool) {
	header := "[[hooks." + nativeKey + "]]"
	lines := strings.Split(content, "\n")
	start := -1
	for i, line := range lines {
		if strings.TrimSpace(line) == header {
			start = i
			break
		}
	}
	if start < 0 {
		return nil, false
	}
	prefix := "[[hooks." + nativeKey
	scope := []string{lines[start]}
	for i := start + 1; i < len(lines); i++ {
		trimmed := strings.TrimSpace(lines[i])
		if strings.HasPrefix(trimmed, "[") && !strings.HasPrefix(trimmed, prefix) {
			break
		}
		scope = append(scope, lines[i])
	}
	return scope, true
}

var shellScriptTokenPattern = regexp.MustCompile(`[A-Za-z0-9_./-]+\.sh`)

func scriptCandidatesIn(scoped []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, s := range scoped {
		for _, m := range shellScriptTokenPattern.FindAllString(s, -1) {
			candidate := strings.TrimPrefix(m, "./")
			if seen[candidate] {
				continue
			}
			seen[candidate] = true
			out = append(out, candidate)
		}
	}
	return out
}

func resolvesToCanonicalValidator(t *testing.T, projectDir, candidate, canonical string) bool {
	t.Helper()
	return resolvesToCanonicalValidatorVisiting(t, projectDir, candidate, canonical, map[string]bool{})
}

func resolvesToCanonicalValidatorVisiting(t *testing.T, projectDir, candidate, canonical string, seen map[string]bool) bool {
	t.Helper()
	candidate = filepath.ToSlash(candidate)
	if candidate == filepath.ToSlash(canonical) {
		return true
	}
	if seen[candidate] {
		return false
	}
	seen[candidate] = true

	candidateData, err := os.ReadFile(filepath.Join(projectDir, filepath.FromSlash(candidate)))
	if err != nil {
		return false
	}
	canonicalData, err := os.ReadFile(filepath.Join(projectDir, filepath.FromSlash(canonical)))
	if err != nil {
		return false
	}
	if bytes.Equal(candidateData, canonicalData) {
		return true
	}
	// G6: mencao textual nao prova delegacao. Antes desta correcao um
	// strings.Contains(body, canonical) aceitava um wrapper que apenas citasse o
	// caminho canonico em comentario ou em mensagem de erro. A prova agora exige
	// que o caminho apareca em POSICAO DE EXECUCAO (exec/bash/sh/source/. ou como
	// comando), inclusive quando chega ali por variavel local.
	for _, delegate := range executedScriptTargets(string(candidateData)) {
		if delegate == filepath.ToSlash(canonical) {
			return true
		}
		if resolvesToCanonicalValidatorVisiting(t, projectDir, delegate, canonical, seen) {
			return true
		}
	}
	return false
}

var (
	shellCommandSeparatorPattern = regexp.MustCompile(`(?:\|\||&&|;;|;|\||&|\n|\$\(|\(|\)|\{|\}|` + "`" + `)`)
	shellAssignmentPattern       = regexp.MustCompile(`^(?:local|export|readonly|declare(?:\s+-\w+)?|typeset)?\s*([A-Za-z_][A-Za-z0-9_]*)=(.*)$`)
	shellVariableRefPattern      = regexp.MustCompile(`^\$\{?([A-Za-z_][A-Za-z0-9_]*)\}?$`)
)

var shellExecutorPrefixes = map[string]bool{
	"exec": true, "env": true, "command": true, "nohup": true, "sudo": true,
	"bash": true, "sh": true, "zsh": true, "source": true, ".": true,
	"if": true, "then": true, "else": true, "elif": true, "while": true,
	"until": true, "do": true, "!": true, "time": true,
}

func stripShellComments(body string) string {
	var out strings.Builder
	for _, line := range strings.Split(body, "\n") {
		if strings.HasPrefix(line, "#!") {
			out.WriteByte('\n')
			continue
		}
		trimmed := line
		if idx := strings.Index(trimmed, "#"); idx >= 0 {
			prefix := trimmed[:idx]
			if strings.TrimSpace(prefix) == "" || strings.HasSuffix(prefix, " ") || strings.HasSuffix(prefix, "\t") {
				trimmed = prefix
			}
		}
		out.WriteString(trimmed)
		out.WriteByte('\n')
	}
	return out.String()
}

var shellPathTokenPattern = regexp.MustCompile(`[A-Za-z0-9_./${}-]+\.sh`)

func normalizeScriptPath(token string) string {
	token = strings.Trim(token, "\"'")
	matches := shellPathTokenPattern.FindAllString(token, -1)
	if len(matches) == 0 {
		return ""
	}
	candidate := filepath.ToSlash(matches[len(matches)-1])
	for strings.HasPrefix(candidate, "$") {
		idx := strings.Index(candidate, "/")
		if idx < 0 {
			return ""
		}
		candidate = candidate[idx+1:]
	}
	candidate = strings.TrimPrefix(candidate, "./")
	candidate = strings.TrimPrefix(candidate, "/")
	if !strings.HasSuffix(candidate, ".sh") || strings.ContainsAny(candidate, "${}") {
		return ""
	}
	return candidate
}

func shellScriptAssignments(body string) map[string][]string {
	assignments := map[string][]string{}
	for _, line := range strings.Split(body, "\n") {
		match := shellAssignmentPattern.FindStringSubmatch(strings.TrimSpace(line))
		if match == nil {
			continue
		}
		for _, token := range shellPathTokenPattern.FindAllString(match[2], -1) {
			if normalized := normalizeScriptPath(token); normalized != "" {
				assignments[match[1]] = append(assignments[match[1]], normalized)
			}
		}
	}
	return assignments
}

// executedScriptTargets devolve os scripts que o corpo realmente executa,
// resolvendo variaveis locais atribuidas no proprio arquivo. Comentarios e
// mensagens (echo/printf) nunca produzem alvo.
func executedScriptTargets(body string) []string {
	body = stripShellComments(body)
	assignments := shellScriptAssignments(body)

	seen := map[string]bool{}
	var out []string
	add := func(target string) {
		if target == "" || seen[target] {
			return
		}
		seen[target] = true
		out = append(out, target)
	}

	for _, segment := range shellCommandSeparatorPattern.Split(body, -1) {
		fields := strings.Fields(segment)
		for _, field := range fields {
			bare := strings.Trim(field, "\"'")
			if shellExecutorPrefixes[bare] {
				continue
			}
			if strings.HasPrefix(bare, "-") {
				continue
			}
			if shellAssignmentPattern.MatchString(bare) && strings.Contains(bare, "=") {
				continue
			}
			if ref := shellVariableRefPattern.FindStringSubmatch(bare); ref != nil {
				for _, target := range assignments[ref[1]] {
					add(target)
				}
				break
			}
			add(normalizeScriptPath(bare))
			break
		}
	}
	return out
}

func installMandatoryMatrixProject(t *testing.T, generateCtx bool) string {
	t.Helper()

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
		ProjectDir:  projectDir,
		SourceDir:   sourceDir,
		Tools:       mandatoryMatrixAgents,
		LinkMode:    skills.LinkCopy,
		GenerateCtx: generateCtx,
	}); err != nil {
		t.Fatalf("install (GenerateCtx=%v): %v", generateCtx, err)
	}
	return projectDir
}

func TestMandatoryMatrixWrittenConfigContainsNativeKeyAndValidator(t *testing.T) {
	t.Parallel()

	for _, generateCtx := range []bool{false, true} {
		t.Run(fmt.Sprintf("GenerateCtx=%v", generateCtx), func(t *testing.T) {
			t.Parallel()
			projectDir := installMandatoryMatrixProject(t, generateCtx)

			catalog := specs.NewCatalog()
			points := catalog.CanonicalPoints()
			if len(points) != 3 {
				t.Fatalf("expected 3 canonical points, got %d", len(points))
			}
			if len(mandatoryMatrixAgents) != 4 {
				t.Fatalf("expected 4 mandatory agents, got %d", len(mandatoryMatrixAgents))
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
				data, err := os.ReadFile(filepath.Join(projectDir, configRelPath))
				if err != nil {
					t.Fatalf("agent %q: read installed native config %q: %v", tool, configRelPath, err)
				}
				content := string(data)

				for _, point := range points {
					cov, ok := agent.Enforcement().CoverageFor(point)
					if !ok {
						t.Errorf("agent %q: registry missing coverage for point %s", tool, point)
						continue
					}

					var scopedStrings []string
					var keyPresent bool
					switch {
					case mandatoryNativeConfigIsJSON(tool):
						scopedStrings, keyPresent = jsonHookKeyScope(t, tool, content, cov.NativeKey())
					case tool == skills.ToolCodex:
						scopedStrings, keyPresent = codexHookKeyScope(content, cov.NativeKey())
					default:
						scopedStrings, keyPresent = jsHookKeyScope(t, tool, content, cov.NativeKey())
					}
					if !keyPresent {
						t.Errorf("agent %q point %s: installed config %q does not structurally declare native key %q", tool, point, configRelPath, cov.NativeKey())
						cellsChecked++
						continue
					}

					candidates := scriptCandidatesIn(scopedStrings)
					found := false
					for _, candidate := range candidates {
						if resolvesToCanonicalValidator(t, projectDir, candidate, cov.ScriptPath()) {
							found = true
							break
						}
					}
					if !found {
						t.Errorf("agent %q point %s: installed config %q key %q invokes %v, none of which resolves to the canonical validator %q",
							tool, point, configRelPath, cov.NativeKey(), candidates, cov.ScriptPath())
					}
					cellsChecked++
				}
			}

			if cellsChecked != 12 {
				t.Fatalf("mandatory matrix must cover 4 agents x 3 canonical points = 12 cells; checked %d", cellsChecked)
			}
		})
	}
}

func TestCodexGovernanceRootKeysSurviveContextGeneration(t *testing.T) {
	t.Parallel()

	for _, generateCtx := range []bool{false, true} {
		t.Run(fmt.Sprintf("GenerateCtx=%v", generateCtx), func(t *testing.T) {
			t.Parallel()
			projectDir := installMandatoryMatrixProject(t, generateCtx)

			data, err := os.ReadFile(filepath.Join(projectDir, ".codex", "config.toml"))
			if err != nil {
				t.Fatalf("read .codex/config.toml: %v", err)
			}
			content := string(data)

			required := []string{
				"sandbox_mode",
				"approval_policy",
				"[[hooks.PreToolUse]]",
				"[[hooks.PostToolUse]]",
				"[[hooks.Stop]]",
				"[[skills.config]]",
			}
			for _, want := range required {
				if !strings.Contains(content, want) {
					t.Errorf("GenerateCtx=%v: .codex/config.toml lost %q; content=\n%s", generateCtx, want, content)
				}
			}

			firstTable := strings.Index(content, "[")
			for _, rootKey := range []string{"sandbox_mode", "approval_policy"} {
				idx := strings.Index(content, rootKey)
				if idx < 0 {
					continue
				}
				if firstTable >= 0 && idx > firstTable {
					t.Errorf("GenerateCtx=%v: root key %q must precede every TOML table to stay top-level", generateCtx, rootKey)
				}
			}
		})
	}
}

func TestCanonicalValidatorScriptsAreInstalledAndExecutable(t *testing.T) {
	t.Parallel()

	for _, generateCtx := range []bool{false, true} {
		t.Run(fmt.Sprintf("GenerateCtx=%v", generateCtx), func(t *testing.T) {
			t.Parallel()
			projectDir := installMandatoryMatrixProject(t, generateCtx)

			catalog := specs.NewCatalog()
			for _, tool := range mandatoryMatrixAgents {
				agent, err := catalog.AgentByID(string(tool))
				if err != nil {
					t.Fatalf("agent %q not in registry: %v", tool, err)
				}
				for _, cov := range agent.Enforcement().Coverage() {
					assertInstalledExecutable(t, projectDir, cov.ScriptPath())
				}
			}
		})
	}
}

func TestOpenCodePluginDeclaredScriptsExistAfterInstall(t *testing.T) {
	t.Parallel()

	projectDir := installMandatoryMatrixProject(t, true)

	pluginPath := filepath.Join(projectDir, ".opencode", "plugin", "governance.js")
	data, err := os.ReadFile(pluginPath)
	if err != nil {
		t.Fatalf("read installed governance plugin: %v", err)
	}

	consts := extractJSConstStrings(string(data))
	for _, name := range []string{"CANONICAL_PRE_TOOL_SCRIPT", "CANONICAL_POST_TOOL_SCRIPT", "CANONICAL_SESSION_END_SCRIPT"} {
		declared, ok := consts[name]
		if !ok {
			t.Fatalf("governance plugin does not declare %s", name)
		}
		assertInstalledExecutable(t, projectDir, declared)
	}
}

func assertInstalledExecutable(t *testing.T, projectDir, relPath string) {
	t.Helper()
	full := filepath.Join(projectDir, filepath.FromSlash(relPath))
	info, err := os.Stat(full)
	if err != nil {
		t.Errorf("validator %q was not installed: %v", relPath, err)
		return
	}
	if info.Mode().Perm()&0o111 == 0 {
		t.Errorf("validator %q is installed but not executable (mode %v)", relPath, info.Mode().Perm())
	}
}
