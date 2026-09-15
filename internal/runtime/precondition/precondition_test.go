package precondition_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/precondition"
	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/specs"
)

type fakeEnvironment struct {
	values map[string]string
}

func (f fakeEnvironment) Getenv(key string) string {
	return f.values[key]
}

func TestEvaluateNoKillSwitch(t *testing.T) {
	t.Parallel()

	t.Run("current when no kill switch set", func(t *testing.T) {
		t.Parallel()
		env := fakeEnvironment{values: map[string]string{}}
		state := precondition.EvaluateNoKillSwitch(env, []string{"FOO_PURE", "FOO_DISABLE"})
		if state != specs.PreconditionCurrent {
			t.Fatalf("state = %v; want current", state)
		}
	})

	t.Run("inert when any kill switch set", func(t *testing.T) {
		t.Parallel()
		env := fakeEnvironment{values: map[string]string{"FOO_DISABLE": "1"}}
		state := precondition.EvaluateNoKillSwitch(env, []string{"FOO_PURE", "FOO_DISABLE"})
		if state != specs.PreconditionInert {
			t.Fatalf("state = %v; want inert", state)
		}
	})
}

func TestEvaluateHandshakeIsAlwaysUnknown(t *testing.T) {
	t.Parallel()
	if got := precondition.EvaluateHandshake(); got != specs.PreconditionUnknown {
		t.Fatalf("EvaluateHandshake() = %v; want unknown", got)
	}
}

type fakeCopilotConfigReader struct {
	data []byte
	err  error
}

func (f fakeCopilotConfigReader) ReadConfig() ([]byte, error) {
	return f.data, f.err
}

func TestEvaluateCopilotTrustedFolder(t *testing.T) {
	t.Parallel()

	projectDir := t.TempDir()

	t.Run("current when project dir listed", func(t *testing.T) {
		t.Parallel()
		reader := fakeCopilotConfigReader{data: []byte(`{"trustedFolders":["` + projectDir + `"]}`)}
		state, err := precondition.EvaluateCopilotTrustedFolder(reader, projectDir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if state != specs.PreconditionCurrent {
			t.Fatalf("state = %v; want current", state)
		}
	})

	t.Run("current when project dir is subdirectory of trusted entry", func(t *testing.T) {
		t.Parallel()
		sub := filepath.Join(projectDir, "sub")
		if err := os.MkdirAll(sub, 0o755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
		reader := fakeCopilotConfigReader{data: []byte(`{"trustedFolders":["` + projectDir + `"]}`)}
		state, err := precondition.EvaluateCopilotTrustedFolder(reader, sub)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if state != specs.PreconditionCurrent {
			t.Fatalf("state = %v; want current", state)
		}
	})

	t.Run("inert when project dir absent from list", func(t *testing.T) {
		t.Parallel()
		reader := fakeCopilotConfigReader{data: []byte(`{"trustedFolders":["/somewhere/else"]}`)}
		state, err := precondition.EvaluateCopilotTrustedFolder(reader, projectDir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if state != specs.PreconditionInert {
			t.Fatalf("state = %v; want inert", state)
		}
	})

	t.Run("inert when config file does not exist", func(t *testing.T) {
		t.Parallel()
		reader := fakeCopilotConfigReader{err: os.ErrNotExist}
		state, err := precondition.EvaluateCopilotTrustedFolder(reader, projectDir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if state != specs.PreconditionInert {
			t.Fatalf("state = %v; want inert", state)
		}
	})

	t.Run("unknown on unexpected read error", func(t *testing.T) {
		t.Parallel()
		reader := fakeCopilotConfigReader{err: errors.New("permission denied")}
		state, err := precondition.EvaluateCopilotTrustedFolder(reader, projectDir)
		if err == nil {
			t.Fatal("expected error")
		}
		if state != specs.PreconditionUnknown {
			t.Fatalf("state = %v; want unknown", state)
		}
	})

	t.Run("strips line comments outside strings without regex", func(t *testing.T) {
		t.Parallel()
		content := "{\n" +
			"  // comment mentioning http://example.com should not corrupt values\n" +
			"  \"trustedFolders\": [\n" +
			"    \"" + projectDir + "\" // trailing comment\n" +
			"  ]\n" +
			"}\n"
		reader := fakeCopilotConfigReader{data: []byte(content)}
		state, err := precondition.EvaluateCopilotTrustedFolder(reader, projectDir)
		if err != nil {
			t.Fatalf("unexpected error parsing JSONC: %v", err)
		}
		if state != specs.PreconditionCurrent {
			t.Fatalf("state = %v; want current", state)
		}
	})

	t.Run("does not corrupt a value containing a URL with double slash", func(t *testing.T) {
		t.Parallel()
		content := `{"trustedFolders":["` + projectDir + `"],"note":"see https://example.com/path for details"}`
		reader := fakeCopilotConfigReader{data: []byte(content)}
		state, err := precondition.EvaluateCopilotTrustedFolder(reader, projectDir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if state != specs.PreconditionCurrent {
			t.Fatalf("state = %v; want current", state)
		}
	})
}

type fakeCodexRPCClient struct {
	result precondition.CodexHooksListResult
	err    error
}

func (f fakeCodexRPCClient) HooksList(_ context.Context) (precondition.CodexHooksListResult, error) {
	return f.result, f.err
}

var codexCanonicalEvents = []string{"PreToolUse", "PostToolUse", "Stop"}

func trustedProjectHooks(events ...string) []precondition.CodexHookStatus {
	hooks := make([]precondition.CodexHookStatus, 0, len(events))
	for _, event := range events {
		hooks = append(hooks, precondition.CodexHookStatus{EventName: event, Source: "project", TrustStatus: "trusted", CurrentHash: "sha256:" + event})
	}
	return hooks
}

func TestEvaluateCodexTrustedHash(t *testing.T) {
	t.Parallel()

	t.Run("unknown when no client provided", func(t *testing.T) {
		t.Parallel()
		report, err := precondition.EvaluateCodexTrustedHash(context.Background(), nil, time.Second, codexCanonicalEvents)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if report.State() != specs.PreconditionUnknown {
			t.Fatalf("state = %v; want unknown", report.State())
		}
	})

	t.Run("current only when every canonical point is trusted", func(t *testing.T) {
		t.Parallel()
		hooks := append([]precondition.CodexHookStatus{
			{EventName: "preToolUse", Source: "user", TrustStatus: "trusted"},
		}, trustedProjectHooks("preToolUse", "postToolUse", "stop")...)
		client := fakeCodexRPCClient{result: precondition.CodexHooksListResult{Hooks: hooks}}
		report, err := precondition.EvaluateCodexTrustedHash(context.Background(), client, time.Second, codexCanonicalEvents)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if report.State() != specs.PreconditionCurrent {
			t.Fatalf("state = %v; want current", report.State())
		}
		if len(report.Points) != len(codexCanonicalEvents) {
			t.Fatalf("points = %d; want one per canonical point", len(report.Points))
		}
	})

	t.Run("partial trust is inert and names the untrusted points", func(t *testing.T) {
		t.Parallel()
		client := fakeCodexRPCClient{result: precondition.CodexHooksListResult{
			Hooks: trustedProjectHooks("preToolUse"),
		}}
		report, err := precondition.EvaluateCodexTrustedHash(context.Background(), client, time.Second, codexCanonicalEvents)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if report.State() != specs.PreconditionInert {
			t.Fatalf("1 of 3 trusted must not be reported as satisfied; state = %v", report.State())
		}
		want := map[string]specs.PreconditionState{
			"PreToolUse":  specs.PreconditionCurrent,
			"PostToolUse": specs.PreconditionInert,
			"Stop":        specs.PreconditionInert,
		}
		for _, point := range report.Points {
			if want[point.EventName] != point.State {
				t.Fatalf("point %q state = %v; want %v", point.EventName, point.State, want[point.EventName])
			}
		}
	})

	t.Run("inert when a canonical project hook is not trusted", func(t *testing.T) {
		t.Parallel()
		hooks := append(trustedProjectHooks("preToolUse", "postToolUse"),
			precondition.CodexHookStatus{EventName: "stop", Source: "project", TrustStatus: "untrusted"})
		client := fakeCodexRPCClient{result: precondition.CodexHooksListResult{Hooks: hooks}}
		report, err := precondition.EvaluateCodexTrustedHash(context.Background(), client, time.Second, codexCanonicalEvents)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if report.State() != specs.PreconditionInert {
			t.Fatalf("state = %v; want inert", report.State())
		}
	})

	t.Run("user scope hooks never satisfy a project precondition", func(t *testing.T) {
		t.Parallel()
		client := fakeCodexRPCClient{result: precondition.CodexHooksListResult{Hooks: []precondition.CodexHookStatus{
			{EventName: "preToolUse", Source: "user", TrustStatus: "trusted"},
			{EventName: "postToolUse", Source: "user", TrustStatus: "trusted"},
			{EventName: "stop", Source: "user", TrustStatus: "trusted"},
		}}}
		report, err := precondition.EvaluateCodexTrustedHash(context.Background(), client, time.Second, codexCanonicalEvents)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if report.State() != specs.PreconditionInert {
			t.Fatalf("state = %v; want inert", report.State())
		}
	})

	t.Run("unknown on RPC failure", func(t *testing.T) {
		t.Parallel()
		client := fakeCodexRPCClient{err: errors.New("connection refused")}
		report, err := precondition.EvaluateCodexTrustedHash(context.Background(), client, time.Second, codexCanonicalEvents)
		if err == nil {
			t.Fatal("expected error")
		}
		if report.State() != specs.PreconditionUnknown {
			t.Fatalf("state = %v; want unknown", report.State())
		}
	})
}

func TestCodexAppServerClientCarriesTheInspectedWorkDir(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	client := precondition.NewCodexAppServerClient("", dir)
	if client.WorkDir != dir {
		t.Fatalf("WorkDir = %q; want %q — without it the RPC answers about the harness CWD, not the inspected project", client.WorkDir, dir)
	}
	if client.Binary != "codex" {
		t.Fatalf("Binary = %q; want codex", client.Binary)
	}
}

func TestEvaluateRoutesByKind(t *testing.T) {
	t.Parallel()
	catalog := specs.NewCatalog()

	noKillSwitch, err := catalog.NewEnforcementPrecondition(specs.PreconditionNoKillSwitch, "unset kill switch", false)
	if err != nil {
		t.Fatalf("NewEnforcementPrecondition: %v", err)
	}
	report := precondition.Evaluate(noKillSwitch, t.TempDir(), precondition.NewFileCopilotConfigReader("/nonexistent"), nil)
	if report.State != specs.PreconditionCurrent {
		t.Fatalf("no-kill-switch report.State = %v; want current", report.State)
	}

	handshake, err := catalog.NewEnforcementPrecondition(specs.PreconditionHandshake, "wait for handshake", true)
	if err != nil {
		t.Fatalf("NewEnforcementPrecondition: %v", err)
	}
	report = precondition.Evaluate(handshake, t.TempDir(), precondition.NewFileCopilotConfigReader("/nonexistent"), nil)
	if report.State != specs.PreconditionUnknown {
		t.Fatalf("handshake report.State = %v; want unknown", report.State)
	}

	trustedHash, err := catalog.NewEnforcementPrecondition(specs.PreconditionTrustedHash, "grant via TUI", true)
	if err != nil {
		t.Fatalf("NewEnforcementPrecondition: %v", err)
	}
	report = precondition.Evaluate(trustedHash, t.TempDir(), precondition.NewFileCopilotConfigReader("/nonexistent"), nil)
	if report.State != specs.PreconditionUnknown {
		t.Fatalf("trusted-hash without codexCheck report.State = %v; want unknown", report.State)
	}

	report = precondition.Evaluate(trustedHash, t.TempDir(), precondition.NewFileCopilotConfigReader("/nonexistent"), func() (precondition.CodexTrustReport, error) {
		return precondition.CodexTrustReport{Points: []precondition.CodexTrustPoint{
			{EventName: "PreToolUse", State: specs.PreconditionCurrent},
			{EventName: "PostToolUse", State: specs.PreconditionCurrent},
			{EventName: "Stop", State: specs.PreconditionCurrent},
		}}, nil
	})
	if report.State != specs.PreconditionCurrent {
		t.Fatalf("trusted-hash with codexCheck report.State = %v; want current", report.State)
	}
	if len(report.Points) != 3 {
		t.Fatalf("trusted-hash report must carry one point per canonical point; got %d", len(report.Points))
	}

	report = precondition.Evaluate(trustedHash, t.TempDir(), precondition.NewFileCopilotConfigReader("/nonexistent"), func() (precondition.CodexTrustReport, error) {
		return precondition.CodexTrustReport{Points: []precondition.CodexTrustPoint{
			{EventName: "PreToolUse", State: specs.PreconditionCurrent},
			{EventName: "PostToolUse", State: specs.PreconditionInert},
			{EventName: "Stop", State: specs.PreconditionInert},
		}}, nil
	})
	if report.State != specs.PreconditionInert {
		t.Fatalf("a precondition satisfied for one point and not another must never be reported as satisfied; got %v", report.State)
	}
}
