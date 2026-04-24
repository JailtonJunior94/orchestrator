package taskloop

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/JailtonJunior94/ai-spec-harness/internal/output"
)

func TestTextPresenterConsume(t *testing.T) {
	t.Parallel()

	startedAt := time.Date(2026, 4, 23, 12, 0, 0, 0, time.UTC)
	task, err := NewTaskRef("4.0", "Presenter textual")
	if err != nil {
		t.Fatalf("NewTaskRef() erro inesperado: %v", err)
	}

	tests := []struct {
		name         string
		events       []LoopEvent
		wantOut      []string
		wantErr      []string
		final        FinalSummary
		wantFinal    []string
		wantFinalErr []string
	}{
		{
			name: "renderiza iteracao e streaming com estado canonico",
			events: []LoopEvent{
				{Time: startedAt, Kind: EventSessionStarted},
				{Time: startedAt.Add(time.Second), Kind: EventIterationSelected, Iteration: 1, Task: task, Tool: ToolCodex, Role: RoleExecutor, Phase: PhasePreparing},
				{Time: startedAt.Add(2 * time.Second), Kind: EventPhaseChanged, Iteration: 1, Task: task, Tool: ToolCodex, Role: RoleExecutor, Phase: PhaseRunning, Message: "subprocesso iniciado"},
				{Time: startedAt.Add(3 * time.Second), Kind: EventOutputObserved, Iteration: 1, Task: task, Tool: ToolCodex, Role: RoleExecutor, Message: "primeira saida observada"},
			},
			wantOut: []string{
				"iteracao 1/4 task 4.0 (Presenter textual) tool=codex role=executor phase=preparando",
				"task 4.0 tool=codex role=executor phase=executando",
				"evento=primeira saida observada",
			},
		},
		{
			name: "renderiza heartbeat e falha tipada com hint acionavel",
			events: []LoopEvent{
				{Time: startedAt, Kind: EventSessionStarted},
				{Time: startedAt.Add(time.Second), Kind: EventIterationSelected, Iteration: 2, Task: task, Tool: ToolClaude, Role: RoleExecutor, Phase: PhasePreparing},
				{Time: startedAt.Add(2 * time.Second), Kind: EventPhaseChanged, Iteration: 2, Task: task, Tool: ToolClaude, Role: RoleExecutor, Phase: PhaseRunning},
				{Time: startedAt.Add(5 * time.Second), Kind: EventHeartbeatObserved, Iteration: 2, Task: task, Tool: ToolClaude, Role: RoleExecutor, Message: "atividade operacional em andamento"},
				{
					Time:      startedAt.Add(6 * time.Second),
					Kind:      EventFailureObserved,
					Iteration: 2,
					Task:      task,
					Tool:      ToolClaude,
					Role:      RoleExecutor,
					Phase:     PhaseAuthRequired,
					ErrorCode: ErrorToolAuthRequired,
					Message:   "claude requer autenticacao",
					Failure:   NewLoopFailure(ErrorToolAuthRequired, "claude requer autenticacao", nil),
				},
			},
			wantOut: []string{
				"elapsed=5s evento=atividade operacional em andamento",
			},
			wantErr: []string{
				"phase=autenticacao pendente",
				"falha=claude requer autenticacao; hint: verifique login, sessao local ou credenciais da ferramenta ativa",
			},
		},
		{
			name: "renderiza resumo final a partir do estado agregado",
			events: []LoopEvent{
				{Time: startedAt, Kind: EventSessionStarted},
				{Time: startedAt.Add(time.Second), Kind: EventIterationSelected, Iteration: 1, Task: task, Tool: ToolGemini, Role: RoleExecutor, Phase: PhasePreparing},
				{Time: startedAt.Add(2 * time.Second), Kind: EventPhaseChanged, Iteration: 1, Task: task, Tool: ToolGemini, Role: RoleExecutor, Phase: PhaseRunning},
				{Time: startedAt.Add(3 * time.Second), Kind: EventPhaseChanged, Iteration: 1, Task: task, Tool: ToolGemini, Role: RoleExecutor, Phase: PhaseDone, PostStatus: "done"},
			},
			final: FinalSummary{
				StopReason:    "todas as tasks completadas ou em estado terminal",
				IterationsRun: 1,
				ReportPath:    "report.md",
				Progress: BatchProgress{
					Total:   1,
					Done:    1,
					Pending: 0,
				},
			},
			wantFinal: []string{
				"resumo final: stop=todas as tasks completadas ou em estado terminal iteracoes=1 lote=done=1 failed=0 blocked=0 needs_input=0 pending=0 in_progress=0 total=1",
				"report=report.md",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			out := &bytes.Buffer{}
			errBuf := &bytes.Buffer{}
			printer := &output.Printer{Out: out, Err: errBuf}
			currentTime := startedAt

			session, err := NewLoopSession("plain", 4, false, 5, startedAt)
			if err != nil {
				t.Fatalf("NewLoopSession() erro inesperado: %v", err)
			}
			presenter := NewTextPresenter(printer, func() SessionSnapshot {
				return session.SnapshotAt(currentTime)
			})
			if err := presenter.Start(session.SnapshotAt(startedAt)); err != nil {
				t.Fatalf("Start() erro inesperado: %v", err)
			}

			for _, event := range tt.events {
				currentTime = event.Time
				if _, err := session.Apply(event); err != nil {
					t.Fatalf("Apply(%s) erro inesperado: %v", event.Kind, err)
				}
				if err := presenter.Consume(event); err != nil {
					t.Fatalf("Consume(%s) erro inesperado: %v", event.Kind, err)
				}
			}

			if (tt.final != FinalSummary{}) {
				if err := presenter.Finish(tt.final); err != nil {
					t.Fatalf("Finish() erro inesperado: %v", err)
				}
			}

			for _, want := range tt.wantOut {
				if !strings.Contains(out.String(), want) {
					t.Fatalf("stdout nao contem %q\nsaida:\n%s", want, out.String())
				}
			}
			for _, want := range tt.wantErr {
				if !strings.Contains(errBuf.String(), want) {
					t.Fatalf("stderr nao contem %q\nsaida:\n%s", want, errBuf.String())
				}
			}
			for _, want := range tt.wantFinal {
				if !strings.Contains(out.String(), want) {
					t.Fatalf("resumo final nao contem %q\nsaida:\n%s", want, out.String())
				}
			}
			for _, want := range tt.wantFinalErr {
				if !strings.Contains(errBuf.String(), want) {
					t.Fatalf("erro final nao contem %q\nsaida:\n%s", want, errBuf.String())
				}
			}
		})
	}
}
