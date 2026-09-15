package evidence

import (
	"strings"
	"testing"
)

func reviewReportWith(reviewedSection, mapSection string) string {
	return `# Relatorio de Review
- Veredito: APPROVED
- Alvo revisado: diff
` + mapSection + `
## Achados
Sem achados.
## Arquivos Revisados
` + reviewedSection + `
## Riscos Residuais
- nenhum
## Validacoes Executadas
- go test ./... -> ok internal/evidence 0.4s
`
}

func reviewFindings(t *testing.T, reviewedSection, mapSection string) []Finding {
	t.Helper()
	return NewValidator().Validate([]byte(reviewReportWith(reviewedSection, mapSection)), KindReview, nil).Findings
}

func TestEvidenceFormRejectsFabricableLinesAndKeepsHonestOnes(t *testing.T) {
	cases := []struct {
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
		{"referencia de arquivo revisado", "internal/sample/foo.go:42", true},
		{"pytest com alvo e contagem", "pytest tests/unit -> 12 passed in 0.4s", true},
		{"lint com contagem de issues", "golangci-lint run -> 0 issues", true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			mapSection := "## Mapa de Criterios de Aceite\n- [atendido] Criterio um -> " + tc.evidence
			findings := reviewFindings(t, "- internal/sample/foo.go", mapSection)
			hasProblem := false
			for _, f := range findings {
				if strings.Contains(f.Label, tc.evidence) {
					hasProblem = true
				}
			}
			if hasProblem == tc.accepted {
				t.Fatalf("evidencia %q: aceita=%v, esperado aceita=%v (findings: %v)", tc.evidence, !hasProblem, tc.accepted, findings)
			}
		})
	}
}

func TestReviewedFilesSectionWithoutFilesFailsClosed(t *testing.T) {
	mapSection := "## Mapa de Criterios de Aceite\n- [atendido] Criterio um -> internal/inventado.go:999"
	findings := reviewFindings(t, "- nenhum arquivo relevante", mapSection)

	if !hasFindingContaining(findings, "sem nenhum arquivo listado") {
		t.Fatalf("esperado finding de secao Arquivos Revisados vazia, got %v", findings)
	}
	if !hasFindingContaining(findings, "fora dos arquivos revisados") {
		t.Fatalf("esperado reprovar referencia inventada com secao vazia, got %v", findings)
	}
}

func TestReferencedFileIsComparedByPathNotBasename(t *testing.T) {
	cases := []struct {
		name     string
		reviewed string
		evidence string
		accepted bool
	}{
		{"caminho identico", "- internal/evidence/form.go", "internal/evidence/form.go:22", true},
		{"caminho declarado com prefixo relativo", "- ./internal/evidence/form.go", "internal/evidence/form.go:22", true},
		{"basename declarado e caminho completo na evidencia", "- form.go", "internal/evidence/form.go:22", true},
		{"mesmo basename em diretorio diferente", "- internal/evidence/form.go", "internal/inventado/form.go:22", false},
		{"arquivo ausente da secao", "- internal/evidence/form.go", "internal/inventado.go:999", false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			mapSection := "## Mapa de Criterios de Aceite\n- [atendido] Criterio um -> " + tc.evidence
			findings := reviewFindings(t, tc.reviewed, mapSection)
			rejected := hasFindingContaining(findings, "fora dos arquivos revisados")
			if rejected == tc.accepted {
				t.Fatalf("evidencia %q contra %q: aceita=%v, esperado aceita=%v (findings: %v)", tc.evidence, tc.reviewed, !rejected, tc.accepted, findings)
			}
		})
	}
}
