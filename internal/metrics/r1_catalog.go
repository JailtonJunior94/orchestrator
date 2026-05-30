package metrics

// Catalog agrupa operacoes stateless do pacote.
type Catalog struct{}

func NewCatalog() *Catalog {
	return &Catalog{}
}
