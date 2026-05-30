package detect

// stack_coverage_test.go valida deteccao de linguagem, framework, toolchain e arquitetura
// para cada stack suportada (Go, Node.js, Python) usando fixtures reais em disco.

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/JailtonJunior94/ai-spec-harness/internal/fs"
	"github.com/JailtonJunior94/ai-spec-harness/internal/skills"
)

type StackCoverageSuite struct {
	suite.Suite
}

func TestStackCoverageSuite(t *testing.T) {
	suite.Run(t, new(StackCoverageSuite))
}

func (s *StackCoverageSuite) TestStackCoverageFixtures() {
	type toolchainExpectation struct {
		key  string
		fmt  string
		test string
		lint string
	}

	scenarios := []struct {
		name         string
		fixture      string
		lang         skills.Lang
		framework    string
		noFrameworks bool
		toolchain    toolchainExpectation
		architecture ArchitectureType
		primaryStack string
	}{
		{
			name:         "deve cobrir fixture go monolith",
			fixture:      "go-monolith",
			lang:         skills.LangGo,
			noFrameworks: true,
			toolchain:    toolchainExpectation{key: "go", fmt: "gofmt -w .", test: "go test ./...", lint: "golangci-lint run"},
			architecture: ArchMonolith,
			primaryStack: "Go",
		},
		{
			name:         "deve cobrir fixture node api",
			fixture:      "node-api",
			lang:         skills.LangNode,
			framework:    "Express",
			toolchain:    toolchainExpectation{key: "node", fmt: "npm run fmt", test: "npm run test", lint: "npm run lint"},
			architecture: ArchMonolith,
			primaryStack: "Node.js",
		},
		{
			name:         "deve cobrir fixture python api",
			fixture:      "python-api",
			lang:         skills.LangPython,
			framework:    "FastAPI",
			toolchain:    toolchainExpectation{key: "python", fmt: "ruff format .", test: "pytest", lint: "ruff check ."},
			architecture: ArchMicroservice,
			primaryStack: "Python",
		},
		{
			name:         "deve cobrir fixture go microservice",
			fixture:      "go-microservice",
			lang:         skills.LangGo,
			framework:    "Gin",
			toolchain:    toolchainExpectation{key: "go", fmt: "gofmt -w .", test: "go test ./...", lint: "golangci-lint run"},
			architecture: ArchMicroservice,
			primaryStack: "Go",
		},
	}

	for _, scenario := range scenarios {
		s.Run(scenario.name, func() {
			dir := fixtureDir(scenario.fixture)
			osfs := fs.NewOSFileSystem()

			s.Run("deve detectar linguagem", func() {
				det := NewFileDetector(osfs)
				langs := det.DetectLangs(dir)
				s.Contains(langs, scenario.lang)
			})

			s.Run("deve detectar framework", func() {
				det := NewFrameworkDetector(osfs)
				frameworks := det.Detect(dir)
				if scenario.noFrameworks {
					s.Empty(frameworks)
					return
				}
				s.Contains(frameworks, scenario.framework)
			})

			s.Run("deve detectar toolchain", func() {
				det := NewToolchainDetector(osfs)
				result := det.Detect(dir)
				s.Require().Contains(result, scenario.toolchain.key)
				entry := result[scenario.toolchain.key]
				s.Equal(scenario.toolchain.fmt, entry.Fmt)
				s.Equal(scenario.toolchain.test, entry.Test)
				s.Equal(scenario.toolchain.lint, entry.Lint)
			})

			s.Run("deve detectar arquitetura", func() {
				det := NewArchitectureDetector(osfs)
				result := det.Detect(dir)
				s.Equal(scenario.architecture, result.Type)
			})

			s.Run("deve detectar stack primaria", func() {
				stacks := NewCatalog().DetectPrimaryStack(osfs, dir)
				s.Contains(stacks, scenario.primaryStack)
			})
		})
	}
}
