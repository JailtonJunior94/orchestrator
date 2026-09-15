package runtime

import (
	"context"

	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/memory/durable"
)

type MemoryPort interface {
	ClaimSession(sessionID string) error
	ReleaseSession(sessionID string) error
	RenewSession(sessionID string) error
	BuildContext(ctx context.Context, scope durable.MemoryScope) (durable.MemoryContext, error)
	RecordSession(ctx context.Context, in durable.SessionFacts) (durable.MemoryReport, error)
}

var _ MemoryPort = (*durable.Facade)(nil)
