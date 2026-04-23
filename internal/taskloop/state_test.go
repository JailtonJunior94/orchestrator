package taskloop

import (
	"errors"
	"testing"
	"time"
)

func TestAgentPhase(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		from AgentPhase
		to   AgentPhase
		want bool
	}{
		{
			name: "idle para preparing",
			from: PhaseIdle,
			to:   PhasePreparing,
			want: true,
		},
		{
			name: "preparing para running",
			from: PhasePreparing,
			to:   PhaseRunning,
			want: true,
		},
		{
			name: "running para streaming",
			from: PhaseRunning,
			to:   PhaseStreaming,
			want: true,
		},
		{
			name: "streaming para reviewing",
			from: PhaseStreaming,
			to:   PhaseReviewing,
			want: true,
		},
		{
			name: "reviewing para done",
			from: PhaseReviewing,
			to:   PhaseDone,
			want: true,
		},
		{
			name: "done nao volta para running",
			from: PhaseDone,
			to:   PhaseRunning,
			want: false,
		},
		{
			name: "idle nao vai direto para streaming",
			from: PhaseIdle,
			to:   PhaseStreaming,
			want: false,
		},
	}

	for _, tt := range testCases {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := tt.from.CanTransitionTo(tt.to)
			if got != tt.want {
				t.Fatalf("CanTransitionTo() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBatchProgress(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		progress BatchProgress
		wantErr  bool
	}{
		{
			name: "contadores consistentes",
			progress: BatchProgress{
				Total:      5,
				Done:       1,
				Failed:     1,
				Blocked:    1,
				NeedsInput: 0,
				Pending:    1,
				InProgress: 1,
			},
			wantErr: false,
		},
		{
			name: "contador negativo",
			progress: BatchProgress{
				Total:      1,
				Done:       -1,
				Failed:     0,
				Blocked:    0,
				NeedsInput: 0,
				Pending:    1,
				InProgress: 1,
			},
			wantErr: true,
		},
		{
			name: "soma inconsistente",
			progress: BatchProgress{
				Total:      2,
				Done:       1,
				Failed:     0,
				Blocked:    0,
				NeedsInput: 0,
				Pending:    0,
				InProgress: 0,
			},
			wantErr: true,
		},
	}

	for _, tt := range testCases {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := tt.progress.Validate()
			if tt.wantErr && err == nil {
				t.Fatal("Validate() deveria falhar")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("Validate() retornou erro inesperado: %v", err)
			}
		})
	}
}

func TestLoopFailure(t *testing.T) {
	t.Parallel()

	timeout := NewLoopFailure(ErrorToolTimeout, "", nil)
	if !errors.Is(timeout, ErrToolTimeout) {
		t.Fatal("errors.Is deveria reconhecer ErrToolTimeout")
	}

	target := &LoopFailure{Code: ErrorToolTimeout}
	if !errors.Is(timeout, target) {
		t.Fatal("errors.Is deveria reconhecer LoopFailure com mesmo codigo")
	}

	wrapped := NewLoopFailure(ErrorToolExecutionFailed, "falhou", ErrToolExecutionFailed)
	var typed *LoopFailure
	if !errors.As(wrapped, &typed) {
		t.Fatal("errors.As deveria extrair *LoopFailure")
	}
	if typed.Code != ErrorToolExecutionFailed {
		t.Fatalf("codigo = %q, want %q", typed.Code, ErrorToolExecutionFailed)
	}
}

func TestLoopSession(t *testing.T) {
	t.Parallel()

	startedAt := time.Date(2026, 4, 22, 18, 0, 0, 0, time.UTC)
	task, err := NewTaskRef("1.0", "Modelo de observabilidade")
	if err != nil {
		t.Fatalf("NewTaskRef() erro inesperado: %v", err)
	}

	t.Run("transicoes validas produzem snapshot determinista", func(t *testing.T) {
		t.Parallel()

		session, err := NewLoopSession("plain", 8, false, 3, startedAt)
		if err != nil {
			t.Fatalf("NewLoopSession() erro inesperado: %v", err)
		}

		progress, err := NewBatchProgress(3, 1, 0, 0, 0, 1, 1)
		if err != nil {
			t.Fatalf("NewBatchProgress() erro inesperado: %v", err)
		}

		events := []LoopEvent{
			{
				Time: startedAt,
				Kind: EventSessionStarted,
			},
			{
				Time:      startedAt.Add(1 * time.Second),
				Kind:      EventIterationSelected,
				Iteration: 1,
				Task:      task,
				Tool:      ToolCodex,
				Role:      RoleExecutor,
				PreStatus: "pending",
			},
			{
				Time:      startedAt.Add(2 * time.Second),
				Kind:      EventPhaseChanged,
				Iteration: 1,
				Tool:      ToolCodex,
				Role:      RoleExecutor,
				Phase:     PhaseRunning,
				Task:      task,
				Message:   "subprocesso iniciado",
			},
			{
				Time:      startedAt.Add(3 * time.Second),
				Kind:      EventOutputObserved,
				Iteration: 1,
				Tool:      ToolCodex,
				Role:      RoleExecutor,
				Task:      task,
				Message:   "primeira saida observada",
			},
			{
				Time:     startedAt.Add(4 * time.Second),
				Kind:     EventProgressUpdated,
				Progress: &progress,
				Message:  "progresso agregado atualizado",
			},
			{
				Time:       startedAt.Add(5 * time.Second),
				Kind:       EventPhaseChanged,
				Iteration:  1,
				Tool:       ToolCodex,
				Role:       RoleExecutor,
				Phase:      PhaseDone,
				Task:       task,
				PostStatus: "done",
				Message:    "iteracao concluida",
			},
		}

		var snapshot SessionSnapshot
		for _, event := range events {
			snapshot, err = session.Apply(event)
			if err != nil {
				t.Fatalf("Apply(%s) erro inesperado: %v", event.Kind, err)
			}
		}

		if snapshot.CurrentPhase != PhaseDone {
			t.Fatalf("fase atual = %q, want %q", snapshot.CurrentPhase, PhaseDone)
		}
		if snapshot.ActiveIteration == nil {
			t.Fatal("snapshot deveria conter iteracao ativa")
		}
		if snapshot.ActiveIteration.TaskID != "1.0" {
			t.Fatalf("task id = %q, want 1.0", snapshot.ActiveIteration.TaskID)
		}
		if snapshot.ActiveIteration.PostStatus != "done" {
			t.Fatalf("post status = %q, want done", snapshot.ActiveIteration.PostStatus)
		}
		if snapshot.Progress.InProgress != 1 {
			t.Fatalf("in_progress = %d, want 1", snapshot.Progress.InProgress)
		}
		if snapshot.Elapsed != 5*time.Second {
			t.Fatalf("elapsed = %s, want %s", snapshot.Elapsed, 5*time.Second)
		}
		if snapshot.ActiveIteration.LastOutputAt == nil || !snapshot.ActiveIteration.LastOutputAt.Equal(startedAt.Add(3*time.Second)) {
			t.Fatal("LastOutputAt deveria refletir o evento de output observado")
		}
	})

	t.Run("transicao invalida retorna erro tipado", func(t *testing.T) {
		t.Parallel()

		session, err := NewLoopSession("plain", 2, false, 2, startedAt)
		if err != nil {
			t.Fatalf("NewLoopSession() erro inesperado: %v", err)
		}

		_, err = session.Apply(LoopEvent{
			Time:      startedAt,
			Kind:      EventPhaseChanged,
			Iteration: 1,
			Phase:     PhaseRunning,
			Task:      task,
		})
		if err == nil {
			t.Fatal("Apply() deveria falhar")
		}
		if !errors.Is(err, ErrInvalidPhaseTransition) {
			t.Fatalf("erro = %v, want errors.Is(..., ErrInvalidPhaseTransition)", err)
		}
	})

	t.Run("falha observada atualiza fase e ultimo erro", func(t *testing.T) {
		t.Parallel()

		session, err := NewLoopSession("plain", 2, false, 3, startedAt)
		if err != nil {
			t.Fatalf("NewLoopSession() erro inesperado: %v", err)
		}
		events := []LoopEvent{
			{Time: startedAt, Kind: EventSessionStarted},
			{Time: startedAt.Add(time.Second), Kind: EventIterationSelected, Iteration: 1, Task: task, Tool: ToolClaude},
			{Time: startedAt.Add(2 * time.Second), Kind: EventPhaseChanged, Iteration: 1, Task: task, Tool: ToolClaude, Phase: PhaseRunning},
			{Time: startedAt.Add(3 * time.Second), Kind: EventFailureObserved, Iteration: 1, Task: task, Tool: ToolClaude, ErrorCode: ErrorToolAuthRequired},
		}

		var snapshot SessionSnapshot
		for _, event := range events {
			snapshot, err = session.Apply(event)
			if err != nil {
				t.Fatalf("Apply(%s) erro inesperado: %v", event.Kind, err)
			}
		}

		if snapshot.CurrentPhase != PhaseAuthRequired {
			t.Fatalf("fase atual = %q, want %q", snapshot.CurrentPhase, PhaseAuthRequired)
		}
		if snapshot.LastError == nil {
			t.Fatal("LastError deveria ser preenchido")
		}
		if !errors.Is(snapshot.LastError, ErrToolAuthRequired) {
			t.Fatalf("LastError = %v, want errors.Is(..., ErrToolAuthRequired)", snapshot.LastError)
		}
	})

	t.Run("buffer limitado remove eventos mais antigos", func(t *testing.T) {
		t.Parallel()

		session, err := NewLoopSession("plain", 2, false, 2, startedAt)
		if err != nil {
			t.Fatalf("NewLoopSession() erro inesperado: %v", err)
		}

		events := []LoopEvent{
			{Time: startedAt, Kind: EventSessionStarted, Message: "sessao"},
			{Time: startedAt.Add(time.Second), Kind: EventIterationSelected, Iteration: 1, Task: task, Message: "selecionada"},
			{Time: startedAt.Add(2 * time.Second), Kind: EventPhaseChanged, Iteration: 1, Task: task, Phase: PhaseRunning, Message: "running"},
		}
		for _, event := range events {
			if _, err := session.Apply(event); err != nil {
				t.Fatalf("Apply(%s) erro inesperado: %v", event.Kind, err)
			}
		}

		snapshot := session.SnapshotAt(startedAt.Add(2 * time.Second))
		if len(snapshot.RecentEvents) != 2 {
			t.Fatalf("len(RecentEvents) = %d, want 2", len(snapshot.RecentEvents))
		}
		if snapshot.RecentEvents[0].Message != "selecionada" {
			t.Fatalf("primeiro evento no buffer = %q, want %q", snapshot.RecentEvents[0].Message, "selecionada")
		}
		if snapshot.RecentEvents[1].Message != "running" {
			t.Fatalf("ultimo evento no buffer = %q, want %q", snapshot.RecentEvents[1].Message, "running")
		}
	})

	t.Run("resumo final preserva iteracoes apos limpar sessao ativa", func(t *testing.T) {
		t.Parallel()

		session, err := NewLoopSession("plain", 2, false, 3, startedAt)
		if err != nil {
			t.Fatalf("NewLoopSession() erro inesperado: %v", err)
		}
		events := []LoopEvent{
			{Time: startedAt, Kind: EventSessionStarted},
			{Time: startedAt.Add(time.Second), Kind: EventIterationSelected, Iteration: 1, Task: task},
			{Time: startedAt.Add(2 * time.Second), Kind: EventPhaseChanged, Iteration: 1, Task: task, Phase: PhaseRunning},
			{Time: startedAt.Add(3 * time.Second), Kind: EventPhaseChanged, Iteration: 1, Task: task, Phase: PhaseDone},
			{Time: startedAt.Add(4 * time.Second), Kind: EventSessionFinished},
		}
		for _, event := range events {
			if _, err := session.Apply(event); err != nil {
				t.Fatalf("Apply(%s) erro inesperado: %v", event.Kind, err)
			}
		}

		summary := session.FinalSummary("limite atingido", "report.md")
		if summary.IterationsRun != 1 {
			t.Fatalf("IterationsRun = %d, want 1", summary.IterationsRun)
		}
		if snapshot := session.SnapshotAt(startedAt.Add(4 * time.Second)); snapshot.ActiveIteration != nil {
			t.Fatal("ActiveIteration deveria ser limpo apos EventSessionFinished")
		}
	})
}
