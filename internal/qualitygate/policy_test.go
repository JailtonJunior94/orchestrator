package qualitygate

import (
	"errors"
	"testing"
)

func TestDecodePolicySet_Valid(t *testing.T) {
	data := []byte(`
version: 1
policies:
  - task_type: feature
    risk: low
    required: [test]
    optional: [lint]
`)
	policySet, err := DecodePolicySet(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	taskType, err := NewTaskType("feature")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	selection, err := policySet.Lookup(taskType, RiskLow)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(selection.Required()) != 1 || selection.Required()[0] != CheckTest {
		t.Fatalf("required = %v, want [test]", selection.Required())
	}
}

func TestDecodePolicySet_RejectsUnsupportedVersion(t *testing.T) {
	data := []byte(`
version: 2
policies:
  - task_type: feature
    risk: low
    required: [test]
`)
	_, err := DecodePolicySet(data)
	if !errors.Is(err, ErrInvalidPolicy) {
		t.Fatalf("expected ErrInvalidPolicy, got %v", err)
	}
}

func TestDecodePolicySet_RejectsUnknownField(t *testing.T) {
	data := []byte(`
version: 1
policies:
  - task_type: feature
    risk: low
    required: [test]
    unknown_field: true
`)
	_, err := DecodePolicySet(data)
	if !errors.Is(err, ErrInvalidPolicy) {
		t.Fatalf("expected ErrInvalidPolicy for unknown field, got %v", err)
	}
}

func TestDecodePolicySet_RejectsUnknownRisk(t *testing.T) {
	data := []byte(`
version: 1
policies:
  - task_type: feature
    risk: critical
    required: [test]
`)
	_, err := DecodePolicySet(data)
	if !errors.Is(err, ErrInvalidPolicy) {
		t.Fatalf("expected ErrInvalidPolicy for unknown risk, got %v", err)
	}
}

func TestDecodePolicySet_RejectsUnknownCheck(t *testing.T) {
	data := []byte(`
version: 1
policies:
  - task_type: feature
    risk: low
    required: [coverage]
`)
	_, err := DecodePolicySet(data)
	if !errors.Is(err, ErrInvalidPolicy) {
		t.Fatalf("expected ErrInvalidPolicy for unknown check, got %v", err)
	}
}

func TestDecodePolicySet_RejectsDuplicateEntry(t *testing.T) {
	data := []byte(`
version: 1
policies:
  - task_type: feature
    risk: low
    required: [test]
  - task_type: feature
    risk: low
    required: [lint]
`)
	_, err := DecodePolicySet(data)
	if !errors.Is(err, ErrInvalidPolicy) {
		t.Fatalf("expected ErrInvalidPolicy for duplicate entry, got %v", err)
	}
}

func TestDecodePolicySet_RejectsEmptyPolicies(t *testing.T) {
	data := []byte(`
version: 1
policies: []
`)
	_, err := DecodePolicySet(data)
	if !errors.Is(err, ErrInvalidPolicy) {
		t.Fatalf("expected ErrInvalidPolicy for empty policies, got %v", err)
	}
}

func TestPolicySet_LookupUndeclaredCombinationIsNeverInferred(t *testing.T) {
	data := []byte(`
version: 1
policies:
  - task_type: feature
    risk: low
    required: [test]
`)
	policySet, err := DecodePolicySet(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	taskType, err := NewTaskType("feature")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	_, err = policySet.Lookup(taskType, RiskHigh)
	if !errors.Is(err, ErrPolicyNotDeclared) {
		t.Fatalf("expected ErrPolicyNotDeclared for undeclared combination, got %v", err)
	}
}

func TestDefaultPolicySet_IsValid(t *testing.T) {
	policySet, err := DefaultPolicySet()
	if err != nil {
		t.Fatalf("embedded default policy must decode cleanly: %v", err)
	}
	if _, err := policySet.Lookup(DefaultTaskType, DefaultRisk); err != nil {
		t.Fatalf("default policy must cover default task type/risk: %v", err)
	}
}
