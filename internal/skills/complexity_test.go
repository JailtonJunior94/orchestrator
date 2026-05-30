package skills

import (
	"math"
	"strings"
	"testing"

	"github.com/stretchr/testify/suite"
)

type ComplexitySuite struct {
	suite.Suite
}

func TestComplexitySuite(t *testing.T) {
	suite.Run(t, new(ComplexitySuite))
}

func (s *ComplexitySuite) TestClassify() {
	scenarios := []struct {
		description string
		want        Complexity
	}{
		// trivial — sem mudanca de comportamento
		{"rename variavel x para y", ComplexityTrivial},
		{"fix typo no comentario", ComplexityTrivial},
		{"adicionar import faltante", ComplexityTrivial},
		{"corrigir formatacao do arquivo", ComplexityTrivial},
		{"ajustar indentacao do bloco", ComplexityTrivial},
		{"organizar imports por grupo", ComplexityTrivial},
		{"ajuste cosmetico no arquivo de config", ComplexityTrivial},

		// standard — mudanca localizada sem keywords de complexidade
		{"adicionar novo metodo ao servico de usuarios", ComplexityStandard},
		{"corrigir bug no parser de flags CLI", ComplexityStandard},
		{"atualizar mensagem de erro de validacao", ComplexityStandard},
		{"adicionar campo opcional ao DTO", ComplexityStandard},
		{"implementar funcao auxiliar de logging", ComplexityStandard},

		// complex — mudanca transversal ou quebra de interface
		{"implementar nova interface de autenticacao", ComplexityComplex},
		{"breaking change na api publica do modulo", ComplexityComplex},
		{"migracao do banco de dados para nova versao", ComplexityComplex},
		{"adicionar nova feature de exportacao de relatorios", ComplexityComplex},
		{"refatorar arquitetura do servico de pagamento", ComplexityComplex},
		{"adicionar middleware de seguranca ao pipeline", ComplexityComplex},
		{"criar novo modulo de dependencia externa", ComplexityComplex},
		{"nova api rest para o endpoint de busca", ComplexityComplex},
	}

	for _, scenario := range scenarios {
		s.Run(scenario.description, func() {
			got := NewCatalog().Classify(scenario.description)
			s.Equal(scenario.want, got, "NewCatalog().Classify(%q)", scenario.description)
		})
	}
}

func (s *ComplexitySuite) TestClassifyEmptyDescription() {
	// Descricao vazia deve retornar standard (conservador)
	got := NewCatalog().Classify("")
	s.Equal(ComplexityStandard, got, "NewCatalog().Classify(\"\")")
}

func (s *ComplexitySuite) TestClassifyComplexOverridesTrivial() {
	// Keywords de complexidade tem prioridade sobre triviais
	got := NewCatalog().Classify("refatorar formatacao da interface publica")
	s.Equal(ComplexityComplex, got, "keywords complexas devem ter prioridade")
}

func (s *ComplexitySuite) TestParseComplexity() {
	scenarios := []struct {
		name   string
		input  string
		want   Complexity
		wantOK bool
	}{
		{name: "deve aceitar trivial", input: "trivial", want: ComplexityTrivial, wantOK: true},
		{name: "deve aceitar standard", input: "standard", want: ComplexityStandard, wantOK: true},
		{name: "deve aceitar complex", input: "complex", want: ComplexityComplex, wantOK: true},
		{name: "deve aceitar trivial em maiusculas", input: "TRIVIAL", want: ComplexityTrivial, wantOK: true},
		{name: "deve aceitar standard em maiusculas", input: "STANDARD", want: ComplexityStandard, wantOK: true},
		{name: "deve aceitar complex capitalizado", input: "Complex", want: ComplexityComplex, wantOK: true},
		{name: "deve ignorar espacos", input: "  trivial  ", want: ComplexityTrivial, wantOK: true},
		{name: "deve rejeitar invalid", input: "invalid", want: "", wantOK: false},
		{name: "deve rejeitar vazio", input: "", want: "", wantOK: false},
		{name: "deve rejeitar medium", input: "medium", want: "", wantOK: false},
		{name: "deve rejeitar hard", input: "hard", want: "", wantOK: false},
	}

	for _, scenario := range scenarios {
		s.Run(scenario.name, func() {
			got, ok := NewCatalog().ParseComplexity(scenario.input)
			s.Equal(scenario.wantOK, ok, "NewCatalog().ParseComplexity(%q) ok", scenario.input)
			s.Equal(scenario.want, got, "NewCatalog().ParseComplexity(%q)", scenario.input)
		})
	}
}

func (s *ComplexitySuite) TestReferencesForComplexityOrdering() {
	trivialRefs := NewCatalog().ReferencesForComplexity(ComplexityTrivial)
	standardRefs := NewCatalog().ReferencesForComplexity(ComplexityStandard)
	complexRefs := NewCatalog().ReferencesForComplexity(ComplexityComplex)

	// trivial nao deve carregar nenhuma referencia
	s.Len(trivialRefs, 0, "trivial deve retornar 0 referencias")

	// trivial carrega menos que standard
	s.True(len(trivialRefs) < len(standardRefs), "trivial (%d refs) deve ter menos referencias que standard (%d refs)", len(trivialRefs), len(standardRefs))

	// standard carrega menos que complex
	s.True(len(standardRefs) < len(complexRefs), "standard (%d refs) deve ter menos referencias que complex (%d refs)", len(standardRefs), len(complexRefs))
}

func (s *ComplexitySuite) TestReferencesForComplexityStandardSubsetOfComplex() {
	standardRefs := NewCatalog().ReferencesForComplexity(ComplexityStandard)
	complexRefs := NewCatalog().ReferencesForComplexity(ComplexityComplex)

	complexSet := make(map[string]bool, len(complexRefs))
	for _, ref := range complexRefs {
		complexSet[ref] = true
	}

	for _, ref := range standardRefs {
		s.True(complexSet[ref], "referencia standard %q nao esta presente em complex — standard deve ser subconjunto de complex", ref)
	}
}

// estimateTokens usa a heuristica chars/3.5 para estimar tokens sem importar metrics.
func estimateTokens(text string) int {
	return int(math.Round(float64(len(text)) / 3.5))
}

// TestReferencesForComplexityTokenEconomy verifica que o nivel trivial carrega
// menos tokens do que standard, e standard menos que complex.
// Usa heuristica chars/3.5 (equivalente ao CharEstimator de internal/metrics).
func (s *ComplexitySuite) TestReferencesForComplexityTokenEconomy() {
	// Simular conteudo de AGENTS.md (~10.000 chars) + referencias (~2.000 chars cada)
	agentsMDChars := 10000
	refChars := 2000

	trivialRefs := NewCatalog().ReferencesForComplexity(ComplexityTrivial)
	standardRefs := NewCatalog().ReferencesForComplexity(ComplexityStandard)
	complexRefs := NewCatalog().ReferencesForComplexity(ComplexityComplex)

	trivialTotal := agentsMDChars + len(trivialRefs)*refChars
	standardTotal := agentsMDChars + len(standardRefs)*refChars
	complexTotal := agentsMDChars + len(complexRefs)*refChars

	trivialTokens := estimateTokens(strings.Repeat("a", trivialTotal))
	standardTokens := estimateTokens(strings.Repeat("a", standardTotal))
	complexTokens := estimateTokens(strings.Repeat("a", complexTotal))

	// trivial deve consumir menos tokens que standard
	s.True(trivialTokens < standardTokens, "trivial (%d tokens) deve consumir menos tokens que standard (%d tokens)", trivialTokens, standardTokens)

	// standard deve consumir menos tokens que complex
	s.True(standardTokens < complexTokens, "standard (%d tokens) deve consumir menos tokens que complex (%d tokens)", standardTokens, complexTokens)

	// Economia de trivial vs complex deve ser >= 15%
	economy := float64(complexTokens-trivialTokens) / float64(complexTokens) * 100
	s.True(economy >= 15, "economia de trivial vs complex deve ser >= 15%%, got %.1f%%", economy)

	s.T().Logf("economia de tokens: trivial=%d standard=%d complex=%d (economia trivial/complex=%.1f%%)",
		trivialTokens, standardTokens, complexTokens, economy)
}

// TestReferencesForComplexityUnknownFallsToComplex verifica que o comportamento default
// (switch default) equivale a ComplexityComplex, garantindo carregamento completo seguro.
func (s *ComplexitySuite) TestReferencesForComplexityUnknownFallsToComplex() {
	unknownRefs := NewCatalog().ReferencesForComplexity(Complexity("unknown"))
	complexRefs := NewCatalog().ReferencesForComplexity(ComplexityComplex)

	s.Len(unknownRefs, len(complexRefs), "valor desconhecido deve fazer fallback para complex")
}
