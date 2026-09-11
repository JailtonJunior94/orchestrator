package specs

var provenDispatchCells = map[string]bool{
	"claude:pre-tool":      true,
	"claude:post-tool":     true,
	"claude:session-end":   true,
	"codex:pre-tool":       true,
	"codex:post-tool":      true,
	"codex:session-end":    true,
	"copilot:pre-tool":     true,
	"copilot:post-tool":    true,
	"copilot:session-end":  true,
	"opencode:pre-tool":    true,
	"opencode:post-tool":   true,
	"opencode:session-end": true,
}

func DispatchProven(agentID string, point CanonicalPoint) bool {
	return provenDispatchCells[agentID+":"+point.String()]
}
