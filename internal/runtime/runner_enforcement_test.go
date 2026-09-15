package runtime

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/client"
	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/events"
	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/specs"
)

type enforcementFakeClient struct {
	childEnv        []string
	childEnvCalled  bool
	handshakeWaiter client.HandshakeWaiter
	waiterCalled    bool
}

func (f *enforcementFakeClient) Open(context.Context, specs.Launcher, string) error { return nil }

func (f *enforcementFakeClient) Updates() <-chan events.Event { return nil }

func (f *enforcementFakeClient) Err() error { return nil }

func (f *enforcementFakeClient) Close() error { return nil }

func (f *enforcementFakeClient) SlowPublishes() uint64 { return 0 }

func (f *enforcementFakeClient) DroppedUpdates() uint64 { return 0 }

func (f *enforcementFakeClient) SetChildEnv(env []string) {
	f.childEnv = env
	f.childEnvCalled = true
}

func (f *enforcementFakeClient) SetHandshakeWaiter(w client.HandshakeWaiter) {
	f.handshakeWaiter = w
	f.waiterCalled = true
}

type fakeSuccessWaiter struct{}

func (fakeSuccessWaiter) Wait(context.Context) error { return nil }

func (fakeSuccessWaiter) Path() string { return "/fake/path" }

type fakeSuccessWaiterFactory struct{}

func (fakeSuccessWaiterFactory) NewWaiter() (client.HandshakeWaiter, func() error, error) {
	return fakeSuccessWaiter{}, func() error { return nil }, nil
}

type fakeErrWaiterFactory struct{ err error }

func (f fakeErrWaiterFactory) NewWaiter() (client.HandshakeWaiter, func() error, error) {
	return nil, nil, f.err
}

func TestApplyEnforcement_OpenCodeSetsChildEnvAndHandshakeWaiter(t *testing.T) {
	r := &ACPRunner{spec: specs.NewCatalog().OpenCode(), handshakeWaiterFactory: fakeSuccessWaiterFactory{}}
	fc := &enforcementFakeClient{}

	cleanup, err := r.applyEnforcement(fc, Job{EvidenceDir: t.TempDir()})
	if err != nil {
		t.Fatalf("applyEnforcement: %v", err)
	}
	defer cleanup()

	if !fc.childEnvCalled {
		t.Fatal("SetChildEnv não foi chamado para o OpenCode")
	}
	if !fc.waiterCalled {
		t.Fatal("SetHandshakeWaiter não foi chamado para o OpenCode")
	}
	for _, killSwitch := range specs.OpenCodeKillSwitchVars {
		for _, kv := range fc.childEnv {
			if strings.HasPrefix(kv, killSwitch+"=") {
				t.Fatalf("childEnv ainda contém o interruptor %q: %v", killSwitch, fc.childEnv)
			}
		}
	}
	foundSentinel := false
	foundOrchestrated := false
	for _, kv := range fc.childEnv {
		if strings.HasPrefix(kv, specs.OpenCodeGovernanceSentinelEnvVar+"=") {
			foundSentinel = true
		}
		if kv == specs.OpenCodeOrchestratedEnvVar+"=1" {
			foundOrchestrated = true
		}
	}
	if !foundSentinel {
		t.Errorf("childEnv não contém %s: %v", specs.OpenCodeGovernanceSentinelEnvVar, fc.childEnv)
	}
	if !foundOrchestrated {
		t.Errorf("childEnv não contém %s=1: %v", specs.OpenCodeOrchestratedEnvVar, fc.childEnv)
	}
}

func TestApplyEnforcement_OtherAgentsAreByteIdenticalNoop(t *testing.T) {
	for _, spec := range []specs.Spec{
		specs.NewCatalog().Claude(),
		specs.NewCatalog().Codex(),
		specs.NewCatalog().Copilot(),
	} {
		r := &ACPRunner{spec: spec, handshakeWaiterFactory: fakeSuccessWaiterFactory{}}
		fc := &enforcementFakeClient{}

		cleanup, err := r.applyEnforcement(fc, Job{})
		if err != nil {
			t.Fatalf("spec %q: applyEnforcement: %v", spec.ID, err)
		}
		cleanup()

		if fc.childEnvCalled {
			t.Errorf("spec %q: SetChildEnv foi chamado — regressão F1 (O-06)", spec.ID)
		}
		if fc.waiterCalled {
			t.Errorf("spec %q: SetHandshakeWaiter foi chamado — regressão F1 (O-06)", spec.ID)
		}
	}
}

func TestApplyEnforcement_HandshakeConstructionFailurePropagates(t *testing.T) {
	wantErr := errors.New("boom")
	r := &ACPRunner{spec: specs.NewCatalog().OpenCode(), handshakeWaiterFactory: fakeErrWaiterFactory{err: wantErr}}
	fc := &enforcementFakeClient{}

	_, err := r.applyEnforcement(fc, Job{EvidenceDir: t.TempDir()})
	if !errors.Is(err, wantErr) {
		t.Fatalf("applyEnforcement error = %v; want wrap of %v", err, wantErr)
	}
}

func TestApplyEnforcement_UnknownAgentIsNoop(t *testing.T) {
	r := &ACPRunner{spec: specs.Spec{ID: "not-in-registry"}, handshakeWaiterFactory: fakeSuccessWaiterFactory{}}
	fc := &enforcementFakeClient{}

	cleanup, err := r.applyEnforcement(fc, Job{})
	if err != nil {
		t.Fatalf("applyEnforcement: %v", err)
	}
	cleanup()

	if fc.childEnvCalled || fc.waiterCalled {
		t.Fatal("agente fora do registro não deveria disparar nenhuma chamada de enforcement")
	}
}

func TestOpenCodeArgvNeverContainsPureFlag(t *testing.T) {
	spec := specs.NewCatalog().OpenCode()
	bootstrap := spec.BootstrapArgs("", "", nil, specs.AccessModeRestricted, "/repo")
	argv := append(append([]string{}, spec.FixedArgs...), bootstrap...)
	for _, arg := range argv {
		if arg == "--pure" {
			t.Fatalf("argv contém --pure: %v", argv)
		}
	}
	for _, fb := range spec.Fallbacks {
		for _, arg := range fb.FixedArgs {
			if arg == "--pure" {
				t.Fatalf("fallback argv contém --pure: %v", fb.FixedArgs)
			}
		}
	}
}

func TestOpenCodeAgentEnvPolicyZeroValueWouldInheritIntact(t *testing.T) {
	var zero specs.EnvPolicy
	if got := zero.Apply(os.Environ()); got != nil {
		t.Fatalf("zero-value EnvPolicy.Apply(os.Environ()) = %v; want nil", got)
	}
}
