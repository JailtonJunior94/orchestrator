package taskloop

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/JailtonJunior94/ai-spec-harness/internal/fs"
)

// ErrAcceptanceFailed indica que a task nao cumpriu todos os criterios de aceite.
var ErrAcceptanceFailed = errors.New("taskloop: criterios de aceite nao atendidos")

// AcceptanceReport resume o resultado da validacao de uma task.
type AcceptanceReport struct {
	TaskID        string
	CriteriaMet   int
	CriteriaTotal int
	GoTestOutput  string
	GoVetOutput   string
	LintOutput    string
	Passed        bool
	MissingItems  []string
}

// AcceptanceGate valida que uma task cumpre 100% dos criterios de aceite
// e que os subprocessos go test, go vet e lint retornam exit 0.
type AcceptanceGate interface {
	Verify(ctx context.Context, task TaskEntry, taskFile string) (AcceptanceReport, error)
}

// CommandRunner executa um comando externo e retorna a saida combinada e o erro.
type CommandRunner interface {
	Run(ctx context.Context, name string, args ...string) (output string, err error)
}

type defaultAcceptanceGate struct {
	fsys   fs.FileSystem
	runner CommandRunner
}

// NewAcceptanceGate cria um AcceptanceGate com filesystem e runner fornecidos.
func NewAcceptanceGate(fsys fs.FileSystem, runner CommandRunner) AcceptanceGate {
	return &defaultAcceptanceGate{fsys: fsys, runner: runner}
}

// Verify valida os criterios de aceite da task e executa go test, go vet e lint.
func (g *defaultAcceptanceGate) Verify(ctx context.Context, task TaskEntry, taskFile string) (AcceptanceReport, error) {
	report := AcceptanceReport{TaskID: task.ID}

	data, err := g.fsys.ReadFile(taskFile)
	if err != nil {
		return report, fmt.Errorf("taskloop: erro ao ler arquivo da task %s: %w", taskFile, err)
	}

	criteria, missing := parseCriteriaFromTaskFile(data)
	report.CriteriaTotal = len(criteria)
	report.CriteriaMet = len(criteria) - len(missing)
	report.MissingItems = missing

	testOut, testErr := g.runner.Run(ctx, "go", "test", "./...")
	report.GoTestOutput = testOut

	vetOut, vetErr := g.runner.Run(ctx, "go", "vet", "./...")
	report.GoVetOutput = vetOut

	lintOut, lintErr := g.runner.Run(ctx, "golangci-lint", "run", "./...")
	report.LintOutput = lintOut

	var failureReasons []string
	if len(missing) > 0 {
		failureReasons = append(failureReasons, fmt.Sprintf("criterios incompletos: %s", strings.Join(missing, ", ")))
	}
	if testErr != nil {
		failureReasons = append(failureReasons, "go test falhou")
	}
	if vetErr != nil {
		failureReasons = append(failureReasons, "go vet falhou")
	}
	if lintErr != nil {
		failureReasons = append(failureReasons, "lint falhou")
	}

	if len(failureReasons) > 0 {
		return report, fmt.Errorf("%w: %s", ErrAcceptanceFailed, strings.Join(failureReasons, "; "))
	}

	report.Passed = true
	return report, nil
}

// parseCriteriaFromTaskFile extrai os criterios de aceite da secao "## Definition of Done"
// ou "## Criterios de Sucesso" do arquivo da task. Retorna todos os criterios e os nao marcados.
func parseCriteriaFromTaskFile(data []byte) (allCriteria []string, missing []string) {
	lines := strings.Split(string(data), "\n")

	inSection := false
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Detectar inicio da secao de criterios
		if isAcceptanceSection(trimmed) {
			inSection = true
			continue
		}

		// Parar na proxima secao de nivel equivalente
		if inSection && strings.HasPrefix(trimmed, "## ") && !isAcceptanceSection(trimmed) {
			break
		}

		if !inSection {
			continue
		}

		// Capturar itens de checklist: "- [ ] texto" ou "- [x] texto"
		if strings.HasPrefix(trimmed, "- [") {
			// item = "x] texto" ou " ] texto"
			item := trimmed[3:]
			if len(item) < 2 {
				continue
			}
			checked := item[0] == 'x' || item[0] == 'X'
			// Remover "] " ou "]"
			rest := item[1:]
			rest = strings.TrimPrefix(rest, "]")
			text := strings.TrimSpace(rest)
			if text == "" {
				continue
			}
			allCriteria = append(allCriteria, text)
			if !checked {
				missing = append(missing, text)
			}
		}
	}

	return allCriteria, missing
}

func isAcceptanceSection(line string) bool {
	lower := strings.ToLower(line)
	return strings.HasPrefix(lower, "## definition of done") ||
		strings.HasPrefix(lower, "## criterios de sucesso") ||
		strings.HasPrefix(lower, "## critérios de sucesso") ||
		strings.HasPrefix(lower, "## acceptance criteria")
}

