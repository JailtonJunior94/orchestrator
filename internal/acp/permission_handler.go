package acp

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	acpsdk "github.com/coder/acp-go-sdk"
	"github.com/jailtonjunior/orchestrator/internal/hitl"
)

// PermissionPolicy controls the default decision path for ACP permission
// requests in non-interactive mode, optionally scoped by workflow/provider.
type PermissionPolicy struct {
	DefaultDecision   hitl.PermissionDecision
	ProviderDecisions map[string]hitl.PermissionDecision
	WorkflowDecisions map[string]hitl.PermissionDecision
}

// PermissionMetadata identifies the workflow context bound to one ACP
// execution. It is copied into the PermissionHandler so policy evaluation can
// vary by provider/workflow.
type PermissionMetadata struct {
	Provider string
	Workflow string
}

// PermissionHandler routes RequestPermission callbacks from the agent to a
// hitl.Prompter. In non-interactive mode it auto-approves the first option.
type PermissionHandler struct {
	prompter    hitl.Prompter
	interactive bool
	logger      *slog.Logger
	policy      PermissionPolicy
	metadata    PermissionMetadata
}

// NewPermissionHandler creates a PermissionHandler backed by the given prompter.
// When interactive is false the handler auto-approves without calling the prompter.
func NewPermissionHandler(prompter hitl.Prompter, interactive bool, policy PermissionPolicy, logger *slog.Logger) *PermissionHandler {
	if logger == nil {
		logger = slog.Default()
	}
	if policy.DefaultDecision == "" {
		policy.DefaultDecision = hitl.PermissionAllow
	}
	return &PermissionHandler{
		prompter:    prompter,
		interactive: interactive,
		logger:      logger,
		policy:      policy,
	}
}

// WithExecution returns a shallow copy of the handler bound to one workflow
// execution, applying any execution-scoped policy overrides on top of the base
// handler policy.
func (h *PermissionHandler) WithExecution(metadata PermissionMetadata, override PermissionPolicy) *PermissionHandler {
	if h == nil {
		return nil
	}
	cp := *h
	cp.metadata = metadata
	cp.policy = mergePermissionPolicies(cp.policy, override)
	return &cp
}

// RequestPermission is called by the agent before a sensitive operation.
func (h *PermissionHandler) RequestPermission(ctx context.Context, params acpsdk.RequestPermissionRequest) (acpsdk.RequestPermissionResponse, error) {
	toolCallID := string(params.ToolCall.ToolCallId)
	title := ""
	if params.ToolCall.Title != nil {
		title = *params.ToolCall.Title
	}
	toolKind := ""
	if params.ToolCall.Kind != nil {
		toolKind = string(*params.ToolCall.Kind)
	}
	details := buildPermissionDetails(params.ToolCall)
	request := hitl.PermissionRequest{
		Provider:   h.metadata.Provider,
		Workflow:   h.metadata.Workflow,
		ToolCallID: toolCallID,
		Title:      title,
		ToolKind:   toolKind,
		Details:    details,
		Options:    convertPermissionOptions(params.Options),
	}

	h.logger.InfoContext(ctx, "acp_permission_requested",
		slog.String("tool_call_id", toolCallID),
		slog.String("provider", h.metadata.Provider),
		slog.String("workflow", h.metadata.Workflow),
	)

	if !h.interactive {
		decision := h.policyDecision()
		outcome := h.buildOutcome(params.Options, decision)
		h.logger.InfoContext(ctx, "acp_permission_outcome",
			slog.String("outcome", describePermissionOutcome(outcome)),
		)
		return acpsdk.RequestPermissionResponse{
			Outcome: outcome,
		}, nil
	}

	decision, err := h.promptDecision(ctx, request)
	if err != nil {
		return acpsdk.RequestPermissionResponse{}, fmt.Errorf("acp permission prompt: %w", err)
	}

	outcome := h.buildOutcome(params.Options, decision)

	h.logger.InfoContext(ctx, "acp_permission_outcome",
		slog.String("outcome", describePermissionOutcome(outcome)),
	)
	return acpsdk.RequestPermissionResponse{
		Outcome: outcome,
	}, nil
}

func (h *PermissionHandler) promptDecision(ctx context.Context, request hitl.PermissionRequest) (hitl.PermissionDecision, error) {
	if permissionPrompter, ok := h.prompter.(hitl.PermissionPrompter); ok {
		result, err := permissionPrompter.PromptPermission(ctx, request)
		if err != nil {
			return "", err
		}
		return result.Decision, nil
	}

	description := fmt.Sprintf("Agent requests permission for tool %q (id: %s)\n%s", request.Title, request.ToolCallID, request.Details)
	result, err := h.prompter.Prompt(ctx, description)
	if err != nil {
		return "", err
	}
	if result.Action == hitl.ActionApprove {
		return hitl.PermissionAllow, nil
	}
	return hitl.PermissionDeny, nil
}

func (h *PermissionHandler) policyDecision() hitl.PermissionDecision {
	if decision, ok := h.policy.WorkflowDecisions[h.metadata.Workflow]; ok {
		return decision
	}
	if decision, ok := h.policy.ProviderDecisions[h.metadata.Provider]; ok {
		return decision
	}
	return h.policy.DefaultDecision
}

func (h *PermissionHandler) buildOutcome(options []acpsdk.PermissionOption, decision hitl.PermissionDecision) acpsdk.RequestPermissionOutcome {
	switch decision {
	case hitl.PermissionDeny:
		return acpsdk.NewRequestPermissionOutcomeSelected(h.firstDenyOption(options))
	case hitl.PermissionCancel:
		return acpsdk.NewRequestPermissionOutcomeCancelled()
	default:
		return acpsdk.NewRequestPermissionOutcomeSelected(h.firstAllowOption(options))
	}
}

func describePermissionOutcome(outcome acpsdk.RequestPermissionOutcome) string {
	switch {
	case outcome.Cancelled != nil:
		return outcome.Cancelled.Outcome
	case outcome.Selected != nil:
		return string(outcome.Selected.OptionId)
	default:
		return ""
	}
}

// firstAllowOption returns the ID of the first allow-typed option, or a synthetic "allow" ID.
func (h *PermissionHandler) firstAllowOption(options []acpsdk.PermissionOption) acpsdk.PermissionOptionId {
	for _, opt := range options {
		if opt.Kind == acpsdk.PermissionOptionKindAllowOnce || opt.Kind == acpsdk.PermissionOptionKindAllowAlways {
			return opt.OptionId
		}
	}
	if len(options) > 0 {
		return options[0].OptionId
	}
	return "allow"
}

// firstDenyOption returns the ID of the first deny-typed option, or a synthetic "deny" ID.
func (h *PermissionHandler) firstDenyOption(options []acpsdk.PermissionOption) acpsdk.PermissionOptionId {
	for _, opt := range options {
		if opt.Kind == acpsdk.PermissionOptionKindRejectOnce || opt.Kind == acpsdk.PermissionOptionKindRejectAlways {
			return opt.OptionId
		}
	}
	return "deny"
}

func convertPermissionOptions(options []acpsdk.PermissionOption) []hitl.PermissionOption {
	converted := make([]hitl.PermissionOption, 0, len(options))
	for _, option := range options {
		converted = append(converted, hitl.PermissionOption{
			ID:   string(option.OptionId),
			Name: option.Name,
			Kind: string(option.Kind),
		})
	}
	return converted
}

func buildPermissionDetails(update acpsdk.ToolCallUpdate) string {
	payload := map[string]any{}
	if update.Kind != nil {
		payload["tool_kind"] = string(*update.Kind)
	}
	if len(update.Locations) > 0 {
		paths := make([]string, 0, len(update.Locations))
		for _, location := range update.Locations {
			paths = append(paths, location.Path)
		}
		payload["locations"] = paths
	}
	if update.RawInput != nil {
		payload["raw_input"] = update.RawInput
	}
	if update.RawOutput != nil {
		payload["raw_output"] = update.RawOutput
	}
	if len(payload) == 0 {
		return "No additional permission details were provided by the agent."
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return "Unable to render permission details."
	}
	return string(data)
}

func mergePermissionPolicies(base PermissionPolicy, override PermissionPolicy) PermissionPolicy {
	merged := base
	if override.DefaultDecision != "" {
		merged.DefaultDecision = override.DefaultDecision
	}
	if len(override.ProviderDecisions) > 0 {
		if merged.ProviderDecisions == nil {
			merged.ProviderDecisions = map[string]hitl.PermissionDecision{}
		}
		for provider, decision := range override.ProviderDecisions {
			merged.ProviderDecisions[provider] = decision
		}
	}
	if len(override.WorkflowDecisions) > 0 {
		if merged.WorkflowDecisions == nil {
			merged.WorkflowDecisions = map[string]hitl.PermissionDecision{}
		}
		for workflow, decision := range override.WorkflowDecisions {
			merged.WorkflowDecisions[workflow] = decision
		}
	}
	return merged
}
