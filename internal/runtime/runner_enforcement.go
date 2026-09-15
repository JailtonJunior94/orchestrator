package runtime

import (
	"context"
	"fmt"
	"os"

	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/client"
	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/handshake"
	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/specs"
)

type HandshakeWaiterFactory interface {
	NewWaiter() (client.HandshakeWaiter, func() error, error)
}

type defaultHandshakeWaiterFactory struct{}

func NewDefaultHandshakeWaiterFactory() HandshakeWaiterFactory {
	return &defaultHandshakeWaiterFactory{}
}

func (f *defaultHandshakeWaiterFactory) NewWaiter() (client.HandshakeWaiter, func() error, error) {
	w, err := handshake.NewCatalog().NewWaiter("", handshake.DefaultTimeout)
	if err != nil {
		return nil, nil, err
	}
	return waiterAdapter{w}, w.Close, nil
}

type waiterAdapter struct {
	w *handshake.Waiter
}

func (a waiterAdapter) Wait(ctx context.Context) error { return a.w.Wait(ctx) }

func (a waiterAdapter) Path() string { return a.w.Path() }

func (r *ACPRunner) applyEnforcement(c client.Client, j Job) (func(), error) {
	noop := func() {}

	agent, err := specs.NewCatalog().AgentByID(r.spec.ID)
	if err != nil {
		return noop, nil
	}

	envPolicy := agent.EnvPolicy()
	needsHandshake := agent.RequiresHandshake()
	if envPolicy.IsZero() && !needsHandshake {
		return noop, nil
	}

	var childEnv []string
	if !envPolicy.IsZero() {
		childEnv = envPolicy.Apply(os.Environ())
	}

	cleanup := noop
	if needsHandshake {
		waiter, closeWaiter, werr := r.handshakeWaiterFactory.NewWaiter()
		if werr != nil {
			return noop, fmt.Errorf("runner: preparar handshake de governança: %w", werr)
		}
		if closeWaiter != nil {
			cleanup = func() { _ = closeWaiter() }
		}
		if pathed, ok := waiter.(interface{ Path() string }); ok {
			if childEnv == nil {
				childEnv = append([]string{}, os.Environ()...)
			}
			childEnv = append(childEnv,
				specs.OpenCodeGovernanceSentinelEnvVar+"="+pathed.Path(),
				specs.OpenCodeOrchestratedEnvVar+"=1",
			)
		}
		if hw, ok := c.(interface{ SetHandshakeWaiter(client.HandshakeWaiter) }); ok {
			hw.SetHandshakeWaiter(waiter)
		}
	}

	if ce, ok := c.(interface{ SetChildEnv([]string) }); ok {
		ce.SetChildEnv(childEnv)
	}
	return cleanup, nil
}
