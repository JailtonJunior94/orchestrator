package events

import "fmt"

// SessionState representa o estado atual de uma sessão ACP.
type SessionState string

const (
	// StateInit é o estado inicial antes de qualquer evento.
	StateInit SessionState = "init"
	// StateRunning é o estado após o primeiro evento de update.
	StateRunning SessionState = "running"
	// StateAwaiting é o estado quando o agente aguarda permissão.
	StateAwaiting SessionState = "awaiting"
	// StateClosed é o estado terminal normal (session_end).
	StateClosed SessionState = "closed"
	// StateCanceled é o estado terminal por cancelamento externo.
	StateCanceled SessionState = "canceled"
)

// _validTransitions define o grafo de transições permitidas.
// RF-16: requestPermission só pode ser cancelado, não retomado.
var _validTransitions = map[SessionState][]SessionState{
	StateInit:     {StateRunning, StateCanceled},
	StateRunning:  {StateAwaiting, StateClosed, StateCanceled},
	StateAwaiting: {StateCanceled},
	StateClosed:   {},
	StateCanceled: {},
}

// Transition tenta avançar para o próximo estado.
// Retorna o novo estado ou ErrInvalidTransition se a transição for proibida.
func (s SessionState) Transition(next SessionState) (SessionState, error) {
	allowed, known := _validTransitions[s]
	if !known {
		return s, fmt.Errorf("%w: estado desconhecido %q", ErrInvalidTransition, s)
	}
	for _, a := range allowed {
		if a == next {
			return next, nil
		}
	}
	return s, fmt.Errorf("%w: %q → %q não permitido", ErrInvalidTransition, s, next)
}
