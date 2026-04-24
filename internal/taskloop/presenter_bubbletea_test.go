package taskloop

import (
	"image/color"
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

func TestResolveLayoutTier(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		width int
		want  layoutTier
	}{
		{"estreito (79)", 79, layoutCompact},
		{"limite medio inferior (80)", 80, layoutMedium},
		{"limite medio superior (119)", 119, layoutMedium},
		{"limite largo inferior (120)", 120, layoutWide},
		{"largo (200)", 200, layoutWide},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := resolveLayoutTier(tt.width)
			if got != tt.want {
				t.Errorf("resolveLayoutTier(%d) = %d, want %d", tt.width, got, tt.want)
			}
		})
	}
}

func TestComputeBlockHeights(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		totalHeight int
		tier        layoutTier
		wantQueueSummary int
		wantEventsMin    int
	}{
		// Modo compacto: queueSummary omitido
		{"compacto pequeno (15)", 15, layoutCompact, 0, 1},
		{"compacto medio (24)", 24, layoutCompact, 0, 1},
		{"compacto grande (40)", 40, layoutCompact, 0, 1},

		// Modo medio: eventos >= 3
		{"medio pequeno (20)", 20, layoutMedium, 5, 3},
		{"medio padrao (24)", 24, layoutMedium, 5, 3},
		{"medio grande (40)", 40, layoutMedium, 5, 3},

		// Modo largo: eventos >= 3
		{"largo pequeno (20)", 20, layoutWide, 5, 3},
		{"largo padrao (24)", 24, layoutWide, 5, 3},
		{"largo grande (40)", 40, layoutWide, 5, 3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := computeBlockHeights(tt.totalHeight, tt.tier)

			if got.queueSummary != tt.wantQueueSummary {
				t.Errorf("queueSummary = %d, want %d", got.queueSummary, tt.wantQueueSummary)
			}
			if got.events < tt.wantEventsMin {
				t.Errorf("events = %d, want >= %d", got.events, tt.wantEventsMin)
			}
			if got.footer != 1 {
				t.Errorf("footer = %d, want 1", got.footer)
			}
			if got.progressBar != 1 {
				t.Errorf("progressBar = %d, want 1", got.progressBar)
			}
		})
	}
}

func TestComputeBlockHeightsCompactQueueZero(t *testing.T) {
	t.Parallel()

	for _, height := range []int{8, 15, 24, 40} {
		got := computeBlockHeights(height, layoutCompact)
		if got.queueSummary != 0 {
			t.Errorf("compacto height=%d: queueSummary = %d, want 0", height, got.queueSummary)
		}
	}
}

func TestComputeBlockHeightsWideEventsMinimum(t *testing.T) {
	t.Parallel()

	for _, tier := range []layoutTier{layoutWide, layoutMedium} {
		for _, height := range []int{20, 24, 30, 40} {
			got := computeBlockHeights(height, tier)
			if got.events < 3 {
				t.Errorf("tier=%d height=%d: events = %d, want >= 3", tier, height, got.events)
			}
		}
	}
}

func TestPhaseStatusLabel(t *testing.T) {
	t.Parallel()

	phases := []AgentPhase{
		PhaseIdle,
		PhasePreparing,
		PhaseRunning,
		PhaseStreaming,
		PhaseReviewing,
		PhaseDone,
		PhaseFailed,
		PhaseTimeout,
		PhaseAuthRequired,
	}

	for _, phase := range phases {
		t.Run(string(phase), func(t *testing.T) {
			t.Parallel()
			got := phaseStatusLabel(phase)
			if got == "" {
				t.Errorf("phaseStatusLabel(%q) = empty, want label PT-BR nao vazio", phase)
			}
		})
	}
}

func TestDefaultThemeColors(t *testing.T) {
	t.Parallel()

	theme := defaultTheme()
	fields := []struct {
		name  string
		color color.Color
	}{
		{"primary", theme.primary},
		{"secondary", theme.secondary},
		{"success", theme.success},
		{"danger", theme.danger},
		{"warning", theme.warning},
		{"info", theme.info},
		{"muted", theme.muted},
		{"background", theme.background},
		{"activeBorder", theme.activeBorder},
		{"normalBorder", theme.normalBorder},
	}

	for _, f := range fields {
		t.Run(f.name, func(t *testing.T) {
			t.Parallel()
			if f.color == nil {
				t.Errorf("defaultTheme().%s = nil, want cor definida", f.name)
			}
		})
	}
}

func TestRenderDashboard(t *testing.T) {
	t.Parallel()

	renderer := newBubbleTeaRenderer()
	snapshot := SessionSnapshot{
		Mode:          "iterativo",
		CurrentTool:   ToolCodex,
		MaxIterations: 10,
	}
	snapshotNoTool := SessionSnapshot{
		Mode:          "simples",
		MaxIterations: 5,
	}

	tests := []struct {
		name     string
		width    int
		snapshot SessionSnapshot
		tier     layoutTier
		wantAll  []string
		wantNone []string
	}{
		{
			name:     "wide (140) renderiza painel com borda e 4 campos",
			width:    140,
			snapshot: snapshot,
			tier:     layoutWide,
			wantAll:  []string{"ai-spec task-loop", "Modo: iterativo", "Tool: codex", "Max: 10"},
		},
		{
			name:     "medium (100) renderiza painel com borda e 4 campos",
			width:    100,
			snapshot: snapshot,
			tier:     layoutMedium,
			wantAll:  []string{"ai-spec task-loop", "Modo: iterativo", "Tool: codex", "Max: 10"},
		},
		{
			name:     "compact (60) renderiza linha unica com separadores",
			width:    60,
			snapshot: snapshot,
			tier:     layoutCompact,
			wantAll:  []string{"ai-spec task-loop", "|"},
		},
		{
			name:     "sem tool ativa exibe n/a",
			width:    120,
			snapshot: snapshotNoTool,
			tier:     layoutWide,
			wantAll:  []string{"Tool: n/a", "Modo: simples", "Max: 5"},
		},
		{
			name:     "compact nao excede largura fornecida",
			width:    60,
			snapshot: snapshot,
			tier:     layoutCompact,
			wantAll:  []string{"ai-spec task-loop"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := renderer.renderDashboard(tt.width, tt.snapshot, tt.tier)
			for _, want := range tt.wantAll {
				if !strings.Contains(got, want) {
					t.Errorf("renderDashboard nao contem %q\nsaida:\n%s", want, got)
				}
			}
			for _, notWant := range tt.wantNone {
				if strings.Contains(got, notWant) {
					t.Errorf("renderDashboard nao deveria conter %q\nsaida:\n%s", notWant, got)
				}
			}
		})
	}
}

func TestRenderDashboardTruncation(t *testing.T) {
	t.Parallel()

	renderer := newBubbleTeaRenderer()
	// Snapshot com campos muito longos para forcar truncamento
	snapshot := SessionSnapshot{
		Mode:          strings.Repeat("modo-muito-longo", 5),
		CurrentTool:   ToolName(strings.Repeat("ferramenta-longa", 5)),
		MaxIterations: 99,
	}

	// Em largura pequena (60), o texto longo deve ser truncado com "..."
	for _, tier := range []layoutTier{layoutWide, layoutMedium, layoutCompact} {
		got := renderer.renderDashboard(60, snapshot, tier)
		if !strings.Contains(got, "...") {
			t.Errorf("tier=%d: texto longo deveria ser truncado com '...'\nsaida:\n%s", tier, got)
		}
	}
}

func TestRenderProgressBar(t *testing.T) {
	t.Parallel()

	renderer := newBubbleTeaRenderer()

	tests := []struct {
		name     string
		width    int
		progress BatchProgress
		wantAll  []string
	}{
		{
			name:  "total=0 exibe mensagem vazia sem panic",
			width: 100,
			progress: BatchProgress{
				Total: 0,
			},
			wantAll: []string{"sem tarefas conhecidas"},
		},
		{
			name:  "0/10 exibe 0%",
			width: 100,
			progress: BatchProgress{
				Total:   10,
				Done:    0,
				Pending: 10,
			},
			wantAll: []string{"0/10 tasks", "0%"},
		},
		{
			name:  "5/10 exibe 50% com barra preenchida",
			width: 100,
			progress: BatchProgress{
				Total:   10,
				Done:    5,
				Pending: 5,
			},
			wantAll: []string{"5/10 tasks", "50%", "█"},
		},
		{
			name:  "10/10 exibe 100% com barra totalmente preenchida",
			width: 100,
			progress: BatchProgress{
				Total: 10,
				Done:  10,
			},
			wantAll: []string{"10/10 tasks", "100%", "█"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := renderer.renderProgressBar(tt.width, tt.progress)
			for _, want := range tt.wantAll {
				if !strings.Contains(got, want) {
					t.Errorf("renderProgressBar nao contem %q\nprogress=%+v\nsaida:\n%s", want, tt.progress, got)
				}
			}
		})
	}
}

func TestRenderProgressBarProportional(t *testing.T) {
	t.Parallel()

	renderer := newBubbleTeaRenderer()
	progress := BatchProgress{
		Total:   10,
		Done:    5,
		Pending: 5,
	}

	got := renderer.renderProgressBar(100, progress)
	filledCount := strings.Count(got, "█")
	emptyCount := strings.Count(got, "░")

	if filledCount == 0 {
		t.Error("renderProgressBar 50%: nenhum caractere preenchido encontrado")
	}
	if emptyCount == 0 {
		t.Error("renderProgressBar 50%: nenhum caractere vazio encontrado")
	}
	// Em 50%, filled ~= empty (com margem de 1 para arredondamento)
	diff := filledCount - emptyCount
	if diff < -2 || diff > 2 {
		t.Errorf("renderProgressBar 50%%: filled=%d, empty=%d — proporcao esperada ~1:1", filledCount, emptyCount)
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

func TestRenderActiveTask(t *testing.T) {
	t.Parallel()

	renderer := newBubbleTeaRenderer()
	startedAt := time.Date(2026, 4, 23, 10, 0, 0, 0, time.UTC)

	snapshotExecutor := SessionSnapshot{
		Mode:          "simples",
		MaxIterations: 8,
		ActiveIteration: &IterationSnapshot{
			Sequence:  3,
			TaskID:    "10.0",
			Title:     "Render do Painel de Execucao",
			Tool:      ToolClaude,
			Role:      RoleExecutor,
			Phase:     PhaseRunning,
			StartedAt: startedAt,
		},
	}
	snapshotReviewer := SessionSnapshot{
		Mode:          "avancado",
		MaxIterations: 5,
		ActiveIteration: &IterationSnapshot{
			Sequence:  2,
			TaskID:    "10.0",
			Title:     "Render do Painel de Execucao",
			Tool:      ToolClaude,
			Role:      RoleReviewer,
			Phase:     PhaseReviewing,
			StartedAt: startedAt,
		},
	}
	snapshotNil := SessionSnapshot{
		Mode:          "simples",
		MaxIterations: 8,
	}

	tests := []struct {
		name     string
		width    int
		snapshot SessionSnapshot
		tier     layoutTier
		wantAll  []string
		wantNot  []string
	}{
		// Wide com executor
		{
			name:     "wide executor mostra 6 linhas com labels",
			width:    120,
			snapshot: snapshotExecutor,
			tier:     layoutWide,
			wantAll: []string{
				"Task Ativa",
				"ID: 10.0",
				"Render do Painel de Execucao",
				"Iteracao: iteracao 3/8",
				"Ferramenta: claude",
				"Papel: executor",
				"Fase:",
				"Tempo:",
			},
		},
		// Medium com reviewer
		{
			name:     "medium reviewer mostra ferramenta e papel",
			width:    100,
			snapshot: snapshotReviewer,
			tier:     layoutMedium,
			wantAll: []string{
				"Task Ativa",
				"ID: 10.0",
				"Ferramenta: claude",
				"Papel: reviewer",
				"Fase:",
				"Tempo:",
			},
		},
		// Compact com executor: 2 linhas densas (sem label "Task Ativa" no cabecalho)
		{
			name:     "compact executor mostra 2 linhas densas",
			width:    79,
			snapshot: snapshotExecutor,
			tier:     layoutCompact,
			wantAll: []string{
				"10.0",
				"Render do Painel de Execucao",
				"executor",
			},
			wantNot: []string{"Task Ativa"},
		},
		// Compact com reviewer
		{
			name:     "compact reviewer mostra papel na linha 2",
			width:    79,
			snapshot: snapshotReviewer,
			tier:     layoutCompact,
			wantAll: []string{
				"10.0",
				"reviewer",
			},
		},
		// ActiveIteration nil — wide
		{
			name:     "nil wide exibe mensagem de espera",
			width:    120,
			snapshot: snapshotNil,
			tier:     layoutWide,
			wantAll: []string{
				"Task Ativa",
				"Aguardando selecao de task...",
			},
		},
		// ActiveIteration nil — medium
		{
			name:     "nil medium exibe mensagem de espera",
			width:    100,
			snapshot: snapshotNil,
			tier:     layoutMedium,
			wantAll: []string{
				"Task Ativa",
				"Aguardando selecao de task...",
			},
		},
		// ActiveIteration nil — compact
		{
			name:     "nil compact exibe mensagem de espera",
			width:    60,
			snapshot: snapshotNil,
			tier:     layoutCompact,
			wantAll: []string{
				"Aguardando selecao de task...",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := renderer.renderActiveTask(tt.width, tt.snapshot, tt.tier)
			for _, want := range tt.wantAll {
				if !strings.Contains(got, want) {
					t.Errorf("renderActiveTask nao contem %q\nsaida:\n%s", want, got)
				}
			}
			for _, notWant := range tt.wantNot {
				if strings.Contains(got, notWant) {
					t.Errorf("renderActiveTask nao deveria conter %q\nsaida:\n%s", notWant, got)
				}
			}
		})
	}
}

func TestRenderActiveTaskTruncation(t *testing.T) {
	t.Parallel()

	renderer := newBubbleTeaRenderer()
	startedAt := time.Date(2026, 4, 23, 10, 0, 0, 0, time.UTC)
	longTitle := strings.Repeat("TituloMuitoLongoDaImplementacao", 10)

	snapshot := SessionSnapshot{
		Mode:          "simples",
		MaxIterations: 8,
		ActiveIteration: &IterationSnapshot{
			Sequence:  1,
			TaskID:    "1.0",
			Title:     longTitle,
			Tool:      ToolClaude,
			Role:      RoleExecutor,
			Phase:     PhaseRunning,
			StartedAt: startedAt,
		},
	}

	for _, tt := range []struct {
		name  string
		width int
		tier  layoutTier
	}{
		{"wide 120", 120, layoutWide},
		{"medium 100", 100, layoutMedium},
		{"compact 79", 79, layoutCompact},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := renderer.renderActiveTask(tt.width, snapshot, tt.tier)
			if !strings.Contains(got, "...") {
				t.Errorf("titulo longo deveria ser truncado com '...' em %s\nsaida:\n%s", tt.name, got)
			}
		})
	}
}

func TestRenderIterationCounterActiveTask(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		current int
		max     int
		want    string
	}{
		{"valor normal 3/8", 3, 8, "iteracao 3/8"},
		{"primeiro 1/5", 1, 5, "iteracao 1/5"},
		{"ultimo 8/8", 8, 8, "iteracao 8/8"},
		{"sem max", 3, 0, "iteracao 3"},
		{"current zero", 0, 8, "iteracao"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := renderIterationCounter(tt.current, tt.max)
			if got != tt.want {
				t.Errorf("renderIterationCounter(%d, %d) = %q, want %q", tt.current, tt.max, got, tt.want)
			}
		})
	}
}

func TestRenderActiveTaskPhaseLabel(t *testing.T) {
	t.Parallel()

	renderer := newBubbleTeaRenderer()
	startedAt := time.Date(2026, 4, 23, 10, 0, 0, 0, time.UTC)

	criticalPhases := []AgentPhase{PhaseFailed, PhaseTimeout, PhaseAuthRequired}
	for _, phase := range criticalPhases {
		t.Run("fase critica textual "+string(phase), func(t *testing.T) {
			t.Parallel()
			snapshot := SessionSnapshot{
				MaxIterations: 4,
				ActiveIteration: &IterationSnapshot{
					Sequence:  1,
					TaskID:    "1.0",
					Title:     "implementacao do renderer",
					Tool:      ToolClaude,
					Role:      RoleExecutor,
					Phase:     phase,
					StartedAt: startedAt,
				},
			}
			got := renderer.renderActiveTask(120, snapshot, layoutWide)
			label := phaseStatusLabel(phase)
			if !strings.Contains(got, label) {
				t.Errorf("renderActiveTask fase=%s nao contem label %q\nsaida:\n%s", phase, label, got)
			}
		})
	}
}
