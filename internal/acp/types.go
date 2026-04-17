package acp

// UpdateKind discriminates the category of a streaming update from the agent.
type UpdateKind string

const (
	// UpdateMessage represents a text output chunk from the agent.
	UpdateMessage UpdateKind = "message"

	// UpdateThought represents a chain-of-thought reasoning chunk.
	UpdateThought UpdateKind = "thought"

	// UpdateToolCall represents the initiation of a tool use event.
	UpdateToolCall UpdateKind = "tool_call"

	// UpdateToolUpdate represents an incremental status update for an in-progress tool call.
	UpdateToolUpdate UpdateKind = "tool_update"

	// UpdatePermission represents a permission request from the agent.
	// V1 auto-approves; the TUI displays it as a notification.
	UpdatePermission UpdateKind = "permission"
)

// ToolCallInfo carries metadata for a single tool call emitted by the agent.
type ToolCallInfo struct {
	ID        string
	Title     string
	Kind      string
	Status    string
	Input     string
	Output    string
	Locations []string
}

// TypedUpdate is a discriminated union representing one streaming event from the agent.
type TypedUpdate struct {
	// Kind identifies which variant this update represents.
	Kind UpdateKind

	// Text holds the content for MessageChunk and ThoughtChunk updates.
	Text string

	// ToolCall holds metadata for ToolCall and ToolCallUpdate updates.
	ToolCall *ToolCallInfo
}
