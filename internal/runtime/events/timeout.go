package events

import (
	"fmt"
	"time"
)

// ActivityTimeout representa o tempo máximo de inatividade antes do cancelamento automático.
// Valor zero indica watchdog desabilitado.
type ActivityTimeout time.Duration

// NewActivityTimeout cria um ActivityTimeout a partir de uma duração.
// Retorna erro se d for negativo.
func NewActivityTimeout(d time.Duration) (ActivityTimeout, error) {
	if d < 0 {
		return 0, fmt.Errorf("activity timeout não pode ser negativo: %v", d)
	}
	return ActivityTimeout(d), nil
}

// Disabled retorna true quando o watchdog está desabilitado (timeout == 0).
func (t ActivityTimeout) Disabled() bool {
	return time.Duration(t) == 0
}

// Duration retorna a duração subjacente.
func (t ActivityTimeout) Duration() time.Duration {
	return time.Duration(t)
}
