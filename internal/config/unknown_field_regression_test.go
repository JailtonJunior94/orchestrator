package config

import (
	"os"
	"testing"

	"github.com/stretchr/testify/suite"
	"gopkg.in/yaml.v3"
)

type UnknownFieldRegressionSuite struct {
	suite.Suite
}

func TestUnknownFieldRegressionSuite(t *testing.T) {
	suite.Run(t, new(UnknownFieldRegressionSuite))
}

func (s *UnknownFieldRegressionSuite) TestConfigYAMLWithUnknownFieldIsStillAccepted() {
	data := []byte("tasks_root: custom-tasks\nunknown_operational_field: true\ninjection: \"'; DROP TABLE users; --\"\n")

	var cfg Runtime
	err := yaml.Unmarshal(data, &cfg)

	s.NoError(err, "unknown fields in config.yaml must remain lenient (RF-06); the harness contract's strict parsing must never leak into operational config")
	s.Equal("custom-tasks", cfg.TasksRoot)
}

func (s *UnknownFieldRegressionSuite) TestResolverAcceptsProjectConfigWithUnknownField() {
	resolver := &DefaultResolver{
		HomeDir: "",
		readFile: func(path string) ([]byte, error) {
			if path == "/project/.claude/config.yaml" {
				return []byte("tasks_root: proj-tasks\nunknown_field_from_future_release: 42\n"), nil
			}
			return nil, os.ErrNotExist
		},
		isDir: func(path string) bool {
			return path == "/project/.git"
		},
	}

	got, err := resolver.Resolve("/project", Runtime{})

	s.Require().NoError(err)
	s.Equal("proj-tasks", got.TasksRoot)
}
