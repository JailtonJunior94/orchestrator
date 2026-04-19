package metrics

import (
	"math"

	tiktoken "github.com/pkoukk/tiktoken-go"
)

// Tokenizer estima a quantidade de tokens de um texto.
type Tokenizer interface {
	EstimateTokens(text string) int
	Name() string
}

// CharEstimator usa a heuristica chars/3.5 (default, sem dependencia externa).
type CharEstimator struct{}

func (CharEstimator) EstimateTokens(text string) int {
	return int(math.Round(float64(len(text)) / 3.5))
}

func (CharEstimator) Name() string { return "chars/3.5" }

// NewCharEstimator retorna o estimador baseado em caracteres.
func NewCharEstimator() Tokenizer { return CharEstimator{} }

// TiktokenEstimator usa tiktoken cl100k_base para contagem precisa de tokens.
type TiktokenEstimator struct {
	enc *tiktoken.Tiktoken
}

// NewTiktokenEstimator carrega o encoding cl100k_base e retorna um TiktokenEstimator.
// Retorna erro se o modelo BPE nao puder ser carregado.
func NewTiktokenEstimator() (Tokenizer, error) {
	enc, err := tiktoken.GetEncoding("cl100k_base")
	if err != nil {
		return nil, err
	}
	return &TiktokenEstimator{enc: enc}, nil
}

func (t *TiktokenEstimator) EstimateTokens(text string) int {
	return len(t.enc.Encode(text, nil, nil))
}

func (t *TiktokenEstimator) Name() string { return "tiktoken/cl100k_base" }

// NewPreciseTokenizer tenta criar um TiktokenEstimator e faz fallback para CharEstimator
// caso o modelo BPE nao possa ser carregado.
// Retorna o tokenizer e um bool indicando se tiktoken foi usado com sucesso.
func NewPreciseTokenizer() (Tokenizer, bool) {
	t, err := NewTiktokenEstimator()
	if err != nil {
		return NewCharEstimator(), false
	}
	return t, true
}
