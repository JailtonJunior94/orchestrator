package specs_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"testing"
)

var adapterFilesUnderGate = []string{
	filepath.Join("internal", "runtime", "specs", "registry.go"),
	filepath.Join("internal", "runtime", "specs", "native_config.go"),
}

var forbiddenAdapterPolicyIdentifiers = map[string]bool{
	"Decision":              true,
	"DecisionAllow":         true,
	"DecisionBlock":         true,
	"DecisionWarn":          true,
	"DecisionNotApplicable": true,
	"DecisionError":         true,
	"NewResult":             true,
}

func scanFileForPolicyDecisionIdentifiers(t *testing.T, path string) []string {
	t.Helper()

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, path, nil, 0)
	if err != nil {
		t.Fatalf("parse %s: %v", path, err)
	}

	var found []string
	ast.Inspect(file, func(node ast.Node) bool {
		switch n := node.(type) {
		case *ast.Ident:
			if forbiddenAdapterPolicyIdentifiers[n.Name] {
				found = append(found, n.Name)
			}
		case *ast.SelectorExpr:
			if forbiddenAdapterPolicyIdentifiers[n.Sel.Name] {
				found = append(found, n.Sel.Name)
			}
		}
		return true
	})
	return found
}

func TestAdapterContainsNoPolicyDecision(t *testing.T) {
	t.Parallel()

	root := repoRoot(t)
	for _, rel := range adapterFilesUnderGate {
		path := filepath.Join(root, rel)
		found := scanFileForPolicyDecisionIdentifiers(t, path)
		if len(found) != 0 {
			t.Errorf("RF-56: adapter %q must contain only translation to native CLI vocabulary, never policy decision logic; found forbidden identifiers %v", rel, found)
		}
	}
}

func TestAdapterPolicyDecisionGateDetectsAPlantedViolation(t *testing.T) {
	t.Parallel()

	const fixture = `package specs

import "github.com/JailtonJunior94/ai-spec-harness/internal/hookcontract"

func evaluatePolicyInsideAdapter(risky bool) hookcontract.Decision {
	if risky {
		return hookcontract.DecisionBlock
	}
	return hookcontract.DecisionAllow
}
`
	dir := t.TempDir()
	path := filepath.Join(dir, "planted_adapter.go")
	if err := os.WriteFile(path, []byte(fixture), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	found := scanFileForPolicyDecisionIdentifiers(t, path)
	if len(found) == 0 {
		t.Fatal("expected the gate to detect policy decision logic planted inside an adapter-shaped file — a gate that never fires proves nothing")
	}
}
