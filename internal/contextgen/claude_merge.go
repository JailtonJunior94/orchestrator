package contextgen

import "strings"

const (
	userContentBegin = "<!-- ai-spec-harness:user-content-begin -->"
	userContentEnd   = "<!-- ai-spec-harness:user-content-end -->"

	governanceSchemaMarker = "<!-- governance-schema:"
)

type mergeRecorder interface {
	MarkMerged(path string)
}

func MergeUserContentMarkdown(generated, existing string, priorGenerated ...string) (string, bool) {
	stripped := stripGeneratedPrefix(generated, existing)
	for _, prior := range priorGenerated {
		if prior == "" {
			continue
		}
		stripped = stripGeneratedPrefix(prior, stripped)
	}
	return mergeGeneratedMarkdown(generated, stripped)
}

func ExtractUserContentBlock(content string) (string, bool) {
	begin := strings.Index(content, userContentBegin)
	if begin == -1 {
		return "", false
	}
	body := strings.TrimPrefix(content[begin+len(userContentBegin):], "\n")
	end := strings.LastIndex(body, userContentEnd)
	if end == -1 {
		return "", false
	}
	return strings.TrimSuffix(body[:end], "\n"), true
}

func mergeGeneratedMarkdown(generated, existing string) (string, bool) {
	preserved := extractUserContent(existing)
	if strings.TrimSpace(preserved) == "" {
		return generated, false
	}
	return generated + "\n" + userContentBegin + "\n" + preserved + "\n" + userContentEnd + "\n", true
}

func extractUserContent(existing string) string {
	if strings.TrimSpace(existing) == "" {
		return ""
	}

	if begin := strings.Index(existing, userContentBegin); begin != -1 {
		body := strings.TrimPrefix(existing[begin+len(userContentBegin):], "\n")
		if end := strings.LastIndex(body, userContentEnd); end != -1 {
			body = strings.TrimSuffix(body[:end], "\n")
		}
		return body
	}

	return existing
}

func (g *Generator) writeMergedMarkdown(path, generated string, priorGenerated ...string) error {
	content := generated
	merged := false
	if existing, err := g.fs.ReadFile(path); err == nil {
		content, merged = MergeUserContentMarkdown(generated, string(existing), priorGenerated...)
	}
	if err := g.fs.WriteFile(path, []byte(content)); err != nil {
		return err
	}
	if merged {
		g.recordMerge(path)
	}
	return nil
}

func stripGeneratedPrefix(generated, existing string) string {
	if strings.Contains(existing, userContentBegin) {
		return existing
	}
	if strings.HasPrefix(strings.TrimLeft(existing, " \t\r\n"), governanceSchemaMarker) {
		return ""
	}
	trimmedGenerated := strings.Trim(generated, "\n")
	if strings.Trim(existing, "\n") == trimmedGenerated {
		return ""
	}
	if trimmedGenerated == "" {
		return existing
	}
	if idx := strings.Index(existing, trimmedGenerated); idx != -1 && strings.TrimSpace(existing[:idx]) == "" {
		return strings.TrimLeft(existing[idx+len(trimmedGenerated):], "\n")
	}
	return existing
}

func (g *Generator) recordMerge(path string) {
	if recorder, ok := g.fs.(mergeRecorder); ok {
		recorder.MarkMerged(path)
	}
}
