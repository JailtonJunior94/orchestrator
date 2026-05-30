package embedded

// Extractor agrupa operacoes stateless do pacote embedded.
type Extractor struct{}

func NewExtractor() *Extractor { return &Extractor{} }
