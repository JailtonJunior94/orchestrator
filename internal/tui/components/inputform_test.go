package components

import (
	"testing"
)

func TestBuildInputForm_textField(t *testing.T) {
	fields := []InputField{
		{Name: "input", Type: "text", Label: "Workflow Input", Placeholder: "enter input..."},
	}

	form, result := BuildInputForm(fields)
	if form == nil {
		t.Fatal("expected non-nil form")
	}
	if _, ok := result.StringValues["input"]; !ok {
		t.Error("expected 'input' key in StringValues")
	}
}

func TestBuildInputForm_multilineField(t *testing.T) {
	fields := []InputField{
		{Name: "description", Type: "multiline", Label: "Description"},
	}

	_, result := BuildInputForm(fields)
	if _, ok := result.StringValues["description"]; !ok {
		t.Error("expected 'description' key in StringValues for multiline type")
	}
}

func TestBuildInputForm_selectField(t *testing.T) {
	fields := []InputField{
		{Name: "env", Type: "select", Label: "Environment", Options: []string{"dev", "staging", "prod"}},
	}

	_, result := BuildInputForm(fields)
	if _, ok := result.StringValues["env"]; !ok {
		t.Error("expected 'env' key in StringValues for select type")
	}
}

func TestBuildInputForm_confirmField(t *testing.T) {
	fields := []InputField{
		{Name: "confirm", Type: "confirm", Label: "Are you sure?"},
	}

	_, result := BuildInputForm(fields)
	if _, ok := result.BoolValues["confirm"]; !ok {
		t.Error("expected 'confirm' key in BoolValues for confirm type")
	}
}

func TestBuildInputForm_multipleFields(t *testing.T) {
	fields := []InputField{
		{Name: "f1", Type: "text", Label: "Field 1"},
		{Name: "f2", Type: "confirm", Label: "Field 2"},
	}

	form, result := BuildInputForm(fields)
	if form == nil {
		t.Fatal("expected non-nil form")
	}
	if len(result.StringValues) != 1 {
		t.Errorf("expected 1 string value, got %d", len(result.StringValues))
	}
	if len(result.BoolValues) != 1 {
		t.Errorf("expected 1 bool value, got %d", len(result.BoolValues))
	}
}
