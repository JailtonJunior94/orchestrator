package bugschema

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeBugsFile(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "bugs.json")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write bugs file: %v", err)
	}
	return path
}

func TestValidate_ValidInput(t *testing.T) {
	path := writeBugsFile(t, `[
		{
			"id": "BUG-001",
			"severity": "high",
			"file": "main.go",
			"line": 42,
			"reproduction": "call foo()",
			"expected": "returns nil",
			"actual": "panics"
		}
	]`)

	if err := Validate(path, ""); err != nil {
		t.Errorf("expected nil error for valid input, got: %v", err)
	}
}

func TestValidate_EmptyArray(t *testing.T) {
	path := writeBugsFile(t, `[]`)

	err := Validate(path, "")
	if err == nil {
		t.Fatal("expected error for empty array")
	}
	if !strings.Contains(err.Error(), "minItems") {
		t.Errorf("expected minItems error, got: %v", err)
	}
}

func TestValidate_MissingSeverity(t *testing.T) {
	path := writeBugsFile(t, `[
		{
			"id": "BUG-001",
			"file": "main.go",
			"line": 42,
			"reproduction": "call foo()",
			"expected": "returns nil",
			"actual": "panics"
		}
	]`)

	err := Validate(path, "")
	if err == nil {
		t.Fatal("expected error for missing severity")
	}
	if !strings.Contains(err.Error(), "severity") {
		t.Errorf("expected error mentioning severity, got: %v", err)
	}
}

func TestValidate_InvalidSeverityEnum(t *testing.T) {
	path := writeBugsFile(t, `[
		{
			"id": "BUG-001",
			"severity": "extreme",
			"file": "main.go",
			"line": 42,
			"reproduction": "call foo()",
			"expected": "returns nil",
			"actual": "panics"
		}
	]`)

	err := Validate(path, "")
	if err == nil {
		t.Fatal("expected error for invalid severity enum")
	}
	if !strings.Contains(err.Error(), "extreme") {
		t.Errorf("expected error mentioning 'extreme', got: %v", err)
	}
}

func TestValidate_AdditionalProperties(t *testing.T) {
	path := writeBugsFile(t, `[
		{
			"id": "BUG-001",
			"severity": "low",
			"file": "main.go",
			"line": 42,
			"reproduction": "call foo()",
			"expected": "returns nil",
			"actual": "panics",
			"extra_field": "not allowed"
		}
	]`)

	err := Validate(path, "")
	if err == nil {
		t.Fatal("expected error for additional properties")
	}
	if !strings.Contains(err.Error(), "extra_field") {
		t.Errorf("expected error mentioning 'extra_field', got: %v", err)
	}
}
