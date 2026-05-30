package config

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/JailtonJunior94/ai-spec-harness/internal/skills"
)

// ConfigTypesSuite cobre os tipos de configuracao expostos pelo pacote
// (LocalSource, InstallOptions, UpgradeOptions) e suas invariantes.
type ConfigTypesSuite struct {
	suite.Suite
}

func TestConfigTypesSuite(t *testing.T) {
	suite.Run(t, new(ConfigTypesSuite))
}

func (s *ConfigTypesSuite) TestLocalSourceSourceDir() {
	scenarios := []struct {
		name string
		dir  string
	}{
		{name: "deve retornar path absoluto", dir: "/tmp/governance"},
		{name: "deve retornar path relativo", dir: "./source"},
		{name: "deve retornar string vazia", dir: ""},
	}

	for _, scenario := range scenarios {
		s.Run(scenario.name, func() {
			src := &LocalSource{Dir: scenario.dir}
			s.Equal(scenario.dir, src.SourceDir())
		})
	}
}

func (s *ConfigTypesSuite) TestLocalSourceImplementsSourceProvider() {
	s.Implements((*SourceProvider)(nil), &LocalSource{})
}

func (s *ConfigTypesSuite) TestInstallOptionsFields() {
	opts := InstallOptions{
		ProjectDir:   "/project",
		SourceDir:    "/source",
		Tools:        []skills.Tool{skills.ToolClaude, skills.ToolGemini},
		Langs:        []skills.Lang{skills.LangGo},
		LinkMode:     skills.LinkSymlink,
		DryRun:       true,
		GenerateCtx:  true,
		CodexProfile: "full",
		FocusPaths:   []string{"src/"},
	}

	s.Equal("/project", opts.ProjectDir)
	s.Len(opts.Tools, 2)
	s.Len(opts.Langs, 1)
	s.True(opts.DryRun)
	s.Equal(skills.LinkSymlink, opts.LinkMode)
}

func (s *ConfigTypesSuite) TestUpgradeOptionsFields() {
	opts := UpgradeOptions{
		ProjectDir:   "/project",
		SourceDir:    "/source",
		CheckOnly:    true,
		Langs:        []skills.Lang{skills.LangNode, skills.LangPython},
		CodexProfile: "lean",
	}

	s.Equal("/project", opts.ProjectDir)
	s.True(opts.CheckOnly)
	s.Len(opts.Langs, 2)
	s.Equal("lean", opts.CodexProfile)
}
