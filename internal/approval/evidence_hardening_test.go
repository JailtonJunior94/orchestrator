package approval

import (
	"testing"

	"github.com/stretchr/testify/require"
)

var adversarialEvidenceLines = []struct {
	name     string
	evidence string
	accepted bool
}{
	{"git diff sem saida verificavel", "git diff -> muitas mudancas", false},
	{"lint com saida generica", "golangci-lint run -> zero issues", false},
	{"pytest sem alvo", "pytest -> 12 passed", false},
	{"contagem nua sem contexto", "go test ./x -> 3", false},
	{"script com prosa", "./run.sh -> saiu bem", false},
	{"comando sem argumento", "make -> pass", false},
	{"prosa disfarcada de comando", "frobnicate tudo -> saida qualquer", false},
	{"registro trivial", "make build -> ok", false},
	{"go test com pacote e duracao", "go test ./internal/sample -count=1 -> ok internal/sample 0.512s", true},
	{"go vet com exit code", "go vet ./... -> exit 0", true},
	{"teste com resultado canonico", "TestFoo -> pass", true},
	{"referencia de arquivo no diff", "internal/a.go:4", true},
	{"pytest com alvo e contagem", "pytest tests/unit -> 12 passed in 0.4s", true},
	{"lint com contagem de issues", "golangci-lint run -> 0 issues", true},
}

func TestEvidenceFormRejectsFabricableLinesAndKeepsHonestOnes(t *testing.T) {
	for _, tc := range adversarialEvidenceLines {
		t.Run(tc.name, func(t *testing.T) {
			criteriaMap := parseWithDiff(t, tc.evidence)
			require.Equal(t, tc.accepted, criteriaMap.Complete())
			if tc.accepted {
				require.Empty(t, criteriaMap.FindingsList())
				return
			}
			require.Len(t, criteriaMap.FindingsList(), 1)
			require.True(t, criteriaMap.FindingsList()[0].Severity().Blocks())
		})
	}
}

func TestFileLineEvidenceIsRejectedWhenTheTargetIsNotAStructuredDiff(t *testing.T) {
	targets := []struct {
		name   string
		target string
	}{
		{"prosa citando a referencia", "revisei os arquivos: internal/inventado.go:999 e outros"},
		{"lista de arquivos", "internal/inventado.go:999\ninternal/outro.go"},
		{"alvo vazio", ""},
	}

	for _, tc := range targets {
		t.Run(tc.name, func(t *testing.T) {
			raw := "## Mapa de Critérios de Aceite\n- [atendido] Criterio um -> internal/inventado.go:999\n"
			criteriaMap, err := ParseCriteriaMap(raw, criteriaRequest(t, tc.target, "Criterio um"))
			require.NoError(t, err)
			require.False(t, criteriaMap.Complete())
			require.Len(t, criteriaMap.FindingsList(), 1)
			require.True(t, criteriaMap.FindingsList()[0].Severity().Blocks())
		})
	}
}
