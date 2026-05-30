package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/suite"
)

// ResolverSuite cobre a resolucao hierarquica de Runtime (flags > projeto >
// global > built-in), incluindo upward-walk, precedencia de candidatos e
// propagacao de erros de config malformada.
type ResolverSuite struct {
	suite.Suite
}

func TestResolverSuite(t *testing.T) {
	suite.Run(t, new(ResolverSuite))
}

// resolverWithFS cria um DefaultResolver com readFile e isDir virtualizados via
// um mapa de arquivos em memoria, sem acessar o FS real.
func (s *ResolverSuite) resolverWithFS(files map[string]string, dirs map[string]bool) *DefaultResolver {
	return &DefaultResolver{
		HomeDir: "",
		readFile: func(path string) ([]byte, error) {
			if data, ok := files[path]; ok {
				return []byte(data), nil
			}
			return nil, os.ErrNotExist
		},
		isDir: func(path string) bool {
			return dirs[path]
		},
	}
}

func (s *ResolverSuite) TestResolve() {
	type args struct {
		homeDir   string
		cwd       string
		overrides Runtime
	}

	scenarios := []struct {
		name   string
		files  map[string]string
		dirs   map[string]bool
		args   args
		expect func(got Runtime, err error)
	}{
		{
			name: "deve retornar DefaultRuntime quando nao ha config global nem projeto",
			args: args{cwd: "/some/cwd"},
			expect: func(got Runtime, err error) {
				s.NoError(err)
				s.Equal(NewRuntimeProvider().DefaultRuntime(), got)
			},
		},
		{
			name: "deve respeitar precedencia flags > projeto > global > built-in",
			files: map[string]string{
				"/home/user/.aispec/config.yaml": "tasks_root: global-tasks\nprd_prefix: g-\ncoverage_threshold: 60\n",
				"/project/.claude/config.yaml":   "tasks_root: project-tasks\nevidence_dir: proj-ev\n",
			},
			dirs: map[string]bool{"/project/.git": true},
			args: args{
				homeDir:   "/home/user",
				cwd:       "/project/sub/dir",
				overrides: Runtime{TasksRoot: "flag-tasks"},
			},
			expect: func(got Runtime, err error) {
				s.NoError(err)
				s.Equal("flag-tasks", got.TasksRoot)
				s.Equal("proj-ev", got.EvidenceDir)
				s.Equal("g-", got.PRDPrefix)
				s.EqualValues(60, got.CoverageThreshold)
			},
		},
		{
			name:  "deve encontrar config de projeto via upward-walk a partir de subdir profundo",
			files: map[string]string{"/repo/.claude/config.yaml": "tasks_root: walked-tasks\n"},
			dirs:  map[string]bool{"/repo/.git": true},
			args:  args{cwd: "/repo/a/b/c/d"},
			expect: func(got Runtime, err error) {
				s.NoError(err)
				s.Equal("walked-tasks", got.TasksRoot)
			},
		},
		{
			name:  "deve tratar ausencia de config global como nao-fatal",
			files: map[string]string{"/project/.claude/config.yaml": "tasks_root: proj-only\n"},
			args:  args{homeDir: "/home/user", cwd: "/project"},
			expect: func(got Runtime, err error) {
				s.NoError(err)
				s.Equal("proj-only", got.TasksRoot)
				s.Equal("prd-", got.PRDPrefix)
			},
		},
		{
			name:  "deve retornar erro ao parsear config global malformada",
			files: map[string]string{"/home/user/.aispec/config.yaml": "tasks_root: [unclosed\n"},
			args:  args{homeDir: "/home/user"},
			expect: func(got Runtime, err error) {
				s.Error(err)
			},
		},
		{
			name:  "deve retornar erro ao parsear config de projeto malformada",
			files: map[string]string{"/project/.claude/config.yaml": "tasks_root: [unclosed\n"},
			args:  args{cwd: "/project"},
			expect: func(got Runtime, err error) {
				s.Error(err)
			},
		},
		{
			name: "deve parar o upward-walk ao encontrar marcador .git",
			dirs: map[string]bool{"/root/.git": true},
			args: args{cwd: "/root/a/b"},
			expect: func(got Runtime, err error) {
				s.NoError(err)
				s.Equal(NewRuntimeProvider().DefaultRuntime(), got)
			},
		},
		{
			name: "deve priorizar .aispec sobre .claude no mesmo diretorio",
			files: map[string]string{
				"/project/.aispec/config.yaml": "tasks_root: aispec-tasks\n",
				"/project/.claude/config.yaml": "tasks_root: claude-tasks\n",
			},
			args: args{cwd: "/project"},
			expect: func(got Runtime, err error) {
				s.NoError(err)
				s.Equal("aispec-tasks", got.TasksRoot)
			},
		},
		{
			name: "deve ler e mesclar as novas chaves operacionais",
			files: map[string]string{
				"/project/.claude/config.yaml": strings.Join([]string{
					"timeout: 30s",
					"max_retries: 3",
					"retry_backoff_multiplier: 1.5",
					"concurrent: 4",
					"batch_size: 10",
					"default_tool: claude",
				}, "\n") + "\n",
			},
			args: args{cwd: "/project"},
			expect: func(got Runtime, err error) {
				s.NoError(err)
				s.Equal("30s", got.Timeout)
				s.Equal(3, got.MaxRetries)
				s.Equal(1.5, got.RetryBackoffMultiplier)
				s.Equal(4, got.Concurrent)
				s.Equal(10, got.BatchSize)
				s.Equal("claude", got.DefaultTool)
			},
		},
		{
			name: "deve preservar comportamento F1 quando campos operacionais sao zero-value",
			args: args{},
			expect: func(got Runtime, err error) {
				s.NoError(err)
				s.Empty(got.Timeout)
				s.Zero(got.MaxRetries)
				s.Zero(got.RetryBackoffMultiplier)
				s.Zero(got.Concurrent)
				s.Zero(got.BatchSize)
				s.Empty(got.DefaultTool)
			},
		},
		{
			name: "deve ignorar config global quando HomeDir e vazio",
			args: args{homeDir: ""},
			expect: func(got Runtime, err error) {
				s.NoError(err)
				s.Equal(NewRuntimeProvider().DefaultRuntime(), got)
			},
		},
		{
			name: "deve mesclar campo a campo sem sobrescrever campos nao declarados",
			files: map[string]string{
				"/home/user/.aispec/config.yaml": "tasks_root: global\nprd_prefix: g-\ncoverage_threshold: 60\n",
				"/project/.claude/config.yaml":   "tasks_root: project\n",
			},
			args: args{homeDir: "/home/user", cwd: "/project"},
			expect: func(got Runtime, err error) {
				s.NoError(err)
				s.Equal("project", got.TasksRoot)
				s.Equal("g-", got.PRDPrefix)
				s.EqualValues(60, got.CoverageThreshold)
			},
		},
	}

	for _, scenario := range scenarios {
		s.Run(scenario.name, func() {
			r := s.resolverWithFS(scenario.files, scenario.dirs)
			r.HomeDir = scenario.args.homeDir

			got, err := r.Resolve(scenario.args.cwd, scenario.args.overrides)

			scenario.expect(got, err)
		})
	}
}

func (s *ResolverSuite) TestLoadRuntime() {
	scenarios := []struct {
		name     string
		projYAML string
		expect   func(got Runtime, err error)
	}{
		{
			name: "deve retornar DefaultRuntime quando nao ha config",
			expect: func(got Runtime, err error) {
				s.NoError(err)
				s.Equal(NewRuntimeProvider().DefaultRuntime(), got)
			},
		},
		{
			name:     "deve mesclar config de projeto com defaults",
			projYAML: "tasks_root: my-tasks\nprd_prefix: my-\n",
			expect: func(got Runtime, err error) {
				s.NoError(err)
				s.Equal("my-tasks", got.TasksRoot)
				s.Equal("my-", got.PRDPrefix)
				s.EqualValues(70.0, got.CoverageThreshold)
			},
		},
	}

	for _, scenario := range scenarios {
		s.Run(scenario.name, func() {
			dir := s.T().TempDir()
			if scenario.projYAML != "" {
				mustWrite(s.T(), filepath.Join(dir, ".claude", "config.yaml"), scenario.projYAML)
			}

			got, err := NewRuntimeProvider().LoadRuntime(dir)

			scenario.expect(got, err)
		})
	}
}
