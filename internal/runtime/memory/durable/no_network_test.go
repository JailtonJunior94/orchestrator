package durable_test

import (
	"context"
	"go/parser"
	"go/token"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/JailtonJunior94/ai-spec-harness/internal/fs"
	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/memory/durable"
)

var forbiddenNetworkImports = []string{
	"net",
	"net/http",
	"net/http/httputil",
	"net/rpc",
	"net/smtp",
	"net/mail",
	"os/exec",
}

func TestDefaultMemoryPathImportsNoNetworkOrExecPackage(t *testing.T) {
	dir := "."
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read durable package directory: %v", err)
	}

	fset := token.NewFileSet()
	checked := 0
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		path := filepath.Join(dir, name)
		file, parseErr := parser.ParseFile(fset, path, nil, parser.ImportsOnly)
		if parseErr != nil {
			t.Fatalf("parse imports of %s: %v", path, parseErr)
		}
		checked++
		for _, imp := range file.Imports {
			importPath, unquoteErr := strconv.Unquote(imp.Path.Value)
			if unquoteErr != nil {
				t.Fatalf("unquote import in %s: %v", path, unquoteErr)
			}
			for _, forbidden := range forbiddenNetworkImports {
				if importPath == forbidden {
					t.Errorf("%s imports %q — RF-10 forbids network/exec packages on the default durable memory path", path, importPath)
				}
			}
		}
	}

	if checked == 0 {
		t.Fatal("no non-test .go files found in the durable package — scan did not run")
	}
}

func TestDefaultMemoryPathNeverDialsNetDefaultResolver(t *testing.T) {
	original := net.DefaultResolver
	t.Cleanup(func() { net.DefaultResolver = original })

	net.DefaultResolver = &net.Resolver{
		PreferGo: true,
		Dial: func(_ context.Context, network, address string) (net.Conn, error) {
			t.Fatalf("RF-10 violation: default durable memory path attempted DNS/network dial (network=%s address=%s)", network, address)
			return nil, nil
		},
	}

	projectDir := t.TempDir()
	tasksDir := t.TempDir()

	facade := durable.NewFacade(fs.NewOSFileSystem(), durable.FacadeConfig{
		ProjectDir: projectDir,
		TasksDir:   tasksDir,
	})

	ctx := context.Background()
	if _, err := facade.RecordSession(ctx, durable.SessionFacts{
		TaskFileName:    "task-network-guard.md",
		ExitStatus:      "none",
		DeclaredSection: "fact recorded while net.DefaultResolver is guarded",
		SessionID:       "session-network-guard",
		CLI:             "claude",
	}); err != nil {
		t.Fatalf("RecordSession under network guard: %v", err)
	}

	if _, err := facade.BuildContext(ctx, durable.MemoryScope{TaskFileName: "task-network-guard.md"}); err != nil {
		t.Fatalf("BuildContext under network guard: %v", err)
	}
}
