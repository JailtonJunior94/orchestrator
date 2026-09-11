package precondition

import (
	"os"

	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/specs"
)

type Environment interface {
	Getenv(key string) string
}

type OSEnvironment struct{}

func (OSEnvironment) Getenv(key string) string {
	return os.Getenv(key)
}

func EvaluateNoKillSwitch(env Environment, killSwitchVars []string) specs.PreconditionState {
	for _, name := range killSwitchVars {
		if env.Getenv(name) != "" {
			return specs.PreconditionInert
		}
	}
	return specs.PreconditionCurrent
}

func EvaluateHandshake() specs.PreconditionState {
	return specs.PreconditionUnknown
}
