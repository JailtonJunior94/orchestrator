package taskcriteria_test

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/JailtonJunior94/ai-spec-harness/internal/taskcriteria"
)

const parityReportTemplate = `<!-- evidence-contract: v2 -->
# Relatório de Execução de Tarefa

## Tarefa
- ID: 1.0
- Arquivo: %s
- Estado: done

## Contexto Carregado
- PRD: (n/a)
- TechSpec: (n/a)

## Comandos Executados
- make test -> pass

## Arquivos Alterados
- internal/taskcriteria/extract.go

## Resultados de Validação
- Testes: pass
- Lint: pass
- Veredito do Revisor: APPROVED

## Suposições
- Nenhuma.

## Riscos Residuais
- Nenhum.
`

var shellCriteriaCountRe = regexp.MustCompile(`task define ([0-9]+) crit`)

func shellCriteriaCount(t *testing.T, taskContent string) (int, string) {
	t.Helper()

	validator, err := filepath.Abs(filepath.Join("..", "..", ".agents", "scripts", "validate-task-evidence.sh"))
	if err != nil {
		t.Fatalf("resolve validator: %v", err)
	}
	if _, statErr := os.Stat(validator); statErr != nil {
		t.Skipf("validator unavailable: %v", statErr)
	}

	dir := t.TempDir()
	taskPath := filepath.Join(dir, "task-1.0.md")
	if writeErr := os.WriteFile(taskPath, []byte(taskContent), 0o600); writeErr != nil {
		t.Fatalf("write task file: %v", writeErr)
	}
	reportPath := filepath.Join(dir, "1.0_execution_report.md")
	report := fmt.Sprintf(parityReportTemplate, "task-1.0.md")
	if writeErr := os.WriteFile(reportPath, []byte(report), 0o600); writeErr != nil {
		t.Fatalf("write report: %v", writeErr)
	}

	output, _ := exec.Command("bash", validator, reportPath).CombinedOutput()
	text := string(output)
	if match := shellCriteriaCountRe.FindStringSubmatch(text); match != nil {
		count, convErr := strconv.Atoi(match[1])
		if convErr != nil {
			t.Fatalf("parse shell count: %v", convErr)
		}
		return count, text
	}
	if strings.Contains(text, "não declara nenhum critério de aceite") {
		return 0, text
	}
	t.Fatalf("validator output does not expose the criteria count:\n%s", text)
	return -1, text
}

func TestGoAndShellAgreeOnTheCriteriaCount(t *testing.T) {
	if _, err := exec.LookPath("bash"); err != nil {
		t.Skip("bash unavailable")
	}

	cases := []struct {
		name string
		task string
	}{
		{"criterios de aceite com bullets simples", "# Tarefa\n\n## Critérios de Aceite\n\n- Primeiro criterio\n- Segundo criterio\n"},
		{"definition of done com checklist", "# Tarefa\n\n## Definition of Done\n\n- [ ] Primeiro\n- [x] Segundo\n"},
		{"acceptance criteria misto", "# Tarefa\n\n## Acceptance Criteria\n\n- [ ] Um\n- Dois\n- [x] Tres\n"},
		{"criterios de sucesso", "# Tarefa\n\n## Critérios de Sucesso\n\n- Evidência é validada.\n"},
		{"sem secao de criterios", "# Tarefa\n\n## Escopo\n\n- Nada a declarar.\n"},
		{"item de checklist vazio", "# Tarefa\n\n## Critérios de Aceite\n\n- [ ] \n"},
		{"secao encerrada pelo proximo heading", "# Tarefa\n\n## Critérios de Aceite\n\n- Um\n\n## Arquivos\n\n- ignorado.go\n"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			goCount := len(taskcriteria.Extract([]byte(tc.task)))
			shellCount, output := shellCriteriaCount(t, tc.task)
			if goCount != shellCount {
				t.Fatalf("contagem divergente: go=%d shell=%d\noutput:\n%s", goCount, shellCount, output)
			}
		})
	}
}
