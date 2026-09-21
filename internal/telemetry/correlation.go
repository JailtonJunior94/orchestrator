package telemetry

import (
	"errors"
	"strings"

	"github.com/JailtonJunior94/ai-spec-harness/internal/hookcontract"
)

var ErrCorrelationMissingSessionID = errors.New("envelope session_id required for correlation")

type Correlation struct {
	SessionID string
	TaskID    string
}

func CorrelationFromEnvelope(envelope hookcontract.Envelope, taskID string) (Correlation, error) {
	if strings.TrimSpace(envelope.SessionID) == "" {
		return Correlation{}, ErrCorrelationMissingSessionID
	}
	return Correlation{SessionID: envelope.SessionID, TaskID: taskID}, nil
}

func (c Correlation) Key() string {
	return c.SessionID + ":" + c.TaskID
}
