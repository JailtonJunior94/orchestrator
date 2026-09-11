package runtime_test

import (
	"context"
	"testing"
	"time"

	airuntime "github.com/JailtonJunior94/ai-spec-harness/internal/runtime"
	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/acpfake"
	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/events"
	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/specs"
)

func TestNonRegression_ThreeRemainingAgents_DefaultJobZeroValue(t *testing.T) {
	cases := []struct {
		name   string
		spec   specs.Spec
		prober airuntime.Prober
	}{
		{name: "claude", spec: specs.NewCatalog().Claude(), prober: proberBinary()},
		{name: "codex", spec: specs.NewCatalog().Codex(), prober: proberCodexBinary()},
		{name: "copilot", spec: specs.NewCatalog().Copilot(), prober: proberCopilotBinary()},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			script := acpfake.NewScript().
				AppendAgentMessage("nao-regressao " + tc.name).
				AppendSessionEnd()

			pfact, _ := newFakePersistenceFactory()
			runner := buildRunnerWithSpec(t, ctx, tc.spec, tc.prober, script, pfact)

			job := airuntime.Job{
				Prompt:      "non-regression default job",
				WorkDir:     workDirWithAgentsMD(t),
				EvidenceDir: t.TempDir(),
				Quiet:       true,
			}

			summary, err := runner.Run(ctx, job)
			if err != nil {
				t.Fatalf("Run: unexpected error: %v", err)
			}
			if summary.CancelReason != events.CancelReasonNone {
				t.Errorf("CancelReason = %q, want none", summary.CancelReason)
			}
			if summary.Launcher != "binary" {
				t.Errorf("Launcher = %q, want binary", summary.Launcher)
			}
			if summary.CycleRounds != nil {
				t.Errorf("CycleRounds = %v; want nil", summary.CycleRounds)
			}
			if summary.CycleStopReason != "" {
				t.Errorf("CycleStopReason = %q, want empty", summary.CycleStopReason)
			}
			if summary.ReviewStatus != "" {
				t.Errorf("ReviewStatus = %q, want empty", summary.ReviewStatus)
			}
			if summary.ReviewPath != "" {
				t.Errorf("ReviewPath = %q, want empty", summary.ReviewPath)
			}
			if summary.RetryAttempts != 0 {
				t.Errorf("RetryAttempts = %d, want 0", summary.RetryAttempts)
			}
			if summary.SlowPublishes != 0 {
				t.Errorf("SlowPublishes = %d, want 0", summary.SlowPublishes)
			}
			if summary.DroppedUpdates != 0 {
				t.Errorf("DroppedUpdates = %d, want 0", summary.DroppedUpdates)
			}
			if !summary.Metrics.IsZero() {
				t.Errorf("Metrics = %+v, want zero-value", summary.Metrics)
			}

			agent, err := specs.NewCatalog().AgentByID(tc.spec.ID)
			if err != nil {
				t.Fatalf("AgentByID(%s): %v", tc.spec.ID, err)
			}
			if !agent.InheritsEnv() {
				t.Errorf("agent %q does not inherit env intact (EnvPolicy is not zero-value)", tc.spec.ID)
			}
			if agent.EnvPolicy().IsZero() == false {
				t.Errorf("agent %q: EnvPolicy is not zero-value", tc.spec.ID)
			}
			if agent.RequiresHandshake() {
				t.Errorf("agent %q now requires governance handshake", tc.spec.ID)
			}
			if got, want := tc.spec.ResolveWindow("qualquer-modelo"), tc.spec.ContextWindow(); got != want {
				t.Errorf("agent %q: ResolveWindow() = %+v; want static window %+v", tc.spec.ID, got, want)
			}
		})
	}
}
