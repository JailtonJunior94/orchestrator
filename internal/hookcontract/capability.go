package hookcontract

import (
	"errors"
	"fmt"
	"strings"
)

type SupportState int

const (
	SupportVerified SupportState = iota + 1
	SupportAdapter
	SupportUnsupported
)

func (s SupportState) Valid() bool {
	return s >= SupportVerified && s <= SupportUnsupported
}

func (s SupportState) String() string {
	switch s {
	case SupportVerified:
		return "verified"
	case SupportAdapter:
		return "adapter"
	case SupportUnsupported:
		return "unsupported"
	default:
		return "unknown"
	}
}

var (
	ErrInvalidCapability  = errors.New("invalid hook capability")
	ErrCapabilityNotFound = errors.New("capability not found in registry")
)

type Capability struct {
	provider   string
	event      EventKind
	state      SupportState
	nativeKey  string
	blocking   bool
	limitation string
	valid      bool
}

func NewCapability(provider string, event EventKind, state SupportState, nativeKey string, blocking bool, limitation string) (Capability, error) {
	if strings.TrimSpace(provider) == "" {
		return Capability{}, fmt.Errorf("%w: empty provider", ErrInvalidCapability)
	}
	if !event.Valid() {
		return Capability{}, fmt.Errorf("%w: invalid event %d", ErrInvalidCapability, int(event))
	}
	if !state.Valid() {
		return Capability{}, fmt.Errorf("%w: invalid support state %d", ErrInvalidCapability, int(state))
	}
	if state == SupportAdapter && strings.TrimSpace(limitation) == "" {
		return Capability{}, fmt.Errorf("%w: provider %q event %s adapter support requires a limitation", ErrInvalidCapability, provider, event)
	}
	return Capability{
		provider:   provider,
		event:      event,
		state:      state,
		nativeKey:  nativeKey,
		blocking:   blocking,
		limitation: limitation,
		valid:      true,
	}, nil
}

func (c Capability) Provider() string {
	return c.provider
}

func (c Capability) Event() EventKind {
	return c.event
}

func (c Capability) State() SupportState {
	return c.state
}

func (c Capability) NativeKey() string {
	return c.nativeKey
}

func (c Capability) Blocking() bool {
	return c.blocking
}

func (c Capability) Limitation() string {
	return c.limitation
}

func (c Capability) Valid() bool {
	return c.valid
}

type CapabilityRegistry interface {
	Lookup(provider string, event EventKind) (Capability, error)
	Providers() []string
	Events() []EventKind
}
