package harness

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

type DuplicateKeysSuite struct {
	suite.Suite
}

func TestDuplicateKeysSuite(t *testing.T) {
	suite.Run(t, new(DuplicateKeysSuite))
}

func (s *DuplicateKeysSuite) TestOperationalKeyNamesReflectsConfigRuntime() {
	names := NewDuplicateKeyChecker().OperationalKeyNames()

	s.Contains(names, "tasks_root")
	s.Contains(names, "prd_prefix")
	s.Contains(names, "evidence_dir")
	s.Contains(names, "coverage_threshold")
	s.Contains(names, "language_default")
	s.Contains(names, "timeout")
	s.Contains(names, "max_retries")
	s.Contains(names, "retry_backoff_multiplier")
	s.Contains(names, "concurrent")
	s.Contains(names, "batch_size")
	s.Contains(names, "default_tool")
	s.Contains(names, "max_bugfix_iterations")
	s.Contains(names, "handoff_lease_ttl")
	s.Contains(names, "durable_memory_enabled")
	s.NotContains(names, "-")
}

func (s *DuplicateKeysSuite) TestSchemaKeyNamesCoversAllContractFields() {
	names, err := NewSchemaInspector().KeyNames()
	s.Require().NoError(err)

	for _, expected := range []string{
		"version", "git", "auto_commit", "auto_push",
		"approval", "require_for_destructive_operations",
		"quality", "require_tests", "require_lint",
		"evidence", "require_execution_report",
		"skills", "discovery_mode",
	} {
		s.Contains(names, expected)
	}
}

func (s *DuplicateKeysSuite) TestCheckNoDuplicateKeysPassesForTheCurrentSchema() {
	err := NewDuplicateKeyChecker().Check()
	s.NoError(err)
}

func (s *DuplicateKeysSuite) TestFindDuplicateKeysDetectsOverlap() {
	scenarios := []struct {
		name       string
		schema     []string
		operations []string
		expect     []string
	}{
		{
			name:       "no overlap",
			schema:     []string{"git", "approval"},
			operations: []string{"tasks_root", "timeout"},
			expect:     nil,
		},
		{
			name:       "one colliding key",
			schema:     []string{"git", "timeout"},
			operations: []string{"tasks_root", "timeout"},
			expect:     []string{"timeout"},
		},
		{
			name:       "multiple colliding keys sorted",
			schema:     []string{"concurrent", "git", "evidence_dir"},
			operations: []string{"evidence_dir", "concurrent", "tasks_root"},
			expect:     []string{"concurrent", "evidence_dir"},
		},
	}

	checker := NewDuplicateKeyChecker()
	for _, scenario := range scenarios {
		s.Run(scenario.name, func() {
			s.Equal(scenario.expect, checker.findDuplicateKeys(scenario.schema, scenario.operations))
		})
	}
}
