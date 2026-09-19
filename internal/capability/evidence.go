package capability

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/specs"
)

func EvidenceCells(m Matrix) []specs.CapabilityCell {
	cells := make([]specs.CapabilityCell, 0, len(m.Cells))
	for _, c := range m.Cells {
		cells = append(cells, specs.CapabilityCell{
			Provider:   c.Provider,
			Capability: c.Capability,
			Supported:  c.State == StateSupported || c.State == StateProviderCapability,
		})
	}
	return cells
}

func ParityTestMethodExists(source []byte, suiteName, methodName string) bool {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "", source, 0)
	if err != nil {
		return false
	}

	receiverType := "*" + suiteName
	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Recv == nil || len(fn.Recv.List) != 1 {
			continue
		}
		if fn.Name.Name != methodName {
			continue
		}
		if exprString(fn.Recv.List[0].Type) == receiverType {
			return true
		}
	}
	return false
}

func exprString(expr ast.Expr) string {
	switch e := expr.(type) {
	case *ast.StarExpr:
		return "*" + exprString(e.X)
	case *ast.Ident:
		return e.Name
	default:
		return ""
	}
}

const evidenceTestSuite = "ParitySuite"

const defaultParityPackagePattern = "./internal/parity/..."

type goTestEvent struct {
	Action string `json:"Action"`
	Test   string `json:"Test"`
	Output string `json:"Output"`
}

func parseParityTestEvidence(output []byte, testName string) (bool, string) {
	passed := false
	skipped := false
	failed := false
	var diag strings.Builder

	scanner := bufio.NewScanner(bytes.NewReader(output))
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
		if event.Test != testName {
			continue
		}
		switch event.Action {
		case "pass":
			passed = true
		case "skip":
			skipped = true
		case "fail":
			failed = true
		}
	}

	if skipped || failed {
		return false, diag.String()
	}
	return passed, diag.String()
}

func runPatternFor(testName string) string {
	segments := strings.Split(testName, "/")
	for i, seg := range segments {
		segments[i] = "^" + regexp.QuoteMeta(seg) + "$"
	}
	return strings.Join(segments, "/")
}

func executionProof(dir, pkgPattern, testName string) bool {
	cmd := exec.Command("go", "test", "-json", "-count=1", "-run", runPatternFor(testName), pkgPattern)
	cmd.Dir = dir
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &bytes.Buffer{}
	_ = cmd.Run()

	passed, _ := parseParityTestEvidence(stdout.Bytes(), testName)
	return passed
}

func dispatchProvenFromTests(source []byte, suiteName, testName, dir, pkgPattern string) specs.CapabilityDispatchProofFunc {
	methodName := testName
	if idx := strings.LastIndex(testName, "/"); idx >= 0 {
		methodName = testName[idx+1:]
	}
	declared := ParityTestMethodExists(source, suiteName, methodName)
	resolved := declared && executionProof(dir, pkgPattern, testName)
	return func(provider, capabilityID string) bool {
		return resolved
	}
}

func repoRootFromWorkingDir() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if _, statErr := os.Stat(filepath.Join(dir, "go.mod")); statErr == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", errors.New("could not locate repo root walking up from working directory")
		}
		dir = parent
	}
}

func DispatchProvenFromParityTests(source []byte) specs.CapabilityDispatchProofFunc {
	root, err := repoRootFromWorkingDir()
	if err != nil {
		return func(provider, capabilityID string) bool { return false }
	}
	return dispatchProvenFromTests(source, evidenceTestSuite, EvidenceTest, root, defaultParityPackagePattern)
}
