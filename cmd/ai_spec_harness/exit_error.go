package aispecharness

import (
	"errors"
	"fmt"
)

// exitError sinaliza um codigo de saida especifico a partir de um handler RunE
// sem chamar os.Exit fora de main (Regra 5.16). O handler e responsavel por
// emitir as mensagens ao usuario antes de retornar este erro; Execute nao
// reimprime a mensagem para preservar a saida original.
type exitError struct {
	code int
}

// newExitError constroi um exitError com o codigo de saida informado.
func newExitError(code int) *exitError {
	return &exitError{code: code}
}

// Error satisfaz a interface error com uma mensagem diagnostica estavel.
func (e *exitError) Error() string {
	return fmt.Sprintf("exit code %d", e.code)
}

// ExitCode retorna o codigo de saida associado.
func (e *exitError) ExitCode() int {
	return e.code
}

// ExitResolver traduz erros retornados por Execute em codigos de saida do
// processo. E um receiver stateless: a logica de mapeamento vive como metodo
// para manter o pacote livre de funcoes standalone (Regra 1).
type ExitResolver struct{}

// NewExitResolver constroi um ExitResolver.
func NewExitResolver() ExitResolver {
	return ExitResolver{}
}

// CodeFor extrai o codigo de saida de um erro retornado por Execute,
// retornando 1 como padrao quando o erro nao carrega um codigo explicito.
func (ExitResolver) CodeFor(err error) int {
	var ee *exitError
	if errors.As(err, &ee) {
		return ee.ExitCode()
	}
	return 1
}
