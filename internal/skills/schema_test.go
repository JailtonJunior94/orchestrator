package skills

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidateFrontmatterSchema_Valid(t *testing.T) {
	tests := []struct {
		name      string
		content   string
		skillName string
	}{
		{
			name: "campos_obrigatorios",
			content: "---\nname: my-skill\nversion: 1.0.0\ndescription: Uma skill valida.\n---\n",
		},
		{
			name: "com_depends_on",
			content: "---\nname: execute-task\nversion: 1.2.3\ndescription: Executa tarefas.\ndepends_on: [review]\n---\n",
		},
		{
			name: "versao_com_prefixo_v",
			content: "---\nname: my-skill\nversion: v2.0.0\ndescription: Skill com versao prefixada.\n---\n",
		},
		{
			name: "versao_pre_release",
			content: "---\nname: my-skill\nversion: 1.0.0-beta\ndescription: Skill pre-release.\n---\n",
		},
		{
			name:      "com_skillname_no_erro",
			content:   "---\nname: my-skill\nversion: 1.0.0\ndescription: Skill com nome.\n---\n",
			skillName: "my-skill",
		},
		{
			name: "com_lang_go",
			content: "---\nname: go-skill\nversion: 1.0.0\ndescription: Skill de Go.\nlang: go\n---\n",
		},
		{
			name: "com_link_mode_symlink",
			content: "---\nname: my-skill\nversion: 1.0.0\ndescription: Skill com link_mode.\nlink_mode: symlink\n---\n",
		},
		{
			name: "com_max_depth",
			content: "---\nname: my-skill\nversion: 1.0.0\ndescription: Skill com max_depth.\nmax_depth: 2\n---\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateFrontmatterSchema([]byte(tt.content), tt.skillName)
			if err != nil {
				t.Errorf("esperava sucesso mas obteve erro: %v", err)
			}
		})
	}
}

func TestValidateFrontmatterSchema_Invalid(t *testing.T) {
	tests := []struct {
		name        string
		content     string
		skillName   string
		wantContain string
	}{
		{
			name:        "sem_name",
			content:     "---\nversion: 1.0.0\ndescription: Sem name.\n---\n",
			wantContain: "name",
		},
		{
			name:        "sem_version",
			content:     "---\nname: my-skill\ndescription: Sem version.\n---\n",
			wantContain: "version",
		},
		{
			name:        "sem_description",
			content:     "---\nname: my-skill\nversion: 1.0.0\n---\n",
			wantContain: "description",
		},
		{
			name:        "version_invalida",
			content:     "---\nname: my-skill\nversion: nao-e-semver\ndescription: Version invalida.\n---\n",
			wantContain: "version",
		},
		{
			name:        "lang_invalido",
			content:     "---\nname: my-skill\nversion: 1.0.0\ndescription: Lang invalido.\nlang: rust\n---\n",
			wantContain: "lang",
		},
		{
			name:        "link_mode_invalido",
			content:     "---\nname: my-skill\nversion: 1.0.0\ndescription: LinkMode invalido.\nlink_mode: hard\n---\n",
			wantContain: "link_mode",
		},
		{
			name:        "com_skillname_no_erro",
			content:     "---\nname: my-skill\nversion: 1.0.0\n---\n",
			skillName:   "my-skill",
			wantContain: "my-skill",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateFrontmatterSchema([]byte(tt.content), tt.skillName)
			if err == nil {
				t.Fatal("esperava erro mas obteve sucesso")
			}
			if tt.wantContain != "" && !strings.Contains(err.Error(), tt.wantContain) {
				t.Errorf("erro %q nao contem %q", err.Error(), tt.wantContain)
			}
		})
	}
}

func TestValidateFrontmatterSchema_Fixtures(t *testing.T) {
	repoRoot := filepath.Join("..", "..")

	t.Run("fixture_valida", func(t *testing.T) {
		path := filepath.Join(repoRoot, "testdata", "baselines", "skill-valid.md")
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("ler fixture: %v", err)
		}
		if err := ValidateFrontmatterSchema(data, "skill-valid"); err != nil {
			t.Errorf("fixture valida falhou na validacao: %v", err)
		}
	})

	t.Run("fixture_invalida", func(t *testing.T) {
		path := filepath.Join(repoRoot, "testdata", "baselines", "skill-invalid.md")
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("ler fixture: %v", err)
		}
		if err := ValidateFrontmatterSchema(data, "skill-invalid"); err == nil {
			t.Error("fixture invalida deveria falhar na validacao")
		}
	})
}

func TestValidateFrontmatterSchema_EmbeddedSkills(t *testing.T) {
	repoRoot := filepath.Join("..", "..")
	skillsDir := filepath.Join(repoRoot, "internal", "embedded", "assets", ".agents", "skills")

	entries, err := os.ReadDir(skillsDir)
	if err != nil {
		t.Fatalf("ler diretorio de skills embarcadas: %v", err)
	}

	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		skillName := e.Name()
		t.Run(skillName, func(t *testing.T) {
			skillFile := filepath.Join(skillsDir, skillName, "SKILL.md")
			data, err := os.ReadFile(skillFile)
			if err != nil {
				t.Fatalf("ler SKILL.md: %v", err)
			}
			if err := ValidateFrontmatterSchema(data, skillName); err != nil {
				t.Errorf("skill embarcada %q falhou no schema: %v", skillName, err)
			}
		})
	}
}
