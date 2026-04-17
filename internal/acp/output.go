package acp

import "time"

// ACPOutput contains the accumulated result of an ACP agent execution.
type ACPOutput struct {
	// Content is the accumulated text from all AgentMessageChunk updates.
	Content string

	// Thoughts is the accumulated text from all AgentThoughtChunk updates.
	Thoughts string

	// ToolCalls collects all tool call events received during execution.
	ToolCalls []ToolCallInfo

	// SessionID is the ACP session identifier, persisted for future resume.
	SessionID string

	// StopReason describes why the agent stopped (e.g. "end_turn", "error").
	StopReason string

	// Duration is the total wall-clock time for the execution.
	Duration time.Duration
}
