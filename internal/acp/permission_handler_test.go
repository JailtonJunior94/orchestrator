package acp_test

import (
	"context"
	"testing"

	acpsdk "github.com/coder/acp-go-sdk"
	"github.com/jailtonjunior/orchestrator/internal/acp"
	"github.com/jailtonjunior/orchestrator/internal/hitl"
)

func TestPermissionHandler_AutoApprove_NonInteractive(t *testing.T) {
	t.Parallel()

	handler := acp.NewPermissionHandler(nil, false, acp.PermissionPolicy{}, nil)
	params := acpsdk.RequestPermissionRequest{
		Options: []acpsdk.PermissionOption{
			{OptionId: "opt-allow", Kind: acpsdk.PermissionOptionKindAllowOnce, Name: "Allow once"},
			{OptionId: "opt-deny", Kind: acpsdk.PermissionOptionKindRejectOnce, Name: "Deny"},
		},
		ToolCall: acpsdk.ToolCallUpdate{ToolCallId: "tc-1"},
	}

	resp, err := handler.RequestPermission(context.Background(), params)
	if err != nil {
		t.Fatal(err)
	}
	if resp.Outcome.Selected == nil {
		t.Fatal("expected Selected outcome, got nil")
	}
	if resp.Outcome.Selected.OptionId != "opt-allow" {
		t.Fatalf("selected = %q, want %q", resp.Outcome.Selected.OptionId, "opt-allow")
	}
}

func TestPermissionHandler_Interactive_Approve(t *testing.T) {
	t.Parallel()

	prompter := hitl.NewFakePrompter(hitl.PromptResult{Action: hitl.ActionApprove})
	handler := acp.NewPermissionHandler(prompter, true, acp.PermissionPolicy{}, nil)

	params := acpsdk.RequestPermissionRequest{
		Options: []acpsdk.PermissionOption{
			{OptionId: "opt-allow", Kind: acpsdk.PermissionOptionKindAllowOnce, Name: "Allow once"},
		},
		ToolCall: acpsdk.ToolCallUpdate{ToolCallId: "tc-2"},
	}

	resp, err := handler.RequestPermission(context.Background(), params)
	if err != nil {
		t.Fatal(err)
	}
	if resp.Outcome.Selected == nil {
		t.Fatal("expected Selected outcome")
	}
	if resp.Outcome.Selected.OptionId != "opt-allow" {
		t.Fatalf("selected = %q, want %q", resp.Outcome.Selected.OptionId, "opt-allow")
	}
}

func TestPermissionHandler_Interactive_Deny(t *testing.T) {
	t.Parallel()

	prompter := hitl.NewFakePrompter(hitl.PromptResult{Action: hitl.ActionExit})
	handler := acp.NewPermissionHandler(prompter, true, acp.PermissionPolicy{}, nil)

	params := acpsdk.RequestPermissionRequest{
		Options: []acpsdk.PermissionOption{
			{OptionId: "opt-allow", Kind: acpsdk.PermissionOptionKindAllowOnce, Name: "Allow"},
			{OptionId: "opt-deny", Kind: acpsdk.PermissionOptionKindRejectOnce, Name: "Deny"},
		},
		ToolCall: acpsdk.ToolCallUpdate{ToolCallId: "tc-3"},
	}

	resp, err := handler.RequestPermission(context.Background(), params)
	if err != nil {
		t.Fatal(err)
	}
	if resp.Outcome.Selected == nil {
		t.Fatal("expected Selected outcome")
	}
	if resp.Outcome.Selected.OptionId != "opt-deny" {
		t.Fatalf("selected = %q, want %q", resp.Outcome.Selected.OptionId, "opt-deny")
	}
}

func TestPermissionHandler_NonInteractive_UsesWorkflowPolicy(t *testing.T) {
	t.Parallel()

	handler := acp.NewPermissionHandler(nil, false, acp.PermissionPolicy{
		DefaultDecision: hitl.PermissionAllow,
		WorkflowDecisions: map[string]hitl.PermissionDecision{
			"security-review": hitl.PermissionDeny,
		},
	}, nil).WithExecution(acp.PermissionMetadata{
		Provider: "claude",
		Workflow: "security-review",
	}, acp.PermissionPolicy{})

	resp, err := handler.RequestPermission(context.Background(), acpsdk.RequestPermissionRequest{
		Options: []acpsdk.PermissionOption{
			{OptionId: "opt-allow", Kind: acpsdk.PermissionOptionKindAllowOnce, Name: "Allow"},
			{OptionId: "opt-deny", Kind: acpsdk.PermissionOptionKindRejectOnce, Name: "Deny"},
		},
		ToolCall: acpsdk.ToolCallUpdate{ToolCallId: "tc-4"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Outcome.Selected == nil {
		t.Fatal("expected Selected outcome")
	}
	if resp.Outcome.Selected.OptionId != "opt-deny" {
		t.Fatalf("selected = %q, want %q", resp.Outcome.Selected.OptionId, "opt-deny")
	}
}

func TestPermissionHandler_Interactive_UsesExplicitPermissionPrompter(t *testing.T) {
	t.Parallel()

	prompter := hitl.NewFakePermissionPrompter(hitl.PermissionResult{Decision: hitl.PermissionDeny})
	handler := acp.NewPermissionHandler(prompter, true, acp.PermissionPolicy{}, nil)

	resp, err := handler.RequestPermission(context.Background(), acpsdk.RequestPermissionRequest{
		Options: []acpsdk.PermissionOption{
			{OptionId: "opt-allow", Kind: acpsdk.PermissionOptionKindAllowOnce, Name: "Allow"},
			{OptionId: "opt-deny", Kind: acpsdk.PermissionOptionKindRejectOnce, Name: "Deny"},
		},
		ToolCall: acpsdk.ToolCallUpdate{ToolCallId: "tc-5"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Outcome.Selected == nil {
		t.Fatal("expected Selected outcome")
	}
	if resp.Outcome.Selected.OptionId != "opt-deny" {
		t.Fatalf("selected = %q, want %q", resp.Outcome.Selected.OptionId, "opt-deny")
	}
}

func TestPermissionHandler_Interactive_CancelUsesCancelledOutcome(t *testing.T) {
	t.Parallel()

	prompter := hitl.NewFakePermissionPrompter(hitl.PermissionResult{Decision: hitl.PermissionCancel})
	handler := acp.NewPermissionHandler(prompter, true, acp.PermissionPolicy{}, nil)

	resp, err := handler.RequestPermission(context.Background(), acpsdk.RequestPermissionRequest{
		Options: []acpsdk.PermissionOption{
			{OptionId: "opt-allow", Kind: acpsdk.PermissionOptionKindAllowOnce, Name: "Allow"},
			{OptionId: "opt-deny", Kind: acpsdk.PermissionOptionKindRejectOnce, Name: "Deny"},
		},
		ToolCall: acpsdk.ToolCallUpdate{ToolCallId: "tc-6"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Outcome.Cancelled == nil {
		t.Fatal("expected Cancelled outcome")
	}
	if resp.Outcome.Selected != nil {
		t.Fatal("expected Selected outcome to be nil on cancel")
	}
}

func TestPermissionHandler_NonInteractive_CancelUsesCancelledOutcome(t *testing.T) {
	t.Parallel()

	handler := acp.NewPermissionHandler(nil, false, acp.PermissionPolicy{
		DefaultDecision: hitl.PermissionCancel,
	}, nil)

	resp, err := handler.RequestPermission(context.Background(), acpsdk.RequestPermissionRequest{
		Options: []acpsdk.PermissionOption{
			{OptionId: "opt-allow", Kind: acpsdk.PermissionOptionKindAllowOnce, Name: "Allow"},
			{OptionId: "opt-deny", Kind: acpsdk.PermissionOptionKindRejectOnce, Name: "Deny"},
		},
		ToolCall: acpsdk.ToolCallUpdate{ToolCallId: "tc-7"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Outcome.Cancelled == nil {
		t.Fatal("expected Cancelled outcome")
	}
}
