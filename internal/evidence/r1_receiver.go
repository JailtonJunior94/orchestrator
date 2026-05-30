package evidence

// Validator agrupa operacoes stateless do pacote evidence.
type Validator struct{}

func NewValidator() *Validator { return &Validator{} }
