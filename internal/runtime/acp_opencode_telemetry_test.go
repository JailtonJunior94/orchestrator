package runtime_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	airuntime "github.com/JailtonJunior94/ai-spec-harness/internal/runtime"
	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/acpfake"
	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/client"
	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/handshake"
	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/specs"
)

type failingHandshakeWaiter struct{ err error }

func (w failingHandshakeWaiter) Wait(context.Context) error { return w.err }

func (w failingHandshakeWaiter) Path() string { return "/fake/path" }

type failingHandshakeWaiterFactory struct{ err error }

func (f failingHandshakeWaiterFactory) NewWaiter() (client.HandshakeWaiter, func() error, error) {
	return failingHandshakeWaiter{err: f.err}, func() error { return nil }, nil
}

func TestACPIntegration_OpenCode_PreconditionRejectionMetricOnlyWithTelemetryOptIn(t *testing.T) {
	script := acpfake.NewScript().AppendAgentMessage("nunca deveria chegar aqui").AppendSessionEnd()

	run := func(t *testing.T, workDir string) error {
		t.Helper()
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		pfact, _ := newFakePersistenceFactory()
		runner := airuntime.NewACPRunner(
			specs.NewCatalog().OpenCode(), airuntime.NewCatalog().
				WithProber(proberOpenCodeBinary()), airuntime.NewCatalog().
				WithClientFactory(&fakeClientFactory{script: script, ctx: ctx, t: t}), airuntime.NewCatalog().
				WithPersistenceFactory(pfact), airuntime.NewCatalog().
				WithRenderer(&discardRenderer{}), airuntime.NewCatalog().
				WithHandshakeWaiterFactory(failingHandshakeWaiterFactory{err: handshake.ErrSignalNotReceived}),
		)
		job := airuntime.Job{
			Prompt:       "opencode precondition metric test",
			WorkDir:      workDir,
			EvidenceDir:  t.TempDir(),
			Quiet:        true,
			AccessMode:   specs.AccessModeRestricted,
			DisableHooks: true,
		}
		_, err := runner.Run(ctx, job)
		return err
	}

	t.Run("without opt-in", func(t *testing.T) {
		workDir := t.TempDir()
		if err := run(t, workDir); err == nil {
			t.Fatal("expected session to be refused without handshake signal")
		}
		if _, statErr := os.Stat(filepath.Join(workDir, ".agents", "telemetry.log")); !os.IsNotExist(statErr) {
			t.Fatalf("telemetry.log must not exist without GOVERNANCE_TELEMETRY=1; stat err = %v", statErr)
		}
	})

	t.Run("with opt-in", func(t *testing.T) {
		t.Setenv("GOVERNANCE_TELEMETRY", "1")
		workDir := t.TempDir()
		if err := run(t, workDir); err == nil {
			t.Fatal("expected session to be refused without handshake signal")
		}
		data, err := os.ReadFile(filepath.Join(workDir, ".agents", "telemetry.log"))
		if err != nil {
			t.Fatalf("telemetry.log missing with GOVERNANCE_TELEMETRY=1: %v", err)
		}
		if !strings.Contains(string(data), "precondition.rejected agent=opencode") {
			t.Fatalf("telemetry.log missing precondition.rejected entry: %s", data)
		}
	})
}
