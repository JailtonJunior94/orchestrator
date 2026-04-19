package bugschema

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const schemaFixture = `{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "$id": "bug-schema.json",
  "type": "array",
  "minItems": 1,
  "items": {
    "type": "object",
    "required": ["id", "severity", "file", "line", "reproduction", "expected", "actual"],
    "additionalProperties": false,
    "properties": {
      "id":           { "type": "string" },
      "severity":     { "type": "string", "enum": ["critical", "high", "medium", "low"] },
      "file":         { "type": "string" },
      "line":         { "type": "integer" },
      "reproduction": { "type": "string" },
      "expected":     { "type": "string" },
      "actual":       { "type": "string" }
    }
  }
}`

func writeFile(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write %s: %v", name, err)
	}
	return path
}

func setup(t *testing.T, bugsJSON string) (bugsPath, schemaPath string) {
	t.Helper()
	dir := t.TempDir()
	schemaPath = writeFile(t, dir, "bug-schema.json", schemaFixture)
	bugsPath = writeFile(t, dir, "bugs.json", bugsJSON)
	return
}

func TestValidate_ValidInput(t *testing.T) {
	bugs, schema := setup(t, `[
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

	if err := Validate(bugs, schema); err != nil {
		t.Errorf("expected nil error for valid input, got: %v", err)
	}
}

func TestValidate_EmptyArray(t *testing.T) {
	bugs, schema := setup(t, `[]`)

	err := Validate(bugs, schema)
	if err == nil {
		t.Fatal("expected error for empty array")
	}
	if !strings.Contains(err.Error(), "minItems") {
		t.Errorf("expected minItems error, got: %v", err)
	}
}

func TestValidate_MissingSeverity(t *testing.T) {
	bugs, schema := setup(t, `[
		{
			"id": "BUG-001",
			"file": "main.go",
			"line": 42,
			"reproduction": "call foo()",
			"expected": "returns nil",
			"actual": "panics"
		}
	]`)

	err := Validate(bugs, schema)
	if err == nil {
		t.Fatal("expected error for missing severity")
	}
	if !strings.Contains(err.Error(), "severity") {
		t.Errorf("expected error mentioning severity, got: %v", err)
	}
}

func TestValidate_InvalidSeverityEnum(t *testing.T) {
	bugs, schema := setup(t, `[
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

	err := Validate(bugs, schema)
	if err == nil {
		t.Fatal("expected error for invalid severity enum")
	}
	if !strings.Contains(err.Error(), "extreme") && !strings.Contains(err.Error(), "enum") && !strings.Contains(err.Error(), "one of") {
		t.Errorf("expected error mentioning invalid severity value, got: %v", err)
	}
}

func TestValidate_AdditionalProperties(t *testing.T) {
	bugs, schema := setup(t, `[
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

	err := Validate(bugs, schema)
	if err == nil {
		t.Fatal("expected error for additional properties")
	}
	if !strings.Contains(err.Error(), "extra_field") && !strings.Contains(err.Error(), "additionalProperties") {
		t.Errorf("expected error mentioning 'extra_field' or 'additionalProperties', got: %v", err)
	}
}

func TestValidate_SchemaMissing(t *testing.T) {
	dir := t.TempDir()
	bugsPath := writeFile(t, dir, "bugs.json", `[{"id":"BUG-001","severity":"low","file":"f.go","line":1,"reproduction":"r","expected":"e","actual":"a"}]`)

	err := Validate(bugsPath, filepath.Join(dir, "nonexistent-schema.json"))
	if err == nil {
		t.Fatal("expected error for missing schema")
	}
	if !strings.Contains(err.Error(), "ler schema") {
		t.Errorf("expected error mentioning schema read failure, got: %v", err)
	}
}

func TestValidate_SchemaInvalid(t *testing.T) {
	dir := t.TempDir()
	schemaPath := writeFile(t, dir, "bad-schema.json", `{ this is not valid json }`)
	bugsPath := writeFile(t, dir, "bugs.json", `[]`)

	err := Validate(bugsPath, schemaPath)
	if err == nil {
		t.Fatal("expected error for invalid schema JSON")
	}
	if !strings.Contains(err.Error(), "schema") {
		t.Errorf("expected error mentioning schema, got: %v", err)
	}
}

func TestValidate_BugsFileMissing(t *testing.T) {
	dir := t.TempDir()
	schemaPath := writeFile(t, dir, "bug-schema.json", schemaFixture)

	err := Validate(filepath.Join(dir, "nonexistent-bugs.json"), schemaPath)
	if err == nil {
		t.Fatal("expected error for missing bugs file")
	}
	if !strings.Contains(err.Error(), "bugs") {
		t.Errorf("expected error mentioning bugs file, got: %v", err)
	}
}
