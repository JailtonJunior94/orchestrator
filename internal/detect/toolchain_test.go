package detect

import (
	"bytes"
	"strings"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/JailtonJunior94/ai-spec-harness/internal/fs"
)

type ToolchainSuite struct {
	suite.Suite
}

func TestToolchainSuite(t *testing.T) {
	suite.Run(t, new(ToolchainSuite))
}

func (s *ToolchainSuite) TestToolchainDetect() {
	type expectedEntry struct {
		key  string
		fmt  string
		test string
		lint string
	}

	scenarios := []struct {
		name   string
		files  map[string][]byte
		dirs   map[string]bool
		expect func(result map[string]ToolchainEntry)
	}{
		{
			name:  "deve detectar toolchain go",
			files: map[string][]byte{"/project/go.mod": []byte("module example")},
			expect: func(result map[string]ToolchainEntry) {
				s.assertToolchainEntry(result, expectedEntry{key: "go", fmt: "gofmt -w .", test: "go test ./...", lint: "golangci-lint run"})
			},
		},
		{
			name: "deve detectar toolchain node",
			files: map[string][]byte{"/project/package.json": []byte(`{
		"name": "test",
		"scripts": {
			"fmt": "prettier --write .",
			"test": "vitest run",
			"lint": "eslint ."
		}
	}`)},
			expect: func(result map[string]ToolchainEntry) {
				s.assertToolchainEntry(result, expectedEntry{key: "node", fmt: "npm run fmt", test: "npm run test", lint: "npm run lint"})
			},
		},
		{
			name: "deve detectar toolchain python com ruff",
			files: map[string][]byte{"/project/pyproject.toml": []byte(`[project]
name = "test"

[tool.ruff]
line-length = 88

[tool.pytest.ini_options]
testpaths = ["tests"]
`)},
			expect: func(result map[string]ToolchainEntry) {
				s.assertToolchainEntry(result, expectedEntry{key: "python", fmt: "ruff format .", test: "pytest", lint: "ruff check ."})
			},
		},
		{
			name: "deve retornar vazio sem manifestos",
			dirs: map[string]bool{"/project": true},
			expect: func(result map[string]ToolchainEntry) {
				s.Empty(result)
			},
		},
		{
			name: "deve detectar projeto polyglot",
			files: map[string][]byte{
				"/project/go.mod":       []byte("module example"),
				"/project/package.json": []byte(`{"scripts":{"test":"jest"}}`),
			},
			expect: func(result map[string]ToolchainEntry) {
				s.Contains(result, "go")
				s.Contains(result, "node")
			},
		},
		{
			name:  "deve usar makefile como fallback",
			files: map[string][]byte{"/project/Makefile": []byte("fmt:\n\tgofmt -w .\ntest:\n\tgo test ./...\nlint:\n\tgolangci-lint run\n")},
			expect: func(result map[string]ToolchainEntry) {
				s.assertToolchainEntry(result, expectedEntry{key: "unknown", fmt: "make fmt", test: "make test", lint: "make lint"})
			},
		},
		{
			name: "deve detectar bun para node",
			files: map[string][]byte{
				"/project/package.json": []byte(`{"scripts":{"test":"jest","lint":"eslint ."}}`),
				"/project/bun.lockb":    []byte(""),
			},
			expect: func(result map[string]ToolchainEntry) {
				s.Require().Contains(result, "node")
				s.Equal("bun run test", result["node"].Test)
			},
		},
		{
			name: "deve detectar dependencias opcionais python",
			files: map[string][]byte{"/project/pyproject.toml": []byte(`[project]
name = "test"

[project.optional-dependencies]
dev = ["ruff>=0.1", "pytest>=7.0"]
`)},
			expect: func(result map[string]ToolchainEntry) {
				s.assertToolchainEntry(result, expectedEntry{key: "python", fmt: "ruff format .", test: "pytest", lint: "ruff check ."})
			},
		},
		{
			name: "deve preferir go quando focus path aponta para subprojeto go",
			files: map[string][]byte{
				"/project/package.json":        []byte(`{"scripts":{"test":"jest","lint":"eslint ."}}`),
				"/project/services/api/go.mod": []byte("module example"),
			},
			dirs: map[string]bool{
				"/project/services":     true,
				"/project/services/api": true,
			},
			expect: func(result map[string]ToolchainEntry) {
				s.Require().Contains(result, "go")
				s.NotContains(result, "node")
				s.Equal("go test ./...", result["go"].Test)
			},
		},
		{
			name: "deve usar deteccao default sem focus paths",
			files: map[string][]byte{
				"/project/go.mod":       []byte("module example"),
				"/project/package.json": []byte(`{"scripts":{"test":"jest"}}`),
			},
			expect: func(result map[string]ToolchainEntry) {
				s.Contains(result, "go")
				s.Contains(result, "node")
			},
		},
		{
			name: "deve escolher manifesto com maior overlap de focus path",
			files: map[string][]byte{
				"/project/services/api/package.json": []byte(`{"name":"api","scripts":{"test":"jest"}}`),
				"/project/services/web/package.json": []byte(`{"name":"web","scripts":{"test":"vitest"}}`),
			},
			dirs: map[string]bool{
				"/project/services":     true,
				"/project/services/api": true,
				"/project/services/web": true,
			},
			expect: func(result map[string]ToolchainEntry) {
				s.Require().Contains(result, "node")
				s.True(strings.Contains(result["node"].Test, "api") || strings.Contains(result["node"].Test, "jest"))
			},
		},
		{
			name:  "deve fazer fallback quando focus path nao casa com manifesto",
			files: map[string][]byte{"/project/go.mod": []byte("module example")},
			expect: func(result map[string]ToolchainEntry) {
				s.Contains(result, "go")
			},
		},
		{
			name: "deve detectar pnpm workspace",
			files: map[string][]byte{
				"/project/pnpm-workspace.yaml": []byte("packages: ['apps/*']"),
				"/project/package.json":        []byte(`{"name":"root"}`),
				"/project/apps/web/package.json": []byte(`{
		"name": "@mono/web",
		"scripts": {
			"fmt": "prettier --write .",
			"test": "vitest run",
			"lint": "eslint ."
		}
	}`),
			},
			expect: func(result map[string]ToolchainEntry) {
				s.Require().Contains(result, "node")
				s.Equal("pnpm --filter @mono/web run fmt", result["node"].Fmt)
			},
		},
	}

	for _, scenario := range scenarios {
		s.Run(scenario.name, func() {
			ffs := fs.NewFakeFileSystem()
			for path, content := range scenario.files {
				ffs.Files[path] = content
			}
			for path, exists := range scenario.dirs {
				ffs.Dirs[path] = exists
			}

			det := NewToolchainDetector(ffs)
			s.configureFocusPaths(det, scenario.name)
			result := det.Detect("/project")

			scenario.expect(result)
		})
	}
}

func (s *ToolchainSuite) TestStrictMode() {
	scenarios := []struct {
		name   string
		strict bool
		expect func(result map[string]ToolchainEntry, warn string)
	}{
		{
			name:   "deve preservar output quando binario esta presente",
			strict: true,
			expect: func(result map[string]ToolchainEntry, warn string) {
				s.Require().Contains(result, "go")
				s.Equal("gofmt -w .", result["go"].Fmt)
			},
		},
		{
			name:   "deve preservar output quando binario esta ausente",
			strict: true,
			expect: func(result map[string]ToolchainEntry, warn string) {
				s.Equal("golangci-lint run", result["go"].Lint)
				if warn != "" {
					s.Contains(warn, "WARNING")
				}
			},
		},
		{
			name:   "deve omitir warnings fora do modo strict",
			strict: false,
			expect: func(result map[string]ToolchainEntry, warn string) {
				s.Empty(warn)
			},
		},
	}

	for _, scenario := range scenarios {
		s.Run(scenario.name, func() {
			ffs := fs.NewFakeFileSystem()
			ffs.Files["/project/go.mod"] = []byte("module example")

			var buf bytes.Buffer
			det := NewToolchainDetectorStrict(ffs, &buf)
			det.strict = scenario.strict
			result := det.Detect("/project")

			scenario.expect(result, buf.String())
		})
	}
}

func (s *ToolchainSuite) TestToolchainDetectFixturePythonMonorepo() {
	osfs := fs.NewOSFileSystem()
	det := NewToolchainDetector(osfs)
	result := det.Detect(fixtureDir("python-monorepo"))

	s.assertToolchainEntry(result, struct {
		key  string
		fmt  string
		test string
		lint string
	}{key: "python", fmt: "ruff format .", test: "pytest", lint: "ruff check ."})
}

func (s *ToolchainSuite) assertToolchainEntry(result map[string]ToolchainEntry, expected struct {
	key  string
	fmt  string
	test string
	lint string
}) {
	s.T().Helper()
	s.Require().Contains(result, expected.key)
	entry := result[expected.key]
	s.Equal(expected.fmt, entry.Fmt)
	s.Equal(expected.test, entry.Test)
	s.Equal(expected.lint, entry.Lint)
}

func (s *ToolchainSuite) configureFocusPaths(det *ToolchainDetector, name string) {
	s.T().Helper()
	switch name {
	case "deve preferir go quando focus path aponta para subprojeto go":
		det.FocusPaths = []string{"services/api/handler.go"}
	case "deve escolher manifesto com maior overlap de focus path":
		det.FocusPaths = []string{"services/api/handler.go"}
	case "deve fazer fallback quando focus path nao casa com manifesto":
		det.FocusPaths = []string{"some/unrelated/path/file.rs"}
	}
}
