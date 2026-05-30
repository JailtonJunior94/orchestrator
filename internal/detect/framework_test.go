package detect

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/JailtonJunior94/ai-spec-harness/internal/fs"
)

type FrameworkSuite struct {
	suite.Suite
}

func TestFrameworkSuite(t *testing.T) {
	suite.Run(t, new(FrameworkSuite))
}

func (s *FrameworkSuite) TestFrameworkDetect() {
	scenarios := []struct {
		name   string
		files  map[string][]byte
		dirs   map[string]bool
		expect func(frameworks []string)
	}{
		{
			name: "deve detectar frameworks go",
			files: map[string][]byte{"/project/go.mod": []byte(`module example

require (
	github.com/gin-gonic/gin v1.10.0
	google.golang.org/grpc v1.65.0
)`)},
			expect: func(frameworks []string) {
				s.Len(frameworks, 2)
				s.Equal("Gin", frameworks[0])
				s.Equal("gRPC", frameworks[1])
			},
		},
		{
			name: "deve detectar frameworks node",
			files: map[string][]byte{"/project/package.json": []byte(`{
		"dependencies": {
			"express": "^4.0.0",
			"next": "^14.0.0"
		}
	}`)},
			expect: func(frameworks []string) {
				s.Len(frameworks, 2)
			},
		},
		{
			name: "deve detectar frameworks python",
			files: map[string][]byte{"/project/pyproject.toml": []byte(`[project]
dependencies = ["fastapi>=0.111"]`)},
			expect: func(frameworks []string) {
				s.Equal([]string{"FastAPI"}, frameworks)
			},
		},
		{
			name: "deve retornar vazio sem manifestos",
			dirs: map[string]bool{"/project": true},
			expect: func(frameworks []string) {
				s.Empty(frameworks)
			},
		},
		{
			name: "deve deduplicar frameworks",
			files: map[string][]byte{
				"/project/go.mod":              []byte("require github.com/gin-gonic/gin v1.10.0"),
				"/project/services/api/go.mod": []byte("require github.com/gin-gonic/gin v1.10.0"),
			},
			expect: func(frameworks []string) {
				s.Len(frameworks, 1)
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

			det := NewFrameworkDetector(ffs)
			frameworks := det.Detect("/project")

			scenario.expect(frameworks)
		})
	}
}

func (s *FrameworkSuite) TestJoinFrameworks() {
	scenarios := []struct {
		name       string
		frameworks []string
		expected   string
	}{
		{name: "deve descrever lista vazia", frameworks: nil, expected: "nenhum framework dominante identificado"},
		{name: "deve juntar dois frameworks", frameworks: []string{"Gin", "gRPC"}, expected: "Gin, gRPC"},
	}

	for _, scenario := range scenarios {
		s.Run(scenario.name, func() {
			got := NewCatalog().JoinFrameworks(scenario.frameworks)
			s.Equal(scenario.expected, got)
		})
	}
}

func (s *FrameworkSuite) TestDetectPrimaryStack() {
	ffs := fs.NewFakeFileSystem()
	ffs.Files["/project/go.mod"] = []byte("module example")
	ffs.Files["/project/package.json"] = []byte("{}")

	stacks := NewCatalog().DetectPrimaryStack(ffs, "/project")

	s.Len(stacks, 2)
	s.Equal("Go", stacks[0])
	s.Equal("Node.js", stacks[1])
}
