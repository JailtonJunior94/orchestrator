package runtime

import (
	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/client"
)

// Option é uma opção funcional para ACPRunner.
type Option func(*ACPRunner)

// WithClock injeta um Clock customizado (útil em testes).
func WithClock(c Clock) Option {
	return func(r *ACPRunner) {
		r.clock = c
	}
}

// WithProber injeta um Prober customizado para resolução do launcher.
func WithProber(p Prober) Option {
	return func(r *ACPRunner) {
		r.prober = p
	}
}

// WithClientFactory injeta uma ClientFactory customizada.
func WithClientFactory(f client.ClientFactory) Option {
	return func(r *ACPRunner) {
		r.factory = f
	}
}

// WithPersistenceFactory injeta uma PersistenceFactory customizada.
func WithPersistenceFactory(pf PersistenceFactory) Option {
	return func(r *ACPRunner) {
		r.persistenceFactory = pf
	}
}

// WithRenderer injeta um Renderer customizado.
func WithRenderer(rend Renderer) Option {
	return func(r *ACPRunner) {
		r.renderer = rend
	}
}
