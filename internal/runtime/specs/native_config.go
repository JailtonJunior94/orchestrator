package specs

import (
	"bytes"
	"encoding/json"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
)

func confrontNativeConfig(agentID string, point CanonicalPoint, cov PointCoverage, resolve ScriptResolver) []ParityViolation {
	sources, known := NativeConfigSourcesFor(agentID)
	if !known {
		return []ParityViolation{{Agent: agentID, Point: point, Reason: "native key never confronted: agent has no declared CLI config file"}}
	}

	var violations []ParityViolation
	for _, source := range sources {
		data, err := resolve(source.path)
		if err != nil {
			if source.required {
				violations = append(violations, ParityViolation{
					Agent:  agentID,
					Point:  point,
					Reason: fmt.Sprintf("versioned CLI config %q is missing, so native key %q is never confronted: %v", source.path, cov.NativeKey(), err),
				})
			}
			continue
		}
		violations = append(violations, confrontNativeConfigSource(agentID, point, cov, source, data, resolve)...)
	}
	return violations
}

func confrontNativeConfigSource(agentID string, point CanonicalPoint, cov PointCoverage, source nativeConfigSource, data []byte, resolve ScriptResolver) []ParityViolation {
	scope, declared := nativeKeyScope(source.format, string(data), cov.NativeKey())
	if !declared {
		return []ParityViolation{{
			Agent:  agentID,
			Point:  point,
			Reason: fmt.Sprintf("CLI config %q does not declare native key %q for point %s", source.path, cov.NativeKey(), point),
		}}
	}
	for _, candidate := range scriptCandidatesIn(scope) {
		if candidateResolvesToValidator(candidate, cov.ScriptPath(), resolve, map[string]bool{}) {
			return nil
		}
	}
	return []ParityViolation{{
		Agent:  agentID,
		Point:  point,
		Reason: fmt.Sprintf("CLI config %q declares native key %q without wiring it to the canonical validator %q", source.path, cov.NativeKey(), cov.ScriptPath()),
	}}
}

func nativeKeyScope(format nativeConfigFormat, content, nativeKey string) ([]string, bool) {
	switch format {
	case nativeConfigTOML:
		return tomlHookKeyScope(content, nativeKey)
	case nativeConfigJS:
		return jsHookKeyScope(content, nativeKey)
	default:
		return jsonHookKeyScope(content, nativeKey)
	}
}

func jsonHookKeyScope(content, nativeKey string) ([]string, bool) {
	var doc struct {
		Hooks map[string]json.RawMessage `json:"hooks"`
	}
	if err := json.Unmarshal([]byte(content), &doc); err != nil {
		return nil, false
	}
	raw, ok := doc.Hooks[nativeKey]
	if !ok {
		return nil, false
	}
	var node any
	if err := json.Unmarshal(raw, &node); err != nil {
		return nil, false
	}
	return collectJSONStrings(node), true
}

func collectJSONStrings(node any) []string {
	switch typed := node.(type) {
	case string:
		return []string{typed}
	case []any:
		var out []string
		for _, item := range typed {
			out = append(out, collectJSONStrings(item)...)
		}
		return out
	case map[string]any:
		var out []string
		for _, item := range typed {
			out = append(out, collectJSONStrings(item)...)
		}
		return out
	default:
		return nil
	}
}

func tomlHookKeyScope(content, nativeKey string) ([]string, bool) {
	header := "[[hooks." + nativeKey + "]]"
	prefix := "[[hooks." + nativeKey
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

var (
	jsConstStringPattern  = regexp.MustCompile(`const\s+([A-Za-z0-9_]+)\s*=\s*"([^"]*)"`)
	jsCallIdentifier      = regexp.MustCompile(`([A-Za-z_][A-Za-z0-9_]*)\(`)
	jsUpperIdentifier     = regexp.MustCompile(`\b[A-Z][A-Z0-9_]{2,}\b`)
	jsStringLiteral       = regexp.MustCompile(`"([^"]*)"`)
	jsEventHandlerPattern = regexp.MustCompile(`event:\s*async\s*\([^)]*\)\s*=>\s*\{`)
)

func jsHookKeyScope(content, nativeKey string) ([]string, bool) {
	body, ok := jsHandlerBody(content, nativeKey)
	if !ok {
		return nil, false
	}

	scope := body
	visited := map[string]bool{}
	for _, match := range jsCallIdentifier.FindAllStringSubmatch(body, -1) {
		name := match[1]
		if visited[name] {
			continue
		}
		visited[name] = true
		if called, found := jsFunctionBody(content, name); found {
			scope += "\n" + called
		}
	}

	consts := map[string]string{}
	for _, match := range jsConstStringPattern.FindAllStringSubmatch(content, -1) {
		consts[match[1]] = match[2]
	}

	var out []string
	for _, match := range jsStringLiteral.FindAllStringSubmatch(scope, -1) {
		out = append(out, match[1])
	}
	for _, ident := range jsUpperIdentifier.FindAllString(scope, -1) {
		if value, found := consts[ident]; found {
			out = append(out, value)
		}
	}
	return out, true
}

func jsHandlerBody(content, nativeKey string) (string, bool) {
	if strings.HasPrefix(nativeKey, "session.") {
		loc := jsEventHandlerPattern.FindStringIndex(content)
		if loc == nil {
			return "", false
		}
		body := balancedBraces(content, loc[1]-1)
		if !strings.Contains(body, `event.type !== "`+nativeKey+`"`) {
			return "", false
		}
		return body, true
	}
	pattern := regexp.MustCompile(`"` + regexp.QuoteMeta(nativeKey) + `":\s*async[^{]*\{`)
	loc := pattern.FindStringIndex(content)
	if loc == nil {
		return "", false
	}
	return balancedBraces(content, loc[1]-1), true
}

func jsFunctionBody(content, name string) (string, bool) {
	pattern := regexp.MustCompile(`function\s+` + regexp.QuoteMeta(name) + `\([^)]*\)\s*\{`)
	loc := pattern.FindStringIndex(content)
	if loc == nil {
		return "", false
	}
	return balancedBraces(content, loc[1]-1), true
}

func balancedBraces(content string, openIdx int) string {
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

var shellScriptToken = regexp.MustCompile(`[A-Za-z0-9_./-]+\.sh`)

func scriptCandidatesIn(scope []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, line := range scope {
		for _, match := range shellScriptToken.FindAllString(line, -1) {
			candidate := strings.TrimPrefix(filepath.ToSlash(match), "./")
			if seen[candidate] {
				continue
			}
			seen[candidate] = true
			out = append(out, candidate)
		}
	}
	return out
}

func candidateResolvesToValidator(candidate, canonical string, resolve ScriptResolver, seen map[string]bool) bool {
	candidate = filepath.ToSlash(candidate)
	if candidate == filepath.ToSlash(canonical) {
		return true
	}
	if seen[candidate] {
		return false
	}
	seen[candidate] = true

	candidateData, err := resolve(candidate)
	if err != nil {
		return false
	}
	canonicalData, err := resolve(canonical)
	if err != nil {
		return false
	}
	if bytes.Equal(candidateData, canonicalData) {
		return true
	}
	if scriptExecutesTarget(candidateData, canonical) {
		return true
	}
	for _, delegate := range scriptCandidatesIn([]string{string(candidateData)}) {
		if candidateResolvesToValidator(delegate, canonical, resolve, seen) {
			return true
		}
	}
	return false
}
