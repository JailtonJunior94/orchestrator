package taskloop

import (
	"encoding/json"
	"strconv"
	"time"

	"github.com/JailtonJunior94/ai-spec-harness/internal/procref"
)

type orchestratorLockIdentity struct {
	PID       int
	Hostname  string
	CLI       string
	Timestamp time.Time
}

func currentOrchestratorLockIdentity() orchestratorLockIdentity {
	ref := procref.CurrentProcessRef()
	return orchestratorLockIdentity{
		PID:       ref.PID,
		Hostname:  ref.Hostname,
		CLI:       "ai-spec",
		Timestamp: time.Now().UTC(),
	}
}

func (id orchestratorLockIdentity) marshal() []byte {
	encoded, err := json.Marshal(id)
	if err != nil {
		return []byte("{}")
	}
	return encoded
}

func parseOrchestratorLockIdentity(data []byte) (orchestratorLockIdentity, bool) {
	var id orchestratorLockIdentity
	if err := json.Unmarshal(data, &id); err != nil {
		return orchestratorLockIdentity{}, false
	}
	if id.PID <= 0 {
		return orchestratorLockIdentity{}, false
	}
	return id, true
}

func describeOrchestratorLockOwner(data []byte) string {
	id, ok := parseOrchestratorLockIdentity(data)
	if !ok {
		return "unknown owner (unidentified lock)"
	}
	return "pid=" + strconv.Itoa(id.PID) + " host=" + id.Hostname + " cli=" + id.CLI
}
