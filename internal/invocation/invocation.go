package invocation

import (
	"fmt"
	"os"
	"strconv"
)

const (
	envDepth   = "AI_INVOCATION_DEPTH"
	envMax     = "AI_INVOCATION_MAX"
	defaultMax = 2
)

// CheckDepth verifica se a profundidade de invocação atual atingiu o limite máximo.
// Retorna erro se depth >= max.
func CheckDepth() error {
	depth := readInt(envDepth, 0)
	max := readInt(envMax, defaultMax)

	if depth >= max {
		return fmt.Errorf("limite de profundidade de invocação atingido (depth=%d, max=%d)", depth, max)
	}
	return nil
}

// IncrementDepth incrementa AI_INVOCATION_DEPTH no ambiente do processo atual.
func IncrementDepth() {
	depth := readInt(envDepth, 0)
	_ = os.Setenv(envDepth, strconv.Itoa(depth+1))
}

func readInt(key string, defaultValue int) int {
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
