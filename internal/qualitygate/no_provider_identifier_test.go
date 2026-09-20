package qualitygate

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

var forbiddenProviderIdentifiers = []string{"claude", "codex", "copilot", "opencode"}

func TestQualityGateHasNoProviderIdentifier(t *testing.T) {
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("read package directory: %v", err)
	}

	fileSet := token.NewFileSet()
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") || strings.HasSuffix(entry.Name(), "_test.go") {
			continue
		}
		file, err := parser.ParseFile(fileSet, filepath.Join(".", entry.Name()), nil, parser.AllErrors)
		if err != nil {
			t.Fatalf("parse %s: %v", entry.Name(), err)
		}
		ast.Inspect(file, func(node ast.Node) bool {
			switch n := node.(type) {
			case *ast.Ident:
				failIfForbiddenProvider(t, entry.Name(), n.Name)
			case *ast.BasicLit:
				if n.Kind == token.STRING {
					failIfForbiddenProvider(t, entry.Name(), n.Value)
				}
			}
			return true
		})
	}
}

func failIfForbiddenProvider(t *testing.T, filename, text string) {
	t.Helper()
	lower := strings.ToLower(text)
	for _, forbidden := range forbiddenProviderIdentifiers {
		if strings.Contains(lower, forbidden) {
			t.Fatalf("%s contains forbidden provider identifier %q in %q", filename, forbidden, text)
		}
	}
}
