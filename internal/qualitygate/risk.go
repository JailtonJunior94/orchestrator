package qualitygate

import (
	"errors"
	"fmt"
	"strings"
)

type Risk int

const (
	RiskLow Risk = iota + 1
	RiskMedium
	RiskHigh
)

var allRisks = []Risk{RiskLow, RiskMedium, RiskHigh}

var ErrUnknownRisk = errors.New("qualitygate: unknown risk level")

func (r Risk) Valid() bool {
	return r >= RiskLow && r <= RiskHigh
}

func (r Risk) String() string {
	switch r {
	case RiskLow:
		return "low"
	case RiskMedium:
		return "medium"
	case RiskHigh:
		return "high"
	default:
		return "unknown"
	}
}

func ParseRisk(s string) (Risk, error) {
	normalized := strings.ToLower(strings.TrimSpace(s))
	for _, risk := range allRisks {
		if risk.String() == normalized {
			return risk, nil
		}
	}
	return 0, fmt.Errorf("%w: %q", ErrUnknownRisk, s)
}

type TaskType string

var ErrEmptyTaskType = errors.New("qualitygate: task type must not be empty")

func NewTaskType(s string) (TaskType, error) {
	trimmed := strings.TrimSpace(s)
	if trimmed == "" {
		return "", ErrEmptyTaskType
	}
	return TaskType(trimmed), nil
}

const DefaultTaskType TaskType = "default"

const DefaultRisk = RiskMedium
