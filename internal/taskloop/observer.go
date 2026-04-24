package taskloop

// LoopObserver consome snapshots e eventos tipados sem acoplar renderizacao ao dominio.
type LoopObserver interface {
	Start(SessionSnapshot) error
	Consume(LoopEvent) error
	Finish(FinalSummary) error
}

// NoopObserver e a implementacao padrao sem efeitos colaterais.
type NoopObserver struct{}

// NewNoopObserver cria o observer default para fallback seguro.
func NewNoopObserver() NoopObserver {
	return NoopObserver{}
}

func (NoopObserver) Start(SessionSnapshot) error { return nil }

func (NoopObserver) Consume(LoopEvent) error { return nil }

func (NoopObserver) Finish(FinalSummary) error { return nil }
