package hookcontract

import (
	"errors"
	"fmt"
	"slices"
)

type EventKind int

const (
	EventSessionStart EventKind = iota + 1
	EventBeforeTool
	EventAfterTool
	EventBeforeComplete
	EventSessionEnd
)

var allEventKinds = []EventKind{
	EventSessionStart,
	EventBeforeTool,
	EventAfterTool,
	EventBeforeComplete,
	EventSessionEnd,
}

var ErrUnknownEvent = errors.New("unknown canonical event")

func (e EventKind) Valid() bool {
	return e >= EventSessionStart && e <= EventSessionEnd
}

func (e EventKind) String() string {
	switch e {
	case EventSessionStart:
		return "session_start"
	case EventBeforeTool:
		return "before_tool"
	case EventAfterTool:
		return "after_tool"
	case EventBeforeComplete:
		return "before_complete"
	case EventSessionEnd:
		return "session_end"
	default:
		return "unknown"
	}
}

func ParseEventKind(s string) (EventKind, error) {
	for _, kind := range allEventKinds {
		if kind.String() == s {
			return kind, nil
		}
	}
	return 0, fmt.Errorf("%w: %q", ErrUnknownEvent, s)
}

func EventKinds() []EventKind {
	return slices.Clone(allEventKinds)
}
