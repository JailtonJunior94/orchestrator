package probe

import "errors"

// ErrLauncherUnavailable é retornado quando nenhum launcher (binary ou npx) está disponível.
// Re-exportado em internal/runtime.ErrLauncherUnavailable para compatibilidade da API pública.
var ErrLauncherUnavailable = errors.New("launcher unavailable")
