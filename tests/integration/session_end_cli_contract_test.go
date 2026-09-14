//go:build integration

package integration

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/JailtonJunior94/ai-spec-harness/internal/skills"
)

type sessionEndRefusalSignal string

const (
	refusalExitCodeStderr sessionEndRefusalSignal = "exit 2 with the reason on stderr"
	refusalStdoutJSON     sessionEndRefusalSignal = "stdout {\"decision\":\"block\",\"reason\":...}"
	refusalThrow          sessionEndRefusalSignal = "exception thrown by the plugin handler"
)

const jsonDecisionModeEnv = "AISPEC_HOOK_DECISION_OUTPUT=json"

type sessionEndCliContract struct {
	BlockingEvent string
	RefusalSignal sessionEndRefusalSignal
	DeclaredEvent string
	Evidence      string
}

var sessionEndCliContracts2026 = map[skills.Tool]sessionEndCliContract{
	skills.ToolClaude: {
		BlockingEvent: "Stop",
		RefusalSignal: refusalExitCodeStderr,
		DeclaredEvent: "Stop",
		Evidence:      "Claude Code hook contract: exit 2 blocks and feeds stderr back to the model; exit 1 is a non-blocking hook error",
	},
	skills.ToolCodex: {
		BlockingEvent: "Stop",
		RefusalSignal: refusalExitCodeStderr,
		DeclaredEvent: "Stop",
		Evidence:      "codex-cli 0.154.0 embedded schema: stop.command.output exists while session-end.command.output does not, and the exit-code-2 blocking literals enumerate PreToolUse, PostToolUse, PermissionRequest, UserPromptSubmit, SubagentStop and Stop, never SessionEnd",
	},
	skills.ToolCopilot: {
		BlockingEvent: "agentStop",
		RefusalSignal: refusalStdoutJSON,
		DeclaredEvent: "agentStop",
		Evidence:      "GitHub Copilot CLI 1.0.83 copilot-sdk/types.d.ts AgentStopHookOutput accepts only decision block plus reason, enqueued as a follow-up user message; a measured exit-code sweep showed exit 0 and exit 2 keep stdout parsed while any other non-zero exit discards it, so refusal must travel on stdout",
	},
	skills.ToolOpenCode: {
		BlockingEvent: "session.idle",
		RefusalSignal: refusalThrow,
		DeclaredEvent: "session.idle",
		Evidence:      "the governance plugin rejects an idle session by throwing from the session.idle handler; the plugin API exposes no exit code and no stdout channel",
	},
}

func TestSessionEndCliBlockingContractIsMeasuredNotAssumed(t *testing.T) {
	t.Parallel()

	if len(sessionEndCliContracts2026) != 4 {
		t.Fatalf("the closing-gate contract must cover all 4 mandatory agents; got %d", len(sessionEndCliContracts2026))
	}

	for tool, contract := range sessionEndCliContracts2026 {
		t.Run(string(tool), func(t *testing.T) {
			if contract.BlockingEvent == "" || contract.RefusalSignal == "" {
				t.Fatalf("tool=%s: the blocking contract must name an event and a refusal signal", tool)
			}
			if contract.Evidence == "" {
				t.Fatalf("tool=%s: every contract entry must carry the measurement that produced it, never an assumption", tool)
			}
			if contract.DeclaredEvent != contract.BlockingEvent {
				t.Fatalf("tool=%s: the gate is declared on %q but this CLI only blocks on %q, so the gate is inert. Evidence: %s",
					tool, contract.DeclaredEvent, contract.BlockingEvent, contract.Evidence)
			}
		})
	}
}

func TestSessionEndRefusalSignalsAreNotUniformAcrossAgents(t *testing.T) {
	t.Parallel()

	signals := map[sessionEndRefusalSignal]int{}
	for _, contract := range sessionEndCliContracts2026 {
		signals[contract.RefusalSignal]++
	}
	if len(signals) < 3 {
		t.Fatalf("the four CLIs do not share a single refusal mechanism; a matrix collapsing them to %d signal(s) asserts the wrong contract: %v", len(signals), signals)
	}
	if signals[refusalStdoutJSON] == 0 {
		t.Fatalf("no agent declares the stdout-JSON refusal signal, yet Copilot blocks agentStop only that way")
	}
}

func TestClaudeSessionEndGateIsWiredToItsBlockingEvent(t *testing.T) {
	t.Parallel()
	projectDir := installMandatoryAgentsForDispatch(t)

	raw, err := os.ReadFile(filepath.Join(projectDir, ".claude", "settings.local.json"))
	if err != nil {
		t.Fatalf("read claude settings: %v", err)
	}
	var doc struct {
		Hooks map[string][]struct {
			Hooks []struct {
				Command string `json:"command"`
			} `json:"hooks"`
		} `json:"hooks"`
	}
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatalf("decode claude settings: %v", err)
	}

	event := sessionEndCliContracts2026[skills.ToolClaude].BlockingEvent
	entries, ok := doc.Hooks[event]
	if !ok || len(entries) == 0 {
		t.Fatalf("the Claude session-end gate must be declared on %q, the only event proven to block", event)
	}
	command := ""
	for _, entry := range entries {
		for _, hook := range entry.Hooks {
			if strings.Contains(hook.Command, "validate-session-end.sh") {
				command = hook.Command
			}
		}
	}
	if command == "" {
		t.Fatalf("the canonical session-end validator must be wired to %q; got %+v", event, entries)
	}
	if strings.Contains(command, jsonDecisionModeEnv) {
		t.Fatalf("Claude blocks by exit code and reads stderr; selecting the stdout-JSON mode would emit a decision payload Claude never parses: %q", command)
	}
}

func codexHookSubtable(content, event string) (string, bool) {
	marker := "[[hooks." + event + ".hooks]]"
	start := strings.Index(content, marker)
	if start < 0 {
		return "", false
	}
	rest := content[start+len(marker):]
	if next := strings.Index(rest, "[[hooks."); next >= 0 {
		rest = rest[:next]
	}
	return rest, true
}

func TestCodexSessionEndGateIsWiredToItsBlockingEvent(t *testing.T) {
	t.Parallel()
	projectDir := installMandatoryAgentsForDispatch(t)

	raw, err := os.ReadFile(filepath.Join(projectDir, ".codex", "config.toml"))
	if err != nil {
		t.Fatalf("read codex config: %v", err)
	}
	content := string(raw)

	event := sessionEndCliContracts2026[skills.ToolCodex].BlockingEvent
	if !strings.Contains(content, "[[hooks."+event+"]]") {
		t.Fatalf("the Codex session-end gate must be declared under [[hooks.%s]], the only closing event whose embedded schema carries a command output and an exit-code-2 blocking path; got:\n%s", event, content)
	}
	if strings.Contains(content, "[[hooks.SessionEnd]]") {
		t.Fatalf("SessionEnd has no command output schema in codex-cli 0.154.0 and never honours exit 2; declaring the gate there leaves it inert:\n%s", content)
	}

	body, ok := codexHookSubtable(content, event)
	if !ok {
		t.Fatalf("the [[hooks.%s.hooks]] sub-table must declare the command; got:\n%s", event, content)
	}
	if !strings.Contains(body, "validate-session-end.sh") {
		t.Fatalf("the canonical session-end validator must execute under [[hooks.%s.hooks]]; got:\n%s", event, body)
	}
	if strings.Contains(body, jsonDecisionModeEnv) {
		t.Fatalf("Codex reads the exit code and stderr on Stop; the stdout-JSON mode is the Copilot adaptation and must not leak here: %q", body)
	}
}

func TestCopilotSessionEndGateSignalsRefusalOnStdout(t *testing.T) {
	t.Parallel()
	projectDir := installMandatoryAgentsForDispatch(t)

	raw, err := os.ReadFile(filepath.Join(projectDir, ".github", "hooks", "governance.json"))
	if err != nil {
		t.Fatalf("read copilot governance hooks: %v", err)
	}
	var doc struct {
		Hooks map[string][]struct {
			Bash string `json:"bash"`
		} `json:"hooks"`
	}
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatalf("decode copilot governance hooks: %v", err)
	}

	event := sessionEndCliContracts2026[skills.ToolCopilot].BlockingEvent
	entries, ok := doc.Hooks[event]
	if !ok || len(entries) == 0 {
		t.Fatalf("the Copilot session-end gate must be declared on %q; got %+v", event, doc.Hooks)
	}

	command := ""
	for _, entry := range entries {
		if strings.Contains(entry.Bash, "validate-session-end.sh") {
			command = entry.Bash
		}
	}
	if command == "" {
		t.Fatalf("the canonical session-end validator must be wired to %q; got %+v", event, entries)
	}
	if !strings.HasPrefix(command, jsonDecisionModeEnv+" ") {
		t.Fatalf("Copilot blocks agentStop only through a stdout decision payload, so the gate must select the JSON decision mode; got %q", command)
	}
	if !strings.Contains(command, ".github/hooks/validate-session-end.sh") {
		t.Fatalf("the Copilot entry must delegate to the mirrored canonical gate, never reimplement the decision; got %q", command)
	}
}

func TestSessionEndAssumedBlockingExitCodeConstantIsGone(t *testing.T) {
	t.Parallel()

	raw, err := os.ReadFile("hooks_matrix_dispatch_test.go")
	if err != nil {
		t.Fatalf("read dispatch matrix test: %v", err)
	}
	if strings.Contains(string(raw), "sessionEndBlockingExitCode") {
		t.Fatalf("sessionEndBlockingExitCode asserted a CLI blocking semantic that was never measured for Codex and Copilot; the script-level contract must be named as such")
	}
}
