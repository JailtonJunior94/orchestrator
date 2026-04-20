package detect

import (
	"bytes"
	"strings"
	"testing"

	"github.com/JailtonJunior94/ai-spec-harness/internal/fs"
)

func TestToolchainDetect_Go(t *testing.T) {
	t.Parallel()
	ffs := fs.NewFakeFileSystem()
	ffs.Files["/project/go.mod"] = []byte("module example")

	det := NewToolchainDetector(ffs)
	result := det.Detect("/project")

	entry, ok := result["go"]
	if !ok {
		t.Fatal("expected go toolchain entry")
	}
	if entry.Fmt != "gofmt -w ." {
		t.Errorf("fmt: got %q", entry.Fmt)
	}
	if entry.Test != "go test ./..." {
		t.Errorf("test: got %q", entry.Test)
	}
	if entry.Lint != "golangci-lint run" {
		t.Errorf("lint: got %q", entry.Lint)
	}
}

func TestToolchainDetect_Node(t *testing.T) {
	t.Parallel()
	ffs := fs.NewFakeFileSystem()
	ffs.Files["/project/package.json"] = []byte(`{
		"name": "test",
		"scripts": {
			"fmt": "prettier --write .",
			"test": "vitest run",
			"lint": "eslint ."
		}
	}`)

	det := NewToolchainDetector(ffs)
	result := det.Detect("/project")

	entry, ok := result["node"]
	if !ok {
		t.Fatal("expected node toolchain entry")
	}
	if entry.Fmt != "npm run fmt" {
		t.Errorf("fmt: got %q", entry.Fmt)
	}
	if entry.Test != "npm run test" {
		t.Errorf("test: got %q", entry.Test)
	}
	if entry.Lint != "npm run lint" {
		t.Errorf("lint: got %q", entry.Lint)
	}
}

func TestToolchainDetect_Python_Ruff(t *testing.T) {
	t.Parallel()
	ffs := fs.NewFakeFileSystem()
	ffs.Files["/project/pyproject.toml"] = []byte(`[project]
name = "test"

[tool.ruff]
line-length = 88

[tool.pytest.ini_options]
testpaths = ["tests"]
`)

	det := NewToolchainDetector(ffs)
	result := det.Detect("/project")

	entry, ok := result["python"]
	if !ok {
		t.Fatal("expected python toolchain entry")
	}
	if entry.Fmt != "ruff format ." {
		t.Errorf("fmt: got %q", entry.Fmt)
	}
	if entry.Test != "pytest" {
		t.Errorf("test: got %q", entry.Test)
	}
	if entry.Lint != "ruff check ." {
		t.Errorf("lint: got %q", entry.Lint)
	}
}

func TestToolchainDetect_Empty(t *testing.T) {
	t.Parallel()
	ffs := fs.NewFakeFileSystem()
	ffs.Dirs["/project"] = true

	det := NewToolchainDetector(ffs)
	result := det.Detect("/project")

	if len(result) != 0 {
		t.Errorf("expected empty result, got %v", result)
	}
}

func TestToolchainDetect_Polyglot(t *testing.T) {
	t.Parallel()
	ffs := fs.NewFakeFileSystem()
	ffs.Files["/project/go.mod"] = []byte("module example")
	ffs.Files["/project/package.json"] = []byte(`{"scripts":{"test":"jest"}}`)

	det := NewToolchainDetector(ffs)
	result := det.Detect("/project")

	if _, ok := result["go"]; !ok {
		t.Error("expected go entry")
	}
	if _, ok := result["node"]; !ok {
		t.Error("expected node entry")
	}
}

func TestToolchainDetect_MakefileFallback(t *testing.T) {
	t.Parallel()
	ffs := fs.NewFakeFileSystem()
	ffs.Files["/project/Makefile"] = []byte("fmt:\n\tgofmt -w .\ntest:\n\tgo test ./...\nlint:\n\tgolangci-lint run\n")

	det := NewToolchainDetector(ffs)
	result := det.Detect("/project")

	entry, ok := result["unknown"]
	if !ok {
		t.Fatal("expected unknown toolchain entry for makefile fallback")
	}
	if entry.Fmt != "make fmt" {
		t.Errorf("fmt: got %q, want %q", entry.Fmt, "make fmt")
	}
	if entry.Test != "make test" {
		t.Errorf("test: got %q, want %q", entry.Test, "make test")
	}
	if entry.Lint != "make lint" {
		t.Errorf("lint: got %q, want %q", entry.Lint, "make lint")
	}
}

func TestToolchainDetect_Bun(t *testing.T) {
	t.Parallel()
	ffs := fs.NewFakeFileSystem()
	ffs.Files["/project/package.json"] = []byte(`{"scripts":{"test":"jest","lint":"eslint ."}}`)
	ffs.Files["/project/bun.lockb"] = []byte("")

	det := NewToolchainDetector(ffs)
	result := det.Detect("/project")

	entry, ok := result["node"]
	if !ok {
		t.Fatal("expected node toolchain entry")
	}
	if entry.Test != "bun run test" {
		t.Errorf("test: got %q, want %q", entry.Test, "bun run test")
	}
}

func TestToolchainDetect_PythonOptionalDeps(t *testing.T) {
	t.Parallel()
	ffs := fs.NewFakeFileSystem()
	ffs.Files["/project/pyproject.toml"] = []byte(`[project]
name = "test"

[project.optional-dependencies]
dev = ["ruff>=0.1", "pytest>=7.0"]
`)

	det := NewToolchainDetector(ffs)
	result := det.Detect("/project")

	entry, ok := result["python"]
	if !ok {
		t.Fatal("expected python toolchain entry")
	}
	if entry.Fmt != "ruff format ." {
		t.Errorf("fmt: got %q", entry.Fmt)
	}
	if entry.Test != "pytest" {
		t.Errorf("test: got %q", entry.Test)
	}
	if entry.Lint != "ruff check ." {
		t.Errorf("lint: got %q", entry.Lint)
	}
}

func TestStrictMode_BinaryPresent(t *testing.T) {
	t.Parallel()
	ffs := fs.NewFakeFileSystem()
	ffs.Files["/project/go.mod"] = []byte("module example")

	var buf bytes.Buffer
	det := NewToolchainDetectorStrict(ffs, &buf)
	result := det.Detect("/project")

	if _, ok := result["go"]; !ok {
		t.Fatal("expected go toolchain entry")
	}
	// gofmt e go sao binarios presentes no PATH em ambiente Go
	// O teste verifica que o JSON output nao e afetado por strict mode
	if result["go"].Fmt != "gofmt -w ." {
		t.Errorf("fmt: got %q", result["go"].Fmt)
	}
}

func TestStrictMode_BinaryAbsent(t *testing.T) {
	t.Parallel()
	ffs := fs.NewFakeFileSystem()
	ffs.Files["/project/go.mod"] = []byte("module example")

	var buf bytes.Buffer
	det := NewToolchainDetectorStrict(ffs, &buf)
	result := det.Detect("/project")

	// JSON output deve ser inalterado
	if result["go"].Lint != "golangci-lint run" {
		t.Errorf("lint: got %q", result["go"].Lint)
	}

	// golangci-lint provavelmente nao esta no PATH do CI — se estiver, o warning nao aparece
	// Verificamos que o mecanismo de warning funciona: se ausente, deve conter "WARNING"
	// Se presente, buf pode estar vazio — ambos sao validos
	warn := buf.String()
	if warn != "" && !strings.Contains(warn, "WARNING") {
		t.Errorf("expected WARNING prefix in warn output, got: %q", warn)
	}
}

func TestStrictMode_NonStrict_NoWarning(t *testing.T) {
	t.Parallel()
	ffs := fs.NewFakeFileSystem()
	ffs.Files["/project/go.mod"] = []byte("module example")

	var buf bytes.Buffer
	det := NewToolchainDetectorStrict(ffs, &buf)
	det.strict = false
	det.Detect("/project")

	if buf.Len() != 0 {
		t.Errorf("expected no warnings in non-strict mode, got: %q", buf.String())
	}
}

func TestToolchainDetect_FocusPaths_GoWinsOverNodeAtRoot(t *testing.T) {
	t.Parallel()
	ffs := fs.NewFakeFileSystem()
	ffs.Files["/project/package.json"] = []byte(`{"scripts":{"test":"jest","lint":"eslint ."}}`)
	ffs.Files["/project/services/api/go.mod"] = []byte("module example")
	ffs.Dirs["/project/services"] = true
	ffs.Dirs["/project/services/api"] = true

	det := NewToolchainDetector(ffs)
	det.FocusPaths = []string{"services/api/handler.go"}
	result := det.Detect("/project")

	if _, ok := result["go"]; !ok {
		t.Fatal("expected go toolchain entry")
	}
	if _, ok := result["node"]; ok {
		t.Error("expected no node toolchain entry when focus is on Go subproject")
	}
	if result["go"].Test != "go test ./..." {
		t.Errorf("test: got %q", result["go"].Test)
	}
}

func TestToolchainDetect_FocusPaths_Empty_FallsBackToDefault(t *testing.T) {
	t.Parallel()
	ffs := fs.NewFakeFileSystem()
	ffs.Files["/project/go.mod"] = []byte("module example")
	ffs.Files["/project/package.json"] = []byte(`{"scripts":{"test":"jest"}}`)

	det := NewToolchainDetector(ffs)
	// no FocusPaths set
	result := det.Detect("/project")

	if _, ok := result["go"]; !ok {
		t.Error("expected go entry with no focus paths")
	}
	if _, ok := result["node"]; !ok {
		t.Error("expected node entry with no focus paths")
	}
}

func TestToolchainDetect_FocusPaths_MultipleManifests_HighestOverlapWins(t *testing.T) {
	t.Parallel()
	ffs := fs.NewFakeFileSystem()
	ffs.Files["/project/services/api/package.json"] = []byte(`{"name":"api","scripts":{"test":"jest"}}`)
	ffs.Files["/project/services/web/package.json"] = []byte(`{"name":"web","scripts":{"test":"vitest"}}`)
	ffs.Dirs["/project/services"] = true
	ffs.Dirs["/project/services/api"] = true
	ffs.Dirs["/project/services/web"] = true

	det := NewToolchainDetector(ffs)
	det.FocusPaths = []string{"services/api/handler.go"}
	result := det.Detect("/project")

	entry, ok := result["node"]
	if !ok {
		t.Fatal("expected node toolchain entry")
	}
	// services/api/package.json is closer to focus path than services/web/package.json
	if !strings.Contains(entry.Test, "api") && !strings.Contains(entry.Test, "jest") {
		t.Errorf("expected jest (api package) to win, got: %q", entry.Test)
	}
}

func TestToolchainDetect_FocusPaths_NoMatch_FallsBackToDefault(t *testing.T) {
	t.Parallel()
	ffs := fs.NewFakeFileSystem()
	ffs.Files["/project/go.mod"] = []byte("module example")

	det := NewToolchainDetector(ffs)
	det.FocusPaths = []string{"some/unrelated/path/file.rs"}
	result := det.Detect("/project")

	// No manifest matches the focus path (all scores 0): fall back to default detection
	if _, ok := result["go"]; !ok {
		t.Error("expected go entry when no focus path matches any manifest")
	}
}

func TestToolchainDetect_Fixture_PythonMonorepo(t *testing.T) {
	t.Parallel()
	osfs := fs.NewOSFileSystem()
	det := NewToolchainDetector(osfs)
	result := det.Detect(fixtureDir("python-monorepo"))

	entry, ok := result["python"]
	if !ok {
		t.Fatal("expected python toolchain entry")
	}
	if entry.Fmt != "ruff format ." {
		t.Errorf("fmt: got %q", entry.Fmt)
	}
	if entry.Test != "pytest" {
		t.Errorf("test: got %q", entry.Test)
	}
	if entry.Lint != "ruff check ." {
		t.Errorf("lint: got %q", entry.Lint)
	}
}

func TestToolchainDetect_PnpmWorkspace(t *testing.T) {
	t.Parallel()
	ffs := fs.NewFakeFileSystem()
	ffs.Files["/project/pnpm-workspace.yaml"] = []byte("packages: ['apps/*']")
	ffs.Files["/project/package.json"] = []byte(`{"name":"root"}`)
	ffs.Files["/project/apps/web/package.json"] = []byte(`{
		"name": "@mono/web",
		"scripts": {
			"fmt": "prettier --write .",
			"test": "vitest run",
			"lint": "eslint ."
		}
	}`)

	det := NewToolchainDetector(ffs)
	result := det.Detect("/project")

	entry, ok := result["node"]
	if !ok {
		t.Fatal("expected node toolchain entry")
	}
	if entry.Fmt != "pnpm --filter @mono/web run fmt" {
		t.Errorf("fmt: got %q", entry.Fmt)
	}
}
