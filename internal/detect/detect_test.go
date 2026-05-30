package detect

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/JailtonJunior94/ai-spec-harness/internal/fs"
	"github.com/JailtonJunior94/ai-spec-harness/internal/skills"
)

type DetectSuite struct {
	suite.Suite
}

func TestDetectSuite(t *testing.T) {
	suite.Run(t, new(DetectSuite))
}

func (s *DetectSuite) TestDetectLangs() {
	scenarios := []struct {
		name   string
		files  map[string][]byte
		dirs   map[string]bool
		expect func(langs []skills.Lang)
	}{
		{
			name: "deve detectar go e node",
			files: map[string][]byte{
				"/project/go.mod":       []byte("module example"),
				"/project/package.json": []byte("{}"),
			},
			expect: func(langs []skills.Lang) {
				s.Len(langs, 2)
				s.Equal(skills.LangGo, langs[0])
				s.Equal(skills.LangNode, langs[1])
			},
		},
		{
			name: "deve retornar vazio quando nao ha manifestos",
			dirs: map[string]bool{"/project": true},
			expect: func(langs []skills.Lang) {
				s.Empty(langs)
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

			det := NewFileDetector(ffs)
			langs := det.DetectLangs("/project")

			scenario.expect(langs)
		})
	}
}

func (s *DetectSuite) TestDetectTools() {
	ffs := fs.NewFakeFileSystem()
	ffs.Files["/project/CLAUDE.md"] = []byte("# Claude")
	ffs.Files["/project/GEMINI.md"] = []byte("# Gemini")

	det := NewFileDetector(ffs)
	tools := det.DetectTools("/project")

	s.Len(tools, 2)
}
