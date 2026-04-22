package skillbump

import (
	"bytes"
	"fmt"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

var versionLinePattern = regexp.MustCompile(`(?m)^([ \t]*)version:\s*.*$`)

// UpdateFrontmatterVersion atualiza o campo version no frontmatter YAML.
func UpdateFrontmatterVersion(content []byte, newVersion string) ([]byte, error) {
	bodyStart, bodyEnd, lineEnding, err := frontmatterBodyRange(content)
	if err != nil {
		return nil, fmt.Errorf("frontmatter invalido: %w", err)
	}

	body := string(content[bodyStart:bodyEnd])
	if versionLinePattern.MatchString(body) {
		replaced := versionLinePattern.ReplaceAllString(body, "${1}version: "+newVersion)
		return joinFrontmatter(content, bodyStart, bodyEnd, replaced), nil
	}

	inserted := body
	if inserted != "" && !strings.HasSuffix(inserted, lineEnding) {
		inserted += lineEnding
	}
	inserted += "version: " + newVersion + lineEnding
	return joinFrontmatter(content, bodyStart, bodyEnd, inserted), nil
}

func joinFrontmatter(content []byte, bodyStart, bodyEnd int, updated string) []byte {
	result := make([]byte, 0, len(content)+len(updated)-(bodyEnd-bodyStart))
	result = append(result, content[:bodyStart]...)
	result = append(result, updated...)
	result = append(result, content[bodyEnd:]...)
	return result
}

func frontmatterBodyRange(content []byte) (int, int, string, error) {
	line, next, ending := readLine(content, 0)
	if strings.TrimSpace(line) != "---" {
		return 0, 0, "", fmt.Errorf("delimitador inicial ausente")
	}

	if ending == "" {
		ending = "\n"
	}

	for pos := next; pos < len(content); {
		currentLine, nextPos, _ := readLine(content, pos)
		if strings.TrimSpace(currentLine) == "---" {
			return next, pos, ending, nil
		}
		pos = nextPos
	}

	return 0, 0, "", fmt.Errorf("delimitador final ausente")
}

func validateFrontmatter(content []byte) error {
	bodyStart, bodyEnd, _, err := frontmatterBodyRange(content)
	if err != nil {
		return err
	}

	var frontmatter map[string]any
	if err := yaml.Unmarshal(content[bodyStart:bodyEnd], &frontmatter); err != nil {
		return fmt.Errorf("parsear frontmatter: %w", err)
	}
	if frontmatter == nil {
		return fmt.Errorf("frontmatter deve ser um objeto YAML")
	}

	return nil
}

func readLine(content []byte, start int) (string, int, string) {
	if start >= len(content) {
		return "", len(content), ""
	}

	remaining := content[start:]
	end := bytes.IndexByte(remaining, '\n')
	if end < 0 {
		line := bytes.TrimSuffix(remaining, []byte("\r"))
		return string(line), len(content), ""
	}

	lineEnd := start + end + 1
	line := remaining[:end]
	line = bytes.TrimSuffix(line, []byte("\r"))
	if end > 0 && remaining[end-1] == '\r' {
		return string(line), lineEnd, "\r\n"
	}

	return string(line), lineEnd, "\n"
}
