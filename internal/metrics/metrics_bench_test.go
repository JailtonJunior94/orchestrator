package metrics

import (
	"strings"
	"testing"
)

func BenchmarkEstimateTokens_CharEstimator(b *testing.B) {
	content := strings.Repeat("exemplo de conteudo para estimativa de tokens ", 100)
	tok := NewCharEstimator()
	for b.Loop() {
		tok.EstimateTokens(content)
	}
}
