package inventory

import (
	"fmt"
	"path/filepath"

	install "github.com/jailtonjunior/orchestrator/internal/install/domain"
)

const fileName = "inventory.json"

// StateDirResolver resolves the user state directory.
type StateDirResolver interface {
	UserStateDir() (string, error)
}

// ResolvePath returns the canonical inventory path for the requested scope.
func ResolvePath(resolver StateDirResolver, scope install.Scope, projectRoot string) (string, error) {
	if resolver == nil {
		return "", fmt.Errorf("dir resolver must not be nil")
	}
	if err := install.ValidateScope(scope); err != nil {
		return "", err
	}

	switch scope {
	case install.ScopeProject:
		if projectRoot == "" {
			return "", fmt.Errorf("project root must not be empty")
		}
		return filepath.Join(projectRoot, ".orq", "install", fileName), nil
	case install.ScopeGlobal:
		stateDir, err := resolver.UserStateDir()
		if err != nil {
			return "", err
		}
		return filepath.Join(stateDir, "orq", "install", fileName), nil
	default:
		return "", fmt.Errorf("unsupported scope %q", scope)
	}
}
