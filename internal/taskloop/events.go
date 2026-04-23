package taskloop

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

// EventKind identifica o tipo canonico de evento observado no task-loop.
type EventKind string

const (
	EventSessionStarted    EventKind = "session_started"
	EventIterationSelected EventKind = "iteration_selected"
	EventPhaseChanged      EventKind = "phase_changed"
	EventOutputObserved    EventKind = "output_observed"
	EventHeartbeatObserved EventKind = "heartbeat_observed"
	EventProgressUpdated   EventKind = "progress_updated"
	EventFailureObserved   EventKind = "failure_observed"
	EventSessionFinished   EventKind = "session_finished"
)

func (k EventKind) Validate() error {
	switch k {
	case EventSessionStarted, EventIterationSelected, EventPhaseChanged, EventOutputObserved,
		EventHeartbeatObserved, EventProgressUpdated, EventFailureObserved, EventSessionFinished:
		return nil
	default:
		return fmt.Errorf("tipo de evento invalido: %q", k)
	}
}

// ToolName representa as ferramentas suportadas pelo task-loop.
type ToolName string

const (
	ToolClaude  ToolName = "claude"
	ToolCodex   ToolName = "codex"
	ToolGemini  ToolName = "gemini"
	ToolCopilot ToolName = "copilot"
)

// NewToolName valida e normaliza a ferramenta.
func NewToolName(raw string) (ToolName, error) {
	tool := ToolName(strings.TrimSpace(strings.ToLower(raw)))
	switch tool {
	case ToolClaude, ToolCodex, ToolGemini, ToolCopilot:
		return tool, nil
	case "":
		return "", errors.New("ferramenta obrigatoria")
	default:
		return "", fmt.Errorf("ferramenta invalida: %q", raw)
	}
}

// AgentRole identifica o papel ativo na iteracao.
type AgentRole string

const (
	RoleExecutor AgentRole = "executor"
	RoleReviewer AgentRole = "reviewer"
)

func (r AgentRole) Validate() error {
	switch r {
	case "", RoleExecutor, RoleReviewer:
		return nil
	default:
		return fmt.Errorf("papel invalido: %q", r)
	}
}

// AgentPhase representa os estados canonicos cross-tool.
type AgentPhase string

const (
	PhaseIdle         AgentPhase = "idle"
	PhasePreparing    AgentPhase = "preparing"
	PhaseRunning      AgentPhase = "running"
	PhaseStreaming    AgentPhase = "streaming"
	PhaseReviewing    AgentPhase = "reviewing"
	PhaseDone         AgentPhase = "done"
	PhaseFailed       AgentPhase = "failed"
	PhaseTimeout      AgentPhase = "timeout"
	PhaseAuthRequired AgentPhase = "auth_required"
)

var validPhaseTransitions = map[AgentPhase]map[AgentPhase]struct{}{
	PhaseIdle: {
		PhaseIdle:      {},
		PhasePreparing: {},
	},
	PhasePreparing: {
		PhasePreparing:    {},
		PhaseRunning:      {},
		PhaseFailed:       {},
		PhaseTimeout:      {},
		PhaseAuthRequired: {},
	},
	PhaseRunning: {
		PhaseRunning:      {},
		PhaseStreaming:    {},
		PhaseReviewing:    {},
		PhaseDone:         {},
		PhaseFailed:       {},
		PhaseTimeout:      {},
		PhaseAuthRequired: {},
	},
	PhaseStreaming: {
		PhaseStreaming:    {},
		PhaseReviewing:    {},
		PhaseDone:         {},
		PhaseFailed:       {},
		PhaseTimeout:      {},
		PhaseAuthRequired: {},
	},
	PhaseReviewing: {
		PhaseReviewing:    {},
		PhaseStreaming:    {},
		PhaseDone:         {},
		PhaseFailed:       {},
		PhaseTimeout:      {},
		PhaseAuthRequired: {},
	},
	PhaseDone: {
		PhaseIdle: {},
	},
	PhaseFailed: {
		PhaseIdle: {},
	},
	PhaseTimeout: {
		PhaseIdle: {},
	},
	PhaseAuthRequired: {
		PhaseIdle: {},
	},
}

// NewAgentPhase valida e normaliza a fase.
func NewAgentPhase(raw string) (AgentPhase, error) {
	phase := AgentPhase(strings.TrimSpace(strings.ToLower(raw)))
	if phase == "" {
		return "", errors.New("fase obrigatoria")
	}
	if err := phase.Validate(); err != nil {
		return "", err
	}
	return phase, nil
}

func (p AgentPhase) Validate() error {
	switch p {
	case "", PhaseIdle, PhasePreparing, PhaseRunning, PhaseStreaming, PhaseReviewing,
		PhaseDone, PhaseFailed, PhaseTimeout, PhaseAuthRequired:
		return nil
	default:
		return fmt.Errorf("fase invalida: %q", p)
	}
}

func (p AgentPhase) IsTerminal() bool {
	switch p {
	case PhaseDone, PhaseFailed, PhaseTimeout, PhaseAuthRequired:
		return true
	default:
		return false
	}
}

func (p AgentPhase) CanTransitionTo(next AgentPhase) bool {
	if err := p.Validate(); err != nil {
		return false
	}
	if err := next.Validate(); err != nil {
		return false
	}
	allowed, ok := validPhaseTransitions[p]
	if !ok {
		return false
	}
	_, ok = allowed[next]
	return ok
}

// ErrorCode classifica falhas tipadas do loop.
type ErrorCode string

const (
	ErrorInteractiveUnavailable ErrorCode = "interactive_unavailable"
	ErrorToolBinaryMissing      ErrorCode = "tool_binary_missing"
	ErrorToolAuthRequired       ErrorCode = "tool_auth_required"
	ErrorToolTimeout            ErrorCode = "tool_timeout"
	ErrorToolExecutionFailed    ErrorCode = "tool_execution_failed"
	ErrorInvalidPhaseTransition ErrorCode = "invalid_phase_transition"
	ErrorTaskIsolationViolation ErrorCode = "task_isolation_violation"
)

var (
	ErrInteractiveUnavailable = errors.New("modo iterativo indisponivel")
	ErrInvalidPhaseTransition = errors.New("transicao de fase invalida")
	ErrToolBinaryMissing      = errors.New("binario da ferramenta nao encontrado")
	ErrToolAuthRequired       = errors.New("autenticacao obrigatoria")
	ErrToolTimeout            = errors.New("timeout da ferramenta")
	ErrToolExecutionFailed    = errors.New("falha na execucao da ferramenta")
	ErrTaskIsolationViolation = errors.New("violacao de isolamento da task")
)

func (c ErrorCode) Validate() error {
	switch c {
	case "", ErrorInteractiveUnavailable, ErrorToolBinaryMissing, ErrorToolAuthRequired,
		ErrorToolTimeout, ErrorToolExecutionFailed, ErrorInvalidPhaseTransition,
		ErrorTaskIsolationViolation:
		return nil
	default:
		return fmt.Errorf("codigo de erro invalido: %q", c)
	}
}

func (c ErrorCode) defaultMessage() string {
	switch c {
	case ErrorInteractiveUnavailable:
		return "modo iterativo indisponivel"
	case ErrorToolBinaryMissing:
		return "binario da ferramenta nao encontrado"
	case ErrorToolAuthRequired:
		return "autenticacao obrigatoria para a ferramenta ativa"
	case ErrorToolTimeout:
		return "timeout da ferramenta"
	case ErrorToolExecutionFailed:
		return "falha na execucao da ferramenta"
	case ErrorInvalidPhaseTransition:
		return "transicao de estado invalida no task-loop"
	case ErrorTaskIsolationViolation:
		return "violacao de isolamento da task"
	default:
		return "falha no task-loop"
	}
}

func (c ErrorCode) sentinel() error {
	switch c {
	case ErrorInteractiveUnavailable:
		return ErrInteractiveUnavailable
	case ErrorToolBinaryMissing:
		return ErrToolBinaryMissing
	case ErrorToolAuthRequired:
		return ErrToolAuthRequired
	case ErrorToolTimeout:
		return ErrToolTimeout
	case ErrorToolExecutionFailed:
		return ErrToolExecutionFailed
	case ErrorInvalidPhaseTransition:
		return ErrInvalidPhaseTransition
	case ErrorTaskIsolationViolation:
		return ErrTaskIsolationViolation
	default:
		return nil
	}
}

func (c ErrorCode) defaultPhase() AgentPhase {
	switch c {
	case ErrorToolTimeout:
		return PhaseTimeout
	case ErrorToolAuthRequired:
		return PhaseAuthRequired
	default:
		return PhaseFailed
	}
}

// TaskRef identifica a task ativa no loop.
type TaskRef struct {
	ID    string
	Title string
}

// NewTaskRef cria uma referencia de task com validacao minima.
func NewTaskRef(id, title string) (TaskRef, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return TaskRef{}, errors.New("task id obrigatorio")
	}
	return TaskRef{
		ID:    id,
		Title: strings.TrimSpace(title),
	}, nil
}

// LoopFailure representa uma falha tipada do dominio do task-loop.
type LoopFailure struct {
	Code      ErrorCode
	Message   string
	Cause     error
	Tool      ToolName
	TaskID    string
	Iteration int
}

// NewLoopFailure cria uma falha com mensagem padrao quando necessario.
func NewLoopFailure(code ErrorCode, message string, cause error) *LoopFailure {
	if message == "" {
		message = code.defaultMessage()
	}
	return &LoopFailure{
		Code:    code,
		Message: message,
		Cause:   cause,
	}
}

func (f *LoopFailure) Error() string {
	if f == nil {
		return ""
	}
	if f.Message != "" {
		return f.Message
	}
	return f.Code.defaultMessage()
}

func (f *LoopFailure) Unwrap() error {
	if f == nil {
		return nil
	}
	if f.Cause != nil {
		return f.Cause
	}
	return f.Code.sentinel()
}

func (f *LoopFailure) Is(target error) bool {
	if f == nil {
		return false
	}
	if target == nil {
		return false
	}
	if target == f.Code.sentinel() {
		return true
	}
	if typed, ok := target.(*LoopFailure); ok {
		return typed != nil && typed.Code != "" && typed.Code == f.Code
	}
	if f.Cause != nil {
		return errors.Is(f.Cause, target)
	}
	return false
}

// LoopEvent representa um evento tipado consumido pelo redutor de sessao.
type LoopEvent struct {
	Time       time.Time
	Kind       EventKind
	Iteration  int
	Role       AgentRole
	Task       TaskRef
	Tool       ToolName
	Phase      AgentPhase
	Message    string
	ErrorCode  ErrorCode
	ExitCode   int
	PreStatus  string
	PostStatus string
	Progress   *BatchProgress
	Failure    *LoopFailure
}

// NewLoopEvent cria um evento com timestamp fixo para uso em testes e reducers.
func NewLoopEvent(kind EventKind, at time.Time) LoopEvent {
	return LoopEvent{
		Time: at,
		Kind: kind,
	}
}
