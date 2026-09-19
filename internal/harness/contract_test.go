package harness

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/stretchr/testify/suite"
	"gopkg.in/yaml.v3"
)

type ContractSuite struct {
	suite.Suite
}

func TestContractSuite(t *testing.T) {
	suite.Run(t, new(ContractSuite))
}

func (s *ContractSuite) readFixture(name string) []byte {
	return readFixtureFile(s.T(), name)
}

func (s *ContractSuite) TestDecodeAndValidateContract() {
	scenarios := []struct {
		name    string
		fixture string
		expect  func(contract Contract, err error)
	}{
		{
			name:    "accepts a minimal valid contract",
			fixture: "valid.yaml",
			expect: func(contract Contract, err error) {
				s.NoError(err)
				s.Equal(1, contract.Version)
				s.False(contract.Git.AutoCommit)
				s.True(contract.Approval.RequireForDestructiveOperations)
				s.Equal("declared", contract.Skills.DiscoveryMode)
			},
		},
		{
			name:    "rejects an unknown field with a typed error",
			fixture: "unknown-field.yaml",
			expect: func(contract Contract, err error) {
				s.Error(err)
				var typed *UnknownFieldError
				s.Require().True(errors.As(err, &typed))
				s.Contains(typed.Fields, "typo_field")
				s.Contains(typed.Error(), "typo_field")
				s.Contains(typed.Error(), "harness-contract.schema.json")
			},
		},
		{
			name:    "rejects an invalid field type with a typed error",
			fixture: "invalid-type.yaml",
			expect: func(contract Contract, err error) {
				s.Error(err)
				var typed *InvalidTypeError
				s.Require().True(errors.As(err, &typed))
				s.Equal("git.auto_commit", typed.Path)
				s.Contains(typed.Error(), "git.auto_commit")
				s.Contains(typed.Error(), ".agents/harness.yaml")
			},
		},
		{
			name:    "rejects a missing version with a typed error naming the supported version",
			fixture: "version-missing.yaml",
			expect: func(contract Contract, err error) {
				s.Error(err)
				var typed *UnsupportedVersionError
				s.Require().True(errors.As(err, &typed))
				s.Equal(SupportedVersion, typed.Supported)
				s.Contains(typed.Error(), "absent")
				s.Contains(typed.Error(), "version: 1")
			},
		},
		{
			name:    "rejects an incompatible version naming found and supported",
			fixture: "version-2.yaml",
			expect: func(contract Contract, err error) {
				s.Error(err)
				var typed *UnsupportedVersionError
				s.Require().True(errors.As(err, &typed))
				s.Equal(int64(2), typed.Found)
				s.Equal(SupportedVersion, typed.Supported)
				s.Contains(typed.Error(), "2")
				s.Contains(typed.Error(), "version: 1")
			},
		},
	}

	for _, scenario := range scenarios {
		s.Run(scenario.name, func() {
			data := s.readFixture(scenario.fixture)
			contract, err := NewContractDecoder(NewSchemaValidator()).Decode(data)
			scenario.expect(contract, err)
		})
	}
}

func (s *ContractSuite) TestDefaultContractIsValidAndReportsSensibleDefaults() {
	contract, err := NewEmbeddedDefault().Contract()
	s.Require().NoError(err)
	s.Equal(SupportedVersion, contract.Version)
	s.False(contract.Git.AutoCommit)
	s.False(contract.Git.AutoPush)
	s.True(contract.Approval.RequireForDestructiveOperations)
	s.True(contract.Quality.RequireTests)
	s.True(contract.Quality.RequireLint)
	s.True(contract.Evidence.RequireExecutionReport)
	s.Equal("declared", contract.Skills.DiscoveryMode)
}

func (s *ContractSuite) TestStaticValidationOnlyNoNetworkNoProcessNoLLM() {
	validator := NewSchemaValidator()
	raw := map[string]any{"version": 1}

	err := validator.Validate(raw)
	s.Error(err)
}

type naiveGitPolicyDoc struct {
	AutoCommit bool `yaml:"auto_commit" json:"auto_commit"`
	AutoPush   bool `yaml:"auto_push" json:"auto_push"`
}

type naiveApprovalPolicyDoc struct {
	RequireForDestructiveOperations bool `yaml:"require_for_destructive_operations" json:"require_for_destructive_operations"`
}

type naiveQualityPolicyDoc struct {
	RequireTests bool `yaml:"require_tests" json:"require_tests"`
	RequireLint  bool `yaml:"require_lint" json:"require_lint"`
}

type naiveEvidencePolicyDoc struct {
	RequireExecutionReport bool `yaml:"require_execution_report" json:"require_execution_report"`
}

type naiveSkillsPolicyDoc struct {
	DiscoveryMode string `yaml:"discovery_mode" json:"discovery_mode"`
}

type naiveContractDoc struct {
	Version  int                    `yaml:"version" json:"version"`
	Git      naiveGitPolicyDoc      `yaml:"git" json:"git"`
	Approval naiveApprovalPolicyDoc `yaml:"approval" json:"approval"`
	Quality  naiveQualityPolicyDoc  `yaml:"quality" json:"quality"`
	Evidence naiveEvidencePolicyDoc `yaml:"evidence" json:"evidence"`
	Skills   naiveSkillsPolicyDoc   `yaml:"skills" json:"skills"`
}

func (s *ContractSuite) TestPreservingBridgeIsTheOnlyReasonUnknownFieldsAreRejected() {
	data := s.readFixture("unknown-field.yaml")

	preservedRaw, err := NewYAMLBridge().Decode(data)
	s.Require().NoError(err)
	s.Contains(preservedRaw, "typo_field")

	err = NewSchemaValidator().Validate(preservedRaw)
	s.Require().Error(err)
	var unknownField *UnknownFieldError
	s.Require().True(errors.As(err, &unknownField))
	s.Contains(unknownField.Fields, "typo_field")

	naiveRaw := s.projectThroughTypedStructBeforeValidating(data)
	s.NotContains(naiveRaw, "typo_field")
	err = NewSchemaValidator().Validate(naiveRaw)
	s.NoError(err, "the naive struct-first path must be accepted by the schema, proving leg 2 is not a tautology")
}

func (s *ContractSuite) projectThroughTypedStructBeforeValidating(data []byte) map[string]any {
	s.T().Helper()

	var doc naiveContractDoc
	s.Require().NoError(yaml.Unmarshal(data, &doc))

	encoded, err := json.Marshal(doc)
	s.Require().NoError(err)

	var naiveRaw map[string]any
	s.Require().NoError(json.Unmarshal(encoded, &naiveRaw))
	return naiveRaw
}
