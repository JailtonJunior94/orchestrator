package wrapper

// Executor agrupa operacoes stateless do pacote wrapper.
type Executor struct{}

func NewExecutor() *Executor { return &Executor{} }
