package skills

import (
	"math"
	"strings"
	"testing"
)

func TestClassify(t *testing.T) {
	cases := []struct {
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

	for _, tc := range cases {
		t.Run(tc.description, func(t *testing.T) {
			got := Classify(tc.description)
			if got != tc.want {
				t.Errorf("Classify(%q) = %q, want %q", tc.description, got, tc.want)
			}
		})
	}
}

func TestClassify_EmptyDescription(t *testing.T) {
	// Descricao vazia deve retornar standard (conservador)
	got := Classify("")
	if got != ComplexityStandard {
		t.Errorf("Classify(\"\") = %q, want %q", got, ComplexityStandard)
	}
}

func TestClassify_ComplexOverridesTrivial(t *testing.T) {
	// Keywords de complexidade tem prioridade sobre triviais
	got := Classify("refatorar formatacao da interface publica")
	if got != ComplexityComplex {
		t.Errorf("keywords complexas devem ter prioridade: got %q, want %q", got, ComplexityComplex)
	}
}

func TestParseComplexity(t *testing.T) {
	cases := []struct {
		input  string
		want   Complexity
		wantOK bool
	}{
		{"trivial", ComplexityTrivial, true},
		{"standard", ComplexityStandard, true},
		{"complex", ComplexityComplex, true},
		{"TRIVIAL", ComplexityTrivial, true},
		{"STANDARD", ComplexityStandard, true},
		{"Complex", ComplexityComplex, true},
		{"  trivial  ", ComplexityTrivial, true},
		{"invalid", "", false},
		{"", "", false},
		{"medium", "", false},
		{"hard", "", false},
	}

	for _, tc := range cases {
		t.Run(tc.input, func(t *testing.T) {
			got, ok := ParseComplexity(tc.input)
			if ok != tc.wantOK {
				t.Errorf("ParseComplexity(%q) ok=%v, want %v", tc.input, ok, tc.wantOK)
			}
			if got != tc.want {
				t.Errorf("ParseComplexity(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}

func TestReferencesForComplexity_Ordering(t *testing.T) {
	trivialRefs := ReferencesForComplexity(ComplexityTrivial)
	standardRefs := ReferencesForComplexity(ComplexityStandard)
	complexRefs := ReferencesForComplexity(ComplexityComplex)

	// trivial nao deve carregar nenhuma referencia
	if len(trivialRefs) != 0 {
		t.Errorf("trivial deve retornar 0 referencias, got %d: %v", len(trivialRefs), trivialRefs)
	}

	// trivial carrega menos que standard
	if len(trivialRefs) >= len(standardRefs) {
		t.Errorf("trivial (%d refs) deve ter menos referencias que standard (%d refs)", len(trivialRefs), len(standardRefs))
	}

	// standard carrega menos que complex
	if len(standardRefs) >= len(complexRefs) {
		t.Errorf("standard (%d refs) deve ter menos referencias que complex (%d refs)", len(standardRefs), len(complexRefs))
	}
}

func TestReferencesForComplexity_StandardSubsetOfComplex(t *testing.T) {
	standardRefs := ReferencesForComplexity(ComplexityStandard)
	complexRefs := ReferencesForComplexity(ComplexityComplex)

	complexSet := make(map[string]bool, len(complexRefs))
	for _, r := range complexRefs {
		complexSet[r] = true
	}

	for _, ref := range standardRefs {
		if !complexSet[ref] {
			t.Errorf("referencia standard %q nao esta presente em complex — standard deve ser subconjunto de complex", ref)
		}
	}
}

// estimateTokens usa a heuristica chars/3.5 para estimar tokens sem importar metrics.
func estimateTokens(text string) int {
	return int(math.Round(float64(len(text)) / 3.5))
}

// simulatedRefContent simula o conteudo de uma referencia com tamanho medio de 2000 chars.
func simulatedRefContent(name string) string {
	return strings.Repeat("a", 2000)
}

// TestReferencesForComplexity_TokenEconomy verifica que o nivel trivial carrega
// menos tokens do que standard, e standard menos que complex.
// Usa heuristica chars/3.5 (equivalente ao CharEstimator de internal/metrics).
func TestReferencesForComplexity_TokenEconomy(t *testing.T) {
	// Simular conteudo de AGENTS.md (~10.000 chars) + referencias (~2.000 chars cada)
	agentsMDChars := 10000
	refChars := 2000

	trivialRefs := ReferencesForComplexity(ComplexityTrivial)
	standardRefs := ReferencesForComplexity(ComplexityStandard)
	complexRefs := ReferencesForComplexity(ComplexityComplex)

	trivialTotal := agentsMDChars + len(trivialRefs)*refChars
	standardTotal := agentsMDChars + len(standardRefs)*refChars
	complexTotal := agentsMDChars + len(complexRefs)*refChars

	trivialTokens := estimateTokens(strings.Repeat("a", trivialTotal))
	standardTokens := estimateTokens(strings.Repeat("a", standardTotal))
	complexTokens := estimateTokens(strings.Repeat("a", complexTotal))

	// trivial deve consumir menos tokens que standard
	if trivialTokens >= standardTokens {
		t.Errorf("trivial (%d tokens) deve consumir menos tokens que standard (%d tokens)", trivialTokens, standardTokens)
	}

	// standard deve consumir menos tokens que complex
	if standardTokens >= complexTokens {
		t.Errorf("standard (%d tokens) deve consumir menos tokens que complex (%d tokens)", standardTokens, complexTokens)
	}

	// Economia de trivial vs complex deve ser >= 15%
	economy := float64(complexTokens-trivialTokens) / float64(complexTokens) * 100
	if economy < 15 {
		t.Errorf("economia de trivial vs complex deve ser >= 15%%, got %.1f%%", economy)
	}

	t.Logf("economia de tokens: trivial=%d standard=%d complex=%d (economia trivial/complex=%.1f%%)",
		trivialTokens, standardTokens, complexTokens, economy)
}

// TestReferencesForComplexity_UnknownFallsToComplex verifica que o comportamento default
// (switch default) equivale a ComplexityComplex, garantindo carregamento completo seguro.
func TestReferencesForComplexity_UnknownFallsToComplex(t *testing.T) {
	unknownRefs := ReferencesForComplexity(Complexity("unknown"))
	complexRefs := ReferencesForComplexity(ComplexityComplex)

	if len(unknownRefs) != len(complexRefs) {
		t.Errorf("valor desconhecido deve fazer fallback para complex: got %d refs, want %d refs", len(unknownRefs), len(complexRefs))
	}
}
