package skills

import (
	"bufio"
	"strings"
)

// Frontmatter contem campos extraidos do cabecalho YAML de um SKILL.md.
type Frontmatter struct {
	Version     string
	Description string
}

// ParseFrontmatter extrai version e description do frontmatter YAML de um SKILL.md.
func ParseFrontmatter(content []byte) Frontmatter {
	var fm Frontmatter
	scanner := bufio.NewScanner(strings.NewReader(string(content)))
	inFrontmatter := false

	for scanner.Scan() {
		line := scanner.Text()
		if strings.TrimSpace(line) == "---" {
			if inFrontmatter {
				break
			}
			inFrontmatter = true
			continue
		}
		if !inFrontmatter {
			continue
		}
		if strings.HasPrefix(line, "version:") {
			fm.Version = strings.TrimSpace(strings.TrimPrefix(line, "version:"))
		}
		if strings.HasPrefix(line, "description:") {
			fm.Description = strings.TrimSpace(strings.TrimPrefix(line, "description:"))
		}
	}
	return fm
}

// SemverGreater retorna true se a > b usando comparacao semver simples.
func SemverGreater(a, b string) bool {
	if a == b {
		return false
	}
	aParts := parseSemver(a)
	bParts := parseSemver(b)

	for i := 0; i < 3; i++ {
		if aParts[i] > bParts[i] {
			return true
		}
		if aParts[i] < bParts[i] {
			return false
		}
	}
	return false
}

func parseSemver(v string) [3]int {
	var parts [3]int
	// Remover prefixo v se presente
	v = strings.TrimPrefix(v, "v")
	// Remover sufixo pre-release
	if idx := strings.IndexByte(v, '-'); idx >= 0 {
		v = v[:idx]
	}
	segments := strings.SplitN(v, ".", 3)
	for i, s := range segments {
		if i >= 3 {
			break
		}
		for _, c := range s {
			if c >= '0' && c <= '9' {
				parts[i] = parts[i]*10 + int(c-'0')
			} else {
				break
			}
		}
	}
	return parts
}
