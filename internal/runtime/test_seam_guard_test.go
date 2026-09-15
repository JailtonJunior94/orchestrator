package runtime_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"strings"
	"testing"
)

func productionSourceFiles(t *testing.T) []string {
	t.Helper()
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("read package dir: %v", err)
	}
	var files []string
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		files = append(files, name)
	}
	if len(files) == 0 {
		t.Fatal("no production sources found in internal/runtime")
	}
	return files
}

func TestProductionSourcesExposeNoVerdictInjectionSeam(t *testing.T) {
	t.Parallel()

	fset := token.NewFileSet()
	for _, name := range productionSourceFiles(t) {
		file, err := parser.ParseFile(fset, name, nil, parser.SkipObjectResolution)
		if err != nil {
			t.Fatalf("parse %s: %v", name, err)
		}
		for _, decl := range file.Decls {
			for _, ident := range exportedDeclNames(decl) {
				if strings.HasSuffix(ident, "ForTest") || strings.HasSuffix(ident, "ForTests") {
					t.Fatalf("RF-05/V-19: %s exports %q; test seams must live in export_test.go so production code cannot inject canned behaviour", name, ident)
				}
				if strings.Contains(ident, "ReviewOutputFn") || strings.Contains(ident, "ReviewOutput") {
					t.Fatalf("RF-05/V-19: %s exports %q; the auto-review verdict must be derived from a real review session, never from an injectable output supplied through the public API", name, ident)
				}
			}
		}
	}
}

func exportedDeclNames(decl ast.Decl) []string {
	var names []string
	switch typed := decl.(type) {
	case *ast.FuncDecl:
		if typed.Name.IsExported() {
			names = append(names, typed.Name.Name)
		}
	case *ast.GenDecl:
		for _, spec := range typed.Specs {
			switch s := spec.(type) {
			case *ast.TypeSpec:
				if s.Name.IsExported() {
					names = append(names, s.Name.Name)
				}
			case *ast.ValueSpec:
				for _, ident := range s.Names {
					if ident.IsExported() {
						names = append(names, ident.Name)
					}
				}
			}
		}
	}
	return names
}
