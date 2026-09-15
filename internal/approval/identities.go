package approval

import (
	"fmt"
	"strings"
)

type TaskIdentity struct {
	value string
}

type AgentIdentity struct {
	value string
}

func NewTaskIdentity(value string) (TaskIdentity, error) {
	v := strings.TrimSpace(value)
	if v == "" {
		return TaskIdentity{}, fmt.Errorf("%w: empty task identity", ErrInvalidIdentity)
	}
	return TaskIdentity{value: v}, nil
}

func NewAgentIdentity(value string) (AgentIdentity, error) {
	v := strings.TrimSpace(value)
	if v == "" {
		return AgentIdentity{}, fmt.Errorf("%w: empty agent identity", ErrInvalidIdentity)
	}
	return AgentIdentity{value: v}, nil
}

func (i TaskIdentity) String() string {
	return i.value
}

func (i TaskIdentity) Zero() bool {
	return i.value == ""
}

func (i AgentIdentity) String() string {
	return i.value
}

func (i AgentIdentity) Zero() bool {
	return i.value == ""
}
