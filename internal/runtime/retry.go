package runtime

import (
	"errors"
	"time"
)

// RetryClassifier decide se um erro de sessão ACP é transitório (reexecutável)
// ou fatal (falha imediata). ADR-018 §"Retry/backoff".
type RetryClassifier interface {
	// IsTransient retorna true quando o erro é candidato a reexecução.
	// Erros fatais como ErrPermissionDenied, ErrUnsupportedTool retornam false.
	IsTransient(err error) bool
}

// defaultRetryClassifier é a implementação de produção do RetryClassifier.
// Erros fatais (lista explícita) retornam false; demais erros retornam true.
type defaultRetryClassifier struct{}

// NewRetryClassifier retorna a implementação de produção de RetryClassifier.
func NewRetryClassifier() RetryClassifier {
	return &defaultRetryClassifier{}
}

// IsTransient classifica o erro como transitório (true) ou fatal (false).
// Erros fatais conhecidos:
//   - ErrPermissionDenied: agente solicitou permissão não concedida.
//   - ErrUnsupportedTool: tool não suportado — não há recuperação possível.
//   - ErrSessionAborted: sessão abortada por erro interno não recuperável.
//   - ErrLauncherUnavailable: nenhum launcher disponível — não é transitório.
//
// Todos os demais erros (IO, inatividade, launcher temporário, contexto)
// são considerados transitórios e passíveis de reexecução.
func (c *defaultRetryClassifier) IsTransient(err error) bool {
	if err == nil {
		return false
	}
	for _, fatal := range []error{
		ErrPermissionDenied,
		ErrUnsupportedTool,
		ErrSessionAborted,
		ErrLauncherUnavailable,
	} {
		if errors.Is(err, fatal) {
			return false
		}
	}
	return true
}

// BackoffDuration calcula o tempo de espera para a tentativa `attempt` (0-indexed).
// Fórmula: base * multiplier^attempt.
// Quando multiplier <= 0, retorna 0 (sem espera — comportamento F1 default).
// Quando base == 0, retorna 0.
func BackoffDuration(base time.Duration, multiplier float64, attempt int) time.Duration {
	if base <= 0 || multiplier <= 0 || attempt <= 0 {
		return 0
	}
	factor := multiplier
	for i := 1; i < attempt; i++ {
		factor *= multiplier
	}
	return time.Duration(float64(base) * factor)
}
