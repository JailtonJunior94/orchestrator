package hookcontract

import (
	"errors"
	"fmt"
)

const MaxInvocationDepth = 1

var ErrRecursionGuard = errors.New("hook to tool to hook recursion blocked by construction")

func CheckRecursionGuard(depth int) (Result, error) {
	if depth < 0 {
		return Result{}, fmt.Errorf("%w: negative invocation depth %d", ErrRecursionGuard, depth)
	}
	if depth >= MaxInvocationDepth {
		reason := fmt.Sprintf(
			"invocation depth %d reached the recursion guard limit %d — hook to tool to hook chain blocked",
			depth, MaxInvocationDepth)
		result, err := NewResult(DecisionBlock, reason, "", "recursion-guard")
		if err != nil {
			return NewNotApplicable(reason), nil
		}
		return result, nil
	}
	return NewAllow(), nil
}

func NextInvocationDepth(envelope Envelope) int {
	return envelope.InvocationDepth + 1
}
