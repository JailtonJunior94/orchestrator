package skills

import (
	"bufio"
	"fmt"
	"strconv"
	"strings"
)

// Frontmatter contem campos extraidos do cabecalho YAML de um SKILL.md.
type Frontmatter struct {
	Name        string
	Version     string
	Description string
	DependsOn   []string
	MaxDepth    int
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
		if strings.HasPrefix(line, "name:") {
			fm.Name = strings.TrimSpace(strings.TrimPrefix(line, "name:"))
		}
		if strings.HasPrefix(line, "description:") {
			fm.Description = strings.TrimSpace(strings.TrimPrefix(line, "description:"))
		}
		if strings.HasPrefix(line, "depends_on:") {
			fm.DependsOn = parseDependsOn(strings.TrimSpace(strings.TrimPrefix(line, "depends_on:")))
		}
		if strings.HasPrefix(line, "max_depth:") {
			val := strings.TrimSpace(strings.TrimPrefix(line, "max_depth:"))
			fm.MaxDepth, _ = strconv.Atoi(val)
		}
	}
	return fm
}

// ParseFrontmatterName extrai o campo name do frontmatter YAML.
func ParseFrontmatterName(content []byte) string {
	return ParseFrontmatter(content).Name
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

// ValidateFrontmatter valida os campos obrigatorios do frontmatter de um SKILL.md.
// dirName e o nome do diretorio da skill (vazio para nao verificar name).
// availableSkills e a lista de skills disponiveis (nil para nao verificar depends_on).
func ValidateFrontmatter(content []byte, dirName string, availableSkills []string) error {
	if !hasFrontmatterBlock(content) {
		return fmt.Errorf("frontmatter ausente: o arquivo nao possui bloco ---...---")
	}

	fm := ParseFrontmatter(content)

	if fm.Description == "" {
		return fmt.Errorf("campo description e obrigatorio no frontmatter")
	}

	if fm.Version != "" && !isValidSemver(fm.Version) {
		return fmt.Errorf("version %q nao e um semver valido (esperado: X.Y.Z)", fm.Version)
	}

	if dirName != "" && fm.Name != "" && fm.Name != dirName {
		return fmt.Errorf("campo name %q diverge do nome do diretorio %q", fm.Name, dirName)
	}

	if len(availableSkills) > 0 && len(fm.DependsOn) > 0 {
		available := make(map[string]bool, len(availableSkills))
		for _, s := range availableSkills {
			available[s] = true
		}
		for _, dep := range fm.DependsOn {
			if !available[dep] {
				return fmt.Errorf("depends_on referencia skill inexistente: %s", dep)
			}
		}
	}

	return nil
}

// hasFrontmatterBlock verifica se o conteudo possui um bloco ---...--- valido.
func hasFrontmatterBlock(content []byte) bool {
	count := 0
	for _, line := range strings.Split(string(content), "\n") {
		if strings.TrimSpace(line) == "---" {
			count++
			if count >= 2 {
				return true
			}
		}
	}
	return false
}

// isValidSemver verifica se a string e um semver valido (X, X.Y ou X.Y.Z com prefixo v opcional).
func isValidSemver(v string) bool {
	v = strings.TrimPrefix(v, "v")
	if idx := strings.IndexByte(v, '-'); idx >= 0 {
		v = v[:idx]
	}
	parts := strings.SplitN(v, ".", 3)
	for _, p := range parts {
		if p == "" {
			return false
		}
		for _, c := range p {
			if c < '0' || c > '9' {
				return false
			}
		}
	}
	return len(parts) >= 1
}

func parseDependsOn(raw string) []string {
	raw = strings.TrimSpace(raw)
	raw = strings.TrimPrefix(raw, "[")
	raw = strings.TrimSuffix(raw, "]")
	if raw == "" {
		return nil
	}

	parts := strings.Split(raw, ",")
	deps := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		part = strings.Trim(part, `"'`)
		if part == "" {
			continue
		}
		deps = append(deps, part)
	}
	return deps
}
