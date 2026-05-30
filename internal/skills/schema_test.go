package skills

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/suite"
)

type FrontmatterSchemaSuite struct {
	suite.Suite
}

func TestFrontmatterSchemaSuite(t *testing.T) {
	suite.Run(t, new(FrontmatterSchemaSuite))
}

func (s *FrontmatterSchemaSuite) TestValidateFrontmatterSchemaValid() {
	scenarios := []struct {
		name      string
		content   string
		skillName string
	}{
		{
			name:    "campos obrigatorios",
			content: "---\nname: my-skill\nversion: 1.0.0\ndescription: Uma skill valida.\n---\n",
		},
		{
			name:    "com depends_on",
			content: "---\nname: execute-task\nversion: 1.2.3\ndescription: Executa tarefas.\ndepends_on: [review]\n---\n",
		},
		{
			name:    "versao com prefixo v",
			content: "---\nname: my-skill\nversion: v2.0.0\ndescription: Skill com versao prefixada.\n---\n",
		},
		{
			name:    "versao pre release",
			content: "---\nname: my-skill\nversion: 1.0.0-beta\ndescription: Skill pre-release.\n---\n",
		},
		{
			name:      "com skillname no erro",
			content:   "---\nname: my-skill\nversion: 1.0.0\ndescription: Skill com nome.\n---\n",
			skillName: "my-skill",
		},
		{
			name:    "com lang go",
			content: "---\nname: go-skill\nversion: 1.0.0\ndescription: Skill de Go.\nlang: go\n---\n",
		},
		{
			name:    "com link mode symlink",
			content: "---\nname: my-skill\nversion: 1.0.0\ndescription: Skill com link_mode.\nlink_mode: symlink\n---\n",
		},
		{
			name:    "com max depth",
			content: "---\nname: my-skill\nversion: 1.0.0\ndescription: Skill com max_depth.\nmax_depth: 2\n---\n",
		},
	}

	for _, scenario := range scenarios {
		s.Run(scenario.name, func() {
			err := NewCatalog().ValidateFrontmatterSchema([]byte(scenario.content), scenario.skillName)
			s.NoError(err, "esperava sucesso mas obteve erro")
		})
	}
}

func (s *FrontmatterSchemaSuite) TestValidateFrontmatterSchemaInvalid() {
	scenarios := []struct {
		name        string
		content     string
		skillName   string
		wantContain string
	}{
		{
			name:        "sem name",
			content:     "---\nversion: 1.0.0\ndescription: Sem name.\n---\n",
			wantContain: "name",
		},
		{
			name:        "sem version",
			content:     "---\nname: my-skill\ndescription: Sem version.\n---\n",
			wantContain: "version",
		},
		{
			name:        "sem description",
			content:     "---\nname: my-skill\nversion: 1.0.0\n---\n",
			wantContain: "description",
		},
		{
			name:        "version invalida",
			content:     "---\nname: my-skill\nversion: nao-e-semver\ndescription: Version invalida.\n---\n",
			wantContain: "version",
		},
		{
			name:        "lang invalido",
			content:     "---\nname: my-skill\nversion: 1.0.0\ndescription: Lang invalido.\nlang: rust\n---\n",
			wantContain: "lang",
		},
		{
			name:        "link mode invalido",
			content:     "---\nname: my-skill\nversion: 1.0.0\ndescription: LinkMode invalido.\nlink_mode: hard\n---\n",
			wantContain: "link_mode",
		},
		{
			name:        "com skillname no erro",
			content:     "---\nname: my-skill\nversion: 1.0.0\n---\n",
			skillName:   "my-skill",
			wantContain: "my-skill",
		},
	}

	for _, scenario := range scenarios {
		s.Run(scenario.name, func() {
			err := NewCatalog().ValidateFrontmatterSchema([]byte(scenario.content), scenario.skillName)
			s.Error(err, "esperava erro mas obteve sucesso")
			if err != nil && scenario.wantContain != "" {
				s.Contains(err.Error(), scenario.wantContain)
			}
		})
	}
}

func (s *FrontmatterSchemaSuite) TestValidateFrontmatterSchemaFixtures() {
	repoRoot := filepath.Join("..", "..")
	scenarios := []struct {
		name      string
		file      string
		skillName string
		expect    func(err error)
	}{
		{
			name:      "fixture valida",
			file:      filepath.Join(repoRoot, "testdata", "baselines", "skill-valid.md"),
			skillName: "skill-valid",
			expect: func(err error) {
				s.NoError(err, "fixture valida falhou na validacao")
			},
		},
		{
			name:      "fixture invalida",
			file:      filepath.Join(repoRoot, "testdata", "baselines", "skill-invalid.md"),
			skillName: "skill-invalid",
			expect: func(err error) {
				s.Error(err, "fixture invalida deveria falhar na validacao")
			},
		},
	}

	for _, scenario := range scenarios {
		s.Run(scenario.name, func() {
			data, err := os.ReadFile(scenario.file)
			s.NoError(err, "ler fixture")

			err = NewCatalog().ValidateFrontmatterSchema(data, scenario.skillName)
			scenario.expect(err)
		})
	}
}

func (s *FrontmatterSchemaSuite) TestValidateFrontmatterSchemaEmbeddedSkills() {
	repoRoot := filepath.Join("..", "..")
	skillsDir := filepath.Join(repoRoot, "internal", "embedded", "assets", ".agents", "skills")

	entries, err := os.ReadDir(skillsDir)
	s.NoError(err, "ler diretorio de skills embarcadas")

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		skillName := entry.Name()
		s.Run(skillName, func() {
			skillFile := filepath.Join(skillsDir, skillName, "SKILL.md")
			data, err := os.ReadFile(skillFile)
			s.NoError(err, "ler SKILL.md")

			err = NewCatalog().ValidateFrontmatterSchema(data, skillName)
			s.NoError(err, "skill embarcada %q falhou no schema", skillName)
		})
	}
}
