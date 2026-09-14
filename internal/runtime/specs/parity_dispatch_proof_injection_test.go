package specs_test

import (
	"regexp"
	"strings"
	"testing"
)

var legacyPassedSubtestPattern = regexp.MustCompile(`(?m)^\s*--- PASS: (\S+)`)

const forgedCell = "TestSessionEndHookDispatchBlocksActiveTaskWithoutApprovedVerdict/codex"

func forgedDispatchProofStream() string {
	return strings.Join([]string{
		`{"Action":"run","Test":"TestPreToolHookDispatchBlocksWhenSkillPrerequisiteMissing/claude"}`,
		`{"Action":"output","Test":"TestPreToolHookDispatchBlocksWhenSkillPrerequisiteMissing/claude","Output":"    proof_test.go:1: \n--- PASS: ` + forgedCell + ` (0.00s)\n"}`,
		`{"Action":"skip","Test":"TestPreToolHookDispatchBlocksWhenSkillPrerequisiteMissing/claude"}`,
		`{"Action":"run","Test":"TestPostToolHookDispatchBlocksGovernanceFileEdit/claude"}`,
		`{"Action":"pass","Test":"TestPostToolHookDispatchBlocksGovernanceFileEdit/claude"}`,
		"",
	}, "\n")
}

func TestLogfInjectionNoLongerForgesDispatchEvidence(t *testing.T) {
	t.Parallel()

	stream := forgedDispatchProofStream()
	evidence, diag := parseDispatchEvidence(strings.NewReader(stream))

	if evidence[forgedCell] {
		t.Fatalf("forged %q via t.Logf must never count as execution evidence", forgedCell)
	}
	if evidence["TestPreToolHookDispatchBlocksWhenSkillPrerequisiteMissing/claude"] {
		t.Fatalf("a skipped cell must never count as execution evidence")
	}
	if !evidence["TestPostToolHookDispatchBlocksGovernanceFileEdit/claude"] {
		t.Fatalf("a genuine pass action must count as execution evidence")
	}
	if !strings.Contains(diag, "--- PASS: "+forgedCell) {
		t.Fatalf("forged text must still reach the diagnostic log; got %q", diag)
	}

	legacy := map[string]bool{}
	for _, m := range legacyPassedSubtestPattern.FindAllStringSubmatch(diag, -1) {
		legacy[m[1]] = true
	}
	if !legacy[forgedCell] {
		t.Fatalf("regression fixture is inert: the removed text parser must have accepted the forged PASS line")
	}
}

func TestSkipActionRevokesPassEvidence(t *testing.T) {
	t.Parallel()

	stream := strings.Join([]string{
		`{"Action":"pass","Test":"TestSomething/opencode"}`,
		`{"Action":"skip","Test":"TestSomething/opencode"}`,
		"",
	}, "\n")
	evidence, _ := parseDispatchEvidence(strings.NewReader(stream))
	if evidence["TestSomething/opencode"] {
		t.Fatalf("a cell reported as skipped must not keep execution evidence")
	}
}
