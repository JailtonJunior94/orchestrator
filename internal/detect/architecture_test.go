package detect

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/JailtonJunior94/ai-spec-harness/internal/fs"
)

func fixtureDir(name string) string {
	return filepath.Join("..", "..", "testdata", name)
}

type ArchitectureSuite struct {
	suite.Suite
}

func TestArchitectureSuite(t *testing.T) {
	suite.Run(t, new(ArchitectureSuite))
}

func (s *ArchitectureSuite) TestDetectArchitectureType() {
	scenarios := []struct {
		name     string
		files    map[string][]byte
		dirs     map[string]bool
		expected ArchitectureType
	}{
		{
			name: "deve detectar monorepo com go work",
			files: map[string][]byte{
				"/project/go.work":             []byte("go 1.23"),
				"/project/services/api/go.mod": []byte("module api"),
			},
			expected: ArchMonorepo,
		},
		{
			name:     "deve detectar monorepo com pnpm workspace",
			files:    map[string][]byte{"/project/pnpm-workspace.yaml": []byte("packages: ['apps/*']")},
			expected: ArchMonorepo,
		},
		{
			name: "deve detectar monorepo com apps e packages",
			files: map[string][]byte{
				"/project/apps/web/index.ts":        []byte(""),
				"/project/packages/shared/index.ts": []byte(""),
			},
			expected: ArchMonorepo,
		},
		{
			name: "deve detectar monolito modular",
			files: map[string][]byte{
				"/project/internal/order/service.go":    []byte(""),
				"/project/internal/customer/service.go": []byte(""),
				"/project/internal/payment/service.go":  []byte(""),
			},
			expected: ArchModular,
		},
		{
			name: "deve detectar microservico",
			files: map[string][]byte{
				"/project/Dockerfile":          []byte("FROM golang"),
				"/project/k8s/deployment.yaml": []byte("apiVersion: apps/v1"),
			},
			expected: ArchMicroservice,
		},
		{
			name:     "deve usar monolito como fallback",
			files:    map[string][]byte{"/project/main.go": []byte("package main")},
			dirs:     map[string]bool{"/project": true},
			expected: ArchMonolith,
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

			det := NewArchitectureDetector(ffs)
			result := det.Detect("/project")

			s.Equal(scenario.expected, result.Type)
		})
	}
}

func (s *ArchitectureSuite) TestDetectArchitecturalPattern() {
	scenarios := []struct {
		name     string
		files    map[string][]byte
		expected string
	}{
		{
			name: "deve detectar clean architecture",
			files: map[string][]byte{
				"/project/domain/user.go":         []byte(""),
				"/project/application/service.go": []byte(""),
			},
			expected: "Predominio de Clean Architecture / Hexagonal com fronteiras explicitas entre dominio, aplicacao e infraestrutura.",
		},
		{
			name: "deve detectar arquitetura em camadas",
			files: map[string][]byte{
				"/project/controllers/handler.go": []byte(""),
				"/project/services/user.go":       []byte(""),
			},
			expected: "Predominio de arquitetura em camadas, com separacao entre transporte, servicos, persistencia e modelos.",
		},
		{
			name:     "deve detectar estrutura internal",
			files:    map[string][]byte{"/project/internal/order/service.go": []byte("")},
			expected: "Predominio de packages internos coesos, com estrutura orientada por dominio ou componente.",
		},
	}

	for _, scenario := range scenarios {
		s.Run(scenario.name, func() {
			ffs := fs.NewFakeFileSystem()
			for path, content := range scenario.files {
				ffs.Files[path] = content
			}

			det := NewArchitectureDetector(ffs)
			result := det.Detect("/project")

			s.NotEmpty(result.Pattern)
			s.Equal(scenario.expected, result.Pattern)
		})
	}
}

func (s *ArchitectureSuite) TestDetectArchitectureFixtures() {
	scenarios := []struct {
		name     string
		fixture  string
		expected ArchitectureType
	}{
		{name: "deve detectar fixture go microservice", fixture: "go-microservice", expected: ArchMicroservice},
		{name: "deve detectar fixture go modular", fixture: "go-modular", expected: ArchModular},
		{name: "deve detectar fixture node monorepo", fixture: "node-monorepo", expected: ArchMonorepo},
		{name: "deve detectar fixture polyglot monorepo", fixture: "polyglot-monorepo", expected: ArchMonorepo},
		{name: "deve detectar fixture python api", fixture: "python-api", expected: ArchMicroservice},
		{name: "deve detectar fixture go monolith", fixture: "go-monolith", expected: ArchMonolith},
		{name: "deve detectar fixture node api", fixture: "node-api", expected: ArchMonolith},
		{name: "deve detectar fixture python monorepo", fixture: "python-monorepo", expected: ArchMonorepo},
	}

	for _, scenario := range scenarios {
		s.Run(scenario.name, func() {
			osfs := fs.NewOSFileSystem()
			det := NewArchitectureDetector(osfs)
			result := det.Detect(fixtureDir(scenario.fixture))

			s.Equal(scenario.expected, result.Type)
		})
	}
}
