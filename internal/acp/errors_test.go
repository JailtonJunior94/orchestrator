package acp_test

import (
	"errors"
	"testing"

	"github.com/jailtonjunior/orchestrator/internal/acp"
)

func TestSentinelErrors_IsIdentifiable(t *testing.T) {
	sentinels := []error{
		acp.ErrProtocolVersionMismatch,
		acp.ErrHandshakeTimeout,
		acp.ErrSessionNotFound,
		acp.ErrSessionExpired,
		acp.ErrAgentNotAvailable,
		acp.ErrPathOutsideWorkDir,
	}

	for _, sentinel := range sentinels {
		wrapped := errors.Join(errors.New("context"), sentinel)
		if !errors.Is(wrapped, sentinel) {
			t.Errorf("errors.Is failed for sentinel %v", sentinel)
		}
	}
}
