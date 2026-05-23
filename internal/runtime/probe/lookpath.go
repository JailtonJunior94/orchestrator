package probe

import "os/exec"

// LookPather é uma interface injetável para resolução de binários no PATH.
// Permite testes sem chamar exec.LookPath diretamente.
type LookPather interface {
	LookPath(name string) (string, error)
}

// osLookPather é a implementação padrão que delega para exec.LookPath.
type osLookPather struct{}

// OsLookPather retorna a implementação padrão de LookPather que usa exec.LookPath.
func OsLookPather() LookPather {
	return osLookPather{}
}

func (osLookPather) LookPath(name string) (string, error) {
	return exec.LookPath(name)
}
