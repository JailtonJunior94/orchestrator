//go:build integration

package skills

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/suite"
)

// skillsLock espelha a estrutura de skills-lock.json.
type skillsLock struct {
	Version int                       `json:"version"`
	Skills  map[string]skillLockEntry `json:"skills"`
}

type skillLockEntry struct {
	Source       string `json:"source"`
	SourceType   string `json:"sourceType"`
	ComputedHash string `json:"computedHash"`
}

type ContractSuite struct {
	suite.Suite
}

func TestContractSuite(t *testing.T) {
	suite.Run(t, new(ContractSuite))
}

// repoRoot retorna o caminho absoluto da raiz do repositorio (dois niveis acima de internal/skills/).
func repoRoot(t *testing.T) string {
	t.Helper()
	// Este arquivo vive em internal/skills/, entao sobe dois diretorios.
	dir, err := filepath.Abs(filepath.Join(".", "..", ".."))
	if err != nil {
		t.Fatalf("resolver raiz do repositorio: %v", err)
	}
	return dir
}

func readLockFile(t *testing.T, root string) skillsLock {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(root, "skills-lock.json"))
	if err != nil {
		t.Fatalf("ler skills-lock.json: %v", err)
	}
	var lock skillsLock
	if err := json.Unmarshal(data, &lock); err != nil {
		t.Fatalf("parsear skills-lock.json: %v", err)
	}
	return lock
}

func (s *ContractSuite) TestComplementarySkillsHaveLockEntry() {
	root := repoRoot(s.T())
	lock := readLockFile(s.T(), root)

	scenarios := make([]struct {
		skill string
	}, len(ComplementarySkills))
	for i, skill := range ComplementarySkills {
		scenarios[i].skill = skill
	}

	for _, scenario := range scenarios {
		s.Run(scenario.skill, func() {
			s.Contains(lock.Skills, scenario.skill, "complementary skill %q is missing from skills-lock.json", scenario.skill)
		})
	}
}

func (s *ContractSuite) TestLockedSkillsDirectoryExists() {
	root := repoRoot(s.T())
	lock := readLockFile(s.T(), root)
	installedDir := filepath.Join(root, ".agents", "skills")

	scenarios := make([]struct {
		skill string
	}, len(ComplementarySkills))
	for i, skill := range ComplementarySkills {
		scenarios[i].skill = skill
	}

	for _, scenario := range scenarios {
		s.Run(scenario.skill, func() {
			if _, ok := lock.Skills[scenario.skill]; !ok {
				s.T().Skipf("skill %q ausente no lock file", scenario.skill)
			}
			dir := filepath.Join(installedDir, scenario.skill)
			info, err := os.Stat(dir)
			s.NoError(err, "diretorio de skill instalada nao existe: %s", dir)
			if err != nil {
				return
			}
			s.True(info.IsDir(), "esperava diretorio mas encontrou arquivo: %s", dir)
		})
	}
}

func (s *ContractSuite) TestInstalledSkillsValidFrontmatter() {
	root := repoRoot(s.T())
	installedDir := filepath.Join(root, ".agents", "skills")

	scenarios := make([]struct {
		skill string
	}, len(ComplementarySkills))
	for i, skill := range ComplementarySkills {
		scenarios[i].skill = skill
	}

	for _, scenario := range scenarios {
		s.Run(scenario.skill, func() {
			skillMD := filepath.Join(installedDir, scenario.skill, "SKILL.md")
			data, err := os.ReadFile(skillMD)
			s.NoError(err, "SKILL.md nao encontrado")

			fm := NewCatalog().ParseFrontmatter(data)
			s.False(fm.Name == "", "SKILL.md frontmatter has empty Name for skill %q", scenario.skill)
		})
	}
}

func (s *ContractSuite) TestEmbeddedSkillsValidSchema() {
	root := repoRoot(s.T())
	embeddedDir := filepath.Join(root, "internal", "embedded", "assets", ".agents", "skills")

	entries, err := os.ReadDir(embeddedDir)
	s.NoError(err, "ler diretorio de skills embarcadas")

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		skillName := entry.Name()
		s.Run(skillName, func() {
			skillMD := filepath.Join(embeddedDir, skillName, "SKILL.md")
			data, err := os.ReadFile(skillMD)
			s.NoError(err, "SKILL.md nao encontrado")

			err = NewCatalog().ValidateFrontmatterSchema(data, skillName)
			s.NoError(err, "skill embarcada %q falhou no JSON Schema", skillName)
		})
	}
}
