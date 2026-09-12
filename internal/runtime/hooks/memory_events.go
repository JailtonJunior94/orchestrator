package hooks

const (
	PointMemoryFactRecorded          = "memory.fact_recorded"
	PointMemoryFactArchived          = "memory.fact_archived"
	PointMemoryFactPromoted          = "memory.fact_promoted"
	PointMemoryContradictionDetected = "memory.contradiction_detected"
	PointMemorySecretRedacted        = "memory.secret_redacted"
	PointMemoryCompactionExecuted    = "memory.compaction_executed"
	PointMemoryBatonTransferred      = "memory.baton_transferred"
)

type MemoryFactRecordedEvent struct {
	SessionID    string
	CLI          string
	TaskFileName string
	Count        int
}

func (e MemoryFactRecordedEvent) Kind() string { return PointMemoryFactRecorded }

type MemoryFactArchivedEvent struct {
	SessionID    string
	CLI          string
	TaskFileName string
	Count        int
}

func (e MemoryFactArchivedEvent) Kind() string { return PointMemoryFactArchived }

type MemoryFactPromotedEvent struct {
	SessionID    string
	CLI          string
	TaskFileName string
	Count        int
}

func (e MemoryFactPromotedEvent) Kind() string { return PointMemoryFactPromoted }

type MemoryContradictionDetectedEvent struct {
	SessionID    string
	CLI          string
	TaskFileName string
	Count        int
}

func (e MemoryContradictionDetectedEvent) Kind() string { return PointMemoryContradictionDetected }

type MemorySecretRedactedEvent struct {
	SessionID    string
	CLI          string
	TaskFileName string
	Count        int
}

func (e MemorySecretRedactedEvent) Kind() string { return PointMemorySecretRedacted }

type MemoryCompactionExecutedEvent struct {
	SessionID    string
	CLI          string
	TaskFileName string
	Count        int
}

func (e MemoryCompactionExecutedEvent) Kind() string { return PointMemoryCompactionExecuted }

type MemoryBatonTransferredEvent struct {
	SessionID    string
	CLI          string
	TaskFileName string
	Claimed      bool
}

func (e MemoryBatonTransferredEvent) Kind() string { return PointMemoryBatonTransferred }
