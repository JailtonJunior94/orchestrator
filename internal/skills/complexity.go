package skills

import "strings"

// Complexity representa o nivel de complexidade de uma tarefa.
// Determina quais referencias de governanca sao carregadas no ciclo de execucao,
// reduzindo consumo de tokens em tarefas que nao requerem carregamento completo.
//
// Economia estimada por ciclo trivial: ~2.500 tokens (referencias ignoradas).
// Reducao potencial: 15-25% por ciclo trivial versus carregamento completo.
type Complexity string

const (
	// ComplexityTrivial representa tarefas sem mudanca de comportamento:
	// rename de variavel, correcao de typo, adicao de import, formatacao.
	// Carregamento: apenas AGENTS.md; todas as referencias sao ignoradas.
	ComplexityTrivial Complexity = "trivial"

	// ComplexityStandard representa mudancas localizadas com testes:
	// novo metodo, fix de bug, refactor local.
	// Carregamento: AGENTS.md + referencias relevantes ao tipo de mudanca
	// (error-handling, testing).
	ComplexityStandard Complexity = "standard"

	// ComplexityComplex representa mudancas transversais:
	// nova feature, mudanca de interface publica, migracao.
	// Carregamento: tudo — AGENTS.md + todas as referencias (comportamento atual).
	ComplexityComplex Complexity = "complex"
)

// trivialKeywords indica descricoes de tarefas triviais (sem mudanca de comportamento).
var trivialKeywords = []string{
	"rename", "typo", "import", "formatacao", "format", "whitespace",
	"espacamento", "indentacao", "indent", "indent", "comentario", "comment",
	"organizar", "reorganizar", "reorder", "sort", "ordenar", "ajustar",
	"ajuste", "cosmetic", "cosmetico", "lint-fix", "lintfix",
}

// complexKeywords indica descricoes que exigem carregamento completo de referencias.
var complexKeywords = []string{
	"interface", "public", "breaking", "migration", "migracao",
	"feature", "funcionalidade", "nova feature", "novo endpoint",
	"nova api", "new api", "refactor", "refatorar", "refatoracao",
	"arquitetura", "architecture", "modulo", "module", "dependencia",
	"dependency", "security", "seguranca", "autenticacao", "authentication",
	"middleware", "pipeline", "transversal", "cross-cutting", "cross cutting",
	"novo servico", "new service", "novo pacote", "new package",
}

// ParseComplexity converte uma string para Complexity.
// Retorna false se o valor nao for um dos tres niveis validos.
// Usado para validar o flag --complexity=<valor>.
func ParseComplexity(s string) (Complexity, bool) {
	switch Complexity(strings.ToLower(strings.TrimSpace(s))) {
	case ComplexityTrivial, ComplexityStandard, ComplexityComplex:
		return Complexity(strings.ToLower(strings.TrimSpace(s))), true
	}
	return "", false
}

// Classify classifica a complexidade de uma tarefa com base em heuristicas de keywords.
// A classificacao e conservadora: descricoes ambiguas ou ausentes retornam ComplexityStandard.
// Keywords de complexidade tem prioridade sobre keywords triviais.
func Classify(description string) Complexity {
	lower := strings.ToLower(description)

	for _, kw := range complexKeywords {
		if strings.Contains(lower, kw) {
			return ComplexityComplex
		}
	}
	for _, kw := range trivialKeywords {
		if strings.Contains(lower, kw) {
			return ComplexityTrivial
		}
	}

	return ComplexityStandard
}

// ReferencesForComplexity retorna a lista de referencias de governanca a carregar
// para o nivel de complexidade informado. Os caminhos sao relativos ao diretorio
// da skill agent-governance.
//
//   - trivial  → nenhuma referencia (apenas AGENTS.md e suficiente)
//   - standard → referencias de error-handling e testing (mudancas localizadas)
//   - complex  → todas as referencias (comportamento atual, sem restricao)
func ReferencesForComplexity(c Complexity) []string {
	switch c {
	case ComplexityTrivial:
		return []string{}
	case ComplexityStandard:
		return []string{
			"references/error-handling.md",
			"references/testing.md",
		}
	default: // ComplexityComplex
		return []string{
			"references/ddd.md",
			"references/error-handling.md",
			"references/security.md",
			"references/testing.md",
			"references/shared-lifecycle.md",
			"references/shared-testing.md",
			"references/shared-architecture.md",
			"references/shared-patterns.md",
		}
	}
}
