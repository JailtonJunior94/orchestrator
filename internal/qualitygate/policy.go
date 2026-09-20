package qualitygate

import (
	"bytes"
	"errors"
	"fmt"

	"gopkg.in/yaml.v3"
)

const SupportedPolicyVersion = 1

var ErrInvalidPolicy = errors.New("qualitygate: invalid quality gate policy declaration")

var ErrPolicyNotDeclared = errors.New("qualitygate: no policy declared for task type and risk")

type policyKey struct {
	taskType TaskType
	risk     Risk
}

type PolicySet struct {
	version int
	entries map[policyKey]Selection
}

func (p PolicySet) Lookup(taskType TaskType, risk Risk) (Selection, error) {
	selection, ok := p.entries[policyKey{taskType: taskType, risk: risk}]
	if !ok {
		return Selection{}, fmt.Errorf("%w: task_type=%s risk=%s", ErrPolicyNotDeclared, taskType, risk)
	}
	return selection, nil
}

func (p PolicySet) Version() int {
	return p.version
}

type rawPolicyFile struct {
	Version  int              `yaml:"version"`
	Policies []rawPolicyEntry `yaml:"policies"`
}

type rawPolicyEntry struct {
	TaskType string   `yaml:"task_type"`
	Risk     string   `yaml:"risk"`
	Required []string `yaml:"required"`
	Optional []string `yaml:"optional"`
}

func DecodePolicySet(data []byte) (PolicySet, error) {
	decoder := yaml.NewDecoder(bytes.NewReader(data))
	decoder.KnownFields(true)

	var raw rawPolicyFile
	if err := decoder.Decode(&raw); err != nil {
		return PolicySet{}, fmt.Errorf("%w: %s", ErrInvalidPolicy, err)
	}

	if raw.Version != SupportedPolicyVersion {
		return PolicySet{}, fmt.Errorf("%w: unsupported version %d, expected %d", ErrInvalidPolicy, raw.Version, SupportedPolicyVersion)
	}
	if len(raw.Policies) == 0 {
		return PolicySet{}, fmt.Errorf("%w: no policies declared", ErrInvalidPolicy)
	}

	entries := make(map[policyKey]Selection, len(raw.Policies))
	for index, entry := range raw.Policies {
		taskType, err := NewTaskType(entry.TaskType)
		if err != nil {
			return PolicySet{}, fmt.Errorf("%w: policies[%d]: %s", ErrInvalidPolicy, index, err)
		}
		risk, err := ParseRisk(entry.Risk)
		if err != nil {
			return PolicySet{}, fmt.Errorf("%w: policies[%d]: %s", ErrInvalidPolicy, index, err)
		}
		required, err := parseCheckKindList(entry.Required)
		if err != nil {
			return PolicySet{}, fmt.Errorf("%w: policies[%d]: %s", ErrInvalidPolicy, index, err)
		}
		optional, err := parseCheckKindList(entry.Optional)
		if err != nil {
			return PolicySet{}, fmt.Errorf("%w: policies[%d]: %s", ErrInvalidPolicy, index, err)
		}
		selection, err := NewSelection(required, optional)
		if err != nil {
			return PolicySet{}, fmt.Errorf("%w: policies[%d]: %s", ErrInvalidPolicy, index, err)
		}

		key := policyKey{taskType: taskType, risk: risk}
		if _, exists := entries[key]; exists {
			return PolicySet{}, fmt.Errorf("%w: duplicate entry for task_type=%s risk=%s", ErrInvalidPolicy, taskType, risk)
		}
		entries[key] = selection
	}

	return PolicySet{version: raw.Version, entries: entries}, nil
}

func parseCheckKindList(values []string) ([]CheckKind, error) {
	kinds := make([]CheckKind, 0, len(values))
	for _, value := range values {
		kind, err := ParseCheckKind(value)
		if err != nil {
			return nil, err
		}
		kinds = append(kinds, kind)
	}
	return kinds, nil
}
