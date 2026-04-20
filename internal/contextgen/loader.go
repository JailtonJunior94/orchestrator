package contextgen

import (
	"regexp"
	"strings"

	"github.com/JailtonJunior94/ai-spec-harness/internal/fs"
)

// ComplexityLevel representa o nivel de complexidade da tarefa.
// Controla quais referencias sao carregadas no contexto gerado.
type ComplexityLevel string

const (
	// ComplexityTrivial carrega apenas AGENTS.md + SKILL.md (zero referencias).
	// Uso: tarefas simples como renomear variavel, ajuste de comentario.
	ComplexityTrivial ComplexityLevel = "trivial"

	// ComplexityStandard carrega SKILL.md + referencias de erro e teste.
	// Uso: implementacao de funcoes, correcao de bug, adicao de teste.
	ComplexityStandard ComplexityLevel = "standard"

	// ComplexityComplex carrega todas as referencias (comportamento atual).
	// Uso: refatoracao arquitetural, novo componente, mudanca de interface.
	ComplexityComplex ComplexityLevel = "complex"
)

// standardRefs sao as referencias carregadas no nivel standard.
// Baseado no mapeamento de agent-governance/SKILL.md.
var standardRefs = map[string]bool{
	"error-handling.md": true,
	"testing.md":        true,
}

// tldrRegexp captura o bloco <!-- TL;DR ... --> no inicio de referencias.
var tldrRegexp = regexp.MustCompile(`(?s)<!--\s*TL;DR\s*(.*?)-->`)

// LoadOptions configura o comportamento de carregamento de referencias.
type LoadOptions struct {
	// Brief carrega apenas o bloco TL;DR de cada referencia (modo economico).
	// Se a referencia nao tiver TL;DR, carrega completo (fallback seguro).
	Brief bool

	// Complexity filtra quais referencias sao carregadas com base no nivel.
	// Default: ComplexityComplex (retrocompativel).
	Complexity ComplexityLevel
}

// Loader carrega SKILL.md e referencias com filtro de complexidade e modo brief.
type Loader struct {
	fs fs.FileSystem
}

// NewLoader retorna um Loader com o filesystem fornecido.
func NewLoader(fsys fs.FileSystem) *Loader {
	return &Loader{fs: fsys}
}

// LoadReference carrega uma referencia com as opcoes fornecidas.
// Em modo brief, retorna apenas o bloco TL;DR (fallback para conteudo completo se ausente).
func (l *Loader) LoadReference(path string, opts LoadOptions) (string, error) {
	data, err := l.fs.ReadFile(path)
	if err != nil {
		return "", err
	}
	content := string(data)

	if opts.Brief {
		return l.extractTLDR(content), nil
	}
	return content, nil
}

// LoadSkillReferences carrega referencias de uma skill com base nas opcoes.
// Retorna mapa de nome-do-arquivo -> conteudo para as referencias aplicaveis.
func (l *Loader) LoadSkillReferences(skillDir string, opts LoadOptions) (map[string]string, error) {
	level := opts.Complexity
	if level == "" {
		level = ComplexityComplex
	}

	if level == ComplexityTrivial {
		return map[string]string{}, nil
	}

	refsDir := skillDir + "/references"
	entries, err := l.fs.ReadDir(refsDir)
	if err != nil {
		return map[string]string{}, nil
	}

	result := make(map[string]string)
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if !strings.HasSuffix(name, ".md") {
			continue
		}

		if level == ComplexityStandard && !standardRefs[name] {
			continue
		}

		content, err := l.LoadReference(refsDir+"/"+name, opts)
		if err != nil {
			continue
		}
		result[name] = content
	}

	return result, nil
}

// extractTLDR extrai o conteudo do bloco <!-- TL;DR ... -->.
// Se nao encontrado, retorna o conteudo completo (fallback seguro).
func (l *Loader) extractTLDR(content string) string {
	m := tldrRegexp.FindStringSubmatch(content)
	if m == nil {
		return content
	}
	return strings.TrimSpace(m[1])
}
