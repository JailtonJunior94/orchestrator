package taskloop

import (
	"strings"
	"sync"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
)

type fakeBubbleTeaProgram struct {
	mu   sync.Mutex
	msgs []tea.Msg
	done chan struct{}
}

func newFakeBubbleTeaProgram() *fakeBubbleTeaProgram {
	return &fakeBubbleTeaProgram{done: make(chan struct{})}
}

func (p *fakeBubbleTeaProgram) Send(msg tea.Msg) {
	p.mu.Lock()
	p.msgs = append(p.msgs, msg)
	p.mu.Unlock()

	if _, ok := msg.(bubbleTeaSummaryMsg); ok {
		select {
		case <-p.done:
		default:
			close(p.done)
		}
	}
}

func (p *fakeBubbleTeaProgram) Run() (tea.Model, error) {
	<-p.done
	return nil, nil
}

func (p *fakeBubbleTeaProgram) Messages() []tea.Msg {
	p.mu.Lock()
	defer p.mu.Unlock()

	out := make([]tea.Msg, len(p.msgs))
	copy(out, p.msgs)
	return out
}

func TestBubbleTeaPresenterLifecycle(t *testing.T) {
	t.Parallel()

	startedAt := time.Date(2026, 4, 23, 12, 0, 0, 0, time.UTC)
	task, err := NewTaskRef("6.0", "Presenter Bubble Tea")
	if err != nil {
		t.Fatalf("NewTaskRef() erro inesperado: %v", err)
	}

	session, err := NewLoopSession("simples", 8, true, 4, startedAt)
	if err != nil {
		t.Fatalf("NewLoopSession() erro inesperado: %v", err)
	}

	program := newFakeBubbleTeaProgram()
	presenter := NewBubbleTeaPresenter(
		TerminalCapabilities{Interactive: true, Width: 96, Height: 24, SupportsAltScreen: true},
		func() SessionSnapshot {
			return session.SnapshotAt(startedAt.Add(3 * time.Second))
		},
	)
	presenter.programFactory = func(model tea.Model, options ...tea.ProgramOption) bubbleTeaProgram {
		return program
	}

	if err := presenter.Start(session.SnapshotAt(startedAt)); err != nil {
		t.Fatalf("Start() erro inesperado: %v", err)
	}

	events := []LoopEvent{
		{Time: startedAt, Kind: EventSessionStarted},
		{Time: startedAt.Add(time.Second), Kind: EventIterationSelected, Iteration: 1, Task: task, Tool: ToolCodex, Role: RoleExecutor, Phase: PhasePreparing},
		{Time: startedAt.Add(2 * time.Second), Kind: EventPhaseChanged, Iteration: 1, Task: task, Tool: ToolCodex, Role: RoleExecutor, Phase: PhaseRunning, Message: "subprocesso iniciado"},
	}
	for _, event := range events {
		if _, err := session.Apply(event); err != nil {
			t.Fatalf("Apply(%s) erro inesperado: %v", event.Kind, err)
		}
		if err := presenter.Consume(event); err != nil {
			t.Fatalf("Consume(%s) erro inesperado: %v", event.Kind, err)
		}
	}

	summary := session.FinalSummary("execucao encerrada", "report.md")
	if err := presenter.Finish(summary); err != nil {
		t.Fatalf("Finish() erro inesperado: %v", err)
	}

	msgs := program.Messages()
	if len(msgs) < 3 {
		t.Fatalf("mensagens enviadas = %d, esperado ao menos 3", len(msgs))
	}
	if _, ok := msgs[0].(bubbleTeaSnapshotMsg); !ok {
		t.Fatalf("primeira mensagem = %T, esperado bubbleTeaSnapshotMsg", msgs[0])
	}
	if _, ok := msgs[len(msgs)-1].(bubbleTeaSummaryMsg); !ok {
		t.Fatalf("ultima mensagem = %T, esperado bubbleTeaSummaryMsg", msgs[len(msgs)-1])
	}
}

func TestBubbleTeaPresenterUsesModeSelection(t *testing.T) {
	t.Parallel()

	session, err := NewLoopSession("simples", 4, true, 3, time.Date(2026, 4, 23, 12, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("NewLoopSession() erro inesperado: %v", err)
	}

	svc := &Service{}
	plain := svc.resolveObserver(UIModePlain, session, TerminalCapabilities{})
	if _, ok := plain.(*TextPresenter); !ok {
		t.Fatalf("observer plain = %T, esperado *TextPresenter", plain)
	}

	tui := svc.resolveObserver(UIModeTUI, session, TerminalCapabilities{Interactive: true})
	if _, ok := tui.(*BubbleTeaPresenter); !ok {
		t.Fatalf("observer tui = %T, esperado *BubbleTeaPresenter", tui)
	}
}

func TestBubbleTeaLayout(t *testing.T) {
	t.Parallel()

	renderer := newBubbleTeaRenderer()
	now := time.Date(2026, 4, 23, 12, 0, 0, 0, time.UTC)
	snapshot := SessionSnapshot{
		Mode:          "simples",
		CurrentPhase:  PhaseRunning,
		Elapsed:       5 * time.Second,
		MaxIterations: 8,
		Progress: BatchProgress{
			Total:      4,
			Done:       1,
			Pending:    2,
			InProgress: 1,
		},
		ActiveIteration: &IterationSnapshot{
			Sequence: 1,
			TaskID:   "6.0",
			Title:    "Presenter Bubble Tea com layout simples",
			Tool:     ToolCodex,
			Role:     RoleExecutor,
			Phase:    PhaseRunning,
		},
		RecentEvents: []RecentEvent{
			{Timestamp: now, Message: "subprocesso iniciado", Severity: SeverityInfo, Task: TaskRef{ID: "6.0"}, Tool: ToolCodex, Phase: PhaseRunning},
			{Timestamp: now.Add(time.Second), Message: "primeira saida observada", Severity: SeverityInfo, Task: TaskRef{ID: "6.0"}, Tool: ToolCodex, Phase: PhaseStreaming},
		},
	}

	tests := []struct {
		name   string
		width  int
		height int
		want   []string
	}{
		{
			name:   "layout largo mostra painel principal e eventos",
			width:  96,
			height: 22,
			want: []string{
				"Execucao",
				"Task: 6.0 Presenter Bubble Tea com layout simples",
				"Papel: executor | Tool: codex",
				"Eventos recentes",
				"INFO task=6.0 tool=codex",
			},
		},
		{
			name:   "layout estreito degrada detalhes sem perder labels",
			width:  52,
			height: 18,
			want: []string{
				"Iteracao/Execucao: iteracao 1/8",
				"Fase: ATIVO EXECUTANDO",
				"UI: tui | Modo: simples",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := renderer.Render(tt.width, tt.height, snapshot, nil)
			for _, want := range tt.want {
				if !strings.Contains(got, want) {
					t.Fatalf("render nao contem %q\nsaida:\n%s", want, got)
				}
			}
		})
	}
}

func TestBubbleTeaResponsive(t *testing.T) {
	t.Parallel()

	renderer := newBubbleTeaRenderer()
	now := time.Date(2026, 4, 23, 12, 0, 0, 0, time.UTC)
	snapshot := SessionSnapshot{
		Mode:         "simples",
		CurrentPhase: PhaseAuthRequired,
		Elapsed:      12 * time.Second,
		Progress: BatchProgress{
			Total:      3,
			Failed:     1,
			Pending:    1,
			InProgress: 1,
		},
		ActiveIteration: &IterationSnapshot{
			Sequence: 2,
			TaskID:   "6.0",
			Title:    "Presenter Bubble Tea com evento longo demais para caber inteiro",
			Tool:     ToolClaude,
			Role:     RoleExecutor,
			Phase:    PhaseAuthRequired,
		},
		RecentEvents: []RecentEvent{
			{
				Timestamp: now,
				Message:   "claude requer autenticacao e exibiu uma mensagem longa demais para a largura minima do terminal",
				Severity:  SeverityError,
				Task:      TaskRef{ID: "6.0"},
				Tool:      ToolClaude,
				Phase:     PhaseAuthRequired,
			},
		},
		LastError: NewLoopFailure(ErrorToolAuthRequired, "claude requer autenticacao", nil),
	}

	got := renderer.Render(44, 16, snapshot, nil)
	if !strings.Contains(got, "CRITICO AUTENTICACAO PENDENTE") {
		t.Fatalf("render deveria manter label critico textual\nsaida:\n%s", got)
	}
	if !strings.Contains(got, "claude requer autenticacao") {
		t.Fatalf("render deveria expor mensagem de falha\nsaida:\n%s", got)
	}
	if !strings.Contains(got, "...") {
		t.Fatalf("render deveria truncar eventos longos\nsaida:\n%s", got)
	}
}
