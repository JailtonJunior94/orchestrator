package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/suite"
)

// RuntimeSuite cobre DefaultRuntime, LoadRuntime (precedencia de arquivos,
// preenchimento parcial e erros) e a projecao de variaveis de ambiente.
type RuntimeSuite struct {
	suite.Suite
}

func TestRuntimeSuite(t *testing.T) {
	suite.Run(t, new(RuntimeSuite))
}

func (s *RuntimeSuite) TestDefaultRuntime() {
	cfg := NewRuntimeProvider().DefaultRuntime()
	s.Equal(".specs", cfg.TasksRoot)
	s.Equal("prd-", cfg.PRDPrefix)
	s.EqualValues(70.0, cfg.CoverageThreshold)
}

func (s *RuntimeSuite) TestLoadRuntime() {
	scenarios := []struct {
		name   string
		files  map[string]string
		repo   func(dir string) string
		expect func(cfg Runtime, err error)
	}{
		{
			name: "deve retornar defaults quando nao ha arquivo",
			expect: func(cfg Runtime, err error) {
				s.NoError(err)
				s.Equal(NewRuntimeProvider().DefaultRuntime(), cfg)
			},
		},
		{
			name: "deve preferir .claude sobre .agents",
			files: map[string]string{
				".claude/config.yaml": "tasks_root: claude-tasks\nprd_prefix: c-\n",
				".agents/config.yaml": "tasks_root: agents-tasks\nprd_prefix: a-\n",
			},
			expect: func(cfg Runtime, err error) {
				s.NoError(err)
				s.Equal("claude-tasks", cfg.TasksRoot)
				s.Equal("c-", cfg.PRDPrefix)
			},
		},
		{
			name: "deve usar .agents quando .claude esta ausente",
			files: map[string]string{
				".agents/config.yaml": "tasks_root: agents-tasks\n",
			},
			expect: func(cfg Runtime, err error) {
				s.NoError(err)
				s.Equal("agents-tasks", cfg.TasksRoot)
			},
		},
		{
			name: "deve preencher campos ausentes com defaults",
			files: map[string]string{
				".claude/config.yaml": "evidence_dir: custom/evidence\n",
			},
			expect: func(cfg Runtime, err error) {
				s.NoError(err)
				s.Equal("custom/evidence", cfg.EvidenceDir)
				s.Equal(".specs", cfg.TasksRoot)
				s.Equal("prd-", cfg.PRDPrefix)
				s.EqualValues(70.0, cfg.CoverageThreshold)
			},
		},
		{
			name: "deve retornar erro para YAML malformado",
			files: map[string]string{
				".claude/config.yaml": "tasks_root: [unclosed\n",
			},
			expect: func(cfg Runtime, err error) {
				s.Error(err)
			},
		},
		{
			name: "deve retornar defaults quando repoRoot e vazio",
			repo: func(dir string) string { return "" },
			expect: func(cfg Runtime, err error) {
				s.NoError(err)
				s.Equal(NewRuntimeProvider().DefaultRuntime(), cfg)
			},
		},
	}

	for _, scenario := range scenarios {
		s.Run(scenario.name, func() {
			dir := s.T().TempDir()
			for rel, content := range scenario.files {
				mustWrite(s.T(), filepath.Join(dir, rel), content)
			}

			repoRoot := dir
			if scenario.repo != nil {
				repoRoot = scenario.repo(dir)
			}

			cfg, err := NewRuntimeProvider().LoadRuntime(repoRoot)

			scenario.expect(cfg, err)
		})
	}
}

func (s *RuntimeSuite) TestEnvVarsProjectsAllKeys() {
	r := Runtime{
		TasksRoot:         "tk",
		PRDPrefix:         "p-",
		EvidenceDir:       "ev",
		CoverageThreshold: 75.5,
		LanguageDefault:   "go",
	}

	env := r.EnvVars()

	want := map[string]string{
		"AI_TASKS_ROOT":         "tk",
		"AI_PRD_PREFIX":         "p-",
		"AI_EVIDENCE_DIR":       "ev",
		"AI_COVERAGE_THRESHOLD": "75.5",
		"AI_LANGUAGE_DEFAULT":   "go",
	}
	for k, v := range want {
		s.Equal(v, env[k], "EnvVars[%q]", k)
	}
}

// mustWrite escreve um arquivo criando os diretorios pais; helper de teste
// compartilhado pelos suites do pacote config.
func mustWrite(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}
