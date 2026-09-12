package runtime

import (
	"time"

	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/events"
	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/memory"
	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/specs"
)

type RuntimeConfig struct {
	Timeout events.ActivityTimeout

	MaxRetries int

	RetryBackoffMultiplier float64

	Concurrent int

	BatchSize           int
	MaxBugfixIterations int

	HandoffLeaseTTL time.Duration

	DurableMemoryEnabled bool
}

func (c *RuntimeConfig) ApplyDefaults() {
	if c.Concurrent <= 0 {
		c.Concurrent = 1
	}
	if c.BatchSize <= 0 {
		c.BatchSize = 1
	}
}

type Job struct {
	Prompt string

	WorkDir string

	EvidenceDir string

	RuntimeConfig

	Quiet bool

	Model string

	ReasoningEffort string

	AccessMode specs.AccessMode

	AddDirs []string

	TaskFileName string

	MCPNested bool

	NoNormalize bool

	MemoryLimits memory.Limits

	DisableHooks bool

	TasksDir string

	MemoryLimitsExplicit bool

	SkipDriftGuard bool

	WindowClass specs.WindowClass

	AutoReview bool
}
