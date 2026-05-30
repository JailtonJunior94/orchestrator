package runtime

import "time"

// Clock abstrai o acesso ao tempo atual para permitir determinismo em testes.
type Clock interface {
	Now() time.Time
}

// realClock é a implementação de produção que usa time.Now().
type realClock struct{}

var _ Clock = realClock{}

// RealClock retorna a implementação de Clock que usa time.Now().
func (c *Catalog) RealClock() Clock {
	return realClock{}
}

func (realClock) Now() time.Time {
	return time.Now()
}
