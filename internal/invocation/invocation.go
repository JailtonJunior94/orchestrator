package invocation

import (
	"fmt"
	"os"
	"strconv"
)

const (
	_envDepth   = "AI_INVOCATION_DEPTH"
	_envMax     = "AI_INVOCATION_MAX"
	_defaultMax = 2
)

// Guard valida a profundidade de invocacao.
type Guard struct{}

// NewGuard cria um Guard stateless.
func NewGuard() *Guard {
	return &Guard{}
}

// CheckDepth verifica se a profundidade de invocação atual atingiu o limite máximo.
// Retorna erro se depth >= max.
func (g *Guard) CheckDepth() error {
	depth := g.readInt(_envDepth, 0)
	max := g.readInt(_envMax, _defaultMax)

	if depth >= max {
		return fmt.Errorf("limite de profundidade de invocação atingido (depth=%d, max=%d)", depth, max)
	}
	return nil
}

// IncrementDepth incrementa AI_INVOCATION_DEPTH no ambiente do processo atual.
func (g *Guard) IncrementDepth() {
	depth := g.readInt(_envDepth, 0)
	_ = os.Setenv(_envDepth, strconv.Itoa(depth+1))
}

func (g *Guard) readInt(key string, defaultValue int) int {
	raw := os.Getenv(key)
	if raw == "" {
		return defaultValue
	}
	v, err := strconv.Atoi(raw)
	if err != nil {
		return defaultValue
	}
	return v
}
