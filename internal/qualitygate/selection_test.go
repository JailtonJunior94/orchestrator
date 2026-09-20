package qualitygate

import "testing"

func TestNewSelection_Valid(t *testing.T) {
	selection, err := NewSelection([]CheckKind{CheckTest, CheckLint}, []CheckKind{CheckFmt})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(selection.Required()) != 2 {
		t.Fatalf("required = %v, want 2 entries", selection.Required())
	}
	if len(selection.Optional()) != 1 {
		t.Fatalf("optional = %v, want 1 entry", selection.Optional())
	}
}

func TestNewSelection_RejectsOverlap(t *testing.T) {
	_, err := NewSelection([]CheckKind{CheckTest}, []CheckKind{CheckTest})
	if err == nil {
		t.Fatalf("expected error for overlapping check between required and optional")
	}
}

func TestNewSelection_RejectsDuplicateWithinRequired(t *testing.T) {
	_, err := NewSelection([]CheckKind{CheckTest, CheckTest}, nil)
	if err == nil {
		t.Fatalf("expected error for duplicate check within required")
	}
}

func TestNewSelection_RejectsInvalidKind(t *testing.T) {
	_, err := NewSelection([]CheckKind{CheckKind(99)}, nil)
	if err == nil {
		t.Fatalf("expected error for invalid check kind")
	}
}

func TestSelection_Empty(t *testing.T) {
	selection, err := NewSelection(nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(selection.Required()) != 0 || len(selection.Optional()) != 0 {
		t.Fatalf("expected empty selection, got %+v", selection)
	}
}
