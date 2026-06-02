package detect

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/JailtonJunior94/ai-spec-harness/internal/fs"
	"github.com/JailtonJunior94/ai-spec-harness/internal/skills"
)

// fakeLookPath simula exec.LookPath com um conjunto de binarios "presentes".
type fakeLookPath struct {
	present map[string]bool
}

func (f fakeLookPath) LookPath(file string) (string, error) {
	if f.present[file] {
		return "/usr/local/bin/" + file, nil
	}
	return "", errors.New("nao encontrado")
}

// fakeHomeDir retorna um home dir fixo para testes.
type fakeHomeDir struct {
	home string
	err  error
}

func (f fakeHomeDir) UserHomeDir() (string, error) {
	return f.home, f.err
}

type AgentSuite struct {
	suite.Suite
}

func TestAgentSuite(t *testing.T) {
	suite.Run(t, new(AgentSuite))
}

func (s *AgentSuite) TestBinaryAgentDetectorDetect() {
	type args struct {
		lookPath   fakeLookPath
		homeDir    fakeHomeDir
		projectDir string
		files      map[string][]byte
		dirs       map[string]bool
		fileDet    bool
	}

	scenarios := []struct {
		name   string
		args   args
		expect func(got []skills.Tool, err error)
	}{
		{
			name: "deve detectar agentes por binarios no path",
			args: args{
				lookPath: fakeLookPath{present: map[string]bool{
					"claude-agent-acp": true,
					"codex-acp":        true,
				}},
				homeDir: fakeHomeDir{home: "/nonexistent-home-dir-xyz"},
			},
			expect: func(got []skills.Tool, err error) {
				s.NoError(err)
				toolSet := toSet(got)
				s.True(toolSet[skills.ToolClaude], "esperava ToolClaude detectado via binario")
				s.True(toolSet[skills.ToolCodex], "esperava ToolCodex detectado via binario")
				s.False(toolSet[skills.ToolGemini], "nao esperava ToolGemini — binario ausente")
				s.False(toolSet[skills.ToolCopilot], "nao esperava ToolCopilot — binario ausente")
			},
		},
		{
			name: "deve detectar agente por arquivo do projeto",
			args: args{
				lookPath:   fakeLookPath{present: map[string]bool{}},
				homeDir:    fakeHomeDir{home: "/nonexistent-home-xyz"},
				projectDir: "/project",
				files:      map[string][]byte{"/project/CLAUDE.md": []byte("# Claude")},
				fileDet:    true,
			},
			expect: func(got []skills.Tool, err error) {
				s.NoError(err)
				toolSet := toSet(got)
				s.True(toolSet[skills.ToolClaude], "esperava ToolClaude detectado via arquivo de projeto")
				s.Len(got, 1)
			},
		},
		{
			name: "deve retornar vazio para repositorio sem sinais",
			args: args{
				lookPath:   fakeLookPath{present: map[string]bool{}},
				homeDir:    fakeHomeDir{home: "/nonexistent-home-xyz"},
				projectDir: "/empty-project",
				dirs:       map[string]bool{"/empty-project": true},
				fileDet:    true,
			},
			expect: func(got []skills.Tool, err error) {
				s.NoError(err)
				s.Empty(got)
			},
		},
		{
			name: "deve degradar graciosamente quando home dir retorna erro",
			args: args{
				lookPath: fakeLookPath{present: map[string]bool{}},
				homeDir:  fakeHomeDir{err: errors.New("HOME nao definido")},
			},
			expect: func(got []skills.Tool, err error) {
				s.NoError(err)
				s.Empty(got)
			},
		},
		{
			name: "deve detectar 3 CLIs inegociaveis sem incluir Gemini (opt-in por projeto)",
			args: args{
				lookPath: fakeLookPath{present: map[string]bool{
					"claude-agent-acp": true,
					"codex-acp":        true,
					"gemini":           true,
					"copilot":          true,
				}},
				homeDir: fakeHomeDir{home: "/nonexistent-home-xyz"},
			},
			expect: func(got []skills.Tool, err error) {
				s.NoError(err)
				// Gemini e opt-in por sinal de projeto; mesmo com binario no
				// PATH, sem .gemini/ ou GEMINI.md no projectDir nao entra.
				s.Len(got, 3)
				for _, t := range got {
					s.NotEqual(skills.ToolGemini, t, "Gemini nao deve aparecer sem sinal de projeto")
				}
			},
		},
		{
			name: "deve incluir Gemini quando ha sinal de projeto (.gemini/ ou GEMINI.md)",
			args: args{
				lookPath:   fakeLookPath{present: map[string]bool{"gemini": true}},
				homeDir:    fakeHomeDir{home: "/nonexistent-home-xyz"},
				projectDir: "/project",
				files:      map[string][]byte{"/project/GEMINI.md": []byte("# Gemini")},
				fileDet:    true,
			},
			expect: func(got []skills.Tool, err error) {
				s.NoError(err)
				hasGemini := false
				for _, t := range got {
					if t == skills.ToolGemini {
						hasGemini = true
					}
				}
				s.True(hasGemini, "Gemini deve aparecer quando ha sinal de projeto")
			},
		},
		{
			name: "deve evitar duplicidade quando agente aparece em binario e arquivo",
			args: args{
				lookPath:   fakeLookPath{present: map[string]bool{"claude-agent-acp": true}},
				homeDir:    fakeHomeDir{home: "/nonexistent-home-xyz"},
				projectDir: "/project",
				files:      map[string][]byte{"/project/CLAUDE.md": []byte("# Claude")},
				fileDet:    true,
			},
			expect: func(got []skills.Tool, err error) {
				s.NoError(err)
				count := 0
				for _, tool := range got {
					if tool == skills.ToolClaude {
						count++
					}
				}
				s.Equal(1, count)
			},
		},
	}

	for _, scenario := range scenarios {
		s.Run(scenario.name, func() {
			ffs := fs.NewFakeFileSystem()
			for path, content := range scenario.args.files {
				ffs.Files[path] = content
			}
			for path, exists := range scenario.args.dirs {
				ffs.Dirs[path] = exists
			}

			var fileDet *FileDetector
			if scenario.args.fileDet {
				fileDet = NewFileDetector(ffs)
			}

			det := NewBinaryAgentDetector(scenario.args.lookPath, scenario.args.homeDir, fileDet)
			got, err := det.Detect(context.Background(), DetectOptions{ProjectDir: scenario.args.projectDir})

			scenario.expect(got, err)
		})
	}
}

func (s *AgentSuite) TestAllEntriesCommandsMatchSpecs() {
	entries := NewCatalog().allEntries()
	s.Require().Len(entries, 4, "esperava 4 entries (claude, codex, gemini, copilot)")
	for _, entry := range entries {
		s.NotEmpty(entry.command, "entry %s tem command vazio", entry.tool)
		s.NotEmpty(entry.tool, "entry com tool vazio")
	}
}

// toSet converte slice de Tool em mapa para lookup O(1).
func toSet(tools []skills.Tool) map[skills.Tool]bool {
	m := make(map[skills.Tool]bool, len(tools))
	for _, t := range tools {
		m[t] = true
	}
	return m
}
