package events

import "fmt"

// CancelReason representa o motivo de cancelamento de uma sessão.
type CancelReason string

const (
	// CancelReasonNone indica ausência de cancelamento.
	CancelReasonNone CancelReason = "none"
	// CancelReasonActivityTimeout indica cancelamento por inatividade.
	CancelReasonActivityTimeout CancelReason = "activity_timeout"
	// CancelReasonContextCanceled indica cancelamento por context cancelado externamente.
	CancelReasonContextCanceled CancelReason = "context_canceled"
	// CancelReasonToolError indica cancelamento por erro em chamada de ferramenta.
	CancelReasonToolError CancelReason = "tool_error"
	// CancelReasonPermissionDenied indica cancelamento por permissão negada.
	CancelReasonPermissionDenied CancelReason = "permission_denied"
)

var _validCancelReasons = map[CancelReason]struct{}{
	CancelReasonNone:             {},
	CancelReasonActivityTimeout:  {},
	CancelReasonContextCanceled:  {},
	CancelReasonToolError:        {},
	CancelReasonPermissionDenied: {},
}

// ParseCancelReason valida e retorna um CancelReason a partir de uma string.
func (c *Catalog) ParseCancelReason(s string) (CancelReason, error) {
	r := CancelReason(s)
	if _, ok := _validCancelReasons[r]; !ok {
		return "", fmt.Errorf("motivo de cancelamento desconhecido: %q", s)
	}
	return r, nil
}
