package runtime

import "errors"

// ErrLauncherUnavailable é retornado quando nenhum launcher (binary ou npx) está disponível.
var ErrLauncherUnavailable = errors.New("launcher unavailable")

// ErrActivityTimeout é retornado (via context.CancelCause) quando o watchdog detecta
// que o intervalo desde o último evento excede o ActivityTimeout configurado.
var ErrActivityTimeout = errors.New("activity timeout")

// ErrPermissionDenied é retornado quando o agente emite requestPermission (RF-16).
var ErrPermissionDenied = errors.New("permission denied")

// ErrSessionAborted é retornado quando a sessão é abortada por erro interno.
var ErrSessionAborted = errors.New("session aborted")

// ErrUnsupportedTool é retornado quando o tool solicitado não é suportado pelo runtime.
var ErrUnsupportedTool = errors.New("unsupported tool")

// ErrInvalidEvent é retornado quando um evento inválido é processado.
var ErrInvalidEvent = errors.New("invalid event")
