package hookcontract

import (
	"fmt"
	"strings"
)

const legacyExitCodeGateID = "legacy-exit-code-translation"

var exitCodeDecisions = map[int]Decision{
	0: DecisionAllow,
	1: DecisionBlock,
	2: DecisionBlock,
}

type ExitCodeTranslator interface {
	ToDecision(code int, stderr string, critical bool) Result
	ToExitCode(result Result) int
}

type strictExitCodeTranslator struct{}

func NewExitCodeTranslator() ExitCodeTranslator {
	return strictExitCodeTranslator{}
}

func (strictExitCodeTranslator) ToDecision(code int, stderr string, critical bool) Result {
	decision, mapped := exitCodeDecisions[code]
	if !mapped {
		return unmappedResult(code, stderr, critical)
	}

	switch decision {
	case DecisionAllow:
		return NewAllow()
	case DecisionBlock:
		reason := strings.TrimSpace(stderr)
		if reason == "" {
			reason = fmt.Sprintf("hook exited with blocking status %d", code)
		}
		result, err := NewResult(DecisionBlock, reason, "", legacyExitCodeGateID)
		if err != nil {
			return unmappedResult(code, stderr, critical)
		}
		return result
	default:
		return unmappedResult(code, stderr, critical)
	}
}

func (strictExitCodeTranslator) ToExitCode(result Result) int {
	switch result.Decision() {
	case DecisionAllow, DecisionNotApplicable, DecisionWarn:
		return 0
	case DecisionBlock, DecisionError:
		return 2
	default:
		return 2
	}
}

func unmappedResult(code int, stderr string, critical bool) Result {
	reason := strings.TrimSpace(stderr)
	if reason == "" {
		reason = fmt.Sprintf("hook exited with unmapped status %d", code)
	}
	if critical {
		result, err := NewResult(DecisionError, reason, "", "")
		if err != nil {
			return NewNotApplicable(reason)
		}
		return result
	}
	result, err := NewResult(DecisionWarn, reason, "", "")
	if err != nil {
		return NewNotApplicable(reason)
	}
	return result
}
