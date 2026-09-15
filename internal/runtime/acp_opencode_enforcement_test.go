package runtime_test

import (
	"context"
	"strings"
	"testing"
	"time"

	airuntime "github.com/JailtonJunior94/ai-spec-harness/internal/runtime"
	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/acpfake"
	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/events"
	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/specs"
)

func TestACPIntegration_OpenCode_GovernanceBlockSurvivesConsecutiveExceptions(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	script := acpfake.NewScript().
		AppendAgentMessage("opencode iniciou").
		AppendToolCall("tc1", "edit").
		AppendToolCallUpdate("tc1", "failed").
		AppendAgentMessage("GOVERNANCE BLOCKED for tool \"edit\": retry after fixing the reported issue").
		AppendToolCall("tc2", "edit").
		AppendToolCallUpdate("tc2", "failed").
		AppendAgentMessage("GOVERNANCE BLOCKED for tool \"edit\": retry after fixing the reported issue").
		AppendToolCall("tc3", "edit").
		AppendToolCallUpdate("tc3", "failed").
		AppendAgentMessage("GOVERNANCE BLOCKED for tool \"edit\": retry after fixing the reported issue").
		AppendAgentMessage("finalizado").
		AppendSessionEnd()

	pfact, persist := newFakePersistenceFactory()
	runner := buildRunnerWithSpec(t, ctx, specs.NewCatalog().OpenCode(), proberOpenCodeBinary(), script, pfact)

	job := airuntime.Job{
		Prompt:       "opencode governance block test",
		WorkDir:      "/workdir",
		EvidenceDir:  t.TempDir(),
		Quiet:        true,
		AccessMode:   specs.AccessModeRestricted,
		DisableHooks: true,
	}

	summary, err := runner.Run(ctx, job)
	if err != nil {
		t.Fatalf("Run (OpenCode blocked): %v", err)
	}
	if summary.CancelReason != events.CancelReasonNone {
		t.Fatalf("CancelReason = %q, want none — session must survive repeated blocks", summary.CancelReason)
	}

	failedCount := 0
	blockedMessageSeen := false
	for _, evt := range persist.events {
		raw := string(evt.Raw())
		if strings.Contains(raw, `"status":"failed"`) {
			failedCount++
		}
		if strings.Contains(raw, "GOVERNANCE BLOCKED") {
			blockedMessageSeen = true
		}
	}
	if failedCount < 3 {
		t.Errorf("failed tool call updates observed = %d, want >= 3 (robustness to consecutive exceptions)", failedCount)
	}
	if !blockedMessageSeen {
		t.Error("corrective exception text was not observed anywhere in the event stream")
	}
	if len(summary.ToolCalls) < 3 {
		t.Fatalf("ToolCalls len = %d, want >= 3", len(summary.ToolCalls))
	}
}

func TestACPIntegration_OpenCode_GovernanceBlockInsideSubagentTask(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	script := acpfake.NewScript().
		AppendAgentMessage("delegando para subagente").
		AppendToolCall("tc_task", "task").
		AppendToolCall("tc_nested_edit", "edit").
		AppendToolCallUpdate("tc_nested_edit", "failed").
		AppendAgentMessage("GOVERNANCE BLOCKED for tool \"edit\": retry after fixing the reported issue").
		AppendToolCallUpdate("tc_task", "completed").
		AppendAgentMessage("finalizado").
		AppendSessionEnd()

	pfact, persist := newFakePersistenceFactory()
	runner := buildRunnerWithSpec(t, ctx, specs.NewCatalog().OpenCode(), proberOpenCodeBinary(), script, pfact)

	job := airuntime.Job{
		Prompt:       "opencode governance subagent block test",
		WorkDir:      "/workdir",
		EvidenceDir:  t.TempDir(),
		Quiet:        true,
		AccessMode:   specs.AccessModeRestricted,
		DisableHooks: true,
	}

	summary, err := runner.Run(ctx, job)
	if err != nil {
		t.Fatalf("Run (OpenCode subagent block): %v", err)
	}
	if summary.CancelReason != events.CancelReasonNone {
		t.Fatalf("CancelReason = %q, want none", summary.CancelReason)
	}

	nestedBlocked := false
	for _, evt := range persist.events {
		raw := string(evt.Raw())
		if strings.Contains(raw, "tc_nested_edit") && strings.Contains(raw, `"status":"failed"`) {
			nestedBlocked = true
		}
	}
	if !nestedBlocked {
		t.Error("tool call executed inside the subagent (task) delegation was not observed as blocked")
	}
}
