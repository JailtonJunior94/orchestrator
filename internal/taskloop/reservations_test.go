package taskloop

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/JailtonJunior94/ai-spec-harness/internal/fs"
)

type stubPrompter struct {
	responses []promptResponse
	calls     int
}

type promptResponse struct {
	action    ResolutionAction
	rationale string
	err       error
}

func (s *stubPrompter) Ask(_ context.Context, _ Finding) (ResolutionAction, string, error) {
	if s.calls >= len(s.responses) {
		return "", "", errors.New("stub: sem resposta configurada")
	}
	r := s.responses[s.calls]
	s.calls++
	return r.action, r.rationale, r.err
}

func sampleFindings() []Finding {
	return []Finding{
		{Severity: SeverityImportant, File: "a.go", Line: 10, Message: "[Important] revisar X"},
		{Severity: SeveritySuggestion, File: "b.go", Line: 20, Message: "[Suggestion] simplificar Y"},
		{Severity: SeverityImportant, File: "c.go", Line: 30, Message: "[Important] cobrir Z"},
	}
}

func TestReservationPlanner_Resolve(t *testing.T) {
	tests := []struct {
		name           string
		nonInteractive bool
		responses      []promptResponse
		findings       []Finding
		wantActions    []ResolutionAction
		wantErrIs      error
	}{
		{
			name:        "implement_document_discard_validos",
			findings:    sampleFindings(),
			responses: []promptResponse{
				{action: ActionImplement, rationale: ""},
				{action: ActionDocument, rationale: "registrar como follow-up"},
				{action: ActionDiscard, rationale: "fora de escopo do PRD"},
			},
			wantActions: []ResolutionAction{ActionImplement, ActionDocument, ActionDiscard},
		},
		{
			name:           "modo_nao_interativo_assume_document",
			nonInteractive: true,
			findings:       sampleFindings(),
			wantActions:    []ResolutionAction{ActionDocument, ActionDocument, ActionDocument},
		},
		{
			name:     "discard_sem_rationale_retorna_erro",
			findings: sampleFindings()[:1],
			responses: []promptResponse{
				{action: ActionDiscard, rationale: "   "},
			},
			wantErrIs: ErrInvalidResolution,
		},
		{
			name:     "acao_desconhecida_retorna_erro",
			findings: sampleFindings()[:1],
			responses: []promptResponse{
				{action: ResolutionAction("Foo"), rationale: ""},
			},
			wantErrIs: ErrInvalidResolution,
		},
		{
			name:        "lista_vazia_retorna_plano_vazio",
			findings:    nil,
			wantActions: nil,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			prompter := &stubPrompter{responses: tc.responses}
			planner := NewReservationPlanner(prompter, tc.nonInteractive)

			plan, err := planner.Resolve(context.Background(), tc.findings)
			if tc.wantErrIs != nil {
				if !errors.Is(err, tc.wantErrIs) {
					t.Fatalf("erro esperado %v, obtido %v", tc.wantErrIs, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("erro inesperado: %v", err)
			}
			if len(plan.Decisions) != len(tc.wantActions) {
				t.Fatalf("len decisoes=%d, esperado %d", len(plan.Decisions), len(tc.wantActions))
			}
			for i, want := range tc.wantActions {
				if plan.Decisions[i].Action != want {
					t.Errorf("decisao %d action=%q esperada %q", i, plan.Decisions[i].Action, want)
				}
			}
		})
	}
}

func TestReservationPlanner_PromptErrorPropaga(t *testing.T) {
	wantErr := errors.New("falha de IO")
	prompter := &stubPrompter{responses: []promptResponse{{err: wantErr}}}
	planner := NewReservationPlanner(prompter, false)

	_, err := planner.Resolve(context.Background(), sampleFindings()[:1])
	if !errors.Is(err, wantErr) {
		t.Fatalf("esperava wrap de %v, obtido %v", wantErr, err)
	}
}

func TestReservationPlanner_ContextoCancelado(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	planner := NewReservationPlanner(&stubPrompter{}, false)

	_, err := planner.Resolve(ctx, sampleFindings())
	if err == nil || !errors.Is(err, context.Canceled) {
		t.Fatalf("esperava context.Canceled, obtido %v", err)
	}
}

func TestReservationPlanner_InterativoSemPrompter(t *testing.T) {
	planner := NewReservationPlanner(nil, false)
	_, err := planner.Resolve(context.Background(), sampleFindings()[:1])
	if !errors.Is(err, ErrInvalidResolution) {
		t.Fatalf("esperava ErrInvalidResolution, obtido %v", err)
	}
}

func TestWriteActionPlanToTaskFile_Idempotente(t *testing.T) {
	fsys := fs.NewFakeFileSystem()
	taskFile := "/tasks/task-x.md"
	original := "# Task X\n\nDescrição inicial.\n"
	fsys.Files[taskFile] = []byte(original)

	plan := ActionPlan{Decisions: []ReservationDecision{
		{Finding: Finding{File: "a.go", Line: 10, Message: "[Important] revisar X"}, Action: ActionImplement},
		{Finding: Finding{File: "b.go", Line: 20, Message: "[Suggestion] simplificar Y"}, Action: ActionDocument, Rationale: "follow-up"},
	}}

	if err := WriteActionPlanToTaskFile(fsys, taskFile, plan); err != nil {
		t.Fatalf("primeira escrita: %v", err)
	}
	first := string(fsys.Files[taskFile])
	if !strings.Contains(first, "## Plano de Ação") {
		t.Fatalf("bloco de plano nao escrito: %s", first)
	}
	if !strings.Contains(first, "[Implement]") || !strings.Contains(first, "[Document]") {
		t.Fatalf("acoes nao formatadas: %s", first)
	}

	if err := WriteActionPlanToTaskFile(fsys, taskFile, plan); err != nil {
		t.Fatalf("segunda escrita: %v", err)
	}
	second := string(fsys.Files[taskFile])
	if first != second {
		t.Fatalf("nao idempotente:\nprimeira:\n%s\nsegunda:\n%s", first, second)
	}
	if !strings.HasPrefix(second, "# Task X") {
		t.Fatalf("conteudo original perdido: %s", second)
	}
}

func TestAppendFollowUpTasks_DocumentCriaLinha(t *testing.T) {
	fsys := fs.NewFakeFileSystem()
	tasksFile := "/prd/tasks.md"
	fsys.Files[tasksFile] = []byte(`# tasks

| # | Título | Status | Dependências | Paralelizável |
|---|--------|--------|--------------|---------------|
| 1.0 | Primeira | done | — | Não |
| 2.0 | Segunda | done | — | Não |

## Outra Seção

texto.
`)

	plan := ActionPlan{Decisions: []ReservationDecision{
		{Finding: Finding{File: "a.go", Line: 10, Message: "[Important] revisar X"}, Action: ActionImplement},
		{Finding: Finding{File: "b.go", Line: 20, Message: "[Suggestion] simplificar Y"}, Action: ActionDocument, Rationale: "registrar"},
		{Finding: Finding{File: "c.go", Message: "[Important] cobrir Z"}, Action: ActionDocument, Rationale: "registrar"},
		{Finding: Finding{File: "d.go", Message: "[Suggestion] descartar"}, Action: ActionDiscard, Rationale: "fora do escopo"},
	}}

	if err := AppendFollowUpTasks(fsys, tasksFile, plan); err != nil {
		t.Fatalf("erro: %v", err)
	}

	updated := string(fsys.Files[tasksFile])
	tasks, err := ParseTasksFile(fsys.Files[tasksFile])
	if err != nil {
		t.Fatalf("parse: %v\n%s", err, updated)
	}
	ids := make([]string, 0, len(tasks))
	for _, t := range tasks {
		ids = append(ids, t.ID)
	}
	wantIDs := []string{"1.0", "2.0", "3.0", "4.0"}
	if strings.Join(ids, ",") != strings.Join(wantIDs, ",") {
		t.Fatalf("ids=%v esperado %v\n%s", ids, wantIDs, updated)
	}
	if !strings.Contains(updated, "Follow-up:") {
		t.Fatalf("titulo follow-up ausente: %s", updated)
	}
	if !strings.Contains(updated, "## Outra Seção") {
		t.Fatalf("secao posterior preservada: %s", updated)
	}
}

func TestAppendFollowUpTasks_SemDocumentNaoModifica(t *testing.T) {
	fsys := fs.NewFakeFileSystem()
	tasksFile := "/prd/tasks.md"
	original := "| # | Título | Status | Dependências | Paralelizável |\n|---|---|---|---|---|\n| 1.0 | A | done | — | Não |\n"
	fsys.Files[tasksFile] = []byte(original)

	plan := ActionPlan{Decisions: []ReservationDecision{
		{Finding: Finding{File: "a.go"}, Action: ActionImplement},
	}}
	if err := AppendFollowUpTasks(fsys, tasksFile, plan); err != nil {
		t.Fatalf("erro: %v", err)
	}
	if string(fsys.Files[tasksFile]) != original {
		t.Fatalf("arquivo modificado sem decisoes Document")
	}
}

func TestOptions_NonInteractiveField(t *testing.T) {
	opts := Options{NonInteractive: true}
	if !opts.NonInteractive {
		t.Fatalf("campo NonInteractive nao persistido")
	}
}
