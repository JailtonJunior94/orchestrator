package contextgen

import "strings"

const (
	codexGeneratedBegin = "# ai-spec-harness:generated-begin"
	codexGeneratedEnd   = "# ai-spec-harness:generated-end"
	codexSkillsTable    = "[[skills.config]]"
)

func MergeCodexInstallConfig(generated, existing string) (string, bool) {
	preserved := StripLegacyCodexGenerated(StripCodexGeneratedRegion(existing))
	if strings.TrimSpace(preserved) == "" {
		return generated, false
	}
	head, tail := splitCodexPreserved(preserved)
	block := dropCodexRootKeysDeclaredBy(generated, preserved)
	return head + codexGeneratedBegin + "\n" + ensureTrailingNewline(block) + codexGeneratedEnd + "\n" + tail, true
}

func StripCodexGeneratedRegion(content string) string {
	begin := strings.Index(content, codexGeneratedBegin)
	if begin == -1 {
		return content
	}
	rest := content[begin+len(codexGeneratedBegin):]
	end := strings.Index(rest, codexGeneratedEnd)
	if end == -1 {
		return content[:begin]
	}
	return content[:begin] + strings.TrimPrefix(rest[end+len(codexGeneratedEnd):], "\n")
}

func HasCodexGeneratedContent(content string) bool {
	return strings.Contains(content, codexGeneratedBegin) || strings.Contains(content, codexSkillsTable)
}

func splitCodexPreserved(content string) (string, string) {
	offset := 0
	for offset < len(content) {
		line := content[offset:]
		next := len(content)
		if lineEnd := strings.IndexByte(line, '\n'); lineEnd != -1 {
			line = line[:lineEnd]
			next = offset + lineEnd + 1
		}
		if strings.HasPrefix(strings.TrimSpace(line), "[") {
			return content[:offset], content[offset:]
		}
		offset = next
	}
	return content, ""
}

func ensureTrailingNewline(s string) string {
	if s == "" || strings.HasSuffix(s, "\n") {
		return s
	}
	return s + "\n"
}

var codexLegacyExactLines = map[string]bool{
	`sandbox_mode = "workspace-write"`: true,
	`approval_policy = "on-request"`:   true,
	`type = "command"`:                 true,
	`[[hooks.PreToolUse]]`:             true,
	`[[hooks.PreToolUse.hooks]]`:       true,
	`[[hooks.PostToolUse]]`:            true,
	`[[hooks.PostToolUse.hooks]]`:      true,
	`[[hooks.SessionEnd]]`:             true,
	`[[hooks.SessionEnd.hooks]]`:       true,
	`[[hooks.Stop]]`:                   true,
	`[[hooks.Stop.hooks]]`:             true,
}

var codexLegacyPrefixes = []string{
	`command = "bash .codex/hooks/`,
	`path = ".agents/skills/`,
	`enabled = true`,
	codexSkillsTable,
	"# Governanca (paridade cross-CLI)",
	"# route-around documentada",
	"# de filesystem e comandos",
	"# Chaves de nivel raiz",
}

func StripLegacyCodexGenerated(content string) string {
	if strings.Contains(content, codexGeneratedBegin) {
		return content
	}
	lines := strings.Split(content, "\n")
	kept := make([]string, 0, len(lines))
	for _, line := range lines {
		if isCodexLegacyGeneratedLine(line) {
			continue
		}
		kept = append(kept, line)
	}
	out := strings.Join(kept, "\n")
	for strings.Contains(out, "\n\n\n") {
		out = strings.ReplaceAll(out, "\n\n\n", "\n\n")
	}
	return strings.TrimLeft(out, "\n")
}

func isCodexLegacyGeneratedLine(line string) bool {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" {
		return false
	}
	if codexLegacyExactLines[trimmed] {
		return true
	}
	for _, prefix := range codexLegacyPrefixes {
		if strings.HasPrefix(trimmed, prefix) {
			return true
		}
	}
	return false
}

var codexOwnedRootKeys = []string{"sandbox_mode", "approval_policy"}

func dropCodexRootKeysDeclaredBy(generated, preserved string) string {
	declared := make(map[string]bool, len(codexOwnedRootKeys))
	for _, key := range codexOwnedRootKeys {
		if declaresCodexRootKey(preserved, key) {
			declared[key] = true
		}
	}
	if len(declared) == 0 {
		return generated
	}
	lines := strings.Split(generated, "\n")
	kept := make([]string, 0, len(lines))
	for _, line := range lines {
		if key, ok := codexRootKeyOf(line); ok && declared[key] {
			continue
		}
		kept = append(kept, line)
	}
	return strings.Join(kept, "\n")
}

func declaresCodexRootKey(content, key string) bool {
	head, _ := splitCodexPreserved(content)
	for _, line := range strings.Split(head, "\n") {
		if found, ok := codexRootKeyOf(line); ok && found == key {
			return true
		}
	}
	return false
}

func codexRootKeyOf(line string) (string, bool) {
	trimmed := strings.TrimSpace(line)
	idx := strings.Index(trimmed, "=")
	if idx <= 0 {
		return "", false
	}
	return strings.TrimSpace(trimmed[:idx]), true
}
