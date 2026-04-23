package taskloop

import (
	"fmt"
	"strings"
	"sync"
	"time"
)

const defaultRecentEventLimit = 10

// EventSeverity categoriza o impacto do evento para apresentacao.
type EventSeverity string

const (
	SeverityInfo  EventSeverity = "info"
	SeverityWarn  EventSeverity = "warn"
	SeverityError EventSeverity = "error"
)

// BatchProgress representa o andamento agregado do lote.
type BatchProgress struct {
	Total      int
	Done       int
	Failed     int
	Blocked    int
	NeedsInput int
	Pending    int
	InProgress int
}

// NewBatchProgress valida e cria um progresso agregado consistente.
func NewBatchProgress(total, done, failed, blocked, needsInput, pending, inProgress int) (BatchProgress, error) {
	progress := BatchProgress{
		Total:      total,
		Done:       done,
		Failed:     failed,
		Blocked:    blocked,
		NeedsInput: needsInput,
		Pending:    pending,
		InProgress: inProgress,
	}
	if err := progress.Validate(); err != nil {
		return BatchProgress{}, err
	}
	return progress, nil
}

// Validate garante somas e contadores nao negativos.
func (p BatchProgress) Validate() error {
	values := map[string]int{
		"total":       p.Total,
		"done":        p.Done,
		"failed":      p.Failed,
		"blocked":     p.Blocked,
		"needs_input": p.NeedsInput,
		"pending":     p.Pending,
		"in_progress": p.InProgress,
	}
	for label, value := range values {
		if value < 0 {
			return fmt.Errorf("contador %s nao pode ser negativo", label)
		}
	}
	sum := p.Done + p.Failed + p.Blocked + p.NeedsInput + p.Pending + p.InProgress
	if sum != p.Total {
		return fmt.Errorf("soma dos contadores inconsistente: total=%d soma=%d", p.Total, sum)
	}
	return nil
}

// RecentEvent representa um evento recente truncado para apresentacao.
type RecentEvent struct {
	Sequence  int
	Timestamp time.Time
	Origin    AgentRole
	Kind      EventKind
	Message   string
	Severity  EventSeverity
	Task      TaskRef
	Tool      ToolName
	Phase     AgentPhase
}

// IterationSnapshot representa a iteracao corrente observada.
type IterationSnapshot struct {
	Sequence     int
	TaskID       string
	Title        string
	PreStatus    string
	PostStatus   string
	Tool         ToolName
	Role         AgentRole
	Phase        AgentPhase
	StartedAt    time.Time
	LastOutputAt *time.Time
}

// SessionSnapshot representa o estado serializavel da sessao.
type SessionSnapshot struct {
	Mode            string
	ActiveIteration *IterationSnapshot
	Progress        BatchProgress
	RecentEvents    []RecentEvent
	CurrentTool     ToolName
	CurrentRole     AgentRole
	CurrentPhase    AgentPhase
	Elapsed         time.Duration
	MaxIterations   int
	Interactive     bool
	LastError       *LoopFailure
}

// FinalSummary resume a execucao ao final do loop.
type FinalSummary struct {
	StopReason    string
	IterationsRun int
	ReportPath    string
	Progress      BatchProgress
	LastFailure   *LoopFailure
}

// LoopSession concentra as invariantes observaveis do task-loop.
type LoopSession struct {
	mu            sync.RWMutex
	mode          string
	maxIterations int
	interactive   bool
	startedAt     time.Time
	recentLimit   int

	progress      BatchProgress
	active        *IterationSnapshot
	recent        []RecentEvent
	lastFailure   *LoopFailure
	currentTool   ToolName
	currentRole   AgentRole
	currentPhase  AgentPhase
	iterationsRun int
	eventSequence int
}

// NewLoopSession cria a sessao observavel com limite configuravel de eventos recentes.
func NewLoopSession(mode string, maxIterations int, interactive bool, recentLimit int, startedAt time.Time) (*LoopSession, error) {
	if recentLimit <= 0 {
		recentLimit = defaultRecentEventLimit
	}
	if startedAt.IsZero() {
		startedAt = time.Now()
	}
	progress, err := NewBatchProgress(0, 0, 0, 0, 0, 0, 0)
	if err != nil {
		return nil, err
	}
	return &LoopSession{
		mode:          strings.TrimSpace(mode),
		maxIterations: maxIterations,
		interactive:   interactive,
		startedAt:     startedAt,
		recentLimit:   recentLimit,
		progress:      progress,
		currentPhase:  PhaseIdle,
	}, nil
}

// Apply reduz um evento tipado e devolve um snapshot determinista da sessao.
func (s *LoopSession) Apply(event LoopEvent) (SessionSnapshot, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := event.Kind.Validate(); err != nil {
		return SessionSnapshot{}, err
	}
	if err := event.Role.Validate(); err != nil {
		return SessionSnapshot{}, err
	}
	if err := event.Phase.Validate(); err != nil {
		return SessionSnapshot{}, err
	}
	if err := event.ErrorCode.Validate(); err != nil {
		return SessionSnapshot{}, err
	}
	if event.Progress != nil {
		if err := event.Progress.Validate(); err != nil {
			return SessionSnapshot{}, err
		}
		s.progress = *event.Progress
	}
	if event.Time.IsZero() {
		event.Time = s.startedAt
	}

	switch event.Kind {
	case EventSessionStarted:
		s.startedAt = event.Time
		s.currentPhase = PhaseIdle
		s.currentRole = ""
		s.currentTool = ""
	case EventIterationSelected:
		if event.Iteration <= 0 {
			return SessionSnapshot{}, fmt.Errorf("iteracao invalida: %d", event.Iteration)
		}
		if event.Task.ID == "" {
			return SessionSnapshot{}, fmt.Errorf("task obrigatoria para %s", event.Kind)
		}
		if s.active != nil && !s.active.Phase.IsTerminal() {
			return SessionSnapshot{}, newInvalidTransitionError(s.active.Phase, PhasePreparing)
		}
		nextPhase := event.Phase
		if nextPhase == "" {
			nextPhase = PhasePreparing
		}
		if !PhaseIdle.CanTransitionTo(nextPhase) {
			return SessionSnapshot{}, newInvalidTransitionError(PhaseIdle, nextPhase)
		}
		s.active = &IterationSnapshot{
			Sequence:   event.Iteration,
			TaskID:     event.Task.ID,
			Title:      event.Task.Title,
			PreStatus:  normalizeStatus(event.PreStatus),
			PostStatus: normalizeStatus(event.PostStatus),
			Tool:       event.Tool,
			Role:       defaultRole(event.Role),
			Phase:      nextPhase,
			StartedAt:  event.Time,
		}
		s.currentTool = event.Tool
		s.currentRole = defaultRole(event.Role)
		s.currentPhase = nextPhase
		if event.Iteration > s.iterationsRun {
			s.iterationsRun = event.Iteration
		}
	case EventPhaseChanged:
		if err := s.applyPhase(event, event.Phase); err != nil {
			return SessionSnapshot{}, err
		}
	case EventOutputObserved:
		if err := s.applyPhase(event, PhaseStreaming); err != nil {
			return SessionSnapshot{}, err
		}
		if s.active != nil {
			when := event.Time
			s.active.LastOutputAt = &when
		}
	case EventHeartbeatObserved:
		if s.active == nil {
			return SessionSnapshot{}, fmt.Errorf("heartbeat sem iteracao ativa")
		}
	case EventFailureObserved:
		failure := event.Failure
		if failure == nil {
			failure = NewLoopFailure(event.ErrorCode, event.Message, nil)
			failure.Tool = event.Tool
			failure.TaskID = event.Task.ID
			failure.Iteration = event.Iteration
		}
		targetPhase := event.Phase
		if targetPhase == "" {
			targetPhase = failure.Code.defaultPhase()
		}
		if err := s.applyPhase(event, targetPhase); err != nil {
			return SessionSnapshot{}, err
		}
		s.lastFailure = cloneFailure(failure)
	case EventProgressUpdated:
		// Validacao do progresso ja aconteceu no inicio da funcao.
	case EventSessionFinished:
		if s.active != nil && !s.active.Phase.IsTerminal() {
			return SessionSnapshot{}, newInvalidTransitionError(s.active.Phase, PhaseIdle)
		}
		s.active = nil
		s.currentTool = ""
		s.currentRole = ""
		s.currentPhase = PhaseIdle
	}

	s.appendRecent(event)

	return s.snapshotAtLocked(event.Time), nil
}

// SnapshotAt gera um snapshot determinista da sessao no instante informado.
func (s *LoopSession) SnapshotAt(now time.Time) SessionSnapshot {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.snapshotAtLocked(now)
}

func (s *LoopSession) snapshotAtLocked(now time.Time) SessionSnapshot {
	if now.IsZero() {
		now = s.startedAt
	}
	return SessionSnapshot{
		Mode:            s.mode,
		ActiveIteration: cloneIteration(s.active),
		Progress:        s.progress,
		RecentEvents:    cloneRecentEvents(s.recent),
		CurrentTool:     s.currentTool,
		CurrentRole:     s.currentRole,
		CurrentPhase:    s.currentPhase,
		Elapsed:         now.Sub(s.startedAt),
		MaxIterations:   s.maxIterations,
		Interactive:     s.interactive,
		LastError:       cloneFailure(s.lastFailure),
	}
}

// FinalSummary materializa o resumo final a partir do estado agregado.
func (s *LoopSession) FinalSummary(stopReason, reportPath string) FinalSummary {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.finalSummaryLocked(stopReason, reportPath)
}

func (s *LoopSession) finalSummaryLocked(stopReason, reportPath string) FinalSummary {
	return FinalSummary{
		StopReason:    strings.TrimSpace(stopReason),
		IterationsRun: s.iterationsRun,
		ReportPath:    strings.TrimSpace(reportPath),
		Progress:      s.progress,
		LastFailure:   cloneFailure(s.lastFailure),
	}
}

func (s *LoopSession) applyPhase(event LoopEvent, targetPhase AgentPhase) error {
	if s.active == nil {
		return newInvalidTransitionError(PhaseIdle, targetPhase)
	}
	if targetPhase == "" {
		return fmt.Errorf("fase obrigatoria para %s", event.Kind)
	}
	if s.active.Sequence != 0 && event.Iteration != 0 && event.Iteration != s.active.Sequence {
		return fmt.Errorf("iteracao %d nao corresponde a ativa %d", event.Iteration, s.active.Sequence)
	}
	current := s.active.Phase
	if current.IsTerminal() {
		return newInvalidTransitionError(current, targetPhase)
	}
	if !current.CanTransitionTo(targetPhase) {
		return newInvalidTransitionError(current, targetPhase)
	}
	s.active.Phase = targetPhase
	s.active.Tool = firstNonEmptyTool(event.Tool, s.active.Tool)
	s.active.Role = firstNonEmptyRole(event.Role, s.active.Role)
	if normalized := normalizeStatus(event.PreStatus); normalized != "" {
		s.active.PreStatus = normalized
	}
	if normalized := normalizeStatus(event.PostStatus); normalized != "" {
		s.active.PostStatus = normalized
	} else if targetPhase.IsTerminal() && s.active.PostStatus == "" {
		s.active.PostStatus = targetPhase.defaultStatus()
	}
	s.currentTool = s.active.Tool
	s.currentRole = s.active.Role
	s.currentPhase = targetPhase
	if targetPhase.IsTerminal() && s.lastFailure != nil && targetPhase == PhaseDone {
		s.lastFailure = nil
	}
	return nil
}

func (s *LoopSession) appendRecent(event LoopEvent) {
	s.eventSequence++
	recent := RecentEvent{
		Sequence:  s.eventSequence,
		Timestamp: event.Time,
		Origin:    firstNonEmptyRole(event.Role, s.currentRole),
		Kind:      event.Kind,
		Message:   strings.TrimSpace(event.Message),
		Severity:  severityFromEvent(event),
		Task:      event.Task,
		Tool:      firstNonEmptyTool(event.Tool, s.currentTool),
		Phase:     firstNonEmptyPhase(event.Phase, s.currentPhase),
	}
	s.recent = append(s.recent, recent)
	if len(s.recent) > s.recentLimit {
		s.recent = append([]RecentEvent(nil), s.recent[len(s.recent)-s.recentLimit:]...)
	}
}

func severityFromEvent(event LoopEvent) EventSeverity {
	switch event.Kind {
	case EventFailureObserved:
		return SeverityError
	case EventHeartbeatObserved:
		return SeverityInfo
	}
	if event.Phase == PhaseTimeout || event.Phase == PhaseAuthRequired {
		return SeverityWarn
	}
	return SeverityInfo
}

func newInvalidTransitionError(from, to AgentPhase) error {
	return &LoopFailure{
		Code:    ErrorInvalidPhaseTransition,
		Message: fmt.Sprintf("transicao de fase invalida: %s -> %s", from, to),
		Cause:   ErrInvalidPhaseTransition,
	}
}

func cloneFailure(failure *LoopFailure) *LoopFailure {
	if failure == nil {
		return nil
	}
	copy := *failure
	return &copy
}

func cloneIteration(iteration *IterationSnapshot) *IterationSnapshot {
	if iteration == nil {
		return nil
	}
	copy := *iteration
	if iteration.LastOutputAt != nil {
		when := *iteration.LastOutputAt
		copy.LastOutputAt = &when
	}
	return &copy
}

func cloneRecentEvents(events []RecentEvent) []RecentEvent {
	if len(events) == 0 {
		return nil
	}
	cloned := make([]RecentEvent, len(events))
	copy(cloned, events)
	return cloned
}

func defaultRole(role AgentRole) AgentRole {
	if role != "" {
		return role
	}
	return RoleExecutor
}

func firstNonEmptyTool(candidate, fallback ToolName) ToolName {
	if candidate != "" {
		return candidate
	}
	return fallback
}

func firstNonEmptyRole(candidate, fallback AgentRole) AgentRole {
	if candidate != "" {
		return candidate
	}
	return fallback
}

func firstNonEmptyPhase(candidate, fallback AgentPhase) AgentPhase {
	if candidate != "" {
		return candidate
	}
	return fallback
}

func (p AgentPhase) defaultStatus() string {
	switch p {
	case PhaseDone:
		return "done"
	case PhaseFailed:
		return "failed"
	case PhaseTimeout:
		return "failed"
	case PhaseAuthRequired:
		return "needs_input"
	default:
		return ""
	}
}
