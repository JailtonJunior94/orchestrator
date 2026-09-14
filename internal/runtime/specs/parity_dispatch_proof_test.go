package specs_test

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"testing"

	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/specs"
)

const (
	integrationTestsRelDir  = "tests/integration"
	dispatchProofRunTimeout = "20m"
)

type dispatchProofCase struct {
	TestName string
	Subtest  string
}

func (c dispatchProofCase) key() string {
	if c.Subtest == "" {
		return c.TestName
	}
	return c.TestName + "/" + c.Subtest
}

var dispatchProofRegistry = map[string]map[specs.CanonicalPoint]dispatchProofCase{
	"claude": {
		specs.PointPreTool:    {TestName: "TestPreToolHookDispatchBlocksWhenSkillPrerequisiteMissing", Subtest: "claude"},
		specs.PointPostTool:   {TestName: "TestPostToolHookDispatchBlocksGovernanceFileEdit", Subtest: "claude"},
		specs.PointSessionEnd: {TestName: "TestSessionEndHookDispatchBlocksActiveTaskWithoutApprovedVerdict", Subtest: "claude"},
	},
	"codex": {
		specs.PointPreTool:    {TestName: "TestPreToolHookDispatchBlocksWhenSkillPrerequisiteMissing", Subtest: "codex"},
		specs.PointPostTool:   {TestName: "TestPostToolHookDispatchBlocksGovernanceFileEdit", Subtest: "codex"},
		specs.PointSessionEnd: {TestName: "TestSessionEndHookDispatchBlocksActiveTaskWithoutApprovedVerdict", Subtest: "codex"},
	},
	"copilot": {
		specs.PointPreTool:    {TestName: "TestPreToolHookDispatchBlocksWhenSkillPrerequisiteMissing", Subtest: "copilot"},
		specs.PointPostTool:   {TestName: "TestPostToolHookDispatchBlocksGovernanceFileEdit", Subtest: "copilot"},
		specs.PointSessionEnd: {TestName: "TestSessionEndHookDispatchBlocksActiveTaskWithoutApprovedVerdict", Subtest: "copilot"},
	},
	"opencode": {
		specs.PointPreTool:    {TestName: "TestOpenCodeGovernancePluginBlocksWhenSkillPrerequisiteMissing"},
		specs.PointPostTool:   {TestName: "TestOpenCodeGovernancePluginToolExecuteAfterObservesValidatorWithoutBlocking"},
		specs.PointSessionEnd: {TestName: "TestOpenCodeGovernancePluginSessionIdleBlocksActiveTaskWithoutApprovedVerdict"},
	},
}

func repoRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	for {
		if _, statErr := os.Stat(filepath.Join(dir, "go.mod")); statErr == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatalf("could not locate repo root walking up from %s", dir)
		}
		dir = parent
	}
}

func integrationTestFunctionNames(t *testing.T) map[string]string {
	t.Helper()
	dir := filepath.Join(repoRoot(t), filepath.FromSlash(integrationTestsRelDir))
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read %s: %v", dir, err)
	}

	fset := token.NewFileSet()
	found := make(map[string]string)
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".go" {
			continue
		}
		path := filepath.Join(dir, entry.Name())
		file, parseErr := parser.ParseFile(fset, path, nil, 0)
		if parseErr != nil {
			t.Fatalf("parse %s: %v", path, parseErr)
		}
		for _, decl := range file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Recv != nil {
				continue
			}
			found[fn.Name.Name] = entry.Name()
		}
	}
	return found
}

func declaredProofTestNames() []string {
	seen := make(map[string]bool)
	names := make([]string, 0, len(dispatchProofRegistry)*3)
	for _, byPoint := range dispatchProofRegistry {
		for _, proof := range byPoint {
			if seen[proof.TestName] {
				continue
			}
			seen[proof.TestName] = true
			names = append(names, proof.TestName)
		}
	}
	sort.Strings(names)
	return names
}

type goTestEvent struct {
	Action string `json:"Action"`
	Test   string `json:"Test"`
	Output string `json:"Output"`
}

// parseDispatchEvidence consome o fluxo estruturado de `go test -json`.
// A evidencia vem exclusivamente das acoes "pass"/"skip"/"fail" emitidas pelo
// proprio test runner. Texto impresso pelo teste chega como acao "output" e
// nunca vira evidencia: antes desta correcao o gate fazia regex sobre o texto
// e um `t.Logf("--- PASS: <outra-celula>")` seguido de t.Skip forjava a prova.
func parseDispatchEvidence(r io.Reader) (map[string]bool, string) {
	passed := make(map[string]bool)
	skipped := make(map[string]bool)
	failed := make(map[string]bool)
	var diag strings.Builder

	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, 64*1024), 8*1024*1024)
	for scanner.Scan() {
		line := bytes.TrimSpace(scanner.Bytes())
		if len(line) == 0 || line[0] != '{' {
			diag.Write(line)
			diag.WriteByte('\n')
			continue
		}
		var event goTestEvent
		if err := json.Unmarshal(line, &event); err != nil {
			diag.Write(line)
			diag.WriteByte('\n')
			continue
		}
		if event.Action == "output" {
			diag.WriteString(event.Output)
			continue
		}
		if event.Test == "" {
			continue
		}
		switch event.Action {
		case "pass":
			passed[event.Test] = true
		case "skip":
			skipped[event.Test] = true
		case "fail":
			failed[event.Test] = true
		}
	}

	evidence := make(map[string]bool, len(passed))
	for name := range passed {
		if skipped[name] || failed[name] {
			continue
		}
		evidence[name] = true
	}
	return evidence, diag.String()
}

var (
	executionEvidenceOnce sync.Once
	executionEvidence     map[string]bool
	executionEvidenceLog  string
	executionEvidenceErr  error
)

func runDispatchProofSuite() (map[string]bool, string, error) {
	root, err := os.Getwd()
	if err != nil {
		return nil, "", err
	}
	for {
		if _, statErr := os.Stat(filepath.Join(root, "go.mod")); statErr == nil {
			break
		}
		parent := filepath.Dir(root)
		if parent == root {
			return nil, "", fmt.Errorf("could not locate repo root")
		}
		root = parent
	}

	pattern := "^(" + strings.Join(declaredProofTestNames(), "|") + ")$"
	cmd := exec.Command("go", "test", "-tags=integration", "./"+integrationTestsRelDir+"/",
		"-count=1", "-json", "-timeout", dispatchProofRunTimeout, "-run", pattern)
	cmd.Dir = root
	cmd.Env = append(os.Environ(), "GOFLAGS=")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	runErr := cmd.Run()

	evidence, diag := parseDispatchEvidence(bytes.NewReader(stdout.Bytes()))
	log := diag + stderr.String()
	if runErr != nil {
		return evidence, log, fmt.Errorf("dispatch proof suite failed: %w", runErr)
	}
	return evidence, log, nil
}

func dispatchExecutionEvidence(t *testing.T) map[string]bool {
	t.Helper()
	executionEvidenceOnce.Do(func() {
		executionEvidence, executionEvidenceLog, executionEvidenceErr = runDispatchProofSuite()
	})
	if executionEvidenceErr != nil {
		t.Fatalf("could not collect dispatch execution evidence: %v\n%s", executionEvidenceErr, executionEvidenceLog)
	}
	return executionEvidence
}

func dispatchProofFor(t *testing.T) specs.DispatchProofFunc {
	t.Helper()
	evidence := dispatchExecutionEvidence(t)
	return func(agentID string, point specs.CanonicalPoint) bool {
		byPoint, ok := dispatchProofRegistry[agentID]
		if !ok {
			return false
		}
		proof, ok := byPoint[point]
		if !ok || proof.TestName == "" {
			return false
		}
		return evidence[proof.key()]
	}
}

func TestDispatchProofRegistryNamesExistingIntegrationTests(t *testing.T) {
	t.Parallel()

	declared := integrationTestFunctionNames(t)
	for agentID, byPoint := range dispatchProofRegistry {
		for point, proof := range byPoint {
			file, ok := declared[proof.TestName]
			if !ok {
				t.Errorf("agent=%s point=%s: dispatch proof %q does not exist in %s", agentID, point, proof.TestName, integrationTestsRelDir)
				continue
			}
			t.Logf("agent=%s point=%s declared by %s (%s)", agentID, point, proof.key(), file)
		}
	}
}

func TestDispatchProofRegistryCoversEveryMandatoryCell(t *testing.T) {
	t.Parallel()

	points := specs.NewCatalog().CanonicalPoints()
	for _, agentID := range mandatoryParityAgents {
		byPoint, ok := dispatchProofRegistry[agentID]
		if !ok {
			t.Errorf("agent %q has no dispatch proof entry", agentID)
			continue
		}
		if len(byPoint) != len(points) {
			t.Errorf("agent %q declares %d dispatch proofs; want %d", agentID, len(byPoint), len(points))
		}
		for _, point := range points {
			if byPoint[point].TestName == "" {
				t.Errorf("agent %q point %s has no dispatch proof test", agentID, point)
			}
		}
	}
}

func TestEveryMandatoryCellHasExecutionEvidence(t *testing.T) {
	evidence := dispatchExecutionEvidence(t)
	points := specs.NewCatalog().CanonicalPoints()

	for _, agentID := range mandatoryParityAgents {
		for _, point := range points {
			proof := dispatchProofRegistry[agentID][point]
			if proof.TestName == "" {
				t.Errorf("agent=%s point=%s has no dispatch proof declared", agentID, point)
				continue
			}
			if !evidence[proof.key()] {
				t.Errorf("agent=%s point=%s has no execution evidence: %q never reported PASS in %s",
					agentID, point, proof.key(), integrationTestsRelDir)
			}
		}
	}
}

func TestParityGateHasZeroViolationsWithDispatchProofs(t *testing.T) {
	violations := specs.ValidateParityMatrix(mandatoryParityCells(t), mandatoryParityAgents, dispatchProofFor(t))
	if len(violations) != 0 {
		t.Fatalf("RF-28 parity gate must report zero violations; got %v", violations)
	}
}
