package taskloop

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/JailtonJunior94/ai-spec-harness/internal/fs"
)

// stubRunner implementa CommandRunner para testes.
type stubRunner struct {
	outputs map[string]string
	errors  map[string]error
}

func newStubRunner() *stubRunner {
	return &stubRunner{
		outputs: make(map[string]string),
		errors:  make(map[string]error),
	}
}

func (r *stubRunner) setResult(cmd string, out string, err error) {
	r.outputs[cmd] = out
	r.errors[cmd] = err
}

// Run usa "name subcmd" (ex: "go test", "go vet") como chave, com fallback para "name".
func (r *stubRunner) Run(_ context.Context, name string, args ...string) (string, error) {
	if len(args) > 0 {
		key := name + " " + args[0]
		if out, ok := r.outputs[key]; ok {
			return out, r.errors[key]
		}
	}
	return r.outputs[name], r.errors[name]
}

func makeTaskFile(definitionOfDone string) []byte {
	return []byte("# Tarefa\n\n## Visao Geral\n\nAlguma descricao.\n\n" + definitionOfDone + "\n\n## Outras Secoes\n\nConteudo.\n")
}

func TestAcceptanceGate_Verify(t *testing.T) {
	allChecked := makeTaskFile("## Definition of Done\n\n- [x] Codigo implementado.\n- [x] Testes escritos.\n- [x] go vet limpo.\n")
	someUnchecked := makeTaskFile("## Definition of Done\n\n- [x] Codigo implementado.\n- [ ] Testes escritos.\n- [ ] go vet limpo.\n")
	noCriteria := makeTaskFile("## Definition of Done\n\nNenhum item de checklist aqui.\n")
	ptBRSection := makeTaskFile("## Criterios de Sucesso\n\n- [x] Retorna Passed=true.\n- [x] Cobertura >= 90%.\n")

	tests := []struct {
		name         string
		taskContent  []byte
		runnerSetup  func(*stubRunner)
		wantPassed   bool
		wantErr      error
		wantMissing  []string
		wantTotal    int
		wantMet      int
	}{
		{
			name:        "sucesso pleno — todos criterios marcados e subprocessos ok",
			taskContent: allChecked,
			runnerSetup: func(r *stubRunner) {
				r.setResult("go test", "ok\n", nil)
				r.setResult("go vet", "", nil)
				r.setResult("golangci-lint", "", nil)
			},
			wantPassed: true,
			wantTotal:  3,
			wantMet:    3,
		},
		{
			name:        "criterios incompletos — retorna ErrAcceptanceFailed",
			taskContent: someUnchecked,
			runnerSetup: func(r *stubRunner) {
				r.setResult("go test", "ok\n", nil)
				r.setResult("go vet", "", nil)
				r.setResult("golangci-lint", "", nil)
			},
			wantPassed:  false,
			wantErr:     ErrAcceptanceFailed,
			wantMissing: []string{"Testes escritos.", "go vet limpo."},
			wantTotal:   3,
			wantMet:     1,
		},
		{
			name:        "go test falha — retorna ErrAcceptanceFailed",
			taskContent: allChecked,
			runnerSetup: func(r *stubRunner) {
				r.setResult("go test", "FAIL\n", errors.New("exit status 1"))
				r.setResult("go vet", "", nil)
				r.setResult("golangci-lint", "", nil)
			},
			wantPassed: false,
			wantErr:    ErrAcceptanceFailed,
			wantTotal:  3,
			wantMet:    3,
		},
		{
			name:        "go vet falha — retorna ErrAcceptanceFailed",
			taskContent: allChecked,
			runnerSetup: func(r *stubRunner) {
				r.setResult("go test", "ok\n", nil)
				r.setResult("go vet", "erro de vet\n", errors.New("exit status 1"))
				r.setResult("golangci-lint", "", nil)
			},
			wantPassed: false,
			wantErr:    ErrAcceptanceFailed,
			wantTotal:  3,
			wantMet:    3,
		},
		{
			name:        "lint falha — retorna ErrAcceptanceFailed",
			taskContent: allChecked,
			runnerSetup: func(r *stubRunner) {
				r.setResult("go test", "ok\n", nil)
				r.setResult("go vet", "", nil)
				r.setResult("golangci-lint", "issues found\n", errors.New("exit status 1"))
			},
			wantPassed: false,
			wantErr:    ErrAcceptanceFailed,
			wantTotal:  3,
			wantMet:    3,
		},
		{
			name:        "sem criterios de checklist — passa se subprocessos ok",
			taskContent: noCriteria,
			runnerSetup: func(r *stubRunner) {
				r.setResult("go test", "ok\n", nil)
				r.setResult("go vet", "", nil)
				r.setResult("golangci-lint", "", nil)
			},
			wantPassed: true,
			wantTotal:  0,
			wantMet:    0,
		},
		{
			name:        "secao em PT-BR Criterios de Sucesso",
			taskContent: ptBRSection,
			runnerSetup: func(r *stubRunner) {
				r.setResult("go test", "ok\n", nil)
				r.setResult("go vet", "", nil)
				r.setResult("golangci-lint", "", nil)
			},
			wantPassed: true,
			wantTotal:  2,
			wantMet:    2,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			fsys := fs.NewFakeFileSystem()
			taskPath := "prd/task-2.0-test.md"
			fsys.Files[taskPath] = tc.taskContent

			runner := newStubRunner()
			if tc.runnerSetup != nil {
				tc.runnerSetup(runner)
			}

			gate := NewAcceptanceGate(fsys, runner)
			entry := TaskEntry{ID: "2.0", Title: "Test Task", Status: "pending"}

			report, err := gate.Verify(context.Background(), entry, taskPath)

			if tc.wantErr != nil {
				if !errors.Is(err, tc.wantErr) {
					t.Errorf("Verify() erro = %v, esperado %v", err, tc.wantErr)
				}
			} else if err != nil {
				t.Errorf("Verify() erro inesperado = %v", err)
			}

			if report.Passed != tc.wantPassed {
				t.Errorf("Passed = %v, esperado %v", report.Passed, tc.wantPassed)
			}
			if report.CriteriaTotal != tc.wantTotal {
				t.Errorf("CriteriaTotal = %d, esperado %d", report.CriteriaTotal, tc.wantTotal)
			}
			if report.CriteriaMet != tc.wantMet {
				t.Errorf("CriteriaMet = %d, esperado %d", report.CriteriaMet, tc.wantMet)
			}
			if len(tc.wantMissing) > 0 {
				got := strings.Join(report.MissingItems, ",")
				want := strings.Join(tc.wantMissing, ",")
				if got != want {
					t.Errorf("MissingItems = %v, esperado %v", report.MissingItems, tc.wantMissing)
				}
			}
		})
	}
}

func TestParseCriteriaFromTaskFile(t *testing.T) {
	tests := []struct {
		name         string
		content      string
		wantTotal    int
		wantMissing  int
	}{
		{
			name:        "todos marcados",
			content:     "## Definition of Done\n\n- [x] Item A\n- [x] Item B\n",
			wantTotal:   2,
			wantMissing: 0,
		},
		{
			name:        "nenhum marcado",
			content:     "## Definition of Done\n\n- [ ] Item A\n- [ ] Item B\n",
			wantTotal:   2,
			wantMissing: 2,
		},
		{
			name:        "sem secao relevante",
			content:     "## Outra Secao\n\n- [ ] Item A\n",
			wantTotal:   0,
			wantMissing: 0,
		},
		{
			name:        "secao encerrada por proxima secao",
			content:     "## Definition of Done\n\n- [x] Item A\n\n## Proxima Secao\n\n- [ ] Item Ignorado\n",
			wantTotal:   1,
			wantMissing: 0,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			all, missing := parseCriteriaFromTaskFile([]byte(tc.content))
			if len(all) != tc.wantTotal {
				t.Errorf("total = %d, esperado %d", len(all), tc.wantTotal)
			}
			if len(missing) != tc.wantMissing {
				t.Errorf("missing = %d, esperado %d", len(missing), tc.wantMissing)
			}
		})
	}
}
