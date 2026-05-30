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
	Triggers    []string
	Lang        string
	LinkMode    string
	DependsOn   []string
	MaxDepth    int
}

// ParseFrontmatterFields extrai todos os campos do bloco frontmatter YAML como um mapa
// de chave → valor. Campos de nível raiz são mapeados diretamente (ex.: "name" → "claude").
// Campos de blocos indentados são mapeados via dot-notation (ex.: "runtime.ide" → "claude").
// Apenas blocos de um nível de indentação são suportados (ex.: "runtime.ide", não "a.b.c").
// A função é agnóstica ao esquema — serve tanto para SKILL.md quanto para AGENT.md.
func (catalog *Catalog) ParseFrontmatterFields(content []byte) map[string]string {
	fields := make(map[string]string)
	scanner := bufio.NewScanner(strings.NewReader(string(content)))
	inFrontmatter := false
	currentBlock := "" // prefixo do bloco indentado atual (ex.: "runtime")

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

		// Linha indentada: pertence ao bloco atual.
		if currentBlock != "" && len(line) > 0 && (line[0] == ' ' || line[0] == '\t') {
			trimmed := strings.TrimSpace(line)
			if idx := strings.Index(trimmed, ":"); idx > 0 {
				key := strings.TrimSpace(trimmed[:idx])
				val := strings.TrimSpace(trimmed[idx+1:])
				fields[currentBlock+"."+key] = val
			}
			continue
		}

		// Linha de nível raiz: encerra qualquer bloco indentado anterior.
		currentBlock = ""

		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}

		idx := strings.Index(trimmed, ":")
		if idx <= 0 {
			continue
		}

		key := strings.TrimSpace(trimmed[:idx])
		val := strings.TrimSpace(trimmed[idx+1:])

		if val == "" {
			// Valor vazio: início de um bloco indentado.
			currentBlock = key
			continue
		}

		fields[key] = val
	}
	return fields
}

// ParseFrontmatter extrai version e description do frontmatter YAML de um SKILL.md.
// É um wrapper fino sobre ParseFrontmatterFields que mapeia o resultado para a struct Frontmatter.
func (catalog *Catalog) ParseFrontmatter(content []byte) Frontmatter {
	fields := NewCatalog().ParseFrontmatterFields(content)
	var fm Frontmatter

	fm.Name = fields["name"]
	fm.Version = fields["version"]
	fm.Description = fields["description"]
	fm.Lang = fields["lang"]
	fm.LinkMode = fields["link_mode"]

	if v, ok := fields["triggers"]; ok {
		fm.Triggers = NewCatalog().parseDependsOn(v)
	}
	if v, ok := fields["depends_on"]; ok {
		fm.DependsOn = NewCatalog().parseDependsOn(v)
	}
	if v, ok := fields["max_depth"]; ok {
		fm.MaxDepth, _ = strconv.Atoi(v)
	}

	return fm
}

// ParseFrontmatterName extrai o campo name do frontmatter YAML.
func (catalog *Catalog) ParseFrontmatterName(content []byte) string {
	return NewCatalog().ParseFrontmatter(content).Name
}

// SemverGreater retorna true se a > b usando comparacao semver simples.
func (catalog *Catalog) SemverGreater(a, b string) bool {
	if a == b {
		return false
	}
	aParts := NewCatalog().parseSemver(a)
	bParts := NewCatalog().parseSemver(b)

	for i := range 3 {
		if aParts[i] > bParts[i] {
			return true
		}
		if aParts[i] < bParts[i] {
			return false
		}
	}
	return false
}

func (catalog *Catalog) parseSemver(v string) [3]int {
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
func (catalog *Catalog) ValidateFrontmatter(content []byte, dirName string, availableSkills []string) error {
	if !NewCatalog().hasFrontmatterBlock(content) {
		return fmt.Errorf("frontmatter ausente: o arquivo nao possui bloco ---...---")
	}

	fm := NewCatalog().ParseFrontmatter(content)

	if fm.Description == "" {
		return fmt.Errorf("campo description e obrigatorio no frontmatter")
	}

	if fm.Version != "" && !NewCatalog().IsValidSemver(fm.Version) {
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
func (catalog *Catalog) hasFrontmatterBlock(content []byte) bool {
	count := 0
	for line := range strings.SplitSeq(string(content), "\n") {
		if strings.TrimSpace(line) == "---" {
			count++
			if count >= 2 {
				return true
			}
		}
	}
	return false
}

// IsValidSemver verifica se a string e um semver valido no formato X.Y.Z com prefixo v opcional.
func (catalog *Catalog) IsValidSemver(v string) bool {
	v = strings.TrimPrefix(v, "v")
	if core, prerelease, hasPrerelease := strings.Cut(v, "-"); hasPrerelease {
		if !NewCatalog().isValidPrerelease(prerelease) {
			return false
		}
		v = core
	}
	parts := strings.Split(v, ".")
	if len(parts) != 3 {
		return false
	}
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
	return true
}

func (catalog *Catalog) isValidPrerelease(v string) bool {
	if v == "" {
		return false
	}

	for part := range strings.SplitSeq(v, ".") {
		if part == "" {
			return false
		}
		for _, c := range part {
			switch {
			case c >= '0' && c <= '9':
			case c >= 'A' && c <= 'Z':
			case c >= 'a' && c <= 'z':
			case c == '-':
			default:
				return false
			}
		}
	}

	return true
}

func (catalog *Catalog) parseDependsOn(raw string) []string {
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
