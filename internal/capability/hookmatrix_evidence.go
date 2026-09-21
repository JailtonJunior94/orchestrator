package capability

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/specs"
)

func hookCapabilityKey(event, family string) string {
	return event + "/" + family
}

func EvidenceHookCells(m HookMatrix) []specs.CapabilityCell {
	cells := make([]specs.CapabilityCell, 0, len(m.Cells))
	for _, c := range m.Cells {
		affirmative := c.State == "verified" || c.State == "adapter"
		cells = append(cells, specs.CapabilityCell{
			Provider:   c.Provider,
			Capability: hookCapabilityKey(c.Event, c.Family),
			Supported:  affirmative,
		})
	}
	return cells
}

func hookTestFor(m HookMatrix, provider, capability string) string {
	parts := strings.SplitN(capability, "/", 2)
	if len(parts) != 2 {
		return ""
	}
	event, family := parts[0], parts[1]
	for _, c := range m.Cells {
		if c.Provider == provider && c.Event == event && c.Family == family {
			return c.Test
		}
	}
	return ""
}

const integrationTestsRelDir = "tests/integration"

func topLevelIntegrationTestNames(dir string) (map[string]bool, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", dir, err)
	}

	fset := token.NewFileSet()
	found := make(map[string]bool)
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".go" {
			continue
		}
		path := filepath.Join(dir, entry.Name())
		file, parseErr := parser.ParseFile(fset, path, nil, 0)
		if parseErr != nil {
			return nil, fmt.Errorf("parse %s: %w", path, parseErr)
		}
		for _, decl := range file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Recv != nil {
				continue
			}
			found[fn.Name.Name] = true
		}
	}
	return found, nil
}

func hookDispatchTestNames(m HookMatrix) []string {
	seen := make(map[string]bool)
	var names []string
	for _, c := range m.Cells {
		if c.Test == "" {
			continue
		}
		if seen[c.Test] {
			continue
		}
		seen[c.Test] = true
		names = append(names, c.Test)
	}
	sort.Strings(names)
	return names
}

func hookDispatchTopLevelName(testName string) string {
	if idx := strings.Index(testName, "/"); idx >= 0 {
		return testName[:idx]
	}
	return testName
}

func runHookDispatchProofSuite(root string, testNames []string) (map[string]bool, string, error) {
	if len(testNames) == 0 {
		return map[string]bool{}, "", nil
	}
	top := make(map[string]bool, len(testNames))
	for _, name := range testNames {
		top[hookDispatchTopLevelName(name)] = true
	}
	patterns := make([]string, 0, len(top))
	for name := range top {
		patterns = append(patterns, "^"+name+"$")
	}
	sort.Strings(patterns)
	pattern := strings.Join(patterns, "|")

	cmd := exec.Command("go", "test", "-tags=integration", "./"+integrationTestsRelDir+"/",
		"-count=1", "-json", "-timeout", "20m", "-run", pattern)
	cmd.Dir = root
	cmd.Env = append(os.Environ(), "GOFLAGS=")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	runErr := cmd.Run()

	evidence, diag := parseParityTestEvidenceMulti(stdout.Bytes())
	log := diag + stderr.String()
	if runErr != nil {
		return evidence, log, fmt.Errorf("hook dispatch proof suite failed: %w", runErr)
	}
	return evidence, log, nil
}

func parseParityTestEvidenceMulti(output []byte) (map[string]bool, string) {
	passed := make(map[string]bool)
	skipped := make(map[string]bool)
	failed := make(map[string]bool)
	var diag strings.Builder

	for _, line := range bytes.Split(output, []byte("\n")) {
		line = bytes.TrimSpace(line)
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

var ErrHookDispatchProofEnvironment = errors.New("hook dispatch proof environment resolution failed")

func HookDispatchProvenFromIntegrationTests(m HookMatrix) (specs.CapabilityDispatchProofFunc, error) {
	root, err := repoRootFromWorkingDir()
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrHookDispatchProofEnvironment, err)
	}

	declared, err := topLevelIntegrationTestNames(filepath.Join(root, filepath.FromSlash(integrationTestsRelDir)))
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrHookDispatchProofEnvironment, err)
	}

	testNames := hookDispatchTestNames(m)
	for _, name := range testNames {
		if !declared[hookDispatchTopLevelName(name)] {
			return nil, fmt.Errorf("hook capability matrix declares test %q, which does not exist in %s", name, integrationTestsRelDir)
		}
	}

	evidence, log, runErr := runHookDispatchProofSuite(root, testNames)
	if runErr != nil {
		return nil, fmt.Errorf("run hook dispatch proof suite: %w\n%s", runErr, log)
	}

	return func(provider, capabilityID string) bool {
		testName := hookTestFor(m, provider, capabilityID)
		if testName == "" {
			return false
		}
		return evidence[testName]
	}, nil
}
