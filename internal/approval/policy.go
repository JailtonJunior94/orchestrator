package approval

const defaultMaxRounds = 5

type ApprovalPolicy struct {
	maxRounds int
}

type PolicyOption func(*policyConfig)

type StopDecision struct {
	Verdict             Verdict
	CurrentRound        int
	CurrentFingerprint  Fingerprint
	PreviousFingerprint Fingerprint
	Changed             bool
}

type policyConfig struct {
	maxRounds int
}

func NewApprovalPolicy(options ...PolicyOption) (ApprovalPolicy, error) {
	cfg := policyConfig{maxRounds: defaultMaxRounds}
	for _, option := range options {
		option(&cfg)
	}
	if cfg.maxRounds < 1 {
		return ApprovalPolicy{}, InvalidMaxRoundsError{Value: cfg.maxRounds}
	}
	return ApprovalPolicy(cfg), nil
}

func WithMaxRounds(maxRounds int) PolicyOption {
	return func(cfg *policyConfig) {
		cfg.maxRounds = maxRounds
	}
}

func (p ApprovalPolicy) MaxRounds() int {
	return p.maxRounds
}

func (p ApprovalPolicy) Decide(input StopDecision) (StopReason, bool) {
	if input.Verdict == VerdictBlocked {
		return ReasonBlockedInput, true
	}
	if input.CurrentFingerprint.Equal(input.PreviousFingerprint) {
		return ReasonNoConvergence, true
	}
	if !input.Changed {
		return ReasonEmptyDiff, true
	}
	if input.CurrentRound >= p.maxRounds {
		return ReasonMaxRounds, true
	}
	return 0, false
}
