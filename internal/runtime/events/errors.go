package events

import "errors"

// ErrInvalidTransition é retornado quando uma transição de estado inválida é tentada.
var ErrInvalidTransition = errors.New("transição de estado inválida")

// ErrInvalidEvent é retornado quando um evento inválido é fornecido.
var ErrInvalidEvent = errors.New("evento inválido")
