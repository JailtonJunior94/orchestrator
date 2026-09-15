package evidence

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

const reviewTwoCriteriaTaskFile = "testdata/task-review-dois-criterios.md"

const reviewNoCriteriaTaskFile = "testdata/task-review-sem-criterios.md"

func reviewReportForTask(taskFile, mapSection string) string {
	return `# Relatorio de Review
- Veredito: APPROVED
- Alvo revisado: diff
- Task file: ` + taskFile + `
` + mapSection + `
## Achados
Sem achados.
## Arquivos Revisados
- foo.go
## Riscos Residuais
- nenhum
## Validacoes Executadas
- go test ./... -> ok
`
}

func TestValidateReview_ConfrontaCompletudeContraTaskFile(t *testing.T) {
	completo := `## Mapa de Criterios de Aceite
- [atendido] Primeiro criterio da fixture -> go test ./... -> PASS
- [atendido] Segundo criterio da fixture -> internal/foo.go:42`

	incompleto := `## Mapa de Criterios de Aceite
- [atendido] Primeiro criterio da fixture -> go test ./... -> PASS`

	cases := []struct {
		name     string
		taskFile string
		mapping  string
		want     bool
		needle   string
	}{
		{name: "mapa cobre os dois criterios", taskFile: reviewTwoCriteriaTaskFile, mapping: completo, want: true},
		{name: "mapa cobre 1 de 2", taskFile: reviewTwoCriteriaTaskFile, mapping: incompleto, want: false, needle: "mapa 1:1 incompleto"},
		{name: "task file ausente", taskFile: "", mapping: completo, want: false, needle: "task file nao resolvivel"},
		{name: "task file inexistente", taskFile: "testdata/nao-existe.md", mapping: completo, want: false, needle: "task file nao resolvivel"},
		{name: "task file without declared criteria", taskFile: reviewNoCriteriaTaskFile, mapping: completo, want: false, needle: "nao declara nenhum criterio de aceite"},
		{name: "task file placeholder does not resolve", taskFile: "<caminho>", mapping: completo, want: false, needle: "task file nao resolvivel"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			content := reviewReportForTask(tc.taskFile, tc.mapping)
			result := NewValidator().Validate([]byte(content), KindReview, nil)
			if result.Pass != tc.want {
				t.Fatalf("Pass=%v, esperado %v; findings: %v", result.Pass, tc.want, result.Findings)
			}
			if tc.needle != "" {
				found := false
				for _, f := range result.Findings {
					if strings.Contains(f.Label, tc.needle) {
						found = true
					}
				}
				if !found {
					t.Fatalf("esperado finding contendo %q; findings: %v", tc.needle, result.Findings)
				}
			}

			shell := filepath.Join("..", "..", ".agents", "scripts", "validate-review-evidence.sh")
			if _, err := os.Stat(shell); err != nil {
				t.Skip("validador shell indisponivel")
			}
			reportPath := filepath.Join(t.TempDir(), "review.md")
			absoluteTask := tc.taskFile
			if absoluteTask != "" {
				if resolved, err := filepath.Abs(absoluteTask); err == nil {
					absoluteTask = resolved
				}
			}
			if err := os.WriteFile(reportPath, []byte(reviewReportForTask(absoluteTask, tc.mapping)), 0o644); err != nil {
				t.Fatal(err)
			}
			shellPass := exec.Command("bash", shell, reportPath).Run() == nil
			if shellPass != tc.want {
				t.Fatalf("validador shell pass=%v, esperado %v (paridade RF-52)", shellPass, tc.want)
			}
		})
	}
}
