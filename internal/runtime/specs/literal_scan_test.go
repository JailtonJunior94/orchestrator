package specs_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

var literalScanAllowlist = map[string]string{
	"registry.go":      "defines the single agent registry (RF-07) — source of truth",
	"flow.go":          "fine-grained per-flow budget matrix (tool x governance x skill x flow); not one of the per-window budget maps migrated in 6.0",
	"parity.go":        "parity test fixture path map, not a runtime enumeration",
	"agent.go":         "legacy --tool dispatch (deprecated invokers, ADR-013); outside the ACP path",
	"compatibility.go": "CompatibilityTable tool x model — compatibility concern, not the agent catalog",
	"profile.go":       "provider-by-IDE map and the mapIDEToSpec phase gate (claude-only in ACP for this phase)",
	"wrapper.go":       "legacy per-tool wrapper generation; outside the ACP path",
}

var agentIDs = []string{"claude", "codex", "copilot", "gemini"}

func TestNoAgentListLiteralOutsideRegistry(t *testing.T) {
	t.Parallel()

	repoRoot, err := filepath.Abs(filepath.Join("..", "..", ".."))
	if err != nil {
		t.Fatalf("abs repo root: %v", err)
	}

	var offenders []string
	for _, sub := range []string{"internal", "cmd"} {
		root := filepath.Join(repoRoot, sub)
		walkErr := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
				return nil
			}
			if _, ok := literalScanAllowlist[filepath.Base(path)]; ok {
				return nil
			}
			if literalEnumeratesAgents(t, path) {
				rel, _ := filepath.Rel(repoRoot, path)
				offenders = append(offenders, rel)
			}
			return nil
		})
		if walkErr != nil {
			t.Fatalf("walk %s: %v", sub, walkErr)
		}
	}

	if len(offenders) > 0 {
		t.Fatalf("agent-list literals outside the registry: %v — derive from the registry or justify in literalScanAllowlist", offenders)
	}
}

func literalEnumeratesAgents(t *testing.T, path string) bool {
	t.Helper()
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, path, nil, 0)
	if err != nil {
		t.Fatalf("parse %s: %v", path, err)
	}
	found := false
	ast.Inspect(file, func(n ast.Node) bool {
		lit, ok := n.(*ast.CompositeLit)
		if !ok {
			return true
		}
		seen := make(map[string]bool)
		for _, elt := range lit.Elts {
			for _, s := range stringLiterals(elt) {
				for _, id := range agentIDs {
					if s == id {
						seen[id] = true
					}
				}
			}
		}
		if len(seen) >= 3 {
			found = true
		}
		return true
	})
	return found
}

func stringLiterals(n ast.Node) []string {
	var out []string
	ast.Inspect(n, func(x ast.Node) bool {
		if bl, ok := x.(*ast.BasicLit); ok && bl.Kind == token.STRING {
			out = append(out, strings.Trim(bl.Value, "`\""))
		}
		return true
	})
	return out
}
