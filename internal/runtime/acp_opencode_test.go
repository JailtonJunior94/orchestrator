package runtime_test

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/JailtonJunior94/ai-spec-harness/internal/fs"
	airuntime "github.com/JailtonJunior94/ai-spec-harness/internal/runtime"
	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/acpfake"
	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/events"
	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/persistence"
	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/specs"
)

func proberOpenCodeBinary() *fakeProber {
	return &fakeProber{available: map[string]string{
		"opencode": "/usr/local/bin/opencode",
	}}
}

func TestACPIntegration_OpenCode_ToolCallsAndReport(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	script := acpfake.NewScript().
		AppendAgentMessage("opencode iniciou").
		AppendToolCall("tc_bash", "bash").
		AppendToolCallUpdate("tc_bash", "completed").
		AppendAgentMessage("finalizado").
		AppendSessionEnd()

	fakeFS := fs.NewFakeFileSystem()
	pfact := persistence.NewSessionPersistenceFactory(fakeFS)

	runner := buildRunnerWithSpec(t, ctx, specs.NewCatalog().OpenCode(), proberOpenCodeBinary(), script, pfact)

	evidenceDir := "/evidence/opencode-parity"
	job := airuntime.Job{
		Prompt:       "opencode parity test",
		WorkDir:      "/workdir",
		EvidenceDir:  evidenceDir,
		Quiet:        true,
		AccessMode:   specs.AccessModeRestricted,
		DisableHooks: true,
	}

	summary, err := runner.Run(ctx, job)
	if err != nil {
		t.Fatalf("Run (OpenCode): %v", err)
	}
	if summary.CancelReason != events.CancelReasonNone {
		t.Errorf("CancelReason = %q, want none", summary.CancelReason)
	}
	if len(summary.ToolCalls) < 1 {
		t.Fatalf("ToolCalls len = %d, want >= 1", len(summary.ToolCalls))
	}

	tcData, err := fakeFS.ReadFile(evidenceDir + "/tool_calls.md")
	if err != nil {
		t.Fatalf("tool_calls.md not found: %v", err)
	}
	if !strings.Contains(string(tcData), "bash") {
		t.Errorf("tool_calls.md = %q, want to contain 'bash'", tcData)
	}

	eventsData, err := fakeFS.ReadFile(evidenceDir + "/events.jsonl")
	if err != nil {
		t.Fatalf("events.jsonl not found: %v", err)
	}
	lines := strings.Split(strings.TrimSpace(string(eventsData)), "\n")
	if len(lines) < 3 {
		t.Fatalf("events.jsonl lines = %d, want >= 3", len(lines))
	}

	var first map[string]json.RawMessage
	if err := json.Unmarshal([]byte(lines[0]), &first); err != nil {
		t.Fatalf("unmarshal first line: %v", err)
	}
	var kind string
	if err := json.Unmarshal(first["kind"], &kind); err != nil {
		t.Fatalf("unmarshal first kind: %v", err)
	}
	if kind != string(events.KindRuntimeInit) {
		t.Fatalf("first kind = %q, want runtime_init", kind)
	}

	var last map[string]json.RawMessage
	if err := json.Unmarshal([]byte(lines[len(lines)-1]), &last); err != nil {
		t.Fatalf("unmarshal last line: %v", err)
	}
	if err := json.Unmarshal(last["kind"], &kind); err != nil {
		t.Fatalf("unmarshal last kind: %v", err)
	}
	if kind != string(events.KindSessionEnd) {
		t.Fatalf("last kind = %q, want session_end", kind)
	}

	reportData, err := fakeFS.ReadFile(evidenceDir + "/execution_report.md")
	if err != nil {
		t.Fatalf("execution_report.md not found: %v", err)
	}
	reportContent := string(reportData)
	for _, want := range []string{"events_count:", "cancel_reason: none", "launcher: binary", "unknown_events_count: 0"} {
		if !strings.Contains(reportContent, want) {
			t.Errorf("execution_report.md missing %q; content=%q", want, reportContent)
		}
	}
}

func TestACPIntegration_OpenCode_ArgvIncludesSubcommandAndCwd(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	script := acpfake.NewScript().
		AppendAgentMessage("hello from opencode").
		AppendSessionEnd()

	pfact, persist := newFakePersistenceFactory()
	runner := buildRunnerWithSpec(t, ctx, specs.NewCatalog().OpenCode(), proberOpenCodeBinary(), script, pfact)

	job := airuntime.Job{
		Prompt:       "opencode argv test",
		WorkDir:      "/repo",
		EvidenceDir:  t.TempDir(),
		Quiet:        true,
		AccessMode:   specs.AccessModeRestricted,
		DisableHooks: true,
	}

	if _, err := runner.Run(ctx, job); err != nil {
		t.Fatalf("Run (OpenCode argv): %v", err)
	}

	args := findRuntimeInitArgs(t, persist)
	joined := strings.Join(args, " ")
	if !strings.Contains(joined, "--cwd /repo") {
		t.Errorf("argv = %v; want to contain '--cwd /repo'", args)
	}
	for _, flag := range []string{"--model", "--acp"} {
		if strings.Contains(joined, flag) {
			t.Errorf("argv = %v; must never presume unconfirmed flag %q (V-02)", args, flag)
		}
	}
}

func TestACPIntegration_OpenCode_ActivityWatchdog(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	script := acpfake.NewScript().
		AppendAgentMessage("opencode watchdog start").
		AppendAgentMessageWithDelay("never arrives", 500*time.Millisecond).
		AppendSessionEnd()

	pfact, persist := newFakePersistenceFactory()

	timeout, err := events.NewActivityTimeout(50 * time.Millisecond)
	if err != nil {
		t.Fatal(err)
	}

	runner := buildRunnerWithSpec(t, ctx, specs.NewCatalog().OpenCode(), proberOpenCodeBinary(), script, pfact)

	job := airuntime.Job{
		Prompt:        "opencode watchdog",
		WorkDir:       workDirWithAgentsMD(t),
		EvidenceDir:   t.TempDir(),
		RuntimeConfig: airuntime.RuntimeConfig{Timeout: timeout},
		Quiet:         true,
		AccessMode:    specs.AccessModeRestricted,
	}

	summary, runErr := runner.Run(ctx, job)

	if runErr != nil {
		if summary.CancelReason != events.CancelReasonActivityTimeout {
			t.Errorf("CancelReason = %q, want activity_timeout when error present", summary.CancelReason)
		}
	} else {
		t.Logf("session ended before timeout; CancelReason=%q", summary.CancelReason)
	}

	if persist.summary == nil {
		t.Error("EnrichReport was not called — execution_report was not enriched")
	}
}
