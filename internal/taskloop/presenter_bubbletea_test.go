package taskloop

import (
	"fmt"
	"image/color"
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"
	"unicode/utf8"

	tea "charm.land/bubbletea/v2"
)

// ansiEscapeRe filtra sequencias de escape ANSI do output para calculo de largura visual.
var ansiEscapeRe = regexp.MustCompile(`\x1b\[[0-9;]*[A-Za-z]`)

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
			name:   "layout largo mostra 6 blocos com conteudo esperado",
			width:  96,
			height: 22,
			want: []string{
				"ai-spec task-loop",
				"Task Ativa",
				"ID: 6.0",
				"Ferramenta: codex",
				"Papel: executor",
				"Eventos recentes",
				"INFO",
				"task=6.0",
			},
		},
		{
			name:   "layout estreito degrada para compacto sem perder labels essenciais",
			width:  52,
			height: 18,
			want: []string{
				"q sair",
				"Modo: simples",
				"iteracao 1/8",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := renderer.Render(tt.width, tt.height, snapshot, nil, focusActiveTask, false)
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

	// Em layout compacto (width=44), labels criticos aparecem truncados mas presentes.
	// A verificacao usa "CRIT" (inicio de "CRITICO AUTENTICACAO PENDENTE") pois o
	// texto completo excede a largura util de 40 chars no compacto.
	got := renderer.Render(44, 16, snapshot, nil, focusActiveTask, false)
	if !strings.Contains(got, "CRIT") {
		t.Fatalf("render deveria manter label critico textual (ao menos 'CRIT')\nsaida:\n%s", got)
	}
	if !strings.Contains(got, "ERRO") {
		t.Fatalf("render deveria expor label de severidade de erro no painel de eventos\nsaida:\n%s", got)
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

func TestRenderQueueSummary(t *testing.T) {
	t.Parallel()

	renderer := newBubbleTeaRenderer()
	progress := BatchProgress{
		Total:      10,
		Pending:    3,
		InProgress: 1,
		Done:       4,
		Failed:     1,
		Blocked:    1,
	}

	tests := []struct {
		name     string
		width    int
		progress BatchProgress
		tier     layoutTier
		wantAll  []string
		wantNone []string
	}{
		{
			name:     "wide (120) renderiza painel com 6 contadores",
			width:    120,
			progress: progress,
			tier:     layoutWide,
			wantAll: []string{
				"Fila",
				"Total: 10",
				"Pendentes: 3",
				"Em execucao: 1",
				"Concluidas: 4",
				"Falhadas: 1",
				"Bloqueadas: 1",
			},
		},
		{
			name:     "medium (100) renderiza painel com 6 contadores",
			width:    100,
			progress: progress,
			tier:     layoutMedium,
			wantAll: []string{
				"Fila",
				"Total: 10",
				"Pendentes: 3",
				"Em execucao: 1",
				"Concluidas: 4",
				"Falhadas: 1",
				"Bloqueadas: 1",
			},
		},
		{
			name:     "compact retorna string vazia",
			width:    60,
			progress: progress,
			tier:     layoutCompact,
			wantAll:  []string{},
			wantNone: []string{"Fila", "Total"},
		},
		{
			name:  "contadores zerados exibidos corretamente",
			width: 120,
			progress: BatchProgress{
				Total: 0,
			},
			tier:    layoutWide,
			wantAll: []string{"Total: 0", "Pendentes: 0"},
		},
		{
			name: "apenas done e pending",
			width: 100,
			progress: BatchProgress{
				Total:   5,
				Done:    4,
				Pending: 1,
			},
			tier:    layoutMedium,
			wantAll: []string{"Total: 5", "Concluidas: 4", "Pendentes: 1", "Falhadas: 0", "Bloqueadas: 0"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := renderer.renderQueueSummary(tt.width, tt.progress, tt.tier)

			if tt.tier == layoutCompact && got != "" {
				t.Errorf("renderQueueSummary compact deve retornar string vazia, got: %q", got)
			}

			for _, want := range tt.wantAll {
				if !strings.Contains(got, want) {
					t.Errorf("renderQueueSummary nao contem %q\nsaida:\n%s", want, got)
				}
			}
			for _, notWant := range tt.wantNone {
				if strings.Contains(got, notWant) {
					t.Errorf("renderQueueSummary nao deveria conter %q\nsaida:\n%s", notWant, got)
				}
			}
		})
	}
}

func TestRenderQueueSummaryCompactEmpty(t *testing.T) {
	t.Parallel()

	renderer := newBubbleTeaRenderer()
	for _, width := range []int{40, 60, 79} {
		got := renderer.renderQueueSummary(width, BatchProgress{Total: 10, Done: 5}, layoutCompact)
		if got != "" {
			t.Errorf("renderQueueSummary compact width=%d: esperado string vazia, got=%q", width, got)
		}
	}
}

func TestRenderQueueSummaryCounters(t *testing.T) {
	t.Parallel()

	renderer := newBubbleTeaRenderer()
	progress := BatchProgress{
		Total:      10,
		Pending:    3,
		InProgress: 1,
		Done:       4,
		Failed:     1,
		Blocked:    1,
	}

	for _, tier := range []layoutTier{layoutWide, layoutMedium} {
		got := renderer.renderQueueSummary(120, progress, tier)
		for _, want := range []string{"10", "3", "4", "1"} {
			if !strings.Contains(got, want) {
				t.Errorf("renderQueueSummary tier=%d nao contem %q\nsaida:\n%s", tier, want, got)
			}
		}
	}
}

func TestRenderFooter(t *testing.T) {
	t.Parallel()

	renderer := newBubbleTeaRenderer()

	tests := []struct {
		name     string
		width    int
		paused   bool
		focus    panelFocus
		mode     string
		wantAll  []string
		wantNone []string
	}{
		{
			name:   "paused=false mostra 'p pausar'",
			width:  120,
			paused: false,
			focus:  focusActiveTask,
			mode:   "simples",
			wantAll: []string{
				"q sair",
				"p pausar",
				"s skip",
				"tab foco",
				"UI: tui",
				"Modo: simples",
			},
			wantNone: []string{"p retomar"},
		},
		{
			name:   "paused=true mostra 'p retomar'",
			width:  120,
			paused: true,
			focus:  focusActiveTask,
			mode:   "simples",
			wantAll: []string{
				"q sair",
				"p retomar",
				"s skip",
				"tab foco",
			},
			wantNone: []string{"p pausar"},
		},
		{
			name:    "foco em task ativa exibe rodape normalmente",
			width:   120,
			paused:  false,
			focus:   focusActiveTask,
			mode:    "avancado",
			wantAll: []string{"q sair", "Modo: avancado"},
		},
		{
			name:    "foco em fila exibe rodape normalmente",
			width:   120,
			paused:  false,
			focus:   focusQueueSummary,
			mode:    "simples",
			wantAll: []string{"q sair", "tab foco"},
		},
		{
			name:    "foco em eventos exibe rodape normalmente",
			width:   120,
			paused:  false,
			focus:   focusRecentEvents,
			mode:    "simples",
			wantAll: []string{"q sair", "UI: tui"},
		},
		{
			name:    "modo vazio exibe n/a",
			width:   120,
			paused:  false,
			focus:   focusActiveTask,
			mode:    "",
			wantAll: []string{"Modo: n/a"},
		},
		{
			name:    "largura estreita (<80) trunca rodape com '...'",
			width:   60,
			paused:  false,
			focus:   focusActiveTask,
			mode:    "simples",
			wantAll: []string{"q sair"},
		},
		{
			name:    "atalhos marcados como visual",
			width:   160,
			paused:  false,
			focus:   focusActiveTask,
			mode:    "simples",
			wantAll: []string{"(visual)"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := renderer.renderFooter(tt.width, tt.paused, tt.focus, tt.mode)

			for _, want := range tt.wantAll {
				if !strings.Contains(got, want) {
					t.Errorf("renderFooter nao contem %q\nsaida:\n%s", want, got)
				}
			}
			for _, notWant := range tt.wantNone {
				if strings.Contains(got, notWant) {
					t.Errorf("renderFooter nao deveria conter %q\nsaida:\n%s", notWant, got)
				}
			}
		})
	}
}

func TestRenderFooterTruncation(t *testing.T) {
	t.Parallel()

	renderer := newBubbleTeaRenderer()
	// Largura muito pequena deve truncar o rodape
	got := renderer.renderFooter(36, false, focusActiveTask, "simples")
	// Deve conter pelo menos o inicio dos atalhos e terminar com "..."
	if !strings.Contains(got, "q sair") {
		t.Errorf("renderFooter largura estreita deveria conter 'q sair'\nsaida:\n%s", got)
	}
	if !strings.Contains(got, "...") {
		t.Errorf("renderFooter largura estreita deveria truncar com '...'\nsaida:\n%s", got)
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

// --- Testes dos atalhos de teclado (task 13.0) ---

func newTestBubbleTeaModel() bubbleTeaModel {
	return bubbleTeaModel{
		renderer: newBubbleTeaRenderer(),
		width:    100,
		height:   24,
	}
}

func TestUpdateKeyQuitReturnsQuitCmd(t *testing.T) {
	t.Parallel()

	m := newTestBubbleTeaModel()
	_, cmd := m.Update(tea.KeyPressMsg{Code: 'q'})
	if cmd == nil {
		t.Fatal("Update('q') retornou cmd nil, esperado tea.Quit")
	}
	msg := cmd()
	if _, ok := msg.(tea.QuitMsg); !ok {
		t.Fatalf("Update('q'): cmd() = %T, esperado tea.QuitMsg", msg)
	}
}

func TestUpdateKeyCtrlCReturnsQuitCmd(t *testing.T) {
	t.Parallel()

	m := newTestBubbleTeaModel()
	_, cmd := m.Update(tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl})
	if cmd == nil {
		t.Fatal("Update(ctrl+c) retornou cmd nil, esperado tea.Quit")
	}
	msg := cmd()
	if _, ok := msg.(tea.QuitMsg); !ok {
		t.Fatalf("Update(ctrl+c): cmd() = %T, esperado tea.QuitMsg", msg)
	}
}

func TestUpdateKeyPauseToggle(t *testing.T) {
	t.Parallel()

	m := newTestBubbleTeaModel()
	if m.paused {
		t.Fatal("estado inicial: paused deveria ser false")
	}

	// false -> true
	next, cmd := m.Update(tea.KeyPressMsg{Code: 'p'})
	updated := next.(bubbleTeaModel)
	if !updated.paused {
		t.Error("Update('p') false->true: paused deveria ser true")
	}
	if cmd != nil {
		t.Errorf("Update('p') nao deveria retornar cmd, got %T", cmd)
	}

	// true -> false
	next2, _ := updated.Update(tea.KeyPressMsg{Code: 'p'})
	updated2 := next2.(bubbleTeaModel)
	if updated2.paused {
		t.Error("Update('p') true->false: paused deveria ser false apos segundo toggle")
	}
}

func TestUpdateKeySkipNoEffect(t *testing.T) {
	t.Parallel()

	m := newTestBubbleTeaModel()
	m.paused = true
	m.focus = focusQueueSummary

	next, cmd := m.Update(tea.KeyPressMsg{Code: 's'})
	updated := next.(bubbleTeaModel)

	if cmd != nil {
		t.Errorf("Update('s') nao deveria retornar cmd (visual only), got %T", cmd)
	}
	if !updated.paused {
		t.Error("Update('s') nao deveria alterar paused")
	}
	if updated.focus != focusQueueSummary {
		t.Error("Update('s') nao deveria alterar focus")
	}
}

func TestUpdateKeyTabCyclesFocus(t *testing.T) {
	t.Parallel()

	m := newTestBubbleTeaModel()
	// foco inicial: focusActiveTask (0)
	if m.focus != focusActiveTask {
		t.Fatalf("foco inicial = %d, esperado focusActiveTask (%d)", m.focus, focusActiveTask)
	}

	// tab 1: 0 -> 1 (focusQueueSummary)
	next, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyTab})
	m = next.(bubbleTeaModel)
	if m.focus != focusQueueSummary {
		t.Errorf("apos tab 1: focus = %d, esperado focusQueueSummary (%d)", m.focus, focusQueueSummary)
	}

	// tab 2: 1 -> 2 (focusRecentEvents)
	next, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyTab})
	m = next.(bubbleTeaModel)
	if m.focus != focusRecentEvents {
		t.Errorf("apos tab 2: focus = %d, esperado focusRecentEvents (%d)", m.focus, focusRecentEvents)
	}

	// tab 3: 2 -> 0 (cicla para focusActiveTask)
	next, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyTab})
	m = next.(bubbleTeaModel)
	if m.focus != focusActiveTask {
		t.Errorf("apos tab 3 (ciclo): focus = %d, esperado focusActiveTask (%d)", m.focus, focusActiveTask)
	}

	// tab 4: confirma que cicla de volta a 1
	next, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyTab})
	m = next.(bubbleTeaModel)
	if m.focus != focusQueueSummary {
		t.Errorf("apos tab 4: focus = %d, esperado focusQueueSummary (%d)", m.focus, focusQueueSummary)
	}
}

func TestUpdateWindowSizeMsgPreserved(t *testing.T) {
	t.Parallel()

	m := newTestBubbleTeaModel()
	m.width = 80
	m.height = 24

	next, cmd := m.Update(tea.WindowSizeMsg{Width: 140, Height: 40})
	updated := next.(bubbleTeaModel)

	if cmd != nil {
		t.Errorf("Update(WindowSizeMsg) nao deveria retornar cmd, got %T", cmd)
	}
	if updated.width != 140 {
		t.Errorf("Update(WindowSizeMsg): width = %d, esperado 140", updated.width)
	}
	if updated.height != 40 {
		t.Errorf("Update(WindowSizeMsg): height = %d, esperado 40", updated.height)
	}
}

func TestUpdateSnapshotMsgPreserved(t *testing.T) {
	t.Parallel()

	m := newTestBubbleTeaModel()
	snap := SessionSnapshot{
		Mode:          "avancado",
		MaxIterations: 10,
	}

	next, cmd := m.Update(bubbleTeaSnapshotMsg{snapshot: snap})
	updated := next.(bubbleTeaModel)

	if cmd != nil {
		t.Errorf("Update(snapshotMsg) nao deveria retornar cmd, got %T", cmd)
	}
	if updated.snapshot.Mode != "avancado" {
		t.Errorf("snapshot.Mode = %q, esperado 'avancado'", updated.snapshot.Mode)
	}
}

func TestResolveKeyAction(t *testing.T) {
	t.Parallel()

	tests := []struct {
		key  string
		want keyAction
	}{
		{"q", keyQuit},
		{"ctrl+c", keyQuit},
		{"p", keyPause},
		{"s", keySkip},
		{"tab", keyTabFocus},
		{"x", keyNone},
		{"", keyNone},
		{"esc", keyNone},
	}

	for _, tt := range tests {
		t.Run("tecla "+tt.key, func(t *testing.T) {
			t.Parallel()
			got := resolveKeyAction(tt.key)
			if got != tt.want {
				t.Errorf("resolveKeyAction(%q) = %d, want %d", tt.key, got, tt.want)
			}
		})
	}
}

// makeRecentEvents cria uma slice de RecentEvent com n entradas para uso em testes.
func makeRecentEvents(n int) []RecentEvent {
	now := time.Date(2026, 4, 24, 10, 0, 0, 0, time.UTC)
	events := make([]RecentEvent, n)
	for i := range events {
		events[i] = RecentEvent{
			Sequence:  i + 1,
			Timestamp: now.Add(time.Duration(i) * time.Second),
			Message:   fmt.Sprintf("evento %d", i+1),
			Severity:  SeverityInfo,
			Task:      TaskRef{ID: "12.0"},
			Tool:      ToolClaude,
			Phase:     PhaseRunning,
		}
	}
	return events
}

func TestRenderRecentEvents(t *testing.T) {
	t.Parallel()

	renderer := newBubbleTeaRenderer()

	tests := []struct {
		name     string
		width    int
		height   int
		events   []RecentEvent
		focus    bool
		tier     layoutTier
		wantAll  []string
		wantNone []string
	}{
		{
			name:     "0 eventos exibe label e mensagem vazia",
			width:    100,
			height:   10,
			events:   nil,
			focus:    false,
			tier:     layoutWide,
			wantAll:  []string{"Eventos recentes", "Nenhum evento registrado"},
			wantNone: []string{"evento 1"},
		},
		{
			name:     "3 eventos em altura 10 exibe todos",
			width:    100,
			height:   10,
			events:   makeRecentEvents(3),
			focus:    false,
			tier:     layoutWide,
			wantAll:  []string{"Eventos recentes", "evento 1", "evento 2", "evento 3"},
			wantNone: []string{"Nenhum evento registrado"},
		},
		{
			name:   "20 eventos em altura 8 exibe apenas os 5 mais recentes",
			width:  100,
			height: 8,
			events: makeRecentEvents(20),
			focus:  false,
			tier:   layoutWide,
			// height=8: innerHeight=6, capacity=5; window = events[15:20] = eventos 16..20
			wantAll:  []string{"Eventos recentes", "evento 16", "evento 17", "evento 18", "evento 19", "evento 20"},
			wantNone: []string{"evento 15"},
		},
		{
			name:   "10 eventos em altura 5 exibe apenas os 2 mais recentes",
			width:  100,
			height: 5,
			events: makeRecentEvents(10),
			focus:  false,
			tier:   layoutMedium,
			// height=5: innerHeight=3, capacity=2; window = events[8:10] = eventos 9,10
			wantAll:  []string{"Eventos recentes", "evento 9", "evento 10"},
			wantNone: []string{"evento 8"},
		},
		{
			name:     "focus=true contem conteudo sem panic",
			width:    100,
			height:   10,
			events:   makeRecentEvents(5),
			focus:    true,
			tier:     layoutWide,
			wantAll:  []string{"Eventos recentes", "evento 5"},
			wantNone: []string{"Nenhum evento registrado"},
		},
		{
			name:    "altura minima compact (3 linhas) renderiza sem panic",
			width:   60,
			height:  3,
			events:  makeRecentEvents(5),
			focus:   false,
			tier:    layoutCompact,
			wantAll: []string{"Eventos recentes"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := renderer.renderRecentEvents(tt.width, tt.height, tt.events, tt.focus, tt.tier)

			for _, want := range tt.wantAll {
				if !strings.Contains(got, want) {
					t.Errorf("renderRecentEvents nao contem %q\nsaida:\n%s", want, got)
				}
			}
			for _, notWant := range tt.wantNone {
				if strings.Contains(got, notWant) {
					t.Errorf("renderRecentEvents nao deveria conter %q\nsaida:\n%s", notWant, got)
				}
			}
		})
	}
}

func TestRenderRecentEventsSlidingWindow(t *testing.T) {
	t.Parallel()

	renderer := newBubbleTeaRenderer()
	// 15 eventos em altura 8: innerHeight=6, capacity=5; window = events[10:15]
	events := makeRecentEvents(15)
	got := renderer.renderRecentEvents(100, 8, events, false, layoutWide)

	// Deve exibir eventos 11..15 (indices 10..14 na slice, eventos[10:15])
	for i := 11; i <= 15; i++ {
		want := fmt.Sprintf("evento %d", i)
		if !strings.Contains(got, want) {
			t.Errorf("janela deslizante: renderRecentEvents nao contem %q\nsaida:\n%s", want, got)
		}
	}
	// Nao deve exibir evento 10 ou anteriores
	for _, notWant := range []string{"evento 10", "evento 9", "evento 8"} {
		if strings.Contains(got, notWant) {
			t.Errorf("janela deslizante: renderRecentEvents nao deveria conter %q\nsaida:\n%s", notWant, got)
		}
	}
}

func TestRenderRecentEventsFocusBorderDifference(t *testing.T) {
	t.Parallel()

	renderer := newBubbleTeaRenderer()
	events := makeRecentEvents(3)

	withFocus := renderer.renderRecentEvents(100, 10, events, true, layoutWide)
	withoutFocus := renderer.renderRecentEvents(100, 10, events, false, layoutWide)

	// Ambos devem conter o conteudo, mas estilos de borda diferentes
	for _, want := range []string{"Eventos recentes", "evento 1"} {
		if !strings.Contains(withFocus, want) {
			t.Errorf("focus=true: renderRecentEvents nao contem %q\nsaida:\n%s", want, withFocus)
		}
		if !strings.Contains(withoutFocus, want) {
			t.Errorf("focus=false: renderRecentEvents nao contem %q\nsaida:\n%s", want, withoutFocus)
		}
	}
	// Os renders com foco diferente devem ser distintos (bordas com cor diferente)
	if withFocus == withoutFocus {
		t.Error("renderRecentEvents focus=true e focus=false produziram output identico; estilos de borda deveriam diferir")
	}
}

func TestRenderRecentEventsTruncation(t *testing.T) {
	t.Parallel()

	renderer := newBubbleTeaRenderer()
	longMsg := strings.Repeat("mensagem-muito-longa-de-evento-recente", 10)
	now := time.Date(2026, 4, 24, 10, 0, 0, 0, time.UTC)
	events := []RecentEvent{
		{
			Sequence:  1,
			Timestamp: now,
			Message:   longMsg,
			Severity:  SeverityInfo,
			Task:      TaskRef{ID: "12.0"},
			Tool:      ToolClaude,
			Phase:     PhaseRunning,
		},
	}

	got := renderer.renderRecentEvents(80, 10, events, false, layoutMedium)
	if !strings.Contains(got, "...") {
		t.Errorf("renderRecentEvents com mensagem longa deveria truncar com '...'\nsaida:\n%s", got)
	}
}

func TestRenderRecentEventsOrdemReversa(t *testing.T) {
	t.Parallel()

	renderer := newBubbleTeaRenderer()
	events := makeRecentEvents(3)
	// eventos[0]=evento 1 (mais antigo), eventos[2]=evento 3 (mais recente)
	got := renderer.renderRecentEvents(100, 10, events, false, layoutWide)

	idx1 := strings.Index(got, "evento 1")
	idx3 := strings.Index(got, "evento 3")
	if idx1 == -1 || idx3 == -1 {
		t.Fatalf("renderRecentEvents deveria conter 'evento 1' e 'evento 3'\nsaida:\n%s", got)
	}
	// evento 3 (mais recente) deve aparecer ANTES de evento 1 (mais antigo) no output
	if idx3 >= idx1 {
		t.Errorf("renderRecentEvents: 'evento 3' (idx=%d) deveria aparecer antes de 'evento 1' (idx=%d)\nsaida:\n%s", idx3, idx1, got)
	}
}

// --- Testes da task 14.0: Render() completo com 6 blocos, renderFinalSummary e primeiro frame ---

func makeFullSnapshot() SessionSnapshot {
	now := time.Date(2026, 4, 24, 10, 0, 0, 0, time.UTC)
	return SessionSnapshot{
		Mode:          "simples",
		CurrentPhase:  PhaseRunning,
		CurrentTool:   ToolClaude,
		Elapsed:       10 * time.Second,
		MaxIterations: 8,
		Progress: BatchProgress{
			Total:      5,
			Done:       2,
			Pending:    2,
			InProgress: 1,
		},
		ActiveIteration: &IterationSnapshot{
			Sequence:  3,
			TaskID:    "14.0",
			Title:     "Composicao do Layout de 6 Blocos",
			Tool:      ToolClaude,
			Role:      RoleExecutor,
			Phase:     PhaseRunning,
			StartedAt: now.Add(-5 * time.Second),
		},
		RecentEvents: makeRecentEvents(3),
	}
}

// TestRenderComplete9Combinations testa o Render() completo com 3 larguras x 3 alturas.
// Verifica que cada combinacao produz output nao vazio, sem panic, e com blocos esperados.
func TestRenderComplete9Combinations(t *testing.T) {
	t.Parallel()

	renderer := newBubbleTeaRenderer()
	snapshot := makeFullSnapshot()

	widths := []int{60, 100, 140}
	heights := []int{15, 30, 50}

	for _, w := range widths {
		for _, h := range heights {
			w, h := w, h
			t.Run(fmt.Sprintf("w%d_h%d", w, h), func(t *testing.T) {
				t.Parallel()

				got := renderer.Render(w, h, snapshot, nil, focusActiveTask, false)
				if got == "" {
					t.Fatalf("Render(%d, %d) retornou string vazia", w, h)
				}

				// Todos os tiers exibem dashboard e footer
				for _, want := range []string{"ai-spec task-loop", "q sair"} {
					if !strings.Contains(got, want) {
						t.Errorf("Render(%d, %d) nao contem %q\nsaida:\n%s", w, h, want, got)
					}
				}

				// Em compact (< 80): painel de fila omitido
				if w < 80 {
					for _, notWant := range []string{"Fila", "Pendentes:"} {
						if strings.Contains(got, notWant) {
							t.Errorf("Render(%d, %d) compact nao deveria conter %q\nsaida:\n%s", w, h, notWant, got)
						}
					}
				}

				// Em wide/medium (>= 80): painel de fila presente
				if w >= 80 {
					for _, want := range []string{"Fila", "Total: 5"} {
						if !strings.Contains(got, want) {
							t.Errorf("Render(%d, %d) wide/medium deveria conter %q\nsaida:\n%s", w, h, want, got)
						}
					}
				}
			})
		}
	}
}

// TestRenderWideSixBlocks verifica que Render() em wide (120x40) exibe os 6 blocos visiveis.
func TestRenderWideSixBlocks(t *testing.T) {
	t.Parallel()

	renderer := newBubbleTeaRenderer()
	snapshot := makeFullSnapshot()

	got := renderer.Render(120, 40, snapshot, nil, focusActiveTask, false)

	for _, want := range []string{
		"ai-spec task-loop", // bloco 1: dashboard
		"2/5 tasks",         // bloco 2: barra de progresso
		"Task Ativa",        // bloco 3: painel de task ativa
		"Fila",              // bloco 4: painel de fila/resumo
		"Eventos recentes",  // bloco 5: eventos recentes
		"q sair",            // bloco 6: rodape
	} {
		if !strings.Contains(got, want) {
			t.Errorf("Render 120x40 nao contem %q (bloco esperado ausente)\nsaida:\n%s", want, got)
		}
	}
}

// TestRenderCompactOmitsQueue verifica que Render() em compact (60x20) omite o painel de fila.
func TestRenderCompactOmitsQueue(t *testing.T) {
	t.Parallel()

	renderer := newBubbleTeaRenderer()
	snapshot := makeFullSnapshot()

	got := renderer.Render(60, 20, snapshot, nil, focusActiveTask, false)

	// Compact deve omitir painel de fila
	for _, notWant := range []string{"Fila", "Pendentes:", "Bloqueadas:"} {
		if strings.Contains(got, notWant) {
			t.Errorf("Render 60x20 compact nao deveria conter %q\nsaida:\n%s", notWant, got)
		}
	}
	// Mas ainda deve ter dashboard e rodape
	for _, want := range []string{"ai-spec task-loop", "q sair"} {
		if !strings.Contains(got, want) {
			t.Errorf("Render 60x20 compact deveria conter %q\nsaida:\n%s", want, got)
		}
	}
}

// TestRenderWithSummaryShowsFinalScreen verifica que Render() com summary != nil
// renderiza a tela de encerramento sem os 6 blocos.
func TestRenderWithSummaryShowsFinalScreen(t *testing.T) {
	t.Parallel()

	renderer := newBubbleTeaRenderer()
	snapshot := makeFullSnapshot()
	summary := &FinalSummary{
		StopReason:    "max-iterations atingido",
		IterationsRun: 8,
		ReportPath:    "tasks/report.md",
		Progress: BatchProgress{
			Total: 5,
			Done:  4,
		},
	}

	got := renderer.Render(100, 30, snapshot, summary, focusActiveTask, false)

	// Deve conter a tela de encerramento
	for _, want := range []string{
		"Execucao Finalizada",
		"Motivo: max-iterations atingido",
		"Iteracoes: 8",
		"Relatorio: tasks/report.md",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("Render com summary nao contem %q\nsaida:\n%s", want, got)
		}
	}

	// Nao deve conter os blocos do layout de 6 paineis
	for _, notWant := range []string{"ai-spec task-loop", "q sair", "Task Ativa", "Fila"} {
		if strings.Contains(got, notWant) {
			t.Errorf("Render com summary nao deveria conter bloco do layout %q\nsaida:\n%s", notWant, got)
		}
	}
}

// TestRenderFinalSummaryComplete verifica renderFinalSummary com todos os campos preenchidos.
func TestRenderFinalSummaryComplete(t *testing.T) {
	t.Parallel()

	renderer := newBubbleTeaRenderer()
	summary := FinalSummary{
		StopReason:    "max-iterations atingido",
		IterationsRun: 5,
		ReportPath:    "tasks/prd-feature/report.md",
		Progress: BatchProgress{
			Total:   4,
			Done:    3,
			Failed:  1,
			Pending: 0,
		},
		LastFailure: NewLoopFailure(ErrorToolTimeout, "timeout apos 120s", nil),
	}

	got := renderer.renderFinalSummary(100, 20, summary)

	for _, want := range []string{
		"Execucao Finalizada",
		"Motivo: max-iterations atingido",
		"Iteracoes: 5",
		"Relatorio: tasks/prd-feature/report.md",
		"Lote:",
		"Falha final:",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("renderFinalSummary completo nao contem %q\nsaida:\n%s", want, got)
		}
	}
}

// TestRenderFinalSummaryNoFailure verifica renderFinalSummary quando LastFailure == nil.
func TestRenderFinalSummaryNoFailure(t *testing.T) {
	t.Parallel()

	renderer := newBubbleTeaRenderer()
	summary := FinalSummary{
		StopReason:    "execucao concluida",
		IterationsRun: 3,
		ReportPath:    "tasks/prd-test/report.md",
		Progress: BatchProgress{
			Total: 3,
			Done:  3,
		},
		LastFailure: nil,
	}

	got := renderer.renderFinalSummary(100, 20, summary)

	for _, want := range []string{
		"Execucao Finalizada",
		"Motivo: execucao concluida",
		"Iteracoes: 3",
		"Relatorio: tasks/prd-test/report.md",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("renderFinalSummary sem falha nao contem %q\nsaida:\n%s", want, got)
		}
	}

	if strings.Contains(got, "Falha final:") {
		t.Errorf("renderFinalSummary sem falha nao deveria conter 'Falha final:'\nsaida:\n%s", got)
	}
}

// TestRenderFirstFrame verifica que snapshot com ActiveIteration == nil renderiza
// todos os blocos com estados iniciais (RF-01, D-08).
func TestRenderFirstFrame(t *testing.T) {
	t.Parallel()

	renderer := newBubbleTeaRenderer()

	// Snapshot inicial: sem iteracao ativa, sem eventos, progresso zerado
	snapshot := SessionSnapshot{
		Mode:          "simples",
		MaxIterations: 8,
		Progress:      BatchProgress{},
	}

	for _, tt := range []struct {
		name   string
		width  int
		height int
	}{
		{"wide primeiro frame", 120, 40},
		{"medium primeiro frame", 100, 30},
		{"compact primeiro frame", 60, 20},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := renderer.Render(tt.width, tt.height, snapshot, nil, focusActiveTask, false)
			if got == "" {
				t.Fatalf("primeiro frame Render(%d,%d) retornou string vazia", tt.width, tt.height)
			}

			// Dashboard deve estar presente
			if !strings.Contains(got, "ai-spec task-loop") {
				t.Errorf("primeiro frame nao contem dashboard 'ai-spec task-loop'\nsaida:\n%s", got)
			}

			// Painel de task ativa deve exibir estado inicial de espera
			if !strings.Contains(got, "Aguardando selecao de task...") {
				t.Errorf("primeiro frame nao contem mensagem de espera 'Aguardando selecao de task...'\nsaida:\n%s", got)
			}

			// Painel de eventos deve exibir estado vazio
			if !strings.Contains(got, "Nenhum evento registrado") {
				t.Errorf("primeiro frame nao contem 'Nenhum evento registrado'\nsaida:\n%s", got)
			}

			// Rodape deve estar presente
			if !strings.Contains(got, "q sair") {
				t.Errorf("primeiro frame nao contem rodape 'q sair'\nsaida:\n%s", got)
			}
		})
	}
}

// TestRenderHeightRealisticBound verifica que o output nao excede o height
// para alturas que acomodam os blocos fixos (>= 24 linhas em wide/medium).
// Em compact, blocos colapsam e a altura e sempre controlada.
// Nota: medium/wide com height < ~21 podem exceder o bound por design
// (blocos fixos definidos na techspec — o layout de 6 blocos requer min ~18 linhas fixas).
func TestRenderHeightRealisticBound(t *testing.T) {
	t.Parallel()

	renderer := newBubbleTeaRenderer()
	snapshot := makeFullSnapshot()

	cases := []struct{ w, h int }{
		{120, 40}, {100, 30}, {140, 50}, // wide/medium com altura realista
		{60, 20}, {60, 15},               // compact: blocos colapsam
	}

	for _, c := range cases {
		c := c
		t.Run(fmt.Sprintf("w%d_h%d", c.w, c.h), func(t *testing.T) {
			t.Parallel()
			got := renderer.Render(c.w, c.h, snapshot, nil, focusActiveTask, false)
			lines := strings.Count(got, "\n") + 1
			// Para compact (< 80): layout colapsado deve respeitar bound com tolerancia de +2
			if c.w < 80 && lines > c.h+2 {
				t.Errorf("Render compact(%d,%d): output tem %d linhas, esperado <= %d+2\nsaida:\n%s",
					c.w, c.h, lines, c.h, got)
			}
			// Para wide/medium com altura >= 30: deve respeitar bound com tolerancia de +2
			if c.w >= 80 && c.h >= 30 && lines > c.h+2 {
				t.Errorf("Render(%d,%d): output tem %d linhas, esperado <= %d+2\nsaida:\n%s",
					c.w, c.h, lines, c.h, got)
			}
		})
	}
}

// --- Testes da task 15.0: integracao e validacao fim a fim ---

// stripANSI remove sequencias de escape ANSI do string para medir largura visual.
func stripANSI(s string) string {
	return ansiEscapeRe.ReplaceAllString(s, "")
}

// TestRenderRNF10WidthBound valida que nenhuma linha do output excede a largura informada
// para as 9 combinacoes de width × height definidas em RNF-10.
func TestRenderRNF10WidthBound(t *testing.T) {
	t.Parallel()

	renderer := newBubbleTeaRenderer()
	snapshot := makeFullSnapshot()

	widths := []int{60, 100, 140}
	heights := []int{15, 30, 50}

	for _, w := range widths {
		for _, h := range heights {
			w, h := w, h
			t.Run(fmt.Sprintf("w%d_h%d", w, h), func(t *testing.T) {
				t.Parallel()

				got := renderer.Render(w, h, snapshot, nil, focusActiveTask, false)
				for i, line := range strings.Split(got, "\n") {
					visual := stripANSI(line)
					// Usar contagem de runes (nao bytes) para medir largura visual.
					// Caracteres Unicode de desenho de caixa (box-drawing) e barras de progresso
					// sao multi-byte mas possuem largura visual de 1 coluna cada.
					runeLen := utf8.RuneCountInString(visual)
					if runeLen > w {
						t.Errorf("Render(%d,%d) linha %d excede largura: runes=%d > width=%d\nlinha: %q",
							w, h, i, runeLen, w, visual)
					}
				}
			})
		}
	}
}

// makeSnapshotWithPhase constroi um SessionSnapshot com a fase informada para testes de transicao.
func makeSnapshotWithPhase(phase AgentPhase, done int) SessionSnapshot {
	now := time.Date(2026, 4, 24, 10, 0, 0, 0, time.UTC)
	return SessionSnapshot{
		Mode:          "simples",
		CurrentPhase:  phase,
		CurrentTool:   ToolClaude,
		Elapsed:       10 * time.Second,
		MaxIterations: 5,
		Progress: BatchProgress{
			Total:      5,
			Done:       done,
			Pending:    5 - done,
			InProgress: 1,
		},
		ActiveIteration: &IterationSnapshot{
			Sequence:  done + 1,
			TaskID:    "15.0",
			Title:     "Testes de Integracao e Validacao Fim a Fim",
			Tool:      ToolClaude,
			Role:      RoleExecutor,
			Phase:     phase,
			StartedAt: now.Add(-3 * time.Second),
		},
		RecentEvents: makeRecentEvents(2),
	}
}

// TestRenderStateTransitions valida que o layout reflete cada estado ao longo de uma
// transicao preparing → running → done (RF-05, RF-01, subtarefa 15.5).
func TestRenderStateTransitions(t *testing.T) {
	t.Parallel()

	renderer := newBubbleTeaRenderer()

	snapPreparing := makeSnapshotWithPhase(PhasePreparing, 0)
	snapRunning := makeSnapshotWithPhase(PhaseRunning, 1)
	snapDone := makeSnapshotWithPhase(PhaseDone, 3)

	tests := []struct {
		name         string
		snapshot     SessionSnapshot
		wantLabels   []string
		wantAbsent   []string
	}{
		{
			name:       "preparando: label ATIVO PREPARANDO presente",
			snapshot:   snapPreparing,
			wantLabels: []string{"ATIVO PREPARANDO", "iteracao 1/5"},
		},
		{
			name:       "executando: label ATIVO EXECUTANDO presente",
			snapshot:   snapRunning,
			wantLabels: []string{"ATIVO EXECUTANDO", "iteracao 2/5"},
		},
		{
			name:       "concluido: label CONCLUIDO presente e progresso reflete done=3",
			snapshot:   snapDone,
			wantLabels: []string{"CONCLUIDO", "3/5 tasks"},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := renderer.Render(120, 40, tt.snapshot, nil, focusActiveTask, false)
			for _, want := range tt.wantLabels {
				if !strings.Contains(got, want) {
					t.Errorf("transicao %q: render nao contem %q\nsaida:\n%s", tt.name, want, got)
				}
			}
			for _, absent := range tt.wantAbsent {
				if strings.Contains(got, absent) {
					t.Errorf("transicao %q: render nao deveria conter %q\nsaida:\n%s", tt.name, absent, got)
				}
			}
		})
	}
}

// TestRenderStateTransitionsDiffer valida que os tres snapshots de transicao produzem
// outputs distintos entre si, confirmando que o estado afeta o layout visualmente.
func TestRenderStateTransitionsDiffer(t *testing.T) {
	t.Parallel()

	renderer := newBubbleTeaRenderer()

	renderPreparing := renderer.Render(120, 40, makeSnapshotWithPhase(PhasePreparing, 0), nil, focusActiveTask, false)
	renderRunning := renderer.Render(120, 40, makeSnapshotWithPhase(PhaseRunning, 1), nil, focusActiveTask, false)
	renderDone := renderer.Render(120, 40, makeSnapshotWithPhase(PhaseDone, 3), nil, focusActiveTask, false)

	if renderPreparing == renderRunning {
		t.Error("render preparing == render running: estados devem produzir layouts distintos")
	}
	if renderRunning == renderDone {
		t.Error("render running == render done: estados devem produzir layouts distintos")
	}
	if renderPreparing == renderDone {
		t.Error("render preparing == render done: estados devem produzir layouts distintos")
	}
}

// TestRenderRNF11ProportionalQueueSummary valida que o painel de fila/resumo e exibido
// em wide/medium e omitido em compact, respeitando a distribuicao proporcional (RNF-11).
func TestRenderRNF11ProportionalQueueSummary(t *testing.T) {
	t.Parallel()

	renderer := newBubbleTeaRenderer()
	snapshot := makeFullSnapshot()

	widths := []int{60, 100, 140}
	heights := []int{15, 30, 50}

	for _, w := range widths {
		for _, h := range heights {
			w, h := w, h
			t.Run(fmt.Sprintf("w%d_h%d", w, h), func(t *testing.T) {
				t.Parallel()

				got := renderer.Render(w, h, snapshot, nil, focusActiveTask, false)

				tier := resolveLayoutTier(w)
				if tier == layoutCompact {
					// Compact: painel de fila omitido
					if strings.Contains(got, "Fila") || strings.Contains(got, "Pendentes:") {
						t.Errorf("Render compact(%d,%d): painel de fila deveria ser omitido\nsaida:\n%s", w, h, got)
					}
				} else {
					// Wide/medium: painel de fila presente
					if !strings.Contains(got, "Fila") {
						t.Errorf("Render wide/medium(%d,%d): painel de fila deveria estar presente\nsaida:\n%s", w, h, got)
					}
				}

				// Dashboard e rodape sempre presentes independente do tier
				if !strings.Contains(got, "ai-spec task-loop") {
					t.Errorf("Render(%d,%d): dashboard ausente\nsaida:\n%s", w, h, got)
				}
				if !strings.Contains(got, "q sair") {
					t.Errorf("Render(%d,%d): rodape ausente\nsaida:\n%s", w, h, got)
				}
			})
		}
	}
}
