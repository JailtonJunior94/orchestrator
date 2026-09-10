package approval

type Event interface {
	EventName() string
}

type RoundStarted struct {
	Number int
}

type RoundCompleted struct {
	Number      int
	Verdict     Verdict
	Fingerprint Fingerprint
}

type ProofRejected struct {
	Number int
	Reason string
}

type FixRequested struct {
	Number int
}

type CycleClosed struct {
	Reason   StopReason
	Approved bool
	Rounds   int
}

func (e RoundStarted) EventName() string {
	return "round_started"
}

func (e RoundCompleted) EventName() string {
	return "round_completed"
}

func (e ProofRejected) EventName() string {
	return "proof_rejected"
}

func (e FixRequested) EventName() string {
	return "fix_requested"
}

func (e CycleClosed) EventName() string {
	return "cycle_closed"
}
